package httppublico

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	catalogosvec "vec-diputacion-granada/internal/modules/bolsa/adapters/catalogosvec"
	ficherobolsa "vec-diputacion-granada/internal/modules/bolsa/adapters/fichero"
	aplicacionbolsa "vec-diputacion-granada/internal/modules/bolsa/application"
	ficherovec "vec-diputacion-granada/internal/vec/adapters/fichero"
)

type relojHTTPFijo struct{}

func (relojHTTPFijo) Ahora() time.Time { return time.Date(2026, 7, 16, 8, 0, 0, 0, time.UTC) }

func handlerPublicoPrueba(t *testing.T) http.Handler {
	t.Helper()
	adaptador, err := ficherobolsa.NuevaConsultaConvocatorias("../../../../../data/demo/convocatorias_publicas.demo.json")
	if err != nil {
		t.Fatal(err)
	}
	paquete, err := ficherovec.NuevaConsultaCatalogos("../../../../../data/catalogos/categorias-profesionales/v1.demo.json")
	if err != nil {
		t.Fatal(err)
	}
	categorias, err := catalogosvec.NuevaConsultaCategorias(paquete, "categorias-profesionales", 1)
	if err != nil {
		t.Fatal(err)
	}
	servicio, err := aplicacionbolsa.NuevoServicioConsultaPublica(adaptador, categorias, relojHTTPFijo{})
	if err != nil {
		t.Fatal(err)
	}
	handler, err := NuevoHandler(servicio)
	if err != nil {
		t.Fatal(err)
	}
	return handler
}

func TestHTTPPublicoListaYDetalleSinIdentidadNiPII(t *testing.T) {
	handler := handlerPublicoPrueba(t)
	peticion := func(conCabeceras bool) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, RutaConvocatorias+"?plazo=abierto", nil)
		if conCabeceras {
			req.Header.Set("X-VEC-Subject", "no-debe-usarse")
			req.Header.Set("X-VEC-Roles", "administrador")
		}
		handler.ServeHTTP(rec, req)
		return rec
	}
	sin, con := peticion(false), peticion(true)
	if sin.Code != http.StatusOK || con.Code != http.StatusOK || sin.Body.String() != con.Body.String() {
		t.Fatalf("respuestas distintas: %d/%d", sin.Code, con.Code)
	}
	for _, prohibido := range []string{`"proceso_ref"`, `"dni"`, `"correo"`, "no-debe-usarse"} {
		if strings.Contains(strings.ToLower(sin.Body.String()), strings.ToLower(prohibido)) {
			t.Fatalf("respuesta contiene %q", prohibido)
		}
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, RutaConvocatorias+"/bolsa-auxiliar-administrativo-demo-2026", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"documentos"`) {
		t.Fatalf("detalle: %d %s", rec.Code, rec.Body.String())
	}
}

func TestHTTPPublicoHEADMetodosYCabeceras(t *testing.T) {
	handler := handlerPublicoPrueba(t)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodHead, RutaConvocatorias, nil))
	if rec.Code != http.StatusOK || rec.Body.Len() != 0 {
		t.Fatalf("HEAD = %d, cuerpo=%q", rec.Code, rec.Body.String())
	}
	cabeceras := map[string]string{
		"Content-Type":                 "application/json",
		"Cache-Control":                "no-store",
		"X-Content-Type-Options":       "nosniff",
		"Content-Security-Policy":      "default-src 'none'",
		"Referrer-Policy":              "no-referrer",
		"Permissions-Policy":           "geolocation=()",
		"Cross-Origin-Resource-Policy": "same-origin",
	}
	for nombre, contiene := range cabeceras {
		if !strings.Contains(rec.Header().Get(nombre), contiene) {
			t.Errorf("%s = %q", nombre, rec.Header().Get(nombre))
		}
	}
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, RutaConvocatorias, nil))
	if rec.Code != http.StatusMethodNotAllowed || rec.Header().Get("Allow") != "GET, HEAD" {
		t.Fatalf("POST = %d, Allow=%q", rec.Code, rec.Header().Get("Allow"))
	}
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodHead, RutaConvocatorias+"?tipo=", nil))
	if rec.Code != http.StatusBadRequest || rec.Body.Len() != 0 {
		t.Fatalf("HEAD erróneo = %d, cuerpo=%q", rec.Code, rec.Body.String())
	}
}

func TestHTTPPublicoCategoriasGETHEADYMinimizacion(t *testing.T) {
	handler := handlerPublicoPrueba(t)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, RutaCategorias, nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"esquema":"vec.bolsa.publico.categorias.v1"`) ||
		!strings.Contains(rec.Body.String(), `"total":58`) {
		t.Fatalf("GET categorias=%d %s", rec.Code, rec.Body.String())
	}
	for _, prohibido := range []string{"source_path", "creado_por", "publicado_por", "aprobacion_ref", "origen_sha256"} {
		if strings.Contains(rec.Body.String(), prohibido) {
			t.Fatalf("respuesta contiene %q", prohibido)
		}
	}
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodHead, RutaCategorias, nil))
	if rec.Code != http.StatusOK || rec.Body.Len() != 0 {
		t.Fatalf("HEAD categorias=%d cuerpo=%q", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, RutaCategorias+"?interno=true", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("consulta categorias no rechazada=%d", rec.Code)
	}
}

func TestHTTPPublicoRechazaConsultaYRutaNoCanonicas(t *testing.T) {
	handler := handlerPublicoPrueba(t)
	for _, ruta := range []string{
		RutaConvocatorias + "?tipo=a&tipo=b", RutaConvocatorias + "?interno=true", RutaConvocatorias + "?tipo=a;b=c",
		RutaConvocatorias + "?texto=", RutaConvocatorias + "?pagina=9999", RutaConvocatorias + "/persona/expediente",
	} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, ruta, nil))
		if rec.Code != http.StatusBadRequest && rec.Code != http.StatusNotFound {
			t.Fatalf("%s = %d", ruta, rec.Code)
		}
	}
	req := httptest.NewRequest(http.MethodGet, RutaConvocatorias+"/bolsa-auxiliar-administrativo-demo-2026", nil)
	req.URL.RawPath = RutaConvocatorias + "/bolsa-auxiliar%2dadministrativo-demo-2026"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("RawPath = %d", rec.Code)
	}
}
