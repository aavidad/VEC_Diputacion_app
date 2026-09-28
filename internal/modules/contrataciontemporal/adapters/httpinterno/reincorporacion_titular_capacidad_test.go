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
	consultar := func(ruta, cuerpo string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, ruta, strings.NewReader(cuerpo))
		r.Header.Set("Content-Type", "application/json")
		h.ServeHTTP(w, r)
		return w
	}
	ruta := RutaCapacidadReincorporacionTitular
	cuerpo := `{"expediente_ref":"expediente:prueba","version_esperada":8}`
	w := consultar(ruta, cuerpo)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"puede_registrar_reincorporacion_titular":false`) ||
		w.Header().Get("Cache-Control") != "no-store, no-transform" || w.Header().Get("Set-Cookie") != "" ||
		strings.Contains(w.Body.String(), "expediente:prueba") ||
		c.canal.OrganizacionRef != "organizacion:prueba" || c.expediente != "expediente:prueba" || c.version != 8 {
		t.Fatalf("denegación preliminar: %d %s", w.Code, w.Body.String())
	}
	c.permitido = true
	if w := consultar(ruta, cuerpo); w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"puede_registrar_reincorporacion_titular":true`) {
		t.Fatalf("permiso preliminar: %d %s", w.Code, w.Body.String())
	}
	if w := consultar(ruta, strings.Replace(cuerpo, `"version_esperada":8`, `"version_esperada":8,"actor_ref":"intruso"`, 1)); w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("actor en cuerpo: %d", w.Code)
	}
	if w := consultar(ruta+"?expediente_ref=expediente:prueba", cuerpo); w.Code != http.StatusNotFound {
		t.Fatalf("referencia en URL: %d", w.Code)
	}
	c.err = errors.New("PDP indisponible")
	if w := consultar(ruta, cuerpo); w.Code != http.StatusServiceUnavailable || strings.Contains(w.Body.String(), "PDP indisponible") {
		t.Fatalf("PDP indisponible: %d %s", w.Code, w.Body.String())
	}
}
