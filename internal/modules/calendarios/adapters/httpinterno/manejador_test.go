package httpinterno

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/calendarios/application"
	"vec-diputacion-granada/internal/modules/calendarios/domain"
	"vec-diputacion-granada/internal/modules/calendarios/ports"
)

type consultaPrueba struct {
	centro ports.SolicitudCalendarioCentro
	plazo  ports.SolicitudCalculoPlazo
	fallo  error
}

func (c *consultaPrueba) Centros(_ context.Context, anio int, _ time.Time) ([]ports.CentroConCalendario, error) {
	if c.fallo != nil {
		return nil, c.fallo
	}
	return []ports.CentroConCalendario{{CentroRef: "centro-530", Denominacion: "Recursos Humanos", MunicipioRef: "municipio:ine:18087", VersionID: "v1", Numero: 1}}, nil
}

func (c *consultaPrueba) CalendarioCentro(_ context.Context, s ports.SolicitudCalendarioCentro) (ports.CalendarioCentro, error) {
	c.centro = s
	if c.fallo != nil {
		return ports.CalendarioCentro{}, c.fallo
	}
	return ports.CalendarioCentro{CentroRef: s.CentroRef, Anio: s.Anio, Zona: domain.ZonaOficial}, nil
}

func (c *consultaPrueba) CalcularPlazo(_ context.Context, s ports.SolicitudCalculoPlazo) (ports.ResultadoCalculoPlazo, error) {
	c.plazo = s
	if c.fallo != nil {
		return ports.ResultadoCalculoPlazo{}, c.fallo
	}
	f, _ := domain.ParsearFechaCivil("2026-11-03")
	return ports.ResultadoCalculoPlazo{ResultadoPlazo: domain.ResultadoPlazo{Vencimiento: f, Inicio: f, PrimerDia: f, FinNominal: f}}, nil
}

func pedir(t *testing.T, h http.Handler, metodo, destino string, cabeceras map[string]string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	r := httptest.NewRequest(metodo, destino, nil)
	for k, v := range cabeceras {
		r.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	var cuerpo map[string]any
	if metodo != http.MethodHead {
		if err := json.Unmarshal(w.Body.Bytes(), &cuerpo); err != nil {
			t.Fatalf("%s: respuesta no JSON: %q", destino, w.Body.String())
		}
	}
	if w.Header().Get("Set-Cookie") != "" || w.Header().Get("Cache-Control") != "no-store, no-transform" || w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("%s: cabeceras inseguras %v", destino, w.Header())
	}
	return w, cuerpo
}

func codigo(c map[string]any) string {
	e, _ := c["error"].(map[string]any)
	s, _ := e["codigo"].(string)
	return s
}

func TestConsultasValidas(t *testing.T) {
	c := &consultaPrueba{}
	h := NuevoManejador(c)
	w, cuerpo := pedir(t, h, http.MethodGet, RutaCalendarioCentro+"?centro=centro-530&anio=2026&conocido_en=2026-04-20T10:00:00%2B02:00", nil)
	if w.Code != http.StatusOK || cuerpo["data"] == nil || c.centro.CentroRef != "centro-530" || !c.centro.ConocidoEn.Equal(time.Date(2026, 4, 20, 8, 0, 0, 0, time.UTC)) {
		t.Fatalf("calendario: %d %v %+v", w.Code, cuerpo, c.centro)
	}
	w, _ = pedir(t, h, http.MethodGet, RutaPlazo+"?notificado_en=2026-10-22T22:30:00Z&unidad=dias_naturales&cantidad=10&sede=municipio:ine:18087", nil)
	if w.Code != http.StatusOK || c.plazo.Cantidad != 10 || c.plazo.NotificadoEn.IsZero() || c.plazo.Inicio.EsValida() {
		t.Fatalf("plazo: %d %+v", w.Code, c.plazo)
	}
	w, cuerpo = pedir(t, h, http.MethodGet, RutaCentros+"?anio=2026", nil)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "centro-530") {
		t.Fatalf("centros: %d %v", w.Code, cuerpo)
	}
	w, _ = pedir(t, h, http.MethodHead, RutaCentros+"?anio=2026", nil)
	if w.Code != http.StatusOK || w.Body.Len() != 0 {
		t.Fatalf("HEAD: %d %q", w.Code, w.Body.String())
	}
}

