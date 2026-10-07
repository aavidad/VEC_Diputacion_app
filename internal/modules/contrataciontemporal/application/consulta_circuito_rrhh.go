package application

import (
	"context"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type ServicioConsultaCircuitoRRHH struct {
	preparador ports.PreparadorConsultaCircuitoRRHH
	lector     ports.LectorCircuitoRRHH
}

func NuevoServicioConsultaCircuitoRRHH(preparador ports.PreparadorConsultaCircuitoRRHH, lector ports.LectorCircuitoRRHH) (*ServicioConsultaCircuitoRRHH, error) {
	if dependenciaNula(preparador) || dependenciaNula(lector) {
		return nil, ports.ErrConsultaCircuitoRRHHNoDisponible
	}
	return &ServicioConsultaCircuitoRRHH{preparador: preparador, lector: lector}, nil
}

func (s *ServicioConsultaCircuitoRRHH) Consultar(ctx context.Context, solicitud ports.SolicitudConsultaCircuitoRRHH) (ports.ResultadoConsultaCircuitoRRHH, error) {
	vacio := ports.ResultadoConsultaCircuitoRRHH{}
	if s == nil || ctx == nil || dependenciaNula(s.preparador) || dependenciaNula(s.lector) {
		return vacio, ports.ErrConsultaCircuitoRRHHNoDisponible
	}
	if solicitud.Validar() != nil {
		return vacio, ports.ErrConsultaCircuitoRRHHInvalida
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	material, err := s.preparador.PrepararConsultaCircuitoRRHH(ctx, solicitud)
	if err != nil {
		return vacio, err
	}
	if material.ValidarPara(solicitud) != nil {
		return vacio, ports.ErrConsultaCircuitoRRHHDenegada
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	resultado, err := s.lector.ConsultarCircuitoRRHH(ctx, material)
	if errContexto := ctx.Err(); errContexto != nil {
		return vacio, errContexto
	}
	if err != nil {
		return vacio, err
	}
	if resultado.ValidarPara(solicitud) != nil {
		return vacio, ports.ErrResultadoCircuitoRRHHNoConfiable
	}
	return resultado, nil
}
