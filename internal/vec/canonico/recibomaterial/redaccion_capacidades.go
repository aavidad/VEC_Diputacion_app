package recibomaterial

import (
	"fmt"
	"io"
	"log/slog"
)

const (
	redaccionSolicitudAtestacion = "[SOLICITUD-ATESTACION-MATERIAL-V2-CONFIDENCIAL-REDACTADA]"
	redaccionDatosAtestacion     = "[DATOS-ATESTACION-MATERIAL-V2-CONFIDENCIALES-REDACTADOS]"
	redaccionPerfilPublicado     = "[PERFIL-PUBLICADO-MATERIAL-V2-CANONICO-REDACTADO]"
	redaccionResultadoReferencia = "[RESULTADO-REFERENCIA-MATERIAL-V2-CONFIDENCIAL-REDACTADO]"
)

func escribirRedaccionMaterial(estado fmt.State, valor string) {
	_, _ = io.WriteString(estado, valor)
}

func (DatosSolicitudAtestacion) String() string     { return redaccionSolicitudAtestacion }
func (d DatosSolicitudAtestacion) GoString() string { return d.String() }
func (d DatosSolicitudAtestacion) Format(estado fmt.State, _ rune) {
	escribirRedaccionMaterial(estado, d.String())
}
func (d DatosSolicitudAtestacion) LogValue() slog.Value         { return slog.StringValue(d.String()) }
func (DatosSolicitudAtestacion) MarshalJSON() ([]byte, error)   { return SerializacionProhibida() }
func (*DatosSolicitudAtestacion) UnmarshalJSON([]byte) error    { return DeserializacionProhibida() }
func (DatosSolicitudAtestacion) MarshalText() ([]byte, error)   { return SerializacionProhibida() }
func (*DatosSolicitudAtestacion) UnmarshalText([]byte) error    { return DeserializacionProhibida() }
func (DatosSolicitudAtestacion) MarshalBinary() ([]byte, error) { return SerializacionProhibida() }
func (*DatosSolicitudAtestacion) UnmarshalBinary([]byte) error  { return DeserializacionProhibida() }

func (DatosAtestacion) String() string     { return redaccionDatosAtestacion }
func (d DatosAtestacion) GoString() string { return d.String() }
func (d DatosAtestacion) Format(estado fmt.State, _ rune) {
	escribirRedaccionMaterial(estado, d.String())
}
func (d DatosAtestacion) LogValue() slog.Value         { return slog.StringValue(d.String()) }
func (DatosAtestacion) MarshalJSON() ([]byte, error)   { return SerializacionProhibida() }
func (*DatosAtestacion) UnmarshalJSON([]byte) error    { return DeserializacionProhibida() }
func (DatosAtestacion) MarshalText() ([]byte, error)   { return SerializacionProhibida() }
func (*DatosAtestacion) UnmarshalText([]byte) error    { return DeserializacionProhibida() }
func (DatosAtestacion) MarshalBinary() ([]byte, error) { return SerializacionProhibida() }
func (*DatosAtestacion) UnmarshalBinary([]byte) error  { return DeserializacionProhibida() }

func (DatosPerfilPublicado) String() string     { return redaccionPerfilPublicado }
func (d DatosPerfilPublicado) GoString() string { return d.String() }
func (d DatosPerfilPublicado) Format(estado fmt.State, _ rune) {
	escribirRedaccionMaterial(estado, d.String())
}
func (d DatosPerfilPublicado) LogValue() slog.Value         { return slog.StringValue(d.String()) }
func (DatosPerfilPublicado) MarshalJSON() ([]byte, error)   { return SerializacionProhibida() }
func (*DatosPerfilPublicado) UnmarshalJSON([]byte) error    { return DeserializacionProhibida() }
func (DatosPerfilPublicado) MarshalText() ([]byte, error)   { return SerializacionProhibida() }
func (*DatosPerfilPublicado) UnmarshalText([]byte) error    { return DeserializacionProhibida() }
func (DatosPerfilPublicado) MarshalBinary() ([]byte, error) { return SerializacionProhibida() }
func (*DatosPerfilPublicado) UnmarshalBinary([]byte) error  { return DeserializacionProhibida() }

func (DatosResultadoReferencia) String() string     { return redaccionResultadoReferencia }
func (d DatosResultadoReferencia) GoString() string { return d.String() }
func (d DatosResultadoReferencia) Format(estado fmt.State, _ rune) {
	escribirRedaccionMaterial(estado, d.String())
}
func (d DatosResultadoReferencia) LogValue() slog.Value { return slog.StringValue(d.String()) }
func (DatosResultadoReferencia) MarshalJSON() ([]byte, error) {
	return SerializacionProhibida()
}
func (*DatosResultadoReferencia) UnmarshalJSON([]byte) error {
	return DeserializacionProhibida()
}
func (DatosResultadoReferencia) MarshalText() ([]byte, error) {
	return SerializacionProhibida()
}
func (*DatosResultadoReferencia) UnmarshalText([]byte) error {
	return DeserializacionProhibida()
}
func (DatosResultadoReferencia) MarshalBinary() ([]byte, error) {
	return SerializacionProhibida()
}
func (*DatosResultadoReferencia) UnmarshalBinary([]byte) error {
	return DeserializacionProhibida()
}
