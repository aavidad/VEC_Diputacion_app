package meritoshttp

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/meritos/application"
	"vec-diputacion-granada/internal/modules/meritos/ports"
)

type proveedorCarteraPrueba struct{ err error }

func (p proveedorCarteraPrueba) PrepararSolicitudCartera(*http.Request) (application.SolicitudCartera, error) {
	return application.SolicitudCartera{}, p.err
}

type lectorCarteraPrueba struct {
	consultas []application.SolicitudCartera
	err       error
}

func (l *lectorCarteraPrueba) Consultar(_ context.Context, q application.SolicitudCartera) (ports.PaginaCartera, error) {
	l.consultas = append(l.consultas, q)
	if l.err != nil {
		return ports.PaginaCartera{}, l.err
	}
	return ports.PaginaCartera{Hechos: []ports.FichaCartera{}, Total: 0,
		Filtro: q.Filtro, CorteEn: time.Date(2026, 10, 12, 14, 0, 0, 0, time.UTC)}, nil
}

func TestCarteraHTTPNoAceptaIdentidadNiEntradaInvalidaSinAuditoria(t *testing.T) {
	lector := &lectorCarteraPrueba{}
	h, err := NuevoHandlerCartera(proveedorCarteraPrueba{}, lector)
	if err != nil {
		t.Fatal(err)
	}
	legitima := httptest.NewRecorder()
	h.ServeHTTP(legitima, httptest.NewRequest(http.MethodGet,
		RutaCartera+"?filtro=retirados&limite=25&lang=en", nil))
	if legitima.Code != http.StatusOK || len(lector.consultas) != 1 ||
		lector.consultas[0].Filtro != "retirados" || lector.consultas[0].Limite != 25 ||
		lector.consultas[0].Idioma != "en" ||
		legitima.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("lectura válida: estado=%d consultas=%+v", legitima.Code, lector.consultas)
	}
	lector.err = ports.ErrCarteraDenegada // El asiento nominal de la entrada inválida quedó confirmado.
	for _, ruta := range []string{
		RutaCartera + "?persona_ref=otra", RutaCartera + "?filtro=todos&filtro=retirados",
		RutaCartera + "?limite=025", RutaCartera + "?cursor=%zz",
		RutaCartera + "?lang=fr", RutaCartera + "?lang=es&lang=en",
	} {
		antes := len(lector.consultas)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, ruta, nil))
		if w.Code != http.StatusBadRequest || len(lector.consultas) != antes+1 ||
			lector.consultas[len(lector.consultas)-1].Filtro != "invalido" {
			t.Fatalf("entrada inválida sin asiento: ruta=%q estado=%d consultas=%d", ruta, w.Code, len(lector.consultas))
		}
	}
	lector.err = ports.ErrCarteraNoDisponible // No hubo asiento: 503, nunca 400 confirmado.
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaCartera+"?persona_ref=otra", nil))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("fallo de auditoría: %d", w.Code)
	}
	sinSesion, err := NuevoHandlerCartera(proveedorCarteraPrueba{err: ports.ErrCarteraSinSesion}, lector)
	if err != nil {
		t.Fatal(err)
	}
	antes := len(lector.consultas)
	w = httptest.NewRecorder()
	sinSesion.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaCartera, nil))
	if w.Code != http.StatusUnauthorized || len(lector.consultas) != antes {
		t.Fatalf("sesión ausente: estado=%d consultas=%d", w.Code, len(lector.consultas))
	}
}

func TestCarteraHTTPRechazaCuerpoYMetodosSinPublicarDatos(t *testing.T) {
	lector := &lectorCarteraPrueba{err: ports.ErrCarteraDenegada}
	h, err := NuevoHandlerCartera(proveedorCarteraPrueba{}, lector)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaCartera, strings.NewReader(`{"persona_ref":"otra"}`)))
	if w.Code != http.StatusBadRequest || len(lector.consultas) != 1 {
		t.Fatalf("cuerpo GET: estado=%d consultas=%d", w.Code, len(lector.consultas))
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, RutaCartera, nil))
	if w.Code != http.StatusMethodNotAllowed || w.Header().Get("Allow") != http.MethodGet ||
		len(lector.consultas) != 1 {
		t.Fatalf("método ajeno: estado=%d consultas=%d", w.Code, len(lector.consultas))
	}
	lector.err = errors.New("causa interna protegida")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaCartera, nil))
	if w.Code != http.StatusServiceUnavailable || strings.Contains(w.Body.String(), "causa interna protegida") {
		t.Fatalf("causa expuesta: estado=%d cuerpo=%s", w.Code, w.Body.String())
	}
}
