package postgresql

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

type marcaPrueba struct{}

func (marcaPrueba) Error() string             { return "marca" }
func (marcaPrueba) CarreraSerializable() bool { return true }

func TestEsCarreraSerializableSoloCodigosDeCarrera(t *testing.T) {
	for _, err := range []error{&pgconn.PgError{Code: "40001"}, fmt.Errorf("envuelto: %w", &pgconn.PgError{Code: "40P01"}), marcaPrueba{}} {
		if !EsCarreraSerializable(err) {
			t.Fatalf("%v debería repetirse", err)
		}
	}
	for _, err := range []error{nil, errors.New("otro"), &pgconn.PgError{Code: "55P03"}, &pgconn.PgError{Code: "23505"}, context.Canceled} {
		if EsCarreraSerializable(err) {
			t.Fatalf("%v no debería repetirse", err)
		}
	}
}

func TestRepetirTrasCarreraSerializablePolitica(t *testing.T) {
	carrera := &pgconn.PgError{Code: "40001"}
	n := 0
	if err := RepetirTrasCarreraSerializable(context.Background(), func() error {
		n++
		if n < 4 {
			return carrera
		}
		return nil
	}); err != nil || n != 4 {
		t.Fatalf("recuperación: err=%v intentos=%d", err, n)
	}
	n = 0
	otro := errors.New("guarda rechazada")
	if err := RepetirTrasCarreraSerializable(context.Background(), func() error { n++; return otro }); err != otro || n != 1 {
		t.Fatalf("otro error: err=%v intentos=%d", err, n)
	}
	n = 0
	if err := RepetirTrasCarreraSerializable(context.Background(), func() error { n++; return carrera }); err != carrera || n != IntentosMaximosCarreraSerializable {
		t.Fatalf("agotamiento: err=%v intentos=%d", err, n)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	n = 0
	if err := RepetirTrasCarreraSerializable(ctx, func() error { n++; cancelar(); return carrera }); err != carrera || n != 1 {
		t.Fatalf("contexto: err=%v intentos=%d", err, n)
	}
}

type contextoCancelarConTemporizadorVencido struct {
	context.Context
	cancelar context.CancelFunc
}

func (c contextoCancelarConTemporizadorVencido) Done() <-chan struct{} {
	// En el reloj virtual, vence cualquier espera posible antes del select.
	time.Sleep(esperaMaximaCarreraSerializable + 2*time.Millisecond)
	c.cancelar()
	return c.Context.Done()
}

func TestRepetirTrasCarreraSerializableNoReintentaConTemporizadorYCancelacionListos(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		for range 20 {
			ctx, cancelar := context.WithCancel(context.Background())
			carrera := &pgconn.PgError{Code: "40001"}
			intentos := 0
			err := RepetirTrasCarreraSerializable(contextoCancelarConTemporizadorVencido{ctx, cancelar}, func() error {
				intentos++
				if intentos > 1 {
					return nil
				}
				return carrera
			})
			if err != carrera || intentos != 1 || ctx.Err() != context.Canceled {
				t.Fatalf("cancelación con temporizador vencido: err=%v intentos=%d contexto=%v", err, intentos, ctx.Err())
			}
		}
	})
}

// Muchas peticiones simultáneas sobre la misma «fila»: todas toman su
// instantánea a la vez y solo la primera en confirmar gana cada ronda, como
// con SELECT ... FOR UPDATE en SERIALIZABLE. Con la política común terminan
// todas.
func TestRepetirTrasCarreraSerializableAbsorbeDecenasDePeticiones(t *testing.T) {
	const peticiones = 60
	var mu sync.Mutex
	version := 0
	var salida, grupo sync.WaitGroup
	salida.Add(peticiones)
	errores := make(chan error, peticiones)
	for range peticiones {
		grupo.Go(func() {
			primero := true
			errores <- RepetirTrasCarreraSerializable(context.Background(), func() error {
				mu.Lock()
				instantanea := version
				mu.Unlock()
				if primero {
					primero = false
					salida.Done()
					salida.Wait()
				}
				mu.Lock()
				defer mu.Unlock()
				if version != instantanea {
					return &pgconn.PgError{Code: "40001"}
				}
				version++
				return nil
			})
		})
	}
	grupo.Wait()
	close(errores)
	for err := range errores {
		if err != nil {
			t.Fatalf("petición fallida: %v", err)
		}
	}
	if version != peticiones {
		t.Fatalf("confirmadas %d de %d", version, peticiones)
	}
}
