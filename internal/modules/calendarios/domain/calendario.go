package domain

import (
	"errors"
	"sort"
	"strconv"
	"time"
	"unicode/utf8"
)

var (
	ErrAmbitoInvalido       = errors.New("calendarios: ambito invalido")
	ErrVersionInvalida      = errors.New("calendarios: version de calendario invalida")
	ErrCalendarioNoCubre    = errors.New("calendarios: no hay calendario publicado para el ambito y año")
	ErrCalculoNoDeterminado = errors.New("calendarios: calculo no determinado")
)

// TipoAmbito identifica quién publica o gobierna un calendario.
type TipoAmbito string

const (
	AmbitoNacional   TipoAmbito = "nacional"
	AmbitoAutonomico TipoAmbito = "autonomico"
	AmbitoLocal      TipoAmbito = "local"
	// AmbitoCentro es el calendario laboral propio de un centro de trabajo.
	AmbitoCentro TipoAmbito = "centro"
)

// ReferenciaNacional es el único ámbito nacional admitido.
const ReferenciaNacional = "es"

type Ambito struct {
	Tipo TipoAmbito `json:"tipo"`
	Ref  string     `json:"ref"`
}

func (a Ambito) Validar() error {
	switch a.Tipo {
	case AmbitoNacional:
		if a.Ref != ReferenciaNacional {
			return ErrAmbitoInvalido
		}
		return nil
	case AmbitoAutonomico, AmbitoLocal, AmbitoCentro:
		if !ReferenciaValida(a.Ref) {
			return ErrAmbitoInvalido
		}
		return nil
	default:
		return ErrAmbitoInvalido
	}
}

func (a Ambito) Clave() string { return string(a.Tipo) + "|" + a.Ref }

// ReferenciaValida limita las referencias opacas a minúsculas, cifras y los
// separadores «:», «_» y «-», con primer carácter alfabético.
func ReferenciaValida(v string) bool {
	if len(v) < 2 || len(v) > 96 || v[0] < 'a' || v[0] > 'z' {
		return false
	}
	for _, c := range v {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == ':' || c == '_' || c == '-') {
			return false
		}
	}
	return true
}

// Efecto es independiente y tipado: un festivo oficial no se confunde con un
// cierre interno que no altera el cómputo administrativo.
type Efecto string

const (
	// EfectoFestivo es una fiesta laboral oficial: inhábil administrativo y
	// no laborable.
	EfectoFestivo Efecto = "festivo"
	// EfectoInhabilAdministrativo solo excluye la fecha del cómputo de plazos.
	EfectoInhabilAdministrativo Efecto = "inhabil_administrativo"
	// EfectoNoLaborable es un día no laborable del calendario del centro que
	// no convierte la fecha en inhábil administrativa.
	EfectoNoLaborable Efecto = "no_laborable"
)

func (e Efecto) Valido() bool {
	return e == EfectoFestivo || e == EfectoInhabilAdministrativo || e == EfectoNoLaborable
}

func (e Efecto) Inhabilita() bool  { return e == EfectoFestivo || e == EfectoInhabilAdministrativo }
func (e Efecto) NoLaborable() bool { return e == EfectoFestivo || e == EfectoNoLaborable }

// Procedencia identifica la norma, acuerdo o dato sintético que publica una
// versión. Una procedencia sintética nunca se presenta como oficial.
type Procedencia struct {
	Norma       string     `json:"norma"`
	Referencia  string     `json:"referencia"`
	PublicadaEn FechaCivil `json:"publicada_en"`
	Sintetica   bool       `json:"sintetica"`
}

func (p Procedencia) Validar() error {
	if !textoValido(p.Norma, 400) || !textoValido(p.Referencia, 400) || !p.PublicadaEn.EsValida() {
		return ErrVersionInvalida
	}
	return nil
}

