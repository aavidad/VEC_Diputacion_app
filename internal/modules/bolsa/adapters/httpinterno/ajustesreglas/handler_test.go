package ajustesreglas

import (
	"encoding/json"
	"testing"

	"vec-diputacion-granada/internal/vec/reglas"
)

func TestVistaReglaEditableCumpleContratoDePanel(t *testing.T) {
	r := reglas.Regla{Clave: "b05.plazo_respuesta", Etiqueta: "Plazo de respuesta", Cantidad: 2,
		Unidad: reglas.Unidad("dias"), Computo: reglas.Computo("habiles"),
		Edicion: &reglas.Edicion{Campos: []string{"cantidad"}, CantidadMinima: 1, CantidadMaxima: 10}}
	b, err := json.Marshal(vistaRegla(r))
	if err != nil {
		t.Fatal(err)
	}
	var v map[string]any
	if json.Unmarshal(b, &v) != nil {
		t.Fatal("JSON inválido")
	}
	e, ok := v["edicion"].(map[string]any)
	if !ok || e["campos"] == nil || e["opciones_unidad"] == nil || e["opciones_computo"] == nil || e["cantidad_minima"] != float64(1) || v["valores"] == nil {
		t.Fatalf("contrato de editor incompleto: %s", b)
	}
}

func TestJSONDuplicadoRechazado(t *testing.T) {
	if validarJSONSinDuplicados([]byte(`{"version_esperada":1,"version_esperada":2}`)) == nil {
		t.Fatal("clave duplicada aceptada")
	}
	if validarJSONSinDuplicados([]byte(`{"cambios":[{"regla_clave":"b05","campo":"cantidad","nuevo":"2"}]}`)) != nil {
		t.Fatal("solicitud sin duplicados rechazada")
	}
}
