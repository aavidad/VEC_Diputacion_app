package postgresimportacionconvoca

import (
	"bytes"
	"crypto/subtle"
	"reflect"
)

func newLectorBytes(contenido []byte) *bytes.Reader {
	return bytes.NewReader(contenido)
}

func bytesIgualesConstantes(a, b []byte) bool {
	return len(a) == len(b) && subtle.ConstantTimeCompare(a, b) == 1
}

func valorNulo(valor any) bool {
	if valor == nil {
		return true
	}
	v := reflect.ValueOf(valor)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}
