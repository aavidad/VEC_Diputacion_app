package ports

import (
	"context"
	"errors"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

// La oferta SAE es un agregado de Bolsa distinto de la oferta publicada B28.
// Cada acción requiere una concesión V3 propia y su consumo durable en Bolsa.
const (
	ModuloOfertaSAE             = "bolsa"
	TipoRecursoOfertaSAE        = "oferta_sae"
	FinalidadOfertaSAE          = "gestion_seleccion_sae"
	AccionPrepararOfertaSAE     = "bolsa.oferta_sae.preparar"
	AccionConsultarOfertaSAE    = "bolsa.oferta_sae.consultar"
	AudienciaPrepararOfertaSAE  = "vec_bolsa_llamamientos.oferta_sae.preparar.v1"
	AudienciaActuarOfertaSAE    = "vec_bolsa_llamamientos.oferta_sae.actuar.v1"
	AudienciaConsultarOfertaSAE = "vec_bolsa_llamamientos.oferta_sae.consultar.v1"
)

var (
	ErrOfertaSAENoDisponible = errors.New("bolsa: oferta SAE no disponible")
	ErrOfertaSAENoEncontrada = errors.New("bolsa: oferta SAE no encontrada")
)

// AmbitoOfertaSAE procede de la autoridad de contexto/organización existente.
// Nunca se acepta desde JSON ni se deduce del perfil o de la referencia CT.
type AmbitoOfertaSAE struct{ UnidadRef, AmbitoRef string }

func (a AmbitoOfertaSAE) Validar() bool { return a.UnidadRef != "" && a.AmbitoRef != "" }

type ResolutorAmbitoOfertaSAE interface {
	ResolverAmbitoOfertaSAE(context.Context, dominiovec.ContextoActor) (AmbitoOfertaSAE, error)
}

// El catálogo publicado procede del gobierno común y queda fotografiado por
// referencia, versión y huella al preparar la oferta.
type LectorCatalogoOfertaSAE interface {
	CatalogoOfertaSAEVigente(context.Context) (dominiobolsa.CatalogoOfertaSAE, error)
}

type SolicitudPrepararOfertaSAE struct {
	Vinculo            dominiovec.VinculoAutenticacionActorV2
	ResultadoContexto  dominiovec.ResultadoContextoActorRegistradoV2
	Datos              dominiobolsa.DatosOfertaSAE
	ClaveIdempotencia  string
	Correlacion        dominiovec.ReferenciaCorrelacionAutorizacionV2
	MotivoAutorizacion dominiovec.ReferenciaEntradaCatalogo
}

type SolicitudActuarOfertaSAE struct {
	Vinculo            dominiovec.VinculoAutenticacionActorV2
	ResultadoContexto  dominiovec.ResultadoContextoActorRegistradoV2
	OfertaRef          string
	Cambio             dominiobolsa.CambioOfertaSAE
	Correlacion        dominiovec.ReferenciaCorrelacionAutorizacionV2
	MotivoAutorizacion dominiovec.ReferenciaEntradaCatalogo
}

type SolicitudConsultarOfertaSAE struct {
	Vinculo            dominiovec.VinculoAutenticacionActorV2
	ResultadoContexto  dominiovec.ResultadoContextoActorRegistradoV2
	OfertaRef          string
	Correlacion        dominiovec.ReferenciaCorrelacionAutorizacionV2
	MotivoAutorizacion dominiovec.ReferenciaEntradaCatalogo
}

type ReciboOfertaSAE struct {
	Oferta      dominiobolsa.OfertaSAE
	ReciboRef   string
	Reutilizado bool
}

type AutorizacionOfertaSAE struct {
	Solicitud    dominiovec.SolicitudAutorizacionLigadaV3
	Decision     dominiovec.DecisionAutorizacionLigadaV3
	Confirmacion puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3
	Material     puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

type OrdenPrepararOfertaSAE struct {
	Oferta                                 dominiobolsa.OfertaSAE
	ActorRef, ClaveIdempotencia, ReciboRef string
	Ambito                                 AmbitoOfertaSAE
	Autorizacion                           AutorizacionOfertaSAE
}
type OrdenActuarOfertaSAE struct {
	OfertaRef    string
	Cambio       dominiobolsa.CambioOfertaSAE
	Ambito       AmbitoOfertaSAE
	Autorizacion AutorizacionOfertaSAE
}
type OrdenConsultarOfertaSAE struct {
	OfertaRef    string
	Ambito       AmbitoOfertaSAE
	Autorizacion AutorizacionOfertaSAE
}

// El adaptador durable carga y valida el agregado en la misma transacción que
// consume la autorización, incrementa OCC y escribe historia/auditoría/outbox.
// En replay devuelve el recibo previo sin duplicar ninguno de esos efectos.
type RepositorioOfertaSAE interface {
	Preparar(context.Context, OrdenPrepararOfertaSAE) (ReciboOfertaSAE, error)
	Actuar(context.Context, OrdenActuarOfertaSAE) (ReciboOfertaSAE, error)
	Consultar(context.Context, OrdenConsultarOfertaSAE) (dominiobolsa.OfertaSAE, error)
}
