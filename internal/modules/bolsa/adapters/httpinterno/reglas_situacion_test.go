package httpinterno

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

type reglasSituacionPrueba struct {
	configurada bool
	err         error
}

func (r reglasSituacionPrueba) DestinosSituacion(context.Context, string) ([]string, bool, error) {
	return nil, false, r.err
}
func (r reglasSituacionPrueba) Configurada() bool { return r.configurada }
func (r reglasSituacionPrueba) CausasBaja(context.Context) ([]puertosbolsa.CausaBajaSituacion, error) {
	return nil, r.err
}
func (r reglasSituacionPrueba) ModalidadesReposicion(context.Context) ([]puertosbolsa.ModalidadReposicion, error) {
	return nil, r.err
}
func (r reglasSituacionPrueba) ProponerReposicion(context.Context, time.Time, string) (puertosbolsa.PropuestaReposicion, error) {
	return puertosbolsa.PropuestaReposicion{}, puertosbolsa.ErrReposicionNoCalculable
}

func TestHandlerReglasSituacionContrato(t *testing.T) {
	if _, err := NuevoHandlerReglasSituacion(nil); err == nil {
		t.Fatal("sin reglas no se compone")
	}
	casos := []struct {
		reglas puertosbolsa.ConsultaReglasSituacion
		metodo string
		ruta   string
		estado int
	}{
		{reglasSituacionPrueba{}, http.MethodGet, RutaReglasSituacion, http.StatusOK},
		{reglasSituacionPrueba{}, http.MethodPost, RutaReglasSituacion, http.StatusMethodNotAllowed},
		{reglasSituacionPrueba{}, http.MethodGet, RutaReglasSituacion + "/x", http.StatusNotFound},
		{reglasSituacionPrueba{err: puertosbolsa.ErrReglasSituacionNoDisponibles}, http.MethodGet, RutaReglasSituacion, http.StatusServiceUnavailable},
		{reglasSituacionPrueba{configurada: true}, http.MethodGet, RutaReglasSituacion + "?fin_relacion=2026-01-31", http.StatusBadRequest},
		{reglasSituacionPrueba{}, http.MethodGet, RutaReglasSituacion + "?fin_relacion=" + strings.Repeat("9", 300), http.StatusBadRequest},
	}
	for _, caso := range casos {
		manejador, _ := NuevoHandlerReglasSituacion(caso.reglas)
		grabadora := httptest.NewRecorder()
		manejador.ServeHTTP(grabadora, httptest.NewRequest(caso.metodo, caso.ruta, nil))
		if grabadora.Code != caso.estado || grabadora.Header().Get("Cache-Control") != "no-store" {
			t.Errorf("%s %s: %d, se esperaba %d", caso.metodo, caso.ruta, grabadora.Code, caso.estado)
		}
	}
}
