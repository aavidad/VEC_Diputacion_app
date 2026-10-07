package reglasbaremo

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"unicode/utf8"
)

// CanonicalizarConjuntoReglasBaremoJSON construye un borrador de simulación
// desde el contrato V1 sin exigir su orden de campos o espacios original.
// No aprueba ni activa reglas. La restauración histórica mantiene sus guardas.
func CanonicalizarConjuntoReglasBaremoJSON(contenido []byte) ([]byte, error) {
	if len(contenido) == 0 || len(contenido) > maximoBytesRepresentacion {
		return nil, nuevoError("representacion_canonica", CodigoFueraDeLimites)
	}
	if !utf8.Valid(contenido) {
		return nil, nuevoError("representacion_canonica", CodigoValorNoCanonico)
	}
	tokens := json.NewDecoder(bytes.NewReader(contenido))
	tokens.UseNumber()
	if err := validarTokensEdicion(tokens, 0); err != nil {
		return nil, err
	}
	if _, err := tokens.Token(); !errors.Is(err, io.EOF) {
		return nil, nuevoError("representacion_canonica", CodigoValorNoCanonico)
	}
	var material materialConjunto
	decoder := json.NewDecoder(bytes.NewReader(contenido))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&material); err != nil {
		return nil, nuevoError("representacion_canonica", CodigoValorNoCanonico)
	}
	if material.Esquema != esquemaConjuntoReglasBaremo {
		return nil, nuevoError("esquema", CodigoEsquemaIncompatible)
	}
	conjunto, err := reconstruirConjunto(material)
	if err != nil {
		return nil, err
	}
	return conjunto.RepresentacionCanonica()
}

func validarTokensEdicion(decoder *json.Decoder, profundidad int) error {
	if profundidad > 32 {
		return nuevoError("representacion_canonica", CodigoFueraDeLimites)
	}
	token, err := decoder.Token()
	if err != nil {
		return nuevoError("representacion_canonica", CodigoValorNoCanonico)
	}
	delimiter, compuesto := token.(json.Delim)
	if !compuesto {
		return nil
	}
	switch delimiter {
	case '{':
		vistas := make(map[string]bool)
		for decoder.More() {
			key, err := decoder.Token()
			if err != nil {
				return nuevoError("representacion_canonica", CodigoValorNoCanonico)
			}
			clave, ok := key.(string)
			if !ok || !claveJSONEdicionASCII(clave) || vistas[clave] {
				return nuevoError("representacion_canonica", CodigoValorDuplicado)
			}
			vistas[clave] = true
			if err := validarTokensEdicion(decoder, profundidad+1); err != nil {
				return err
			}
		}
	case '[':
		for decoder.More() {
			if err := validarTokensEdicion(decoder, profundidad+1); err != nil {
				return err
			}
		}
	default:
		return nuevoError("representacion_canonica", CodigoValorNoCanonico)
	}
	final, err := decoder.Token()
	if err != nil || delimiter == '{' && final != json.Delim('}') || delimiter == '[' && final != json.Delim(']') {
		return nuevoError("representacion_canonica", CodigoValorNoCanonico)
	}
	return nil
}

// encoding/json compara nombres sin distinguir mayúsculas e incluye alias
// Unicode. El contrato solo contiene claves ASCII minúsculas y guiones bajos.
func claveJSONEdicionASCII(clave string) bool {
	if len(clave) == 0 {
		return false
	}
	for _, c := range clave {
		if c != '_' && (c < 'a' || c > 'z') && (c < '0' || c > '9') {
			return false
		}
	}
	return true
}
