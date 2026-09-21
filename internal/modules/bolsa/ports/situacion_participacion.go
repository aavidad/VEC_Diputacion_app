package ports

import (
	"context"
	"errors"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	AccionCambiarSituacionParticipacion    = "bolsa.situacion_participacion.cambiar"
	FinalidadCambiarSituacionParticipacion = "gestion_situacion_participacion"
	AudienciaCambiarSituacionParticipacion = "vec_bolsa_llamamientos.situacion_participacion.cambiar.v1"
	ModuloSituacionParticipacion           = "bolsa"
	TipoRecursoSituacionParticipacion      = "participacion_bolsa"
)

var (
	ErrSituacionParticipacionNoDisponible = errors.New("bolsa: situacion de participacion no disponible")
	ErrSituacionParticipacionNoEncontrada = errors.New("bolsa: participacion no encontrada")
)

type SituacionParticipacion struct {
	ParticipacionRef, Situacion string
	Desde                       time.Time
	FechaDisponible             *time.Time
}
type RegistroSituacionParticipacion struct {
	Reutilizada bool
	ReciboRef   string
	Motivo      string
	SituacionParticipacion
}

type ContextoSituacionParticipacionResuelto struct{ UnidadRef, AmbitoRef string }

func (c ContextoSituacionParticipacionResuelto) Validar() error {
	if c.UnidadRef == "" || c.AmbitoRef == "" {
		return ErrSituacionParticipacionNoDisponible
	}
	return nil
}

type SolicitudCambiarSituacionParticipacion struct {
	Vinculo            dominiovec.VinculoAutenticacionActorV2
	ResultadoContexto  dominiovec.ResultadoContextoActorRegistradoV2
	ParticipacionRef   string
	BolsaRef           string
	Destino            string
	Motivo             string
	ClaveIdempotencia  string
	FechaDisponible    *time.Time
	Correlacion        dominiovec.ReferenciaCorrelacionAutorizacionV2
	MotivoAutorizacion dominiovec.ReferenciaEntradaCatalogo
}

func (s SolicitudCambiarSituacionParticipacion) Validar() error {
	if s.ResultadoContexto.Validar() != nil || s.Vinculo.ValidarPara(s.ResultadoContexto) != nil ||
		s.ParticipacionRef == "" || s.BolsaRef == "" || s.Destino == "" || s.Motivo == "" || s.ClaveIdempotencia == "" ||
		s.Correlacion.Validar() != nil || !dominiovec.ReferenciaMotivoAutorizacionV2Valida(s.MotivoAutorizacion) {
		return ErrSituacionParticipacionNoDisponible
	}
	return nil
}

type ComandoCambiarSituacionParticipacion struct {
	Cambio                dominiobolsa.CambioSituacionParticipacion
	Actor                 string
	BolsaRef              string
	ClaveIdempotencia     string
	ReciboRef             string
	SolicitudAutorizacion dominiovec.SolicitudAutorizacionLigadaV3
	Decision              dominiovec.DecisionAutorizacionLigadaV3
	Confirmacion          puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3
	Material              puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

type ResolutorContextoSituacionParticipacion interface {
	ResolverContextoSituacionParticipacion(context.Context, dominiovec.ContextoActor, string, string) (ContextoSituacionParticipacionResuelto, error)
}

type AutorizadorSituacionParticipacionV3 interface {
	EmitirMaterialAutorizacionAtestadaV3(context.Context, dominiovec.SolicitudAutorizacionLigadaV3, dominiovec.ResultadoContextoActorRegistradoV2) (dominiovec.DecisionAutorizacionLigadaV3, puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3, puertosvec.ExportadorMaterialConsumoAutorizacionAtestadaV3, error)
}

type RepositorioSituacionParticipacion interface {
	ParticipacionPerteneceABolsa(context.Context, string, string) (bool, error)
	SituacionVigente(context.Context, string) (SituacionParticipacion, error)
	BuscarRegistroSituacion(context.Context, string, string) (RegistroSituacionParticipacion, error)
	RegistrarSituacion(context.Context, ComandoCambiarSituacionParticipacion) (RegistroSituacionParticipacion, error)
}
