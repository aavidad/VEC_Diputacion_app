package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type autoridadRutasDietasPrueba struct {
	llamadas     int
	err          error
	ruta, metodo string
}

func (a *autoridadRutasDietasPrueba) AutorizarPeticionRutaDietas(_ context.Context, r *http.Request) error {
	a.llamadas++
	a.ruta = r.URL.Path
	a.metodo = r.Method
	return a.err
}

// La prueba entra por la ruta pública: no fabrica Principal ni llama helpers
// internos. La autoridad doble sirve sólo para comprobar el orden de la frontera.
func TestDietasServeHTTPExigeAutoridadNominal(t *testing.T) {
	for _, ruta := range []struct{ url, metodo string }{{"/api/vec/dietas/route-catalog", "GET"}, {"/api/vec/dietas/road-route", "POST"}} {
		sinProveedor := http.StatusServiceUnavailable
		if ruta.url == "/api/vec/dietas/road-route" {
			sinProveedor = http.StatusForbidden
		}
		for _, c := range []struct {
			nombre  string
			err     error
			ausente bool
			estado  int
		}{
			{"sin proveedor", nil, true, sinProveedor}, {"sin identidad", ErrRutaDietasNoAutenticada, false, 401},
			{"denegada", ErrRutaDietasDenegada, false, 403}, {"caida", errors.New("secreto no transportable"), false, 503}, {"concedida", nil, false, 200},
		} {
			t.Run(ruta.metodo+c.nombre, func(t *testing.T) {
				autoridad := &autoridadRutasDietasPrueba{err: c.err}
				datos := 0
				destino := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { datos++; w.WriteHeader(200) })
				opciones := HandlerOptions{ManejadorCatalogoRutaDietas: destino, ManejadorRutaDietas: destino}
				if !c.ausente {
					opciones.AutoridadRutasDietas = autoridad
				}
				h := newTestHandlerWithOptions(t, opciones)
				rr := httptest.NewRecorder()
				h.ServeHTTP(rr, httptest.NewRequest(ruta.metodo, ruta.url, nil))
				if rr.Code != c.estado {
					t.Fatalf("estado %d esperado%d", rr.Code, c.estado)
				}
				if (datos == 1) != (c.estado == 200) {
					t.Fatalf("datos expuestos antes de autorización")
				}
				if !(c.ausente && ruta.url == "/api/vec/dietas/road-route") && rr.Header().Get("Cache-Control") != "no-store" {
					t.Fatal("respuesta cacheable")
				}
				if !c.ausente && (autoridad.llamadas != 1 || autoridad.ruta != ruta.url || autoridad.metodo != ruta.metodo) {
					t.Fatal("autoridad omitida")
				}
			})
		}
	}
}
func TestDietasServeHTTPRechazaMetodosYAliasAntesDeConsumo(t *testing.T) {
	a := &autoridadRutasDietasPrueba{}
	datos := 0
	destino := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { datos++ })
	h := newTestHandlerWithOptions(t, HandlerOptions{AutoridadRutasDietas: a, ManejadorCatalogoRutaDietas: destino, ManejadorRutaDietas: destino})
	for _, c := range []struct {
		metodo, ruta string
		estado       int
	}{
		{"POST", "/api/vec/dietas/route-catalog", 405}, {"GET", "/api/vec/dietas/road-route", 405},
		{"GET", "/api/vec/dietas/route-catalog?perfil=ajeno", 404}, {"GET", "/api/vec/dietas/route-%63atalog", 404},
	} {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest(c.metodo, c.ruta, nil))
		if rr.Code != c.estado {
			t.Fatalf("%s: %d", c.ruta, rr.Code)
		}
	}
	if a.llamadas != 0 || datos != 0 {
		t.Fatal("metodo o alias consumió autorización")
	}
}
