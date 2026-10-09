package httppersonal

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

func TestMiBolsaNombrePropioSoloSiLoResolvioElServicio(t *testing.T) {
	i := puertosbolsa.InstantaneaMiBolsa{ConsultadaEn: time.Date(2026, 10, 9, 10, 0, 0, 0, time.UTC)}
	sin, err := json.Marshal(nuevaRespuesta(i))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(sin), "nombre_visible") || strings.Contains(string(sin), "iniciales") {
		t.Fatalf("nombre inventado: %s", sin)
	}
	i.NombrePropio = &puertosbolsa.NombrePropioMiBolsa{Visible: "Lucía Ortega Pérez", Iniciales: "LO"}
	con, err := json.Marshal(filtrarRespuesta(nuevaRespuesta(i), map[string]bool{}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(con), `"nombre_visible":"Lucía Ortega Pérez"`) || !strings.Contains(string(con), `"iniciales":"LO"`) {
		t.Fatalf("falta el nombre propio: %s", con)
	}
}