func TestPeticionesRechazadas(t *testing.T) {
	h := NuevoManejador(&consultaPrueba{})
	casos := []struct {
		metodo, destino string
		cabeceras       map[string]string
		estado          int
	}{
		{http.MethodPost, RutaCentros + "?anio=2026", nil, http.StatusMethodNotAllowed},
		{http.MethodGet, RutaCentros, nil, http.StatusBadRequest},
		{http.MethodGet, RutaCentros + "?anio=2026&anio=2027", nil, http.StatusBadRequest},
		{http.MethodGet, RutaCentros + "?anio=02026", nil, http.StatusBadRequest},
		{http.MethodGet, RutaCentros + "?anio=2026&otro=1", nil, http.StatusBadRequest},
		{http.MethodGet, RutaCentros + "?anio=2026", map[string]string{"Cookie": "sesion=1"}, http.StatusBadRequest},
		{http.MethodGet, RutaCentros + "?anio=2026", map[string]string{"X-Forwarded-User": "rrhh"}, http.StatusBadRequest},
		{http.MethodGet, RutaCentros + "?anio=2026", map[string]string{"X-Vec-Rol": "rrhh"}, http.StatusBadRequest},
		{http.MethodGet, RutaCalendarioCentro + "?centro=Centro%20530&anio=2026", nil, http.StatusBadRequest},
		{http.MethodGet, RutaCalendarioCentro + "?centro=centro-530&anio=2026&conocido_en=ayer", nil, http.StatusBadRequest},
		{http.MethodGet, RutaPlazo + "?inicio=2026-01-01&notificado_en=2026-01-01T00:00:00Z&unidad=meses&cantidad=1&sede=municipio:ine:18087", nil, http.StatusBadRequest},
		{http.MethodGet, RutaPlazo + "?inicio=2026-01-01&unidad=horas&cantidad=1&sede=municipio:ine:18087", nil, http.StatusBadRequest},
		{http.MethodGet, RutaPlazo + "?inicio=2026-02-30&unidad=meses&cantidad=1&sede=municipio:ine:18087", nil, http.StatusBadRequest},
		{http.MethodGet, RutaPlazo + "?inicio=2026-01-01&unidad=meses&cantidad=1", nil, http.StatusBadRequest},
		{http.MethodGet, "/api/vec/calendarios/otra", nil, http.StatusNotFound},
	}
	for _, c := range casos {
		w, _ := pedir(t, h, c.metodo, c.destino, c.cabeceras)
		if w.Code != c.estado {
			t.Fatalf("%s %s %v: %d, esperado %d", c.metodo, c.destino, c.cabeceras, w.Code, c.estado)
		}
	}
}

func TestErroresTraducidosSinDetallesInternos(t *testing.T) {
	cobertura := &domain.ErrorCobertura{Anio: 2027, Faltan: []domain.Ambito{{Tipo: domain.AmbitoNacional, Ref: "es"}}}
	w, cuerpo := pedir(t, NuevoManejador(&consultaPrueba{fallo: cobertura}), http.MethodGet, RutaCalendarioCentro+"?centro=centro-530&anio=2027", nil)
	if w.Code != http.StatusUnprocessableEntity || codigo(cuerpo) != "calendario_no_publicado" || !strings.Contains(w.Body.String(), `"faltan":[{"tipo":"nacional","ref":"es"}]`) {
		t.Fatalf("cobertura: %d %s", w.Code, w.Body.String())
	}
	w, cuerpo = pedir(t, NuevoManejador(&consultaPrueba{fallo: errors.New("dial tcp 10.0.0.1: contraseña")}), http.MethodGet, RutaCentros+"?anio=2026", nil)
	if w.Code != http.StatusServiceUnavailable || codigo(cuerpo) != "servicio_no_disponible" || strings.Contains(w.Body.String(), "10.0.0.1") {
		t.Fatalf("fallo interno: %d %s", w.Code, w.Body.String())
	}
	w, cuerpo = pedir(t, NuevoManejador(&consultaPrueba{fallo: application.ErrSolicitudInvalida}), http.MethodGet, RutaCentros+"?anio=1800", nil)
	if w.Code != http.StatusBadRequest || codigo(cuerpo) != "solicitud_invalida" {
		t.Fatalf("solicitud: %d", w.Code)
	}
	w, cuerpo = pedir(t, NuevoManejador(&consultaPrueba{fallo: domain.ErrCalculoNoDeterminado}), http.MethodGet, RutaCentros+"?anio=2026", nil)
	if w.Code != http.StatusUnprocessableEntity || codigo(cuerpo) != "plazo_no_determinado" {
		t.Fatalf("plazo no determinado: %d %s", w.Code, w.Body.String())
	}
	w, cuerpo = pedir(t, NuevoManejador(nil), http.MethodGet, RutaCentros+"?anio=2026", nil)
	if w.Code != http.StatusServiceUnavailable || codigo(cuerpo) != "servicio_no_disponible" {
		t.Fatalf("sin base: %d", w.Code)
	}
}
