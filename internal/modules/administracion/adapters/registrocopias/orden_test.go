package registrocopias

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"vec-diputacion-granada/internal/modules/administracion/domain/operacionescopias"
	port "vec-diputacion-granada/internal/modules/administracion/ports/registrocopias"
)

func orden(s operacionescopias.Solicitud) port.RecepcionOrden {
	return port.RecepcionOrden{Orden: "orden:uno", Operacion: s.Operacion, SHA256: strings.Repeat("d", 64), SolicitudSHA256: s.SHA256, Destino: s.Destino, Epoca: "epoca:sintetica", Fence: 1}
}

func terminarFallida(t *testing.T, f *Fichero, s operacionescopias.Solicitud) {
	t.Helper()
	h := strings.Repeat("b", 64)
	commands := []operacionescopias.Comando{
		{Clave: "c:0", VersionEsperada: 0, SolicitudSHA256: s.SHA256, Accion: "iniciar_captura"},
		{Clave: "c:1", VersionEsperada: 1, SolicitudSHA256: s.SHA256, Accion: "confirmar_captura", ManifiestoSHA256: h},
		{Clave: "c:2", VersionEsperada: 2, SolicitudSHA256: s.SHA256, Accion: "iniciar_verificacion", ManifiestoSHA256: h, Ejecucion: "ensayo:uno"},
		{Clave: "c:3", VersionEsperada: 3, SolicitudSHA256: s.SHA256, Accion: "declarar_ensayo", Evidencia: &operacionescopias.Evidencia{Modo: "fisico", Conjunto: s.Conjunto, ManifiestoSHA256: h, Ejecucion: "ensayo:uno", Referencia: "evidencia:fallida", SHA256: h, Resultado: "fallido"}},
	}
	for _, cmd := range commands {
		if _, e := f.Aplicar(context.Background(), declaracion, s.Operacion, cmd); e != nil {
			t.Fatal(e)
		}
	}
}

func TestOrdenDurableVinculoUnicidadFenceYReplay(t *testing.T) {
	cfg := pruebaConfig(t)
	f := abrir(t, cfg)
	s := solicitud()
	o := orden(s)
	if _, e := f.AceptarOrden(context.Background(), declaracion, o); !errors.Is(e, port.ErrNoExiste) {
		t.Fatal(e)
	}
	reservar(t, f)
	bad := o
	bad.SolicitudSHA256 = strings.Repeat("e", 64)
	if _, e := f.AceptarOrden(context.Background(), declaracion, bad); !errors.Is(e, operacionescopias.ErrVinculo) {
		t.Fatal(e)
	}
	a, e := f.AceptarOrden(context.Background(), declaracion, o)
	if e != nil || a.Replay || a.Recibo.Referencia == "" {
		t.Fatal(e, a)
	}
	f = abrir(t, cfg)
	r, e := f.AceptarOrden(context.Background(), declaracion, o)
	if e != nil || !r.Replay || r.Recibo != a.Recibo || r.Auditoria.Referencia == a.Auditoria.Referencia {
		t.Fatal(e, r)
	}
	bad = o
	bad.SHA256 = strings.Repeat("f", 64)
	if _, e = f.AceptarOrden(context.Background(), declaracion, bad); !errors.Is(e, port.ErrOrdenConflicto) {
		t.Fatal(e)
	}
	bad = o
	bad.Orden = "orden:otra"
	bad.Fence = 2
	if _, e = f.AceptarOrden(context.Background(), declaracion, bad); !errors.Is(e, port.ErrOrdenConflicto) {
		t.Fatal(e)
	}
	terminarFallida(t, f, s)
	s.Operacion, s.Clave = "op:dos", "clave:dos"
	if _, e = f.Reservar(context.Background(), declaracion, s); e != nil {
		t.Fatal("terminal must release destination", e)
	}
	second := orden(s)
	second.Orden = "orden:dos"
	if _, e = f.AceptarOrden(context.Background(), declaracion, second); !errors.Is(e, port.ErrFence) {
		t.Fatal(e)
	}
	second.Fence = 2
	bad = second
	bad.Epoca = "epoca:otra"
	if _, e = f.AceptarOrden(context.Background(), declaracion, bad); !errors.Is(e, operacionescopias.ErrVinculo) {
		t.Fatal(e)
	}
	if _, e = f.AceptarOrden(context.Background(), declaracion, second); e != nil {
		t.Fatal(e)
	}
	r, e = abrir(t, cfg).AceptarOrden(context.Background(), declaracion, o)
	if e != nil || !r.Replay || r.Recibo != a.Recibo {
		t.Fatal("old exact replay", e)
	}
	q, e := f.Consultar(context.Background(), declaracion, s.Operacion)
	if e != nil || q.Recibo.Version != 0 {
		t.Fatal("acceptance changed progress", e)
	}
}

func TestConcurrentAcceptanceExactlyOneOrder(t *testing.T) {
	cfg := pruebaConfig(t)
	reservar(t, abrir(t, cfg))
	var wg sync.WaitGroup
	out := make(chan error, 2)
	for _, key := range []string{"orden:a", "orden:b"} {
		wg.Add(1)
		go func(key string) {
			defer wg.Done()
			o := orden(solicitud())
			o.Orden = key
			_, e := abrir(t, cfg).AceptarOrden(context.Background(), declaracion, o)
			out <- e
		}(key)
	}
	wg.Wait()
	a, b := <-out, <-out
	if (a == nil) == (b == nil) || (a != nil && !errors.Is(a, port.ErrOrdenConflicto)) || (b != nil && !errors.Is(b, port.ErrOrdenConflicto)) {
		t.Fatal(a, b)
	}
}
