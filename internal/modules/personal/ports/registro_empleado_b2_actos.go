package ports

import (
	"context"
	"time"

	"vec-diputacion-granada/internal/modules/personal/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// Personal consume V3 y acredita el objetivo B1 en la misma transacción SQL.
type ProveedorAutorizacionActosRegistroEmpleadoB2 interface {
	AutorizarActoRegistroEmpleadoB2(context.Context, domain.MaterialActoRegistroEmpleadoB2) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}
type OrdenAltaEmpleadoB2 struct {
	Material     domain.MaterialActoRegistroEmpleadoB2
	Autorizacion vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}
type OrdenHechoEmpleadoB2 struct {
	Material     domain.MaterialActoRegistroEmpleadoB2
	Autorizacion vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}
type ReciboActoRegistroEmpleadoB2 struct {
	ReciboRef              string    `json:"recibo_ref"`
	EmpleadoRef            string    `json:"empleado_ref"`
	RelacionRef            string    `json:"relacion_ref"`
	ProyeccionRef          string    `json:"proyeccion_ref,omitempty"`
	HechoRef               string    `json:"hecho_ref,omitempty"`
	Tipo                   string    `json:"tipo"`
	Version                int64     `json:"version"`
	EficaciaAdministrativa bool      `json:"eficacia_administrativa"`
	FirmaOficial           bool      `json:"firma_oficial"`
	RegistradoEn           time.Time `json:"registrado_en"`
	DecisionRef            string    `json:"decision_ref"`
	EfectoRef              string    `json:"efecto_ref"`
	ConsumoHuellaSHA256    string    `json:"consumo_huella_sha256"`
	AuditoriaRef           string    `json:"auditoria_ref"`
}
type AccesoActualRegistroEmpleadoB2 struct {
	DecisionRef         string    `json:"decision_ref"`
	EfectoRef           string    `json:"efecto_ref"`
	ConsumoHuellaSHA256 string    `json:"consumo_huella_sha256"`
	AuditoriaRef        string    `json:"auditoria_ref"`
	ConsultadaEn        time.Time `json:"consultada_en"`
	EstadoReplay        string    `json:"estado_replay"`
}
type ResultadoAltaEmpleadoB2 struct {
	Recibo       ReciboActoRegistroEmpleadoB2   `json:"recibo"`
	AccesoActual AccesoActualRegistroEmpleadoB2 `json:"acceso_actual"`
}
type ResultadoHechoEmpleadoB2 struct {
	Recibo       ReciboActoRegistroEmpleadoB2   `json:"recibo"`
	AccesoActual AccesoActualRegistroEmpleadoB2 `json:"acceso_actual"`
}
type RepositorioActosRegistroEmpleadoB2 interface {
	RegistrarEmpleadoRRHH(context.Context, OrdenAltaEmpleadoB2) (ResultadoAltaEmpleadoB2, error)
	RegistrarHechoEmpleadoRRHH(context.Context, OrdenHechoEmpleadoB2) (ResultadoHechoEmpleadoB2, error)
}
