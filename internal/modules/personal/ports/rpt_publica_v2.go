package ports

import (
	"context"
	"time"

	"vec-diputacion-granada/internal/modules/personal/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type FuenteRPTPublicaV2 interface {
	ObtenerRPTPublicaV2(context.Context) (domain.SnapshotRPTPublicaV2, error)
}

// La identidad procede del canal interno. La implementación usa el PDP y el
// emisor V3 comunes, con acción, audiencia, motivo y capacidad propios de RPT.
type ProveedorAutorizacionRPTPublicaV2 interface {
	AutorizarConsultaRPTPublicaV2(context.Context, domain.MaterialConsultaRPTPublicaV2) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

type OrdenLecturaRPTPublicaV2 struct {
	Material     domain.MaterialConsultaRPTPublicaV2
	Autorizacion vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

type EvidenciaLecturaRPTPublicaV2 struct {
	ReciboRef           string    `json:"recibo_ref"`
	DecisionRef         string    `json:"decision_ref"`
	EfectoRef           string    `json:"efecto_ref"`
	ConsumoHuellaSHA256 string    `json:"consumo_huella_sha256"`
	AuditoriaRef        string    `json:"auditoria_ref"`
	ConsultadaEn        time.Time `json:"consultada_en"`
}

// ConsumirLectura confirma la autorización y la auditoría en una transacción.
// La instantánea del fichero sólo se entrega tras obtener evidencia válida.
type ConsumidorLecturaRPTPublicaV2 interface {
	ConsumirLecturaRPTPublicaV2(context.Context, OrdenLecturaRPTPublicaV2) (EvidenciaLecturaRPTPublicaV2, error)
}

type PaginaRPTPublicaV2 struct {
	Vista                     string                         `json:"vista"`
	Total                     int                            `json:"total"`
	Limite                    int                            `json:"limit"`
	Offset                    int                            `json:"offset"`
	Categorias                []domain.CategoriaRPTPublicaV2 `json:"-"`
	Puestos                   []domain.PuestoRPTPublicoV2    `json:"-"`
	PublicacionRef            string                         `json:"publicacion_ref"`
	Corte                     string                         `json:"corte"`
	HuellaSHA256              string                         `json:"huella_sha256"`
	Estado                    string                         `json:"estado"`
	Fuente                    domain.FuenteRPTPublica        `json:"fuente"`
	Resumen                   domain.ResumenRPTPublica       `json:"resumen"`
	CategoriasPendientesGrupo []string                       `json:"categorias_pendientes_grupo"`
	Evidencia                 EvidenciaLecturaRPTPublicaV2   `json:"evidencia"`
}
