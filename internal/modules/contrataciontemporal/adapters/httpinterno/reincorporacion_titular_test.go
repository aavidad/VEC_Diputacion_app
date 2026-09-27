package httpinterno

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type ejecutorReincorporacionTitularPrueba struct {
	solicitud application.SolicitudRegistrarReincorporacionTitular
	err       error
}

func (e *ejecutorReincorporacionTitularPrueba) RegistrarReincorporacionTitular(_ context.Context, s application.SolicitudRegistrarReincorporacionTitular) (ports.ReciboReincorporacionTitular, error) {
	e.solicitud = s
	return ports.ReciboReincorporacionTitular{
		Operacion:       ports.OperacionRegistrarReincorporacionTitular,
		OrganizacionRef: "organizacion:prueba", ExpedienteRef: "expediente:prueba",
		RelacionRef: "relacion:prueba", FechaEfectiva: "2027-02-15",
		VersionAnterior: 8, VersionResultante: 9,
		ReciboRef: "recibo:retorno", EventoRef: "evento:retorno",
		CeseReciboRef: "recibo:cese", CeseEventoRef: "evento:cese",
		ActorRef: "persona:rrhh", RegistradaEn: time.Date(2027, 2, 15, 12, 0, 0, 0, time.UTC),
	}, e.err
}

func TestReincorporacionTitularHTTPUsaCanalYExponeVinculoCese(t *testing.T) {
	e := &ejecutorReincorporacionTitularPrueba{}
	h, err := NuevoManejadorReincorporacionTitular(autoridadSeguimientoPrueba{}, e)
	if err != nil {
		t.Fatal(err)
	}
	cuerpo := `{"expediente_ref":"expediente:prueba","relacion_ref":"relacion:prueba","fecha_efectiva":"2027-02-15","documento_ref":"documento:retorno","documento_sha256":"` + strings.Repeat("a", 64) + `","version_esperada":8,"clave_idempotencia":"11111111-1111-4111-8111-111111111111"}`
	peticion := func(body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodPost, RutaReincorporacionesTitular, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	w := peticion(cuerpo)
	if w.Code != http.StatusCreated || e.solicitud.Canal.OrganizacionRef != "organizacion:prueba" ||
		e.solicitud.RelacionRef != "relacion:prueba" || !strings.Contains(w.Body.String(), `"cese_evento_ref":"evento:cese"`) ||
		!strings.Contains(w.Body.String(), `"estado_bolsa":"pendiente_confirmacion"`) {
		t.Fatalf("retorno: %d %s", w.Code, w.Body.String())
	}
	if w := peticion(strings.Replace(cuerpo, `"version_esperada":8`, `"version_esperada":8,"actor_ref":"intruso"`, 1)); w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("actor aportado por cliente: %d", w.Code)
	}
	e.err = ports.ErrReincorporacionCeseNoCoincide
	if w := peticion(cuerpo); w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), "cese_no_coincide") {
		t.Fatalf("cese distinto: %d %s", w.Code, w.Body.String())
	}
}
