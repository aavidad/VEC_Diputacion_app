package contactopropio

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/shared/i18n"
	"vec-diputacion-granada/internal/vec/ports"
)

type ejecutorContactoPropioPrueba struct {
	llamadas int
	correo   string
	version  uint64
	err      error
}

func (e *ejecutorContactoPropioPrueba) Guardar(_ context.Context, correo string, version uint64) (ports.ReciboContactoUsuario, error) {
	e.llamadas++
	e.correo, e.version = correo, version
	if e.err != nil {
		return ports.ReciboContactoUsuario{}, e.err
	}
	return ports.ReciboContactoUsuario{Version: 7, EvidenciaCentral: ports.EvidenciaAuditoriaCentralContactoUsuario{Referencia: "acc_sintetica_opaca"}}, nil
}

func catalogoContactoPropioPrueba(t *testing.T) *i18n.Catalog {
	t.Helper()
	catalogo, err := i18n.LoadDir("../../../locales")
	if err != nil {
		t.Fatalf("cargar catalogo real: %v", err)
	}
	return catalogo
}

func TestManejadorContactoPropioRechazaEntradaAntesDeInvocarServicio(t *testing.T) {
	casos := []struct {
		nombre, metodo, destino, tipo, cuerpo string
		esperado                              int
	}{
		{"metodo", http.MethodGet, RutaContactoPropio, "application/json", `{"correo":"ana@example.test","version_esperada":1}`, http.StatusMethodNotAllowed},
		{"query", http.MethodPost, RutaContactoPropio + "?perfil=forjado", "application/json", `{"correo":"ana@example.test","version_esperada":1}`, http.StatusNotFound},
		{"tipo", http.MethodPost, RutaContactoPropio, "text/plain", `{"correo":"ana@example.test","version_esperada":1}`, http.StatusBadRequest},
		{"desconocido", http.MethodPost, RutaContactoPropio, "application/json", `{"correo":"ana@example.test","version_esperada":1,"perfil":"forjado"}`, http.StatusBadRequest},
		{"duplicado", http.MethodPost, RutaContactoPropio, "application/json", `{"correo":"ana@example.test","correo":"eve@example.test","version_esperada":1}`, http.StatusBadRequest},
		{"falta correo", http.MethodPost, RutaContactoPropio, "application/json", `{"version_esperada":1}`, http.StatusBadRequest},
		{"version nula", http.MethodPost, RutaContactoPropio, "application/json", `{"correo":"ana@example.test","version_esperada":null}`, http.StatusBadRequest},
		{"crlf", http.MethodPost, RutaContactoPropio, "application/json", "{\"correo\":\"ana@example.test\\r\\nBcc: x@example.test\",\"version_esperada\":1}", http.StatusBadRequest},
		{"demasiado grande", http.MethodPost, RutaContactoPropio, "application/json", `{"correo":"` + strings.Repeat("a", maximoCuerpoContacto) + `@example.test","version_esperada":1}`, http.StatusBadRequest},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			ejecutor := new(ejecutorContactoPropioPrueba)
			h, err := NuevoManejador(ejecutor, catalogoContactoPropioPrueba(t))
			if err != nil {
				t.Fatalf("NuevoManejador: %v", err)
			}
			r := httptest.NewRequest(caso.metodo, caso.destino, strings.NewReader(caso.cuerpo))
			r.Header.Set("Content-Type", caso.tipo)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != caso.esperado || ejecutor.llamadas != 0 {
				t.Fatalf("status=%d llamadas=%d", w.Code, ejecutor.llamadas)
			}
			if strings.Contains(w.Body.String(), "ana@example.test") || strings.Contains(w.Body.String(), "eve@example.test") {
				t.Fatalf("la respuesta revela correo: %q", w.Body.String())
			}
		})
	}
}

func TestManejadorContactoPropioGuardaYPublicaSoloReciboMinimo(t *testing.T) {
	ejecutor := new(ejecutorContactoPropioPrueba)
	h, err := NuevoManejador(ejecutor, catalogoContactoPropioPrueba(t))
	if err != nil {
		t.Fatalf("NuevoManejador: %v", err)
	}
	r := httptest.NewRequest(http.MethodPost, RutaContactoPropio, strings.NewReader(`{"correo":"ana@example.test","version_esperada":0}`))
	r.Header.Set("Content-Type", "application/json; charset=utf-8")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusCreated || ejecutor.llamadas != 1 || ejecutor.correo != "ana@example.test" || ejecutor.version != 0 {
		t.Fatalf("status=%d llamadas=%d correo=%q version=%d", w.Code, ejecutor.llamadas, ejecutor.correo, ejecutor.version)
	}
	if got, want := strings.TrimSpace(w.Body.String()), `{"recibo_ref":"acc_sintetica_opaca","version":7}`; got != want {
		t.Fatalf("respuesta=%s; se esperaba %s", got, want)
	}
	if w.Header().Get("Set-Cookie") != "" || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("cabeceras de privacidad inesperadas: %#v", w.Header())
	}
}

func TestManejadorContactoPropioLocalizaErroresYNoExponeCorreo(t *testing.T) {
	ejecutor := &ejecutorContactoPropioPrueba{err: ErrContactoPropioNoDisponible}
	h, err := NuevoManejador(ejecutor, catalogoContactoPropioPrueba(t))
	if err != nil {
		t.Fatalf("NuevoManejador: %v", err)
	}
	r := httptest.NewRequest(http.MethodPost, RutaContactoPropio, strings.NewReader(`{"correo":"ana@example.test","version_esperada":1}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden || !strings.Contains(w.Body.String(), "Permisos insuficientes") || strings.Contains(w.Body.String(), "ana@example.test") {
		t.Fatalf("respuesta no localizada o reveladora: status=%d body=%q", w.Code, w.Body.String())
	}
	ejecutor.err = ErrContactoPropioInvalido
	r = httptest.NewRequest(http.MethodPost, RutaContactoPropio, strings.NewReader(`{"correo":"ana@example.test","version_esperada":1}`))
	r.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "Solicitud no valida") {
		t.Fatalf("error invalido: status=%d body=%q", w.Code, w.Body.String())
	}
}

func TestNuevoManejadorContactoPropioCierraDependenciasNulas(t *testing.T) {
	catalogo := catalogoContactoPropioPrueba(t)
	if _, err := NuevoManejador(nil, catalogo); !errors.Is(err, ErrManejadorContactoPropioInvalido) {
		t.Fatalf("ejecutor nulo: %v", err)
	}
	if _, err := NuevoManejador(new(ejecutorContactoPropioPrueba), nil); !errors.Is(err, ErrManejadorContactoPropioInvalido) {
		t.Fatalf("catalogo nulo: %v", err)
	}
	if _, err := NuevasRutas(nil, catalogo); !errors.Is(err, ErrManejadorContactoPropioInvalido) {
		t.Fatalf("servicio nulo: %v", err)
	}
}

func TestTextoContactoPropioNoEntraEnPanicoConCatalogoNulo(t *testing.T) {
	if got := textoContactoPropio(nil, claveErrorPeticion); got != claveErrorPeticion {
		t.Fatalf("texto con catalogo nulo=%q", got)
	}
}
