package calculoexperienciaoficial

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"unicode/utf8"
)

const (
	maximaProfundidadJSONV1 = 12
	maximosCamposObjetoV1   = 24
)

func decodificarJSONEstricto(contenido []byte, destino any) error {
	if len(contenido) == 0 || len(contenido) > maximoBytesRepresentacionV1 {
		return nuevoError("representacion_canonica", CodigoFueraDeLimites)
	}
	if !utf8.Valid(contenido) {
		return nuevoError("representacion_canonica", CodigoValorNoCanonico)
	}
	if err := comprobarClavesJSONUnicas(contenido); err != nil {
		return err
	}
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(destino); err != nil {
		return nuevoError("representacion_canonica", CodigoValorNoCanonico)
	}
	var sobrante struct{}
	if err := decodificador.Decode(&sobrante); !errors.Is(err, io.EOF) {
		return nuevoError("representacion_canonica", CodigoValorNoCanonico)
	}
	return nil
}

func comprobarClavesJSONUnicas(contenido []byte) error {
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.UseNumber()
	if err := recorrerValorJSON(decodificador, 0); err != nil {
		return err
	}
	var sobrante any
	if err := decodificador.Decode(&sobrante); !errors.Is(err, io.EOF) {
		return nuevoError("representacion_canonica", CodigoValorNoCanonico)
	}
	return nil
}

func recorrerValorJSON(decodificador *json.Decoder, profundidad int) error {
	if profundidad > maximaProfundidadJSONV1 {
		return nuevoError("representacion_canonica", CodigoFueraDeLimites)
	}
	token, err := decodificador.Token()
	if err != nil {
		return nuevoError("representacion_canonica", CodigoValorNoCanonico)
	}
	delimitador, compuesto := token.(json.Delim)
	if !compuesto {
		return nil
	}
	switch delimitador {
	case '{':
		claves := make(map[string]struct{}, maximosCamposObjetoV1)
		for decodificador.More() {
			tokenClave, err := decodificador.Token()
			clave, valida := tokenClave.(string)
			if err != nil || !valida || len(claves) >= maximosCamposObjetoV1 {
				return nuevoError("representacion_canonica", CodigoFueraDeLimites)
			}
			if _, existe := claves[clave]; existe {
				return nuevoError("representacion_canonica", CodigoValorNoCanonico)
			}
			claves[clave] = struct{}{}
			if err := recorrerValorJSON(decodificador, profundidad+1); err != nil {
				return err
			}
		}
		cierre, err := decodificador.Token()
		if err != nil || cierre != json.Delim('}') {
			return nuevoError("representacion_canonica", CodigoValorNoCanonico)
		}
	case '[':
		for decodificador.More() {
			if err := recorrerValorJSON(decodificador, profundidad+1); err != nil {
				return err
			}
		}
		cierre, err := decodificador.Token()
		if err != nil || cierre != json.Delim(']') {
			return nuevoError("representacion_canonica", CodigoValorNoCanonico)
		}
	default:
		return nuevoError("representacion_canonica", CodigoValorNoCanonico)
	}
	return nil
}
