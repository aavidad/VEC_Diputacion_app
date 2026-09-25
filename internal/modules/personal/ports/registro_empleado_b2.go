package ports

import (
	"context"
	"time"

	"vec-diputacion-granada/internal/modules/personal/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// El proveedor V3 solicita una concesión central positiva, nominal y vigente.
// El repositorio consume esa misma atestación en la transacción de lectura y
// registra auditoría. Ninguno de los dos infiere identidad del selector HTTP.
type ProveedorAutorizacionRegistroEmpleadoB2 interface {
	AutorizarConsultaRegistroEmpleadoB2(context.Context, domain.MaterialConsultaRegistroEmpleadoB2) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

type OrdenFichaEmpleadoB2 struct {
	Material     domain.MaterialConsultaRegistroEmpleadoB2
	Autorizacion vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}
type OrdenVacantesB2 struct {
	Material     domain.MaterialConsultaRegistroEmpleadoB2
	Autorizacion vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}
type EvidenciaRegistroEmpleadoB2 struct {
	ReciboRef           string    `json:"recibo_ref"`
	DecisionRef         string    `json:"decision_ref"`
	EfectoRef           string    `json:"efecto_ref"`
	ConsumoHuellaSHA256 string    `json:"consumo_huella_sha256"`
	AuditoriaRef        string    `json:"auditoria_ref"`
	ConsultadaEn        time.Time `json:"consultada_en"`
}
type ResultadoFichaEmpleadoB2 struct {
	Ficha     domain.FichaEmpleadoB2      `json:"ficha"`
	Evidencia EvidenciaRegistroEmpleadoB2 `json:"evidencia"`
}
type ResultadoVacantesB2 struct {
	Pagina    domain.PaginaVacantesB2     `json:"pagina"`
	Evidencia EvidenciaRegistroEmpleadoB2 `json:"evidencia"`
}
type RepositorioRegistroEmpleadoB2 interface {
	ConsultarFichaRRHH(context.Context, OrdenFichaEmpleadoB2) (ResultadoFichaEmpleadoB2, error)
	ListarVacantesRRHH(context.Context, OrdenVacantesB2) (ResultadoVacantesB2, error)
}
