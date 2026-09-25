package domain

import (
	"reflect"
	"testing"
)

func TestCamposContactoCambiados(t *testing.T) {
	previo := DatosContactoParticipacion{Correo: "a@ejemplo.es", Telefono1: "600000001"}
	casos := []struct {
		nombre   string
		anterior *DatosContactoParticipacion
		nuevo    DatosContactoParticipacion
		esperado []string
	}{
		{"primera versión", nil, previo, []string{CampoTrazaCorreo, CampoTrazaTelefono1}},
		{"sin cambios", &previo, previo, []string{}},
		{"segundo teléfono añadido", &previo, DatosContactoParticipacion{Correo: "a@ejemplo.es", Telefono1: "600000001", Telefono2: "600000002"}, []string{CampoTrazaTelefono2}},
		{"correo y teléfono", &previo, DatosContactoParticipacion{Correo: "b@ejemplo.es", Telefono1: "600000003"}, []string{CampoTrazaCorreo, CampoTrazaTelefono1}},
	}
	for _, c := range casos {
		if got := CamposContactoCambiados(c.anterior, c.nuevo); !reflect.DeepEqual(got, c.esperado) {
			t.Errorf("%s: %v, esperado %v", c.nombre, got, c.esperado)
		}
	}
}
