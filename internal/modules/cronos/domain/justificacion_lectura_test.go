package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestMaterialesLecturaJustificacionLigadosAlObjetivo(t *testing.T) {
	consulta := MaterialConsultaJustificacion{ActorRef: "per_0123456789abcdefghijkl", PerfilRef: "prf_0123456789abcdefghijkl",
		EmpleadoRef: "emp_0123456789abcdefghijkl", SolicitudRef: "permiso:cronos:solicitud:ensayo001"}
	b, err := consulta.Canonico()
	if err != nil || string(b) != `{"actor_ref":"per_0123456789abcdefghijkl","perfil_ref":"prf_0123456789abcdefghijkl","empleado_ref":"emp_0123456789abcdefghijkl","solicitud_ref":"permiso:cronos:solicitud:ensayo001"}` {
		t.Fatal("consulta no canónica", err, string(b))
	}
	recibo := MaterialReciboJustificacion{ActorRef: consulta.ActorRef, PerfilRef: consulta.PerfilRef, EmpleadoRef: consulta.EmpleadoRef,
		SolicitudRef: consulta.SolicitudRef, ClaveOperacion: "ref:" + strings.Repeat("a", 64), HuellaMaterial: strings.Repeat("b", 64)}
	primero, err := recibo.Canonico()
	if err != nil || HuellaMaterialLecturaJustificacion(primero) == "" {
		t.Fatal("recibo no canónico", err)
	}
	recibo.HuellaMaterial = strings.Repeat("c", 64)
	segundo, err := recibo.Canonico()
	if err != nil || HuellaMaterialLecturaJustificacion(primero) == HuellaMaterialLecturaJustificacion(segundo) {
		t.Fatal("material original sustituible", err)
	}
	recibo.ClaveOperacion = ""
	if _, err := recibo.Canonico(); !errors.Is(err, ErrJustificacionInvalida) {
		t.Fatal("recuperación sin clave", err)
	}
	consulta.EmpleadoRef = ""
	if _, err := consulta.Canonico(); !errors.Is(err, ErrJustificacionInvalida) {
		t.Fatal("consulta sin empleado objetivo", err)
	}
}
