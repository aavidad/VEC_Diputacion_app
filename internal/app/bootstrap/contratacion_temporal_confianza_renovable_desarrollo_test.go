package bootstrap

import (
	"context"
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	core "vec-diputacion-granada/internal/vec/domain"
)

type relojRenovableCTPrueba struct{ instante atomic.Int64 }

func (r *relojRenovableCTPrueba) Ahora() time.Time  { return time.UnixMicro(r.instante.Load()).UTC() }
func (r *relojRenovableCTPrueba) fijar(t time.Time) { r.instante.Store(t.UnixMicro()) }

func materialRenovableCTPrueba(t *testing.T, ahora time.Time) materialAtestacionContratacionTemporalDesarrollo {
	t.Helper()
	cfg, _ := generarMaterialDesarrolloPrueba(t)
	composicion, err := NuevaComposicionSeguridadDesarrollo(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	m, err := nuevoMaterialAtestacionContratacionTemporalDesarrollo(composicion.derivadorIdempotencia, ahora)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(m.borrarCopiasEfimeras)
	return m
}
func avanzarMaterialRenovableCTPrueba(t *testing.T, m materialAtestacionContratacionTemporalDesarrollo) materialAtestacionContratacionTemporalDesarrollo {
	t.Helper()
	m.publicadaEn = m.expiraEn
	m.expiraEn = m.expiraEn.Add(24 * time.Hour)
	m.configuracionOrden++
	m.configuracionRef = "confianza:atestacion:ct:desarrollo:" + m.publicadaEn.Format("2006-01-02")
	var err error
	m.configuracion, err = confianza.NuevaConfiguracionConfianzaAtestacionAutorizacionV3(m.configuracionRef, m.configuracionOrden, m.publicadaEn, m.expiraEn, m.raiz)
	if err != nil {
		t.Fatal(err)
	}
	m.configuracionHuella, err = m.configuracion.HuellaSHA256ParaGobierno()
	if err != nil {
		t.Fatal(err)
	}
	return m
}
func fuenteRenovableCTPrueba(t *testing.T, m materialAtestacionContratacionTemporalDesarrollo, r *relojRenovableCTPrueba) *fuenteConfianzaRenovableCTDesarrollo {
	t.Helper()
	v, err := confianza.NuevoServicioConfianzaAtestacionAutorizacionV3(m.configuracion, r)
	if err != nil {
		t.Fatal(err)
	}
	f := &fuenteConfianzaRenovableCTDesarrollo{reloj: r, material: m, actual: v,
		leer: func(context.Context, materialAtestacionContratacionTemporalDesarrollo, time.Time) (materialAtestacionContratacionTemporalDesarrollo, error) {
			return m, nil
		}}
	f.lector, err = f.nuevoLector()
	if err != nil {
		t.Fatal(err)
	}
	return f
}
func TestConfianzaRenovableCTCambioUTCConcurrente(t *testing.T) {
	m := materialRenovableCTPrueba(t, time.Date(2026, 9, 18, 23, 59, 59, 0, time.UTC))
	r := &relojRenovableCTPrueba{}
	r.fijar(m.expiraEn.Add(-time.Microsecond))
	f := fuenteRenovableCTPrueba(t, m, r)
	nuevo := avanzarMaterialRenovableCTPrueba(t, m)
	var llamadas atomic.Int32
	f.renovar = func(context.Context, materialAtestacionContratacionTemporalDesarrollo, time.Time) (materialAtestacionContratacionTemporalDesarrollo, error) {
		llamadas.Add(1)
		return nuevo, nil
	}
	f.leer = func(context.Context, materialAtestacionContratacionTemporalDesarrollo, time.Time) (materialAtestacionContratacionTemporalDesarrollo, error) {
		if llamadas.Load() > 0 {
			return nuevo, nil
		}
		return m, nil
	}
	anterior, err := f.instantanea(context.Background())
	if err != nil || llamadas.Load() != 0 {
		t.Fatalf("renovación prematura: %v", err)
	}
	r.fijar(m.expiraEn)
	var wg sync.WaitGroup
	resultados := make(chan *confianza.ServicioConfianzaAtestacionAutorizacionV3, 32)
	for range 32 {
		wg.Go(func() {
			s, e := f.instantanea(context.Background())
			if e != nil {
				t.Errorf("renovación: %v", e)
			}
			resultados <- s
		})
	}
	wg.Wait()
	close(resultados)
	for s := range resultados {
		if s == nil || s == anterior || s != f.actual {
			t.Fatal("instantáneas divergentes")
		}
	}
	if llamadas.Load() != 1 || f.material.configuracionRef != nuevo.configuracionRef || f.material.expiraEn.Sub(f.material.publicadaEn) != 24*time.Hour {
		t.Fatal("no conserva renovación diaria única")
	}
}
func TestConfianzaRenovableCTFalloConservaCaducadaYRecupera(t *testing.T) {
	m := materialRenovableCTPrueba(t, time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC))
	r := &relojRenovableCTPrueba{}
	r.fijar(m.expiraEn)
	f := fuenteRenovableCTPrueba(t, m, r)
	vieja := f.actual
	fallo := errors.New("publicación rechazada")
	f.renovar = func(context.Context, materialAtestacionContratacionTemporalDesarrollo, time.Time) (materialAtestacionContratacionTemporalDesarrollo, error) {
		return materialAtestacionContratacionTemporalDesarrollo{}, fallo
	}
	for range 2 {
		if s, e := f.instantanea(context.Background()); s != nil || !errors.Is(e, fallo) {
			t.Fatal("aceptó confianza sin publicación")
		}
	}
	if f.actual != vieja || f.material.configuracionRef != m.configuracionRef {
		t.Fatal("publicó memoria ante error")
	}
	nuevo := avanzarMaterialRenovableCTPrueba(t, m)
	f.renovar = func(context.Context, materialAtestacionContratacionTemporalDesarrollo, time.Time) (materialAtestacionContratacionTemporalDesarrollo, error) {
		return nuevo, nil
	}
	f.leer = func(context.Context, materialAtestacionContratacionTemporalDesarrollo, time.Time) (materialAtestacionContratacionTemporalDesarrollo, error) {
		return nuevo, nil
	}
	if s, e := f.instantanea(context.Background()); e != nil || s == vieja {
		t.Fatal("no recuperó publicación confirmada")
	}
	r.fijar(m.publicadaEn)
	if s, e := f.instantanea(context.Background()); e == nil || s != nil {
		t.Fatal("aceptó retroceso del reloj")
	}
	r.fijar(m.validaHasta)
	if s, e := f.instantanea(context.Background()); e == nil || s != nil {
		t.Fatal("renovó raíz caducada")
	}
}

