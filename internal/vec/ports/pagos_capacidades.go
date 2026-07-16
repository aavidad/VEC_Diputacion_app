package ports

import (
	"encoding"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
)

var ErrSerializacionTokenReservaCobroProhibida = errors.New(
	"vec: serializacion de token de reserva de cobro prohibida",
)

const (
	dominioHuellaTokenAltaCobro       = "vec:pagos:token-reserva-alta:v1"
	dominioHuellaTokenDevolucionCobro = "vec:pagos:token-reserva-devolucion:v1"
)

// TokenReservaOrdenCobro es una capacidad efimera y nominal para confirmar o
// abandonar exclusivamente una reserva de alta. Nunca se serializa ni revela;
// la huella con separacion de dominio es el unico material persistible. El
// secreto vive solo en un cierre privado e inmutable ligado a dicho dominio.
type TokenReservaOrdenCobro struct{ operar operacionCapacidadReserva }

func NuevoTokenReservaOrdenCobro() (TokenReservaOrdenCobro, error) {
	operar, err := nuevaOperacionCapacidadReserva(dominioHuellaTokenAltaCobro)
	if err != nil {
		return TokenReservaOrdenCobro{}, ErrReservaOrdenCobroInvalida
	}
	return TokenReservaOrdenCobro{operar: operar}, nil
}

func (t TokenReservaOrdenCobro) Valido() bool { return operacionCapacidadReservaValida(t.operar) }

func (t TokenReservaOrdenCobro) HuellaSHA256() (string, error) {
	huella, valida := huellaCapacidadReserva(t.operar)
	if !valida {
		return "", ErrReservaOrdenCobroInvalida
	}
	return huella, nil
}

func (t TokenReservaOrdenCobro) CoincideConHuellaSHA256(huella string) bool {
	return coincideHuellaCapacidadReserva(t.operar, huella)
}

func (TokenReservaOrdenCobro) String() string     { return "[TOKEN-RESERVA-ALTA-COBRO-REDACTADO]" }
func (t TokenReservaOrdenCobro) GoString() string { return t.String() }
func (t TokenReservaOrdenCobro) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, t.String())
}
func (t TokenReservaOrdenCobro) LogValue() slog.Value { return slog.StringValue(t.String()) }
func (TokenReservaOrdenCobro) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionTokenReservaCobroProhibida
}
func (*TokenReservaOrdenCobro) UnmarshalJSON([]byte) error {
	return ErrSerializacionTokenReservaCobroProhibida
}
func (TokenReservaOrdenCobro) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionTokenReservaCobroProhibida
}
func (*TokenReservaOrdenCobro) UnmarshalText([]byte) error {
	return ErrSerializacionTokenReservaCobroProhibida
}
func (TokenReservaOrdenCobro) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionTokenReservaCobroProhibida
}
func (*TokenReservaOrdenCobro) UnmarshalBinary([]byte) error {
	return ErrSerializacionTokenReservaCobroProhibida
}
func (TokenReservaOrdenCobro) MarshalXML(*xml.Encoder, xml.StartElement) error {
	return ErrSerializacionTokenReservaCobroProhibida
}
func (*TokenReservaOrdenCobro) UnmarshalXML(*xml.Decoder, xml.StartElement) error {
	return ErrSerializacionTokenReservaCobroProhibida
}

// TokenReservaDevolucionCobro no es intercambiable con el token de alta. Su
// huella usa otro dominio aun cuando ambos secretos tengan la misma longitud.
// La ligadura se realiza al crear el cierre, no al invocarlo.
type TokenReservaDevolucionCobro struct{ operar operacionCapacidadReserva }

func NuevoTokenReservaDevolucionCobro() (TokenReservaDevolucionCobro, error) {
	operar, err := nuevaOperacionCapacidadReserva(dominioHuellaTokenDevolucionCobro)
	if err != nil {
		return TokenReservaDevolucionCobro{}, ErrReservaOrdenCobroInvalida
	}
	return TokenReservaDevolucionCobro{operar: operar}, nil
}

