package application

import (
	"context"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type ServicioDerivacionNombramiento struct {
	preparador  ports.PreparadorDerivacionNombramiento
	autorizador ports.AutorizadorDerivacionNombramiento
	transaccion ports.TransaccionDerivacionNombramiento
}

func NuevoServicioDerivacionNombramiento(p ports.PreparadorDerivacionNombramiento,
	a ports.AutorizadorDerivacionNombramiento, t ports.TransaccionDerivacionNombramiento,
) (*ServicioDerivacionNombramiento, error) {
	if dependenciaNula(p) || dependenciaNula(a) || dependenciaNula(t) {
		return nil, ports.ErrDerivacionNombramientoNoDisponible
	}
	return &ServicioDerivacionNombramiento{p, a, t}, nil
}

func (s *ServicioDerivacionNombramiento) Derivar(ctx context.Context,
	e ports.EntradaDerivacionNombramiento,
) (ports.ResultadoDerivacionNombramiento, error) {
	vacio := ports.ResultadoDerivacionNombramiento{}
	if s == nil || ctx == nil || dependenciaNula(s.preparador) || dependenciaNula(s.autorizador) || dependenciaNula(s.transaccion) {
		return vacio, ports.ErrDerivacionNombramientoNoDisponible
	}
	if err := e.Validar(); err != nil {
		return vacio, err
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	sol, err := s.preparador.PrepararDerivacionNombramiento(ctx, e)
	if err != nil {
		return vacio, err
	}
	if _, err = sol.Canon(); err != nil || sol.Entrada() != e {
		return vacio, ports.ErrDerivacionNombramientoNoConfiable
	}
	if err = ctx.Err(); err != nil {
		return vacio, err
	}
	capacidad, err := s.autorizador.AutorizarDerivacionNombramiento(ctx, sol)
	if err != nil {
		return vacio, err
	}
	if err = ctx.Err(); err != nil {
		return vacio, err
	}
	resultado, err := s.transaccion.ConfirmarDerivacionNombramiento(ctx, sol, capacidad)
	if err != nil {
		return vacio, err
	}
	if resultado.ValidarPara(sol) != nil {
		return vacio, ports.ErrDerivacionNombramientoNoConfiable
	}
	return resultado, nil
}
