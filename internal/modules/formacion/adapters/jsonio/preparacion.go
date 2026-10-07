package jsonio

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"io"
	"strings"

	"vec-diputacion-granada/internal/modules/formacion/application"
	"vec-diputacion-granada/internal/modules/formacion/domain"
)

const MaxEntrada = 65536

//go:embed fuentes.json
var fuentesJSON []byte

type Fuente struct {
	Referencia string `json:"referencia"`
	Version    string `json:"version"`
	Escenario  string `json:"escenario"`
}
type FuenteCorporativa struct {
	PlanURL       string `json:"plan_url"`
	PlataformaURL string `json:"plataforma_url"`
}
type Configuracion struct {
	Version     string   `json:"version"`
	Modalidades []string `json:"modalidades"`
	Prioridades []string `json:"prioridades"`
}
type Accion struct {
	Referencia     string `json:"referencia"`
	TituloClave    string `json:"titulo_clave"`
	NecesidadClave string `json:"necesidad_clave"`
}
type Plan struct {
	Referencia          string        `json:"referencia"`
	TituloClave         string        `json:"titulo_clave"`
	Desde               string        `json:"desde"`
	Hasta               string        `json:"hasta"`
	PresupuestoCentimos *int64        `json:"presupuesto_centimos"`
	Plazas              *int          `json:"plazas"`
	Configuracion       Configuracion `json:"configuracion"`
	Acciones            []Accion      `json:"acciones"`
}
type Edicion struct {
	Referencia          string `json:"referencia"`
	AccionReferencia    string `json:"accion_referencia"`
	TituloClave         string `json:"titulo_clave"`
	ModalidadClave      string `json:"modalidad_clave"`
	Desde               string `json:"desde"`
	Hasta               string `json:"hasta"`
	Plazas              *int   `json:"plazas"`
	Horas               *int   `json:"horas"`
	NecesidadClave      string `json:"necesidad_clave"`
	PresupuestoCentimos *int64 `json:"presupuesto_centimos"`
	PrioridadClave      string `json:"prioridad_clave"`
}
type Entrada struct {
	Alcance   string    `json:"alcance"`
	Version   int       `json:"version"`
	Fuente    Fuente    `json:"fuente"`
	Plan      Plan      `json:"plan"`
	Ediciones []Edicion `json:"ediciones"`
}
type EdicionPreparada struct {
	Edicion
	Pendientes []string `json:"pendientes"`
}
type Comprobacion struct {
	Clave      string `json:"clave"`
	Estado     string `json:"estado"`
	Referencia string `json:"referencia"`
}
type Proyeccion struct {
	Alcance           string             `json:"alcance"`
	Version           int                `json:"version"`
	Fuente            Fuente             `json:"fuente"`
	FuenteCorporativa FuenteCorporativa  `json:"fuente_corporativa"`
	Plan              Plan               `json:"plan"`
	Ediciones         []EdicionPreparada `json:"ediciones"`
	Pendientes        []string           `json:"pendientes"`
	Checklist         []Comprobacion     `json:"checklist"`
}

