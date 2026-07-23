package confianzaatestacion

import (
	"bytes"
	"errors"
	"time"
)

var (
	ErrConfiguracionCapacidadAtestacionV3Invalida = errors.New(
		"vec: configuracion de capacidad de atestacion V3 invalida",
	)
	ErrCapacidadAtestacionV3NoDisponible = errors.New(
		"vec: capacidad de atestacion V3 no disponible",
	)
	ErrCapacidadAtestacionV3Invalida = errors.New(
		"vec: capacidad de atestacion V3 invalida",
	)
	ErrSerializacionCapacidadAtestacionV3Prohibida = errors.New(
		"vec: serializacion generica de capacidad de atestacion V3 prohibida",
	)
)

const (
	VigenciaMaximaCapacidadAtestacionAutorizacionV3 = 5 * time.Second

	minimoBytesClaveHMACCapacidadV3 = 32
	maximoBytesClaveHMACCapacidadV3 = 4096
)

type EstadoClaveHMACCapacidadAtestacionV3 string

const (
	EstadoClaveHMACCapacidadAtestacionV3Emision      EstadoClaveHMACCapacidadAtestacionV3 = "emision"
	EstadoClaveHMACCapacidadAtestacionV3Verificacion EstadoClaveHMACCapacidadAtestacionV3 = "verificacion"
	EstadoClaveHMACCapacidadAtestacionV3Revocada     EstadoClaveHMACCapacidadAtestacionV3 = "revocada"
)

func (e EstadoClaveHMACCapacidadAtestacionV3) valida() bool {
	switch e {
	case EstadoClaveHMACCapacidadAtestacionV3Emision,
		EstadoClaveHMACCapacidadAtestacionV3Verificacion,
		EstadoClaveHMACCapacidadAtestacionV3Revocada:
		return true
	default:
		return false
	}
}

// ClaveHMACCapacidadAtestacionV3 representa material ya resuelto por el
// gobierno de claves. Este constructor no es un HSM: la composicion productiva
// debe sustituirlo por un broker con clave no exportable.
type ClaveHMACCapacidadAtestacionV3 struct {
	bloqueoSerializacionCapacidadV3
	claveID           string
	version           uint64
	material          []byte
	emisorID          string
	audienciaConsumo  string
	estado            EstadoClaveHMACCapacidadAtestacionV3
	validaDesde       time.Time
	validaHasta       time.Time
	revocadaEn        time.Time
	revisionGobierno  uint64
	huellaGobiernoRef string
}

func NuevaClaveHMACCapacidadAtestacionAutorizacionV3(
	claveID string,
	version uint64,
	material []byte,
	emisorID string,
	audienciaConsumo string,
	estado EstadoClaveHMACCapacidadAtestacionV3,
	validaDesde time.Time,
	validaHasta time.Time,
	revocadaEn time.Time,
	revisionGobierno uint64,
	huellaGobiernoRef string,
) (ClaveHMACCapacidadAtestacionV3, error) {
	clave := ClaveHMACCapacidadAtestacionV3{
		claveID: claveID, version: version,
		material: append([]byte(nil), material...),
		emisorID: emisorID, audienciaConsumo: audienciaConsumo,
		estado: estado, validaDesde: validaDesde, validaHasta: validaHasta,
		revocadaEn: revocadaEn, revisionGobierno: revisionGobierno,
		huellaGobiernoRef: huellaGobiernoRef,
	}
	if clave.validar() != nil {
		borrarBytesConfianzaAtestacion(clave.material)
		return ClaveHMACCapacidadAtestacionV3{},
			ErrConfiguracionCapacidadAtestacionV3Invalida
	}
	return clave, nil
}

func (c ClaveHMACCapacidadAtestacionV3) validar() error {
	if !referenciaPruebaConfianzaValida(c.claveID) || c.version == 0 ||
		len(c.material) < minimoBytesClaveHMACCapacidadV3 ||
		len(c.material) > maximoBytesClaveHMACCapacidadV3 ||
		bytes.Equal(c.material, make([]byte, len(c.material))) ||
		!referenciaPruebaConfianzaValida(c.emisorID) ||
		!referenciaPruebaConfianzaValida(c.audienciaConsumo) ||
		!c.estado.valida() ||
		!instanteCanonicoConfianza(c.validaDesde) ||
		!instanteCanonicoConfianza(c.validaHasta) ||
		!c.validaHasta.After(c.validaDesde) ||
		c.revisionGobierno == 0 ||
		!huellaSHA256ConfianzaValida(c.huellaGobiernoRef) {
		return ErrConfiguracionCapacidadAtestacionV3Invalida
	}
	switch c.estado {
	case EstadoClaveHMACCapacidadAtestacionV3Emision,
		EstadoClaveHMACCapacidadAtestacionV3Verificacion:
		if !c.revocadaEn.IsZero() {
			return ErrConfiguracionCapacidadAtestacionV3Invalida
		}
	case EstadoClaveHMACCapacidadAtestacionV3Revocada:
		if !instanteCanonicoConfianza(c.revocadaEn) ||
			c.revocadaEn.Before(c.validaDesde) {
			return ErrConfiguracionCapacidadAtestacionV3Invalida
		}
	default:
		return ErrConfiguracionCapacidadAtestacionV3Invalida
	}
	return nil
}

func (c ClaveHMACCapacidadAtestacionV3) validaParaEmitirEn(
	instante time.Time,
) bool {
	return c.validar() == nil &&
		c.estado == EstadoClaveHMACCapacidadAtestacionV3Emision &&
		c.revocadaEn.IsZero() &&
		!instante.Before(c.validaDesde) &&
		instante.Before(c.validaHasta)
}

func (c ClaveHMACCapacidadAtestacionV3) validaParaVerificarEn(
	instante time.Time,
) bool {
	return c.validar() == nil &&
		(c.estado == EstadoClaveHMACCapacidadAtestacionV3Emision ||
			c.estado == EstadoClaveHMACCapacidadAtestacionV3Verificacion) &&
		c.revocadaEn.IsZero() &&
		!instante.Before(c.validaDesde) &&
		instante.Before(c.validaHasta)
}

func clonarClaveHMACCapacidadAtestacionV3(
	clave ClaveHMACCapacidadAtestacionV3,
) (ClaveHMACCapacidadAtestacionV3, error) {
	if clave.validar() != nil {
		return ClaveHMACCapacidadAtestacionV3{},
			ErrConfiguracionCapacidadAtestacionV3Invalida
	}
	clave.material = append([]byte(nil), clave.material...)
	return clave, nil
}
