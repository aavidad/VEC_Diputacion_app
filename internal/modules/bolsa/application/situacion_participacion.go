package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"slices"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

var ErrCambioSituacionParticipacionNoDisponible = errors.New("bolsa: cambio de situacion de participacion no disponible")

type ServicioSituacionParticipacion struct {
	contexto    puertosbolsa.ResolutorContextoSituacionParticipacion
	autorizador puertosbolsa.AutorizadorSituacionParticipacionV3
	repositorio puertosbolsa.RepositorioSituacionParticipacion
	reloj       func() time.Time
	// reglas restringe la tabla compilada de transiciones con el catálogo
	// versionado. Nula: rige solo la tabla compilada.
	reglas puertosbolsa.ReglasTransicionesSituacion
}

func NuevoServicioSituacionParticipacion(c puertosbolsa.ResolutorContextoSituacionParticipacion, a puertosbolsa.AutorizadorSituacionParticipacionV3, r puertosbolsa.RepositorioSituacionParticipacion, reloj func() time.Time) (*ServicioSituacionParticipacion, error) {
	if c == nil || a == nil || r == nil || reloj == nil {
		return nil, ErrCambioSituacionParticipacionNoDisponible
	}
	return &ServicioSituacionParticipacion{contexto: c, autorizador: a, repositorio: r, reloj: reloj}, nil
}

// EstablecerReglasTransiciones compone el catálogo de transiciones. Se llama
// solo durante la composición, antes de atender peticiones.
func (s *ServicioSituacionParticipacion) EstablecerReglasTransiciones(reglas puertosbolsa.ReglasTransicionesSituacion) {
	if s != nil {
		s.reglas = reglas
	}
}

// transicionAdmitida aplica el catálogo sobre la tabla compilada, que ya ha
// validado el dominio y replica la base de datos: el catálogo puede cerrar una
// transición, nunca abrir otra. Un catálogo ilegible no se interpreta como
// permiso.
func (s *ServicioSituacionParticipacion) transicionAdmitida(ctx context.Context, origen, destino string) error {
	if s.reglas == nil {
		return nil
	}
	destinos, configurada, err := s.reglas.DestinosSituacion(ctx, origen)
	if err != nil {
		return ErrCambioSituacionParticipacionNoDisponible
	}
	if configurada && !slices.Contains(destinos, destino) {
		return dominiobolsa.ErrCambioSituacionParticipacionInvalido
	}
	return nil
}

