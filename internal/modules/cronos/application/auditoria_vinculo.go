package application

import (
	"context"
	"errors"
	"reflect"

	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

// RegistroDenegacionVinculo confirma en la auditoría común una denegación
// ocurrida después de resolver la identidad y antes de acceder al repositorio.
type RegistroDenegacionVinculo interface {
	RegistrarDenegacionVinculo(context.Context, vecdomain.ContextoActor) error
}

type claveRegistroDenegacionVinculo struct{}

// ConRegistroDenegacionVinculo incorpora la capacidad creada por la frontera
// confiable para esta petición. No transporta una decisión ni concede acceso.
func ConRegistroDenegacionVinculo(ctx context.Context, registro RegistroDenegacionVinculo) context.Context {
	if ctx == nil || registroDenegacionVinculoNulo(registro) {
		return ctx
	}
	return context.WithValue(ctx, claveRegistroDenegacionVinculo{}, registro)
}

func registrarDenegacionVinculo(ctx context.Context, actor vecdomain.ContextoActor) error {
	if ctx == nil {
		return ports.ErrDependenciaNoDisponible
	}
	registro, ok := ctx.Value(claveRegistroDenegacionVinculo{}).(RegistroDenegacionVinculo)
	if !ok || registroDenegacionVinculoNulo(registro) {
		return ports.ErrDependenciaNoDisponible
	}
	if err := registro.RegistrarDenegacionVinculo(ctx, actor); err != nil {
		return errors.Join(ports.ErrDependenciaNoDisponible, err)
	}
	return nil
}

func registroDenegacionVinculoNulo(registro RegistroDenegacionVinculo) bool {
	if registro == nil {
		return true
	}
	v := reflect.ValueOf(registro)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}
