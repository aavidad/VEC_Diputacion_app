package httpinterno

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type lectorContratosHTTPPrueba struct {
	items []ports.ContratoParticipacion
	err   error
	q     *ports.SolicitudCambiarSituacionParticipacion
}

func (l lectorContratosHTTPPrueba) ListarContratos(_ context.Context, q ports.SolicitudCambiarSituacionParticipacion) ([]ports.ContratoParticipacion, error) {
	if l.q != nil {
		*l.q = q
	}
	return l.items, l.err
}

const rutaContratosPrueba = RutaBolsasGestion + "/bolsa:01/candidatos/participacion:01/contratos"

func peticionContratos(metodo, ruta string) *http.Request {
	r := httptest.NewRequest(metodo, ruta, nil)
	r.Header.Set("Accept", "application/json")
	return r
}

func TestContratosParticipacionContratoHTTP(t *testing.T) {
	inicio := time.Date(2027, 1, 4, 0, 0, 0, 0, time.UTC)
	fin := time.Date(2027, 3, 31, 0, 0, 0, 0, time.UTC)
	var q ports.SolicitudCambiarSituacionParticipacion
	h, err := NuevoHandlerContratosParticipacion(preparadorSituacionHTTPPrueba{}, lectorContratosHTTPPrueba{q: &q, items: []ports.ContratoParticipacion{
		{EventoRef: "evento:1", Tipo: "incorporacion", Inicio: &inicio, FinPrevisto: &fin, ModalidadClave: "sustitucion", CategoriaRef: "categoria:c2", CausaClave: "necesidad_temporal", ExpedienteRef: "expediente:ct:1", LlamamientoRef: "llamamiento:1", OcurridoEn: inicio},
		{EventoRef: "evento:2", Tipo: "incorporacion", OcurridoEn: inicio},
	}})
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionContratos(http.MethodGet, rutaContratosPrueba))
	if w.Code != http.StatusOK || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("status=%d cabeceras=%v", w.Code, w.Header())
	}
	var cuerpo struct {
		Data struct {
			Esquema string           `json:"esquema"`
			Items   []map[string]any `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &cuerpo); err != nil || cuerpo.Data.Esquema != EsquemaContratosParticipacion || len(cuerpo.Data.Items) != 2 {
		t.Fatalf("cuerpo=%s err=%v", w.Body.String(), err)
	}
	primero, segundo := cuerpo.Data.Items[0], cuerpo.Data.Items[1]
	if primero["inicio"] != "2027-01-04T00:00:00Z" || primero["fin_previsto"] != "2027-03-31T00:00:00Z" || primero["causa_clave"] != "necesidad_temporal" {
		t.Fatalf("primero=%v", primero)
	}
	if segundo["inicio"] != nil || segundo["fin_previsto"] != nil {
		t.Fatalf("fechas ausentes deben ser null: %v", segundo)
	}
	if _, expuesto := primero["llamamiento_ref"]; expuesto {
		t.Fatal("la respuesta no debe exponer el llamamiento")
	}
	if q.BolsaRef != "bolsa:01" || q.ParticipacionRef != "participacion:01" {
		t.Fatalf("solicitud=%+v", q)
	}
}

func TestContratosParticipacionRechazos(t *testing.T) {
	h, _ := NuevoHandlerContratosParticipacion(preparadorSituacionHTTPPrueba{}, lectorContratosHTTPPrueba{})
	casos := []struct {
		nombre string
		r      *http.Request
		estado int
	}{
		{"POST", peticionContratos(http.MethodPost, rutaContratosPrueba), http.StatusMethodNotAllowed},
		{"consulta", peticionContratos(http.MethodGet, rutaContratosPrueba+"?x=1"), http.StatusNotFound},
		{"otra ruta", peticionContratos(http.MethodGet, RutaBolsasGestion+"/bolsa:01/candidatos/participacion:01/otros"), http.StatusNotFound},
		{"escapado", peticionContratos(http.MethodGet, RutaBolsasGestion+"/bolsa%2F01/candidatos/p/contratos"), http.StatusNotFound},
	}
	sinAccept := httptest.NewRequest(http.MethodGet, rutaContratosPrueba, nil)
	casos = append(casos, struct {
		nombre string
		r      *http.Request
		estado int
	}{"sin Accept", sinAccept, http.StatusBadRequest})
	for _, c := range casos {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, c.r)
		if w.Code != c.estado {
			t.Errorf("%s: status=%d body=%s", c.nombre, w.Code, w.Body.String())
		}
	}
}

func TestContratosParticipacionErroresSinDatos(t *testing.T) {
	for _, c := range []struct {
		err    error
		estado int
	}{
		{dominiovec.ErrAutorizacionDenegada, http.StatusForbidden},
		{ports.ErrSituacionParticipacionNoEncontrada, http.StatusNotFound},
		{ports.ErrContratosParticipacionNoDisponible, http.StatusServiceUnavailable},
		{errors.New("detalle interno de PostgreSQL"), http.StatusServiceUnavailable},
	} {
		h, _ := NuevoHandlerContratosParticipacion(preparadorSituacionHTTPPrueba{}, lectorContratosHTTPPrueba{err: c.err})
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionContratos(http.MethodGet, rutaContratosPrueba))
		if w.Code != c.estado || strings.Contains(w.Body.String(), "participacion:01") || strings.Contains(w.Body.String(), "PostgreSQL") {
			t.Errorf("%v: status=%d body=%s", c.err, w.Code, w.Body.String())
		}
	}
	if _, err := NuevoHandlerContratosParticipacion(nil, lectorContratosHTTPPrueba{}); err == nil {
		t.Fatal("handler sin preparador")
	}
}

func TestContratosParticipacionAuditaLaConsulta(t *testing.T) {
	accion, clase, ok := intentoAuditableBorradorLlamamiento(peticionContratos(http.MethodGet, rutaContratosPrueba))
	if !ok || accion == "" || clase == "" {
		t.Fatalf("la consulta B13 debe auditarse: %q %q %v", accion, clase, ok)
	}
}
