package application

import (
	"context"
	"sync"
	"testing"
	"time"
)

// contextoPlazoCoberturaPrueba permite activar de forma determinista las dos
// causas públicas de cancelación sin depender del planificador ni del reloj de
// pared del equipo de pruebas.
type contextoPlazoCoberturaPrueba struct {
	padre context.Context
	done  chan struct{}
	mu    sync.RWMutex
	err   error
	una   sync.Once
}

func nuevoContextoPlazoCoberturaPrueba(
	padre context.Context,
) *contextoPlazoCoberturaPrueba {
	return &contextoPlazoCoberturaPrueba{
		padre: padre,
		done:  make(chan struct{}),
	}
}

func (c *contextoPlazoCoberturaPrueba) Deadline() (time.Time, bool) {
	return time.Time{}, false
}

func (c *contextoPlazoCoberturaPrueba) Done() <-chan struct{} {
	return c.done
}

func (c *contextoPlazoCoberturaPrueba) Err() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.err
}

func (c *contextoPlazoCoberturaPrueba) Value(clave any) any {
	return c.padre.Value(clave)
}

func (c *contextoPlazoCoberturaPrueba) finalizar(err error) {
	c.una.Do(func() {
		c.mu.Lock()
		c.err = err
		c.mu.Unlock()
		close(c.done)
	})
}

func (e *entornoCoberturaAplicacionPrueba) usarPlazoControlado(
	t *testing.T,
) *contextoPlazoCoberturaPrueba {
	t.Helper()
	controlado := nuevoContextoPlazoCoberturaPrueba(context.Background())
	e.servicio.crearPlazo = func(
		padre context.Context,
		_ time.Duration,
	) (context.Context, context.CancelFunc) {
		controlado.padre = padre
		return controlado, func() {}
	}
	return controlado
}
