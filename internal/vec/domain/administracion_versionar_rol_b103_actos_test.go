package domain

import (
	"strings"
	"testing"
)

func TestMaterialPropuestaVersionarRolBolsaB103FijaCuatroSelecciones(t *testing.T) {
	c, i, ahora := escenarioVersionarRolBolsaB103(t)
	plan, err := PrepararPlanVersionarRolBolsaB103(c, i, ahora)
	if err != nil {
		t.Fatal(err)
	}
	m := MaterialPropuestaVersionarRolBolsaB103{
		OperacionRef:         "propuesta_admin:" + strings.Repeat("1", 32),
		ProponentePersonaRef: "per_" + strings.Repeat("p", 22), PerfilActivoRef: "prf_" + strings.Repeat("q", 22),
		AsignacionPerfilRef: "asignacion:admin:uno", Plan: plan,
	}
	h, err := m.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	copia := m.Copia()
	copia.Plan.Selecciones[0].EntradaRef = "accion:alterada"
	copia.Plan.Asignaciones[0].Documento.Ambitos[0].Valores[0] = "unidad:alterada"
	if m.Plan.Selecciones[0].EntradaRef == copia.Plan.Selecciones[0].EntradaRef ||
		m.Plan.Asignaciones[0].Documento.Ambitos[0].Valores[0] == copia.Plan.Asignaciones[0].Documento.Ambitos[0].Valores[0] {
		t.Fatal("el material comparte memoria mutable")
	}
	h2, err := m.HuellaSHA256()
	if err != nil || h != h2 {
		t.Fatal("la copia alteró la huella original")
	}
	copia = m.Copia()
	copia.Plan.DefinicionNueva.Concesiones[1].CamposPermitidos[0] = "dni"
	if _, err := copia.HuellaSHA256(); err == nil {
		t.Fatal("material alterado aceptado")
	}
}
