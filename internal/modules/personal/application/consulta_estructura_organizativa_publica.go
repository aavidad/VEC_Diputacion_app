package application

import (
	"context"
	"reflect"
	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
)

type ServicioConsultaEstructuraOrganizativaPublica struct {
	consulta ports.ConsultaEstructuraOrganizativaPublica
}

func NuevoServicioConsultaEstructuraOrganizativaPublica(consulta ports.ConsultaEstructuraOrganizativaPublica) (*ServicioConsultaEstructuraOrganizativaPublica, error) {
	if consultaEstructuraPublicaNula(consulta) {
		return nil, domain.ErrConsultaEstructuraPublicaInvalida
	}
	return &ServicioConsultaEstructuraOrganizativaPublica{consulta: consulta}, nil
}
func (s *ServicioConsultaEstructuraOrganizativaPublica) Obtener(ctx context.Context) (domain.EstructuraOrganizativaPublica, error) {
	if ctx == nil || s == nil || consultaEstructuraPublicaNula(s.consulta) {
		return domain.EstructuraOrganizativaPublica{}, domain.ErrConsultaEstructuraPublicaInvalida
	}
	if err := ctx.Err(); err != nil {
		return domain.EstructuraOrganizativaPublica{}, err
	}
	resultado, err := s.consulta.ObtenerEstructuraOrganizativaPublica(ctx)
	if err != nil {
		return domain.EstructuraOrganizativaPublica{}, err
	}
	if err := resultado.Validar(); err != nil {
		return domain.EstructuraOrganizativaPublica{}, domain.ErrEstructuraPublicaNoDisponible
	}
	return resultado.Clonar(), nil
}
func consultaEstructuraPublicaNula(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	return (r.Kind() == reflect.Pointer || r.Kind() == reflect.Interface || r.Kind() == reflect.Func || r.Kind() == reflect.Map || r.Kind() == reflect.Slice) && r.IsNil()
}
