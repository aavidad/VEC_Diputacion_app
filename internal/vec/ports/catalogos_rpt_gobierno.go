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
	DecisionRef         string
	EfectoRef           string
	HuellaEfectoSHA256  string
	ConsumoHuellaSHA256 string
	AuditoriaRef        string
	ConsumidaEn         time.Time
	ConsumoNuevo        bool
}

type ResultadoGobiernoCategoriaRPT struct {
	PropuestaRef      string
	HuellaSHA256      string
	Revision          int64
	Estado            string
	ReciboRef         string
	Accion            string
	Version           int
	RevisionCategoria int64
	Evidencia         EvidenciaGobiernoCategoriaRPT
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
