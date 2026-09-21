package domain

import (
	"strings"
	"testing"
)

const participacionContactoPrueba = "participacion:bolsa:prueba:0001"

func TestDatosContactoNormalizaYValida(t *testing.T) {
	d := DatosContactoParticipacion{ParticipacionRef: participacionContactoPrueba, Correo: " Ana.Perez@Dipgra.ES ", Telefono1: "+34 958 24-70 00", Telefono2: "(600) 123 456"}.Normalizar()
	if err := d.Validar(); err != nil {
		t.Fatalf("validar: %v", err)
	}
	if d.Correo != "Ana.Perez@dipgra.es" || d.Telefono1 != "+34958247000" || d.Telefono2 != "600123456" {
		t.Fatalf("normalización: %+v", d)
	}
	claro, err := d.Canonico()
	if err != nil {
		t.Fatalf("canónico: %v", err)
	}
	if strings.Contains(string(claro), participacionContactoPrueba) {
		t.Fatal("el canónico no debe contener la referencia de participación")
	}
	otra, err := DatosContactoParticipacionDesdeCanonico(participacionContactoPrueba, claro)
	if err != nil || !otra.Igual(d) {
		t.Fatalf("ida y vuelta: %v %+v", err, otra)
	}
	m := d.Enmascarados()
	if m.Correo != "A***@dipgra.es" || m.Telefono1 != "***7000" || m.Telefono2 != "***3456" {
		t.Fatalf("enmascarado: %+v", m)
	}
	if !strings.Contains(d.String(), "redactado") {
		t.Fatal("String debe redactar")
	}
}

func TestDatosContactoRechazaLoInvalido(t *testing.T) {
	casos := map[string]DatosContactoParticipacion{
		"sin canal":               {ParticipacionRef: participacionContactoPrueba},
		"participacion vacia":     {Correo: "a@b.es"},
		"correo sin dominio":      {ParticipacionRef: participacionContactoPrueba, Correo: "ana@localhost"},
		"correo con espacios":     {ParticipacionRef: participacionContactoPrueba, Correo: "ana perez@dipgra.es"},
		"telefono corto":          {ParticipacionRef: participacionContactoPrueba, Telefono1: "95824"},
		"telefono fijo raro":      {ParticipacionRef: participacionContactoPrueba, Telefono1: "123456789"},
		"segundo sin primero":     {ParticipacionRef: participacionContactoPrueba, Correo: "a@b.es", Telefono2: "600123456"},
		"telefonos iguales":       {ParticipacionRef: participacionContactoPrueba, Telefono1: "600123456", Telefono2: "600123456"},
		"internacional muy largo": {ParticipacionRef: participacionContactoPrueba, Telefono1: "+3412345678901234"},
	}
	for nombre, d := range casos {
		if err := d.Normalizar().Validar(); err == nil {
			t.Errorf("%s: debía rechazarse", nombre)
		}
	}
	if _, err := DatosContactoParticipacionDesdeCanonico(participacionContactoPrueba, []byte(`{"esquema":"otro","correo":"a@b.es"}`)); err == nil {
		t.Fatal("un canónico de otro esquema debe rechazarse")
	}
}
