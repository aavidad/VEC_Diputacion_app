package domain

import (
	"encoding/json"
	"errors"
	"regexp"
	"time"
)

var ErrMarcajeInvalido = errors.New("cronos marcaje invalido")

// AcreditacionCanalMarcaje is supplied by trusted versioned channel policy.
type AcreditacionCanalMarcaje struct{ politicaVersionRef, canalRef, origenRef, calidadRef string }
type DatosAcreditacionCanalMarcaje struct{ PoliticaVersionRef, CanalRef, OrigenRef, CalidadRef string }

func NuevaAcreditacionCanalMarcaje(d DatosAcreditacionCanalMarcaje) (AcreditacionCanalMarcaje, error) {
	a := AcreditacionCanalMarcaje{d.PoliticaVersionRef, d.CanalRef, d.OrigenRef, d.CalidadRef}
	if err := a.Validar(); err != nil {
		return AcreditacionCanalMarcaje{}, err
	}
	return a, nil
}
func (a AcreditacionCanalMarcaje) Validar() error {
	if !referenciaCanal(a.politicaVersionRef) || !referenciaCanal(a.canalRef) || !referenciaCanal(a.origenRef) || !referenciaCanal(a.calidadRef) {
		return ErrMarcajeInvalido
	}
	return nil
}

// Los valores sólo los emite la política de canal acreditada; estos accesores
// permiten ligarlos al material durable sin aceptar cabeceras del cliente.
func (a AcreditacionCanalMarcaje) PoliticaVersionRef() string { return a.politicaVersionRef }
func (a AcreditacionCanalMarcaje) CanalRef() string           { return a.canalRef }
func (a AcreditacionCanalMarcaje) OrigenRef() string          { return a.origenRef }
func (a AcreditacionCanalMarcaje) CalidadRef() string         { return a.calidadRef }

// MaterialAutorizacionMarcajePropio is server-built business material bound to V3 and the Cronos durable function.
type MaterialAutorizacionMarcajePropio struct {
	ActorRef, PerfilRef, EmpleadoRef, ClaveOperacion string
	Movimiento                                       PunchKind
	InstanteUTC                                      time.Time
	Canal                                            AcreditacionCanalMarcaje
}

func (m MaterialAutorizacionMarcajePropio) Validar() error {
	if !referenciaIdentidadMarcaje(m.ActorRef, "per_") || !referenciaIdentidadMarcaje(m.PerfilRef, "prf_") || !referenciaIdentidadMarcaje(m.EmpleadoRef, "emp_") || !claveOperacionMarcaje(m.ClaveOperacion) || m.InstanteUTC.IsZero() || m.InstanteUTC.Location() != time.UTC || m.InstanteUTC.Nanosecond()%1000 != 0 || m.Canal.Validar() != nil {
		return ErrMarcajeInvalido
	}
	switch m.Movimiento {
	case PunchEntry, PunchExit, PunchPauseStart, PunchPauseEnd:
		return nil
	}
	return ErrMarcajeInvalido
}
func (m MaterialAutorizacionMarcajePropio) Canonico() ([]byte, error) {
	if m.Validar() != nil {
		return nil, ErrMarcajeInvalido
	}
	return json.Marshal(map[string]any{"actor_ref": m.ActorRef, "perfil_ref": m.PerfilRef, "empleado_ref": m.EmpleadoRef, "clave_operacion": m.ClaveOperacion, "movimiento": m.Movimiento, "instante_utc": m.InstanteUTC.Format("2006-01-02T15:04:05.000000Z07:00"), "canal": map[string]string{"politica_version_ref": m.Canal.PoliticaVersionRef(), "canal_ref": m.Canal.CanalRef(), "origen_ref": m.Canal.OrigenRef(), "calidad_ref": m.Canal.CalidadRef()}})
}

// MarcajeOriginal es un hecho de solo adicion. Las rectificaciones se representan por un hecho compensatorio ligado al original, nunca lo alteran.
type MarcajeOriginal struct {
	EmpleadoRef     string
	ClaveOperacion  string
	Movimiento      PunchKind
	InstanteUTC     time.Time
	CanalAcreditado AcreditacionCanalMarcaje
}

func (m MarcajeOriginal) Validate() error {
	if !referenciaMarcaje(m.EmpleadoRef) || !claveOperacionMarcaje(m.ClaveOperacion) || m.InstanteUTC.IsZero() || m.InstanteUTC.Location() != time.UTC || m.InstanteUTC.Nanosecond()%1000 != 0 {
		return ErrMarcajeInvalido
	}
	switch m.Movimiento {
	case PunchEntry, PunchExit, PunchPauseStart, PunchPauseEnd:
	default:
		return ErrMarcajeInvalido
	}
	if m.CanalAcreditado.Validar() != nil {
		return ErrMarcajeInvalido
	}
	return nil
}

type MarcajeCompensatorio struct {
	MarcajeOriginalRef string
	ClaveOperacion     string
	Movimiento         PunchKind
	InstanteUTC        time.Time
	MotivoRef          string
}

func (m MarcajeCompensatorio) Validate() error {
	if !referenciaMarcaje(m.MarcajeOriginalRef) || !claveOperacionMarcaje(m.ClaveOperacion) || !referenciaMarcaje(m.MotivoRef) || m.InstanteUTC.IsZero() || m.InstanteUTC.Location() != time.UTC || m.InstanteUTC.Nanosecond()%1000 != 0 {
		return ErrMarcajeInvalido
	}
	switch m.Movimiento {
	case PunchEntry, PunchExit, PunchPauseStart, PunchPauseEnd:
		return nil
	default:
		return ErrMarcajeInvalido
	}
}

var referenciaMarcajePatron = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/#-]{0,159}$`)
var tokenIdentidadMarcajePatron = regexp.MustCompile(`^[A-Za-z0-9_-]{22,128}$`)
var claveOperacionMarcajePatron = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{7,127}$`)

func referenciaMarcaje(valor string) bool { return referenciaMarcajePatron.MatchString(valor) }
func referenciaCanal(valor string) bool   { return len(valor) <= 128 && referenciaMarcaje(valor) }
func referenciaIdentidadMarcaje(valor, prefijo string) bool {
	return len(valor) > len(prefijo) && valor[:len(prefijo)] == prefijo && tokenIdentidadMarcajePatron.MatchString(valor[len(prefijo):])
}
func claveOperacionMarcaje(valor string) bool { return claveOperacionMarcajePatron.MatchString(valor) }
