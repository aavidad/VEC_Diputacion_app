package domain

import (
	"encoding/json"
	"errors"
	"regexp"
	"strconv"
	"time"
	"unicode"
	"unicode/utf8"
)

var ErrNotificacionInvalida = errors.New("cronos notificacion invalida")

var referenciaNotificacion = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_:-]{7,127}$`)
var fechaCivilNotificacion = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}$`)

// Notificacion es la declaración de una persona autorizada. No acredita
// despacho, entrega ni resolución de la incidencia comunicada.
type Notificacion struct {
	EmpleadoRef    string
	TipoRef        string
	TipoVersion    int64
	FechaReferida  string // Fecha civil YYYY-MM-DD; nunca se convierte a UTC.
	Texto          string
	AdjuntoRef     string // Referencia opaca de custodia externa, sin bytes.
	ClaveOperacion string
	InstanteUTC    time.Time
}

func (n Notificacion) Validar() error {
	if !referenciaMarcaje(n.EmpleadoRef) || !referenciaNotificacion.MatchString(n.TipoRef) || n.TipoVersion < 1 ||
		!fechaCivilNotificacion.MatchString(n.FechaReferida) || !referenciaNotificacion.MatchString(n.AdjuntoRef) ||
		!claveOperacionMarcaje(n.ClaveOperacion) || !utf8.ValidString(n.Texto) ||
		utf8.RuneCountInString(n.Texto) == 0 || utf8.RuneCountInString(n.Texto) > 512 ||
		n.InstanteUTC.IsZero() || n.InstanteUTC.Location() != time.UTC || n.InstanteUTC.Nanosecond()%1000 != 0 {
		return ErrNotificacionInvalida
	}
	fecha, err := time.Parse("2006-01-02", n.FechaReferida)
	if err != nil || fecha.Format("2006-01-02") != n.FechaReferida {
		return ErrNotificacionInvalida
	}
	for _, r := range n.Texto {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return ErrNotificacionInvalida
		}
	}
	return nil
}

// MaterialAutorizacionNotificacion liga la concesión nominal al contenido exacto.
// Canonico puede contener texto personal: nunca debe registrarse en logs.
type MaterialAutorizacionNotificacion struct {
	ActorRef, PerfilRef string
	Notificacion        Notificacion
}

func (m MaterialAutorizacionNotificacion) Validar() error {
	if !referenciaIdentidadMarcaje(m.ActorRef, "per_") ||
		!referenciaIdentidadMarcaje(m.PerfilRef, "prf_") || m.Notificacion.Validar() != nil {
		return ErrNotificacionInvalida
	}
	return nil
}

func (m MaterialAutorizacionNotificacion) Canonico() ([]byte, error) {
	if m.Validar() != nil {
		return nil, ErrNotificacionInvalida
	}
	// El instante no forma parte de la identidad semántica: un reintento con la
	// misma clave y contenido debe recuperar el recibo, aunque ocurra después.
	return json.Marshal(map[string]string{
		"version": "cronos.notificacion.v1", "actor_ref": m.ActorRef,
		"perfil_ref": m.PerfilRef, "empleado_ref": m.Notificacion.EmpleadoRef,
		"tipo_ref": m.Notificacion.TipoRef, "tipo_version": strconv.FormatInt(m.Notificacion.TipoVersion, 10),
		"fecha_referida": m.Notificacion.FechaReferida,
		"texto":          m.Notificacion.Texto, "adjunto_ref": m.Notificacion.AdjuntoRef,
		"clave_operacion": m.Notificacion.ClaveOperacion,
	})
}
