package domain

import (
	"strings"
	"testing"
	"time"
)

func TestMarcaOrigenDatosContactoValidaYVence(t *testing.T) {
	marca := MarcaOrigenDatosContacto{
		Origen: OrigenDatosContactoConvoca, VigenteHasta: time.Date(2027, 9, 28, 22, 0, 0, 0, time.UTC),
		UltimoDia: "2027-09-28", ReglaRef: "vec.bolsa.reglas:1:b29.contacto_origen_convoca", ReglaHuella: strings.Repeat("a", 64),
	}
	if err := marca.Validar(); err != nil {
		t.Fatal(err)
	}
	if marca.Estado(time.Date(2027, 9, 28, 21, 59, 59, 0, time.UTC)) != EstadoOrigenContactoVigente ||
		marca.Estado(marca.VigenteHasta) != EstadoOrigenContactoVencido {
		t.Fatal("la marca vence en el primer instante tras su último día")
	}
	if !marca.Igual(marca) {
		t.Fatal("una marca es igual a sí misma")
	}
	for nombre, mutar := range map[string]func(*MarcaOrigenDatosContacto){
		"origen":   func(m *MarcaOrigenDatosContacto) { m.Origen = "persona" },
		"hasta":    func(m *MarcaOrigenDatosContacto) { m.VigenteHasta = time.Time{} },
		"dia":      func(m *MarcaOrigenDatosContacto) { m.UltimoDia = "28/09/2027" },
		"regla":    func(m *MarcaOrigenDatosContacto) { m.ReglaRef = " b29" },
		"huella":   func(m *MarcaOrigenDatosContacto) { m.ReglaHuella = "XYZ" },
		"sinRegla": func(m *MarcaOrigenDatosContacto) { m.ReglaRef = "" },
	} {
		otra := marca
		mutar(&otra)
		if otra.Validar() == nil {
			t.Errorf("%s: debe rechazarse", nombre)
		}
	}
	if !OrigenDatosContactoAdmitido("") || !OrigenDatosContactoAdmitido("convoca") || OrigenDatosContactoAdmitido("persona") {
		t.Fatal("solo se admiten el contacto propio y el de origen CONVOCA")
	}
}
