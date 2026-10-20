package ports

import (
	"context"
	"errors"
	"time"

	vd "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

// F1-95: RRHH declara desde la web dónde y cuándo se publicaron oficialmente
// las bases de una versión de convocatoria. El recurso es la versión vigente
// (referencia canónica cv1_<sha256>_v<secuencia>) y sus ámbitos los fija la
// política de gestión publicada para esa versión, nunca el navegador.
const (
	AccionDeclararPublicacionOficialBases      = "bolsa.convocatoria.publicacion_oficial.declarar"
	AudienciaDeclararPublicacionOficialBases   = "vec.bolsa.publicacion_oficial_bases.v1"
	FinalidadDeclararPublicacionOficialBases   = "declarar_publicacion_oficial_bases"
	TipoRecursoVersionConvocatoria             = "version_convocatoria"
	ModuloPublicacionOficialBases              = "bolsa"
	LimiteConsultaPublicacionesOficialesBases  = 100
	MaximoSolicitudPublicacionOficialBasesJSON = 4096
)

var (
	ErrPublicacionOficialInvalida     = errors.New("bolsa: declaracion de publicacion oficial invalida")
	ErrPublicacionOficialConflicto    = errors.New("bolsa: declaracion de publicacion oficial en conflicto")
	ErrPublicacionOficialNoVigente    = errors.New("bolsa: version de convocatoria no vigente")
	ErrPublicacionOficialNoDisponible = errors.New("bolsa: declaracion de publicacion oficial no disponible")
	ErrPublicacionOficialNoConfirmada = errors.New("bolsa: resultado de la declaracion no confirmado")
)

// DeclaracionPublicacionOficial es exactamente el objeto que recibe BC15; el
// orden de los campos no importa porque PostgreSQL lo normaliza como jsonb.
type DeclaracionPublicacionOficial struct {
	AnuncioRef          string  `json:"anuncio_ref"`
	BoletinNumero       *int64  `json:"boletin_numero"`
	ClaveIdempotencia   string  `json:"clave_idempotencia"`
	ConvocatoriaRef     string  `json:"convocatoria_ref"`
	Direccion           *string `json:"direccion"`
	FechaPublicacion    string  `json:"fecha_publicacion"`
	HuellaVersionSHA256 string  `json:"huella_version_sha256"`
	Medio               string  `json:"medio"`
	SustituyeRef        *string `json:"sustituye_ref"`
}

type SolicitudDeclaracionPublicacionOficial struct {
	Vinculo            vd.VinculoAutenticacionActorV2
	ResultadoContexto  vd.ResultadoContextoActorRegistradoV2
	Correlacion        vd.ReferenciaCorrelacionAutorizacionV2
	MotivoAutorizacion vd.ReferenciaEntradaCatalogo
	Declaracion        DeclaracionPublicacionOficial
}

// PublicacionOficialVigente es la declaración que rige hoy para la versión.
type PublicacionOficialVigente struct {
	PublicacionRef   string    `json:"publicacion_ref"`
	Medio            string    `json:"medio"`
	BoletinNumero    *int64    `json:"boletin_numero"`
	AnuncioRef       string    `json:"anuncio_ref"`
	FechaPublicacion string    `json:"fecha_publicacion"`
	Direccion        *string   `json:"direccion"`
	RegistradaEn     time.Time `json:"registrada_en"`
}

// VersionPublicacionOficial es una versión publicada y vigente con su ámbito
// de gestión. UnidadRef y AmbitoRef solo sirven para autorizar; no salen por HTTP.
type VersionPublicacionOficial struct {
	ConvocatoriaRef        string                     `json:"convocatoria_ref"`
	Secuencia              int64                      `json:"secuencia"`
	IdentificadorPublico   string                     `json:"identificador_publico"`
	Titulo                 string                     `json:"titulo"`
	HuellaVersionSHA256    string                     `json:"huella_version_sha256"`
	UnidadRef              string                     `json:"unidad_ref"`
	AmbitoRef              string                     `json:"ambito_ref"`
	PlazoInscripcionAbreEn *time.Time                 `json:"plazo_inscripcion_abre_en"`
	Publicacion            *PublicacionOficialVigente `json:"publicacion"`
}

type PaginaVersionesPublicacionOficial struct {
	Items    []VersionPublicacionOficial `json:"items"`
	Completa bool                        `json:"completa"`
}

type ReciboPublicacionOficial struct {
	PublicacionRef   string    `json:"publicacion_ref"`
	ReciboRef        string    `json:"recibo_ref"`
	EventoRef        string    `json:"evento_ref"`
	ConvocatoriaRef  string    `json:"convocatoria_ref"`
	Secuencia        int64     `json:"secuencia"`
	Medio            string    `json:"medio"`
	BoletinNumero    *int64    `json:"boletin_numero"`
	AnuncioRef       string    `json:"anuncio_ref"`
	FechaPublicacion string    `json:"fecha_publicacion"`
	Direccion        *string   `json:"direccion"`
	SustituyeRef     *string   `json:"sustituye_ref"`
	AuditoriaRef     string    `json:"auditoria_ref"`
	RegistradaEn     time.Time `json:"registrada_en"`
	Reutilizada      bool      `json:"reutilizada"`
}

type IntencionPublicacionOficial struct {
	Solicitud       SolicitudDeclaracionPublicacionOficial
	CanonPostgreSQL string
	HuellaSHA256    string
	Material        vp.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

type RepositorioPublicacionOficial interface {
	// ConsultarVersiones devuelve una versión concreta (ref no vacía) o la
	// lista de versiones vigentes con ámbito de gestión publicado.
	ConsultarVersiones(context.Context, string, int) (PaginaVersionesPublicacionOficial, error)
	NormalizarDeclaracion(context.Context, DeclaracionPublicacionOficial) (string, string, error)
	Declarar(context.Context, IntencionPublicacionOficial) (ReciboPublicacionOficial, error)
}

type ServicioPublicacionOficial interface {
	Declarar(context.Context, SolicitudDeclaracionPublicacionOficial) (ReciboPublicacionOficial, error)
}
