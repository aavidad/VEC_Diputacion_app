package domain

import (
	"reflect"
	"testing"
)

const referenciaEnsayo = "hecho:00000000000000000000000000000001"

func declaracionCompleta() DeclaracionActividad {
	return DeclaracionActividad{
		Tipo: ActividadPrivada, ActividadRef: referenciaEnsayo,
		FuncionesRef: referenciaEnsayo, TitularRef: referenciaEnsayo,
		JornadaRef: referenciaEnsayo, HorarioRef: referenciaEnsayo,
		RelacionConPuesto: RelacionDesconocida,
	}
}

func TestDeclaracionCompruebaEstructuraSinDecidirCompatibilidad(t *testing.T) {
	for _, tipo := range []TipoActividad{SegundaActividadPublica, ActividadPrivada, ActividadExceptuada} {
		d := declaracionCompleta()
		d.Tipo = tipo
		if got := d.CamposInvalidos(); len(got) != 0 {
			t.Fatalf("tipo %q: campos inválidos %v", tipo, got)
		}
	}
	d := declaracionCompleta()
	d.Tipo = "concedida"
	d.HorarioRef = "Sábados"
	d.FuncionesRef = "hecho:0000000000000000000000000000000Z"
	d.RelacionConPuesto = ""
	if got, want := d.CamposInvalidos(), []string{"tipo", "funciones_ref", "horario_ref", "relacion_con_puesto"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("campos inválidos: %v, esperado %v", got, want)
	}
}
