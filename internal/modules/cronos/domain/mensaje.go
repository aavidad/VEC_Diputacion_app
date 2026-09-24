package domain

import (
	"errors"
	"time"
	"unicode/utf8"
)

var ErrMensajeInvalido = errors.New("cronos mensaje invalido")

func ReferenciaMensajeValida(valor string) bool { return referenciaMarcaje(valor) }
func ClaveMensajeValida(valor string) bool      { return claveOperacionMarcaje(valor) }

type EstadoMensaje string

const (
	MensajePendiente EstadoMensaje = "pendiente"
	MensajeArchivado EstadoMensaje = "archivado"
)

// MensajeResolucion es un aviso sobre una resolución ya constatada por su
// autoridad. El texto visible se obtiene de esa fuente, nunca del cliente.
type MensajeResolucion struct {
	Referencia        string
	EmpleadoRef       string
	ResolucionRef     string
	ResolucionVersion int64
	Texto             string // Generado por la resolución constatada, nunca del cliente.
	Estado            EstadoMensaje
	Version           int64
	CreadoUTC         time.Time
	ArchivadoUTC      time.Time
}

func (m MensajeResolucion) Validar() error {
	if !referenciaMarcaje(m.Referencia) || !referenciaMarcaje(m.EmpleadoRef) ||
		!referenciaMarcaje(m.ResolucionRef) || m.ResolucionVersion < 1 || m.Version < 1 ||
		!utf8.ValidString(m.Texto) || utf8.RuneCountInString(m.Texto) == 0 || utf8.RuneCountInString(m.Texto) > 4096 ||
		m.CreadoUTC.IsZero() || m.CreadoUTC.Location() != time.UTC || m.CreadoUTC.Nanosecond()%1000 != 0 {
		return ErrMensajeInvalido
	}
	switch m.Estado {
	case MensajePendiente:
		if !m.ArchivadoUTC.IsZero() {
			return ErrMensajeInvalido
		}
	case MensajeArchivado:
		if m.ArchivadoUTC.IsZero() || m.ArchivadoUTC.Location() != time.UTC || m.ArchivadoUTC.Nanosecond()%1000 != 0 || m.ArchivadoUTC.Before(m.CreadoUTC) {
			return ErrMensajeInvalido
		}
	default:
		return ErrMensajeInvalido
	}
	return nil
}

type ArchivoMensaje struct {
	MensajeRef, EmpleadoRef, ClaveOperacion string
	VersionEsperada                         int64
	InstanteUTC                             time.Time
}

func (a ArchivoMensaje) Validar() error {
	if !referenciaMarcaje(a.MensajeRef) || !referenciaMarcaje(a.EmpleadoRef) ||
		!claveOperacionMarcaje(a.ClaveOperacion) || a.VersionEsperada < 1 ||
		a.InstanteUTC.IsZero() || a.InstanteUTC.Location() != time.UTC || a.InstanteUTC.Nanosecond()%1000 != 0 {
		return ErrMensajeInvalido
	}
	return nil
}
