package ports

import (
	"context"
	"time"

	"vec-diputacion-granada/internal/modules/personal/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// El proveedor entrega material V3 nominal; PostgreSQL vuelve a consumirlo
// dentro de la transacción de consulta o cambio y conserva auditoría.
type ProveedorAutorizacionCatalogosRegistroEmpleadoB2 interface {
	AutorizarCatalogoRegistroEmpleadoB2(context.Context, domain.MaterialCatalogoEmpleadoB2) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

type OrdenCatalogoEmpleadoB2 struct {
	Material     domain.MaterialCatalogoEmpleadoB2
	Autorizacion vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

type CursorCatalogoEmpleadoB2 struct {
	Ref     string `json:"ref"`
	Version int64  `json:"version"`
}

type EvidenciaCatalogoEmpleadoB2 struct {
	DecisionRef         string    `json:"decision_ref"`
	AuditoriaRef        string    `json:"auditoria_ref"`
	ConsumoHuellaSHA256 string    `json:"consumo_huella_sha256"`
	EfectoRef           string    `json:"efecto_ref"`
	ConsultadaEn        time.Time `json:"consultada_en"`
}

type ResultadoConsultaCatalogoEmpleadoB2 struct {
	Entradas        []domain.EntradaCatalogoRegistroEmpleadoB2 `json:"entradas"`
	CursorSiguiente *CursorCatalogoEmpleadoB2                  `json:"cursor_siguiente"`
	Evidencia       EvidenciaCatalogoEmpleadoB2                `json:"evidencia"`
}

type ReciboCatalogoEmpleadoB2 struct {
	DecisionRef         string    `json:"decision_ref"`
	AuditoriaRef        string    `json:"auditoria_ref"`
	ConsumoHuellaSHA256 string    `json:"consumo_huella_sha256"`
	RegistradoEn        time.Time `json:"registrado_en"`
}

type AccesoActualCatalogoEmpleadoB2 struct {
	DecisionRef         string    `json:"decision_ref"`
	AuditoriaRef        string    `json:"auditoria_ref"`
	ConsumoHuellaSHA256 string    `json:"consumo_huella_sha256"`
	RegistradoEn        time.Time `json:"registrado_en"`
	EstadoReplay        string    `json:"estado_replay"`
}

type ResultadoCambioCatalogoEmpleadoB2 struct {
	Entrada      domain.EntradaCatalogoRegistroEmpleadoB2 `json:"entrada"`
	Recibo       ReciboCatalogoEmpleadoB2                 `json:"recibo"`
	AccesoActual AccesoActualCatalogoEmpleadoB2           `json:"acceso_actual"`
}

type RepositorioCatalogosRegistroEmpleadoB2 interface {
	ConsultarRRHH(context.Context, OrdenCatalogoEmpleadoB2) (ResultadoConsultaCatalogoEmpleadoB2, error)
	CambiarRRHH(context.Context, OrdenCatalogoEmpleadoB2) (ResultadoCambioCatalogoEmpleadoB2, error)
}