func TestConfianzaRenovableCTRevocacionInmediataYReinicioLector(t *testing.T) {
	m := materialRenovableCTPrueba(t, time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC))
	r := &relojRenovableCTPrueba{}
	r.fijar(m.publicadaEn.Add(time.Hour))
	f := fuenteRenovableCTPrueba(t, m, r)
	f.renovar = func(context.Context, materialAtestacionContratacionTemporalDesarrollo, time.Time) (materialAtestacionContratacionTemporalDesarrollo, error) {
		t.Fatal("no debe publicar antes de caducidad")
		return materialAtestacionContratacionTemporalDesarrollo{}, nil
	}
	var revocada atomic.Bool
	f.leer = func(context.Context, materialAtestacionContratacionTemporalDesarrollo, time.Time) (materialAtestacionContratacionTemporalDesarrollo, error) {
		if revocada.Load() {
			return materialAtestacionContratacionTemporalDesarrollo{}, errors.New("raíz revocada")
		}
		return m, nil
	}
	if _, err := f.instantanea(context.Background()); err != nil {
		t.Fatal(err)
	}
	revocada.Store(true)
	if s, err := f.instantanea(context.Background()); s != nil || err == nil {
		t.Fatal("revocación inmediata ignorada")
	}
	// Un nuevo proceso no conserva una autorización en memoria ante la misma
	// revocación publicada; vuelve a consultar el gobierno.
	reiniciada := fuenteRenovableCTPrueba(t, m, r)
	reiniciada.renovar = f.renovar
	reiniciada.leer = f.leer
	if s, err := reiniciada.instantanea(context.Background()); s != nil || err == nil {
		t.Fatal("reinicio recuperó confianza revocada")
	}
}
func TestConfianzaRenovableCTProveedorYEmisorTomanFuenteCompartida(t *testing.T) {
	m := materialRenovableCTPrueba(t, time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC))
	r := &relojRenovableCTPrueba{}
	r.fijar(m.expiraEn)
	f := fuenteRenovableCTPrueba(t, m, r)
	fallo := errors.New("renovación rechazada")
	var llamadas atomic.Int32
	f.renovar = func(context.Context, materialAtestacionContratacionTemporalDesarrollo, time.Time) (materialAtestacionContratacionTemporalDesarrollo, error) {
		llamadas.Add(1)
		return materialAtestacionContratacionTemporalDesarrollo{}, fallo
	}
	p := &proveedorMaterialAltaContratacionTemporalDesarrollo{confianza: f.actual, fuenteConfianza: f}
	if _, e := p.instantaneaConfianza(context.Background()); !errors.Is(e, fallo) {
		t.Fatal("proveedor ignora fuente")
	}
	e := &emisorMaterialRenovableCTDesarrollo{proveedor: p}
	if _, _, _, err := e.EmitirMaterialAutorizacionAtestadaV3(context.Background(), core.SolicitudAutorizacionLigadaV3{}, core.ResultadoContextoActorRegistradoV2{}); !errors.Is(err, fallo) {
		t.Fatal("emisor ignora fuente")
	}
	if llamadas.Load() != 2 {
		t.Fatal("consumidores no revalidaron fuente compartida")
	}
}

