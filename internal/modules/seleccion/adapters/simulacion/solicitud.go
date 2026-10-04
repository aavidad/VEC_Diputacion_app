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
	NotasPrueba   []NotaPrueba         `json:"notas_prueba,omitempty"`
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
	var forma struct {
		EjemploRef    string               `json:"ejemplo_ref"`
		Configuracion domain.Configuracion `json:"configuracion"`
		NotasPrueba   json.RawMessage      `json:"notas_prueba"`
	}
	if err := d.Decode(&forma); err != nil || len(forma.EjemploRef) == 0 || len(forma.EjemploRef) > 64 {
		return Solicitud{}, ErrSolicitud
	}
	s := Solicitud{EjemploRef: forma.EjemploRef, Configuracion: forma.Configuracion}
	if len(forma.NotasPrueba) == 0 {
		return s, nil
	}
	if bytes.Equal(forma.NotasPrueba, []byte("null")) {
		return Solicitud{}, ErrSolicitud
	}
	nd := json.NewDecoder(bytes.NewReader(forma.NotasPrueba))
	nd.DisallowUnknownFields()
	var propuestas []notaPruebaJSON
	if err := nd.Decode(&propuestas); err != nil || propuestas == nil {
		return Solicitud{}, ErrSolicitud
	}
	for _, propuesta := range propuestas {
		if propuesta.SolicitudRef == "" || propuesta.FaseRef == "" || len(propuesta.PuntosMicropuntos) == 0 {
			return Solicitud{}, ErrSolicitud
		}
		var puntos *int64
		if err := json.Unmarshal(propuesta.PuntosMicropuntos, &puntos); err != nil {
			return Solicitud{}, ErrSolicitud
		}
		s.NotasPrueba = append(s.NotasPrueba, NotaPrueba{SolicitudRef: propuesta.SolicitudRef, FaseRef: propuesta.FaseRef, PuntosMicropuntos: puntos})
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
