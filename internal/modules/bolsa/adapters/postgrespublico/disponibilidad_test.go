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

func TestComprobarDisponibilidadSeguidoresNoEsperanNiDuplicanSonda(t *testing.T) {
	f := nuevaFuenteDisponibilidadPrueba()
	var llamadas atomic.Int32
	inicio := make(chan struct{})
	liberar := make(chan struct{})
	defer close(liberar)
	f.sondaDisponibilidadPrueba = func(context.Context) error {
		llamadas.Add(1)
		close(inicio)
		<-liberar
		return nil
	}
	liderTerminado := make(chan error, 1)
	go func() { liderTerminado <- f.ComprobarDisponibilidad(context.Background()) }()
	<-inicio

	var grupo sync.WaitGroup
	const seguidores = 500
	errores := make(chan error, seguidores)
	for range seguidores {
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			errores <- f.ComprobarDisponibilidad(context.Background())
		}()
	}
	seguidoresTerminados := make(chan struct{})
	go func() { grupo.Wait(); close(seguidoresTerminados) }()
	select {
	case <-seguidoresTerminados:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("los seguidores quedaron esperando a la sonda")
	}
	close(errores)
	for err := range errores {
		if err == nil {
			t.Fatal("seguidor acepto disponibilidad sin sonda terminada")
		}
	}
	if llamadas.Load() != 1 {
		t.Fatalf("sondas durante bloqueo=%d", llamadas.Load())
	}
	liberar <- struct{}{}
	if err := <-liderTerminado; err != nil {
		t.Fatalf("sonda lider: %v", err)
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
