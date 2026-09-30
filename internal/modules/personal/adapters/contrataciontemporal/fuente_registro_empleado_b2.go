package contrataciontemporal

import (
	"context"

	"vec-diputacion-granada/internal/modules/personal/application"
	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
)

// FuenteRegistroEmpleadoB2 usa el servicio propietario de consulta nominal.
// No lee tablas, conserva caches ni convierte recibos B2 en altas de ejercicio.
type FuenteRegistroEmpleadoB2 struct {
	servicio *application.ServicioRegistroEmpleadoB2
}

func NuevaFuenteRegistroEmpleadoB2(s *application.ServicioRegistroEmpleadoB2) (*FuenteRegistroEmpleadoB2, error) {
	if s == nil {
		return nil, domain.ErrRegistroEmpleadoB2NoDisponible
	}
	return &FuenteRegistroEmpleadoB2{servicio: s}, nil
}

func (f *FuenteRegistroEmpleadoB2) ConsultarFicha(ctx context.Context, s domain.SolicitudFichaEmpleadoB2) (ports.ResultadoFichaEmpleadoB2, error) {
	if f == nil || f.servicio == nil {
		return ports.ResultadoFichaEmpleadoB2{}, domain.ErrRegistroEmpleadoB2NoDisponible
	}
	return f.servicio.ConsultarFicha(ctx, s)
}

var _ ports.FuenteFichaIncorporacionCT = (*FuenteRegistroEmpleadoB2)(nil)
