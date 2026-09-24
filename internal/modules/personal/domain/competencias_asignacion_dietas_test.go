package domain

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestMaterialCompetenciasAsignacionSeLigaSoloAlActor(t *testing.T) {
	actor := actorAsignacionPrueba(t)
	fecha, _ := NuevaFechaCivil("2026-09-20")
	m, err := NuevoMaterialCompetenciasAsignacionDietas(SolicitudCompetenciasAsignacionDietas{Actor: actor, FechaReferencia: fecha})
	if err != nil {
		t.Fatal(err)
	}
	if m.Recurso().Referencia != actor.PersonaRef || m.Recurso().Tipo != "asignacion_dietas_competencias" ||
		m.Recurso().Ambitos["persona_ref"] != actor.PersonaRef || len(m.Recurso().Ambitos) != 1 ||
		m.Recurso().Atributos["operacion"] != "lista" {
		t.Fatal("recurso ajeno al actor")
	}
	var c map[string]any
	if err := json.Unmarshal(m.Canonico(), &c); err != nil {
		t.Fatal(err)
	}
	if len(c) != 3 || c["esquema"] != "vec.personal.asignacion-dietas.competencias.v1" || c["fecha_referencia"] != "2026-09-20" {
		t.Fatal("material inesperado")
	}
	b := m.Canonico()
	b[0] = 'X'
	if bytes.Equal(b, m.Canonico()) {
		t.Fatal("material mutable")
	}
	r := m.Recurso()
	r.Ambitos["persona_ref"] = "per_" + strings.Repeat("b", 24)
	if m.Recurso().Ambitos["persona_ref"] != actor.PersonaRef {
		t.Fatal("ámbito mutable")
	}
}

func TestCompetenciaAsignacionRechazaRolYVigenciaFalsos(t *testing.T) {
	fecha, _ := NuevaFechaCivil("2026-09-20")
	c := CompetenciaAsignacionDietas{AsignacionRef: "ads_" + strings.Repeat("a", 24), RelacionRef: "rel_" + strings.Repeat("a", 24), UnidadRef: "unidad_sintetica", Rol: "administrativo", VigenteDesde: fecha, Version: 1}
	if err := c.Validar(fecha); err != nil {
		t.Fatal(err)
	}
	c.Rol = "rrhh"
	if c.Validar(fecha) == nil {
		t.Fatal("aceptó rol no acreditado")
	}
	c.Rol = "responsable"
	c.VigenteDesde, _ = NuevaFechaCivil("2026-09-21")
	if c.Validar(fecha) == nil {
		t.Fatal("aceptó vigencia futura")
	}
}
