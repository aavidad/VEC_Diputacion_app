package ports

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/x509"
	"encoding/binary"
	"encoding/xml"
	"errors"
	"fmt"
	"hash"
	"io"
	"log/slog"
	"time"
)

const (
	// LimitesMaterialConsumoAutorizacionAtestadaV3 replica la frontera
	// estructural del consumidor PostgreSQL común. El consumidor sigue
	// obligado a validar canon, firmas, gobierno, revocación y vigencia.
	TamanoMinimoCapacidadCanonicaV3      = 512
	TamanoMaximoCapacidadCanonicaV3      = 32 * 1024
	TamanoMaximoDecisionCanonicaV3       = 512 * 1024
	TamanoMaximoMotivoCanonicoV3         = 64 * 1024
	TamanoMaximoContextoActorCanonicoV3  = 256 * 1024
	TamanoMaximoPayloadVECAD3            = 1024 * 1024
	TamanoMaximoSobreCOSESign1V3         = 1024 * 1024
	TamanoMaximoEvidenciaVerificacionV3  = 256 * 1024
	TamanoRaizPublicaSPKIEd25519V3       = 44
	VersionMaximaExactaMaterialConsumoV3 = uint64(1<<53 - 1)
)

var ErrExportacionMaterialConsumoAutorizacionAtestadaV3Invalida = errors.New(
	"vec: exportacion de material de consumo de autorizacion atestada V3 invalida",
)

// ExportacionMaterialConsumoAutorizacionAtestadaV3 es una instantánea
// coherente de las diez entradas que recibe la transacción durable VEC-AD-3.
// Las ocho piezas binarias y las dos versiones se obtienen del mismo material
// nominal. Sus accesores deben devolver copias defensivas.
//
// La exportación no acredita autorización: PostgreSQL debe volver a comprobar
// la capacidad, la decisión, COSE, la raíz gobernada, revocación, vigencia y
// consumo único dentro de la transacción que realiza el efecto.
type ExportacionMaterialConsumoAutorizacionAtestadaV3 struct {
	bloqueoSerializacionMaterialConsumoV3
	capacidadCanonica     []byte
	resumenCapacidad      ResumenCapacidadAtestacionAutorizacionV3
	decisionCanonica      []byte
	motivoCanonico        []byte
	contextoActorCanonico []byte
	personaVersion        uint64
	perfilVersion         uint64
	payloadVECAD3         []byte
	sobreCOSESign1        []byte
	evidenciaVerificacion []byte
	raizPublicaSPKI       []byte
	huellaConjunto        [sha256.Size]byte
}

// NuevaExportacionMaterialConsumoAutorizacionAtestadaV3 crea únicamente el
// transporte estructural. No convierte bytes en una autorización: solo la
// implementación nominal de confianza debe alimentarlo en producción y el
// consumidor PostgreSQL debe verificar el conjunto otra vez.
func NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(
	capacidadCanonica []byte,
	resumenCapacidad ResumenCapacidadAtestacionAutorizacionV3,
	decisionCanonica, motivoCanonico,
	contextoActorCanonico []byte,
	personaVersion, perfilVersion uint64,
	payloadVECAD3, sobreCOSESign1, evidenciaVerificacion,
	raizPublicaSPKI []byte,
) (ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	e := ExportacionMaterialConsumoAutorizacionAtestadaV3{
		capacidadCanonica:     bytes.Clone(capacidadCanonica),
		resumenCapacidad:      resumenCapacidad,
		decisionCanonica:      bytes.Clone(decisionCanonica),
		motivoCanonico:        bytes.Clone(motivoCanonico),
		contextoActorCanonico: bytes.Clone(contextoActorCanonico),
		personaVersion:        personaVersion,
		perfilVersion:         perfilVersion,
		payloadVECAD3:         bytes.Clone(payloadVECAD3),
		sobreCOSESign1:        bytes.Clone(sobreCOSESign1),
		evidenciaVerificacion: bytes.Clone(evidenciaVerificacion),
		raizPublicaSPKI:       bytes.Clone(raizPublicaSPKI),
	}
	if e.validarPiezas() != nil {
		return ExportacionMaterialConsumoAutorizacionAtestadaV3{},
			ErrExportacionMaterialConsumoAutorizacionAtestadaV3Invalida
	}
	e.huellaConjunto = e.calcularHuella()
	return e, nil
}

func (e ExportacionMaterialConsumoAutorizacionAtestadaV3) ValidarEstructura() error {
	if e.validarPiezas() != nil {
		return ErrExportacionMaterialConsumoAutorizacionAtestadaV3Invalida
	}
	actual := e.calcularHuella()
	if subtle.ConstantTimeCompare(actual[:], e.huellaConjunto[:]) != 1 {
		return ErrExportacionMaterialConsumoAutorizacionAtestadaV3Invalida
	}
	return nil
}

