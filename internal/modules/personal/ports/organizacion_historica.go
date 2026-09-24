package ports

import (
	"context"
	"time"

	"vec-diputacion-granada/internal/modules/personal/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// El proveedor debe obtener una concesión central positiva y exacta para el
// material, ámbito, finalidad y perfil. Debe devolver
// domain.ErrConsultaOrganizacionHistoricaDenegada ante denegación explícita,
// auditada en la autoridad central. El repositorio nominal debe consumir esa
// atestación al leer y registrar la auditoría; sin ello no se monta.
type ProveedorAutorizacionConsultaOrganizacionHistorica interface {
	AutorizarConsultaOrganizacionHistorica(context.Context, domain.MaterialConsultaOrganizacionHistorica) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

type OrdenConsultaOrganizacionHistorica struct {
	Material     domain.MaterialConsultaOrganizacionHistorica
	Autorizacion vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

// Cada página comprende como máximo Selector.Limite hechos entre todas las
// colecciones. CursorSiguiente sólo permite continuar el mismo filtro y corte.
// Cobertura enumera cada fuente: completa, parcial o sin_datos; un array vacío
// por sí solo nunca acredita que el universo carezca de elementos.
type CoberturaFuentesOrganizacionHistorica struct {
	Unidades            string `json:"unidades"`
	PuestosTipo         string `json:"puestos_tipo"`
	Dotaciones          string `json:"dotaciones"`
	Plazas              string `json:"plazas"`
	PuestosIndividuales string `json:"puestos_individuales"`
	Vinculos            string `json:"vinculos"`
}

type PaginaOrganizacionHistorica struct {
	Selector            domain.SelectorOrganizacionHistorica           `json:"selector"`
	VersionRPTRef       string                                         `json:"version_rpt_ref"`
	VersionPlantillaRef string                                         `json:"version_plantilla_ref"`
	Cobertura           CoberturaFuentesOrganizacionHistorica          `json:"cobertura"`
	Unidades            []domain.UnidadOrganizacionHistorica           `json:"unidades"`
	PuestosTipo         []domain.PuestoTipoOrganizacionHistorica       `json:"puestos_tipo"`
	Dotaciones          []domain.DotacionOrganizacionHistorica         `json:"dotaciones"`
	Plazas              []domain.PlazaOrganizacionHistorica            `json:"plazas"`
	PuestosIndividuales []domain.PuestoIndividualOrganizacionHistorica `json:"puestos_individuales"`
	Vinculos            []domain.VinculoPlazaPuestoHistorico           `json:"vinculos"`
	CursorSiguiente     string                                         `json:"cursor_siguiente,omitempty"`
}

type EvidenciaConsultaOrganizacionHistorica struct {
	ReciboRef           string    `json:"recibo_ref"`
	DecisionRef         string    `json:"decision_ref"`
	EfectoRef           string    `json:"efecto_ref"`
	ConsumoHuellaSHA256 string    `json:"consumo_huella_sha256"`
	AuditoriaRef        string    `json:"auditoria_ref"`
	ConsultadaEn        time.Time `json:"consultada_en"`
}

type ResultadoConsultaOrganizacionHistorica struct {
	Pagina    PaginaOrganizacionHistorica            `json:"pagina"`
	Evidencia EvidenciaConsultaOrganizacionHistorica `json:"evidencia"`
}

type RepositorioOrganizacionHistorica interface {
	ConsultarOrganizacionHistorica(context.Context, OrdenConsultaOrganizacionHistorica) (ResultadoConsultaOrganizacionHistorica, error)
}
