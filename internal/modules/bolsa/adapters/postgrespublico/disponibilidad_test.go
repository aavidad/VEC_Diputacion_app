package postgrespublico

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func nuevaFuenteDisponibilidadPrueba() *Fuente {
	f := &Fuente{
		pool: &pgxpool.Pool{}, catalogoCategorias: "categorias-profesionales", versionCategorias: 1,
		huellaCategoriasGobernadaHex: strings.Repeat("a", 64), huellaProyeccionCategoriasHex: strings.Repeat("b", 64),
		manifiestoSHA256: strings.Repeat("c", 64),
	}
	f.cacheManifiesto.Store(&cacheManifiestoPublico{})
	return f
}

func TestComprobarDisponibilidadRequiereCache(t *testing.T) {
	f := nuevaFuenteDisponibilidadPrueba()
	f.cacheManifiesto.Store(nil)
	if err := f.ComprobarDisponibilidad(context.Background()); err == nil {
		t.Fatal("cache ausente aceptada")
	}
}

func TestComprobarDisponibilidadCoalesceYCacheaExito(t *testing.T) {
	f := nuevaFuenteDisponibilidadPrueba()
	var llamadas atomic.Int32
	inicio := make(chan struct{})
	liberar := make(chan struct{})
	f.sondaDisponibilidadPrueba = func(context.Context) error {
		llamadas.Add(1)
		close(inicio)
		<-liberar
		return nil
	}
	var grupo sync.WaitGroup
	for range 12 {
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			if err := f.ComprobarDisponibilidad(context.Background()); err != nil {
				t.Errorf("sonda: %v", err)
			}
		}()
	}
	<-inicio
	close(liberar)
	grupo.Wait()
	if llamadas.Load() != 1 {
		t.Fatalf("sondas=%d", llamadas.Load())
	}
	if err := f.ComprobarDisponibilidad(context.Background()); err != nil || llamadas.Load() != 1 {
		t.Fatalf("cache exito err=%v llamadas=%d", err, llamadas.Load())
	}
}

func TestComprobarDisponibilidadCacheaFalloBrevemente(t *testing.T) {
	f := nuevaFuenteDisponibilidadPrueba()
	var llamadas atomic.Int32
	f.sondaDisponibilidadPrueba = func(context.Context) error { llamadas.Add(1); return context.DeadlineExceeded }
	if f.ComprobarDisponibilidad(context.Background()) == nil || f.ComprobarDisponibilidad(context.Background()) == nil || llamadas.Load() != 1 {
		t.Fatalf("fallo no cacheado: llamadas=%d", llamadas.Load())
	}
	f.disponibilidadMu.Lock()
	f.disponibilidadHasta = time.Now().Add(-time.Nanosecond)
	f.disponibilidadMu.Unlock()
	_ = f.ComprobarDisponibilidad(context.Background())
	if llamadas.Load() != 2 {
		t.Fatalf("TTL de fallo no expiro: %d", llamadas.Load())
	}
}
