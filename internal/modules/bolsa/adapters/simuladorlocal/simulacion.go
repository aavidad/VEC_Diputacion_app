// Package simuladorlocal expone únicamente ejemplos sintéticos del motor común.
package simuladorlocal

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"

	"vec-diputacion-granada/internal/modules/bolsa/application/simulacionbaremo"
	"vec-diputacion-granada/internal/modules/bolsa/domain/calculomeritos"
	"vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
)

var ErrSolicitud = errors.New("solicitud_invalida")

type Ejemplo struct {
	Referencia string          `json:"referencia"`
	Modo       string          `json:"modo"`
	Reglas     json.RawMessage `json:"reglas"`
}

type Solicitud struct {
	Modo       string          `json:"modo"`
	EjemploRef string          `json:"ejemplo_ref"`
	Reglas     json.RawMessage `json:"reglas"`
}

type Motor struct{}

func (Motor) Ejemplos() []Ejemplo {
	rE, _ := simulacionbaremo.DatosEjemploExperiencia()
	rM, _ := simulacionbaremo.DatosEjemploMeritos()
	return []Ejemplo{{"experiencia_sintetica_v1", "experiencia", rE}, {"meritos_sinteticos_v1", "meritos", rM}}
}

// Simular no admite entrada de personas ni una referencia a un fichero.
// Canonicaliza las reglas en Go y conserva la entrada embebida del ejemplo.
func (Motor) Simular(s Solicitud) ([]byte, error) {
	var reglas, entrada []byte
	var err error
	switch {
	case s.Modo == "experiencia" && s.EjemploRef == "experiencia_sintetica_v1":
		reglas, err = reglasbaremo.CanonicalizarConjuntoReglasBaremoJSON(s.Reglas)
		_, entrada = simulacionbaremo.DatosEjemploExperiencia()
	case s.Modo == "meritos" && s.EjemploRef == "meritos_sinteticos_v1":
		var conjunto calculomeritos.Conjunto
		dec := json.NewDecoder(bytes.NewReader(s.Reglas))
		dec.DisallowUnknownFields()
		err = dec.Decode(&conjunto)
		if err == nil {
			reglas, err = conjunto.RepresentacionCanonica()
		}
		_, entrada = simulacionbaremo.DatosEjemploMeritos()
	default:
		return nil, ErrSolicitud
	}
	if err != nil {
		return nil, ErrSolicitud
	}
	solicitud := simulacionbaremo.Solicitud{ConjuntoCanonico: reglas, HuellaConjuntoSHA256: huella(reglas), EntradaCanonica: entrada, HuellaEntradaSHA256: huella(entrada)}
	if s.Modo == "experiencia" {
		resultado, err := (simulacionbaremo.Servicio{}).Simular(solicitud)
		if err != nil {
			return nil, ErrSolicitud
		}
		return resultado.RepresentacionCanonica(), nil
	}
	resultado, err := (simulacionbaremo.ServicioMeritos{}).SimularMeritos(solicitud)
	if err != nil {
		return nil, ErrSolicitud
	}
	return resultado.RepresentacionCanonica(), nil
}

func huella(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}
