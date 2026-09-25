package ports

import (
	"context"
	"time"

	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

// CambioValorParticipacion es una fila de la traza de valores (petición RRHH
// p.4). Situación y fecha de disponibilidad llevan su valor; los campos de
// contacto llevan la referencia a la versión cifrada ("version:N").
type CambioValorParticipacion struct {
	Instante      time.Time
	ReciboRef     string
	Campo         string
	ValorAnterior *string
	ValorNuevo    *string
	Actor         string
}

// HistorialParticipacion reúne, con una sola autorización consumida, las
// operaciones B8 y la traza de valores de una participación.
type HistorialParticipacion struct {
	Operaciones []RegistroOperacionSituacion
	Cambios     []CambioValorParticipacion
}

type RepositorioHistorialParticipacion interface {
	ListarHistorial(context.Context, string, string, puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3) (HistorialParticipacion, error)
}
