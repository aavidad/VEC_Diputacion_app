package gatewaypersonal

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type sesionesPrueba struct{ cerrada bool }

func (s *sesionesPrueba) Abrir(context.Context, string, string) (string, time.Time, error) {
	return "", time.Time{}, ErrAlmacen
}

func TestActivosAllowlistCabecerasYHead(t *testing.T) {
	raiz := t.TempDir()
	if err := os.Mkdir(filepath.Join(raiz, "locales"), 0o700); err != nil {
		t.Fatal(err)
	}
	for nombre, contenido := range map[string]string{"index.html": "<html>", "acceso.js": "x", "gateway.css": "x", "locales/es.json": "{}"} {
		ruta := filepath.Join(raiz, nombre)
		if err := os.WriteFile(ruta, []byte(contenido), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	h := NuevoHandler(&sesionesPrueba{}, "https://auth.example.test", nil, nil, nil)
	if err := h.ConfigurarActivos(raiz); err != nil {
		t.Fatal(err)
	}
	rutas := h.Rutas()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "https://auth.example.test/acceso/", nil)
	rutas.ServeHTTP(w, r)
	if w.Code != 200 || w.Header().Get("Content-Security-Policy") == "" || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("activo: %d %#v", w.Code, w.Header())
	}
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodHead, "https://auth.example.test/acceso/acceso.js", nil)
	rutas.ServeHTTP(w, r)
	if w.Code != 200 || w.Body.Len() != 0 {
		t.Fatalf("head: %d", w.Code)
	}
	for _, ruta := range []string{"/acceso/../index.html", "/acceso/otro.js", "/acceso/acceso.js?x=1"} {
		w = httptest.NewRecorder()
		r = httptest.NewRequest(http.MethodGet, "https://auth.example.test"+ruta, nil)
		rutas.ServeHTTP(w, r)
		if w.Code != 404 {
			t.Fatalf("ruta admitida %q: %d", ruta, w.Code)
		}
	}
}
func (s *sesionesPrueba) Activa(context.Context, string) (bool, time.Time, error) {
	return true, time.Date(2026, 9, 20, 11, 0, 0, 0, time.UTC), nil
}
func (s *sesionesPrueba) Cerrar(context.Context, string) error { s.cerrada = true; return nil }
func TestSesionNoExponeIdentidadNiAlmacena(t *testing.T) {
	h := NuevoHandler(&sesionesPrueba{}, "https://auth.example.test", nil, nil, nil).Rutas()
	r := httptest.NewRequest(http.MethodGet, "https://auth.example.test/api/acceso/sesion", nil)
	r.AddCookie(&http.Cookie{Name: nombreCookie, Value: "a"})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 || w.Header().Get("Cache-Control") != "no-store" || w.Body.String() != "{\"data\":{\"autenticada\":true,\"expira_en\":\"2026-09-20T11:00:00Z\"}}\n" {
		t.Fatalf("respuesta: %d %q", w.Code, w.Body.String())
	}
}
func TestCerrarExigeOrigenYFetchMetadata(t *testing.T) {
	s := new(sesionesPrueba)
	h := NuevoHandler(s, "https://auth.example.test", nil, nil, nil).Rutas()
	r := httptest.NewRequest(http.MethodPost, "https://auth.example.test/api/acceso/cerrar", nil)
	r.Host = "auth.example.test"
	r.Header.Set("Origin", "https://attacker.test")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden || s.cerrada {
		t.Fatalf("CSRF admitido: %d", w.Code)
	}
}

func TestCerrarRevocaAntesDeBorrarCookie(t *testing.T) {
	s := new(sesionesPrueba)
	h := NuevoHandler(s, "https://auth.example.test", nil, nil, nil).Rutas()
	r := httptest.NewRequest(http.MethodPost, "https://auth.example.test/api/acceso/cerrar", nil)
	r.Host = "auth.example.test"
	r.Header.Set("Origin", "https://auth.example.test")
	r.Header.Set("Sec-Fetch-Site", "same-origin")
	r.Header.Set("Sec-Fetch-Mode", "same-origin")
	r.AddCookie(&http.Cookie{Name: nombreCookie, Value: "sesion"})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusNoContent || !s.cerrada || w.Result().Cookies()[0].MaxAge != -1 {
		t.Fatalf("cierre: %d cerrada=%v", w.Code, s.cerrada)
	}
}
