package httpinterno

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestSeleccionLlamamientoPublicaPausaDeComprobacion(t *testing.T) {
	e := clasificarErrorSeleccionLlamamiento(ports.ErrCriteriosOfrecimientoPendientesRRHH)
	if e.estado != http.StatusConflict || e.codigo != "comprobacion_ofrecimiento_pendiente" ||
		e.claveI18n != "api.contratacion_temporal.seleccion_llamamiento.error.comprobacion_ofrecimiento_pendiente" {
		t.Fatalf("pausa sin contrato nominal: %+v", e)
	}
	envuelta := clasificarErrorSeleccionLlamamiento(errors.Join(ports.ErrCriteriosOfrecimientoPendientesRRHH, errors.New("dato interno")))
	if envuelta != e {
		t.Fatalf("envoltura perdió el contrato: %+v", envuelta)
	}
	causa := &ports.ErrorCriteriosOfrecimientoPendientesRRHH{Comprobacion: ports.ComprobacionOfrecimientoPendiente{
		BolsaRef: "bolsa:prueba", OrdenActa: 2, VersionPoliticaOfertas: 3,
		MotivoClave: "pendiente_comprobacion_rrhh",
	}}
	grabador := httptest.NewRecorder()
	responderErrorSeleccionLlamamiento(grabador, httptest.NewRequest(http.MethodPost, "/", nil),
		clasificarErrorSeleccionLlamamiento(causa), causa)
	var cuerpo struct {
		Error struct {
			Codigo string `json:"codigo"`
		} `json:"error"`
		Comprobacion ports.ComprobacionOfrecimientoPendiente `json:"comprobacion_ofrecimiento"`
	}
	if grabador.Code != http.StatusConflict || json.Unmarshal(grabador.Body.Bytes(), &cuerpo) != nil ||
		cuerpo.Error.Codigo != "comprobacion_ofrecimiento_pendiente" || cuerpo.Comprobacion != causa.Comprobacion {
		t.Fatalf("409 perdió posición y regla: %d %s", grabador.Code, grabador.Body.String())
	}
}
