package confianzaatestacion

import (
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
)

type bloqueoSerializacionV3 struct{}

func (bloqueoSerializacionV3) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionConfianzaAtestacionV3Prohibida
}
func (*bloqueoSerializacionV3) UnmarshalJSON([]byte) error {
	return ErrSerializacionConfianzaAtestacionV3Prohibida
}
func (bloqueoSerializacionV3) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionConfianzaAtestacionV3Prohibida
}
func (*bloqueoSerializacionV3) UnmarshalText([]byte) error {
	return ErrSerializacionConfianzaAtestacionV3Prohibida
}
func (bloqueoSerializacionV3) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionConfianzaAtestacionV3Prohibida
}
func (*bloqueoSerializacionV3) UnmarshalBinary([]byte) error {
	return ErrSerializacionConfianzaAtestacionV3Prohibida
}
func (bloqueoSerializacionV3) GobEncode() ([]byte, error) {
	return nil, ErrSerializacionConfianzaAtestacionV3Prohibida
}
func (*bloqueoSerializacionV3) GobDecode([]byte) error {
	return ErrSerializacionConfianzaAtestacionV3Prohibida
}
func (bloqueoSerializacionV3) MarshalCBOR() ([]byte, error) {
	return nil, ErrSerializacionConfianzaAtestacionV3Prohibida
}
func (*bloqueoSerializacionV3) UnmarshalCBOR([]byte) error {
	return ErrSerializacionConfianzaAtestacionV3Prohibida
}
func (bloqueoSerializacionV3) MarshalYAML() (any, error) {
	return nil, ErrSerializacionConfianzaAtestacionV3Prohibida
}
func (*bloqueoSerializacionV3) UnmarshalYAML(func(any) error) error {
	return ErrSerializacionConfianzaAtestacionV3Prohibida
}
func (bloqueoSerializacionV3) MarshalXML(*xml.Encoder, xml.StartElement) error {
	return ErrSerializacionConfianzaAtestacionV3Prohibida
}
func (*bloqueoSerializacionV3) UnmarshalXML(*xml.Decoder, xml.StartElement) error {
	return ErrSerializacionConfianzaAtestacionV3Prohibida
}
func (bloqueoSerializacionV3) String() string {
	return "[CONFIANZA-ATESTACION-AUTORIZACION-V3-REDACTADA]"
}
func (b bloqueoSerializacionV3) GoString() string { return b.String() }
func (b bloqueoSerializacionV3) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, b.String())
}
func (b bloqueoSerializacionV3) LogValue() slog.Value {
	return slog.StringValue(b.String())
}
