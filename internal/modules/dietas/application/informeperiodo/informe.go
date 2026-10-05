// Package informeperiodo selecciona comisiones de ejemplo por persona, unidad,
// situación y período con el criterio de una configuración versionada.
//
// Reproduce en el servidor la selección de la vista local de informes de
// Dietas: mismos filtros, mismos conceptos incluidos y los importes
// conservados en cada comisión, sin recalcularlos con tarifas actuales. Solo
// admite paquetes marcados como sintéticos; no consulta expedientes, no
// consume autorizaciones y no acredita una exportación nominal.
package informeperiodo

import (
	"encoding/json"
	"errors"
	"math"
	"regexp"
	"strings"
	"time"

	"golang.org/x/text/currency"
	"golang.org/x/text/language"
)

var (
	// ErrDatosInvalidos agrupa cualquier incoherencia de configuración o registros.
	ErrDatosInvalidos = errors.New("informe_dietas_datos_invalidos")
	// ErrFiltrosInvalidos indica fechas mal formadas o un período invertido.
	ErrFiltrosInvalidos = errors.New("informe_dietas_filtros_invalidos")
)

const (
	maxRegistros = 10000
	maxClaves    = 20
	maxTexto     = 256
)

// CamposFecha son los campos de la comisión que puede elegir la configuración.
var CamposFecha = [...]string{"fecha_inicio", "fecha_liquidacion", "fecha_fiscalizacion"}

