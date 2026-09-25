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
