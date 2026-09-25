package ports

import (
	"context"

	"vec-diputacion-granada/internal/modules/personal/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// ProveedorAutorizacionFichaPropia pide a la autoridad común V3 una concesión
// positiva, nominal y vigente para la persona de la petición en curso; nunca
// para un empleado indicado por el cliente.
type ProveedorAutorizacionFichaPropia interface {
	AutorizarFichaPropia(context.Context, domain.MaterialFichaPropia) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

type OrdenFichaPropia struct {
	Material     domain.MaterialFichaPropia
	Autorizacion vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

type ResultadoFichaPropia struct {
	Ficha     domain.FichaPropia          `json:"ficha"`
	Evidencia EvidenciaRegistroEmpleadoB2 `json:"evidencia"`
}

// RepositorioFichaPropia consume la concesión, comprueba que el empleado es
// el canónico de la persona y lee su ficha en una misma transacción.
type RepositorioFichaPropia interface {
	ConsultarFichaPropia(context.Context, OrdenFichaPropia) (ResultadoFichaPropia, error)
}

// DenegacionFichaPropia es el hecho minimizado de una denegación de la
// frontera HTTP. ActorRef solo se informa con la identidad ya acreditada.
type DenegacionFichaPropia struct {
	CorrelacionRef string
	Motivo         string
	EstadoHTTP     int
	ActorRef       string
}

type RegistroDenegacionFichaPropia interface {
	RegistrarDenegacionFichaPropia(context.Context, DenegacionFichaPropia) error
}
