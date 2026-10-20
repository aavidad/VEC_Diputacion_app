package httppersonal

import (
	"encoding/json"
	"strings"
	"testing"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

// El indicador solo sale cuando la vista ya aplica la elección de modalidades.
func TestRespuestaIndicaOfertasSegunModalidades(t *testing.T) {
	for _, aplicada := range []bool{false, true} {
		cuerpo, err := json.Marshal(nuevaRespuesta(puertosbolsa.InstantaneaMiBolsa{OfertasSegunModalidades: aplicada}))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(cuerpo), `"ofertas_segun_modalidades":true`) != aplicada ||
			strings.Contains(string(cuerpo), `"ofertas_segun_modalidades":false`) {
			t.Fatalf("aplicada=%v cuerpo=%s", aplicada, cuerpo)
		}
	}
}
