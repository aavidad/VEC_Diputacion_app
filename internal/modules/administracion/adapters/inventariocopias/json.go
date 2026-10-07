package inventariocopias

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strings"

	"vec-diputacion-granada/internal/modules/administracion/domain/copias"
)

const MaxDocumentoBytes = 4 << 20

var ErrDocumento = errors.New("inventario_documento_no_valido")

func LeerDescriptor(origen io.Reader) (Descriptor, error) {
	return leerJSON[Descriptor](origen)
}

func LeerInventario(origen io.Reader) (copias.Inventario, error) {
	return leerJSON[copias.Inventario](origen)
}

func leerJSON[T any](origen io.Reader) (T, error) {
	var resultado T
	datos, err := io.ReadAll(io.LimitReader(origen, MaxDocumentoBytes+1))
	if err != nil || len(datos) > MaxDocumentoBytes {
		return resultado, ErrDocumento
	}
	datos = bytes.TrimSpace(datos)
	if len(datos) == 0 || datos[0] != '{' {
		return resultado, ErrDocumento
	}
	comprobador := json.NewDecoder(bytes.NewReader(datos))
	if err := comprobarJSON(comprobador, 0); err != nil {
		return resultado, ErrDocumento
	}
	if _, err := comprobador.Token(); err != io.EOF {
		return resultado, ErrDocumento
	}
	var forma any
	dec := json.NewDecoder(bytes.NewReader(datos))
	dec.UseNumber()
	if dec.Decode(&forma) != nil || !comprobarForma(forma, reflect.TypeFor[T]()) {
		return resultado, ErrDocumento
	}
	dec = json.NewDecoder(bytes.NewReader(datos))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&resultado); err != nil {
		return resultado, ErrDocumento
	}
	return resultado, nil
}

// El decoder estándar acepta claves repetidas; aquí se rechazan para que un
// descriptor tenga una sola interpretación. La profundidad y bytes se limitan.
func comprobarJSON(dec *json.Decoder, profundidad int) error {
	if profundidad > 32 {
		return ErrDocumento
	}
	token, err := dec.Token()
	if err != nil || token == nil {
		return ErrDocumento
	}
	delimitador, compuesto := token.(json.Delim)
	if !compuesto {
		return nil
	}
	if delimitador != '{' && delimitador != '[' {
		return ErrDocumento
	}
	vistas := map[string]bool{}
	for dec.More() {
		if delimitador == '{' {
			claveToken, err := dec.Token()
			clave, ok := claveToken.(string)
			// encoding/json admite también variantes de mayúsculas de las
			// etiquetas: el formato usa únicamente sus claves canónicas.
			if err != nil || !ok || clave != strings.ToLower(clave) || vistas[clave] {
				return ErrDocumento
			}
			vistas[clave] = true
		}
		if err := comprobarJSON(dec, profundidad+1); err != nil {
			return err
		}
	}
	_, err = dec.Token()
	return err
}
