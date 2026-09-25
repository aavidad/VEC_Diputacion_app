package httpinterno

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// La fiscalización favorable de una modificación tras el nombramiento (CT120)
// publica la vuelta al nombramiento; cualquier otra fase sigue rechazándose.
func TestManejadorFiscalizacionPublicaVueltaAlNombramiento(t *testing.T) {
	for fase, estado := range map[domain.ClaveFase]int{
		domain.FaseNombramiento:    http.StatusCreated,
		domain.FaseInformeJuridico: http.StatusBadGateway,
	} {
		recibo := reciboFiscalizacionHTTPPrueba()
		recibo.Resultado = domain.FiscalizacionFavorable
		recibo.FaseResultante, recibo.EstadoResultante = fase, domain.EstadoEnCurso
		recibo.UnidadRetornoRef, recibo.ResponsableRetornoRef = "", ""
		manejador, err := NuevoManejadorFiscalizacion(
			&autoridadFiscalizacionPrueba{contexto: contextoCanalFiscalizacionPrueba()},
			&ejecutorFiscalizacionPrueba{recibo: recibo},
		)
		if err != nil {
			t.Fatal(err)
		}
		respuesta := httptest.NewRecorder()
		manejador.ServeHTTP(respuesta, nuevaPeticionFiscalizacionPrueba(cuerpoFiscalizacionPrueba("favorable", "")))
		if respuesta.Code != estado && !(estado == http.StatusCreated && respuesta.Code == http.StatusOK) {
			t.Fatalf("%s: estado=%d cuerpo=%s", fase, respuesta.Code, respuesta.Body.String())
		}
	}
}