// esperaProgramadaPrueba avanza el reloj simulado lo que pide el temporizador
// y detiene el bucle cuando `parar` lo indica. Registra cada espera.
type esperaProgramadaPrueba struct {
	mu       sync.Mutex
	reloj    *relojRenovableCTPrueba
	esperas  []time.Duration
	cancelar context.CancelFunc
	parar    func(n int) bool
	// congelado mantiene el reloj: el uso concurrente decide cuándo renovar.
	congelado bool
}

func (e *esperaProgramadaPrueba) esperar(ctx context.Context, d time.Duration) error {
	e.mu.Lock()
	e.esperas = append(e.esperas, d)
	n := len(e.esperas)
	e.mu.Unlock()
	if e.parar(n) {
		e.cancelar()
		return ctx.Err()
	}
	if !e.congelado {
		e.reloj.fijar(e.reloj.Ahora().Add(d))
	}
	return ctx.Err()
}

func TestRenovacionProgramadaCambioDeDiaSinTraficoCT(t *testing.T) {
	m := materialRenovableCTPrueba(t, time.Date(2026, 9, 25, 22, 0, 0, 0, time.UTC))
	r := &relojRenovableCTPrueba{}
	r.fijar(m.expiraEn.Add(-90 * time.Minute))
	f := fuenteRenovableCTPrueba(t, m, r)
	dia2 := avanzarMaterialRenovableCTPrueba(t, m)
	dia3 := avanzarMaterialRenovableCTPrueba(t, dia2)
	var renovaciones atomic.Int32
	f.renovar = func(_ context.Context, _ materialAtestacionContratacionTemporalDesarrollo, ahora time.Time) (materialAtestacionContratacionTemporalDesarrollo, error) {
		if ahora.Before(m.expiraEn) {
			t.Error("renovó antes de que el protocolo lo admita")
		}
		if renovaciones.Add(1) == 1 {
			return dia2, nil
		}
		return dia3, nil
	}
	f.leer = func(context.Context, materialAtestacionContratacionTemporalDesarrollo, time.Time) (materialAtestacionContratacionTemporalDesarrollo, error) {
		switch renovaciones.Load() {
		case 0:
			return m, nil
		case 1:
			return dia2, nil
		default:
			return dia3, nil
		}
	}
	ctx, cancelar := context.WithCancel(context.Background())
	defer cancelar()
	espera := &esperaProgramadaPrueba{reloj: r, cancelar: cancelar, parar: func(int) bool { return renovaciones.Load() >= 2 }}
	f.mantenerRenovacionProgramada(ctx, espera.esperar)
	// Hasta la medianoche UTC y después el día siguiente completo, en tramos
	// acotados que se recalculan, sin reintentos. Cada cambio de día renueva
	// una sola vez.
	var total time.Duration
	for _, d := range espera.esperas[:len(espera.esperas)-1] {
		if d > maximoEsperaRenovacionProgramadaCT {
			t.Fatalf("espera sin acotar: %v", d)
		}
		total += d
	}
	if total != 90*time.Minute+24*time.Hour || renovaciones.Load() != 2 {
		t.Fatalf("espera total %v (%d tramos), renovaciones %d", total, len(espera.esperas), renovaciones.Load())
	}
	if f.material.configuracionRef != dia3.configuracionRef || !f.material.expiraEn.Equal(dia3.expiraEn) || f.actual == nil {
		t.Fatal("el temporizador no adoptó la configuración de cada nuevo día")
	}
}

