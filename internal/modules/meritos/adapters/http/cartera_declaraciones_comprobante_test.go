package meritoshttp

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDeclaracionSoloAceptaReferenciaDocumentalOpcional(t *testing.T) {
	base := `{"tipo":"titulacion","denominacion":"Título sintético","fecha_desde":"2026-01-01","declarada_en":"2026-10-14T10:00:00Z","clave_idempotencia":"ref:declaracion"`
	casos := []struct {
		nombre, cuerpo string
		valido         bool
	}{
		{"sin_comprobante", base + `}`, true},
		{"con_comprobante", base + `,"comprobante_ref":"ref:documento_1"}`, true},
		{"evidencia_arbitraria", base + `,"evidencias":[{"id":"ref:ajeno","version":1}]}`, false},
		{"comprobante_nulo", base + `,"comprobante_ref":null}`, false},
		{"comprobante_vacio", base + `,"comprobante_ref":""}`, false},
		{"comprobante_duplicado", base + `,"comprobante_ref":"ref:uno","comprobante_ref":"ref:dos"}`, false},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, RutaDeclaracionesCartera, strings.NewReader(caso.cuerpo))
			r.Header.Set("Content-Type", "application/json")
			entrada, err := leerDeclaracionCartera(httptest.NewRecorder(), r)
			if (err == nil) != caso.valido || caso.nombre == "con_comprobante" && entrada.ComprobanteRef != "ref:documento_1" {
				t.Fatalf("entrada=%+v error=%v", entrada, err)
			}
		})
	}
}
