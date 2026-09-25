package bootstrap

import (
	"context"
	"errors"
	"reflect"
	"time"

	dietascomp "vec-diputacion-granada/internal/modules/dietas/adapters/composicion"
	personalcomp "vec-diputacion-granada/internal/modules/personal/adapters/composicion"
	personalhttp "vec-diputacion-granada/internal/modules/personal/adapters/httpinterno"
	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var ErrIdentidadPersonalDietasNoDisponible = errors.New("bootstrap: identidad Personal-Dietas no disponible")

type resolutorContextoPersonalDietas interface {
	ResolverContexto(context.Context) (contextoSeguridadComunDesarrollo, error)
}

// identidadPersonalDietas usa una sola sesión de canal para la consulta de
// Personal y la actuación de Dietas. La selección de relación la hace Personal.
type identidadPersonalDietas struct {
	seguridad resolutorContextoPersonalDietas
	reloj     vecports.Reloj
	zona      *time.Location
}

func nuevaIdentidadPersonalDietas(seguridad resolutorContextoPersonalDietas, reloj vecports.Reloj) (*identidadPersonalDietas, error) {
	if dependenciaDietasNula(seguridad) || dependenciaDietasNula(reloj) {
		return nil, ErrIdentidadPersonalDietasNoDisponible
	}
	zona, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		return nil, ErrIdentidadPersonalDietasNoDisponible
	}
	return &identidadPersonalDietas{seguridad: seguridad, reloj: reloj, zona: zona}, nil
}

func (i *identidadPersonalDietas) ResolverIdentidadRelacionDietas(ctx context.Context) (personalcomp.IdentidadRegistradaRelacionDietas, error) {
	vacia := personalcomp.IdentidadRegistradaRelacionDietas{}
	if i == nil || dependenciaDietasNula(i.seguridad) || dependenciaDietasNula(i.reloj) || ctx == nil {
		return vacia, ErrIdentidadPersonalDietasNoDisponible
	}
	resultado, err := i.seguridad.ResolverContexto(ctx)
	if err != nil || resultado.Resultado.Validar() != nil || resultado.Vinculo.ValidarPara(resultado.Resultado) != nil {
		return vacia, ErrIdentidadPersonalDietasNoDisponible
	}
	return personalcomp.IdentidadRegistradaRelacionDietas{Vinculo: resultado.Vinculo, Resultado: resultado.Resultado}, nil
}

func (i *identidadPersonalDietas) ResolverIdentidadRegistradaBorrador(ctx context.Context) (dietascomp.IdentidadRegistradaBorrador, error) {
	vacia := dietascomp.IdentidadRegistradaBorrador{}
	base, err := i.ResolverIdentidadRelacionDietas(ctx)
	if err != nil || i.zona == nil {
		return vacia, ErrIdentidadPersonalDietasNoDisponible
	}
	fecha, err := personaldomain.NuevaFechaCivil(i.reloj.Ahora().In(i.zona).Format("2006-01-02"))
	if err != nil {
		return vacia, ErrIdentidadPersonalDietasNoDisponible
	}
	return dietascomp.IdentidadRegistradaBorrador{Contexto: base.Resultado, Vinculo: base.Vinculo, FechaReferencia: fecha}, nil
}

func (i *identidadPersonalDietas) ResolverIdentidadRelacionesDietas(ctx context.Context) (vecdomain.ContextoActor, personaldomain.FechaCivil, error) {
	base, err := i.ResolverIdentidadRegistradaBorrador(ctx)
	if err != nil {
		return vecdomain.ContextoActor{}, "", ErrIdentidadPersonalDietasNoDisponible
	}
	actor, err := base.Contexto.Contexto.Clonar()
	if err != nil {
		return vecdomain.ContextoActor{}, "", ErrIdentidadPersonalDietasNoDisponible
	}
	return actor, base.FechaReferencia, nil
}

func (i *identidadPersonalDietas) ResolverIdentidadAsignacionDietas(ctx context.Context) (vecdomain.ContextoActor, error) {
	base, err := i.ResolverIdentidadRelacionDietas(ctx)
	if err != nil {
		return vecdomain.ContextoActor{}, ErrIdentidadPersonalDietasNoDisponible
	}
	actor, err := base.Resultado.Contexto.Clonar()
	if err != nil {
		return vecdomain.ContextoActor{}, ErrIdentidadPersonalDietasNoDisponible
	}
	return actor, nil
}

var _ personalcomp.ResolutorIdentidadRelacionDietas = (*identidadPersonalDietas)(nil)
var _ dietascomp.ResolutorIdentidadRegistradaBorrador = (*identidadPersonalDietas)(nil)
var _ personalhttp.ResolutorIdentidadRelacionesDietas = (*identidadPersonalDietas)(nil)
var _ personalhttp.ResolutorIdentidadAsignacionDietas = (*identidadPersonalDietas)(nil)

func dependenciaDietasNula(valor any) bool {
	if valor == nil {
		return true
	}
	v := reflect.ValueOf(valor)
	return (v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface || v.Kind() == reflect.Func || v.Kind() == reflect.Map || v.Kind() == reflect.Slice) && v.IsNil()
}