func TestRenovacionProgramadaFalloReintentaSinAdoptar(t *testing.T) {
	m := materialRenovableCTPrueba(t, time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC))
	r := &relojRenovableCTPrueba{}
	r.fijar(m.expiraEn)
	f := fuenteRenovableCTPrueba(t, m, r)
	vieja := f.actual
	nuevo := avanzarMaterialRenovableCTPrueba(t, m)
	var intentos atomic.Int32
	f.renovar = func(context.Context, materialAtestacionContratacionTemporalDesarrollo, time.Time) (materialAtestacionContratacionTemporalDesarrollo, error) {
		if intentos.Add(1) == 1 {
			return materialAtestacionContratacionTemporalDesarrollo{}, errors.New("gobierno no disponible")
		}
		return nuevo, nil
	}
	f.leer = func(context.Context, materialAtestacionContratacionTemporalDesarrollo, time.Time) (materialAtestacionContratacionTemporalDesarrollo, error) {
		if intentos.Load() > 1 {
			return nuevo, nil
		}
		return m, nil
	}
	ctx, cancelar := context.WithCancel(context.Background())
	defer cancelar()
	var adoptadaTrasFallo bool
	espera := &esperaProgramadaPrueba{reloj: r, cancelar: cancelar, parar: func(n int) bool {
		if n == 2 {
			adoptadaTrasFallo = f.actual != vieja
		}
		return n > 3
	}}
	f.mantenerRenovacionProgramada(ctx, espera.esperar)
	if espera.esperas[0] != 0 || espera.esperas[1] != reintentoRenovacionProgramadaCTDesarrollo || adoptadaTrasFallo || intentos.Load() != 2 || f.material.configuracionRef != nuevo.configuracionRef {
		t.Fatalf("reintento incorrecto: esperas %v intentos %d adoptada %t", espera.esperas, intentos.Load(), adoptadaTrasFallo)
	}
}

