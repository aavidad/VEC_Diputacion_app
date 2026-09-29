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
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/reglas"
)

type relojFijo time.Time

func (r relojFijo) Ahora() time.Time { return time.Time(r) }

type consultaFallida struct{ err error }

func (c consultaFallida) Reglas(context.Context) ([]reglas.Regla, error) { return nil, c.err }

type consultaFija struct{ reglas []reglas.Regla }

func (c consultaFija) Reglas(context.Context) ([]reglas.Regla, error) { return c.reglas, nil }

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

type ajustesFijos struct{ v reglas.VersionAjustes }

func (a ajustesFijos) AjustesVigentesEn(context.Context, string, time.Time) (reglas.VersionAjustes, bool, error) {
	return a.v, true, nil
}

// Una regla ajustada no rompe la identidad del catálogo: se muestra con la
// versión y la huella de la base y cita su ajuste.
func TestReglasVigentesConReglaAjustada(t *testing.T) {
	consulta, err := fichero.NuevaConsultaCatalogos("../../../../data/demo/reglas/ct_reglas.ejemplo.demo.json")
	if err != nil {
		t.Fatal(err)
	}
	ajustes := map[string]map[string]string{reglas.CTPlazoFiscalizacion: {reglas.CampoCantidad: "7"}}
	huella, err := reglas.HuellaAjustes(ajustes)
	if err != nil {
		t.Fatal(err)
	}
	r, err := reglas.NuevoResolutor(reglas.Configuracion{
		Consulta: consulta, Metadatos: consulta, CatalogoID: reglas.CatalogoContratacionTemporal,
		ModuloID: reglas.ModuloContratacionTemporal, Reloj: relojFijo(time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)),
		Ajustes: ajustesFijos{reglas.VersionAjustes{
			CatalogoID: reglas.CatalogoAjustesDe(reglas.CatalogoContratacionTemporal), Version: 2, HuellaSHA256: huella,
			VigenteDesde: time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC), Ajustes: ajustes,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	m, err := NuevoManejador(Fuente{Modulo: reglas.ModuloContratacionTemporal, CatalogoID: reglas.CatalogoContratacionTemporal, Consulta: r})
	if err != nil {
		t.Fatal(err)
	}
	var respuesta Respuesta
	if err := json.Unmarshal(pedir(t, m, http.MethodGet, RutaReglasVigentes, nil).Body.Bytes(), &respuesta); err != nil {
		t.Fatal(err)
	}
	c := respuesta.Data.Catalogos[0]
	if c.Estado != EstadoDisponible || c.Version != 1 || len(c.HuellaSHA256) != 64 {
		t.Fatalf("catálogo: %+v", c)
	}
	for _, regla := range c.Reglas {
		if regla.Clave == reglas.CTPlazoFiscalizacion && (regla.Cantidad != 7 || regla.Version != 1 ||
			regla.Origen != string(reglas.OrigenEjemplo) ||
			regla.Referencia != "vec.contratacion_temporal.reglas.ajustes:2:c03.plazo_fiscalizacion") {
			t.Fatalf("regla ajustada: %+v", regla)
		}
	}
}

func TestReglasVigentesConAjusteNoAplicableConservaCatalogoYOtraRegla(t *testing.T) {
	base := domain.ReferenciaEntradaCatalogo{
		CatalogoID: reglas.CatalogoContratacionTemporal, CatalogoVersion: 3,
		CatalogoHuellaSHA256: strings.Repeat("a", 64),
	}
	incompatible := reglas.Regla{
		Clave: reglas.CTPlazoFiscalizacion, Etiqueta: "Fiscalizacion", Descripcion: "Plazo de fiscalizacion",
		Unidad: reglas.UnidadDiasHabiles, Cantidad: 10, Valor: "valor base",
		Origen: reglas.OrigenEjemplo, Norma: "norma", Duda: "pendiente",
		Referencia:        "vec.contratacion_temporal.reglas:3:c03.plazo_fiscalizacion",
		ReferenciaEntrada: base, AjusteNoAplicable: true,
	}
	sana := reglas.Regla{
		Clave: reglas.CTPlazoSubsanacion, Etiqueta: "Subsanacion", Descripcion: "Plazo de subsanacion",
		Unidad: reglas.UnidadDiasHabiles, Cantidad: 5,
		Origen: reglas.OrigenEjemplo, Norma: "norma", Duda: "pendiente",
		Referencia:        "vec.contratacion_temporal.reglas:3:c04.plazo_subsanacion",
		ReferenciaEntrada: base,
	}
	m, err := NuevoManejador(Fuente{
		Modulo: reglas.ModuloContratacionTemporal, CatalogoID: reglas.CatalogoContratacionTemporal,
		Consulta: consultaFija{reglas: []reglas.Regla{incompatible, sana}},
	})
	if err != nil {
		t.Fatal(err)
	}
	w := pedir(t, m, http.MethodGet, RutaReglasVigentes, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("GET: %d %s", w.Code, w.Body.String())
	}
	esperado := `{"data":{"esquema":"vec.reglas.vigentes.v1","catalogos":[{"modulo":"contratacion_temporal","catalogo_id":"vec.contratacion_temporal.reglas","estado":"disponible","version":3,"huella_sha256":"` + strings.Repeat("a", 64) + `","paquete_ejemplo":false,"reglas":[{"clave":"c03.plazo_fiscalizacion","etiqueta":"Fiscalizacion","descripcion":"Plazo de fiscalizacion","unidad":"dias_habiles","ajuste_no_aplicable":true,"origen":"ejemplo","norma":"norma","duda":"pendiente","version":3,"referencia":"vec.contratacion_temporal.reglas:3:c03.plazo_fiscalizacion","paquete_ejemplo":false},{"clave":"c04.plazo_subsanacion","etiqueta":"Subsanacion","descripcion":"Plazo de subsanacion","unidad":"dias_habiles","cantidad":5,"origen":"ejemplo","norma":"norma","duda":"pendiente","version":3,"referencia":"vec.contratacion_temporal.reglas:3:c04.plazo_subsanacion","paquete_ejemplo":false}]}]}}`
	if got := w.Body.String(); got != esperado {
		t.Fatalf("JSON inesperado\nobtenido: %s\nesperado: %s", got, esperado)
	}
}
