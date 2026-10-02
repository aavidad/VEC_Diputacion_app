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
	// ErrOfertaRespuestaAbierta: la persona aún está en plazo para responder.
	ErrOfertaRespuestaAbierta = errors.New("bolsa: el plazo de respuesta de la plaza sigue abierto")
	// ErrOfertaPoliticaSinPlazas: la versión de política no tiene el apartado
	// de plazas y la oferta pide más de una.
	ErrOfertaPoliticaSinPlazas = errors.New("bolsa: la politica de la bolsa no admite varias plazas")
	// ErrPlazoOfertaNoConfigurado: sin catálogo de reglas no hay plazo b10 y
	// no se publica; nunca se inventa un plazo por defecto.
	ErrPlazoOfertaNoConfigurado = errors.New("bolsa: plazo de disposicion sin regla configurada")
)

// PlazoOferta conserva la regla del catálogo que fijó el vencimiento y el
// cálculo hecho con Calendarios en el momento de publicar.
type PlazoOferta struct {
	Notificacion    *dominiobolsa.NotificacionOferta `json:"notificacion,omitempty"`
	ReglaRef        string                           `json:"regla_ref"`
	HuellaCatalogo  string                           `json:"huella_catalogo"`
	Unidad          string                           `json:"unidad"`
	Cantidad        int                              `json:"cantidad"`
	Computo         string                           `json:"computo"`
	UltimoDia       string                           `json:"ultimo_dia"`
	Ejemplo         bool                             `json:"ejemplo"`
	Articulo        string                           `json:"articulo,omitempty"`
	Calendarios     []string                         `json:"calendarios,omitempty"`
	PoliticaVersion int64                            `json:"politica_version,omitempty"`
	MunicipioSede   string                           `json:"municipio_sede,omitempty"`
	// En horas naturales estos dos instantes UTC forman parte del recibo y
	// del material autorizado. UltimoDia queda solo como ayuda de presentación.
	AperturaEn string `json:"apertura_en,omitempty"`
	VenceEn    string `json:"vence_en,omitempty"`
}

// CalculadoraPlazoOferta resuelve el plazo de disposición desde la
// notificación acreditada por RRHH. Devuelve ErrPlazoOfertaNoConfigurado
// si no hay regla.
type CalculadoraPlazoOferta interface {
	PlazoDisposicion(context.Context, time.Time) (PlazoOferta, time.Time, error)
}

// CalculadoraPlazoOfertaPorBolsa selecciona la versión de política propia de
// la bolsa. La interfaz anterior sigue sirviendo a pruebas y ofertas previas.
type CalculadoraPlazoOfertaPorBolsa interface {
	PlazoDisposicionBolsa(context.Context, string, time.Time) (PlazoOferta, time.Time, error)
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

// ActoPlazaOferta es un paso de la historia de una plaza, sin actor.
type ActoPlazaOferta struct {
	Secuencia    int       `json:"secuencia"`
	Tipo         string    `json:"tipo"`
	OrdenVigente *int64    `json:"orden_vigente"`
	RegistradoEn time.Time `json:"registrado_en"`
	ReciboRef    string    `json:"recibo_ref"`
}

// PlazaOferta es el estado de una plaza en un instante: quién la ocupa, hasta
// cuándo puede responder, qué propone VEC y su historia. Secuencia es la
// versión que debe confirmar el siguiente acto.
type PlazaOferta struct {
	NumeroDePlaza     int               `json:"numero_de_plaza"`
	Estado            string            `json:"estado"`
	Secuencia         int               `json:"secuencia"`
	ParticipacionRef  *string           `json:"participacion_ref"`
	OrdenVigente      *int64            `json:"orden_vigente"`
	ResponderAntesDe  *time.Time        `json:"responder_antes_de"`
	PuedeSinRespuesta bool              `json:"puede_sin_respuesta"`
	Propuesta         *PropuestaOferta  `json:"propuesta"`
	Historial         []ActoPlazaOferta `json:"historial"`
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
	// ConfirmacionAdjudicacion procede de la versión de política inmovilizada
	// por la oferta. Vacía en políticas anteriores a la aceptación previa.
	ConfirmacionAdjudicacion string `json:"confirmacion_adjudicacion,omitempty"`
	// NumeroPlazas, la política de plazas aplicada (nula si la versión no la
	// tiene) y el estado de cada plaza.
	NumeroPlazas   int                                 `json:"numero_plazas"`
	PoliticaPlazas *dominiobolsa.PlazasPoliticaOfertas `json:"politica_plazas"`
	Plazas         []PlazaOferta                       `json:"plazas"`
	Reutilizada    bool                                `json:"reutilizada"`
}

type SolicitudPublicarOferta struct {
	Notificacion       dominiobolsa.NotificacionOferta
	Vinculo            dominiovec.VinculoAutenticacionActorV2
	ResultadoContexto  dominiovec.ResultadoContextoActorRegistradoV2
	BolsaRef           string
	Datos              dominiobolsa.DatosOferta
	NumeroPlazas       int
	ClaveIdempotencia  string
	Correlacion        dominiovec.ReferenciaCorrelacionAutorizacionV2
	MotivoAutorizacion dominiovec.ReferenciaEntradaCatalogo
}

type SolicitudConsultarOfertas struct {
	ContextoActor dominiovec.ContextoActor
	BolsaRef      string
	Limite        int
}

// SolicitudResolverOferta registra un acto sobre una plaza: confirmar la
// propuesta (adjudicada o llamamiento_directo) o la respuesta de quien la
// ocupa (aceptada, renuncia o sin_respuesta). ParticipacionRef va vacía solo
// en el llamamiento directo. SecuenciaEsperada es la versión de la plaza que
// vio RRHH.
type SolicitudResolverOferta struct {
	Vinculo            dominiovec.VinculoAutenticacionActorV2
	ResultadoContexto  dominiovec.ResultadoContextoActorRegistradoV2
	BolsaRef           string
	OfertaRef          string
	NumeroDePlaza      int
	Tipo               string
	SecuenciaEsperada  int
	ParticipacionRef   string
	ClaveIdempotencia  string
	Correlacion        dominiovec.ReferenciaCorrelacionAutorizacionV2
	MotivoAutorizacion dominiovec.ReferenciaEntradaCatalogo
}

type ComandoPublicarOferta struct {
	OfertaRef, ReciboRef, BolsaRef, ActorRef, ClaveIdempotencia string
	UnidadRef, AmbitoRef                                        string
	Datos                                                       dominiobolsa.DatosOferta
	NumeroPlazas                                                int
	Plazo                                                       PlazoOferta
	PublicadaEn, VenceAntesDe                                   time.Time
	Material                                                    puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

type ComandoResolverOferta struct {
	OfertaRef, ReciboRef, BolsaRef, ParticipacionRef, ActorRef, ClaveIdempotencia string
	Tipo                                                                          string
	NumeroDePlaza, SecuenciaEsperada                                              int
	Material                                                                      puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

type RepositorioOfertasPublicadas interface {
	Publicar(context.Context, ComandoPublicarOferta) (OfertaPublicada, error)
	Resolver(context.Context, ComandoResolverOferta) (OfertaPublicada, error)
	Listar(context.Context, string, time.Time, int) ([]OfertaPublicada, error)
}
