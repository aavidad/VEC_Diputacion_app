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

type ServicioContactoParticipacion struct {
	contexto    puertosbolsa.ResolutorContextoSituacionParticipacion
	autorizador puertosbolsa.AutorizadorSituacionParticipacionV3
	repositorio puertosbolsa.RepositorioContactoParticipacion
	reloj       func() time.Time
}

func NuevoServicioContactoParticipacion(c puertosbolsa.ResolutorContextoSituacionParticipacion, a puertosbolsa.AutorizadorSituacionParticipacionV3, r puertosbolsa.RepositorioContactoParticipacion, reloj func() time.Time) (*ServicioContactoParticipacion, error) {
	if c == nil || a == nil || r == nil || reloj == nil {
		return nil, puertosbolsa.ErrContactoParticipacionNoDisponible
	}
	return &ServicioContactoParticipacion{c, a, r, reloj}, nil
}

func (s *ServicioContactoParticipacion) RegistrarContactoParticipacion(ctx context.Context, solicitud puertosbolsa.SolicitudRegistrarContactoParticipacion) (puertosbolsa.RegistroContactoParticipacion, error) {
	if ctx == nil || s == nil || solicitud.Validar() != nil {
		return puertosbolsa.RegistroContactoParticipacion{}, puertosbolsa.ErrContactoParticipacionNoDisponible
	}
	actor := solicitud.ResultadoContexto.Contexto
	resuelto, err := s.contexto.ResolverContextoSituacionParticipacion(ctx, actor, solicitud.BolsaRef, solicitud.ParticipacionRef)
	if err != nil || resuelto.Validar() != nil || actor.PersonaRef == "" {
		if errors.Is(err, dominiovec.ErrAutorizacionDenegada) || errors.Is(err, dominiovec.ErrPermissionDenied) {
			return puertosbolsa.RegistroContactoParticipacion{}, err
		}
		return puertosbolsa.RegistroContactoParticipacion{}, puertosbolsa.ErrContactoParticipacionNoDisponible
	}
	pertenece, err := s.repositorio.ParticipacionPerteneceABolsa(ctx, solicitud.BolsaRef, solicitud.ParticipacionRef)
	if err != nil {
		return puertosbolsa.RegistroContactoParticipacion{}, err
	}
	if !pertenece {
		return puertosbolsa.RegistroContactoParticipacion{}, dominiovec.ErrAutorizacionDenegada
	}
	h := sha256.Sum256([]byte(solicitud.ParticipacionRef + "\x1f" + solicitud.ClaveIdempotencia))
	sufijo := hex.EncodeToString(h[:])
	contacto := dominiobolsa.ContactoParticipacion{ContactoRef: "contacto:" + sufijo, BolsaRef: solicitud.BolsaRef, ParticipacionRef: solicitud.ParticipacionRef, LlamamientoRef: solicitud.LlamamientoRef, Canal: solicitud.Canal, Instante: solicitud.Instante.UTC().Truncate(time.Microsecond), Actor: actor.PersonaRef, Resultado: solicitud.Resultado, Anotacion: solicitud.Anotacion}
	if contacto.Instante.IsZero() {
		contacto.Instante = s.reloj().UTC().Truncate(time.Microsecond)
	}
	if contacto.Validar() != nil {
		return puertosbolsa.RegistroContactoParticipacion{}, dominiobolsa.ErrContactoParticipacionInvalido
	}
	recurso := dominiovec.RecursoAutorizable{Referencia: solicitud.ParticipacionRef, ModuloID: puertosbolsa.ModuloSituacionParticipacion, Tipo: puertosbolsa.TipoRecursoSituacionParticipacion, Ambitos: map[string]string{"unidad_ref": resuelto.UnidadRef, "ambito_ref": resuelto.AmbitoRef}}
	auth, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(dominiovec.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: solicitud.Vinculo, ReferenciaMotivo: solicitud.MotivoAutorizacion, Accion: puertosbolsa.AccionRegistrarContactoParticipacion, Recurso: recurso, Finalidad: puertosbolsa.FinalidadRegistrarContactoParticipacion, Correlacion: solicitud.Correlacion})
	if err != nil {
		return puertosbolsa.RegistroContactoParticipacion{}, dominiovec.ErrAutorizacionDenegada
	}
	decision, confirmacion, exportador, err := s.autorizador.EmitirMaterialAutorizacionAtestadaV3(ctx, auth, solicitud.ResultadoContexto)
	if err != nil || exportador == nil || decision.ValidarPara(auth) != nil {
		return puertosbolsa.RegistroContactoParticipacion{}, errorDependenciaSituacion(err)
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !materialAutorizacionBorradorLlamamientoExacto(auth, decision, confirmacion, solicitud.ResultadoContexto, solicitud.MotivoAutorizacion, material, puertosbolsa.AudienciaRegistrarContactoParticipacion) {
		return puertosbolsa.RegistroContactoParticipacion{}, errorDependenciaSituacion(err)
	}
	return s.repositorio.RegistrarContacto(ctx, puertosbolsa.ComandoRegistrarContactoParticipacion{Contacto: contacto, ClaveIdempotencia: solicitud.ClaveIdempotencia, ReciboRef: "recibo:contacto:" + sufijo, SolicitudAutorizacion: auth, Decision: decision, Confirmacion: confirmacion, Material: material})
}
func (s *ServicioContactoParticipacion) ListarContactosParticipacion(ctx context.Context, q puertosbolsa.ConsultaContactosParticipacion) (puertosbolsa.PaginaContactosParticipacion, error) {
	if ctx == nil || q.BolsaRef == "" || q.ParticipacionRef == "" || q.Limite < 1 || q.Limite > 100 {
		return puertosbolsa.PaginaContactosParticipacion{}, puertosbolsa.ErrContactoParticipacionNoDisponible
	}
	return s.repositorio.ListarContactosParticipacion(ctx, q)
}
func (s *ServicioContactoParticipacion) ListarContactosBolsa(ctx context.Context, bolsa, cursor string, limite int) (puertosbolsa.PaginaContactosParticipacion, error) {
	if ctx == nil || bolsa == "" || limite < 1 || limite > 100 {
		return puertosbolsa.PaginaContactosParticipacion{}, puertosbolsa.ErrContactoParticipacionNoDisponible
	}
	return s.repositorio.ListarContactosBolsa(ctx, bolsa, cursor, limite)
}
