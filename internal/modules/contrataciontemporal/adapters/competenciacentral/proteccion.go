package competenciacentral

import (
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

// La acreditación completa no atraviesa un canal ni un log. La proyección
// explícita Proyeccion se entrega únicamente al consumidor autorizado de CT.
type proteccionAcreditacion struct{}

func (proteccionAcreditacion) MarshalJSON() ([]byte, error) {
	return nil, vecdomain.ErrSerializacionAsignacionCompetencialV1Prohibida
}
func (*proteccionAcreditacion) UnmarshalJSON([]byte) error {
	return vecdomain.ErrSerializacionAsignacionCompetencialV1Prohibida
}
func (proteccionAcreditacion) MarshalText() ([]byte, error) {
	return nil, vecdomain.ErrSerializacionAsignacionCompetencialV1Prohibida
}
func (*proteccionAcreditacion) UnmarshalText([]byte) error {
	return vecdomain.ErrSerializacionAsignacionCompetencialV1Prohibida
}
func (proteccionAcreditacion) MarshalXML(*xml.Encoder, xml.StartElement) error {
	return vecdomain.ErrSerializacionAsignacionCompetencialV1Prohibida
}
func (*proteccionAcreditacion) UnmarshalXML(*xml.Decoder, xml.StartElement) error {
	return vecdomain.ErrSerializacionAsignacionCompetencialV1Prohibida
}
func (proteccionAcreditacion) String() string               { return "[ACREDITACION-COMPETENCIA-INTERNA]" }
func (p proteccionAcreditacion) GoString() string           { return p.String() }
func (p proteccionAcreditacion) Format(w fmt.State, _ rune) { _, _ = io.WriteString(w, p.String()) }
func (p proteccionAcreditacion) LogValue() slog.Value       { return slog.StringValue(p.String()) }
