package confianzaatestacion

import (
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
)

// bloqueoSerializacion impide tratar configuracion, claves o pruebas como DTO.
// Cada adaptador durable debe persistir un esquema explicito y validado.
type bloqueoSerializacion struct{}

func (bloqueoSerializacion) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionConfianzaAtestacionV2Prohibida
}

func (*bloqueoSerializacion) UnmarshalJSON([]byte) error {
	return ErrSerializacionConfianzaAtestacionV2Prohibida
}

func (bloqueoSerializacion) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionConfianzaAtestacionV2Prohibida
}

func (*bloqueoSerializacion) UnmarshalText([]byte) error {
	return ErrSerializacionConfianzaAtestacionV2Prohibida
}

func (bloqueoSerializacion) MarshalXML(*xml.Encoder, xml.StartElement) error {
	return ErrSerializacionConfianzaAtestacionV2Prohibida
}

func (*bloqueoSerializacion) UnmarshalXML(*xml.Decoder, xml.StartElement) error {
	return ErrSerializacionConfianzaAtestacionV2Prohibida
}

func (bloqueoSerializacion) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionConfianzaAtestacionV2Prohibida
}

func (*bloqueoSerializacion) UnmarshalBinary([]byte) error {
	return ErrSerializacionConfianzaAtestacionV2Prohibida
}

func (bloqueoSerializacion) GobEncode() ([]byte, error) {
	return nil, ErrSerializacionConfianzaAtestacionV2Prohibida
}

func (*bloqueoSerializacion) GobDecode([]byte) error {
	return ErrSerializacionConfianzaAtestacionV2Prohibida
}

func (bloqueoSerializacion) MarshalCBOR() ([]byte, error) {
	return nil, ErrSerializacionConfianzaAtestacionV2Prohibida
}

func (*bloqueoSerializacion) UnmarshalCBOR([]byte) error {
	return ErrSerializacionConfianzaAtestacionV2Prohibida
}

func (bloqueoSerializacion) MarshalYAML() (any, error) {
	return nil, ErrSerializacionConfianzaAtestacionV2Prohibida
}

func (*bloqueoSerializacion) UnmarshalYAML(func(any) error) error {
	return ErrSerializacionConfianzaAtestacionV2Prohibida
}

func (bloqueoSerializacion) String() string {
	return "[CONFIANZA-ATESTACION-AUTORIZACION-V2-REDACTADA]"
}

func (b bloqueoSerializacion) GoString() string { return b.String() }

func (b bloqueoSerializacion) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, b.String())
}

func (b bloqueoSerializacion) LogValue() slog.Value {
	return slog.StringValue(b.String())
}
