package memory

import (
	"context"
	"sync"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// AlmacenContextoActor es un adaptador inmutable para pruebas y composiciones
// locales. Conserva copias privadas; el mutex protege tambien futuras lecturas
// concurrentes frente a cambios accidentales en la implementacion.
type AlmacenContextoActor struct {
	mu           sync.RWMutex
	instantaneas []domain.InstantaneaContextoActor
}

func NuevoAlmacenContextoActor(
	instantaneas ...domain.InstantaneaContextoActor,
) (*AlmacenContextoActor, error) {
	almacen := &AlmacenContextoActor{
		instantaneas: make([]domain.InstantaneaContextoActor, 0, len(instantaneas)),
	}
	for _, instantanea := range instantaneas {
		canonica, err := instantanea.ClonarCanonica()
		if err != nil {
			return nil, domain.ErrInstantaneaContextoActorInvalida
		}
		almacen.instantaneas = append(almacen.instantaneas, canonica)
	}
	return almacen, nil
}

func (a *AlmacenContextoActor) BuscarInstantaneasContextoActor(
	ctx context.Context,
	solicitud domain.SolicitudContextoActor,
) ([]domain.InstantaneaContextoActor, error) {
	if a == nil || ctx == nil || solicitud.Validar() != nil {
		return nil, ports.ErrFuenteContextoActorNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	a.mu.RLock()
	defer a.mu.RUnlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	resultado := make([]domain.InstantaneaContextoActor, 0)
	for _, instantanea := range a.instantaneas {
		if instantanea.CuentaRef != solicitud.Cuenta.CuentaRef ||
			instantanea.PerfilActivoRef != solicitud.PerfilActivoRef {
			continue
		}
		copia, err := instantanea.ClonarCanonica()
		if err != nil {
			return nil, ports.ErrFuenteContextoActorNoDisponible
		}
		resultado = append(resultado, copia)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return resultado, nil
}

var _ ports.FuenteContextoActor = (*AlmacenContextoActor)(nil)
