package domain

import (
	"errors"
	"regexp"
	"slices"
	"time"
)

const Alcance = "preparacion_sintetica"

type Fuente struct{ Referencia, Version, Escenario string }
type Configuracion struct {
	Version                  string
	Modalidades, Prioridades []string
}
type Accion struct{ Referencia, TituloClave, NecesidadClave string }
type Edicion struct {
	Referencia, AccionReferencia, TituloClave, ModalidadClave string
	Desde, Hasta                                              string
	Plazas, Horas                                             *int
	NecesidadClave                                            string
	PresupuestoCentimos                                       *int64
	PrioridadClave                                            string
}
type Datos struct {
	Version                               int
	Fuente                                Fuente
	Referencia, TituloClave, Desde, Hasta string
	PresupuestoCentimos                   *int64
	Plazas                                *int
	Configuracion                         Configuracion
	Acciones                              []Accion
	Ediciones                             []Edicion
}

// Plan conserva una copia del escenario; no representa publicación o aprobación.
type Plan struct{ datos Datos }

var clave = regexp.MustCompile(`^formacion\.[a-z][a-z0-9_.]{0,119}$`)
var referencia = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9:_-]{0,95}$`)

func NuevaPreparacion(d Datos) (Plan, error) {
	if d.Version != 1 || d.Fuente.Escenario != "sintetico" || d.Fuente.Referencia != "FOR-001" || !referencia.MatchString(d.Fuente.Version) || !referencia.MatchString(d.Referencia) || !clave.MatchString(d.TituloClave) {
		return Plan{}, errors.New("formacion.error.escenario")
	}
	if len(d.Acciones) > 32 || len(d.Ediciones) > 64 || len(d.Configuracion.Modalidades) > 16 || len(d.Configuracion.Prioridades) > 16 {
		return Plan{}, errors.New("formacion.error.limites")
	}
	if err := fechas(d.Desde, d.Hasta); err != nil {
		return Plan{}, err
	}
	if err := cantidades(d.Plazas, nil, d.PresupuestoCentimos); err != nil {
		return Plan{}, err
	}
	if d.Configuracion.Version != "" && !referencia.MatchString(d.Configuracion.Version) {
		return Plan{}, errors.New("formacion.error.configuracion")
	}
	for _, lista := range [][]string{d.Configuracion.Modalidades, d.Configuracion.Prioridades} {
		seen := map[string]bool{}
		for _, c := range lista {
			if !clave.MatchString(c) || seen[c] {
				return Plan{}, errors.New("formacion.error.configuracion")
			}
			seen[c] = true
		}
	}
	acciones := map[string]Accion{}
	for _, a := range d.Acciones {
		if !referencia.MatchString(a.Referencia) || !clave.MatchString(a.TituloClave) || (a.NecesidadClave != "" && !clave.MatchString(a.NecesidadClave)) {
			return Plan{}, errors.New("formacion.error.accion")
		}
		if _, ok := acciones[a.Referencia]; ok {
			return Plan{}, errors.New("formacion.error.duplicado")
		}
		acciones[a.Referencia] = a
	}
	seen := map[string]bool{}
	var total int64
	for _, e := range d.Ediciones {
		if !referencia.MatchString(e.Referencia) || !clave.MatchString(e.TituloClave) || (e.AccionReferencia != "" && !referencia.MatchString(e.AccionReferencia)) {
			return Plan{}, errors.New("formacion.error.edicion")
		}
		if seen[e.Referencia] {
			return Plan{}, errors.New("formacion.error.duplicado")
		}
		seen[e.Referencia] = true
		for _, c := range []string{e.ModalidadClave, e.PrioridadClave, e.NecesidadClave} {
			if c != "" && !clave.MatchString(c) {
				return Plan{}, errors.New("formacion.error.edicion")
			}
		}
		if err := fechas(e.Desde, e.Hasta); err != nil {
			return Plan{}, err
		}
		for _, fecha := range []string{e.Desde, e.Hasta} {
			if fecha != "" && ((d.Desde != "" && fecha < d.Desde) || (d.Hasta != "" && fecha > d.Hasta)) {
				return Plan{}, errors.New("formacion.error.fuera_plan")
			}
		}
		if err := cantidades(e.Plazas, e.Horas, e.PresupuestoCentimos); err != nil {
			return Plan{}, err
		}
		if e.PresupuestoCentimos != nil {
			total += *e.PresupuestoCentimos
		}
	}
	if d.PresupuestoCentimos != nil && total > *d.PresupuestoCentimos {
		return Plan{}, errors.New("formacion.error.presupuesto_plan")
	}
	return Plan{datos: clonar(d)}, nil
}

func fechas(desde, hasta string) error {
	for _, f := range []string{desde, hasta} {
		if f != "" {
			t, err := time.Parse(time.DateOnly, f)
			if err != nil || t.Format(time.DateOnly) != f {
				return errors.New("formacion.error.fecha")
			}
		}
	}
	if desde != "" && hasta != "" && desde > hasta {
		return errors.New("formacion.error.orden_fechas")
	}
	return nil
}
func cantidades(plazas, horas *int, presupuesto *int64) error {
	if plazas != nil && (*plazas <= 0 || *plazas > 100000) {
		return errors.New("formacion.error.plazas")
	}
	if horas != nil && (*horas <= 0 || *horas > 10000) {
		return errors.New("formacion.error.horas")
	}
	if presupuesto != nil && (*presupuesto < 0 || *presupuesto > 100000000000) {
		return errors.New("formacion.error.presupuesto")
	}
	return nil
}
func copia[T any](p *T) *T {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}
func clonar(d Datos) Datos {
	d.Plazas = copia(d.Plazas)
	d.PresupuestoCentimos = copia(d.PresupuestoCentimos)
	d.Configuracion.Modalidades = slices.Clone(d.Configuracion.Modalidades)
	d.Configuracion.Prioridades = slices.Clone(d.Configuracion.Prioridades)
	d.Acciones = slices.Clone(d.Acciones)
	d.Ediciones = slices.Clone(d.Ediciones)
	for i := range d.Ediciones {
		e := &d.Ediciones[i]
		e.Plazas = copia(e.Plazas)
		e.Horas = copia(e.Horas)
		e.PresupuestoCentimos = copia(e.PresupuestoCentimos)
	}
	return d
}
func (p Plan) Datos() Datos { return clonar(p.datos) }
