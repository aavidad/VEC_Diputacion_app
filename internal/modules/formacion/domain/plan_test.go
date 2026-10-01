package domain

import "testing"

func ptr[T any](v T) *T { return &v }
func datosValidos() Datos {
	return Datos{Version: 1, Fuente: Fuente{"FOR-001", "2026-10-01", "sintetico"}, Referencia: "plan:2027", TituloClave: "formacion.plan.2027", Desde: "2027-01-01", Hasta: "2027-12-31", PresupuestoCentimos: ptr(int64(100)), Plazas: ptr(10), Configuracion: Configuracion{"v1", []string{"formacion.modalidad.mixta"}, []string{"formacion.prioridad.alta"}}, Acciones: []Accion{{"accion:1", "formacion.accion.expedientes", "formacion.necesidad.expedientes"}}, Ediciones: []Edicion{{Referencia: "edicion:1", TituloClave: "formacion.accion.expedientes", AccionReferencia: "accion:1", Desde: "2027-03-01", Hasta: "2027-03-02", Plazas: ptr(10), Horas: ptr(5), PresupuestoCentimos: ptr(int64(100)), ModalidadClave: "formacion.modalidad.mixta", PrioridadClave: "formacion.prioridad.alta", NecesidadClave: "formacion.necesidad.expedientes"}}}
}
func TestIntegridad(t *testing.T) {
	casos := []struct {
		nombre  string
		cambiar func(*Datos)
	}{
		{"fecha inexistente", func(d *Datos) { d.Ediciones[0].Desde = "2027-02-30" }},
		{"fechas invertidas", func(d *Datos) { d.Ediciones[0].Hasta = "2027-02-01" }},
		{"fuera plan", func(d *Datos) { d.Ediciones[0].Desde = "2026-12-31" }},
		{"solo fin antes plan", func(d *Datos) { d.Ediciones[0].Desde = ""; d.Ediciones[0].Hasta = "2026-12-31" }},
		{"solo inicio despues plan", func(d *Datos) { d.Ediciones[0].Desde = "2028-01-01"; d.Ediciones[0].Hasta = "" }},
		{"cero plazas", func(d *Datos) { d.Ediciones[0].Plazas = ptr(0) }},
		{"plazas negativas", func(d *Datos) { d.Ediciones[0].Plazas = ptr(-1) }},
		{"presupuesto negativo", func(d *Datos) { d.Ediciones[0].PresupuestoCentimos = ptr(int64(-1)) }},
		{"presupuesto excedido", func(d *Datos) { d.PresupuestoCentimos = ptr(int64(99)) }},
		{"duplicado", func(d *Datos) { d.Ediciones = append(d.Ediciones, d.Ediciones[0]) }},
		{"fuente real", func(d *Datos) { d.Fuente.Escenario = "real" }},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			d := datosValidos()
			c.cambiar(&d)
			if _, err := NuevaPreparacion(d); err == nil {
				t.Fatal("aceptó escenario inválido")
			}
		})
	}
}
func TestPlanNoComparteMutaciones(t *testing.T) {
	d := datosValidos()
	p, err := NuevaPreparacion(d)
	if err != nil {
		t.Fatal(err)
	}
	*d.Ediciones[0].Plazas = 9
	d.Configuracion.Modalidades[0] = "formacion.modalidad.otra"
	d.Acciones[0].TituloClave = "formacion.accion.otra"
	x := p.Datos()
	if *x.Ediciones[0].Plazas != 10 || x.Configuracion.Modalidades[0] != "formacion.modalidad.mixta" || x.Acciones[0].TituloClave != "formacion.accion.expedientes" {
		t.Fatal("entrada modificó plan")
	}
	*x.Ediciones[0].Plazas = 1
	x.Configuracion.Prioridades[0] = "formacion.prioridad.otra"
	if *p.Datos().Ediciones[0].Plazas != 10 || p.Datos().Configuracion.Prioridades[0] != "formacion.prioridad.alta" {
		t.Fatal("salida modificó plan")
	}
}
