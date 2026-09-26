package httpinterno

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type errorPublicoPrueba struct {
	Error struct {
		Codigo    string `json:"codigo"`
		ClaveI18n string `json:"clave_i18n"`
	} `json:"error"`
}

// Duda 5: si el catálogo exige informe nuevo tras subsanar, la nueva
// fiscalización sin él responde un conflicto con código propio.
func TestManejadorFiscalizacionExplicaInformeNuevoPendiente(t *testing.T) {
	manejador, err := NuevoManejadorFiscalizacion(
		&autoridadFiscalizacionPrueba{contexto: contextoCanalFiscalizacionPrueba()},
		&ejecutorFiscalizacionPrueba{err: ports.ErrInformeNuevoPendiente},
	)
	if err != nil {
		t.Fatal(err)
	}
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, nuevaPeticionFiscalizacionPrueba(cuerpoFiscalizacionPrueba("favorable", "")))
	var cuerpo errorPublicoPrueba
	if respuesta.Code != http.StatusConflict || json.Unmarshal(respuesta.Body.Bytes(), &cuerpo) != nil ||
		cuerpo.Error.Codigo != "informe_nuevo_pendiente" ||
		cuerpo.Error.ClaveI18n != "api.contratacion_temporal.fiscalizacion.error.informe_nuevo_pendiente" {
		t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body.String())
	}
}

// Un informe nuevo que el catálogo no prevé se rechaza como conflicto.
func TestManejadorInformeJuridicoExplicaInformeNuevoNoPrevisto(t *testing.T) {
	manejador, err := NuevoManejadorInformeJuridico(
		&autoridadInformeJuridicoPrueba{contexto: contextoCanalInformeJuridicoPrueba()},
		&ejecutorInformeJuridicoPrueba{err: ports.ErrInformeNuevoNoPrevisto},
	)
	if err != nil {
		t.Fatal(err)
	}
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, nuevaPeticionInformeJuridicoPrueba(cuerpoInformeJuridicoPrueba()))
	var cuerpo errorPublicoPrueba
	if respuesta.Code != http.StatusConflict || json.Unmarshal(respuesta.Body.Bytes(), &cuerpo) != nil ||
		cuerpo.Error.Codigo != "informe_nuevo_no_previsto" {
		t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body.String())
	}
}
