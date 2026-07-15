package application

import (
	"context"
	"errors"
	"reflect"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// ServicioVinculoAutenticacionActorV1 es la unica ruta de aplicacion prevista
// para construir el bloque que consumira el PDP. Exige un resultado obtenido
// de la autoridad de sesion y un ContextoActor resuelto por su propio servicio.
type ServicioVinculoAutenticacionActorV1 struct {
	revalidador ports.RevalidadorAutenticacionActorV1
	reloj       ports.Reloj
}

func NuevoServicioVinculoAutenticacionActorV1(
	revalidador ports.RevalidadorAutenticacionActorV1,
	reloj ports.Reloj,
) (*ServicioVinculoAutenticacionActorV1, error) {
	if dependenciaVinculoAutenticacionActorNula(revalidador) ||
		dependenciaVinculoAutenticacionActorNula(reloj) {
		return nil, domain.ErrVinculoAutenticacionActorInvalido
	}
	return &ServicioVinculoAutenticacionActorV1{revalidador: revalidador, reloj: reloj}, nil
}

func (s *ServicioVinculoAutenticacionActorV1) Crear(
	ctx context.Context,
	solicitud domain.SolicitudRevalidacionAutenticacionActorV1,
	actor domain.ContextoActor,
) (domain.VinculoAutenticacionActorV1, error) {
	if ctx == nil || s == nil || dependenciaVinculoAutenticacionActorNula(s.revalidador) ||
		dependenciaVinculoAutenticacionActorNula(s.reloj) {
		return domain.VinculoAutenticacionActorV1{}, errorVinculoAutenticacionActor(nil)
	}
	if err := ctx.Err(); err != nil {
		return domain.VinculoAutenticacionActorV1{}, errorVinculoAutenticacionActor(err)
	}
	if solicitud.Validar() != nil || actor.Validar() != nil {
		return domain.VinculoAutenticacionActorV1{}, errorVinculoAutenticacionActor(nil)
	}
	ahora := s.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if ahora.IsZero() {
		return domain.VinculoAutenticacionActorV1{}, errorVinculoAutenticacionActor(nil)
	}
	vinculo, err := domain.CrearVinculoAutenticacionActorV1(ctx, s.revalidador, solicitud, actor, ahora)
	if err != nil {
		if contextoErr := ctx.Err(); contextoErr != nil {
			return domain.VinculoAutenticacionActorV1{}, errorVinculoAutenticacionActor(contextoErr)
		}
		return domain.VinculoAutenticacionActorV1{}, errors.Join(
			domain.ErrAutorizacionDenegada,
			domain.ErrVinculoAutenticacionActorInvalido,
			ports.ErrRevalidacionAutenticacionActorNoDisponible,
			err,
		)
	}
	if err := ctx.Err(); err != nil {
		return domain.VinculoAutenticacionActorV1{}, errorVinculoAutenticacionActor(err)
	}
	return vinculo, nil
}

func errorVinculoAutenticacionActor(causa error) error {
	if causa == nil {
		return errors.Join(domain.ErrAutorizacionDenegada, domain.ErrVinculoAutenticacionActorInvalido)
	}
	return errors.Join(domain.ErrAutorizacionDenegada, domain.ErrVinculoAutenticacionActorInvalido, causa)
}

func dependenciaVinculoAutenticacionActorNula(valor any) bool {
	if valor == nil {
		return true
	}
	reflejo := reflect.ValueOf(valor)
	switch reflejo.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflejo.IsNil()
	default:
		return false
	}
}
