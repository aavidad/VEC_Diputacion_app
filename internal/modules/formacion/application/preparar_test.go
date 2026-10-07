package application

import (
	"reflect"
	"slices"
	"testing"
	"vec-diputacion-granada/internal/modules/formacion/domain"
)

func TestFaltasPendientesSinAprobaciones(t *testing.T) {
	d := domain.Datos{Version: 1, Fuente: domain.Fuente{Referencia: "FOR-001", Version: "2026-10-01", Escenario: "sintetico"}, Referencia: "plan:2027", TituloClave: "formacion.plan.2027", Ediciones: []domain.Edicion{{Referencia: "edicion:1", TituloClave: "formacion.accion.expedientes"}}}
	p, err := Preparar(d)
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"formacion.pendiente.aprobacion_plan", "formacion.pendiente.publicacion", "formacion.pendiente.meritos", "formacion.pendiente.presupuesto_plan"} {
		if !slices.Contains(p.Pendientes, k) {
			t.Errorf("falta %s", k)
		}
	}
	for _, k := range []string{"formacion.pendiente.accion", "formacion.pendiente.fechas_edicion", "formacion.pendiente.plazas_edicion", "formacion.pendiente.modalidad", "formacion.pendiente.prioridad"} {
		if !slices.Contains(p.Ediciones[0].Pendientes, k) {
			t.Errorf("falta %s", k)
		}
	}
	for _, c := range p.Checklist {
		if c.Estado != "pendiente" {
			t.Fatal("emitió aprobación")
		}
	}
	otra, err := Preparar(d)
	if err != nil || !reflect.DeepEqual(p, otra) {
		t.Fatal("proyección no determinista")
	}
	if d.Ediciones[0].Plazas != nil || d.Desde != "" {
		t.Fatal("inventó valores de entrada")
	}
}
