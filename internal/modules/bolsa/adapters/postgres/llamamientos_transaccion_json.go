package postgres

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

const maximaProfundidadJSONLlamamiento = 64

// validarJSONLlamamientoNoAmbiguo recorre tokens antes de decodificar el DTO.
// encoding/json conserva silenciosamente la ultima clave duplicada, tambien
// dentro de objetos anidados; un recibo durable no puede aceptar esa eleccion.
func validarJSONLlamamientoNoAmbiguo(contenido []byte) error {
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.UseNumber()
	if err := consumirValorJSONLlamamiento(decodificador, 0); err != nil {
		return err
	}
	if _, err := decodificador.Token(); !errors.Is(err, io.EOF) {
		return errors.New("contenido JSON adicional")
	}
	return nil
}

func consumirValorJSONLlamamiento(decodificador *json.Decoder, profundidad int) error {
	if profundidad > maximaProfundidadJSONLlamamiento {
		return errors.New("profundidad JSON excesiva")
	}
	token, err := decodificador.Token()
	if err != nil {
		return errors.New("token JSON invalido")
	}
	delimitador, compuesto := token.(json.Delim)
	if !compuesto {
		return nil
	}
	switch delimitador {
	case '{':
		claves := make(map[string]struct{})
		for decodificador.More() {
			tokenClave, err := decodificador.Token()
			clave, esCadena := tokenClave.(string)
			if err != nil || !esCadena {
				return errors.New("clave JSON invalida")
			}
			if _, duplicada := claves[clave]; duplicada {
				return errors.New("clave JSON duplicada")
			}
			claves[clave] = struct{}{}
			if err := consumirValorJSONLlamamiento(decodificador, profundidad+1); err != nil {
				return err
			}
		}
		cierre, err := decodificador.Token()
		if err != nil || cierre != json.Delim('}') {
			return errors.New("objeto JSON sin cierre")
		}
	case '[':
		for decodificador.More() {
			if err := consumirValorJSONLlamamiento(decodificador, profundidad+1); err != nil {
				return err
			}
		}
		cierre, err := decodificador.Token()
		if err != nil || cierre != json.Delim(']') {
			return errors.New("lista JSON sin cierre")
		}
	default:
		return errors.New("delimitador JSON inesperado")
	}
	return nil
}
