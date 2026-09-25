package ports

import (
	"context"

	"vec-diputacion-granada/internal/modules/personal/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// OrdenEmpleadosB2 liga la lista a la misma concesión V3 que se consume en
// la transacción de lectura. El organismo lo fija el servidor, nunca HTTP.
type OrdenEmpleadosB2 struct {
	Material     domain.MaterialConsultaRegistroEmpleadoB2
	Autorizacion vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

type ResultadoEmpleadosB2 struct {
	Pagina    domain.PaginaEmpleadosB2    `json:"pagina"`
	Evidencia EvidenciaRegistroEmpleadoB2 `json:"evidencia"`
}

type RepositorioEmpleadosRegistroB2 interface {
	ListarEmpleadosRRHH(context.Context, OrdenEmpleadosB2) (ResultadoEmpleadosB2, error)
}
