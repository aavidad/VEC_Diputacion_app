package simulacion

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"unicode/utf8"
	"vec-diputacion-granada/internal/modules/seleccion/domain"
)

var ErrSolicitud = errors.New("solicitud_invalida")

const MaximoBytes = 64 * 1024

type Solicitud struct {
	EjemploRef    string               `json:"ejemplo_ref"`
	Configuracion domain.Configuracion `json:"configuracion"`
}

func Decodificar(datos []byte) (Solicitud, error) {
	if len(datos) == 0 || len(datos) > MaximoBytes || !utf8.Valid(datos) {
		return Solicitud{}, ErrSolicitud
	}
	d := json.NewDecoder(bytes.NewReader(datos))
	d.UseNumber()
	if err := tokens(d, 0); err != nil {
		return Solicitud{}, err
	}
	if _, err := d.Token(); err != io.EOF {
		return Solicitud{}, ErrSolicitud
	}
	d = json.NewDecoder(bytes.NewReader(datos))
	d.DisallowUnknownFields()
	var s Solicitud
	if err := d.Decode(&s); err != nil || len(s.EjemploRef) == 0 || len(s.EjemploRef) > 64 {
		return Solicitud{}, ErrSolicitud
	}
	return s, nil
}

// Claves exactas en ASCII impiden que encoding/json acepte alias Unicode o
// cambios de mayúsculas. Un límite anterior al DTO acota profundidad y nodos.
func tokens(d *json.Decoder, profundidad int) error {
	if profundidad > 12 {
		return ErrSolicitud
	}
	t, err := d.Token()
	if err != nil {
		return ErrSolicitud
	}
	delim, coleccion := t.(json.Delim)
	if profundidad == 0 && (!coleccion || delim != '{') {
		return ErrSolicitud
	}
	if !coleccion {
		if s, ok := t.(string); ok && len(s) > 256 {
			return ErrSolicitud
		}
		return nil
	}
	if delim != '{' && delim != '[' {
		return ErrSolicitud
	}
	vistas := map[string]bool{}
	for n := 0; d.More(); n++ {
		if n >= 128 {
			return ErrSolicitud
		}
		if delim == '{' {
			t, err := d.Token()
			clave, ok := t.(string)
			if err != nil || !ok || vistas[clave] || len(clave) == 0 || len(clave) > 64 {
				return ErrSolicitud
			}
			for _, r := range clave {
				if r != '_' && (r < 'a' || r > 'z') && (r < '0' || r > '9') {
					return ErrSolicitud
				}
			}
			vistas[clave] = true
		}
		if err := tokens(d, profundidad+1); err != nil {
			return err
		}
	}
	cierre, err := d.Token()
	if err != nil || (delim == '{' && cierre != json.Delim('}')) || (delim == '[' && cierre != json.Delim(']')) {
		return ErrSolicitud
	}
	return nil
}
