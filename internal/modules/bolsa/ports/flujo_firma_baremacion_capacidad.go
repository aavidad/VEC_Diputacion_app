package ports

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"time"
)

const tamanoTokenArrendamientoFlujoFirma = 32

type operacionHMACTokenArrendamientoFlujoFirma func(
	clave, huellaEsperada []byte,
) (huellaCalculada []byte, coincide bool, err error)

// TokenArrendamientoFlujoFirmaBaremacion es la capacidad aleatoria que
// autoriza a usar un arrendamiento. Su valor solo vive en el cierre privado e
// inmutable operarHMAC: no tiene representacion generica ni ofrece una
// operacion para revelarlo. Los adaptadores conservan exclusivamente una
// huella HMAC y la verifican antes de aceptar cualquier cambio o liberacion.
type TokenArrendamientoFlujoFirmaBaremacion struct {
	operarHMAC operacionHMACTokenArrendamientoFlujoFirma
}

// NuevoTokenArrendamientoFlujoFirmaBaremacion crea una capacidad de 256 bits
// mediante el generador criptografico del sistema operativo.
func NuevoTokenArrendamientoFlujoFirmaBaremacion() (TokenArrendamientoFlujoFirmaBaremacion, error) {
	var secreto [tamanoTokenArrendamientoFlujoFirma]byte
	if _, err := io.ReadFull(rand.Reader, secreto[:]); err != nil || bytesTokenArrendamientoNulos(secreto[:]) {
		return TokenArrendamientoFlujoFirmaBaremacion{}, ErrArrendamientoFlujoFirmaInvalido
	}
	operarHMAC := func(clave, huellaEsperada []byte) ([]byte, bool, error) {
		if len(clave) < sha256.Size {
			return nil, false, ErrArrendamientoFlujoFirmaInvalido
		}
		mac := hmac.New(sha256.New, clave)
		_, _ = mac.Write([]byte("bolsa.firma.token-arrendamiento.hmac.v1\x00"))
		_, _ = mac.Write(secreto[:])
		huellaCalculada := mac.Sum(nil)
		coincide := len(huellaEsperada) == sha256.Size && hmac.Equal(huellaCalculada, huellaEsperada)
		return huellaCalculada, coincide, nil
	}
	return TokenArrendamientoFlujoFirmaBaremacion{operarHMAC: operarHMAC}, nil
}

func (t TokenArrendamientoFlujoFirmaBaremacion) Validar() error {
	if t.operarHMAC == nil {
		return ErrArrendamientoFlujoFirmaInvalido
	}
	return nil
}

// HuellaHMACSHA256 calcula el comprobante almacenable de la capacidad. Nunca
// devuelve el token y exige una clave de, al menos, 256 bits.
func (t TokenArrendamientoFlujoFirmaBaremacion) HuellaHMACSHA256(clave []byte) ([]byte, error) {
	if t.Validar() != nil {
		return nil, ErrArrendamientoFlujoFirmaInvalido
	}
	huella, _, err := t.operarHMAC(clave, nil)
	return huella, err
}

// CoincideHuellaHMACSHA256 autentica la capacidad mediante comparacion en
// tiempo constante. Una capacidad nula, una clave invalida o una huella con
// longitud incorrecta fallan cerradas.
func (t TokenArrendamientoFlujoFirmaBaremacion) CoincideHuellaHMACSHA256(
	clave, esperada []byte,
) bool {
	if t.Validar() != nil {
		return false
	}
	_, coincide, err := t.operarHMAC(clave, esperada)
	return err == nil && coincide
}

func bytesTokenArrendamientoNulos(valor []byte) bool {
	acumulado := byte(0)
	for _, elemento := range valor {
		acumulado |= elemento
	}
	return acumulado == 0
}

