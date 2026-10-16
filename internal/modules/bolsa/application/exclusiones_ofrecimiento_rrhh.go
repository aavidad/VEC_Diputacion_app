package application

import (
	"context"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

type ServicioExclusionesOfrecimientoRRHH struct {
	lector ports.LectorExclusionesOfrecimientoRRHH
}

func NuevoServicioExclusionesOfrecimientoRRHH(lector ports.LectorExclusionesOfrecimientoRRHH) (*ServicioExclusionesOfrecimientoRRHH, error) {
	if lector == nil {
		return nil, ports.ErrExclusionesOfrecimientoRRHHNoDisponibles
	}
	return &ServicioExclusionesOfrecimientoRRHH{lector: lector}, nil
}

func (s *ServicioExclusionesOfrecimientoRRHH) ConsultarExclusionesOfrecimientoRRHH(
	ctx context.Context, q ports.ConsultaExclusionesOfrecimientoRRHH,
) (ports.PaginaExclusionesOfrecimientoRRHH, error) {
	vacia := ports.PaginaExclusionesOfrecimientoRRHH{}
	if s == nil || s.lector == nil || ctx == nil || ctx.Err() != nil ||
		q.BolsaRef == "" || q.LlamamientoRef == "" || q.DesdeOrdenActa < 0 ||
		q.Limite < 1 || q.Limite > ports.LimitePaginaExclusionesOfrecimientoRRHH ||
		q.ResultadoContexto.Validar() != nil || q.Vinculo.ValidarPara(q.ResultadoContexto) != nil ||
		q.Captura.BolsaRef != q.BolsaRef || q.Captura.LlamamientoRef != q.LlamamientoRef {
		return vacia, ports.ErrExclusionesOfrecimientoRRHHNoDisponibles
	}
	r, err := s.lector.ConsultarExclusionesOfrecimientoRRHH(ctx, q)
	if err != nil {
		return vacia, err
	}
	if !r.ValidaPara(q) {
		return vacia, ports.ErrExclusionesOfrecimientoRRHHNoDisponibles
	}
	return r, nil
}
