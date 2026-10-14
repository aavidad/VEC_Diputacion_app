package bootstrap

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
)

type filaPreflightC9 struct {
	disponible bool
	err        error
}

func (f filaPreflightC9) Scan(destinos ...any) error {
	if f.err != nil {
		return f.err
	}
	*destinos[0].(*bool) = f.disponible
	return nil
}

type consultorPreflightC9 struct {
	fila     filaPreflightC9
	llamadas int
}

func (c *consultorPreflightC9) QueryRow(_ context.Context, sql string, _ ...any) pgx.Row {
	c.llamadas++
	if sql != preflightNotificacionesCronosC9 {
		return filaPreflightC9{err: errors.New("consulta distinta")}
	}
	return c.fila
}

func TestPreflightNotificacionesCronosC9FallaCerrado(t *testing.T) {
	if !errors.Is(comprobarNotificacionesCronosC9(context.Background(), nil), ErrCronosFaltaC9) {
		t.Fatal("acepta dependencia ausente")
	}
	for _, fila := range []filaPreflightC9{{disponible: false}, {err: errors.New("sin catálogo")}} {
		c := &consultorPreflightC9{fila: fila}
		if !errors.Is(comprobarNotificacionesCronosC9(context.Background(), c), ErrCronosFaltaC9) || c.llamadas != 1 {
			t.Fatal("acepta preflight ausente o fallido")
		}
	}
	c := &consultorPreflightC9{fila: filaPreflightC9{disponible: true}}
	if err := comprobarNotificacionesCronosC9(context.Background(), c); err != nil || c.llamadas != 1 {
		t.Fatalf("preflight completo: %v", err)
	}
}
