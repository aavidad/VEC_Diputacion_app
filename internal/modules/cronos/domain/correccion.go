package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"
)

var ErrCorreccionInvalida = errors.New("cronos correccion de marcaje invalida")

type EstadoCorreccion string

const (
	CorreccionPendienteResponsable EstadoCorreccion = "pendiente_responsable"
	CorreccionPendienteRRHH        EstadoCorreccion = "pendiente_rrhh"
	CorreccionDenegadaResponsable  EstadoCorreccion = "denegada_responsable"
	CorreccionDenegadaRRHH         EstadoCorreccion = "denegada_rrhh"
	CorreccionPendienteAplicacion  EstadoCorreccion = "pendiente_aplicacion"
	CorreccionAplicada             EstadoCorreccion = "aplicada"
)

type PasoCorreccion string

const (
	PasoSolicitudCorreccion PasoCorreccion = "solicitud"
	PasoDecisionResponsable PasoCorreccion = "decision_responsable"
	PasoResolucionRRHH      PasoCorreccion = "resolucion_rrhh"
	PasoAplicacion          PasoCorreccion = "aplicacion"
)

type ResultadoCorreccion string

const (
	ResultadoFavorable    ResultadoCorreccion = "favorable"
	ResultadoDesfavorable ResultadoCorreccion = "desfavorable"
)

const MotivoOlvidoMarcaje = "olvido_marcaje"

// SolicitudCorreccion es un hecho declarado, no altera el marcaje original ni
// acredita tiempo trabajado. FechaCivil y HoraPretendida no se convierten a UTC.
type SolicitudCorreccion struct {
	EmpleadoRef, ActorRef, PerfilRef string
	ClaveOperacion                   string
	MarcajeOriginalRef               string
	HuecoDeclarado                   bool
	Movimiento                       PunchKind
	FechaCivil                       string // YYYY-MM-DD
	HoraPretendida                   string // HH:MM
	MotivoCodigo                     string
	SolicitadaEnUTC                  time.Time
}

func (s SolicitudCorreccion) Validar() error {
	if !referenciaIdentidadMarcaje(s.EmpleadoRef, "emp_") ||
		!referenciaIdentidadMarcaje(s.ActorRef, "per_") ||
		!referenciaIdentidadMarcaje(s.PerfilRef, "prf_") ||
		!claveOperacionMarcaje(s.ClaveOperacion) ||
		!instanteCorreccionValido(s.SolicitadaEnUTC) ||
		s.MotivoCodigo != MotivoOlvidoMarcaje ||
		!fechaCivilCorreccionValida(s.FechaCivil) ||
		!horaCorreccionValida(s.HoraPretendida) ||
		(s.MarcajeOriginalRef == "") != s.HuecoDeclarado {
		return ErrCorreccionInvalida
	}
	if s.MarcajeOriginalRef != "" && !referenciaOriginalCorreccionValida(s.MarcajeOriginalRef) {
		return ErrCorreccionInvalida
	}
	if !movimientoCorreccionValido(s.Movimiento) {
		return ErrCorreccionInvalida
	}
	return nil
}

