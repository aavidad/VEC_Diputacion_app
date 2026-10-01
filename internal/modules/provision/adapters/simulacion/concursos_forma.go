package simulacion

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
)

// El DTO JSON exige sus claves exactas. encoding/json por sí solo permite
// mayúsculas equivalentes y convierte null u omisión de bool a false.
func validarForma(datos []byte, t reflect.Type) error {
	if t.Kind() == reflect.Pointer {
		if bytes.Equal(bytes.TrimSpace(datos), []byte("null")) {
			return nil
		}
		return validarForma(datos, t.Elem())
	}
	if reflect.PointerTo(t).Implements(reflect.TypeFor[json.Unmarshaler]()) {
		return nil
	}
	switch t.Kind() {
	case reflect.Struct:
		var campos map[string]json.RawMessage
		if err := json.Unmarshal(datos, &campos); err != nil || campos == nil {
			return errors.New("json_objeto_requerido")
		}
		permitidos := map[string]reflect.StructField{}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			tag := f.Tag.Get("json")
			clave, _, _ := strings.Cut(tag, ",")
			if clave != "" && clave != "-" {
				permitidos[clave] = f
			}
		}
		for clave, valor := range campos {
			f, ok := permitidos[clave]
			if !ok {
				return errors.New("json_clave_no_canonica")
			}
			if err := validarForma(valor, f.Type); err != nil {
				return err
			}
		}
		for clave, f := range permitidos {
			if !strings.Contains(f.Tag.Get("json"), ",omitempty") {
				if _, ok := campos[clave]; !ok {
					return errors.New("json_campo_requerido")
				}
			}
		}
	case reflect.Slice:
		if bytes.Equal(bytes.TrimSpace(datos), []byte("null")) {
			return nil
		}
		var elementos []json.RawMessage
		if err := json.Unmarshal(datos, &elementos); err != nil {
			return err
		}
		for _, valor := range elementos {
			if err := validarForma(valor, t.Elem()); err != nil {
				return err
			}
		}
	default:
		if bytes.Equal(bytes.TrimSpace(datos), []byte("null")) {
			return errors.New("json_valor_nulo")
		}
	}
	return nil
}
