package httpinterno

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// Sin la firma que habilita la remisión a Intervención, la API responde un
// conflicto con código propio para que la web explique por qué no se remite.
func TestManejadorFiscalizacionExplicaFirmaRemisionPendiente(t *testing.T) {
	manejador, err := NuevoManejadorFiscalizacion(
		&autoridadFiscalizacionPrueba{contexto: contextoCanalFiscalizacionPrueba()},
		&ejecutorFiscalizacionPrueba{err: ports.ErrFirmaRemisionPendiente},
	)
	if err != nil {
		t.Fatal(err)
	}
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, nuevaPeticionFiscalizacionPrueba(cuerpoFiscalizacionPrueba("favorable", "")))
	var cuerpo struct {
		Error struct {
			Codigo    string `json:"codigo"`
			ClaveI18n string `json:"clave_i18n"`
		} `json:"error"`
	}
	if respuesta.Code != http.StatusConflict || json.Unmarshal(respuesta.Body.Bytes(), &cuerpo) != nil ||
		cuerpo.Error.Codigo != "firma_remision_pendiente" ||
		cuerpo.Error.ClaveI18n != "api.contratacion_temporal.fiscalizacion.error.firma_remision_pendiente" {
		t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body.String())
	}
}
