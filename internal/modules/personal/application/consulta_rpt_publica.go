package application

import (
	"context"
	"reflect"
	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
)

type ServicioConsultaRPTPublica struct{ consulta ports.ConsultaRPTPublica }

func NuevoServicioConsultaRPTPublica(consulta ports.ConsultaRPTPublica) (*ServicioConsultaRPTPublica, error) {
	if nulaRPTPublica(consulta) {
		return nil, domain.ErrConsultaRPTPublicaInvalida
	}
	return &ServicioConsultaRPTPublica{consulta: consulta}, nil
}
func (s *ServicioConsultaRPTPublica) Listar(ctx context.Context) (domain.CatalogoRPTPublica, error) {
	if ctx == nil || s == nil || nulaRPTPublica(s.consulta) {
		return domain.CatalogoRPTPublica{}, domain.ErrConsultaRPTPublicaInvalida
	}
	if err := ctx.Err(); err != nil {
		return domain.CatalogoRPTPublica{}, err
	}
	catalogo, err := s.consulta.ObtenerRPTPublica(ctx)
	if err != nil {
		return domain.CatalogoRPTPublica{}, err
	}
	if err := catalogo.Validar(); err != nil {
		return domain.CatalogoRPTPublica{}, domain.ErrRPTPublicaNoDisponible
	}
	return catalogo.Clonar(), nil
}
func nulaRPTPublica(v any) bool {
	if v == nil {
		return true
	}
	valor := reflect.ValueOf(v)
	return (valor.Kind() == reflect.Pointer || valor.Kind() == reflect.Interface || valor.Kind() == reflect.Func || valor.Kind() == reflect.Map || valor.Kind() == reflect.Slice) && valor.IsNil()
}
