package ajustesreglas

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
)

var errJSONAjustes = errors.New("json de ajustes invalido")

func validarJSONSinDuplicados(b []byte) error {
	d := json.NewDecoder(bytes.NewReader(b))
	tokens := 0
	var leer func(int) error
	leer = func(n int) error {
		tokens++
		if n > 32 || tokens > 100_000 {
			return errJSONAjustes
		}
		t, e := d.Token()
		if e != nil {
			return errJSONAjustes
		}
		delim, ok := t.(json.Delim)
		if !ok {
			return nil
		}
		switch delim {
		case '{':
			vistas := map[string]bool{}
			for d.More() {
				k, e := d.Token()
				s, ok := k.(string)
				if e != nil || !ok || s != strings.ToLower(s) || vistas[s] {
					return errJSONAjustes
				}
				vistas[s] = true
				if e := leer(n + 1); e != nil {
					return e
				}
			}
			c, e := d.Token()
			if e != nil || c != json.Delim('}') {
				return errJSONAjustes
			}
		case '[':
			for d.More() {
				if e := leer(n + 1); e != nil {
					return e
				}
			}
			c, e := d.Token()
			if e != nil || c != json.Delim(']') {
				return errJSONAjustes
			}
		default:
			return errJSONAjustes
		}
		return nil
	}
	if leer(0) != nil {
		return errJSONAjustes
	}
	if _, e := d.Token(); e != io.EOF {
		return errJSONAjustes
	}
	return nil
}
