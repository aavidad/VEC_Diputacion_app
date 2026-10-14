package domain

import (
	"bytes"
	"strings"
	"testing"
)

func TestMaterialRecuperacionCorreccionPropiaLigaPasoYVersionInicial(t *testing.T) {
	m := MaterialRecuperacionCorreccionPropia{
		ActorRef: "per_" + strings.Repeat("a", 24), PerfilRef: "prf_" + strings.Repeat("b", 24),
		EmpleadoRef:  "emp_" + strings.Repeat("c", 24),
		SolicitudRef: "correccion:cronos:olvido_0001", ClaveOperacion: "olvido_0001",
	}
	b, err := m.Canonico()
	if err != nil || !bytes.Contains(b, []byte(`"paso":"solicitud"`)) || !bytes.Contains(b, []byte(`"version_recibo":"1"`)) {
		t.Fatal("material inicial", err, string(b))
	}
	ajeno := m
	ajeno.EmpleadoRef = "emp_" + strings.Repeat("d", 24)
	b2, err := ajeno.Canonico()
	if err != nil || bytes.Equal(b, b2) {
		t.Fatal("empleado no liga la autoridad", err)
	}
	m.SolicitudRef = "correccion:cronos:otra_0001"
	if _, err := m.Canonico(); err == nil {
		t.Fatal("solicitud y clave distintas")
	}
}
