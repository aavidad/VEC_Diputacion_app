package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

var ErrRegistroDatosContactoParticipacionNoDisponible = errors.New("bolsa: registro de datos de contacto de participacion no disponible")

// ServicioDatosContactoParticipacion (B4) registra y consulta el correo y los
// dos teléfonos de una participación. Reutiliza el contexto, la autorización V3
// y la pertenencia de B2; el agregado se cifra antes de cruzar al repositorio y
// solo se descifra para la lectura autorizada de RRHH.
type ServicioDatosContactoParticipacion struct {
	contexto    puertosbolsa.ResolutorContextoSituacionParticipacion
	autorizador puertosbolsa.AutorizadorSituacionParticipacionV3
	pertenencia puertosbolsa.VerificadorPertenenciaParticipacion
	cifrador    puertosbolsa.CifradorDatosContactoParticipacion
	repositorio puertosbolsa.RepositorioDatosContactoParticipacion
	reloj       func() time.Time
}

func NuevoServicioDatosContactoParticipacion(
	c puertosbolsa.ResolutorContextoSituacionParticipacion,
	a puertosbolsa.AutorizadorSituacionParticipacionV3,
	p puertosbolsa.VerificadorPertenenciaParticipacion,
	cifrador puertosbolsa.CifradorDatosContactoParticipacion,
	r puertosbolsa.RepositorioDatosContactoParticipacion,
	reloj func() time.Time,
) (*ServicioDatosContactoParticipacion, error) {
	if c == nil || a == nil || p == nil || cifrador == nil || r == nil || reloj == nil {
		return nil, ErrRegistroDatosContactoParticipacionNoDisponible
	}
	return &ServicioDatosContactoParticipacion{contexto: c, autorizador: a, pertenencia: p, cifrador: cifrador, repositorio: r, reloj: reloj}, nil
}

func (s *ServicioDatosContactoParticipacion) Registrar(ctx context.Context, solicitud puertosbolsa.SolicitudRegistrarDatosContactoParticipacion) (puertosbolsa.RegistroDatosContactoParticipacion, error) {
	if ctx == nil || s == nil || solicitud.Validar() != nil {
		return puertosbolsa.RegistroDatosContactoParticipacion{}, ErrRegistroDatosContactoParticipacionNoDisponible
	}
	datos := solicitud.Datos.Normalizar()
	if err := datos.Validar(); err != nil {
		return puertosbolsa.RegistroDatosContactoParticipacion{}, err
	}
	actor := solicitud.ResultadoContexto.Contexto
	resuelto, err := s.contexto.ResolverContextoSituacionParticipacion(ctx, actor, solicitud.BolsaRef, solicitud.ParticipacionRef)
	if err != nil || resuelto.Validar() != nil || actor.PersonaRef == "" {
		return puertosbolsa.RegistroDatosContactoParticipacion{}, errorDependenciaDatosContacto(err)
	}
	pertenece, err := s.pertenencia.ParticipacionPerteneceABolsa(ctx, solicitud.BolsaRef, solicitud.ParticipacionRef)
	if err != nil {
		return puertosbolsa.RegistroDatosContactoParticipacion{}, err
	}
	if !pertenece {
		return puertosbolsa.RegistroDatosContactoParticipacion{}, dominiovec.ErrAutorizacionDenegada
	}
	recurso := dominiovec.RecursoAutorizable{Referencia: solicitud.ParticipacionRef, ModuloID: puertosbolsa.ModuloSituacionParticipacion, Tipo: puertosbolsa.TipoRecursoSituacionParticipacion, Ambitos: map[string]string{"unidad_ref": resuelto.UnidadRef, "ambito_ref": resuelto.AmbitoRef}}
	auth, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(dominiovec.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: solicitud.Vinculo, ReferenciaMotivo: solicitud.MotivoAutorizacion, Accion: puertosbolsa.AccionRegistrarDatosContactoParticipacion, Recurso: recurso, Finalidad: puertosbolsa.FinalidadRegistrarDatosContactoParticipacion, Correlacion: solicitud.Correlacion})
	if err != nil {
		return puertosbolsa.RegistroDatosContactoParticipacion{}, dominiovec.ErrAutorizacionDenegada
	}
	decision, confirmacion, exportador, err := s.autorizador.EmitirMaterialAutorizacionAtestadaV3(ctx, auth, solicitud.ResultadoContexto)
	if err != nil || exportador == nil || decision.ValidarPara(auth) != nil {
		return puertosbolsa.RegistroDatosContactoParticipacion{}, errorDependenciaDatosContacto(err)
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !materialAutorizacionBorradorLlamamientoExacto(auth, decision, confirmacion, solicitud.ResultadoContexto, solicitud.MotivoAutorizacion, material, puertosbolsa.AudienciaRegistrarDatosContactoParticipacion) {
		return puertosbolsa.RegistroDatosContactoParticipacion{}, errorDependenciaDatosContacto(err)
	}
	// Recuperación tras autorizar: misma clave y mismos datos devuelven el
	// recibo original sin versión nueva; misma clave con otros datos, rechazo.
	previo, err := s.repositorio.BuscarRegistroDatosContacto(ctx, solicitud.ParticipacionRef, solicitud.ClaveIdempotencia)
	if err == nil {
		iguales, errComparacion := s.mismosDatos(ctx, previo, datos)
		if errComparacion != nil {
			return puertosbolsa.RegistroDatosContactoParticipacion{}, errComparacion
		}
		if !iguales || previo.Motivo != solicitud.Motivo {
			return puertosbolsa.RegistroDatosContactoParticipacion{}, dominiobolsa.ErrDatosContactoParticipacionInvalidos
		}
		previo.Reutilizada = true
		return previo, nil
	}
	if !errors.Is(err, puertosbolsa.ErrDatosContactoParticipacionNoEncontrados) {
		return puertosbolsa.RegistroDatosContactoParticipacion{}, err
	}
	version := uint64(1)
	vigente, err := s.repositorio.DatosContactoVigentes(ctx, solicitud.ParticipacionRef)
	switch {
	case err == nil:
		version = vigente.Version + 1
	case errors.Is(err, puertosbolsa.ErrDatosContactoParticipacionNoEncontrados):
	default:
		return puertosbolsa.RegistroDatosContactoParticipacion{}, err
	}
	claro, err := datos.Canonico()
	if err != nil {
		return puertosbolsa.RegistroDatosContactoParticipacion{}, err
	}
	sobre, err := s.cifrador.CifrarDatosContactoParticipacion(ctx, solicitud.ParticipacionRef, version, claro)
	if err != nil || sobre.Validar() != nil || sobre.Version != version {
		return puertosbolsa.RegistroDatosContactoParticipacion{}, ErrRegistroDatosContactoParticipacionNoDisponible
	}
	ahora := s.reloj().UTC().Truncate(time.Microsecond)
	h := sha256.Sum256([]byte(solicitud.ParticipacionRef + "\x1f" + solicitud.ClaveIdempotencia))
	return s.repositorio.RegistrarDatosContacto(ctx, puertosbolsa.ComandoRegistrarDatosContactoParticipacion{
		ParticipacionRef: solicitud.ParticipacionRef, BolsaRef: solicitud.BolsaRef, Sobre: sobre, Motivo: solicitud.Motivo,
		Actor: actor.PersonaRef, RegistradaEn: ahora, ClaveIdempotencia: solicitud.ClaveIdempotencia,
		ReciboRef:             "recibo:datos-contacto:" + hex.EncodeToString(h[:]),
		SolicitudAutorizacion: auth, Decision: decision, Confirmacion: confirmacion, Material: material,
	})
}

