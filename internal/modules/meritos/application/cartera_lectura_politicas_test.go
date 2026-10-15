package application

import (
	"encoding/json"
	"testing"

	vec "vec-diputacion-granada/internal/vec/domain"
)

func TestCapturaCarteraSerializaPoliticasEvaluadasComoArray(t *testing.T) {
	politica := vec.ReferenciaHuellaPoliticaLecturaActualV3{
		Referencia: "politica:prueba:v1", HuellaSHA256: "huella-de-prueba",
	}
	casos := []struct {
		nombre   string
		entrada  []vec.ReferenciaHuellaPoliticaLecturaActualV3
		esperado string
	}{
		{"nil", nil, `[]`},
		{"vacia", []vec.ReferenciaHuellaPoliticaLecturaActualV3{}, `[]`},
		{"con politica", []vec.ReferenciaHuellaPoliticaLecturaActualV3{politica}, `[{"referencia":"politica:prueba:v1","huella_sha256":"huella-de-prueba"}]`},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			orden, err := json.Marshal(map[string]any{
				"politicas_evaluadas": politicasEvaluadasCartera(caso.entrada),
			})
			if err != nil {
				t.Fatal(err)
			}
			var campos map[string]json.RawMessage
			if err := json.Unmarshal(orden, &campos); err != nil {
				t.Fatal(err)
			}
			if string(campos["politicas_evaluadas"]) != caso.esperado {
				t.Fatalf("politicas_evaluadas = %s; se esperaba %s", campos["politicas_evaluadas"], caso.esperado)
			}
		})
	}
}