// HuellaSemantica omite el instante de recepción: un replay autorizado de la
// misma declaración conserva identidad, contenido y clave aunque llegue luego.
func (s SolicitudCorreccion) HuellaSemantica() (string, error) {
	if s.Validar() != nil {
		return "", ErrCorreccionInvalida
	}
	b, err := json.Marshal(struct {
		EmpleadoRef, ActorRef, PerfilRef, ClaveOperacion string
		MarcajeOriginalRef                               string
		HuecoDeclarado                                   bool
		Movimiento                                       PunchKind
		FechaCivil, HoraPretendida, MotivoCodigo         string
	}{s.EmpleadoRef, s.ActorRef, s.PerfilRef, s.ClaveOperacion, s.MarcajeOriginalRef, s.HuecoDeclarado, s.Movimiento, s.FechaCivil, s.HoraPretendida, s.MotivoCodigo})
	if err != nil {
		return "", ErrCorreccionInvalida
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}

// ActuacionCorreccion representa cada paso posterior como hecho nuevo. El
// repositorio resuelve el empleado del expediente bajo bloqueo y verifica
// versión, competencia, separación de funciones y autorización exacta.
type ActuacionCorreccion struct {
	SolicitudRef, ActorRef, PerfilRef, ClaveOperacion string
	Paso                                              PasoCorreccion
	Resultado                                         ResultadoCorreccion
	VersionEsperada                                   uint64
	RegistradaEnUTC                                   time.Time
}

func (a ActuacionCorreccion) Validar() error {
	if !referenciaSolicitudCorreccionValida(a.SolicitudRef) ||
		!referenciaIdentidadMarcaje(a.ActorRef, "per_") ||
		!referenciaIdentidadMarcaje(a.PerfilRef, "prf_") ||
		!claveOperacionMarcaje(a.ClaveOperacion) ||
		!instanteCorreccionValido(a.RegistradaEnUTC) {
		return ErrCorreccionInvalida
	}
	switch a.Paso {
	case PasoDecisionResponsable:
		if a.VersionEsperada != 1 || !resultadoCorreccionValido(a.Resultado) {
			return ErrCorreccionInvalida
		}
	case PasoResolucionRRHH:
		if a.VersionEsperada != 2 || !resultadoCorreccionValido(a.Resultado) {
			return ErrCorreccionInvalida
		}
	case PasoAplicacion:
		if a.VersionEsperada != 3 || a.Resultado != "" {
			return ErrCorreccionInvalida
		}
	default:
		return ErrCorreccionInvalida
	}
	return nil
}

func (a ActuacionCorreccion) HuellaSemantica() (string, error) {
	if a.Validar() != nil {
		return "", ErrCorreccionInvalida
	}
	b, err := json.Marshal(struct {
		SolicitudRef, ActorRef, PerfilRef, ClaveOperacion string
		Paso                                              PasoCorreccion
		Resultado                                         ResultadoCorreccion
		VersionEsperada                                   uint64
	}{a.SolicitudRef, a.ActorRef, a.PerfilRef, a.ClaveOperacion, a.Paso, a.Resultado, a.VersionEsperada})
	if err != nil {
		return "", ErrCorreccionInvalida
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}

// SiguienteEstadoCorreccion no permite aplicar una solicitud sin las dos
// decisiones favorables. La proyección de jornada se actualiza sólo al aplicar.
func SiguienteEstadoCorreccion(actual EstadoCorreccion, paso PasoCorreccion, resultado ResultadoCorreccion) (EstadoCorreccion, error) {
	switch {
	case actual == CorreccionPendienteResponsable && paso == PasoDecisionResponsable && resultado == ResultadoFavorable:
		return CorreccionPendienteRRHH, nil
	case actual == CorreccionPendienteResponsable && paso == PasoDecisionResponsable && resultado == ResultadoDesfavorable:
		return CorreccionDenegadaResponsable, nil
	case actual == CorreccionPendienteRRHH && paso == PasoResolucionRRHH && resultado == ResultadoFavorable:
		return CorreccionPendienteAplicacion, nil
	case actual == CorreccionPendienteRRHH && paso == PasoResolucionRRHH && resultado == ResultadoDesfavorable:
		return CorreccionDenegadaRRHH, nil
	case actual == CorreccionPendienteAplicacion && paso == PasoAplicacion && resultado == "":
		return CorreccionAplicada, nil
	default:
		return "", ErrCorreccionInvalida
	}
}

// MaterialAutorizacionCorreccion liga una concesión V3 al paso, empleado y
// huella del comando exacto. El repositorio deriva EmpleadoRef del expediente
// para decisiones y revalida todo dentro de la transacción.
type MaterialAutorizacionCorreccion struct {
	ActorRef, PerfilRef, EmpleadoRef string
	SolicitudRef, ClaveOperacion     string
	Paso                             PasoCorreccion
	ComandoSHA256                    string
	InstanteUTC                      time.Time
}

func (m MaterialAutorizacionCorreccion) Validar() error {
	if !referenciaIdentidadMarcaje(m.ActorRef, "per_") || !referenciaIdentidadMarcaje(m.PerfilRef, "prf_") ||
		!referenciaIdentidadMarcaje(m.EmpleadoRef, "emp_") ||
		!referenciaSolicitudCorreccionValida(m.SolicitudRef) ||
		!claveOperacionMarcaje(m.ClaveOperacion) ||
		!huellaCorreccionPatron.MatchString(m.ComandoSHA256) ||
		!instanteCorreccionValido(m.InstanteUTC) {
		return ErrCorreccionInvalida
	}
	switch m.Paso {
	case PasoSolicitudCorreccion, PasoDecisionResponsable, PasoResolucionRRHH, PasoAplicacion:
		return nil
	default:
		return ErrCorreccionInvalida
	}
}

var (
	horaCorreccionPatron   = regexp.MustCompile(`^[0-2][0-9]:[0-5][0-9]$`)
	huellaCorreccionPatron = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

func fechaCivilCorreccionValida(fecha string) bool {
	if len(fecha) != 10 {
		return false
	}
	parsed, err := time.Parse("2006-01-02", fecha)
	return err == nil && parsed.Format("2006-01-02") == fecha
}

func horaCorreccionValida(hora string) bool {
	return horaCorreccionPatron.MatchString(hora) && hora[:2] <= "23"
}

func instanteCorreccionValido(i time.Time) bool {
	return !i.IsZero() && i.Location() == time.UTC && i.Nanosecond()%1000 == 0
}

func referenciaOriginalCorreccionValida(ref string) bool {
	return strings.HasPrefix(ref, "marcaje:cronos:") && claveOperacionMarcaje(strings.TrimPrefix(ref, "marcaje:cronos:"))
}

func referenciaSolicitudCorreccionValida(ref string) bool {
	return strings.HasPrefix(ref, "correccion:cronos:") && claveOperacionMarcaje(strings.TrimPrefix(ref, "correccion:cronos:"))
}

func movimientoCorreccionValido(m PunchKind) bool {
	switch m {
	case PunchEntry, PunchExit, PunchPauseStart, PunchPauseEnd:
		return true
	default:
		return false
	}
}

func resultadoCorreccionValido(r ResultadoCorreccion) bool {
	return r == ResultadoFavorable || r == ResultadoDesfavorable
}

type ClaveRecuperacionCorreccion struct {
	SolicitudRef, ClaveOperacion string
	Paso                         PasoCorreccion
}

func (c ClaveRecuperacionCorreccion) Validar() error {
	if !referenciaSolicitudCorreccionValida(c.SolicitudRef) || !claveOperacionMarcaje(c.ClaveOperacion) {
		return ErrCorreccionInvalida
	}
	switch c.Paso {
	case PasoSolicitudCorreccion, PasoDecisionResponsable, PasoResolucionRRHH, PasoAplicacion:
		return nil
	default:
		return ErrCorreccionInvalida
	}
}