func (e ExportacionMaterialConsumoAutorizacionAtestadaV3) validarPiezas() error {
	enRango := func(valor []byte, minimo, maximo int) bool {
		return len(valor) >= minimo && len(valor) <= maximo
	}
	if !enRango(e.capacidadCanonica, TamanoMinimoCapacidadCanonicaV3, TamanoMaximoCapacidadCanonicaV3) ||
		e.resumenCapacidad.ValidarEstructura() != nil ||
		!enRango(e.decisionCanonica, 1, TamanoMaximoDecisionCanonicaV3) ||
		!enRango(e.motivoCanonico, 1, TamanoMaximoMotivoCanonicoV3) ||
		!enRango(e.contextoActorCanonico, 1, TamanoMaximoContextoActorCanonicoV3) ||
		e.personaVersion == 0 || e.personaVersion > VersionMaximaExactaMaterialConsumoV3 ||
		e.perfilVersion == 0 || e.perfilVersion > VersionMaximaExactaMaterialConsumoV3 ||
		!enRango(e.payloadVECAD3, 1, TamanoMaximoPayloadVECAD3) ||
		!enRango(e.sobreCOSESign1, 1, TamanoMaximoSobreCOSESign1V3) ||
		!enRango(e.evidenciaVerificacion, 1, TamanoMaximoEvidenciaVerificacionV3) ||
		len(e.raizPublicaSPKI) != TamanoRaizPublicaSPKIEd25519V3 {
		return ErrExportacionMaterialConsumoAutorizacionAtestadaV3Invalida
	}
	clave, err := x509.ParsePKIXPublicKey(e.raizPublicaSPKI)
	publica, correcta := clave.(ed25519.PublicKey)
	if err != nil || !correcta || len(publica) != ed25519.PublicKeySize {
		return ErrExportacionMaterialConsumoAutorizacionAtestadaV3Invalida
	}
	canonica, err := x509.MarshalPKIXPublicKey(publica)
	if err != nil || !bytes.Equal(canonica, e.raizPublicaSPKI) {
		return ErrExportacionMaterialConsumoAutorizacionAtestadaV3Invalida
	}
	return nil
}

func (e ExportacionMaterialConsumoAutorizacionAtestadaV3) calcularHuella() [sha256.Size]byte {
	calculador := sha256.New()
	for _, contenido := range [][]byte{
		e.capacidadCanonica,
		[]byte(e.resumenCapacidad.DecisionRef()),
		[]byte(e.resumenCapacidad.DecisionHuellaSHA256()),
		[]byte(e.resumenCapacidad.MotivoHuellaSHA256()),
		[]byte(e.resumenCapacidad.ContextoRef()),
		[]byte(e.resumenCapacidad.ContextoHuellaSHA256()),
		[]byte(e.resumenCapacidad.Operacion()),
		[]byte(e.resumenCapacidad.EfectoRef()),
		[]byte(e.resumenCapacidad.EfectoHuellaSHA256()),
		[]byte(e.resumenCapacidad.AudienciaConsumo()),
		[]byte(e.resumenCapacidad.EmitidaEn().Format(time.RFC3339Nano)),
		[]byte(e.resumenCapacidad.ExpiraEn().Format(time.RFC3339Nano)),
		e.decisionCanonica,
		e.motivoCanonico,
		e.contextoActorCanonico,
	} {
		escribirBloqueHuellaMaterialConsumoV3(calculador, contenido)
	}
	var version [8]byte
	binary.BigEndian.PutUint64(version[:], e.personaVersion)
	escribirBloqueHuellaMaterialConsumoV3(calculador, version[:])
	binary.BigEndian.PutUint64(version[:], e.perfilVersion)
	escribirBloqueHuellaMaterialConsumoV3(calculador, version[:])
	for _, contenido := range [][]byte{
		e.payloadVECAD3, e.sobreCOSESign1, e.evidenciaVerificacion,
		e.raizPublicaSPKI,
	} {
		escribirBloqueHuellaMaterialConsumoV3(calculador, contenido)
	}
	var resultado [sha256.Size]byte
	copy(resultado[:], calculador.Sum(nil))
	return resultado
}

func escribirBloqueHuellaMaterialConsumoV3(calculador hash.Hash, contenido []byte) {
	var longitud [8]byte
	binary.BigEndian.PutUint64(longitud[:], uint64(len(contenido)))
	_, _ = calculador.Write(longitud[:])
	_, _ = calculador.Write(contenido)
}

func (e ExportacionMaterialConsumoAutorizacionAtestadaV3) CapacidadCanonica() []byte {
	return bytes.Clone(e.capacidadCanonica)
}
func (e ExportacionMaterialConsumoAutorizacionAtestadaV3) ResumenCapacidad() ResumenCapacidadAtestacionAutorizacionV3 {
	return e.resumenCapacidad
}
func (e ExportacionMaterialConsumoAutorizacionAtestadaV3) DecisionCanonica() []byte {
	return bytes.Clone(e.decisionCanonica)
}
func (e ExportacionMaterialConsumoAutorizacionAtestadaV3) MotivoCanonico() []byte {
	return bytes.Clone(e.motivoCanonico)
}
func (e ExportacionMaterialConsumoAutorizacionAtestadaV3) ContextoActorCanonico() []byte {
	return bytes.Clone(e.contextoActorCanonico)
}
func (e ExportacionMaterialConsumoAutorizacionAtestadaV3) PersonaVersion() uint64 {
	return e.personaVersion
}
func (e ExportacionMaterialConsumoAutorizacionAtestadaV3) PerfilVersion() uint64 {
	return e.perfilVersion
}
func (e ExportacionMaterialConsumoAutorizacionAtestadaV3) PayloadVECAD3() []byte {
	return bytes.Clone(e.payloadVECAD3)
}
func (e ExportacionMaterialConsumoAutorizacionAtestadaV3) SobreCOSESign1() []byte {
	return bytes.Clone(e.sobreCOSESign1)
}
func (e ExportacionMaterialConsumoAutorizacionAtestadaV3) EvidenciaVerificacion() []byte {
	return bytes.Clone(e.evidenciaVerificacion)
}
func (e ExportacionMaterialConsumoAutorizacionAtestadaV3) RaizPublicaSPKI() []byte {
	return bytes.Clone(e.raizPublicaSPKI)
}

