package httpseguridad

import (
	"errors"
	"fmt"
	"log/slog"
)

var ErrDatosSesionNoSerializables = errors.New("los datos autoritativos de sesion no se reconstruyen desde una serializacion")

const (
	altaSesionRedactada         = "[ALTA DE SESION CONFIDENCIAL]"
	confirmacionSesionRedactada = "[CONFIRMACION DE SESION CONFIDENCIAL]"
	consultaSesionRedactada     = "[CONSULTA DE SESION CONFIDENCIAL]"
)

func (AltaSesionAtomica) String() string   { return altaSesionRedactada }
func (AltaSesionAtomica) GoString() string { return altaSesionRedactada }
func (AltaSesionAtomica) Format(estado fmt.State, _ rune) {
	_, _ = estado.Write([]byte(altaSesionRedactada))
}
func (AltaSesionAtomica) LogValue() slog.Value { return slog.StringValue(altaSesionRedactada) }
func (AltaSesionAtomica) MarshalJSON() ([]byte, error) {
	return []byte(`{"alta_sesion":"[CONFIDENCIAL]"}`), nil
}
func (*AltaSesionAtomica) UnmarshalJSON([]byte) error  { return ErrDatosSesionNoSerializables }
func (AltaSesionAtomica) MarshalText() ([]byte, error) { return []byte(altaSesionRedactada), nil }
func (*AltaSesionAtomica) UnmarshalText([]byte) error  { return ErrDatosSesionNoSerializables }
func (AltaSesionAtomica) MarshalBinary() ([]byte, error) {
	return []byte(altaSesionRedactada), nil
}
func (*AltaSesionAtomica) UnmarshalBinary([]byte) error { return ErrDatosSesionNoSerializables }
func (AltaSesionAtomica) GobEncode() ([]byte, error)    { return []byte(altaSesionRedactada), nil }
func (*AltaSesionAtomica) GobDecode([]byte) error       { return ErrDatosSesionNoSerializables }

func (ConfirmacionAltaSesion) String() string   { return confirmacionSesionRedactada }
func (ConfirmacionAltaSesion) GoString() string { return confirmacionSesionRedactada }
func (ConfirmacionAltaSesion) Format(estado fmt.State, _ rune) {
	_, _ = estado.Write([]byte(confirmacionSesionRedactada))
}
func (ConfirmacionAltaSesion) LogValue() slog.Value {
	return slog.StringValue(confirmacionSesionRedactada)
}
func (ConfirmacionAltaSesion) MarshalJSON() ([]byte, error) {
	return []byte(`{"confirmacion_sesion":"[CONFIDENCIAL]"}`), nil
}
func (*ConfirmacionAltaSesion) UnmarshalJSON([]byte) error { return ErrDatosSesionNoSerializables }
func (ConfirmacionAltaSesion) MarshalText() ([]byte, error) {
	return []byte(confirmacionSesionRedactada), nil
}
func (*ConfirmacionAltaSesion) UnmarshalText([]byte) error { return ErrDatosSesionNoSerializables }
func (ConfirmacionAltaSesion) MarshalBinary() ([]byte, error) {
	return []byte(confirmacionSesionRedactada), nil
}
func (*ConfirmacionAltaSesion) UnmarshalBinary([]byte) error { return ErrDatosSesionNoSerializables }
func (ConfirmacionAltaSesion) GobEncode() ([]byte, error) {
	return []byte(confirmacionSesionRedactada), nil
}
func (*ConfirmacionAltaSesion) GobDecode([]byte) error { return ErrDatosSesionNoSerializables }

func (ConsultaSesionActiva) String() string   { return consultaSesionRedactada }
func (ConsultaSesionActiva) GoString() string { return consultaSesionRedactada }
func (ConsultaSesionActiva) Format(estado fmt.State, _ rune) {
	_, _ = estado.Write([]byte(consultaSesionRedactada))
}
func (ConsultaSesionActiva) LogValue() slog.Value { return slog.StringValue(consultaSesionRedactada) }
func (ConsultaSesionActiva) MarshalJSON() ([]byte, error) {
	return []byte(`{"consulta_sesion":"[CONFIDENCIAL]"}`), nil
}
func (*ConsultaSesionActiva) UnmarshalJSON([]byte) error { return ErrDatosSesionNoSerializables }
func (ConsultaSesionActiva) MarshalText() ([]byte, error) {
	return []byte(consultaSesionRedactada), nil
}
func (*ConsultaSesionActiva) UnmarshalText([]byte) error { return ErrDatosSesionNoSerializables }
func (ConsultaSesionActiva) MarshalBinary() ([]byte, error) {
	return []byte(consultaSesionRedactada), nil
}
func (*ConsultaSesionActiva) UnmarshalBinary([]byte) error { return ErrDatosSesionNoSerializables }
func (ConsultaSesionActiva) GobEncode() ([]byte, error) {
	return []byte(consultaSesionRedactada), nil
}
func (*ConsultaSesionActiva) GobDecode([]byte) error { return ErrDatosSesionNoSerializables }
