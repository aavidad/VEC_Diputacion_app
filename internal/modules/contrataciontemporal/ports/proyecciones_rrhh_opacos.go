package ports

import (
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
)

// bloqueoSerializacionConsultaRRHH impide que capacidades o material
// probatorio crucen accidentalmente una frontera de transporte o registro.
type bloqueoSerializacionConsultaRRHH struct{}

func (bloqueoSerializacionConsultaRRHH) MarshalJSON() ([]byte, error) {
	return nil, ErrMaterialConsultaRRHHSensible
}
func (*bloqueoSerializacionConsultaRRHH) UnmarshalJSON([]byte) error {
	return ErrMaterialConsultaRRHHSensible
}
func (bloqueoSerializacionConsultaRRHH) MarshalText() ([]byte, error) {
	return nil, ErrMaterialConsultaRRHHSensible
}
func (*bloqueoSerializacionConsultaRRHH) UnmarshalText([]byte) error {
	return ErrMaterialConsultaRRHHSensible
}
func (bloqueoSerializacionConsultaRRHH) MarshalBinary() ([]byte, error) {
	return nil, ErrMaterialConsultaRRHHSensible
}
func (*bloqueoSerializacionConsultaRRHH) UnmarshalBinary([]byte) error {
	return ErrMaterialConsultaRRHHSensible
}
func (bloqueoSerializacionConsultaRRHH) GobEncode() ([]byte, error) {
	return nil, ErrMaterialConsultaRRHHSensible
}
func (*bloqueoSerializacionConsultaRRHH) GobDecode([]byte) error {
	return ErrMaterialConsultaRRHHSensible
}
func (bloqueoSerializacionConsultaRRHH) MarshalCBOR() ([]byte, error) {
	return nil, ErrMaterialConsultaRRHHSensible
}
func (*bloqueoSerializacionConsultaRRHH) UnmarshalCBOR([]byte) error {
	return ErrMaterialConsultaRRHHSensible
}
func (bloqueoSerializacionConsultaRRHH) MarshalYAML() (any, error) {
	return nil, ErrMaterialConsultaRRHHSensible
}
func (*bloqueoSerializacionConsultaRRHH) UnmarshalYAML(func(any) error) error {
	return ErrMaterialConsultaRRHHSensible
}
func (bloqueoSerializacionConsultaRRHH) MarshalXML(*xml.Encoder, xml.StartElement) error {
	return ErrMaterialConsultaRRHHSensible
}
func (*bloqueoSerializacionConsultaRRHH) UnmarshalXML(*xml.Decoder, xml.StartElement) error {
	return ErrMaterialConsultaRRHHSensible
}
func (bloqueoSerializacionConsultaRRHH) String() string {
	return "[MATERIAL-CONSULTA-RRHH-OPACO]"
}
func (b bloqueoSerializacionConsultaRRHH) GoString() string { return b.String() }
func (b bloqueoSerializacionConsultaRRHH) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, b.String())
}
func (b bloqueoSerializacionConsultaRRHH) LogValue() slog.Value {
	return slog.StringValue(b.String())
}
