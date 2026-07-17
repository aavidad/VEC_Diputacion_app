package gobiernoreglasbaremo

import (
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
)

type bloqueoSerializacion struct{}

func (bloqueoSerializacion) String() string     { return "[GOBIERNO-REGLAS-BAREMO-OPACO]" }
func (b bloqueoSerializacion) GoString() string { return b.String() }
func (b bloqueoSerializacion) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, b.String())
}
func (b bloqueoSerializacion) LogValue() slog.Value { return slog.StringValue(b.String()) }
func (bloqueoSerializacion) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionProhibida
}
func (*bloqueoSerializacion) UnmarshalJSON([]byte) error {
	return ErrSerializacionProhibida
}
func (bloqueoSerializacion) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionProhibida
}
func (*bloqueoSerializacion) UnmarshalText([]byte) error {
	return ErrSerializacionProhibida
}
func (bloqueoSerializacion) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionProhibida
}
func (*bloqueoSerializacion) UnmarshalBinary([]byte) error {
	return ErrSerializacionProhibida
}
func (bloqueoSerializacion) GobEncode() ([]byte, error) { return nil, ErrSerializacionProhibida }
func (*bloqueoSerializacion) GobDecode([]byte) error    { return ErrSerializacionProhibida }
func (bloqueoSerializacion) MarshalCBOR() ([]byte, error) {
	return nil, ErrSerializacionProhibida
}
func (*bloqueoSerializacion) UnmarshalCBOR([]byte) error {
	return ErrSerializacionProhibida
}
func (bloqueoSerializacion) MarshalYAML() (any, error) {
	return nil, ErrSerializacionProhibida
}
func (*bloqueoSerializacion) UnmarshalYAML(func(any) error) error {
	return ErrSerializacionProhibida
}
func (bloqueoSerializacion) MarshalXML(*xml.Encoder, xml.StartElement) error {
	return ErrSerializacionProhibida
}
func (*bloqueoSerializacion) UnmarshalXML(*xml.Decoder, xml.StartElement) error {
	return ErrSerializacionProhibida
}
