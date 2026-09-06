package ports

import (
	"context"
	"errors"
)

// ErrInformeDefinitivoRRHHNoDisponible indica que el detalle autorizado no
// cumple las precondiciones del borrador; no equivale a denegar la consulta.
var ErrInformeDefinitivoRRHHNoDisponible = errors.New("contratacion temporal: informe definitivo RRHH no disponible")

// RenderizadorInformeDefinitivoRRHH transforma únicamente el detalle reducido
// ya autorizado. No consulta otra fuente, concede permisos ni acredita firma.
type RenderizadorInformeDefinitivoRRHH interface {
	RenderizarInforme(context.Context, DetalleExpedienteRRHH) ([]byte, error)
}
