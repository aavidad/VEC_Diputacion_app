// Package httpmodulos añade una restricción operativa al handler del módulo.
// El handler conserva su autenticación y autorización: el guard no concede nada.
package httpmodulos

import (
	"context"
	"errors"
	"net/http"
	"reflect"

	domain "vec-diputacion-granada/internal/modules/administracion/domain/modulos"
	ports "vec-diputacion-granada/internal/modules/administracion/ports/modulos"
)

func Proteger(moduloID string, control ports.ControlOperativo, siguiente http.Handler) (http.Handler, error) {
	if moduloID == "" || esNulo(control) || esNulo(siguiente) {
		return nil, domain.ErrConfiguracion
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// La lectura es por petición, sin caché de un estado habilitado anterior.
		if err := control.ExigirHabilitado(r.Context(), moduloID); err != nil {
			w.Header().Set("Cache-Control", "no-store")
			status := http.StatusServiceUnavailable
			if errors.Is(err, domain.ErrDesactivado) {
				status = http.StatusNotFound
			}
			if errors.Is(err, context.Canceled) {
				return
			}
			w.WriteHeader(status)
			return
		}
		siguiente.ServeHTTP(w, r)
	}), nil
}

func esNulo(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Func, reflect.Map, reflect.Slice, reflect.Chan:
		return r.IsNil()
	}
	return false
}
