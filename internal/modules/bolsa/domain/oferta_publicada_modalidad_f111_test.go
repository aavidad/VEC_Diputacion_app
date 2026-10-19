package domain

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestDatosOfertaModalidadYCausaF111(t *testing.T) {
	d := DatosOferta{Categoria: "Auxiliar", Centro: "Residencia", FechaInicio: "2026-10-01", Descripcion: "Sustitución"}
	legado, err := json.Marshal(d)
	if err != nil || string(legado) != `{"categoria":"Auxiliar","centro":"Residencia","fecha_inicio":"2026-10-01","descripcion":"Sustitución"}` {
		t.Fatalf("JSON legado cambiado: %s, %v", legado, err)
	}
	if err := d.Validar(); err != nil {
		t.Fatal(err)
	}
	d.ModalidadClave, d.CausaClave = "interinidad", "sustitucion.temporal"
	if err := d.Validar(); err != nil {
		t.Fatalf("par válido: %v", err)
	}
	for _, par := range [][2]string{{"interinidad", ""}, {"", "sustitucion"}, {"A", "sustitucion"}, {"i", "sustitucion"}, {"interinidad", "causa con espacio"}, {"interinidad", strings.Repeat("a", 81)}} {
		d.ModalidadClave, d.CausaClave = par[0], par[1]
		if err := d.Validar(); !errors.Is(err, ErrDatosOfertaInvalidos) {
			t.Fatalf("par %q aceptado: %v", par, err)
		}
	}
}