// El temporizador y el disparo por uso comparten la misma exclusión y la
// misma adopción: en el cambio de día sólo se publica una vez.
func TestRenovacionProgramadaConcurrenteConUsoPublicaUnaVez(t *testing.T) {
	m := materialRenovableCTPrueba(t, time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC))
	r := &relojRenovableCTPrueba{}
	r.fijar(m.expiraEn)
	f := fuenteRenovableCTPrueba(t, m, r)
	nuevo := avanzarMaterialRenovableCTPrueba(t, m)
	var renovaciones atomic.Int32
	f.renovar = func(context.Context, materialAtestacionContratacionTemporalDesarrollo, time.Time) (materialAtestacionContratacionTemporalDesarrollo, error) {
		renovaciones.Add(1)
		return nuevo, nil
	}
	f.leer = func(context.Context, materialAtestacionContratacionTemporalDesarrollo, time.Time) (materialAtestacionContratacionTemporalDesarrollo, error) {
		if renovaciones.Load() > 0 {
			return nuevo, nil
		}
		return m, nil
	}
	ctx, cancelar := context.WithCancel(context.Background())
	defer cancelar()
	var wg sync.WaitGroup
	for range 16 {
		wg.Go(func() {
			if _, err := f.instantanea(context.Background()); err != nil {
				t.Errorf("uso: %v", err)
			}
		})
	}
	espera := &esperaProgramadaPrueba{reloj: r, cancelar: cancelar, parar: func(n int) bool { return n > 1 }, congelado: true}
	f.mantenerRenovacionProgramada(ctx, espera.esperar)
	wg.Wait()
	if renovaciones.Load() != 1 || f.material.configuracionRef != nuevo.configuracionRef {
		t.Fatalf("renovaciones duplicadas: %d", renovaciones.Load())
	}
}

func TestRenovacionProgramadaCancelacionDetieneTemporizador(t *testing.T) {
	m := materialRenovableCTPrueba(t, time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC))
	r := &relojRenovableCTPrueba{}
	r.fijar(m.publicadaEn.Add(time.Hour))
	f := fuenteRenovableCTPrueba(t, m, r)
	f.renovar = func(context.Context, materialAtestacionContratacionTemporalDesarrollo, time.Time) (materialAtestacionContratacionTemporalDesarrollo, error) {
		t.Error("renovó sin vencimiento")
		return materialAtestacionContratacionTemporalDesarrollo{}, nil
	}
	detener := iniciarRenovacionProgramadaCTDesarrollo(f, esperarTemporizadorCTDesarrollo)
	hecho := make(chan struct{})
	go func() {
		detener()
		detener() // idempotente
		close(hecho)
	}()
	select {
	case <-hecho:
	case <-time.After(5 * time.Second):
		t.Fatal("la cancelación no detuvo el temporizador")
	}
	iniciarRenovacionProgramadaCTDesarrollo(nil, esperarTemporizadorCTDesarrollo)()
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if err := esperarTemporizadorCTDesarrollo(ctx, time.Hour); !errors.Is(err, context.Canceled) {
		t.Fatal("la espera ignora la cancelación")
	}
	if err := esperarTemporizadorCTDesarrollo(context.Background(), 0); err != nil {
		t.Fatal(err)
	}
}

// Un intento bloqueado (red cortada, cerrojo consultivo ajeno) vence por su
// propio plazo: el temporizador no retiene f.mu indefinidamente y el uso CT
// vuelve a poder entrar.
func TestRenovacionProgramadaIntentoBloqueadoTienePlazo(t *testing.T) {
	m := materialRenovableCTPrueba(t, time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC))
	r := &relojRenovableCTPrueba{}
	r.fijar(m.expiraEn)
	f := fuenteRenovableCTPrueba(t, m, r)
	f.plazoIntento = 50 * time.Millisecond
	var plazo atomic.Bool
	f.renovar = func(ctx context.Context, _ materialAtestacionContratacionTemporalDesarrollo, _ time.Time) (materialAtestacionContratacionTemporalDesarrollo, error) {
		limite, ok := ctx.Deadline()
		plazo.Store(ok && time.Until(limite) <= f.plazoIntento)
		<-ctx.Done()
		return materialAtestacionContratacionTemporalDesarrollo{}, ctx.Err()
	}
	ctx, cancelar := context.WithCancel(context.Background())
	defer cancelar()
	espera := &esperaProgramadaPrueba{reloj: r, cancelar: cancelar, parar: func(n int) bool { return n > 1 }}
	hecho := make(chan struct{})
	go func() { defer close(hecho); f.mantenerRenovacionProgramada(ctx, espera.esperar) }()
	select {
	case <-hecho:
	case <-time.After(5 * time.Second):
		t.Fatal("el intento programado no respetó su plazo")
	}
	if !plazo.Load() {
		t.Fatal("el intento programado no llevaba plazo propio")
	}
}