// Consultar entrega a RRHH los datos vigentes, en claro y enmascarados, tras
// resolver el contexto de unidad y ámbito sobre la bolsa y la participación.
func (s *ServicioDatosContactoParticipacion) Consultar(ctx context.Context, solicitud puertosbolsa.SolicitudConsultarDatosContactoParticipacion) (puertosbolsa.DatosContactoParticipacionLeidos, error) {
	if ctx == nil || s == nil || solicitud.BolsaRef == "" || solicitud.ParticipacionRef == "" || solicitud.ContextoActor.PersonaRef == "" {
		return puertosbolsa.DatosContactoParticipacionLeidos{}, ErrRegistroDatosContactoParticipacionNoDisponible
	}
	resuelto, err := s.contexto.ResolverContextoSituacionParticipacion(ctx, solicitud.ContextoActor, solicitud.BolsaRef, solicitud.ParticipacionRef)
	if err != nil || resuelto.Validar() != nil {
		return puertosbolsa.DatosContactoParticipacionLeidos{}, errorDependenciaDatosContacto(err)
	}
	pertenece, err := s.pertenencia.ParticipacionPerteneceABolsa(ctx, solicitud.BolsaRef, solicitud.ParticipacionRef)
	if err != nil {
		return puertosbolsa.DatosContactoParticipacionLeidos{}, err
	}
	if !pertenece {
		return puertosbolsa.DatosContactoParticipacionLeidos{}, dominiovec.ErrAutorizacionDenegada
	}
	vigente, err := s.repositorio.DatosContactoVigentes(ctx, solicitud.ParticipacionRef)
	if err != nil {
		return puertosbolsa.DatosContactoParticipacionLeidos{}, err
	}
	datos, err := s.descifrar(ctx, vigente)
	if err != nil {
		return puertosbolsa.DatosContactoParticipacionLeidos{}, err
	}
	return puertosbolsa.DatosContactoParticipacionLeidos{ParticipacionRef: vigente.ParticipacionRef, Version: vigente.Version, RegistradaEn: vigente.RegistradaEn, Datos: datos, Enmascarados: datos.Enmascarados()}, nil
}

func (s *ServicioDatosContactoParticipacion) descifrar(ctx context.Context, registro puertosbolsa.RegistroDatosContactoParticipacion) (dominiobolsa.DatosContactoParticipacion, error) {
	var datos dominiobolsa.DatosContactoParticipacion
	err := s.cifrador.ConDatosContactoParticipacionDescifrados(ctx, registro.ParticipacionRef, registro.Sobre, func(claro []byte) error {
		var errDatos error
		datos, errDatos = dominiobolsa.DatosContactoParticipacionDesdeCanonico(registro.ParticipacionRef, claro)
		return errDatos
	})
	if err != nil {
		return dominiobolsa.DatosContactoParticipacion{}, ErrRegistroDatosContactoParticipacionNoDisponible
	}
	return datos, nil
}

func (s *ServicioDatosContactoParticipacion) mismosDatos(ctx context.Context, registro puertosbolsa.RegistroDatosContactoParticipacion, datos dominiobolsa.DatosContactoParticipacion) (bool, error) {
	previos, err := s.descifrar(ctx, registro)
	if err != nil {
		return false, err
	}
	return previos.Igual(datos), nil
}

func errorDependenciaDatosContacto(err error) error {
	if errors.Is(err, dominiovec.ErrAutorizacionDenegada) || errors.Is(err, dominiovec.ErrPermissionDenied) {
		return err
	}
	return ErrRegistroDatosContactoParticipacionNoDisponible
}
