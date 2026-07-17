package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

const (
	// VersionFormatoAtestacionDenegacionAutorizacionV1 identifica el contrato
	// binario VEC-AD-D-1. Es un espacio de versiones propio: el valor 1 no lo
	// convierte en VEC-AD-1 ni permite tratar una denegacion como concesion.
	VersionFormatoAtestacionDenegacionAutorizacionV1 uint16 = 1

	// EsquemaMensajeAtestacionDenegacionAutorizacionV1 separa de forma
	// criptografica una prueba negativa de VEC-AD-1, VEC-AD-2 y cualquier otro
	// protocolo. El byte cero posterior tambien forma parte del mensaje.
	EsquemaMensajeAtestacionDenegacionAutorizacionV1 = "VEC-AUTORIZACION-DENEGACION-ATESTACION-V1-SOLICITUD-LIGADA-MOTIVO-CATALOGADO"

	// TamanoMaximoMensajeAtestacionDenegacionAutorizacionV1 fija el mismo techo
	// operativo de 512 KiB mediante una constante independiente.
	TamanoMaximoMensajeAtestacionDenegacionAutorizacionV1 = 512 * 1024
)

// CabeceraAtestacionDenegacionAutorizacionV1 es nominalmente distinta de las
// cabeceras de concesion. Suite, clave y audiencia se fijan en composicion; el
// tipo solo define los bytes canonicos y no aprueba un proveedor de firma.
type CabeceraAtestacionDenegacionAutorizacionV1 struct {
	FormatoVersion uint16
	Suite          string
	ClaveID        string
	Audiencia      string
}

func (c CabeceraAtestacionDenegacionAutorizacionV1) Validar() error {
	if c.FormatoVersion != VersionFormatoAtestacionDenegacionAutorizacionV1 ||
		!identificadorCabeceraAtestacionValido(c.Suite, 128) ||
		!identificadorCabeceraAtestacionValido(c.ClaveID, 512) ||
		!identificadorCabeceraAtestacionValido(c.Audiencia, 512) {
		return errors.Join(
			ErrConfiguracionAccesoInvalida,
			ErrMensajeAtestacionAutorizacionInvalido,
		)
	}
	return nil
}

// SerializarMensajeAtestacionDenegacionAutorizacionV1 produce VEC-AD-D-1: la
// prueba binaria de una decision negativa V2 y de la referencia catalogada que
// el PDP resolvio al evaluarla. Nunca acepta una concesion ni emite VEC-AD-2.
// GarantiaMinima puede estar vacia en denegaciones anteriores a seleccionar
// una concesion; si esta presente debe pertenecer al vocabulario gobernado.
// Las listas deben llegar ya ordenadas por bytes UTF-8, igual que en VEC-AD-2.
func SerializarMensajeAtestacionDenegacionAutorizacionV1(
	cabecera CabeceraAtestacionDenegacionAutorizacionV1,
	decision DecisionAutorizacion,
	referenciaMotivo ReferenciaEntradaCatalogo,
) ([]byte, error) {
	if !comprobarEsquemaDecisionAtestacionAutorizacionV2() ||
		!limitesEscritorAtestacionDenegacionAutorizacionV1Compatibles(
			TamanoMaximoMensajeAtestacionDenegacionAutorizacionV1,
			TamanoMaximoMensajeAtestacionAutorizacionV1,
		) {
		return nil, errors.Join(
			ErrConfiguracionAccesoInvalida,
			ErrMensajeAtestacionAutorizacionInvalido,
		)
	}
	if err := cabecera.Validar(); err != nil {
		return nil, err
	}
	if err := validarDecisionParaAtestacionDenegacionAutorizacionV1(
		decision,
		referenciaMotivo,
	); err != nil {
		return nil, err
	}

	escritor := nuevoEscritorAtestacionAutorizacionV1()
	escritor.escribirBytes([]byte(EsquemaMensajeAtestacionDenegacionAutorizacionV1))
	escritor.escribirByte(0)
	escritor.escribirUint16(cabecera.FormatoVersion)
	escritor.escribirTexto(cabecera.Suite)
	escritor.escribirTexto(cabecera.ClaveID)
	escritor.escribirTexto(cabecera.Audiencia)
	escribirDecisionAtestacionAutorizacionSolicitudLigadaV2(escritor, decision)
	escribirReferenciaMotivoAtestacionAutorizacionSolicitudLigadaV2(
		escritor,
		referenciaMotivo,
	)
	if escritor.err != nil {
		return nil, escritor.err
	}

	if escritor.buffer.Len() > TamanoMaximoMensajeAtestacionDenegacionAutorizacionV1-8 {
		return nil, errorMensajeAtestacionAutorizacionInvalido()
	}
	longitudTotal := uint64(escritor.buffer.Len() + 8)
	escritor.escribirUint64(longitudTotal)
	if escritor.err != nil || escritor.buffer.Len() != int(longitudTotal) ||
		escritor.buffer.Len() > TamanoMaximoMensajeAtestacionDenegacionAutorizacionV1 {
		return nil, errorMensajeAtestacionAutorizacionInvalido()
	}
	return append([]byte(nil), escritor.buffer.Bytes()...), nil
}

// HuellaSHA256MensajeAtestacionDenegacionAutorizacionV1 publica un vector de
// integridad. La huella no acredita al PDP y no sustituye la firma del sobre.
func HuellaSHA256MensajeAtestacionDenegacionAutorizacionV1(
	cabecera CabeceraAtestacionDenegacionAutorizacionV1,
	decision DecisionAutorizacion,
	referenciaMotivo ReferenciaEntradaCatalogo,
) (string, error) {
	mensaje, err := SerializarMensajeAtestacionDenegacionAutorizacionV1(
		cabecera,
		decision,
		referenciaMotivo,
	)
	if err != nil {
		return "", err
	}
	suma := sha256.Sum256(mensaje)
	return hex.EncodeToString(suma[:]), nil
}

func validarDecisionParaAtestacionDenegacionAutorizacionV1(
	decision DecisionAutorizacion,
	referenciaMotivo ReferenciaEntradaCatalogo,
) error {
	if err := validarDecisionParaAtestacionAutorizacionSolicitudLigadaV2(
		decision,
		referenciaMotivo,
	); err != nil || decision.Concedida || decision.Codigo == "concedida" ||
		(decision.GarantiaMinima != "" && !decision.GarantiaMinima.Valida()) {
		return errors.Join(errorMensajeAtestacionAutorizacionInvalido(), err)
	}
	return nil
}

func limitesEscritorAtestacionDenegacionAutorizacionV1Compatibles(
	limiteDenegacion,
	limiteEscritorV1 int,
) bool {
	return limiteDenegacion == 512*1024 && limiteDenegacion == limiteEscritorV1
}
