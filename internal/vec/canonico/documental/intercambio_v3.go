package documental

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
)

var (
	ErrOrdenDespachoDocumentalV3Invalida  = errors.New("vec: orden de despacho documental v3 invalida")
	ErrTokenCercadoDocumentalV3Invalido   = errors.New("vec: token de cercado documental v3 invalido")
	ErrSelloEvidenciaDocumentalV3Invalido = errors.New("vec: sello de evidencia documental v3 invalido")
	ErrReconciliacionDocumentalV3Invalida = errors.New("vec: reconciliacion documental v3 invalida")
	ErrSerializacionSecretoDocumentalV3   = errors.New("vec: serializacion de secreto documental v3 prohibida")
)

func (d DatosSolicitudReclamacionV3) Validar() error {
	if !d.EsValida() {
		return ErrOrdenDespachoDocumentalV3Invalida
	}
	return nil
}

func (DatosSolicitudReclamacionV3) String() string {
	return "[SOLICITUD-RECLAMAR-DESPACHO-DOCUMENTAL-V3-REDACTADA]"
}
func (d DatosSolicitudReclamacionV3) GoString() string { return d.String() }
func (d DatosSolicitudReclamacionV3) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, d.String())
}
func (d DatosSolicitudReclamacionV3) LogValue() slog.Value {
	return slog.StringValue(d.String())
}
func (DatosSolicitudReclamacionV3) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*DatosSolicitudReclamacionV3) UnmarshalJSON([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (DatosSolicitudReclamacionV3) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*DatosSolicitudReclamacionV3) UnmarshalText([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (DatosSolicitudReclamacionV3) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*DatosSolicitudReclamacionV3) UnmarshalBinary([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}

func (VinculosMaterialDespachoV3) String() string {
	return "[VINCULOS-CRUDOS-VERIFICACION-DESPACHO-V3-NOMINALES-NO-AUTORITATIVOS-REDACTADOS]"
}
func (v VinculosMaterialDespachoV3) GoString() string { return v.String() }
func (v VinculosMaterialDespachoV3) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, v.String())
}
func (v VinculosMaterialDespachoV3) LogValue() slog.Value {
	return slog.StringValue(v.String())
}
func (VinculosMaterialDespachoV3) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*VinculosMaterialDespachoV3) UnmarshalJSON([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (VinculosMaterialDespachoV3) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*VinculosMaterialDespachoV3) UnmarshalText([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (VinculosMaterialDespachoV3) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*VinculosMaterialDespachoV3) UnmarshalBinary([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
