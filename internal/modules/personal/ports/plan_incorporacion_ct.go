package ports

import (
	"context"
	"vec-diputacion-granada/internal/modules/personal/domain"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type SelectorOrganizacionPlanCT = domain.SelectorOrganizacionPlanCT
type SeleccionPlanIncorporacionCT = domain.SolicitudSeleccionPlanIncorporacionCT
type ResultadoSeleccionPlanIncorporacionCT struct {
	Seleccion domain.SeleccionOrganizacionPlanCT `json:"seleccion"`
	Evidencia EvidenciaRegistroEmpleadoB2        `json:"evidencia"`
}

type ConsultaClasesOcupacionCT = domain.ConsultaClasesOcupacionCT
type ResultadoClasesOcupacionCT struct {
	Catalogo  domain.CatalogoClasesOcupacionCT `json:"catalogo"`
	Evidencia EvidenciaRegistroEmpleadoB2      `json:"evidencia"`
}
type ServicioClasesOcupacionCT interface {
	ConsultarClasesOcupacion(context.Context, ConsultaClasesOcupacionCT) (ResultadoClasesOcupacionCT, error)
}

type DatosPlanIncorporacionCT = domain.DatosPlanIncorporacionCT
type SolicitudPlanIncorporacionCT = domain.SolicitudPlanIncorporacionCT
type ConsultaPlanIncorporacionCT = domain.ConsultaPlanIncorporacionCT
type PlanIncorporacionCT = domain.PlanIncorporacionCT

type EstadoPlanIncorporacionCT struct {
	Plan                  PlanIncorporacionCT           `json:"plan"`
	Estado                string                        `json:"estado"`
	ReciboAltaRelacion    *ReciboActoRegistroEmpleadoB2 `json:"recibo_alta_relacion"`
	ReciboOcupacion       *ReciboActoRegistroEmpleadoB2 `json:"recibo_ocupacion"`
	EjecucionReciboRef    string                        `json:"ejecucion_recibo_ref"`
	EjecucionHuellaSHA256 string                        `json:"ejecucion_huella_sha256"`
	Evidencia             EvidenciaRegistroEmpleadoB2   `json:"evidencia"`
}
type ServicioPlanIncorporacionCT interface {
	ResolverSeleccion(context.Context, SeleccionPlanIncorporacionCT) (ResultadoSeleccionPlanIncorporacionCT, error)
	PrepararPlan(context.Context, SolicitudPlanIncorporacionCT) (EstadoPlanIncorporacionCT, error)
	ConsultarPlan(context.Context, ConsultaPlanIncorporacionCT) (EstadoPlanIncorporacionCT, error)
	EjecutarPlan(context.Context, ConsultaPlanIncorporacionCT) (EstadoPlanIncorporacionCT, error)
}
type ProveedorAutorizacionPlanIncorporacionCT interface {
	AutorizarPlanIncorporacionCT(context.Context, domain.MaterialPlanIncorporacionCT) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}
type OrdenPlanIncorporacionCT struct {
	Material     domain.MaterialPlanIncorporacionCT
	Autorizacion vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}
type RepositorioPlanIncorporacionCT interface {
	ConsultarClasesOcupacion(context.Context, OrdenPlanIncorporacionCT) (ResultadoClasesOcupacionCT, error)
	ResolverSeleccion(context.Context, OrdenPlanIncorporacionCT) (ResultadoSeleccionPlanIncorporacionCT, error)
	PrepararPlan(context.Context, OrdenPlanIncorporacionCT) (EstadoPlanIncorporacionCT, error)
	ConsultarPlan(context.Context, OrdenPlanIncorporacionCT) (EstadoPlanIncorporacionCT, error)
	ConfirmarPlan(context.Context, OrdenPlanIncorporacionCT) (EstadoPlanIncorporacionCT, error)
}

// La acreditación se obtiene del gestor propietario de usos RPT, en servidor.
// Las referencias declaradas por el navegador no constituyen esta evidencia.
type EvidenciaReservaRPTPlanCT struct {
	UsoRPTRef        string
	ReservaRPTRef    string
	PlanHuellaSHA256 string
	PlazaRef         string
	PuestoRef        string
	CatalogoID       string
	Modulo           string
	Categoria        string
	Version          int64
	HuellaSHA256     string
	ReciboRef        string
}
type FuenteReservaRPTPlanCT interface {
	AcreditarReservaPlanCT(context.Context, PlanIncorporacionCT, core.ContextoActor) (EvidenciaReservaRPTPlanCT, error)
}
type ActosPlanIncorporacionCT interface {
	RegistrarEmpleado(context.Context, domain.SolicitudAltaEmpleadoB2) (ResultadoAltaEmpleadoB2, error)
	RegistrarHecho(context.Context, domain.SolicitudHechoEmpleadoB2) (ResultadoHechoEmpleadoB2, error)
}
