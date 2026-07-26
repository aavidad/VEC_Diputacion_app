package postgres

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

func validarEncuadreJSONExpedienteO3(contenido []byte) error {
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.UseNumber()
	elementos := uint32(0)
	if err := recorrerJSONExpedienteO3(
		decodificador,
		1,
		&elementos,
	); err != nil {
		return err
	}
	if _, err := decodificador.Token(); !errors.Is(err, io.EOF) {
		return coberturaJSONExpedienteO3NoConfiable
	}
	return nil
}

var coberturaJSONExpedienteO3NoConfiable = errors.New("expediente durable O3 no confiable")

func recorrerJSONExpedienteO3(
	decodificador *json.Decoder,
	profundidad uint32,
	elementos *uint32,
) error {
	if profundidad > maximaProfundidadExpedienteO3 ||
		elementos == nil ||
		*elementos >= maximosElementosJSONExpedienteO3 {
		return coberturaJSONExpedienteO3NoConfiable
	}
	*elementos = *elementos + 1
	token, err := decodificador.Token()
	if err != nil {
		return err
	}
	delimitador, compuesto := token.(json.Delim)
	if !compuesto {
		return nil
	}
	switch delimitador {
	case '{':
		vistas := make(map[string]struct{})
		for decodificador.More() {
			tokenClave, err := decodificador.Token()
			if err != nil {
				return err
			}
			clave, valida := tokenClave.(string)
			if !valida {
				return coberturaJSONExpedienteO3NoConfiable
			}
			if _, duplicada := vistas[clave]; duplicada {
				return coberturaJSONExpedienteO3NoConfiable
			}
			vistas[clave] = struct{}{}
			if err := recorrerJSONExpedienteO3(
				decodificador,
				profundidad+1,
				elementos,
			); err != nil {
				return err
			}
		}
		cierre, err := decodificador.Token()
		if err != nil || cierre != json.Delim('}') {
			return coberturaJSONExpedienteO3NoConfiable
		}
	case '[':
		for decodificador.More() {
			if err := recorrerJSONExpedienteO3(
				decodificador,
				profundidad+1,
				elementos,
			); err != nil {
				return err
			}
		}
		cierre, err := decodificador.Token()
		if err != nil || cierre != json.Delim(']') {
			return coberturaJSONExpedienteO3NoConfiable
		}
	default:
		return coberturaJSONExpedienteO3NoConfiable
	}
	return nil
}
