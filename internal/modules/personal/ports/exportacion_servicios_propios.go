package ports

import (
	"context"
	"vec-diputacion-granada/internal/modules/personal/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type ProveedorFormatoExportacionServiciosPropios interface {
	FormatoParaIdioma(context.Context, string) (domain.FormatoExportacionServiciosPropios, error)
}
type ProveedorAutorizacionExportacionServiciosPropios interface {
	AutorizarExportacionServiciosPropios(context.Context, domain.MaterialExportacionServiciosPropios) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}
type OrdenExportacionServiciosPropios struct {
	Material     domain.MaterialExportacionServiciosPropios
	Autorizacion vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

// La evidencia acredita generación autorizada del CSV, no su recepción ni
// guardado por la persona. El recibo original sigue siendo sólo el antecedente.
type ResultadoExportacionServiciosPropios struct {
	Corte                          domain.CorteEmpleadoB2
	ContenidoCSV                   []byte
	NombreArchivo, ContenidoSHA256 string
	Evidencia                      EvidenciaRegistroEmpleadoB2
}

// El repositorio revalida el recibo/corte y consume V3 propio. La decodificación
// y serialización CSV terminan antes de confirmar la transacción de generación.
type RepositorioExportacionServiciosPropios interface {
	ExportarServiciosPropios(context.Context, OrdenExportacionServiciosPropios) (ResultadoExportacionServiciosPropios, error)
}
type RegistroIntentosExportacionServiciosPropios interface {
	VerificarRegistroExportacionServiciosPropios(context.Context) error
	RegistrarIntentoExportacionServiciosPropios(context.Context, IntentoFichaPropia) error
}
