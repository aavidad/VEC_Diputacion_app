package httpinterno

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type lectorCategoriasRPTPrueba struct {
	llamadas int
	cursor   string
	pagina   PaginaCategoriasRPTB2
	err      error
}

func (l *lectorCategoriasRPTPrueba) ListarCategoriasRPT(_ context.Context, cursor string) (PaginaCategoriasRPTB2, error) {
	l.llamadas++
	l.cursor = cursor
	return l.pagina, l.err
}

func categoriaRPTPrueba(id string) CategoriaRPTB2 {
	return CategoriaRPTB2{CategoriaID: id, Etiqueta: "Administrativo", Atributos: map[string]string{"grupos": "C1"},
		CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("b", 64),
		FuenteRef: "ejemplo:rpt-publica", AprobacionRef: "ejemplo:sin-aprobacion-juridica"}
}

func servirCategoriasRPTPrueba(t *testing.T, l *lectorCategoriasRPTPrueba, r *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	h, err := NuevoManejadorCategoriasRPTB2(l)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

// La página llega tal cual con versión, huella, fuente y aprobación, que son
// los campos que el registro del vínculo copia; el cursor pasa al lector.
func TestCategoriasRPTB2DevuelvePaginaConPublicacion(t *testing.T) {
	l := &lectorCategoriasRPTPrueba{pagina: PaginaCategoriasRPTB2{Categorias: []CategoriaRPTB2{
		categoriaRPTPrueba("categoria:rpt:administrativo"), categoriaRPTPrueba("categoria:rpt:auxiliar-administrativo")},
		HayMas: true, SiguienteCursor: "categoria:rpt:auxiliar-administrativo"}}
	w := servirCategoriasRPTPrueba(t, l, peticionHTTPB2Prueba(http.MethodGet, RutaCategoriasRPTB2+"?cursor=categoria:rpt:a", ""))
	if w.Code != 200 || l.llamadas != 1 || l.cursor != "categoria:rpt:a" {
		t.Fatalf("HTTP %d llamadas=%d cursor=%q: %s", w.Code, l.llamadas, l.cursor, w.Body.String())
	}
	var out struct {
		Data PaginaCategoriasRPTB2 `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil || len(out.Data.Categorias) != 2 || !out.Data.HayMas ||
		out.Data.Categorias[0].CatalogoHuellaSHA256 != strings.Repeat("b", 64) || out.Data.Categorias[0].FuenteRef != "ejemplo:rpt-publica" {
		t.Fatalf("página distinta: %+v %v", out, err)
	}
	vacia := &lectorCategoriasRPTPrueba{}
	w = servirCategoriasRPTPrueba(t, vacia, peticionHTTPB2Prueba(http.MethodGet, RutaCategoriasRPTB2, ""))
	if w.Code != 200 || vacia.cursor != "" || !strings.Contains(w.Body.String(), `"categorias":[]`) {
		t.Fatalf("página vacía: HTTP %d %s", w.Code, w.Body.String())
	}
}

// Consulta, método o cuerpo inesperados se rechazan sin llamar al lector.
func TestCategoriasRPTB2RechazaPeticionesNoPrevistasSinLeer(t *testing.T) {
	casos := []struct {
		r      *http.Request
		estado int
	}{
		{peticionHTTPB2Prueba(http.MethodPost, RutaCategoriasRPTB2, `{}`), 405},
		{peticionHTTPB2Prueba(http.MethodGet, RutaCategoriasRPTB2+"?cursor=A", ""), 400},
		{peticionHTTPB2Prueba(http.MethodGet, RutaCategoriasRPTB2+"?cursor=ab", ""), 400},
		{peticionHTTPB2Prueba(http.MethodGet, RutaCategoriasRPTB2+"?cursor=categoria&cursor=otra", ""), 400},
		{peticionHTTPB2Prueba(http.MethodGet, RutaCategoriasRPTB2+"?cursor=categoria:rpt:a&%zz", ""), 400},
		{peticionHTTPB2Prueba(http.MethodGet, RutaCategoriasRPTB2+"?otro=1&cursor=categoria:rpt:a", ""), 400},
		{peticionHTTPB2Prueba(http.MethodGet, RutaCategoriasRPTB2+"?limite=5", ""), 400},
		{peticionHTTPB2Prueba(http.MethodGet, RutaCategoriasRPTB2+"?expediente_ref=expediente:x", ""), 400},
		{peticionHTTPB2Prueba(http.MethodGet, RutaCategoriasRPTB2+"/", ""), 400},
	}
	conIdentidad := peticionHTTPB2Prueba(http.MethodGet, RutaCategoriasRPTB2, "")
	conIdentidad.Header.Set("X-Usuario", "alguien")
	casos = append(casos, struct {
		r      *http.Request
		estado int
	}{conIdentidad, 400})
	for i, c := range casos {
		l := &lectorCategoriasRPTPrueba{}
		w := servirCategoriasRPTPrueba(t, l, c.r)
		if w.Code != c.estado || l.llamadas != 0 {
			t.Fatalf("caso %d: HTTP %d (se esperaba %d) llamadas=%d", i, w.Code, c.estado, l.llamadas)
		}
	}
}

// Una página incoherente del lector o una negativa no se convierten en 200.
func TestCategoriasRPTB2NoEntregaPaginaIncoherente(t *testing.T) {
	sinHuella := categoriaRPTPrueba("categoria:rpt:b")
	sinHuella.CatalogoHuellaSHA256 = ""
	for i, p := range []PaginaCategoriasRPTB2{
		{Categorias: []CategoriaRPTB2{categoriaRPTPrueba("categoria:rpt:b"), categoriaRPTPrueba("categoria:rpt:a")}},
		{Categorias: []CategoriaRPTB2{sinHuella}},
		{Categorias: []CategoriaRPTB2{categoriaRPTPrueba("categoria:rpt:a")}, HayMas: true, SiguienteCursor: "categoria:rpt:z"},
		{Categorias: []CategoriaRPTB2{categoriaRPTPrueba("categoria:rpt:a")}, SiguienteCursor: "categoria:rpt:a"},
	} {
		w := servirCategoriasRPTPrueba(t, &lectorCategoriasRPTPrueba{pagina: p}, peticionHTTPB2Prueba(http.MethodGet, RutaCategoriasRPTB2, ""))
		if w.Code != 503 {
			t.Fatalf("caso %d: HTTP %d %s", i, w.Code, w.Body.String())
		}
	}
	w := servirCategoriasRPTPrueba(t, &lectorCategoriasRPTPrueba{err: ports.ErrAutorizacionDenegada}, peticionHTTPB2Prueba(http.MethodGet, RutaCategoriasRPTB2, ""))
	if w.Code != 403 {
		t.Fatalf("negativa: HTTP %d", w.Code)
	}
}