// VersionCalendario es inmutable. Una corrección se publica como número
// siguiente y conserva la versión sustituida; lo vigente en un instante es la
// última versión conocida hasta entonces.
type VersionCalendario struct {
	ID            string      `json:"id"`
	Ambito        Ambito      `json:"ambito"`
	Anio          int         `json:"anio"`
	Numero        int         `json:"numero"`
	SustituyeID   string      `json:"sustituye_id,omitempty"`
	Denominacion  string      `json:"denominacion"`
	Procedencia   Procedencia `json:"procedencia"`
	ConocidoDesde time.Time   `json:"conocido_desde"`
	// Solo para calendarios locales y de centro: comunidad y municipio a los
	// que se aplica el ámbito territorial.
	ComunidadRef string `json:"comunidad_ref,omitempty"`
	MunicipioRef string `json:"municipio_ref,omitempty"`
}

func (v VersionCalendario) Validar() error {
	if !ReferenciaValida(v.ID) || v.Ambito.Validar() != nil || v.Anio < anioMinimo || v.Anio > anioMaximo || v.Numero < 1 ||
		(v.Numero == 1) != (v.SustituyeID == "") || (v.SustituyeID != "" && !ReferenciaValida(v.SustituyeID)) ||
		!textoValido(v.Denominacion, 240) || v.Procedencia.Validar() != nil || v.ConocidoDesde.IsZero() {
		return ErrVersionInvalida
	}
	territorial := v.Ambito.Tipo == AmbitoLocal || v.Ambito.Tipo == AmbitoCentro
	if territorial != (v.ComunidadRef != "") || (v.Ambito.Tipo == AmbitoCentro) != (v.MunicipioRef != "") ||
		(v.ComunidadRef != "" && !ReferenciaValida(v.ComunidadRef)) || (v.MunicipioRef != "" && !ReferenciaValida(v.MunicipioRef)) {
		return ErrVersionInvalida
	}
	return nil
}

type DiaSenalado struct {
	Fecha        FechaCivil `json:"fecha"`
	Efecto       Efecto     `json:"efecto"`
	Denominacion string     `json:"denominacion"`
}

// VersionConDias agrupa una versión con los días que declara.
type VersionConDias struct {
	Version VersionCalendario `json:"version"`
	Dias    []DiaSenalado     `json:"dias"`
}

func (v VersionConDias) Validar() error {
	if err := v.Version.Validar(); err != nil {
		return err
	}
	vistos := map[string]struct{}{}
	for _, d := range v.Dias {
		if !d.Fecha.EsValida() || d.Fecha.Anio() != v.Version.Anio || !d.Efecto.Valido() || !textoValido(d.Denominacion, 240) {
			return ErrVersionInvalida
		}
		// Un calendario oficial no declara cierres internos ni un centro
		// declara fiestas oficiales: cada autoridad publica lo suyo.
		if (v.Version.Ambito.Tipo == AmbitoCentro) != (d.Efecto == EfectoNoLaborable) {
			return ErrVersionInvalida
		}
		clave := d.Fecha.String() + "|" + string(d.Efecto)
		if _, repetido := vistos[clave]; repetido {
			return ErrVersionInvalida
		}
		vistos[clave] = struct{}{}
	}
	return nil
}

// Motivo explica por qué una fecha está señalada y qué versión lo publica.
type Motivo struct {
	Ambito       Ambito `json:"ambito"`
	Efecto       Efecto `json:"efecto"`
	Denominacion string `json:"denominacion"`
	VersionID    string `json:"version_id"`
	Sintetico    bool   `json:"sintetico"`
}

// Calendario combina versiones de uno o varios años. Un año no cargado no se
// supone hábil de lunes a viernes: la consulta falla de forma cerrada.
type Calendario struct {
	anios   map[int]struct{}
	motivos map[FechaCivil][]Motivo
}

// NuevoCalendario exige una versión por ámbito y año. El llamador decide qué
// ámbitos son obligatorios; aquí solo se evita mezclar dos versiones del mismo.
func NuevoCalendario(anios []int, versiones []VersionConDias) (*Calendario, error) {
	c := &Calendario{anios: map[int]struct{}{}, motivos: map[FechaCivil][]Motivo{}}
	for _, a := range anios {
		if a < anioMinimo || a > anioMaximo {
			return nil, ErrVersionInvalida
		}
		c.anios[a] = struct{}{}
	}
	vistas := map[string]struct{}{}
	for _, v := range versiones {
		if err := v.Validar(); err != nil {
			return nil, err
		}
		if _, cubierto := c.anios[v.Version.Anio]; !cubierto {
			return nil, ErrVersionInvalida
		}
		clave := v.Version.Ambito.Clave() + "|" + strconv.Itoa(v.Version.Anio)
		if _, repetida := vistas[clave]; repetida {
			return nil, ErrVersionInvalida
		}
		vistas[clave] = struct{}{}
		for _, d := range v.Dias {
			c.motivos[d.Fecha] = append(c.motivos[d.Fecha], Motivo{
				Ambito: v.Version.Ambito, Efecto: d.Efecto, Denominacion: d.Denominacion,
				VersionID: v.Version.ID, Sintetico: v.Version.Procedencia.Sintetica,
			})
		}
	}
	for f := range c.motivos {
		ordenarMotivos(c.motivos[f])
	}
	return c, nil
}

