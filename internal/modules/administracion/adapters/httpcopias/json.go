package httpcopias

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"time"
	p "vec-diputacion-granada/internal/modules/administracion/ports/httpcopias"
)

func (h *Handler) decode(w http.ResponseWriter, r *http.Request, s p.Sesion, dst any) bool {
	if r.Body == nil {
		h.denegar(w, r, s, p.ErrSolicitud, "escribir", "")
		return false
	}
	data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxCuerpo))
	if err != nil {
		var excess *http.MaxBytesError
		if errors.As(err, &excess) {
			h.denegarStatus(w, r, s, 413, "solicitud_invalida", "escribir", "")
		} else {
			h.denegar(w, r, s, p.ErrSolicitud, "escribir", "")
		}
		return false
	}
	scan := json.NewDecoder(bytes.NewReader(data))
	if err := scanObject(scan, 0); err != nil {
		h.denegar(w, r, s, p.ErrSolicitud, "escribir", "")
		return false
	}
	if _, err := scan.Token(); err != io.EOF {
		h.denegar(w, r, s, p.ErrSolicitud, "escribir", "")
		return false
	}
	if !requiredFields(data, reflect.TypeOf(dst).Elem()) {
		h.denegar(w, r, s, p.ErrSolicitud, "escribir", "")
		return false
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		h.denegar(w, r, s, p.ErrSolicitud, "escribir", "")
		return false
	}
	return true
}

// Reject duplicate keys, null roots and nested collections before typed decoding.
func scanObject(d *json.Decoder, depth int) error {
	token, err := d.Token()
	if err != nil || token != json.Delim('{') {
		return p.ErrSolicitud
	}
	return scanCollection(d, '{', depth)
}
func scanCollection(d *json.Decoder, kind byte, depth int) error {
	if depth > 16 {
		return p.ErrSolicitud
	}
	keys := map[string]bool{}
	for d.More() {
		if kind == '{' {
			t, e := d.Token()
			k, ok := t.(string)
			if e != nil || !ok || keys[k] {
				return p.ErrSolicitud
			}
			keys[k] = true
		}
		token, e := d.Token()
		if e != nil {
			return p.ErrSolicitud
		}
		if token == nil {
			return p.ErrSolicitud
		}
		if delim, ok := token.(json.Delim); ok {
			if delim != '{' && delim != '[' {
				return p.ErrSolicitud
			}
			if err := scanCollection(d, byte(delim), depth+1); err != nil {
				return err
			}
		}
	}
	end, e := d.Token()
	if e != nil {
		return p.ErrSolicitud
	}
	if kind == '{' && end != json.Delim('}') || kind == '[' && end != json.Delim(']') {
		return p.ErrSolicitud
	}
	return nil
}

// Require exact JSON names before encoding/json can apply case-insensitive aliases.
// Required zero-valued fields (including CAS=0 and false) must be explicit.
func requiredFields(data []byte, t reflect.Type) bool {
	if t.Kind() != reflect.Struct || t == reflect.TypeFor[time.Time]() {
		return true
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(data, &fields) != nil {
		return false
	}
	allowed := make(map[string]struct{}, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag := strings.Split(f.Tag.Get("json"), ",")
		if len(tag) == 0 || tag[0] == "" || tag[0] == "-" {
			continue
		}
		allowed[tag[0]] = struct{}{}
		raw, ok := fields[tag[0]]
		optional := len(tag) > 1 && tag[1] == "omitempty"
		if !ok {
			if optional {
				continue
			}
			return false
		}
		if f.Type.Kind() == reflect.Struct && !requiredFields(raw, f.Type) {
			return false
		}
	}
	for key := range fields {
		if _, ok := allowed[key]; !ok {
			return false
		}
	}
	return true
}
