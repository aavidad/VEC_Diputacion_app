package interna

import (
	"fmt"
	"log/slog"
)

const configuracionInternaRedactada = "configuracion_interna_redactada"

// String evita que el formateo accidental revele topologia, rutas de
// certificados o identificadores de confianza. Los campos siguen disponibles
// de forma tipada para la raiz de composicion.
func (Configuracion) String() string {
	return configuracionInternaRedactada
}

// GoString aplica la misma politica al formato de depuracion %#v.
func (Configuracion) GoString() string {
	return configuracionInternaRedactada
}

// Format cubre todos los verbos de fmt y, por extension, log.Printf.
func (Configuracion) Format(estado fmt.State, _ rune) {
	_, _ = estado.Write([]byte(configuracionInternaRedactada))
}

// MarshalJSON impide que una respuesta o evidencia generica serialice los
// campos exportados. La configuracion se consume por su API Go tipada, no como
// documento de intercambio.
func (Configuracion) MarshalJSON() ([]byte, error) {
	return []byte(`"` + configuracionInternaRedactada + `"`), nil
}

// MarshalText protege serializadores que respetan encoding.TextMarshaler.
func (Configuracion) MarshalText() ([]byte, error) {
	return []byte(configuracionInternaRedactada), nil
}

// LogValue evita que slog inspeccione los campos exportados como un grupo.
func (Configuracion) LogValue() slog.Value {
	return slog.StringValue(configuracionInternaRedactada)
}
