package application

import (
	"context"
	"errors"
	"reflect"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// ServicioContextoActor resuelve la cuenta tecnica autenticada a una unica
// persona canonica para el perfil solicitado expresamente. No autentica, no
// autoriza y no infiere perfiles; produce el contexto cerrado que consumiran
// despues los casos de uso y el PDP.
type ServicioContextoActor struct {
	fuente ports.FuenteContextoActor
	reloj  ports.Reloj
}

func NuevoServicioContextoActor(
	fuente ports.FuenteContextoActor,
	reloj ports.Reloj,
) (*ServicioContextoActor, error) {
	if dependenciaContextoActorNula(fuente) || dependenciaContextoActorNula(reloj) {
		return nil, domain.ErrContextoActorInvalido
	}
	return &ServicioContextoActor{fuente: fuente, reloj: reloj}, nil
}

func (s *ServicioContextoActor) Resolver(
	ctx context.Context,
	solicitud domain.SolicitudContextoActor,
) (domain.ContextoActor, error) {
	if ctx == nil {
		return domain.ContextoActor{}, errorResolucionContextoActor(domain.ErrSolicitudContextoActorInvalida)
	}
	if err := ctx.Err(); err != nil {
		return domain.ContextoActor{}, errorResolucionContextoActor(err)
	}
	if s == nil || dependenciaContextoActorNula(s.fuente) || dependenciaContextoActorNula(s.reloj) {
		return domain.ContextoActor{}, errorResolucionContextoActor(ports.ErrFuenteContextoActorNoDisponible)
	}
	if err := solicitud.Validar(); err != nil {
		return domain.ContextoActor{}, errorResolucionContextoActor(err)
	}

	instante := s.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if instante.IsZero() {
		return domain.ContextoActor{}, errorResolucionContextoActor(domain.ErrContextoActorInvalido)
	}
	instantaneas, err := s.fuente.BuscarInstantaneasContextoActor(ctx, solicitud)
	if err != nil {
		if contextoErr := ctx.Err(); contextoErr != nil {
			return domain.ContextoActor{}, errorResolucionContextoActor(contextoErr)
		}
		return domain.ContextoActor{}, errorResolucionContextoActor(ports.ErrFuenteContextoActorNoDisponible)
	}
	if err := ctx.Err(); err != nil {
		return domain.ContextoActor{}, errorResolucionContextoActor(err)
	}
	// Cero y mas de una coincidencia reciben exactamente el mismo resultado. No
	// se revela existencia ni se elige la primera aunque apunte a la misma persona.
	if len(instantaneas) != 1 {
		return domain.ContextoActor{}, errorResolucionContextoActor(nil)
	}
	instantanea := instantaneas[0]
	if instantanea.Validar() != nil ||
		instantanea.CuentaRef != solicitud.Cuenta.CuentaRef ||
		instantanea.PerfilActivoRef != solicitud.PerfilActivoRef ||
		!instantanea.VigenteEn(instante) {
		return domain.ContextoActor{}, errorResolucionContextoActor(nil)
	}

	resultado, err := domain.NuevoContextoActor(solicitud.Cuenta, instantanea, instante)
	if err != nil || resultado.PerfilActivoRef != solicitud.PerfilActivoRef {
		return domain.ContextoActor{}, errorResolucionContextoActor(nil)
	}
	copia, err := resultado.Clonar()
	if err != nil {
		return domain.ContextoActor{}, errorResolucionContextoActor(nil)
	}
	return copia, nil
}

func errorResolucionContextoActor(causa error) error {
	if causa == nil {
		return domain.ErrContextoActorNoResuelto
	}
	return errors.Join(domain.ErrContextoActorNoResuelto, causa)
}

func dependenciaContextoActorNula(valor any) bool {
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
