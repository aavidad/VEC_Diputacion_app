package inventariocopias

import (
	"encoding/json"
	"reflect"
	"strings"
)

// El formato cerrado exige todos los campos del DTO, incluso listas vacías.
// Una lista omitida o null no equivale a una observación explícitamente vacía.
func comprobarForma(valor any, tipo reflect.Type) bool {
	if valor == nil {
		return false
	}
	switch tipo.Kind() {
	case reflect.Struct:
		objeto, ok := valor.(map[string]any)
		if !ok || len(objeto) != tipo.NumField() {
			return false
		}
		for i := 0; i < tipo.NumField(); i++ {
			campo := tipo.Field(i)
			clave := strings.Split(campo.Tag.Get("json"), ",")[0]
			dato, existe := objeto[clave]
			if !existe || !comprobarForma(dato, campo.Type) {
				return false
			}
		}
		return true
	case reflect.Slice:
		lista, ok := valor.([]any)
		if !ok {
			return false
		}
		for _, elemento := range lista {
			if !comprobarForma(elemento, tipo.Elem()) {
				return false
			}
		}
		return true
	case reflect.String:
		_, ok := valor.(string)
		return ok
	case reflect.Bool:
		_, ok := valor.(bool)
		return ok
	case reflect.Int, reflect.Int64:
		_, ok := valor.(json.Number)
		return ok // Decode comprueba rango y representación entera.
	default:
		return false
	}
}
