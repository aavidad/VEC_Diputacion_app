package cobertura

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"
)

const (
	MaximoEnteroSeguroOperacionDecisionCobertura    uint64 = 1<<53 - 1
	MaximoLeaseOperacionDecisionCobertura                  = 5 * time.Second
	bytesTokenPropietarioOperacionDecisionCobertura        = 32
	redaccionOperacionDecisionCobertura                    = "[OPERACION-DECISION-COBERTURA-OPACA]"
)

var (
	ErrOperacionDecisionCoberturaIdempotenteInvalida = errors.New(
		"contratacion temporal: operacion de decision de cobertura idempotente invalida",
	)
	ErrSerializacionOperacionDecisionCoberturaProhibida = errors.New(
		"contratacion temporal: serializacion de operacion de decision de cobertura prohibida",
	)
	ErrTokenPropietarioOperacionDecisionCoberturaInvalido = errors.New(
		"contratacion temporal: token propietario de decision de cobertura invalido",
	)
	ErrClaveOperacionDecisionCoberturaUsada = errors.New(
		"contratacion temporal: clave de decision de cobertura usada con otra semantica",
	)
)

// TokenPropietarioOperacionDecisionCobertura es secreto efímero generado por
// CSPRNG. Solo su SHA-256 puede persistirse. Ni el token ni su hash conceden
// autoridad: la transacción durable debe cotejar además el cercado vigente.
type TokenPropietarioOperacionDecisionCobertura struct {
	bloqueoSerializacionOperacionDecisionCobertura
	secreto [bytesTokenPropietarioOperacionDecisionCobertura]byte
}

func GenerarTokenPropietarioOperacionDecisionCobertura() (
	TokenPropietarioOperacionDecisionCobertura,
	error,
) {
	var token TokenPropietarioOperacionDecisionCobertura
	if _, err := io.ReadFull(rand.Reader, token.secreto[:]); err != nil ||
		token.esCero() {
		return TokenPropietarioOperacionDecisionCobertura{},
			ErrTokenPropietarioOperacionDecisionCoberturaInvalido
	}
	return token, nil
}

func (t TokenPropietarioOperacionDecisionCobertura) HuellaSHA256() (
	string,
	error,
) {
	if t.esCero() {
		return "", ErrTokenPropietarioOperacionDecisionCoberturaInvalido
	}
	huella := sha256.Sum256(t.secreto[:])
	return hex.EncodeToString(huella[:]), nil
}

func (t TokenPropietarioOperacionDecisionCobertura) CoincideConHuellaSHA256(
	huella string,
) bool {
	calculada, err := t.HuellaSHA256()
	return err == nil && huellaSHA256OperacionDecisionCoberturaValida(huella) &&
		subtle.ConstantTimeCompare([]byte(calculada), []byte(huella)) == 1
}

func (t TokenPropietarioOperacionDecisionCobertura) esCero() bool {
	var cero [bytesTokenPropietarioOperacionDecisionCobertura]byte
	return subtle.ConstantTimeCompare(t.secreto[:], cero[:]) == 1
}

type bloqueoSerializacionOperacionDecisionCobertura struct{}

func (bloqueoSerializacionOperacionDecisionCobertura) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionOperacionDecisionCoberturaProhibida
}
func (*bloqueoSerializacionOperacionDecisionCobertura) UnmarshalJSON([]byte) error {
	return ErrSerializacionOperacionDecisionCoberturaProhibida
}
func (bloqueoSerializacionOperacionDecisionCobertura) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error {
	return ErrSerializacionOperacionDecisionCoberturaProhibida
}
func (*bloqueoSerializacionOperacionDecisionCobertura) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error {
	return ErrSerializacionOperacionDecisionCoberturaProhibida
}
func (bloqueoSerializacionOperacionDecisionCobertura) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionOperacionDecisionCoberturaProhibida
}
func (*bloqueoSerializacionOperacionDecisionCobertura) UnmarshalText([]byte) error {
	return ErrSerializacionOperacionDecisionCoberturaProhibida
}
func (bloqueoSerializacionOperacionDecisionCobertura) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionOperacionDecisionCoberturaProhibida
}
func (*bloqueoSerializacionOperacionDecisionCobertura) UnmarshalBinary([]byte) error {
	return ErrSerializacionOperacionDecisionCoberturaProhibida
}
func (bloqueoSerializacionOperacionDecisionCobertura) GobEncode() ([]byte, error) {
	return nil, ErrSerializacionOperacionDecisionCoberturaProhibida
}
func (*bloqueoSerializacionOperacionDecisionCobertura) GobDecode([]byte) error {
	return ErrSerializacionOperacionDecisionCoberturaProhibida
}
func (bloqueoSerializacionOperacionDecisionCobertura) MarshalCBOR() ([]byte, error) {
	return nil, ErrSerializacionOperacionDecisionCoberturaProhibida
}
func (*bloqueoSerializacionOperacionDecisionCobertura) UnmarshalCBOR([]byte) error {
	return ErrSerializacionOperacionDecisionCoberturaProhibida
}
func (bloqueoSerializacionOperacionDecisionCobertura) MarshalYAML() (any, error) {
	return nil, ErrSerializacionOperacionDecisionCoberturaProhibida
}
func (*bloqueoSerializacionOperacionDecisionCobertura) UnmarshalYAML(func(any) error) error {
	return ErrSerializacionOperacionDecisionCoberturaProhibida
}
func (bloqueoSerializacionOperacionDecisionCobertura) String() string {
	return redaccionOperacionDecisionCobertura
}
func (b bloqueoSerializacionOperacionDecisionCobertura) GoString() string {
	return b.String()
}
func (b bloqueoSerializacionOperacionDecisionCobertura) Format(
	estado fmt.State,
	_ rune,
) {
	_, _ = io.WriteString(estado, b.String())
}
func (b bloqueoSerializacionOperacionDecisionCobertura) LogValue() slog.Value {
	return slog.StringValue(b.String())
}

func huellaSHA256OperacionDecisionCoberturaValida(valor string) bool {
	if len(valor) != sha256.Size*2 || valor == strings.Repeat("0", sha256.Size*2) {
		return false
	}
	for _, caracter := range valor {
		if (caracter < '0' || caracter > '9') &&
			(caracter < 'a' || caracter > 'f') {
			return false
		}
	}
	return true
}

func instanteOperacionDecisionCoberturaValido(instante time.Time) bool {
	return !instante.IsZero() && instante.Location() == time.UTC &&
		instante.Year() >= 1 && instante.Year() <= 9999 &&
		instante.Nanosecond()%1_000 == 0
}

func referenciasOperacionDecisionCoberturaIguales(
	primera string,
	segunda string,
) bool {
	return len(primera) == len(segunda) &&
		subtle.ConstantTimeCompare([]byte(primera), []byte(segunda)) == 1
}

func referenciaDecisionCoberturaLigadaAHuella(
	referencia string,
	huella string,
) bool {
	if !huellaSHA256OperacionDecisionCoberturaValida(huella) {
		return false
	}
	esperada := "decision-cobertura:sha256:" + huella
	return referenciasOperacionDecisionCoberturaIguales(referencia, esperada)
}
