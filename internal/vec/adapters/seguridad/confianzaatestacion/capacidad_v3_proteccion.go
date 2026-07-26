package confianzaatestacion

import (
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
)

type bloqueoSerializacionCapacidadV3 struct{}

func (bloqueoSerializacionCapacidadV3) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionCapacidadAtestacionV3Prohibida
}
func (*bloqueoSerializacionCapacidadV3) UnmarshalJSON([]byte) error {
	return ErrSerializacionCapacidadAtestacionV3Prohibida
}
func (bloqueoSerializacionCapacidadV3) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionCapacidadAtestacionV3Prohibida
}
func (*bloqueoSerializacionCapacidadV3) UnmarshalText([]byte) error {
	return ErrSerializacionCapacidadAtestacionV3Prohibida
}
func (bloqueoSerializacionCapacidadV3) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionCapacidadAtestacionV3Prohibida
}
func (*bloqueoSerializacionCapacidadV3) UnmarshalBinary([]byte) error {
	return ErrSerializacionCapacidadAtestacionV3Prohibida
}
func (bloqueoSerializacionCapacidadV3) GobEncode() ([]byte, error) {
	return nil, ErrSerializacionCapacidadAtestacionV3Prohibida
}
func (*bloqueoSerializacionCapacidadV3) GobDecode([]byte) error {
	return ErrSerializacionCapacidadAtestacionV3Prohibida
}
func (bloqueoSerializacionCapacidadV3) MarshalCBOR() ([]byte, error) {
	return nil, ErrSerializacionCapacidadAtestacionV3Prohibida
}
func (*bloqueoSerializacionCapacidadV3) UnmarshalCBOR([]byte) error {
	return ErrSerializacionCapacidadAtestacionV3Prohibida
}
func (bloqueoSerializacionCapacidadV3) MarshalYAML() (any, error) {
	return nil, ErrSerializacionCapacidadAtestacionV3Prohibida
}
func (*bloqueoSerializacionCapacidadV3) UnmarshalYAML(func(any) error) error {
	return ErrSerializacionCapacidadAtestacionV3Prohibida
}
func (bloqueoSerializacionCapacidadV3) MarshalXML(*xml.Encoder, xml.StartElement) error {
	return ErrSerializacionCapacidadAtestacionV3Prohibida
}
func (*bloqueoSerializacionCapacidadV3) UnmarshalXML(*xml.Decoder, xml.StartElement) error {
	return ErrSerializacionCapacidadAtestacionV3Prohibida
}
func (bloqueoSerializacionCapacidadV3) String() string {
	return "[CAPACIDAD-ATESTACION-AUTORIZACION-V3-REDACTADA]"
}
func (b bloqueoSerializacionCapacidadV3) GoString() string { return b.String() }
func (b bloqueoSerializacionCapacidadV3) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, b.String())
}
func (b bloqueoSerializacionCapacidadV3) LogValue() slog.Value {
	return slog.StringValue(b.String())
}
