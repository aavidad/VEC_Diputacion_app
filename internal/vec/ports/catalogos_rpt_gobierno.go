package ports

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrGobiernoCategoriaRPTInvalido     = errors.New("vec: acto de gobierno RPT invalido")
	ErrGobiernoCategoriaRPTDenegado     = errors.New("vec: acto de gobierno RPT denegado")
	ErrGobiernoCategoriaRPTConflicto    = errors.New("vec: acto de gobierno RPT en conflicto")
	ErrGobiernoCategoriaRPTNoDisponible = errors.New("vec: gobierno RPT no disponible")
	ErrGobiernoCategoriaRPTNoConfiable  = errors.New("vec: recibo de gobierno RPT no confiable")
)

const (
	AccionProponerGobiernoCategoriaRPT  = "vec.catalogos.categorias.gobierno.proponer"
	AccionAprobarGobiernoCategoriaRPT   = "vec.catalogos.categorias.gobierno.aprobar"
	AccionConfirmarGobiernoCategoriaRPT = "vec.catalogos.categorias.gobierno.confirmar"
	AudienciaGobiernoCategoriaRPT       = "vec_catalogos_configurables.gobierno_categorias.v1"
	FinalidadGobiernoCategoriaRPT       = "gobernar_categorias_rpt"
	TipoRecursoGobiernoCategoriaRPT     = "propuesta_categoria"
)

// EntradaPropuestaGobiernoCategoriaRPT es el contrato HTTP de propuesta. Las referencias
// de actor, motivo, cuenta, perfil y autorización se resuelven en el servidor.
// La clave del recibo procede de Idempotency-Key. Aprobar usa revisión 1;
// confirmar usa revisión 2 en EntradaAvanceGobiernoCategoriaRPT. El adaptador
// rechaza campos ajenos a cada acción, duplicados y referencias de autoridad.
type EntradaPropuestaGobiernoCategoriaRPT struct {
	PropuestaRef       string                                                 `json:"propuesta_ref"`
	Accion             string                                                 `json:"accion"`
	CatalogoID         string                                                 `json:"catalogo_id"`
	ModuloID           string                                                 `json:"modulo_id"`
	Version            int                                                    `json:"version"`
	DocumentoCanonico  *string                                                `json:"documento_canonico"`
	PreimagenesControl map[string]domain.PreimagenControlGobiernoCategoriaRPT `json:"preimagenes_control"`
	CategoriaID        *string                                                `json:"categoria_id"`
	RevisionEsperada   *int64                                                 `json:"revision_esperada"`
	FuenteRef          string                                                 `json:"fuente_ref"`
}

type EntradaAvanceGobiernoCategoriaRPT struct {
	PropuestaRef     string `json:"propuesta_ref"`
	CatalogoID       string `json:"catalogo_id"`
	ModuloID         string `json:"modulo_id"`
	HuellaSHA256     string `json:"huella_sha256"`
	RevisionEsperada int64  `json:"revision_esperada"`
}

type RespuestaGobiernoCategoriaRPT struct {
	Data ResultadoGobiernoCategoriaRPT `json:"data"`
}

type MaterialPropuestaGobiernoCategoriaRPT struct {
	PropuestaRef string                               `json:"propuesta_ref"`
	Contenido    domain.ContenidoGobiernoCategoriaRPT `json:"contenido"`
	HuellaSHA256 string                               `json:"huella_sha256"`
	ReciboRef    string                               `json:"recibo_ref"`
}

// El borrador no lleva huellas de JSONB. El preparador las calcula en
// PostgreSQL sin lectura privada y devuelve el material final exacto.
type BorradorPropuestaGobiernoCategoriaRPT struct {
	PropuestaRef string
	Contenido    domain.ContenidoGobiernoCategoriaRPT
	ReciboRef    string
}

type MaterialAvanceGobiernoCategoriaRPT struct {
	PropuestaRef     string `json:"propuesta_ref"`
	HuellaSHA256     string `json:"huella_sha256"`
	ReciboRef        string `json:"recibo_ref"`
	RevisionEsperada int64  `json:"revision_esperada"`
	CatalogoID       string `json:"catalogo_id"`
	ModuloID         string `json:"modulo_id"`
}