func (TokenArrendamientoFlujoFirmaBaremacion) String() string {
	return "[TOKEN-ARRENDAMIENTO-FLUJO-FIRMA-REDACTADO]"
}
func (t TokenArrendamientoFlujoFirmaBaremacion) GoString() string { return t.String() }
func (t TokenArrendamientoFlujoFirmaBaremacion) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, t.String())
}
func (t TokenArrendamientoFlujoFirmaBaremacion) LogValue() slog.Value {
	return slog.StringValue(t.String())
}
func (TokenArrendamientoFlujoFirmaBaremacion) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionArrendamientoProhibida
}
func (*TokenArrendamientoFlujoFirmaBaremacion) UnmarshalJSON([]byte) error {
	return ErrSerializacionArrendamientoProhibida
}
func (TokenArrendamientoFlujoFirmaBaremacion) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionArrendamientoProhibida
}
func (*TokenArrendamientoFlujoFirmaBaremacion) UnmarshalText([]byte) error {
	return ErrSerializacionArrendamientoProhibida
}
func (TokenArrendamientoFlujoFirmaBaremacion) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionArrendamientoProhibida
}
func (*TokenArrendamientoFlujoFirmaBaremacion) UnmarshalBinary([]byte) error {
	return ErrSerializacionArrendamientoProhibida
}
func (TokenArrendamientoFlujoFirmaBaremacion) GobEncode() ([]byte, error) {
	return nil, ErrSerializacionArrendamientoProhibida
}
func (*TokenArrendamientoFlujoFirmaBaremacion) GobDecode([]byte) error {
	return ErrSerializacionArrendamientoProhibida
}
func (TokenArrendamientoFlujoFirmaBaremacion) MarshalXML(*xml.Encoder, xml.StartElement) error {
	return ErrSerializacionArrendamientoProhibida
}
func (*TokenArrendamientoFlujoFirmaBaremacion) UnmarshalXML(*xml.Decoder, xml.StartElement) error {
	return ErrSerializacionArrendamientoProhibida
}

type ArrendamientoFlujoFirmaBaremacion struct {
	FlujoRef         string
	PropietarioRef   string
	SecuenciaCercado uint64
	ExpiraEn         time.Time
	Token            TokenArrendamientoFlujoFirmaBaremacion
}

// Validar solo acredita la forma nominal del sobre. La autoridad procede de
// verificar Token contra la huella HMAC que conserva el repositorio.
func (a ArrendamientoFlujoFirmaBaremacion) Validar() error {
	if !referenciaValida(a.FlujoRef, 512) || !referenciaValida(a.PropietarioRef, 512) ||
		a.SecuenciaCercado < 1 || a.ExpiraEn.IsZero() || a.Token.Validar() != nil {
		return ErrArrendamientoFlujoFirmaInvalido
	}
	return nil
}

func (ArrendamientoFlujoFirmaBaremacion) String() string {
	return "[ARRENDAMIENTO-FLUJO-FIRMA-REDACTADO]"
}
func (a ArrendamientoFlujoFirmaBaremacion) GoString() string { return a.String() }
func (a ArrendamientoFlujoFirmaBaremacion) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, a.String())
}
func (a ArrendamientoFlujoFirmaBaremacion) LogValue() slog.Value {
	return slog.StringValue(a.String())
}
func (ArrendamientoFlujoFirmaBaremacion) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionArrendamientoProhibida
}
func (*ArrendamientoFlujoFirmaBaremacion) UnmarshalJSON([]byte) error {
	return ErrSerializacionArrendamientoProhibida
}
func (ArrendamientoFlujoFirmaBaremacion) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionArrendamientoProhibida
}
func (*ArrendamientoFlujoFirmaBaremacion) UnmarshalText([]byte) error {
	return ErrSerializacionArrendamientoProhibida
}
func (ArrendamientoFlujoFirmaBaremacion) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionArrendamientoProhibida
}
func (*ArrendamientoFlujoFirmaBaremacion) UnmarshalBinary([]byte) error {
	return ErrSerializacionArrendamientoProhibida
}
func (ArrendamientoFlujoFirmaBaremacion) GobEncode() ([]byte, error) {
	return nil, ErrSerializacionArrendamientoProhibida
}
func (*ArrendamientoFlujoFirmaBaremacion) GobDecode([]byte) error {
	return ErrSerializacionArrendamientoProhibida
}
func (ArrendamientoFlujoFirmaBaremacion) MarshalXML(*xml.Encoder, xml.StartElement) error {
	return ErrSerializacionArrendamientoProhibida
}
func (*ArrendamientoFlujoFirmaBaremacion) UnmarshalXML(*xml.Decoder, xml.StartElement) error {
	return ErrSerializacionArrendamientoProhibida
}
