package application

import (
	"slices"
	"sort"

	"vec-diputacion-granada/internal/modules/formacion/domain"
)

type Comprobacion struct{ Clave, Estado, Referencia string }
type EdicionPreparada struct {
	Datos      domain.Edicion
	Pendientes []string
}
type Preparacion struct {
	Plan       domain.Datos
	Ediciones  []EdicionPreparada
	Pendientes []string
	Checklist  []Comprobacion
}

// Preparar revisa el plan sin registrar, seleccionar, certificar o publicar nada.
func Preparar(datos domain.Datos) (Preparacion, error) {
	plan, err := domain.NuevaPreparacion(datos)
	if err != nil {
		return Preparacion{}, err
	}
	d := plan.Datos()
	pendientes := []string{"formacion.pendiente.fuente_maestra", "formacion.pendiente.aprobacion_plan", "formacion.pendiente.publicacion", "formacion.pendiente.criterios_seleccion", "formacion.pendiente.asistencia_certificacion", "formacion.pendiente.meritos"}
	if d.Desde == "" || d.Hasta == "" {
		pendientes = append(pendientes, "formacion.pendiente.fechas_plan")
	}
	if d.Plazas == nil {
		pendientes = append(pendientes, "formacion.pendiente.plazas_plan")
	}
	if d.PresupuestoCentimos == nil {
		pendientes = append(pendientes, "formacion.pendiente.presupuesto_plan")
	}
	if d.Configuracion.Version == "" || len(d.Configuracion.Modalidades) == 0 || len(d.Configuracion.Prioridades) == 0 {
		pendientes = append(pendientes, "formacion.pendiente.configuracion")
	}
	if len(d.Acciones) == 0 {
		pendientes = append(pendientes, "formacion.pendiente.acciones")
	}
	if len(d.Ediciones) == 0 {
		pendientes = append(pendientes, "formacion.pendiente.ediciones")
	}
	acciones := map[string]domain.Accion{}
	for _, a := range d.Acciones {
		acciones[a.Referencia] = a
		if a.NecesidadClave == "" {
			pendientes = append(pendientes, "formacion.pendiente.necesidad")
		}
	}
	out := Preparacion{Plan: d, Ediciones: []EdicionPreparada{}, Pendientes: unicas(pendientes), Checklist: []Comprobacion{}}
	for _, c := range out.Pendientes {
		out.Checklist = append(out.Checklist, Comprobacion{c, "pendiente", d.Referencia})
	}
	ediciones := slices.Clone(d.Ediciones)
	sort.Slice(ediciones, func(i, j int) bool { return ediciones[i].Referencia < ediciones[j].Referencia })
	for _, e := range ediciones {
		p := []string{"formacion.pendiente.aprobacion_edicion", "formacion.pendiente.inscripciones"}
		a, ok := acciones[e.AccionReferencia]
		if !ok {
			p = append(p, "formacion.pendiente.accion")
		}
		if e.NecesidadClave == "" || (ok && a.NecesidadClave != "" && a.NecesidadClave != e.NecesidadClave) {
			p = append(p, "formacion.pendiente.necesidad")
		}
		if e.Desde == "" || e.Hasta == "" {
			p = append(p, "formacion.pendiente.fechas_edicion")
		}
		if e.Plazas == nil {
			p = append(p, "formacion.pendiente.plazas_edicion")
		}
		if e.Horas == nil {
			p = append(p, "formacion.pendiente.horas")
		}
		if e.PresupuestoCentimos == nil {
			p = append(p, "formacion.pendiente.presupuesto_edicion")
		}
		if e.ModalidadClave == "" || !slices.Contains(d.Configuracion.Modalidades, e.ModalidadClave) {
			p = append(p, "formacion.pendiente.modalidad")
		}
		if e.PrioridadClave == "" || !slices.Contains(d.Configuracion.Prioridades, e.PrioridadClave) {
			p = append(p, "formacion.pendiente.prioridad")
		}
		p = unicas(p)
		out.Ediciones = append(out.Ediciones, EdicionPreparada{e, p})
		for _, c := range p {
			out.Checklist = append(out.Checklist, Comprobacion{c, "pendiente", e.Referencia})
		}
	}
	return out, nil
}
func unicas(claves []string) []string { sort.Strings(claves); return slices.Compact(claves) }
