package httpinterno

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type lectorRetornosHTTPPrueba struct {
	items []ports.ReincorporacionTitularFicha
	err   error
}

func (l lectorRetornosHTTPPrueba) ListarReincorporacionesTitular(context.Context, ports.SolicitudConsultarReincorporacionesTitular) ([]ports.ReincorporacionTitularFicha, error) {
	return l.items, l.err
}

type preparadorRetornosHTTPPrueba struct{}

func (preparadorRetornosHTTPPrueba) PrepararConsultaReincorporacionesTitular(_ context.Context, bolsa, participacion string) (ports.SolicitudConsultarReincorporacionesTitular, error) {
	return ports.SolicitudConsultarReincorporacionesTitular{BolsaRef: bolsa, ParticipacionRef: participacion}, nil
}

const rutaRetornosPrueba = RutaBolsasGestion + "/bolsa:01/candidatos/participacion:01/reincorporaciones-titular"

func peticionRetornos(metodo, ruta string) *http.Request {
	r := httptest.NewRequest(metodo, ruta, nil)
	r.Header.Set("Accept", "application/json")
	return r
}

func TestReincorporacionesTitularFichaYDenegacion(t *testing.T) {
	fecha := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	disponible := time.Date(2027, 2, 1, 0, 0, 0, 0, time.UTC)
	version := int64(3)
	h, err := NuevoHandlerReincorporacionesTitularCT(preparadorRetornosHTTPPrueba{}, lectorRetornosHTTPPrueba{items: []ports.ReincorporacionTitularFicha{{
		EventoRef: "evento:ct:retorno:uno", ExpedienteRef: "expediente:uno", RelacionRef: "relacion:uno",
		FechaEfectiva: fecha, ReciboCTRef: "recibo:ct:uno", CeseEventoRef: "evento:ct:cese:uno",
		Estado: "cese_aplicado", DisponibleDesde: &disponible, ReglaVersion: &version,
		ReglaHuellaSHA256: strings.Repeat("a", 64),
	}}})
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionRetornos(http.MethodGet, rutaRetornosPrueba))
	var body struct {
		Data struct {
			Esquema string           `json:"esquema"`
			Items   []map[string]any `json:"items"`
		} `json:"data"`
	}
	if w.Code != http.StatusOK || w.Header().Get("Cache-Control") != "no-store" ||
		json.Unmarshal(w.Body.Bytes(), &body) != nil || body.Data.Esquema != EsquemaReincorporacionesTitularCT ||
		len(body.Data.Items) != 1 || body.Data.Items[0]["fecha_efectiva"] != "2026-09-01" ||
		body.Data.Items[0]["disponible_desde"] != "2027-02-01" {
		t.Fatalf("respuesta: %d %s", w.Code, w.Body.String())
	}
	denegado, _ := NuevoHandlerReincorporacionesTitularCT(preparadorRetornosHTTPPrueba{}, lectorRetornosHTTPPrueba{err: dominiovec.ErrAutorizacionDenegada})
	w = httptest.NewRecorder()
	denegado.ServeHTTP(w, peticionRetornos(http.MethodGet, rutaRetornosPrueba))
	if w.Code != http.StatusForbidden || strings.Contains(w.Body.String(), "relacion:uno") {
		t.Fatalf("denegación: %d %s", w.Code, w.Body.String())
	}
}

func TestReincorporacionesTitularRutaCerrada(t *testing.T) {
	h, _ := NuevoHandlerReincorporacionesTitularCT(preparadorRetornosHTTPPrueba{}, lectorRetornosHTTPPrueba{})
	for _, c := range []struct {
		ruta   string
		metodo string
		codigo int
	}{
		{rutaRetornosPrueba, http.MethodPost, http.StatusMethodNotAllowed},
		{rutaRetornosPrueba + "?x=1", http.MethodGet, http.StatusNotFound},
		{RutaBolsasGestion + "/bolsa%2F01/candidatos/p/reincorporaciones-titular", http.MethodGet, http.StatusNotFound},
	} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionRetornos(c.metodo, c.ruta))
		if w.Code != c.codigo {
			t.Errorf("%s: %d %s", c.ruta, w.Code, w.Body.String())
		}
	}
}