func (t TokenReservaDevolucionCobro) Valido() bool {
	return operacionCapacidadReservaValida(t.operar)
}

func (t TokenReservaDevolucionCobro) HuellaSHA256() (string, error) {
	huella, valida := huellaCapacidadReserva(t.operar)
	if !valida {
		return "", ErrReservaOrdenCobroInvalida
	}
	return huella, nil
}

func (t TokenReservaDevolucionCobro) CoincideConHuellaSHA256(huella string) bool {
	return coincideHuellaCapacidadReserva(t.operar, huella)
}

func (TokenReservaDevolucionCobro) String() string {
	return "[TOKEN-RESERVA-DEVOLUCION-COBRO-REDACTADO]"
}
func (t TokenReservaDevolucionCobro) GoString() string { return t.String() }
func (t TokenReservaDevolucionCobro) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, t.String())
}
func (t TokenReservaDevolucionCobro) LogValue() slog.Value {
	return slog.StringValue(t.String())
}
func (TokenReservaDevolucionCobro) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionTokenReservaCobroProhibida
}
func (*TokenReservaDevolucionCobro) UnmarshalJSON([]byte) error {
	return ErrSerializacionTokenReservaCobroProhibida
}
func (TokenReservaDevolucionCobro) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionTokenReservaCobroProhibida
}
func (*TokenReservaDevolucionCobro) UnmarshalText([]byte) error {
	return ErrSerializacionTokenReservaCobroProhibida
}
func (TokenReservaDevolucionCobro) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionTokenReservaCobroProhibida
}
func (*TokenReservaDevolucionCobro) UnmarshalBinary([]byte) error {
	return ErrSerializacionTokenReservaCobroProhibida
}
func (TokenReservaDevolucionCobro) MarshalXML(*xml.Encoder, xml.StartElement) error {
	return ErrSerializacionTokenReservaCobroProhibida
}
func (*TokenReservaDevolucionCobro) UnmarshalXML(*xml.Decoder, xml.StartElement) error {
	return ErrSerializacionTokenReservaCobroProhibida
}

var (
	_ fmt.Formatter              = TokenReservaOrdenCobro{}
	_ fmt.GoStringer             = TokenReservaOrdenCobro{}
	_ fmt.Stringer               = TokenReservaOrdenCobro{}
	_ slog.LogValuer             = TokenReservaOrdenCobro{}
	_ json.Marshaler             = TokenReservaOrdenCobro{}
	_ json.Unmarshaler           = (*TokenReservaOrdenCobro)(nil)
	_ encoding.TextMarshaler     = TokenReservaOrdenCobro{}
	_ encoding.TextUnmarshaler   = (*TokenReservaOrdenCobro)(nil)
	_ encoding.BinaryMarshaler   = TokenReservaOrdenCobro{}
	_ encoding.BinaryUnmarshaler = (*TokenReservaOrdenCobro)(nil)
	_ fmt.Formatter              = TokenReservaDevolucionCobro{}
	_ fmt.GoStringer             = TokenReservaDevolucionCobro{}
	_ fmt.Stringer               = TokenReservaDevolucionCobro{}
	_ slog.LogValuer             = TokenReservaDevolucionCobro{}
	_ json.Marshaler             = TokenReservaDevolucionCobro{}
	_ json.Unmarshaler           = (*TokenReservaDevolucionCobro)(nil)
	_ encoding.TextMarshaler     = TokenReservaDevolucionCobro{}
	_ encoding.TextUnmarshaler   = (*TokenReservaDevolucionCobro)(nil)
	_ encoding.BinaryMarshaler   = TokenReservaDevolucionCobro{}
	_ encoding.BinaryUnmarshaler = (*TokenReservaDevolucionCobro)(nil)
)
