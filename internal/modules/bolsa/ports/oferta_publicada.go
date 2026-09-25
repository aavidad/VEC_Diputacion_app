package ports

import (
	"context"
	"errors"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

// La publicación y la resolución de ofertas son modalidades del llamamiento
// del art. 8.1 del Reglamento: usan la acción, la finalidad y la audiencia de
// la emisión B7 (AccionEmitirLlamamiento) sobre la bolsa constituida.

var (
	ErrOfertaNoDisponible      = errors.New("bolsa: ofertas publicadas no disponibles")
	ErrOfertaInvalida          = errors.New("bolsa: oferta invalida")
	ErrOfertaConflicto         = errors.New("bolsa: clave de oferta reutilizada con otro comando")
	ErrOfertaYaResuelta        = errors.New("bolsa: la oferta ya esta resuelta")
	ErrOfertaPlazoAbierto      = errors.New("bolsa: el plazo de disposicion sigue abierto")
	ErrOfertaPropuestaCambiada = errors.New("bolsa: la propuesta de adjudicacion ha cambiado")
	// ErrPlazoOfertaNoConfigurado: sin catálogo de reglas no hay plazo b10 y
	// no se publica; nunca se inventa un plazo por defecto.
	ErrPlazoOfertaNoConfigurado = errors.New("bolsa: plazo de disposicion sin regla configurada")
)

// PlazoOferta conserva la regla del catálogo que fijó el vencimiento y el
// cálculo hecho con Calendarios en el momento de publicar.
type PlazoOferta struct {
	ReglaRef       string   `json:"regla_ref"`
	HuellaCatalogo string   `json:"huella_catalogo"`
	Unidad         string   `json:"unidad"`
	Cantidad       int      `json:"cantidad"`
	Computo        string   `json:"computo"`
	UltimoDia      string   `json:"ultimo_dia"`
	Ejemplo        bool     `json:"ejemplo"`
	Articulo       string   `json:"articulo,omitempty"`
	Calendarios    []string `json:"calendarios,omitempty"`
}

// CalculadoraPlazoOferta resuelve el plazo de disposición desde la
// publicación. Devuelve ErrPlazoOfertaNoConfigurado si no hay regla.
type CalculadoraPlazoOferta interface {
	PlazoDisposicion(context.Context, time.Time) (PlazoOferta, time.Time, error)
}

type DisposicionOferta struct {
	ParticipacionRef string    `json:"participacion_ref"`
	ManifestadaEn    time.Time `json:"manifestada_en"`
	OrdenVigente     *int64    `json:"orden_vigente"`
	Situacion        *string   `json:"situacion"`
}

type PropuestaOferta struct {
	Tipo             string `json:"tipo"`
	ParticipacionRef string `json:"participacion_ref,omitempty"`
	OrdenVigente     *int64 `json:"orden_vigente,omitempty"`
}

type ResolucionOferta struct {
	ReciboRef          string    `json:"recibo_ref"`
	Tipo               string    `json:"tipo"`
	ParticipacionRef   *string   `json:"participacion_ref"`
	OrdenVigente       *int64    `json:"orden_vigente"`
	DisposicionesTotal int       `json:"disposiciones_total"`
	ResueltaEn         time.Time `json:"resuelta_en"`
}

// OfertaPublicada es la proyección de una oferta en un instante.
type OfertaPublicada struct {
	OfertaRef          string                   `json:"oferta_ref"`
	ReciboRef          string                   `json:"recibo_ref"`
	BolsaRef           string                   `json:"bolsa_ref"`
	Datos              dominiobolsa.DatosOferta `json:"datos"`
	Plazo              PlazoOferta              `json:"plazo"`
	PublicadaEn        time.Time                `json:"publicada_en"`
	VenceAntesDe       time.Time                `json:"vence_antes_de"`
	Estado             string                   `json:"estado"`
	Disposiciones      []DisposicionOferta      `json:"disposiciones"`
	DisposicionesTotal int                      `json:"disposiciones_total"`
	Propuesta          *PropuestaOferta         `json:"propuesta"`
	Resolucion         *ResolucionOferta        `json:"resolucion"`
	Reutilizada        bool                     `json:"reutilizada"`
}

type SolicitudPublicarOferta struct {
	Vinculo            dominiovec.VinculoAutenticacionActorV2
	ResultadoContexto  dominiovec.ResultadoContextoActorRegistradoV2
	BolsaRef           string
	Datos              dominiobolsa.DatosOferta
	ClaveIdempotencia  string
	Correlacion        dominiovec.ReferenciaCorrelacionAutorizacionV2
	MotivoAutorizacion dominiovec.ReferenciaEntradaCatalogo
}

type SolicitudConsultarOfertas struct {
	ContextoActor dominiovec.ContextoActor
	BolsaRef      string
	Limite        int
}

// SolicitudResolverOferta confirma la propuesta vigente: ParticipacionRef
// vacía significa pasar a llamamiento directo.
type SolicitudResolverOferta struct {
	Vinculo            dominiovec.VinculoAutenticacionActorV2
	ResultadoContexto  dominiovec.ResultadoContextoActorRegistradoV2
	BolsaRef           string
	OfertaRef          string
	ParticipacionRef   string
	ClaveIdempotencia  string
	Correlacion        dominiovec.ReferenciaCorrelacionAutorizacionV2
	MotivoAutorizacion dominiovec.ReferenciaEntradaCatalogo
}

type ComandoPublicarOferta struct {
	OfertaRef, ReciboRef, BolsaRef, ActorRef, ClaveIdempotencia string
	Datos                                                       dominiobolsa.DatosOferta
	Plazo                                                       PlazoOferta
	PublicadaEn, VenceAntesDe                                   time.Time
	Material                                                    puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

type ComandoResolverOferta struct {
	OfertaRef, ReciboRef, BolsaRef, ParticipacionRef, ActorRef, ClaveIdempotencia string
	Material                                                                      puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

type RepositorioOfertasPublicadas interface {
	Publicar(context.Context, ComandoPublicarOferta) (OfertaPublicada, error)
	Resolver(context.Context, ComandoResolverOferta) (OfertaPublicada, error)
	Listar(context.Context, string, time.Time, int) ([]OfertaPublicada, error)
}