func (s *ServicioSituacionParticipacion) Cambiar(ctx context.Context, solicitud puertosbolsa.SolicitudCambiarSituacionParticipacion) (puertosbolsa.RegistroSituacionParticipacion, error) {
	if ctx == nil || s == nil || s.contexto == nil || s.autorizador == nil || s.repositorio == nil || solicitud.Validar() != nil {
		return puertosbolsa.RegistroSituacionParticipacion{}, ErrCambioSituacionParticipacionNoDisponible
	}
	actor := solicitud.ResultadoContexto.Contexto
	resuelto, err := s.contexto.ResolverContextoSituacionParticipacion(ctx, actor, solicitud.BolsaRef, solicitud.ParticipacionRef)
	if err != nil || resuelto.Validar() != nil || actor.PersonaRef == "" {
		if errors.Is(err, dominiovec.ErrAutorizacionDenegada) || errors.Is(err, dominiovec.ErrPermissionDenied) {
			return puertosbolsa.RegistroSituacionParticipacion{}, err
		}
		return puertosbolsa.RegistroSituacionParticipacion{}, ErrCambioSituacionParticipacionNoDisponible
	}
	pertenece, err := s.repositorio.ParticipacionPerteneceABolsa(ctx, solicitud.BolsaRef, solicitud.ParticipacionRef)
	if err != nil {
		return puertosbolsa.RegistroSituacionParticipacion{}, err
	}
	if !pertenece {
		return puertosbolsa.RegistroSituacionParticipacion{}, dominiovec.ErrAutorizacionDenegada
	}
	recurso := dominiovec.RecursoAutorizable{Referencia: solicitud.ParticipacionRef, ModuloID: puertosbolsa.ModuloSituacionParticipacion, Tipo: puertosbolsa.TipoRecursoSituacionParticipacion, Ambitos: map[string]string{"unidad_ref": resuelto.UnidadRef, "ambito_ref": resuelto.AmbitoRef}}
	auth, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(dominiovec.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: solicitud.Vinculo, ReferenciaMotivo: solicitud.MotivoAutorizacion, Accion: puertosbolsa.AccionCambiarSituacionParticipacion, Recurso: recurso, Finalidad: puertosbolsa.FinalidadCambiarSituacionParticipacion, Correlacion: solicitud.Correlacion})
	if err != nil {
		return puertosbolsa.RegistroSituacionParticipacion{}, dominiovec.ErrAutorizacionDenegada
	}
	decision, confirmacion, exportador, err := s.autorizador.EmitirMaterialAutorizacionAtestadaV3(ctx, auth, solicitud.ResultadoContexto)
	if err != nil || exportador == nil || decision.ValidarPara(auth) != nil {
		return puertosbolsa.RegistroSituacionParticipacion{}, errorDependenciaSituacion(err)
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !materialAutorizacionBorradorLlamamientoExacto(auth, decision, confirmacion, solicitud.ResultadoContexto, solicitud.MotivoAutorizacion, material, puertosbolsa.AudienciaCambiarSituacionParticipacion) {
		return puertosbolsa.RegistroSituacionParticipacion{}, errorDependenciaSituacion(err)
	}
	// La recuperación se hace después de la autorización positiva. Así una
	// clave conocida no permite consultar un recibo fuera de ámbito y el
	// reintento no vuelve a evaluar la transición ya registrada.
	previo, err := s.repositorio.BuscarRegistroSituacion(ctx, solicitud.ParticipacionRef, solicitud.ClaveIdempotencia)
	repeticion := err == nil
	if err == nil {
		if previo.Situacion != solicitud.Destino || previo.ParticipacionRef != solicitud.ParticipacionRef ||
			previo.Motivo != solicitud.Motivo ||
			!mismaFechaDisponible(previo.FechaDisponible, solicitud.FechaDisponible) {
			return puertosbolsa.RegistroSituacionParticipacion{}, dominiobolsa.ErrCambioSituacionParticipacionInvalido
		}
	}
	if err != nil && !errors.Is(err, puertosbolsa.ErrSituacionParticipacionNoEncontrada) {
		return puertosbolsa.RegistroSituacionParticipacion{}, err
	}
	vigente, err := s.repositorio.SituacionVigente(ctx, solicitud.ParticipacionRef)
	if err != nil {
		return puertosbolsa.RegistroSituacionParticipacion{}, err
	}
	ahora := s.reloj().UTC().Truncate(time.Microsecond)
	cambio := dominiobolsa.CambioSituacionParticipacion{ParticipacionRef: solicitud.ParticipacionRef, Origen: vigente.Situacion, Destino: solicitud.Destino, Desde: ahora, Motivo: solicitud.Motivo, FechaDisponible: solicitud.FechaDisponible, RegistradaEn: ahora}
	if !repeticion && (cambio.Validar() != nil || cambio.Desde.Before(vigente.Desde)) {
		return puertosbolsa.RegistroSituacionParticipacion{}, dominiobolsa.ErrCambioSituacionParticipacionInvalido
	}
	if !repeticion {
		if err := s.transicionAdmitida(ctx, cambio.Origen, cambio.Destino); err != nil {
			return puertosbolsa.RegistroSituacionParticipacion{}, err
		}
	}
	h := sha256.Sum256([]byte(solicitud.ParticipacionRef + "\x1f" + solicitud.ClaveIdempotencia))
	recibo := "recibo:situacion:" + hex.EncodeToString(h[:])
	return s.repositorio.RegistrarSituacion(ctx, puertosbolsa.ComandoCambiarSituacionParticipacion{Cambio: cambio, Actor: actor.PersonaRef, BolsaRef: solicitud.BolsaRef, ClaveIdempotencia: solicitud.ClaveIdempotencia, ReciboRef: recibo, SolicitudAutorizacion: auth, Decision: decision, Confirmacion: confirmacion, Material: material})
}

func errorDependenciaSituacion(err error) error {
	if errors.Is(err, dominiovec.ErrAutorizacionDenegada) || errors.Is(err, dominiovec.ErrPermissionDenied) {
		return err
	}
	return ErrCambioSituacionParticipacionNoDisponible
}

func mismaFechaDisponible(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.UTC().Truncate(time.Microsecond).Equal(b.UTC().Truncate(time.Microsecond))
}
