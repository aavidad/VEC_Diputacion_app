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

func TestComprobarDisponibilidadCoalesceSeguidoresEnUnaSondaAcotada(t *testing.T) {
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
	// Ningun seguidor puede crear otra sonda mientras la compartida esta
	// instalada; al liberarla todos observan el mismo resultado.
	if llamadas.Load() != 1 {
		t.Fatalf("sondas durante bloqueo=%d", llamadas.Load())
	}
	close(liberar)
	if err := <-liderTerminado; err != nil {
		t.Fatalf("sonda lider: %v", err)
	}
	seguidoresTerminados := make(chan struct{})
	go func() {
		grupo.Wait()
		close(seguidoresTerminados)
	}()
	select {
	case <-seguidoresTerminados:
	case <-time.After(time.Second):
		t.Fatal("los seguidores no recibieron el resultado compartido")
	}
	close(errores)
	for err := range errores {
		if err != nil {
			t.Fatalf("seguidor no recibio verde compartido: %v", err)
		}
	}
	if llamadas.Load() != 1 {
		t.Fatalf("sondas compartidas=%d", llamadas.Load())
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

func TestCancelacionSolicitanteNoCancelaNiEnvenenaSondaCompartida(t *testing.T) {
	f := nuevaFuenteDisponibilidadPrueba()
	iniciada := make(chan struct{})
	liberar := make(chan struct{})
	var llamadas atomic.Int32
	var estadosMu sync.Mutex
	var estados []bool
	observada := make(chan bool, 1)
	f.observadorDisponibilidad = func(disponible bool) {
		estadosMu.Lock()
		estados = append(estados, disponible)
		estadosMu.Unlock()
		observada <- disponible
	}
	f.sondaDisponibilidadPrueba = func(ctx context.Context) error {
		llamadas.Add(1)
		close(iniciada)
		select {
		case <-liberar:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelada := make(chan error, 1)
	go func() { cancelada <- f.ComprobarDisponibilidad(ctx) }()
	<-iniciada
	cancelar()
	select {
	case err := <-cancelada:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelacion no preservada: %v", err)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("el solicitante cancelado quedo esperando a la sonda")
	}
	f.disponibilidadMu.Lock()
	enCurso := f.disponibilidadSondaTerminada != nil
	hasta := f.disponibilidadHasta
	f.disponibilidadMu.Unlock()
	estadosMu.Lock()
	estadosAntes := append([]bool(nil), estados...)
	estadosMu.Unlock()
	if !enCurso || !hasta.IsZero() || len(estadosAntes) != 0 {
		t.Fatalf("cancelacion contamino estado global: en_curso=%t hasta=%s estados=%v", enCurso, hasta, estadosAntes)
	}
	seguidora := make(chan error, 1)
	go func() { seguidora <- f.ComprobarDisponibilidad(context.Background()) }()
	close(liberar)
	if err := <-seguidora; err != nil || llamadas.Load() != 1 {
		t.Fatalf("resultado compartido tras cancelar: err=%v llamadas=%d", err, llamadas.Load())
	}
	select {
	case disponible := <-observada:
		if !disponible {
			t.Fatal("la sonda compartida publicó indisponibilidad")
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("la sonda compartida no publicó su transición")
	}
	estadosMu.Lock()
	defer estadosMu.Unlock()
	if len(estados) != 1 || !estados[0] {
		t.Fatalf("observabilidad contaminada por cancelacion: %v", estados)
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
	var estadosMu sync.Mutex
	var estados []bool
	f.observadorDisponibilidad = func(disponible bool) {
		estadosMu.Lock()
		estados = append(estados, disponible)
		estadosMu.Unlock()
	}
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
	estadosMu.Lock()
	defer estadosMu.Unlock()
	if len(estados) != 1 || estados[0] {
		t.Fatalf("la invalidacion fue sobrescrita en observabilidad: %v", estados)
	}
	f.disponibilidadMu.Lock()
	defer f.disponibilidadMu.Unlock()
	if f.disponibilidadErr == nil {
		t.Fatal("la invalidacion fue sobrescrita en cache")
	}
}

func TestObservabilidadSoloEmiteTransicionesSaneadas(t *testing.T) {
	f := nuevaFuenteDisponibilidadPrueba()
	estados := make(chan bool, 2)
	f.observadorDisponibilidad = func(disponible bool) { estados <- disponible }
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
	for _, esperada := range []bool{true, false} {
		select {
		case observada := <-estados:
			if observada != esperada {
				t.Fatalf("transicion observada = %t; esperada = %t", observada, esperada)
			}
		case <-time.After(time.Second):
			t.Fatalf("no se observo la transicion esperada = %t", esperada)
		}
	}
	select {
	case observada := <-estados:
		t.Fatalf("transicion adicional observada = %t", observada)
	default:
	}
}

func TestObservabilidadOrdenaCallbacksReentrantesSinDeadlock(t *testing.T) {
	f := nuevaFuenteDisponibilidadPrueba()
	var estados []bool
	f.observadorDisponibilidad = func(disponible bool) {
		estados = append(estados, disponible)
		if disponible {
			f.registrarEstadoDisponibilidad(false)
		}
	}
	terminada := make(chan struct{})
	go func() {
		f.registrarEstadoDisponibilidad(true)
		close(terminada)
	}()
	select {
	case <-terminada:
	case <-time.After(time.Second):
		t.Fatal("callback reentrante produjo deadlock")
	}
	if len(estados) != 2 || !estados[0] || estados[1] {
		t.Fatalf("callbacks fuera de orden: %v", estados)
	}
}

func TestLimitesIntegridadReadinessSonAcotados(t *testing.T) {
	if maximoFilasManifiestoArranque != 12_000 || maximoBytesManifiestoArranque != 256<<20 ||
		duracionIntegridadDisponibilidad != 30*time.Second || periodoIntegridadDisponibilidad < 5*time.Minute ||
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

func TestCerrarCancelaYEsperaSondaDisponibilidadActiva(t *testing.T) {
	f := nuevaFuenteDisponibilidadPrueba()
	iniciada := make(chan struct{})
	detenida := make(chan struct{})
	f.sondaDisponibilidadPrueba = func(ctx context.Context) error {
		close(iniciada)
		<-ctx.Done()
		close(detenida)
		return ctx.Err()
	}
	solicitudTerminada := make(chan error, 1)
	go func() {
		solicitudTerminada <- f.ComprobarDisponibilidad(context.Background())
	}()
	<-iniciada
	// El pool vacio solo sirve para superar la validacion previa a la sonda.
	// La sonda de prueba no lo usa y se retira antes de probar Cerrar.
	f.pool = nil
	f.Cerrar()
	select {
	case <-detenida:
	default:
		t.Fatal("Cerrar no espero la sonda de disponibilidad")
	}
	if err := <-solicitudTerminada; err == nil {
		t.Fatal("la solicitud acepto una fuente cerrada")
	}
}
