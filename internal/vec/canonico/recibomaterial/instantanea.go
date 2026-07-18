package recibomaterial

import (
	"crypto/sha256"
	"time"

	almacencanonico "vec-diputacion-granada/internal/vec/canonico/almacen"
)

// Instantanea conserva los hechos originales del objeto, no los del reintento.
type Instantanea struct {
	Esquema              string
	VersionEsquema       uint16
	ConectorLogicoID     string
	ObjetoRef            string
	ObjetoVersion        string
	Zona                 almacencanonico.Zona
	MIME                 string
	Tamano               int64
	HuellaContenido      [sha256.Size]byte
	EvidenciaCreacionRef string
	AlmacenadoEn         time.Time
	TieneRetencion       bool
	RetenidoHasta        time.Time
	EstadoInmovilizacion string
	EstadoObjeto         string
}

func InstantaneaValida(i Instantanea) bool {
	if i.Esquema != EsquemaInstantanea || i.VersionEsquema != EsquemaVersion ||
		!AliasLogicoValido(i.ConectorLogicoID, 128) || !AliasLogicoValido(i.ObjetoRef, 512) ||
		!AliasLogicoValido(i.ObjetoVersion, 256) || !i.Zona.Valida() || !MIMEValido(i.MIME) ||
		i.Tamano < 1 || i.HuellaContenido == ([sha256.Size]byte{}) ||
		!AliasLogicoValido(i.EvidenciaCreacionRef, 512) || !InstanteValido(i.AlmacenadoEn) ||
		(i.EstadoInmovilizacion != EstadoNoInmovilizado && i.EstadoInmovilizacion != EstadoInmovilizado) ||
		i.EstadoObjeto != EstadoObjetoActivo {
		return false
	}
	if i.TieneRetencion {
		return InstanteValido(i.RetenidoHasta) && i.RetenidoHasta.After(i.AlmacenadoEn)
	}
	return i.RetenidoHasta.IsZero()
}

func CanonicoInstantanea(i Instantanea) ([]byte, error) {
	if !InstantaneaValida(i) {
		return nil, ErrReciboNoValido
	}
	var canonico []byte
	canonico = AnexarTLV(canonico, 0, []byte(i.Esquema))
	canonico = AnexarTLV(canonico, 1, Uint16(i.VersionEsquema))
	canonico = AnexarTLV(canonico, 2, []byte(i.ConectorLogicoID))
	canonico = AnexarTLV(canonico, 3, []byte(i.ObjetoRef))
	canonico = AnexarTLV(canonico, 4, []byte(i.ObjetoVersion))
	canonico = AnexarTLV(canonico, 5, []byte(i.Zona))
	canonico = AnexarTLV(canonico, 6, []byte(i.MIME))
	canonico = AnexarTLV(canonico, 7, Int64(i.Tamano))
	canonico = AnexarTLV(canonico, 8, i.HuellaContenido[:])
	canonico = AnexarTLV(canonico, 9, []byte(i.EvidenciaCreacionRef))
	canonico = AnexarTLV(canonico, 10, Int64(i.AlmacenadoEn.UnixMicro()))
	canonico = AnexarTLV(canonico, 11, Bool(i.TieneRetencion))
	if i.TieneRetencion {
		canonico = AnexarTLV(canonico, 12, Int64(i.RetenidoHasta.UnixMicro()))
	}
	canonico = AnexarTLV(canonico, 13, []byte(i.EstadoInmovilizacion))
	canonico = AnexarTLV(canonico, 14, []byte(i.EstadoObjeto))
	return canonico, nil
}

func HuellaInstantanea(i Instantanea) ([sha256.Size]byte, error) {
	canonico, err := CanonicoInstantanea(i)
	if err != nil {
		return [sha256.Size]byte{}, err
	}
	return sha256.Sum256(canonico), nil
}
