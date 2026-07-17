package calculoexperienciaoficial

import (
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
)

type bloqueoSerializacion struct{}

func (bloqueoSerializacion) String() string     { return "[CAPACIDAD-CALCULO-OFICIAL-OPACA]" }
func (b bloqueoSerializacion) GoString() string { return b.String() }
func (b bloqueoSerializacion) Format(estado fmt.State, _ rune) {
	escribirOpaco(estado, b.String())
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
func (bloqueoSerializacion) GobEncode() ([]byte, error) {
	return nil, ErrSerializacionProhibida
}
func (*bloqueoSerializacion) GobDecode([]byte) error {
	return ErrSerializacionProhibida
}
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

func escribirOpaco(estado fmt.State, etiqueta string) {
	_, _ = io.WriteString(estado, etiqueta)
}

func (OrdenCalculoExperienciaOficial) String() string {
	return "[ORDEN-CALCULO-EXPERIENCIA-OFICIAL-OPACA]"
}
func (o OrdenCalculoExperienciaOficial) GoString() string { return o.String() }
func (o OrdenCalculoExperienciaOficial) Format(estado fmt.State, _ rune) {
	escribirOpaco(estado, o.String())
}
func (o OrdenCalculoExperienciaOficial) LogValue() slog.Value {
	return slog.StringValue(o.String())
}
func (OrdenCalculoExperienciaOficial) MarshalJSON() ([]byte, error) {
	return bloqueoSerializacion{}.MarshalJSON()
}
func (*OrdenCalculoExperienciaOficial) UnmarshalJSON(datos []byte) error {
	return (&bloqueoSerializacion{}).UnmarshalJSON(datos)
}

func (ResultadoEjecucion) String() string {
	return "[RESULTADO-CALCULO-EXPERIENCIA-OFICIAL-OPACO]"
}
func (r ResultadoEjecucion) GoString() string { return r.String() }
func (r ResultadoEjecucion) Format(estado fmt.State, _ rune) {
	escribirOpaco(estado, r.String())
}
func (r ResultadoEjecucion) LogValue() slog.Value { return slog.StringValue(r.String()) }
func (ResultadoEjecucion) MarshalJSON() ([]byte, error) {
	return bloqueoSerializacion{}.MarshalJSON()
}
func (*ResultadoEjecucion) UnmarshalJSON(datos []byte) error {
	return (&bloqueoSerializacion{}).UnmarshalJSON(datos)
}

func (SolicitudConfirmacionDuradera) String() string {
	return "[SOLICITUD-CONFIRMACION-CALCULO-OFICIAL-OPACA]"
}
func (s SolicitudConfirmacionDuradera) GoString() string { return s.String() }
func (s SolicitudConfirmacionDuradera) Format(estado fmt.State, _ rune) {
	escribirOpaco(estado, s.String())
}
func (s SolicitudConfirmacionDuradera) LogValue() slog.Value {
	return slog.StringValue(s.String())
}
func (SolicitudConfirmacionDuradera) MarshalJSON() ([]byte, error) {
	return bloqueoSerializacion{}.MarshalJSON()
}
func (*SolicitudConfirmacionDuradera) UnmarshalJSON(datos []byte) error {
	return (&bloqueoSerializacion{}).UnmarshalJSON(datos)
}

func (ResultadoConfirmacionDuradera) String() string {
	return "[RESULTADO-CONFIRMACION-CALCULO-OFICIAL-OPACO]"
}
func (r ResultadoConfirmacionDuradera) GoString() string { return r.String() }
func (r ResultadoConfirmacionDuradera) Format(estado fmt.State, _ rune) {
	escribirOpaco(estado, r.String())
}
func (r ResultadoConfirmacionDuradera) LogValue() slog.Value {
	return slog.StringValue(r.String())
}

func (SolicitudReconciliacionDuradera) String() string {
	return "[SOLICITUD-RECONCILIACION-CALCULO-OFICIAL-OPACA]"
}
func (s SolicitudReconciliacionDuradera) GoString() string { return s.String() }
func (s SolicitudReconciliacionDuradera) Format(estado fmt.State, _ rune) {
	escribirOpaco(estado, s.String())
}
func (s SolicitudReconciliacionDuradera) LogValue() slog.Value {
	return slog.StringValue(s.String())
}
