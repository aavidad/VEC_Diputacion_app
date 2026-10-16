package ports

import (
	"encoding/json"
	"errors"
)

var ErrOrigenOfertaCTInvalido = errors.New("bolsa: origen de oferta CT invalido")

// OrigenOfertaCT liga una oferta de varias plazas al expediente y al corte
// autoritativo B110. Bolsa vuelve a verificar la constitución, el orden y la
// cantidad elegible antes de escribir; CT no decide posiciones ni candidatos.
type OrigenOfertaCT struct {
	OrganizacionRef         string              `json:"organizacion_ref"`
	UnidadRef               string              `json:"unidad_ref"`
	ExpedienteRef           string              `json:"expediente_ref"`
	VersionExpediente       uint64              `json:"version_expediente"`
	ClaveCT                 string              `json:"clave_ct"`
	NecesidadRef            string              `json:"necesidad_ref"`
	ActaRef                 string              `json:"acta_ref"`
	CategoriaRef            string              `json:"categoria_ref"`
	BolsaRef                string              `json:"bolsa_ref"`
	VersionBolsa            uint64              `json:"version_bolsa"`
	HuellaBolsaSHA256       string              `json:"huella_bolsa_sha256"`
	InstantaneaRef          string              `json:"instantanea_ref"`
	VersionInstantanea      uint64              `json:"version_instantanea"`
	HuellaInstantaneaSHA256 string              `json:"huella_instantanea_sha256"`
	PoliticaRef             string              `json:"politica_ref"`
	VersionPolitica         uint64              `json:"version_politica"`
	HuellaPoliticaSHA256    string              `json:"huella_politica_sha256"`
	Lectura                 OrigenLlamamientoCT `json:"lectura"`
}

func (o OrigenOfertaCT) NumeroElegibles() int {
	n := 0
	for _, entrada := range o.Lectura.Entradas {
		if entrada.OrdenVigente != nil && entrada.Elegible {
			n++
		}
	}
	return n
}

func (o OrigenOfertaCT) ValidarPara(bolsa string, numeroPlazas int) error {
	if o.OrganizacionRef == "" || o.UnidadRef == "" || o.ExpedienteRef == "" ||
		o.VersionExpediente == 0 || o.ClaveCT == "" || o.NecesidadRef == "" ||
		o.ActaRef == "" || o.CategoriaRef == "" || o.BolsaRef != bolsa ||
		o.VersionBolsa == 0 || o.HuellaBolsaSHA256 == "" || o.InstantaneaRef == "" ||
		o.VersionInstantanea == 0 || o.HuellaInstantaneaSHA256 == "" ||
		o.PoliticaRef == "" || o.VersionPolitica == 0 || o.HuellaPoliticaSHA256 == "" ||
		o.Lectura.ActaRef != o.ActaRef || o.Lectura.CategoriaRef != o.CategoriaRef ||
		o.Lectura.BolsaRef != o.BolsaRef || o.Lectura.VersionBolsa != o.VersionBolsa ||
		o.Lectura.HuellaBolsaSHA256 != o.HuellaBolsaSHA256 ||
		o.Lectura.InstantaneaRef != o.InstantaneaRef ||
		o.Lectura.VersionInstantanea != o.VersionInstantanea ||
		o.Lectura.HuellaInstantaneaSHA256 != o.HuellaInstantaneaSHA256 ||
		o.Lectura.PoliticaRef != o.PoliticaRef ||
		o.Lectura.VersionPolitica != o.VersionPolitica ||
		o.Lectura.HuellaPoliticaSHA256 != o.HuellaPoliticaSHA256 ||
		o.Lectura.Corte.IsZero() || o.Lectura.Corte.Nanosecond()%1000 != 0 ||
		numeroPlazas < 2 || numeroPlazas > 100 || o.NumeroElegibles() < numeroPlazas {
		return ErrOrigenOfertaCTInvalido
	}
	for _, entrada := range o.Lectura.Entradas {
		if entrada.Elegible != (entrada.OrdenVigente != nil) {
			return ErrOrigenOfertaCTInvalido
		}
	}
	return nil
}

// CorteCanonico conserva precisión de microsegundos para los materiales V3.
func (o OrigenOfertaCT) CorteCanonico() string {
	return o.Lectura.Corte.UTC().Format("2006-01-02T15:04:05.000000Z")
}

// JSONCanonico conserva la captura que intercambian las fachadas nominales.
// El corte mantiene los seis decimales de la operación original.
func (o OrigenOfertaCT) JSONCanonico() ([]byte, error) {
	type lectura struct {
		OrigenLlamamientoCT
		Corte string `json:"corte"`
	}
	type origen struct {
		OrigenOfertaCT
		Lectura lectura `json:"lectura"`
	}
	return json.Marshal(origen{OrigenOfertaCT: o, Lectura: lectura{OrigenLlamamientoCT: o.Lectura, Corte: o.CorteCanonico()}})
}
