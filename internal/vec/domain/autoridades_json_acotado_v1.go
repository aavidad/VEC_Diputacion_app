package domain

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

var errEstructuraJSONAutoridadNoAcotada = errors.New("vec: estructura JSON de autoridad no acotada")

const (
	maximaProfundidadJSONAutoridadV1 = 16
	maximosCamposObjetoAutoridadV1   = 64
	maximosElementosArrayGenericoV1  = 128
)

var limitesArrayJSONAutoridadV1 = map[string]int{
	"ambitos":            maximoAmbitosFuenteAutoridad,
	"valores_clave":      maximoValoresAmbitoAutoridad,
	"preceptos":          maximoPreceptosFuenteAutoridad,
	"ediciones_borrador": maximoEdicionesFuenteAutoridad,
	"transiciones":       maximoTransicionesFuenteAutoridad,
	"firmas_refs":        maximoFirmasActoFuenteAutoridad,
}

// validarEstructuraJSONAutoridadV1 recorre tokens antes de decodificar DTO.
// Así un estado de pocos MiB no puede forzar arrays con millones de objetos y
// asignaciones desproporcionadas antes de que actúen las invariantes.
func validarEstructuraJSONAutoridadV1(datos []byte) error {
	decodificador := json.NewDecoder(bytes.NewReader(datos))
	decodificador.UseNumber()
	if err := consumirValorJSONAutoridadV1(decodificador, "", 0); err != nil {
		return errEstructuraJSONAutoridadNoAcotada
	}
	if _, err := decodificador.Token(); !errors.Is(err, io.EOF) {
		return errEstructuraJSONAutoridadNoAcotada
	}
	return nil
}

func consumirValorJSONAutoridadV1(decodificador *json.Decoder, clave string, profundidad int) error {
	if profundidad > maximaProfundidadJSONAutoridadV1 {
		return errEstructuraJSONAutoridadNoAcotada
	}
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
		campos := 0
		vistos := make(map[string]struct{}, maximosCamposObjetoAutoridadV1)
		for decodificador.More() {
			campos++
			if campos > maximosCamposObjetoAutoridadV1 {
				return errEstructuraJSONAutoridadNoAcotada
			}
			tokenClave, err := decodificador.Token()
			if err != nil {
				return err
			}
			claveCampo, valida := tokenClave.(string)
			if !valida {
				return errEstructuraJSONAutoridadNoAcotada
			}
			if _, repetida := vistos[claveCampo]; repetida {
				return errEstructuraJSONAutoridadNoAcotada
			}
			vistos[claveCampo] = struct{}{}
			if err := consumirValorJSONAutoridadV1(decodificador, claveCampo, profundidad+1); err != nil {
				return err
			}
		}
		cierre, err := decodificador.Token()
		if err != nil || cierre != json.Delim('}') {
			return errEstructuraJSONAutoridadNoAcotada
		}
		return nil
	case '[':
		limite := maximosElementosArrayGenericoV1
		if configurado, existe := limitesArrayJSONAutoridadV1[clave]; existe {
			limite = configurado
		}
		elementos := 0
		for decodificador.More() {
			elementos++
			if elementos > limite {
				return errEstructuraJSONAutoridadNoAcotada
			}
			if err := consumirValorJSONAutoridadV1(decodificador, clave, profundidad+1); err != nil {
				return err
			}
		}
		cierre, err := decodificador.Token()
		if err != nil || cierre != json.Delim(']') {
			return errEstructuraJSONAutoridadNoAcotada
		}
		return nil
	default:
		return errEstructuraJSONAutoridadNoAcotada
	}
}
