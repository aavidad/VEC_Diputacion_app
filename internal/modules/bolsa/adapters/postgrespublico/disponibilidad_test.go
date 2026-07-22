package postgrespublico

import (
	"context"
	"errors"
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
	f.integridadHasta = time.Now().Add(time.Minute)
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

func TestComprobarDisponibilidadPropagaCancelacionYLimpiaLider(t *testing.T) {
	f := nuevaFuenteDisponibilidadPrueba()
	iniciada := make(chan struct{})
	var llamadas atomic.Int32
	f.sondaDisponibilidadPrueba = func(ctx context.Context) error {
		llamadas.Add(1)
		close(iniciada)
		<-ctx.Done()
		return ctx.Err()
	}
	ctx, cancelar := context.WithCancel(context.Background())
	terminada := make(chan error, 1)
	go func() { terminada <- f.ComprobarDisponibilidad(ctx) }()
	<-iniciada
	cancelar()
	select {
	case err := <-terminada:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelacion no preservada: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("la sonda cancelada no termino")
	}
	f.disponibilidadMu.Lock()
	enCurso := f.disponibilidadEnCurso
	// Expira el backoff de cancelacion para acreditar que puede elegirse un
	// nuevo lider; mientras dura, las peticiones fallan sin churn.
	f.disponibilidadHasta = time.Now().Add(-time.Nanosecond)
	f.disponibilidadMu.Unlock()
	if enCurso {
		t.Fatal("lider retenido tras cancelar la peticion")
	}
	f.sondaDisponibilidadPrueba = func(context.Context) error { llamadas.Add(1); return nil }
	if err := f.ComprobarDisponibilidad(context.Background()); err != nil || llamadas.Load() != 2 {
		t.Fatalf("la fuente no se recupero: err=%v llamadas=%d", err, llamadas.Load())
	}
}

func TestSondaDisponibilidadDerivaTimeoutMaximoDelContexto(t *testing.T) {
	f := nuevaFuenteDisponibilidadPrueba()
	f.sondaDisponibilidadPrueba = func(ctx context.Context) error {
		limite, existe := ctx.Deadline()
		if !existe || time.Until(limite) <= 0 || time.Until(limite) > duracionSondaDisponibilidadPublica+100*time.Millisecond {
			t.Fatalf("deadline de sonda = %v", limite)
		}
		return nil
	}
	if err := f.ComprobarDisponibilidad(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestDisponibilidadExigeIntegridadReciente(t *testing.T) {
	f := nuevaFuenteDisponibilidadPrueba()
	var llamadas atomic.Int32
	f.sondaDisponibilidadPrueba = func(context.Context) error { llamadas.Add(1); return nil }
	f.integridadHasta = time.Now().Add(-time.Nanosecond)
	if err := f.ComprobarDisponibilidad(context.Background()); err == nil || llamadas.Load() != 0 {
		t.Fatalf("integridad caducada aceptada: err=%v llamadas=%d", err, llamadas.Load())
	}
}

func TestSondaLigeraNoPublicaVerdeSiIntegridadCambiaEnVuelo(t *testing.T) {
	f := nuevaFuenteDisponibilidadPrueba()
	iniciada := make(chan struct{})
	continuar := make(chan struct{})
	f.sondaDisponibilidadPrueba = func(context.Context) error {
		close(iniciada)
		<-continuar
		return nil
	}
	terminada := make(chan error, 1)
	go func() { terminada <- f.ComprobarDisponibilidad(context.Background()) }()
	<-iniciada
	f.actualizarIntegridad(ErrDatosPostgreSQLPublicosNoConfiables)
	close(continuar)
	if err := <-terminada; err == nil {
		t.Fatal("la sonda ligera sobrescribio una deriva integral concurrente")
	}
}

func TestObservabilidadSoloEmiteTransicionesSaneadas(t *testing.T) {
	f := nuevaFuenteDisponibilidadPrueba()
	var estados []bool
	f.observadorDisponibilidad = func(disponible bool) { estados = append(estados, disponible) }
	f.sondaDisponibilidadPrueba = func(context.Context) error { return nil }
	if err := f.ComprobarDisponibilidad(context.Background()); err != nil {
		t.Fatal(err)
	}
	// Un hit de cache no vuelve a observar el mismo estado.
	_ = f.ComprobarDisponibilidad(context.Background())
	f.disponibilidadMu.Lock()
	f.disponibilidadHasta = time.Now().Add(-time.Nanosecond)
	f.disponibilidadMu.Unlock()
	f.sondaDisponibilidadPrueba = func(context.Context) error { return errors.New("dsn=secreto payload=prohibido") }
	if err := f.ComprobarDisponibilidad(context.Background()); err == nil {
		t.Fatal("fallo de sonda aceptado")
	}
	if len(estados) != 2 || !estados[0] || estados[1] {
		t.Fatalf("transiciones observadas = %v", estados)
	}
}

func TestLimitesIntegridadReadinessSonAcotados(t *testing.T) {
	if maximoFilasManifiestoArranque != 12_000 || maximoBytesManifiestoArranque != 256<<20 ||
		duracionIntegridadDisponibilidad > 30*time.Second || periodoIntegridadDisponibilidad < 5*time.Minute ||
		vigenciaIntegridadDisponibilidad < periodoIntegridadDisponibilidad+periodoIntegridadDisponibilidad/10+
			duracionIntegridadDisponibilidad {
		t.Fatalf("limites inesperados: filas=%d bytes=%d timeout=%s periodo=%s",
			maximoFilasManifiestoArranque, maximoBytesManifiestoArranque,
			duracionIntegridadDisponibilidad, periodoIntegridadDisponibilidad)
	}
}

func TestJitterYBackoffIntegridadPermanecenAcotados(t *testing.T) {
	instante := time.Unix(1_750_000_000, 123_456_789)
	for _, base := range []time.Duration{time.Minute, 2 * time.Minute, 4 * time.Minute, 8 * time.Minute, 15 * time.Minute} {
		obtenida := duracionIntegridadConJitter(base, instante)
		minimo := base - base/10
		maximo := base + base/10
		if maximo > reintentoIntegridadMaximo {
			maximo = reintentoIntegridadMaximo
		}
		if obtenida < minimo || obtenida > maximo {
			t.Fatalf("jitter(%s) = %s; rango [%s,%s]", base, obtenida, minimo, maximo)
		}
	}
}

func TestValidacionIntegridadRecuperaPanicoComoFallo(t *testing.T) {
	f := nuevaFuenteDisponibilidadPrueba()
	f.sondaIntegridadPrueba = func(context.Context) error { panic("payload secreto") }
	if err := f.ejecutarIntegridadProtegida(context.Background()); !errors.Is(err, ErrPostgreSQLPublicoNoDisponible) {
		t.Fatalf("panico no convertido en fallo saneado: %v", err)
	}
}

func TestCerrarDetieneWorkerIntegridadSinFuga(t *testing.T) {
	f := &Fuente{}
	f.iniciarVigilanciaIntegridad()
	f.integridadMu.RLock()
	terminada := f.integridadTerminada
	f.integridadMu.RUnlock()
	f.Cerrar()
	select {
	case <-terminada:
	default:
		t.Fatal("Cerrar retorno con la worker de integridad activa")
	}
	// El cierre es idempotente y no intenta cerrar de nuevo el canal.
	f.Cerrar()
}
