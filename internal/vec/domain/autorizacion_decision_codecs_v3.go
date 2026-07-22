package domain

import (
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
)

type bloqueoSerializacionEvidenciaEvaluacionAutorizacionV3 struct{}

func (bloqueoSerializacionEvidenciaEvaluacionAutorizacionV3) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionEvidenciaEvaluacionAutorizacionV3Prohibida
}
func (*bloqueoSerializacionEvidenciaEvaluacionAutorizacionV3) UnmarshalJSON([]byte) error {
	return ErrSerializacionEvidenciaEvaluacionAutorizacionV3Prohibida
}
func (bloqueoSerializacionEvidenciaEvaluacionAutorizacionV3) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionEvidenciaEvaluacionAutorizacionV3Prohibida
}
func (*bloqueoSerializacionEvidenciaEvaluacionAutorizacionV3) UnmarshalText([]byte) error {
	return ErrSerializacionEvidenciaEvaluacionAutorizacionV3Prohibida
}
func (bloqueoSerializacionEvidenciaEvaluacionAutorizacionV3) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionEvidenciaEvaluacionAutorizacionV3Prohibida
}
func (*bloqueoSerializacionEvidenciaEvaluacionAutorizacionV3) UnmarshalBinary([]byte) error {
	return ErrSerializacionEvidenciaEvaluacionAutorizacionV3Prohibida
}
func (bloqueoSerializacionEvidenciaEvaluacionAutorizacionV3) GobEncode() ([]byte, error) {
	return nil, ErrSerializacionEvidenciaEvaluacionAutorizacionV3Prohibida
}
func (*bloqueoSerializacionEvidenciaEvaluacionAutorizacionV3) GobDecode([]byte) error {
	return ErrSerializacionEvidenciaEvaluacionAutorizacionV3Prohibida
}
func (bloqueoSerializacionEvidenciaEvaluacionAutorizacionV3) MarshalCBOR() ([]byte, error) {
	return nil, ErrSerializacionEvidenciaEvaluacionAutorizacionV3Prohibida
}
func (*bloqueoSerializacionEvidenciaEvaluacionAutorizacionV3) UnmarshalCBOR([]byte) error {
	return ErrSerializacionEvidenciaEvaluacionAutorizacionV3Prohibida
}
func (bloqueoSerializacionEvidenciaEvaluacionAutorizacionV3) MarshalYAML() (any, error) {
	return nil, ErrSerializacionEvidenciaEvaluacionAutorizacionV3Prohibida
}
func (*bloqueoSerializacionEvidenciaEvaluacionAutorizacionV3) UnmarshalYAML(func(any) error) error {
	return ErrSerializacionEvidenciaEvaluacionAutorizacionV3Prohibida
}
func (bloqueoSerializacionEvidenciaEvaluacionAutorizacionV3) MarshalXML(*xml.Encoder, xml.StartElement) error {
	return ErrSerializacionEvidenciaEvaluacionAutorizacionV3Prohibida
}
func (*bloqueoSerializacionEvidenciaEvaluacionAutorizacionV3) UnmarshalXML(*xml.Decoder, xml.StartElement) error {
	return ErrSerializacionEvidenciaEvaluacionAutorizacionV3Prohibida
}
func (bloqueoSerializacionEvidenciaEvaluacionAutorizacionV3) String() string {
	return "[EVIDENCIA-EVALUACION-AUTORIZACION-V3-OPACA]"
}
func (b bloqueoSerializacionEvidenciaEvaluacionAutorizacionV3) GoString() string { return b.String() }
func (b bloqueoSerializacionEvidenciaEvaluacionAutorizacionV3) Format(s fmt.State, _ rune) {
	_, _ = io.WriteString(s, b.String())
}
func (b bloqueoSerializacionEvidenciaEvaluacionAutorizacionV3) LogValue() slog.Value {
	return slog.StringValue(b.String())
}

type bloqueoSerializacionDecisionAutorizacionLigadaV3 struct{}

func (bloqueoSerializacionDecisionAutorizacionLigadaV3) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionDecisionAutorizacionLigadaV3Prohibida
}
func (*bloqueoSerializacionDecisionAutorizacionLigadaV3) UnmarshalJSON([]byte) error {
	return ErrSerializacionDecisionAutorizacionLigadaV3Prohibida
}
func (bloqueoSerializacionDecisionAutorizacionLigadaV3) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionDecisionAutorizacionLigadaV3Prohibida
}
func (*bloqueoSerializacionDecisionAutorizacionLigadaV3) UnmarshalText([]byte) error {
	return ErrSerializacionDecisionAutorizacionLigadaV3Prohibida
}
func (bloqueoSerializacionDecisionAutorizacionLigadaV3) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionDecisionAutorizacionLigadaV3Prohibida
}
func (*bloqueoSerializacionDecisionAutorizacionLigadaV3) UnmarshalBinary([]byte) error {
	return ErrSerializacionDecisionAutorizacionLigadaV3Prohibida
}
func (bloqueoSerializacionDecisionAutorizacionLigadaV3) GobEncode() ([]byte, error) {
	return nil, ErrSerializacionDecisionAutorizacionLigadaV3Prohibida
}
func (*bloqueoSerializacionDecisionAutorizacionLigadaV3) GobDecode([]byte) error {
	return ErrSerializacionDecisionAutorizacionLigadaV3Prohibida
}
func (bloqueoSerializacionDecisionAutorizacionLigadaV3) MarshalCBOR() ([]byte, error) {
	return nil, ErrSerializacionDecisionAutorizacionLigadaV3Prohibida
}
func (*bloqueoSerializacionDecisionAutorizacionLigadaV3) UnmarshalCBOR([]byte) error {
	return ErrSerializacionDecisionAutorizacionLigadaV3Prohibida
}
func (bloqueoSerializacionDecisionAutorizacionLigadaV3) MarshalYAML() (any, error) {
	return nil, ErrSerializacionDecisionAutorizacionLigadaV3Prohibida
}
func (*bloqueoSerializacionDecisionAutorizacionLigadaV3) UnmarshalYAML(func(any) error) error {
	return ErrSerializacionDecisionAutorizacionLigadaV3Prohibida
}
func (bloqueoSerializacionDecisionAutorizacionLigadaV3) MarshalXML(*xml.Encoder, xml.StartElement) error {
	return ErrSerializacionDecisionAutorizacionLigadaV3Prohibida
}
func (*bloqueoSerializacionDecisionAutorizacionLigadaV3) UnmarshalXML(*xml.Decoder, xml.StartElement) error {
	return ErrSerializacionDecisionAutorizacionLigadaV3Prohibida
}
func (bloqueoSerializacionDecisionAutorizacionLigadaV3) String() string {
	return "[DECISION-AUTORIZACION-LIGADA-V3-OPACA]"
}
func (b bloqueoSerializacionDecisionAutorizacionLigadaV3) GoString() string { return b.String() }
func (b bloqueoSerializacionDecisionAutorizacionLigadaV3) Format(s fmt.State, _ rune) {
	_, _ = io.WriteString(s, b.String())
}
func (b bloqueoSerializacionDecisionAutorizacionLigadaV3) LogValue() slog.Value {
	return slog.StringValue(b.String())
}
