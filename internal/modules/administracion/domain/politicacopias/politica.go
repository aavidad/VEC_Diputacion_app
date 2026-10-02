// Package politicacopias defines versioned backup schedules and retention plans.
// It never grants authority, verifies backups or removes backup content.
package politicacopias

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"sort"
	"time"
)

var (
	ErrEntrada     = errors.New("politica_entrada_invalida")
	ErrVersion     = errors.New("politica_version_distinta")
	ErrHistoria    = errors.New("politica_historia_invalida")
	ErrDenegado    = errors.New("politica_denegada")
	ErrDependencia = errors.New("politica_dependencia_pendiente")
	ErrVentana     = errors.New("politica_fuera_ventana")
	ErrAviso       = errors.New("politica_aviso_fallido")
	ref            = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9:_-]{0,95}$`)
)

func Referencia(s string) bool { return ref.MatchString(s) }

// Politica is external configuration. No frequency or conservation period is
// inferred from law. A nil DobleControl means true, including during replay.
type Politica struct {
	Formato      uint64    `json:"formato"`
	Referencia   string    `json:"referencia"`
	Destino      string    `json:"destino"`
	ZonaHoraria  string    `json:"zona_horaria"`
	FechaInicial string    `json:"fecha_inicial"`
	CadaDias     int       `json:"cada_dias"`
	DiasSemana   []int     `json:"dias_semana,omitempty"`
	Ventana      Ventana   `json:"ventana"`
	Retencion    Retencion `json:"retencion"`
}
type Ventana struct {
	Inicio string `json:"inicio"`
	Fin    string `json:"fin"`
}
type Retencion struct {
	ConservarMinimo  int      `json:"conservar_minimo"`
	EdadMaximaDias   int      `json:"edad_maxima_dias"`
	Protegidas       []string `json:"protegidas,omitempty"`
	BorradoPermitido bool     `json:"borrado_permitido"`
	DobleControl     *bool    `json:"doble_control,omitempty"`
}

func (r Retencion) ExigeDobleControl() bool { return r.DobleControl == nil || *r.DobleControl }

func (p Politica) Validar() error {
	if p.Formato != 1 || !Referencia(p.Referencia) || !Referencia(p.Destino) || len(p.ZonaHoraria) > 96 || p.ZonaHoraria == "" || p.ZonaHoraria == "Local" || p.CadaDias < 1 || p.CadaDias > 366 || p.Retencion.ConservarMinimo < 1 || p.Retencion.ConservarMinimo > 10000 || p.Retencion.EdadMaximaDias < 1 || p.Retencion.EdadMaximaDias > 36500 || len(p.Retencion.Protegidas) > 1024 || len(p.DiasSemana) > 7 {
		return ErrEntrada
	}
	if _, e := time.LoadLocation(p.ZonaHoraria); e != nil {
		return ErrEntrada
	}
	if _, e := time.Parse("2006-01-02", p.FechaInicial); e != nil {
		return ErrEntrada
	}
	inicio, e := minuto(p.Ventana.Inicio)
	if e != nil {
		return e
	}
	fin, e := minuto(p.Ventana.Fin)
	if e != nil || fin <= inicio {
		return ErrEntrada
	}
	seen := map[int]bool{}
	for _, d := range p.DiasSemana {
		if d < 0 || d > 6 || seen[d] {
			return ErrEntrada
		}
		seen[d] = true
	}
	refs := map[string]bool{}
	for _, r := range p.Retencion.Protegidas {
		if !Referencia(r) || refs[r] {
			return ErrEntrada
		}
		refs[r] = true
	}
	return nil
}
func minuto(s string) (int, error) {
	t, e := time.Parse("15:04", s)
	if e != nil || t.Format("15:04") != s {
		return 0, ErrEntrada
	}
	return t.Hour()*60 + t.Minute(), nil
}

// Normalizar returns a private canonical copy; sorting never mutates the input.
func (p Politica) Normalizar() Politica {
	p.DiasSemana = append([]int(nil), p.DiasSemana...)
	sort.Ints(p.DiasSemana)
	p.Retencion.Protegidas = append([]string(nil), p.Retencion.Protegidas...)
	sort.Strings(p.Retencion.Protegidas)
	b := p.Retencion.ExigeDobleControl()
	p.Retencion.DobleControl = &b
	return p
}
func (p Politica) SHA256() string {
	b, _ := json.Marshal(p.Normalizar())
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

type EventoAgenda struct {
	Fecha      time.Time `json:"fecha"`
	FinVentana time.Time `json:"fin_ventana"`
	FechaCivil string    `json:"fecha_civil"`
}

// Proximos chooses one occurrence per civil date. A missing DST start skips that
// date; a repeated start chooses its first UTC occurrence. Windows never cross
// midnight, avoiding an implicit date/frequency convention.
func (p Politica) Proximos(desde time.Time, cantidad int) ([]EventoAgenda, error) {
	if p.Validar() != nil || desde.IsZero() || desde.Year() < 1 || desde.Year() > 9990 || cantidad < 1 || cantidad > 64 {
		return nil, ErrEntrada
	}
	loc, _ := time.LoadLocation(p.ZonaHoraria)
	t := desde.In(loc)
	dia := time.Date(t.Year(), t.Month(), t.Day(), 12, 0, 0, 0, loc)
	anchor, _ := time.Parse("2006-01-02", p.FechaInicial)
	inicio, _ := minuto(p.Ventana.Inicio)
	fin, _ := minuto(p.Ventana.Fin)
	events := make([]EventoAgenda, 0, cantidad)
	for n := 0; n < 36600 && len(events) < cantidad; n++ {
		civil := time.Date(dia.Year(), dia.Month(), dia.Day(), 0, 0, 0, 0, time.UTC)
		days := int((civil.Unix() - anchor.Unix()) / 86400)
		if !civil.Before(anchor) && days%p.CadaDias == 0 && p.admiteDia(dia.Weekday()) {
			start, ok := horaCivil(dia, inicio, loc, false)
			end, okEnd := horaCivil(dia, fin, loc, true)
			if ok && okEnd && end.After(start) && !start.Before(desde) {
				events = append(events, EventoAgenda{start, end, civil.Format("2006-01-02")})
			}
		}
		dia = dia.AddDate(0, 0, 1)
	}
	if len(events) != cantidad {
		return nil, ErrEntrada
	}
	return events, nil
}
func (p Politica) admiteDia(d time.Weekday) bool {
	if len(p.DiasSemana) == 0 {
		return true
	}
	for _, x := range p.DiasSemana {
		if x == int(d) {
			return true
		}
	}
	return false
}

// Search around the civil noon in UTC, without trusting time.Date's choice for
// nonexistent or ambiguous wall-clock times. Last end includes both fold hours.
func horaCivil(d time.Time, minute int, loc *time.Location, last bool) (time.Time, bool) {
	candidate := time.Date(d.Year(), d.Month(), d.Day(), minute/60, minute%60, 0, 0, loc).UTC()
	var found time.Time
	for offset := -180; offset <= 180; offset++ {
		u := candidate.Add(time.Duration(offset) * time.Minute)
		t := u.In(loc)
		if t.Year() == d.Year() && t.Month() == d.Month() && t.Day() == d.Day() && t.Hour()*60+t.Minute() == minute {
			if found.IsZero() || last {
				found = u
			}
		}
	}
	return found, !found.IsZero()
}
