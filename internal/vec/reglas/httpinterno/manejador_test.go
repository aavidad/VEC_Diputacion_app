package httpinterno

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/adapters/fichero"
	"vec-diputacion-granada/internal/vec/reglas"
)

type relojFijo time.Time

func (r relojFijo) Ahora() time.Time { return time.Time(r) }

type consultaFallida struct{ err error }

func (c consultaFallida) Reglas(context.Context) ([]reglas.Regla, error) { return nil, c.err }

func resolutorEjemplo(t *testing.T, ruta, catalogo, modulo string) *reglas.Resolutor {
	t.Helper()
	consulta, err := fichero.NuevaConsultaCatalogos(ruta)
	if err != nil {
		t.Fatal(err)
	}
	r, err := reglas.NuevoResolutor(reglas.Configuracion{
		Consulta: consulta, Metadatos: consulta, CatalogoID: catalogo, ModuloID: modulo,
		Reloj: relojFijo(time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)),
	})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func pedir(t *testing.T, m http.Handler, metodo, destino string, cabeceras map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(metodo, destino, nil)
	for k, v := range cabeceras {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	m.ServeHTTP(w, req)
	return w
}

func TestReglasVigentesDeLosCatalogosDeEjemplo(t *testing.T) {
	var nulo *reglas.Resolutor
	m, err := NuevoManejador(
		Fuente{Modulo: reglas.ModuloBolsa, CatalogoID: reglas.CatalogoBolsa,
			Consulta: resolutorEjemplo(t, "../../../../data/demo/reglas/bolsa_reglas.ejemplo.demo.json", reglas.CatalogoBolsa, reglas.ModuloBolsa)},
		Fuente{Modulo: reglas.ModuloContratacionTemporal, CatalogoID: reglas.CatalogoContratacionTemporal,
			Consulta: resolutorEjemplo(t, "../../../../data/demo/reglas/ct_reglas.ejemplo.demo.json", reglas.CatalogoContratacionTemporal, reglas.ModuloContratacionTemporal)},
		Fuente{Modulo: "otro", CatalogoID: "vec.otro.reglas", Consulta: nulo},
		Fuente{Modulo: "roto", CatalogoID: "vec.roto.reglas", Consulta: consultaFallida{err: reglas.ErrReglasNoDisponibles}},
	)
	if err != nil {
		t.Fatal(err)
	}
	w := pedir(t, m, http.MethodGet, RutaReglasVigentes, nil)
	if w.Code != http.StatusOK || w.Header().Get("Cache-Control") != "no-store, no-transform" || w.Header().Get("Set-Cookie") != "" {
		t.Fatalf("GET: %d %v", w.Code, w.Header())
	}
	var r Respuesta
	if err := json.Unmarshal(w.Body.Bytes(), &r); err != nil {
		t.Fatal(err)
	}
	if r.Data.Esquema != Esquema || len(r.Data.Catalogos) != 4 {
		t.Fatalf("respuesta: %+v", r.Data)
	}
	estados := []string{EstadoDisponible, EstadoDisponible, EstadoSinCatalogo, EstadoNoDisponible}
	for i, c := range r.Data.Catalogos {
		if c.Estado != estados[i] {
			t.Fatalf("catálogo %d: %q", i, c.Estado)
		}
	}
	bolsa := r.Data.Catalogos[0]
	if !bolsa.PaqueteEjemplo || bolsa.Version < 1 || len(bolsa.HuellaSHA256) != 64 || len(bolsa.Reglas) == 0 {
		t.Fatalf("bolsa: %+v", bolsa)
	}
	var reglamento, ejemplo bool
	for _, regla := range append(bolsa.Reglas, r.Data.Catalogos[1].Reglas...) {
		if regla.Clave == "" || regla.Etiqueta == "" || regla.Duda == "" || regla.Norma == "" || regla.Version < 1 ||
			!strings.Contains(regla.Referencia, ":") || regla.Unidad == "" {
			t.Fatalf("regla incompleta: %+v", regla)
		}
		switch regla.Origen {
		case string(reglas.OrigenReglamento):
			reglamento = reglamento || regla.Articulo != ""
		case string(reglas.OrigenEjemplo):
			ejemplo = true
			if regla.Articulo != "" {
				t.Fatalf("una regla de ejemplo no cita artículo: %+v", regla)
			}
		default:
			t.Fatalf("origen desconocido: %+v", regla)
		}
	}
	if !reglamento || !ejemplo {
		t.Fatal("se esperan reglas del Reglamento y de ejemplo")
	}
	if w := pedir(t, m, http.MethodHead, RutaReglasVigentes, nil); w.Code != http.StatusOK || w.Body.Len() != 0 {
		t.Fatalf("HEAD: %d %d", w.Code, w.Body.Len())
	}
}

func TestReglasVigentesRechazaPeticionesNoCanonicas(t *testing.T) {
	m, err := NuevoManejador(Fuente{Modulo: "bolsa", CatalogoID: reglas.CatalogoBolsa})
	if err != nil {
		t.Fatal(err)
	}
	casos := []struct {
		metodo, destino string
		cabeceras       map[string]string
		estado          int
	}{
		{http.MethodPost, RutaReglasVigentes, nil, http.StatusMethodNotAllowed},
		{http.MethodGet, RutaReglasVigentes + "?modulo=bolsa", nil, http.StatusBadRequest},
		{http.MethodGet, RutaReglasVigentes, map[string]string{"Cookie": "s=1"}, http.StatusBadRequest},
		{http.MethodGet, RutaReglasVigentes, map[string]string{"X-VEC-Rol": "rrhh"}, http.StatusBadRequest},
		{http.MethodGet, "/api/vec/reglas/otra", nil, http.StatusNotFound},
	}
	for _, c := range casos {
		if w := pedir(t, m, c.metodo, c.destino, c.cabeceras); w.Code != c.estado || !strings.Contains(w.Body.String(), "api.reglas.error.") {
			t.Fatalf("%s %s: %d %s", c.metodo, c.destino, w.Code, w.Body.String())
		}
	}
	w := pedir(t, m, http.MethodGet, RutaReglasVigentes, nil)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"estado":"sin_catalogo"`) {
		t.Fatalf("sin catálogo: %d %s", w.Code, w.Body.String())
	}
}

func TestReglasVigentesConfiguracionYCancelacion(t *testing.T) {
	if _, err := NuevoManejador(); err == nil {
		t.Fatal("sin fuentes no se compone")
	}
	if _, err := NuevoManejador(Fuente{Modulo: " bolsa", CatalogoID: "x"}); err == nil {
		t.Fatal("módulo no canónico")
	}
	var m *Manejador
	if w := pedir(t, m, http.MethodGet, RutaReglasVigentes, nil); w.Code != http.StatusServiceUnavailable {
		t.Fatalf("manejador nulo: %d", w.Code)
	}
	cancelada, _ := NuevoManejador(Fuente{Modulo: "bolsa", CatalogoID: reglas.CatalogoBolsa, Consulta: consultaFallida{err: context.Canceled}})
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	req := httptest.NewRequest(http.MethodGet, RutaReglasVigentes, nil).WithContext(ctx)
	w := httptest.NewRecorder()
	cancelada.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("petición cancelada: %d", w.Code)
	}
}
