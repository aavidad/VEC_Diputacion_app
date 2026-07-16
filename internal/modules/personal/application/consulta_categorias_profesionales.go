package application

import (
	"context"
	"reflect"
	"time"

	"vec-diputacion-granada/internal/modules/personal/ports"
)

// RelojCategoriasProfesionales impide que un consumidor elija el instante de
// vigencia del catalogo.
type RelojCategoriasProfesionales interface {
	Ahora() time.Time
}

type RelojSistemaCategoriasProfesionales struct{}

func (RelojSistemaCategoriasProfesionales) Ahora() time.Time {
	return time.Now().UTC().Truncate(time.Microsecond)
}

type ServicioConsultaCategoriasProfesionales struct {
	consulta ports.ConsultaCategoriasProfesionales
	reloj    RelojCategoriasProfesionales
}

func NuevoServicioConsultaCategoriasProfesionales(
	consulta ports.ConsultaCategoriasProfesionales,
	reloj RelojCategoriasProfesionales,
) (*ServicioConsultaCategoriasProfesionales, error) {
	if dependenciaCategoriasProfesionalesNula(consulta) || dependenciaCategoriasProfesionalesNula(reloj) {
		return nil, ports.ErrConsultaCategoriasProfesionalesInvalida
	}
	return &ServicioConsultaCategoriasProfesionales{consulta: consulta, reloj: reloj}, nil
}

func (s *ServicioConsultaCategoriasProfesionales) ListarVigentes(
	ctx context.Context,
) (ports.CatalogoCategoriasProfesionalesConsultable, error) {
	if ctx == nil || s == nil || dependenciaCategoriasProfesionalesNula(s.consulta) ||
		dependenciaCategoriasProfesionalesNula(s.reloj) {
		return ports.CatalogoCategoriasProfesionalesConsultable{}, ports.ErrConsultaCategoriasProfesionalesInvalida
	}
	if err := ctx.Err(); err != nil {
		return ports.CatalogoCategoriasProfesionalesConsultable{}, err
	}
	instante := s.reloj.Ahora()
	if instante.IsZero() {
		return ports.CatalogoCategoriasProfesionalesConsultable{}, ports.ErrConsultaCategoriasProfesionalesInvalida
	}
	instante = instante.UTC().Truncate(time.Microsecond)
	catalogo, err := s.consulta.ObtenerVigentes(ctx, instante)
	if err != nil {
		return ports.CatalogoCategoriasProfesionalesConsultable{}, err
	}
	if err := ctx.Err(); err != nil {
		return ports.CatalogoCategoriasProfesionalesConsultable{}, err
	}
	if err := catalogo.Validar(); err != nil {
		return ports.CatalogoCategoriasProfesionalesConsultable{}, ports.ErrCatalogoCategoriasProfesionalesNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return ports.CatalogoCategoriasProfesionalesConsultable{}, err
	}
	return catalogo.Clonar(), nil
}

func dependenciaCategoriasProfesionalesNula(dependencia any) bool {
	if dependencia == nil {
		return true
	}
	valor := reflect.ValueOf(dependencia)
	switch valor.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return valor.IsNil()
	default:
		return false
	}
}
