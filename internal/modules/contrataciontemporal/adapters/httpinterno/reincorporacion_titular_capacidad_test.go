package httpinterno

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
)

type comprobadorCapacidadReincorporacionPrueba struct {
	canal      application.ContextoCanalSeguimiento
	expediente string
	version    uint64
	permitido  bool
	err        error
}

func (c *comprobadorCapacidadReincorporacionPrueba) ComprobarCapacidadReincorporacionTitular(_ context.Context, canal application.ContextoCanalSeguimiento, exp string, v uint64) (bool, error) {
	c.canal, c.expediente, c.version = canal, exp, v
	return c.permitido, c.err
}

func TestCapacidadReincorporacionSoloReflejaPDPDelCanal(t *testing.T) {
	c := &comprobadorCapacidadReincorporacionPrueba{}
	h, err := NuevoManejadorCapacidadReincorporacionTitular(autoridadSeguimientoPrueba{}, c)
	if err != nil {
		t.Fatal(err)
	}
	consultar := func(ruta string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, ruta, nil))
		return w
	}
	ruta := RutaCapacidadReincorporacionTitular + "?expediente_ref=expediente:prueba&version_esperada=8"
	w := consultar(ruta)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"puede_registrar_reincorporacion_titular":false`) ||
		c.canal.OrganizacionRef != "organizacion:prueba" || c.expediente != "expediente:prueba" || c.version != 8 {
		t.Fatalf("denegación preliminar: %d %s", w.Code, w.Body.String())
	}
	c.permitido = true
	if w := consultar(ruta); w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"puede_registrar_reincorporacion_titular":true`) {
		t.Fatalf("permiso preliminar: %d %s", w.Code, w.Body.String())
	}
	if w := consultar(ruta + "&actor_ref=intruso"); w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("actor en query: %d", w.Code)
	}
	c.err = errors.New("PDP indisponible")
	if w := consultar(ruta); w.Code != http.StatusServiceUnavailable || strings.Contains(w.Body.String(), "PDP indisponible") {
		t.Fatalf("PDP indisponible: %d %s", w.Code, w.Body.String())
	}
}
