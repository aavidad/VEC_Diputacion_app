package ports

import (
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"strings"
)

const maximoLongitudRepresentacionTokenReservaBaremacion = 128

// operacionTokenReservaBaremacion mantiene la representacion de la capacidad
// dentro de un unico cierre privado e inmutable. Sin destino ni huella
// esperada devuelve exclusivamente el selector SHA-256 historico; con destino
// escribe una parte canonica completa y con huella realiza la comparacion
// autoritativa sin revelar el material capturado.
type operacionTokenReservaBaremacion func(
	destino *bytes.Buffer,
	huellaEsperada *string,
) (huella string, coincide bool)

// TokenReservaBaremacion es una capacidad temporal, nunca un identificador de
// negocio. Su representacion Base64URL solo vive capturada por un cierre
// privado: el valor no puede recuperarse mediante la API de reflexion segura.
//
// La huella durable conserva deliberadamente el contrato historico exacto
// SHA-256(Base64URL), sin decodificar ni anadir un dominio. Cambiar esa formula
// impediria localizar reservas V1/V3 y alteraria los vectores probatorios.
type TokenReservaBaremacion struct {
	operar operacionTokenReservaBaremacion
}

// NuevoTokenReservaBaremacion valida e importa la representacion Base64URL
// canonica empleada por el contrato historico. El tipo resultante no ofrece
// ninguna operacion publica para volver a obtenerla.
func NuevoTokenReservaBaremacion(valor string) (TokenReservaBaremacion, error) {
	if !tokenBase64URLValido(valor) {
		return TokenReservaBaremacion{}, ErrTokenReservaBaremacionInvalido
	}
	var representacion [maximoLongitudRepresentacionTokenReservaBaremacion]byte
	longitud := copy(representacion[:], valor)
	operar := nuevaOperacionTokenReservaBaremacion(representacion, longitud)
	clear(representacion[:])
	return TokenReservaBaremacion{operar: operar}, nil
}

// nuevaOperacionTokenReservaBaremacion recibe el array por valor para que el
// cierre capture una copia independiente del buffer que borra el constructor.
func nuevaOperacionTokenReservaBaremacion(
	representacion [maximoLongitudRepresentacionTokenReservaBaremacion]byte,
	longitud int,
) operacionTokenReservaBaremacion {
	return func(destino *bytes.Buffer, huellaEsperada *string) (string, bool) {
		if longitud < 1 || longitud > len(representacion) ||
			(destino != nil && huellaEsperada != nil) {
			return "", false
		}
		material := representacion[:longitud]
		if destino != nil {
			var longitudCanonica [8]byte
			binary.BigEndian.PutUint64(longitudCanonica[:], uint64(longitud))
			_, _ = destino.Write(longitudCanonica[:])
			_, _ = destino.Write(material)
			return "", true
		}

		suma := sha256.Sum256(material)
		if huellaEsperada == nil {
			return hex.EncodeToString(suma[:]), true
		}
		if len(*huellaEsperada) != hex.EncodedLen(sha256.Size) ||
			strings.ToLower(*huellaEsperada) != *huellaEsperada {
			return "", false
		}
		var esperada [sha256.Size]byte
		if _, err := hex.Decode(esperada[:], []byte(*huellaEsperada)); err != nil {
			return "", false
		}
		return "", subtle.ConstantTimeCompare(suma[:], esperada[:]) == 1
	}
}

func (t TokenReservaBaremacion) Validar() error {
	if t.operar == nil {
		return ErrTokenReservaBaremacionInvalido
	}
	return nil
}

// HuellaSHA256 devuelve exclusivamente el selector durable historico. El
// material de la capacidad no forma parte del valor devuelto.
func (t TokenReservaBaremacion) HuellaSHA256() (string, error) {
	if t.Validar() != nil {
		return "", ErrTokenReservaBaremacionInvalido
	}
	huella, valida := t.operar(nil, nil)
	if !valida {
		return "", ErrTokenReservaBaremacionInvalido
	}
	return huella, nil
}

// CoincideConHuellaSHA256 compara los 32 bytes en tiempo constante y rechaza
// representaciones hexadecimales no canonicas.
func (t TokenReservaBaremacion) CoincideConHuellaSHA256(huella string) bool {
	if t.Validar() != nil {
		return false
	}
	_, coincide := t.operar(nil, &huella)
	return coincide
}

// escribirParteCanonica conserva la disposicion binaria historica sin
// devolver la representacion de la capacidad al llamador.
func (t TokenReservaBaremacion) escribirParteCanonica(destino *bytes.Buffer) bool {
	if t.Validar() != nil || destino == nil {
		return false
	}
	_, escrita := t.operar(destino, nil)
	return escrita
}

func tokenBase64URLValido(valor string) bool {
	if valor == "" || valor != strings.TrimSpace(valor) || len(valor) < 32 ||
		len(valor) > maximoLongitudRepresentacionTokenReservaBaremacion || strings.Contains(valor, "=") {
		return false
	}
	decodificado, err := base64.RawURLEncoding.DecodeString(valor)
	if len(decodificado) != 0 {
		defer clear(decodificado)
	}
	return err == nil && len(decodificado) >= 24 && len(decodificado) <= 96 &&
		base64.RawURLEncoding.EncodeToString(decodificado) == valor
}

func (TokenReservaBaremacion) String() string { return "[TOKEN-RESERVA-OCULTO]" }
func (TokenReservaBaremacion) GoString() string {
	return "ports.TokenReservaBaremacion{[OCULTO]}"
}
func (t TokenReservaBaremacion) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, t.String())
}
func (t TokenReservaBaremacion) LogValue() slog.Value {
	return slog.StringValue(t.String())
}
func (TokenReservaBaremacion) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionTokenReservaProhibida
}
func (*TokenReservaBaremacion) UnmarshalJSON([]byte) error {
	return ErrSerializacionTokenReservaProhibida
}
func (TokenReservaBaremacion) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionTokenReservaProhibida
}
func (*TokenReservaBaremacion) UnmarshalText([]byte) error {
	return ErrSerializacionTokenReservaProhibida
}
func (TokenReservaBaremacion) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionTokenReservaProhibida
}
func (*TokenReservaBaremacion) UnmarshalBinary([]byte) error {
	return ErrSerializacionTokenReservaProhibida
}
func (TokenReservaBaremacion) GobEncode() ([]byte, error) {
	return nil, ErrSerializacionTokenReservaProhibida
}
func (*TokenReservaBaremacion) GobDecode([]byte) error {
	return ErrSerializacionTokenReservaProhibida
}
func (TokenReservaBaremacion) MarshalXML(*xml.Encoder, xml.StartElement) error {
	return ErrSerializacionTokenReservaProhibida
}
func (*TokenReservaBaremacion) UnmarshalXML(*xml.Decoder, xml.StartElement) error {
	return ErrSerializacionTokenReservaProhibida
}