func (c *Calendario) Cubre(anio int) bool {
	if c == nil {
		return false
	}
	_, ok := c.anios[anio]
	return ok
}

// Clasificacion separa las preguntas que un único «es laborable» mezclaría.
type Clasificacion struct {
	Fecha                 FechaCivil `json:"fecha"`
	FinDeSemana           bool       `json:"fin_de_semana"`
	FestivoOficial        bool       `json:"festivo_oficial"`
	InhabilAdministrativo bool       `json:"inhabil_administrativo"`
	Laborable             bool       `json:"laborable"`
	Motivos               []Motivo   `json:"motivos"`
}

// Clasificar aplica la Ley 39/2015 (art. 30.2): sábados, domingos y festivos
// son inhábiles. Laborable describe el calendario del centro, no el cómputo.
func (c *Calendario) Clasificar(f FechaCivil) (Clasificacion, error) {
	if !f.EsValida() {
		return Clasificacion{}, ErrFechaInvalida
	}
	if !c.Cubre(f.Anio()) {
		return Clasificacion{}, ErrCalendarioNoCubre
	}
	r := Clasificacion{Fecha: f, FinDeSemana: f.EsFinDeSemana(), Motivos: append([]Motivo(nil), c.motivos[f]...)}
	r.InhabilAdministrativo = r.FinDeSemana
	noLaborable := r.FinDeSemana
	for _, m := range r.Motivos {
		r.FestivoOficial = r.FestivoOficial || m.Efecto == EfectoFestivo
		r.InhabilAdministrativo = r.InhabilAdministrativo || m.Efecto.Inhabilita()
		noLaborable = noLaborable || m.Efecto.NoLaborable()
	}
	r.Laborable = !noLaborable
	return r, nil
}

func (c *Calendario) EsInhabil(f FechaCivil) (bool, []Motivo, error) {
	r, err := c.Clasificar(f)
	if err != nil {
		return false, nil, err
	}
	return r.InhabilAdministrativo, r.Motivos, nil
}

func ordenarMotivos(m []Motivo) {
	orden := map[TipoAmbito]int{AmbitoNacional: 0, AmbitoAutonomico: 1, AmbitoLocal: 2, AmbitoCentro: 3}
	sort.SliceStable(m, func(i, j int) bool {
		if orden[m[i].Ambito.Tipo] != orden[m[j].Ambito.Tipo] {
			return orden[m[i].Ambito.Tipo] < orden[m[j].Ambito.Tipo]
		}
		if m[i].Ambito.Ref != m[j].Ambito.Ref {
			return m[i].Ambito.Ref < m[j].Ambito.Ref
		}
		return m[i].Efecto < m[j].Efecto
	})
}

func textoValido(v string, maximo int) bool {
	if v == "" || !utf8.ValidString(v) || utf8.RuneCountInString(v) > maximo || v[0] == ' ' || v[len(v)-1] == ' ' {
		return false
	}
	for _, c := range v {
		if c < 32 || c == 127 {
			return false
		}
	}
	return true
}

// ErrorCobertura enumera los ámbitos sin calendario publicado para un año.
// Equivale a ErrCalendarioNoCubre y nunca se resuelve suponiendo días hábiles.
type ErrorCobertura struct {
	Anio   int
	Faltan []Ambito
}

func (e *ErrorCobertura) Error() string { return ErrCalendarioNoCubre.Error() }
func (e *ErrorCobertura) Unwrap() error { return ErrCalendarioNoCubre }