// La preparacion calcula el contexto exacto en PostgreSQL sin consultar tablas
// privadas ni conceder autoridad. El adaptador compara de nuevo dentro de la
// transaccion de efecto el JSONB recibido y consume una decision V3 nueva.
type PreparacionGobiernoCategoriaRPT struct {
	Accion          string
	Finalidad       string
	Audiencia       string
	Recurso         domain.RecursoAutorizable
	HuellaPropuesta string
}

type PreparacionPropuestaGobiernoCategoriaRPT struct {
	Material    MaterialPropuestaGobiernoCategoriaRPT
	Autorizable PreparacionGobiernoCategoriaRPT
}

type OrdenPropuestaGobiernoCategoriaRPT struct {
	Material     MaterialPropuestaGobiernoCategoriaRPT
	Solicitud    domain.SolicitudAutorizacionLigadaV3
	Autorizacion ExportacionMaterialConsumoAutorizacionAtestadaV3
}

type OrdenAvanceGobiernoCategoriaRPT struct {
	Material     MaterialAvanceGobiernoCategoriaRPT
	Solicitud    domain.SolicitudAutorizacionLigadaV3
	Autorizacion ExportacionMaterialConsumoAutorizacionAtestadaV3
}

type EvidenciaGobiernoCategoriaRPT struct {
	DecisionRef         string    `json:"decision_ref"`
	EfectoRef           string    `json:"efecto_ref"`
	HuellaEfectoSHA256  string    `json:"huella_efecto_sha256"`
	ConsumoHuellaSHA256 string    `json:"consumo_huella_sha256"`
	AuditoriaRef        string    `json:"auditoria_ref"`
	ConsumidaEn         time.Time `json:"consumida_en"`
	ConsumoNuevo        bool      `json:"consumo_nuevo"`
}

type ResultadoGobiernoCategoriaRPT struct {
	PropuestaRef      string                        `json:"propuesta_ref"`
	HuellaSHA256      string                        `json:"huella_sha256"`
	Revision          int64                         `json:"revision"`
	Estado            string                        `json:"estado"`
	ReciboRef         string                        `json:"recibo_ref"`
	Accion            string                        `json:"accion"`
	Version           int                           `json:"version"`
	RevisionCategoria int64                         `json:"revision_categoria"`
	Evidencia         EvidenciaGobiernoCategoriaRPT `json:"evidencia"`
}

type PreparadorGobiernoCategoriaRPT interface {
	PrepararPropuestaGobiernoCategoriaRPT(context.Context, BorradorPropuestaGobiernoCategoriaRPT) (PreparacionPropuestaGobiernoCategoriaRPT, error)
	PrepararAprobacionGobiernoCategoriaRPT(context.Context, MaterialAvanceGobiernoCategoriaRPT) (PreparacionGobiernoCategoriaRPT, error)
	PrepararConfirmacionGobiernoCategoriaRPT(context.Context, MaterialAvanceGobiernoCategoriaRPT) (PreparacionGobiernoCategoriaRPT, error)
}

type GestorGobiernoCategoriaRPT interface {
	ProponerGobiernoCategoriaRPT(context.Context, OrdenPropuestaGobiernoCategoriaRPT) (ResultadoGobiernoCategoriaRPT, error)
	AprobarGobiernoCategoriaRPT(context.Context, OrdenAvanceGobiernoCategoriaRPT) (ResultadoGobiernoCategoriaRPT, error)
	ConfirmarGobiernoCategoriaRPT(context.Context, OrdenAvanceGobiernoCategoriaRPT) (ResultadoGobiernoCategoriaRPT, error)
}

type AutorizadorGobiernoCategoriaRPT interface {
	EmitirMaterialAutorizacionAtestadaV3(context.Context, domain.SolicitudAutorizacionLigadaV3, domain.ResultadoContextoActorRegistradoV2) (domain.DecisionAutorizacionLigadaV3, ConfirmacionRegistroConcesionAutorizacionLigadaV3, ExportadorMaterialConsumoAutorizacionAtestadaV3, error)
}