// ExportadorMaterialConsumoAutorizacionAtestadaV3 es el puerto neutral del
// consumidor durable. Una única llamada fija la instantánea completa para
// evitar mezclar piezas procedentes de dos decisiones o verificaciones.
//
// El puerto no depende de HTTP, de una operación funcional concreta ni de una
// base de datos. La composición productiva debe inyectar la implementación
// nominal de confianza; nunca debe reconstruirla desde bytes del cliente.
type ExportadorMaterialConsumoAutorizacionAtestadaV3 interface {
	fmt.Stringer
	slog.LogValuer

	ExportarMaterialParaConsumidor() (
		ExportacionMaterialConsumoAutorizacionAtestadaV3,
		error,
	)
}

type bloqueoSerializacionMaterialConsumoV3 struct{}

func (bloqueoSerializacionMaterialConsumoV3) MarshalJSON() ([]byte, error) {
	return nil, ErrExportacionMaterialConsumoAutorizacionAtestadaV3Invalida
}
func (*bloqueoSerializacionMaterialConsumoV3) UnmarshalJSON([]byte) error {
	return ErrExportacionMaterialConsumoAutorizacionAtestadaV3Invalida
}
func (bloqueoSerializacionMaterialConsumoV3) MarshalText() ([]byte, error) {
	return nil, ErrExportacionMaterialConsumoAutorizacionAtestadaV3Invalida
}
func (*bloqueoSerializacionMaterialConsumoV3) UnmarshalText([]byte) error {
	return ErrExportacionMaterialConsumoAutorizacionAtestadaV3Invalida
}
func (bloqueoSerializacionMaterialConsumoV3) MarshalBinary() ([]byte, error) {
	return nil, ErrExportacionMaterialConsumoAutorizacionAtestadaV3Invalida
}
func (*bloqueoSerializacionMaterialConsumoV3) UnmarshalBinary([]byte) error {
	return ErrExportacionMaterialConsumoAutorizacionAtestadaV3Invalida
}
func (bloqueoSerializacionMaterialConsumoV3) GobEncode() ([]byte, error) {
	return nil, ErrExportacionMaterialConsumoAutorizacionAtestadaV3Invalida
}
func (*bloqueoSerializacionMaterialConsumoV3) GobDecode([]byte) error {
	return ErrExportacionMaterialConsumoAutorizacionAtestadaV3Invalida
}
func (bloqueoSerializacionMaterialConsumoV3) MarshalCBOR() ([]byte, error) {
	return nil, ErrExportacionMaterialConsumoAutorizacionAtestadaV3Invalida
}
func (*bloqueoSerializacionMaterialConsumoV3) UnmarshalCBOR([]byte) error {
	return ErrExportacionMaterialConsumoAutorizacionAtestadaV3Invalida
}
func (bloqueoSerializacionMaterialConsumoV3) MarshalYAML() (any, error) {
	return nil, ErrExportacionMaterialConsumoAutorizacionAtestadaV3Invalida
}
func (*bloqueoSerializacionMaterialConsumoV3) UnmarshalYAML(func(any) error) error {
	return ErrExportacionMaterialConsumoAutorizacionAtestadaV3Invalida
}
func (bloqueoSerializacionMaterialConsumoV3) MarshalXML(*xml.Encoder, xml.StartElement) error {
	return ErrExportacionMaterialConsumoAutorizacionAtestadaV3Invalida
}
func (*bloqueoSerializacionMaterialConsumoV3) UnmarshalXML(*xml.Decoder, xml.StartElement) error {
	return ErrExportacionMaterialConsumoAutorizacionAtestadaV3Invalida
}
func (bloqueoSerializacionMaterialConsumoV3) String() string {
	return "[MATERIAL-CONSUMO-AUTORIZACION-ATESTADA-V3-OPACO]"
}
func (b bloqueoSerializacionMaterialConsumoV3) GoString() string {
	return b.String()
}
func (b bloqueoSerializacionMaterialConsumoV3) Format(s fmt.State, _ rune) {
	_, _ = io.WriteString(s, b.String())
}
func (b bloqueoSerializacionMaterialConsumoV3) LogValue() slog.Value {
	return slog.StringValue(b.String())
}
