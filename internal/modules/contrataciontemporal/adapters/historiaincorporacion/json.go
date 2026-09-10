package historiaincorporacion

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"time"
)

func hash(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func sha(s string) bool {
	b, e := hex.DecodeString(s)
	return e == nil && len(b) == 32 && hex.EncodeToString(b) == s
}

// Rechaza ambigüedad ANTES de deserializar. Límites operativos del transporte,
// no nuevos límites del dominio. El tamaño se comprueba antes de clonar bytes.
func jsonValido(b []byte, max int) error {
	if len(b) == 0 || len(b) > max {
		return ErrHistoria
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	n := 0
	var valor func(int) error
	valor = func(prof int) error {
		n++
		if prof > 32 || n > 500000 {
			return ErrHistoria
		}
		t, e := d.Token()
		if e != nil {
			return ErrHistoria
		}
		delim, ok := t.(json.Delim)
		if !ok {
			return nil
		}
		switch delim {
		case '{':
			vistos := map[string]bool{}
			for d.More() {
				k, e := d.Token()
				s, ok := k.(string)
				if e != nil || !ok || vistos[s] {
					return ErrHistoria
				}
				vistos[s] = true
				if e = valor(prof + 1); e != nil {
					return e
				}
			}
			t, e = d.Token()
			if e != nil || t != json.Delim('}') {
				return ErrHistoria
			}
		case '[':
			for d.More() {
				if e := valor(prof + 1); e != nil {
					return e
				}
			}
			t, e = d.Token()
			if e != nil || t != json.Delim(']') {
				return ErrHistoria
			}
		default:
			return ErrHistoria
		}
		return nil
	}
	if e := valor(0); e != nil {
		return e
	}
	if _, e := d.Token(); e != io.EOF {
		return ErrHistoria
	}
	return nil
}

func jsonOrdenado(b []byte) ([]byte, error) {
	var v any
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	if d.Decode(&v) != nil {
		return nil, ErrHistoria
	}
	return json.Marshal(v)
}

// Exige forma/tipos/case/null/miembros obligatorios sin reflexión propia. El
// roundtrip del DTO cerrado no omite campos obligatorios ni redondea uint64.
func decodificarDocumento(b []byte) (Documento, error) {
	var d Documento
	if jsonValido(b, MaximoBytesDocumento) != nil {
		return d, ErrHistoria
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if dec.Decode(&d) != nil {
		return Documento{}, ErrHistoria
	}
	original, e := jsonOrdenado(b)
	if e != nil {
		return Documento{}, ErrHistoria
	}
	canon, e := json.Marshal(d)
	if e != nil {
		return Documento{}, ErrHistoria
	}
	normal, e := jsonOrdenado(canon)
	if e != nil || !bytes.Equal(original, normal) {
		return Documento{}, ErrHistoria
	}
	return d, nil
}
func fecha(s string) (time.Time, error) {
	t, e := time.Parse("2006-01-02T15:04:05.000000Z", s)
	if e != nil || t.Format("2006-01-02T15:04:05.000000Z") != s {
		return time.Time{}, ErrHistoria
	}
	return t, nil
}
func fechaCapacidad(s string) (time.Time, error) {
	t, e := time.Parse(time.RFC3339Nano, s)
	if e != nil || t.Format(time.RFC3339Nano) != s || t.Location() != time.UTC || t.Nanosecond()%1000 != 0 {
		return time.Time{}, ErrHistoria
	}
	return t, nil
}
