package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math"
)

const (
	// VersionFormatoAtestacionAutorizacionV3 identifica VEC-AD-3. VEC-AD-2
	// permanece inmutable para decisiones ligadas V2.
	VersionFormatoAtestacionAutorizacionV3 uint16 = 3

	// EsquemaMensajeAtestacionAutorizacionV3 separa criptograficamente las
	// decisiones V3 ligadas a un contexto de actor V2 registrado.
	EsquemaMensajeAtestacionAutorizacionV3 = "VEC-AUTORIZACION-ATESTACION-V3-CONTEXTO-ACTOR-V2-MOTIVO-CATALOGADO"

	TamanoMaximoMensajeAtestacionAutorizacionV3 = 512 * 1024
)

// CabeceraAtestacionAutorizacionV3 debe resolverla la composicion confiable.
// No es una entrada de usuario ni selecciona por si sola un proveedor.
type CabeceraAtestacionAutorizacionV3 struct {
	FormatoVersion uint16
	Suite          string
	ClaveID        string
	Audiencia      string
}

func (c CabeceraAtestacionAutorizacionV3) Validar() error {
	if c.FormatoVersion != VersionFormatoAtestacionAutorizacionV3 ||
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

// SerializarMensajeAtestacionAutorizacionV3 produce la unica preimagen
// VEC-AD-3. Compromete los documentos canonicos completos de decision, motivo,
// contexto y procedencia, ademas de la referencia durable de contexto.
//
// El mensaje no es una capacidad de efecto. La firma, la confianza, la
// revocacion y el consumo unico se comprueban en la frontera atestada.
func SerializarMensajeAtestacionAutorizacionV3(
	cabecera CabeceraAtestacionAutorizacionV3,
	decision DecisionAutorizacionLigadaV3,
	referenciaMotivo ReferenciaEntradaCatalogo,
	resultadoContexto ResultadoContextoActorRegistradoV2,
) ([]byte, error) {
	if err := cabecera.Validar(); err != nil {
		return nil, err
	}
	if err := validarDecisionParaAtestacionAutorizacionV3(
		decision,
		referenciaMotivo,
		resultadoContexto,
	); err != nil {
		return nil, err
	}

	decisionCanonica, err := RepresentacionCanonicaDecisionAutorizacionV3(decision)
	if err != nil {
		return nil, errorMensajeAtestacionAutorizacionInvalido()
	}
	motivoCanonico, err := RepresentacionCanonicaMotivoAutorizacionV2(referenciaMotivo)
	if err != nil {
		return nil, errorMensajeAtestacionAutorizacionInvalido()
	}
	contexto, err := resultadoContexto.Clonar()
	if err != nil {
		return nil, errorMensajeAtestacionAutorizacionInvalido()
	}

	escritor := nuevoEscritorAtestacionAutorizacionV1()
	escritor.escribirBytes([]byte(EsquemaMensajeAtestacionAutorizacionV3))
	escritor.escribirByte(0)
	escritor.escribirUint16(cabecera.FormatoVersion)
	escritor.escribirTexto(cabecera.Suite)
	escritor.escribirTexto(cabecera.ClaveID)
	escritor.escribirTexto(cabecera.Audiencia)
	escribirBloqueAtestacionAutorizacionV3(escritor, decisionCanonica)
	escribirBloqueAtestacionAutorizacionV3(escritor, motivoCanonico)
	escritor.escribirTexto(contexto.RegistroContextoRef)
	escribirBloqueAtestacionAutorizacionV3(escritor, contexto.RepresentacionCanonica)
	escritor.escribirTexto(contexto.HuellaSHA256)
	escribirBloqueAtestacionAutorizacionV3(
		escritor,
		contexto.ManifiestoProcedenciaCanonico,
	)
	escritor.escribirTexto(contexto.ManifiestoProcedenciaHuellaSHA256)
	escritor.escribirTexto(string(contexto.AutoridadEfectiva))
	escritor.escribirInstante(contexto.ResueltoEnAutoritativo)
	if escritor.err != nil ||
		escritor.buffer.Len() > TamanoMaximoMensajeAtestacionAutorizacionV3-8 {
		return nil, errorMensajeAtestacionAutorizacionInvalido()
	}

	longitudTotal := uint64(escritor.buffer.Len() + 8)
	escritor.escribirUint64(longitudTotal)
	if escritor.err != nil || escritor.buffer.Len() != int(longitudTotal) ||
		escritor.buffer.Len() > TamanoMaximoMensajeAtestacionAutorizacionV3 {
		return nil, errorMensajeAtestacionAutorizacionInvalido()
	}
	return append([]byte(nil), escritor.buffer.Bytes()...), nil
}

// HuellaSHA256MensajeAtestacionAutorizacionV3 publica un vector de integridad,
// no una firma ni una concesion.
func HuellaSHA256MensajeAtestacionAutorizacionV3(
	cabecera CabeceraAtestacionAutorizacionV3,
	decision DecisionAutorizacionLigadaV3,
	referenciaMotivo ReferenciaEntradaCatalogo,
	resultadoContexto ResultadoContextoActorRegistradoV2,
) (string, error) {
	mensaje, err := SerializarMensajeAtestacionAutorizacionV3(
		cabecera,
		decision,
		referenciaMotivo,
		resultadoContexto,
	)
	if err != nil {
		return "", err
	}
	suma := sha256.Sum256(mensaje)
	return hex.EncodeToString(suma[:]), nil
}

func validarDecisionParaAtestacionAutorizacionV3(
	decision DecisionAutorizacionLigadaV3,
	referenciaMotivo ReferenciaEntradaCatalogo,
	resultadoContexto ResultadoContextoActorRegistradoV2,
) error {
	if decision.Validar() != nil || resultadoContexto.Validar() != nil ||
		!ReferenciaMotivoAutorizacionV2Valida(referenciaMotivo) ||
		decision.datos == nil {
		return errorMensajeAtestacionAutorizacionInvalido()
	}
	concedida, codigo, err := decision.Resultado()
	huellaMotivo, errMotivo := HuellaSHA256MotivoAutorizacionV2(referenciaMotivo)
	if err != nil || errMotivo != nil || !concedida || codigo != "concedida" ||
		decision.datos.motivoHuellaSHA256 != huellaMotivo ||
		decision.datos.vinculoAutenticacionActor.ValidarPara(resultadoContexto) != nil ||
		!decision.datos.vinculoAutenticacionActor.VigenteEn(
			decision.datos.emitidaEn,
			resultadoContexto,
		) {
		return errors.Join(
			errorMensajeAtestacionAutorizacionInvalido(),
			err,
			errMotivo,
		)
	}
	return nil
}

func escribirBloqueAtestacionAutorizacionV3(
	escritor *escritorAtestacionAutorizacionV1,
	contenido []byte,
) {
	if escritor == nil || len(contenido) == 0 || uint64(len(contenido)) > math.MaxUint32 {
		if escritor != nil && escritor.err == nil {
			escritor.err = errorMensajeAtestacionAutorizacionInvalido()
		}
		return
	}
	escritor.escribirUint32(uint32(len(contenido)))
	escritor.escribirBytes(contenido)
}
