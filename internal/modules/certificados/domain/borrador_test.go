package domain

import "testing"

func TestPlantillaVersionadaOrdenaBloquesYConservaLimites(t *testing.T) {
	p := Plantilla{Esquema: "vec.certificados.plantilla-ensayo.v1", ID: "servicios", Version: 2, Estado: "ensayo", Bloques: []string{"persona", "corte", "fuente", "plantilla", "criterio", "limite"}}
	if p.Validar("servicios", 2) != nil {
		t.Fatal("catalogue order rejected")
	}
	p.Bloques[5] = "persona"
	if p.Validar("servicios", 2) == nil {
		t.Fatal("safety block could be omitted")
	}
}
