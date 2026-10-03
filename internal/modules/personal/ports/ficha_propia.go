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

// IntentoFichaPropia describe un fallo nominal. La identidad, el empleado y la
// correlación proceden de la captura del servidor, nunca de estos datos.
type IntentoFichaPropia struct {
	Motivo string
}

// RegistroIntentosFichaPropia adapta el destino común de auditoría. Se invoca
// después de cerrar la consulta original; un fallo impide devolver sus datos.
type RegistroIntentosFichaPropia interface {
	VerificarRegistroFichaPropia(context.Context) error
	RegistrarIntentoFichaPropia(context.Context, IntentoFichaPropia) error
}
