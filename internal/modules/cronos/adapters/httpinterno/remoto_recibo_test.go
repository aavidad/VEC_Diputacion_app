package httpinterno

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type proveedorRecuperacionPrueba struct{}

func (proveedorRecuperacionPrueba) ProveerMaterialRecuperacionMarcajeRemoto(context.Context, domain.MaterialRecuperacionMarcajeRemoto) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, nil
}

type resolverRecuperacionPrueba struct {
	contexto ports.ContextoRecuperacionMarcajeRemoto
	err      error
	llamadas int
}

func (r *resolverRecuperacionPrueba) ResolverRecuperacionMarcajeRemoto(*http.Request) (ports.ContextoRecuperacionMarcajeRemoto, error) {
	r.llamadas++
	return r.contexto, r.err
}

type casoRecuperacionPrueba struct {
	err       error
	llamadas  int
	solicitud ports.SolicitudMarcajePropio
}

func (c *casoRecuperacionPrueba) RecuperarReciboMarcajeRemoto(_ context.Context, _ ports.ContextoRecuperacionMarcajeRemoto, s ports.SolicitudMarcajePropio) (ports.ReciboMarcajePropio, error) {
	c.llamadas++
	c.solicitud = s
	return ports.ReciboMarcajePropio{Referencia: "recibo:cronos:00000000-0000-4000-8000-000000000001", MarcajeOriginalRef: "marcaje:cronos:" + s.ClaveOperacion, InstanteUTC: time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC), Replay: true}, c.err
}

func contextoRecuperacionPrueba(t *testing.T) ports.ContextoRecuperacionMarcajeRemoto {
	t.Helper()
	actor, err := ordenSaldoPrueba(t).ContextoActor()
	if err != nil {
		t.Fatal(err)
	}
	orden, err := ports.NuevaOrdenLecturaMarcajeRemoto(actor, proveedorRecuperacionPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	return ports.ContextoRecuperacionMarcajeRemoto{CanalAcreditado: contextoRemotoPrueba(t, domain.OrigenMarcajeRemoto).CanalAcreditado, OrdenLectura: orden}
}

func peticionRecuperacionPrueba() *http.Request {
	r := httptest.NewRequest(http.MethodGet, RutaRecuperarReciboMarcajeRemoto, nil)
	r.Header.Set(cabeceraClaveOperacionRemota, "op-cronos-0001")
	r.Header.Set(cabeceraMovimientoRemoto, string(domain.PunchEntry))
	return r
}

func TestRecuperacionMarcajeRemotoLeeReciboSinClaveEnURL(t *testing.T) {
	resolver := &resolverRecuperacionPrueba{contexto: contextoRecuperacionPrueba(t)}
	caso := &casoRecuperacionPrueba{}
	h, err := NuevoManejadorRecuperacionMarcajeRemoto(caso, resolver)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionRecuperacionPrueba())
	if w.Code != http.StatusOK || caso.llamadas != 1 || caso.solicitud.ClaveOperacion != "op-cronos-0001" || caso.solicitud.Movimiento != domain.PunchEntry || !strings.Contains(w.Body.String(), `"replay":true`) || w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("Set-Cookie") != "" {
		t.Fatalf("recuperacion no acreditada: %d %s", w.Code, w.Body.String())
	}
}

func TestRecuperacionMarcajeRemotoRechazaEntradaNoCanonica(t *testing.T) {
	for _, alterar := range []func(*http.Request){
		func(r *http.Request) { r.Header.Del(cabeceraClaveOperacionRemota) },
		func(r *http.Request) { r.Header.Add(cabeceraClaveOperacionRemota, "op-cronos-0002") },
		func(r *http.Request) { r.Header.Set(cabeceraClaveOperacionRemota, "op corta") },
		func(r *http.Request) { r.Header.Set(cabeceraMovimientoRemoto, "otro") },
		func(r *http.Request) { r.URL.RawQuery = "clave_operacion=op-cronos-0001" },
		func(r *http.Request) { r.Body = http.NoBody; r.ContentLength = 1 },
	} {
		resolver := &resolverRecuperacionPrueba{contexto: contextoRecuperacionPrueba(t)}
		caso := &casoRecuperacionPrueba{}
		h, _ := NuevoManejadorRecuperacionMarcajeRemoto(caso, resolver)
		r := peticionRecuperacionPrueba()
		alterar(r)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if (w.Code != http.StatusBadRequest && w.Code != http.StatusNotFound) || resolver.llamadas != 0 || caso.llamadas != 0 {
			t.Fatalf("entrada no canonica: %d", w.Code)
		}
	}
}

func TestRecuperacionMarcajeRemotoDistingueAusenciaYError(t *testing.T) {
	for _, tc := range []struct {
		err    error
		estado int
		codigo string
	}{
		{ErrAutenticacionCronosRequerida, 401, "autenticacion_requerida"},
		{ErrAccesoCronosDenegado, 403, "acceso_denegado"},
		{ports.ErrMarcajeRemotoNoEncontrado, 404, "ausencia_confirmada"},
		{errors.Join(ports.ErrMarcajeRemotoNoEncontrado, ports.ErrDependenciaNoDisponible), 503, "no_disponible"},
		{ports.ErrClaveOperacionEnConflicto, 409, "conflicto"},
		{ports.ErrDependenciaNoDisponible, 503, "no_disponible"},
		{errors.New("detalle_privado"), 503, "no_disponible"},
	} {
		resolver := &resolverRecuperacionPrueba{contexto: contextoRecuperacionPrueba(t)}
		caso := &casoRecuperacionPrueba{err: tc.err}
		h, _ := NuevoManejadorRecuperacionMarcajeRemoto(caso, resolver)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionRecuperacionPrueba())
		if w.Code != tc.estado || !strings.Contains(w.Body.String(), `"error":"`+tc.codigo+`"`) || strings.Contains(w.Body.String(), "detalle_privado") {
			t.Fatalf("error recuperacion: %d %s", w.Code, w.Body.String())
		}
	}
}