// Leer admite un único documento, acota tamaño y profundidad y rechaza claves repetidas.
func Leer(r io.Reader) (domain.Datos, error) {
	b, err := io.ReadAll(io.LimitReader(r, MaxEntrada+1))
	if err != nil {
		return domain.Datos{}, errors.New("formacion.error.lectura")
	}
	if len(b) > MaxEntrada {
		return domain.Datos{}, errors.New("formacion.error.limites")
	}
	if err := documentoUnico(b); err != nil {
		return domain.Datos{}, errors.New("formacion.error.json")
	}
	var in Entrada
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		return domain.Datos{}, errors.New("formacion.error.json")
	}
	if in.Alcance != domain.Alcance {
		return domain.Datos{}, errors.New("formacion.error.escenario")
	}
	d := domain.Datos{Version: in.Version, Fuente: domain.Fuente{Referencia: in.Fuente.Referencia, Version: in.Fuente.Version, Escenario: in.Fuente.Escenario}, Referencia: in.Plan.Referencia, TituloClave: in.Plan.TituloClave, Desde: in.Plan.Desde, Hasta: in.Plan.Hasta, PresupuestoCentimos: in.Plan.PresupuestoCentimos, Plazas: in.Plan.Plazas, Configuracion: domain.Configuracion{Version: in.Plan.Configuracion.Version, Modalidades: in.Plan.Configuracion.Modalidades, Prioridades: in.Plan.Configuracion.Prioridades}, Acciones: []domain.Accion{}, Ediciones: []domain.Edicion{}}
	for _, a := range in.Plan.Acciones {
		d.Acciones = append(d.Acciones, domain.Accion{Referencia: a.Referencia, TituloClave: a.TituloClave, NecesidadClave: a.NecesidadClave})
	}
	for _, e := range in.Ediciones {
		d.Ediciones = append(d.Ediciones, domain.Edicion{Referencia: e.Referencia, AccionReferencia: e.AccionReferencia, TituloClave: e.TituloClave, ModalidadClave: e.ModalidadClave, Desde: e.Desde, Hasta: e.Hasta, Plazas: e.Plazas, Horas: e.Horas, NecesidadClave: e.NecesidadClave, PresupuestoCentimos: e.PresupuestoCentimos, PrioridadClave: e.PrioridadClave})
	}
	return d, nil
}
func documentoUnico(b []byte) error {
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	if err := valor(d, 0); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return errors.New("extra")
	}
	return nil
}
func valor(d *json.Decoder, nivel int) error {
	if nivel > 12 {
		return errors.New("depth")
	}
	t, err := d.Token()
	if err != nil {
		return err
	}
	delim, ok := t.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := map[string]bool{}
		for d.More() {
			t, err := d.Token()
			if err != nil {
				return err
			}
			k, ok := t.(string)
			if !ok || k != strings.ToLower(k) || seen[k] {
				return errors.New("duplicate")
			}
			seen[k] = true
			if err := valor(d, nivel+1); err != nil {
				return err
			}
		}
	case '[':
		for d.More() {
			if err := valor(d, nivel+1); err != nil {
				return err
			}
		}
	default:
		return errors.New("delimiter")
	}
	_, err = d.Token()
	return err
}
func Proyectar(p application.Preparacion) (Proyeccion, error) {
	var fuentes FuenteCorporativa
	if err := json.Unmarshal(fuentesJSON, &fuentes); err != nil {
		return Proyeccion{}, errors.New("formacion.error.configuracion")
	}
	d := p.Plan
	if d.Configuracion.Modalidades == nil {
		d.Configuracion.Modalidades = []string{}
	}
	if d.Configuracion.Prioridades == nil {
		d.Configuracion.Prioridades = []string{}
	}
	o := Proyeccion{Alcance: domain.Alcance, Version: d.Version, Fuente: Fuente{d.Fuente.Referencia, d.Fuente.Version, d.Fuente.Escenario}, FuenteCorporativa: fuentes, Plan: Plan{Referencia: d.Referencia, TituloClave: d.TituloClave, Desde: d.Desde, Hasta: d.Hasta, PresupuestoCentimos: d.PresupuestoCentimos, Plazas: d.Plazas, Configuracion: Configuracion{d.Configuracion.Version, d.Configuracion.Modalidades, d.Configuracion.Prioridades}, Acciones: []Accion{}}, Ediciones: []EdicionPreparada{}, Pendientes: p.Pendientes, Checklist: []Comprobacion{}}
	for _, a := range d.Acciones {
		o.Plan.Acciones = append(o.Plan.Acciones, Accion{a.Referencia, a.TituloClave, a.NecesidadClave})
	}
	for _, e := range p.Ediciones {
		v := e.Datos
		o.Ediciones = append(o.Ediciones, EdicionPreparada{Edicion: Edicion{Referencia: v.Referencia, AccionReferencia: v.AccionReferencia, TituloClave: v.TituloClave, ModalidadClave: v.ModalidadClave, Desde: v.Desde, Hasta: v.Hasta, Plazas: v.Plazas, Horas: v.Horas, NecesidadClave: v.NecesidadClave, PresupuestoCentimos: v.PresupuestoCentimos, PrioridadClave: v.PrioridadClave}, Pendientes: e.Pendientes})
	}
	for _, c := range p.Checklist {
		o.Checklist = append(o.Checklist, Comprobacion{c.Clave, c.Estado, c.Referencia})
	}
	return o, nil
}
func Escribir(w io.Writer, p application.Preparacion) error {
	o, err := Proyectar(p)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(o)
}
