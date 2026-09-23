package contactopropio

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/ports"
)

type ejecutorVersionPrueba struct {
	llamadas      int
	errorConsulta error
}

func (e *ejecutorVersionPrueba) VersionPropia(context.Context) (ports.ResultadoVersionContactoUsuario, error) {
	e.llamadas++
	if e.errorConsulta != nil {
		return ports.ResultadoVersionContactoUsuario{}, e.errorConsulta
	}
	return ports.ResultadoVersionContactoUsuario{Encontrado: true, Version: 2}, nil
}

func TestRutaContactoVersionGETNoExponeCorreoNiPermiteSujetoLibre(t *testing.T) {
	post := new(ejecutorContactoPropioPrueba)
	manejadorPost, err := NuevoManejador(post, catalogoContactoPropioPrueba(t))
	if err != nil {
		t.Fatal(err)
	}
	version := new(ejecutorVersionPrueba)
	h := &manejadorContactoConVersion{post: manejadorPost, version: version, catalogo: catalogoContactoPropioPrueba(t)}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaContactoPropio, nil))
	if w.Code != http.StatusOK || strings.TrimSpace(w.Body.String()) != `{"encontrado":true,"version":2}` || version.llamadas != 1 || post.llamadas != 0 || w.Header().Get("Set-Cookie") != "" {
		t.Fatalf("GET selector status=%d cuerpo=%q", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaContactoPropio+"?persona=ajena", nil))
	if w.Code != http.StatusNotFound || version.llamadas != 1 {
		t.Fatal("query de persona cruzó frontera")
	}
	w = httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, RutaContactoPropio, strings.NewReader(`{"correo":"ana@example.test","version_esperada":0}`))
	r.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(w, r)
	if w.Code != http.StatusCreated || post.llamadas != 1 || version.llamadas != 1 {
		t.Fatal("POST dejó de delegar en alta")
	}
}