var (
	patronClave   = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	patronMoneda  = regexp.MustCompile(`^[A-Z]{3}$`)
	patronIdioma  = regexp.MustCompile(`^[a-z]{2,3}(?:-[a-z0-9]{2,8})*$`)
	patronInstUTC = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$`)
)

// Configuracion es el catálogo de ejemplo `dietas-informes-config-demo`.
type Configuracion struct {
	Schema     string            `json:"schema"`
	Naturaleza string            `json:"naturaleza"`
	Referencia string            `json:"referencia"`
	Version    int               `json:"version"`
	Criterio   Criterio          `json:"criterio"`
	Historia   []EntradaHistoria `json:"historia"`
}

// Criterio fija la fecha del período, las situaciones y los conceptos incluidos.
type Criterio struct {
	CampoFecha         string   `json:"campo_fecha"`
	EstadosIncluidos   []string `json:"estados_incluidos"`
	ConceptosIncluidos []string `json:"conceptos_incluidos"`
}

// EntradaHistoria conserva quién preparó cada versión del ejemplo y por qué.
type EntradaHistoria struct {
	Version      int    `json:"version"`
	ActorRef     string `json:"actor_ref"`
	ActorNombre  string `json:"actor_nombre"`
	Fecha        string `json:"fecha"`
	IdiomaMotivo string `json:"idioma_motivo"`
	Motivo       string `json:"motivo"`
}

// Datos es el paquete de registros `dietas-informes-demo`.
type Datos struct {
	Naturaleza           string     `json:"naturaleza"`
	Schema               string     `json:"schema"`
	Version              int        `json:"version"`
	FechaCorte           string     `json:"fecha_corte"`
	ConfiguracionRef     string     `json:"configuracion_ref"`
	ConfiguracionVersion int        `json:"configuracion_version"`
	Moneda               string     `json:"moneda"`
	Registros            []Registro `json:"registros"`
}

// TextoOpcional distingue un campo ausente del JSON de un campo nulo.
type TextoOpcional struct {
	Presente bool
	Valor    *string
}

// UnmarshalJSON marca el campo como presente también cuando vale null.
func (f *TextoOpcional) UnmarshalJSON(b []byte) error {
	f.Presente = true
	if string(b) == "null" {
		f.Valor = nil
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	f.Valor = &s
	return nil
}

// Registro es una comisión con los importes que conservó su documento.
type Registro struct {
	Referencia         string           `json:"referencia"`
	VersionComision    int              `json:"version_comision"`
	Situacion          string           `json:"situacion"`
	PersonaRef         string           `json:"persona_ref"`
	Persona            string           `json:"persona"`
	UnidadRef          string           `json:"unidad_ref"`
	Unidad             string           `json:"unidad"`
	FechaInicio        string           `json:"fecha_inicio"`
	FechaLiquidacion   TextoOpcional    `json:"fecha_liquidacion"`
	FechaFiscalizacion TextoOpcional    `json:"fecha_fiscalizacion"`
	Moneda             TextoOpcional    `json:"moneda"`
	TotalCentimos      int64            `json:"total_centimos"`
	ConceptosCentimos  map[string]int64 `json:"conceptos_centimos"`
}

// Filtros son los mismos que ofrece la vista; vacío significa «todos».
type Filtros struct {
	Persona   string
	Unidad    string
	Situacion string
	Desde     string
	Hasta     string
}

// Fila es una comisión seleccionada, sin referencias opacas de persona o unidad.
type Fila struct {
	Referencia              string
	VersionComision         int
	Situacion               string
	Persona                 string
	Unidad                  string
	Fecha                   time.Time
	ConceptosCentimos       []int64
	ImporteIncluidoCentimos int64
}

// Informe es la selección resultante. Conceptos fija el orden de las columnas.
// Desde y Hasta repiten el período pedido; nil significa sin límite.
type Informe struct {
	CampoFecha        string
	Desde             *time.Time
	Hasta             *time.Time
	Moneda            string
	Conceptos         []string
	Filas             []Fila
	ConceptosCentimos []int64
	TotalCentimos     int64
}

// Preparar valida configuración y registros y aplica los filtros. Una
// situación que la configuración no incluye produce una selección vacía.
func Preparar(c Configuracion, d Datos, f Filtros) (Informe, error) {
	if err := validarConfiguracion(c, d); err != nil {
		return Informe{}, err
	}
	if err := validarDatos(d, c.Criterio); err != nil {
		return Informe{}, err
	}
	desde, hasta, err := validarFiltros(f)
	if err != nil {
		return Informe{}, err
	}
	conceptos := append([]string(nil), c.Criterio.ConceptosIncluidos...)
	inf := Informe{CampoFecha: c.Criterio.CampoFecha, Desde: desde, Hasta: hasta, Moneda: d.Moneda, Conceptos: conceptos,
		Filas: []Fila{}, ConceptosCentimos: make([]int64, len(conceptos))}
	for _, r := range d.Registros {
		if !contiene(c.Criterio.EstadosIncluidos, r.Situacion) ||
			(f.Situacion != "" && r.Situacion != f.Situacion) ||
			(f.Persona != "" && r.PersonaRef != f.Persona) ||
			(f.Unidad != "" && r.UnidadRef != f.Unidad) {
			continue
		}
		fecha, _ := fechaDia(*fechaDe(r, c.Criterio.CampoFecha))
		if (desde != nil && fecha.Before(*desde)) || (hasta != nil && fecha.After(*hasta)) {
			continue
		}
		fila := Fila{Referencia: r.Referencia, VersionComision: r.VersionComision, Situacion: r.Situacion,
			Persona: r.Persona, Unidad: r.Unidad, Fecha: fecha, ConceptosCentimos: make([]int64, len(conceptos))}
		for i, clave := range conceptos {
			v := r.ConceptosCentimos[clave]
			fila.ConceptosCentimos[i] = v
			if !sumar(&fila.ImporteIncluidoCentimos, v) || !sumar(&inf.ConceptosCentimos[i], v) {
				return Informe{}, ErrDatosInvalidos
			}
		}
		if !sumar(&inf.TotalCentimos, fila.ImporteIncluidoCentimos) {
			return Informe{}, ErrDatosInvalidos
		}
		inf.Filas = append(inf.Filas, fila)
	}
	return inf, nil
}

func validarConfiguracion(c Configuracion, d Datos) error {
	cr := c.Criterio
	if c.Schema != "dietas-informes-config-demo" || c.Naturaleza != "sintetica" || !texto(c.Referencia) ||
		c.Version < 1 || c.Referencia != d.ConfiguracionRef || c.Version != d.ConfiguracionVersion ||
		!contiene(CamposFecha[:], cr.CampoFecha) || !listaClaves(cr.EstadosIncluidos) ||
		!listaClaves(cr.ConceptosIncluidos) || len(c.Historia) != c.Version {
		return ErrDatosInvalidos
	}
	for i, h := range c.Historia {
		if h.Version != i+1 || !strings.HasPrefix(h.ActorRef, "actor:ejemplo:") || !texto(h.ActorRef) ||
			!instanteUTC(h.Fecha) || !texto(h.ActorNombre) || !idioma(h.IdiomaMotivo) ||
			strings.TrimSpace(h.Motivo) == "" || len(h.Motivo) > 2048 {
			return ErrDatosInvalidos
		}
	}
	return nil
}

func validarDatos(d Datos, cr Criterio) error {
	if d.Naturaleza != "sintetica" || d.Schema != "dietas-informes-demo" || d.Version != 1 ||
		!monedaDosDecimales(d.Moneda) || !fechaValida(d.FechaCorte) || d.Registros == nil ||
		len(d.Registros) > maxRegistros {
		return ErrDatosInvalidos
	}
	vistas := make(map[string]struct{}, len(d.Registros))
	for _, r := range d.Registros {
		if !texto(r.Referencia) || !texto(r.PersonaRef) || !texto(r.Persona) || !texto(r.UnidadRef) ||
			!texto(r.Unidad) || !fechaValida(r.FechaInicio) || !opcionalValida(r.FechaLiquidacion) ||
			!opcionalValida(r.FechaFiscalizacion) || r.VersionComision < 1 || !patronClave.MatchString(r.Situacion) ||
			(r.Moneda.Presente && (r.Moneda.Valor == nil || *r.Moneda.Valor != d.Moneda)) {
			return ErrDatosInvalidos
		}
		if contiene(cr.EstadosIncluidos, r.Situacion) {
			if f := fechaDe(r, cr.CampoFecha); f == nil || !fechaValida(*f) {
				return ErrDatosInvalidos
			}
		}
		if _, repetida := vistas[r.Referencia]; repetida {
			return ErrDatosInvalidos
		}
		vistas[r.Referencia] = struct{}{}
		if len(r.ConceptosCentimos) == 0 || len(r.ConceptosCentimos) > maxClaves {
			return ErrDatosInvalidos
		}
		var total int64
		for clave, v := range r.ConceptosCentimos {
			if !patronClave.MatchString(clave) || v < 0 || !sumar(&total, v) {
				return ErrDatosInvalidos
			}
		}
		for _, clave := range cr.ConceptosIncluidos {
			if _, ok := r.ConceptosCentimos[clave]; !ok {
				return ErrDatosInvalidos
			}
		}
		if total != r.TotalCentimos {
			return ErrDatosInvalidos
		}
	}
	return nil
}

func validarFiltros(f Filtros) (*time.Time, *time.Time, error) {
	for _, v := range []string{f.Persona, f.Unidad, f.Situacion} {
		if len(v) > maxTexto {
			return nil, nil, ErrFiltrosInvalidos
		}
	}
	leer := func(s string) (*time.Time, error) {
		if s == "" {
			return nil, nil
		}
		t, ok := fechaDia(s)
		if !ok {
			return nil, ErrFiltrosInvalidos
		}
		return &t, nil
	}
	desde, err := leer(f.Desde)
	if err != nil {
		return nil, nil, err
	}
	hasta, err := leer(f.Hasta)
	if err != nil {
		return nil, nil, err
	}
	if desde != nil && hasta != nil && desde.After(*hasta) {
		return nil, nil, ErrFiltrosInvalidos
	}
	return desde, hasta, nil
}

func fechaDe(r Registro, campo string) *string {
	switch campo {
	case "fecha_inicio":
		return &r.FechaInicio
	case "fecha_liquidacion":
		return r.FechaLiquidacion.Valor
	case "fecha_fiscalizacion":
		return r.FechaFiscalizacion.Valor
	}
	return nil
}

func fechaDia(s string) (time.Time, bool) {
	t, err := time.Parse(time.DateOnly, s)
	return t, err == nil && t.Format(time.DateOnly) == s
}

func fechaValida(s string) bool { _, ok := fechaDia(s); return ok }

func opcionalValida(f TextoOpcional) bool {
	return f.Presente && (f.Valor == nil || fechaValida(*f.Valor))
}

func instanteUTC(s string) bool {
	if !patronInstUTC.MatchString(s) {
		return false
	}
	t, err := time.Parse(time.RFC3339, s)
	return err == nil && t.UTC().Format(time.RFC3339) == s
}

func idioma(s string) bool {
	if !patronIdioma.MatchString(s) {
		return false
	}
	_, err := language.Parse(s)
	return err == nil
}

func monedaDosDecimales(s string) bool {
	if !patronMoneda.MatchString(s) {
		return false
	}
	// Un código desconocido no es moneda válida: el error se traduce en false.
	unidad, err := currency.ParseISO(s)
	escala, _ := currency.Standard.Rounding(unidad)
	return err == nil && escala == 2
}

func listaClaves(v []string) bool {
	if len(v) == 0 || len(v) > maxClaves {
		return false
	}
	vistas := make(map[string]struct{}, len(v))
	for _, s := range v {
		if _, ok := vistas[s]; ok || !patronClave.MatchString(s) {
			return false
		}
		vistas[s] = struct{}{}
	}
	return true
}

func texto(s string) bool { return strings.TrimSpace(s) != "" && len(s) <= maxTexto }

func contiene(lista []string, v string) bool {
	for _, s := range lista {
		if s == v {
			return true
		}
	}
	return false
}

func sumar(acumulado *int64, v int64) bool {
	if v > math.MaxInt64-*acumulado {
		return false
	}
	*acumulado += v
	return true
}
