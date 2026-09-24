package bootstrap

import (
	"net/http"
	"net/http/httptest"
	"testing"

	personalhttp "vec-diputacion-granada/internal/modules/personal/adapters/httpinterno"
)

// La apertura del alta y la corrección completa de D7 exige el catálogo
// gobernado de validadores competentes de Base. Si alguien rellena la bandera
// o registra esas rutas sin él, esta prueba falla y obliga a revisar el consumo.
func TestAsignacionDietasEscrituraCerradaSinCatalogoCompetente(t *testing.T) {
	if escrituraAsignacionDietasAbierta(catalogoValidadoresCompetentesAsignacionDietas) {
		t.Fatal("alta y corrección D7 abiertas: exigen el catálogo gobernado de validadores competentes publicado por Base y su consumo en Personal")
	}
	relacion := personalhttp.RutaAsignacionesDietas + "/rel_aaaaaaaaaaaaaaaaaaaaaa"
	for _, caso := range []struct {
		ruta, metodo string
		admitido     bool
	}{
		{personalhttp.RutaAsignacionesDietas, http.MethodPost, false},
		{relacion, http.MethodPut, false},
		{relacion, http.MethodGet, true},
		{relacion + "/grupo", http.MethodPut, true},
	} {
		if obtenido := metodoComisionesDietasValido(caso.ruta, caso.metodo); obtenido != caso.admitido {
			t.Fatalf("frontera %s %s admitido=%t", caso.metodo, caso.ruta, obtenido)
		}
	}

	servidas := 0
	manejador := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { servidas++; w.WriteHeader(http.StatusOK) })
	exactas, colecciones := rutasAsignacionDietas(manejador, catalogoValidadoresCompetentesAsignacionDietas)
	for _, exacta := range exactas {
		if exacta.Ruta == personalhttp.RutaAsignacionesDietas {
			t.Fatal("alta D7 registrada sin catálogo competente")
		}
	}
	if len(colecciones) != 1 || colecciones[0].Prefijo != personalhttp.RutaAsignacionesDietas {
		t.Fatalf("colección D7 inesperada: %+v", colecciones)
	}
	for _, caso := range []struct {
		metodo, ruta     string
		estado, servidas int
	}{
		{http.MethodPut, relacion, http.StatusNotFound, 0},
		{http.MethodPost, personalhttp.RutaAsignacionesDietas, http.StatusNotFound, 0},
		{http.MethodGet, relacion, http.StatusOK, 1},
		{http.MethodPut, relacion + "/grupo", http.StatusOK, 2},
	} {
		w := httptest.NewRecorder()
		colecciones[0].Manejador.ServeHTTP(w, httptest.NewRequest(caso.metodo, caso.ruta, nil))
		if w.Code != caso.estado || servidas != caso.servidas || w.Header().Get("Set-Cookie") != "" {
			t.Fatalf("%s %s: estado=%d servidas=%d", caso.metodo, caso.ruta, w.Code, servidas)
		}
	}
}

// Con el catálogo presente, la composición registra alta y corrección tal cual.
func TestAsignacionDietasEscrituraSeAbreSoloConCatalogo(t *testing.T) {
	servidas := 0
	manejador := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { servidas++ })
	exactas, colecciones := rutasAsignacionDietas(manejador, "catalogo:base:validadores-competentes:v1")
	if len(exactas) != 1 || exactas[0].Ruta != personalhttp.RutaAsignacionesDietas || len(colecciones) != 1 {
		t.Fatalf("apertura incompleta: %+v %+v", exactas, colecciones)
	}
	colecciones[0].Manejador.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPut, personalhttp.RutaAsignacionesDietas+"/rel_aaaaaaaaaaaaaaaaaaaaaa", nil))
	if servidas != 1 {
		t.Fatal("corrección no llegó al manejador con catálogo")
	}
	if escrituraAsignacionDietasAbierta("  ") {
		t.Fatal("catálogo en blanco abrió la escritura")
	}
}
