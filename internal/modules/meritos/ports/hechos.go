package ports

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/modules/meritos/domain"
	vec "vec-diputacion-granada/internal/vec/domain"
)

var ErrHechosPreparacion = errors.New("meritos.error.hechos_preparacion")

// SelectorHechosPreparacion selecciona datos aportados en un paquete sintético.
// PersonaRef es una afirmación del ensayo; no autentica ni concede acceso.
type SelectorHechosPreparacion struct {
	PersonaRef string                       `json:"persona_ref"`
	FechaCorte string                       `json:"fecha_corte"`
	Hechos     []ReferenciaHechoPreparacion `json:"hechos"`
}

type ReferenciaHechoPreparacion struct {
	Referencia      string `json:"referencia"`
	VersionEsperada int    `json:"version_esperada"`
}

// HechoPreparado omite nombres, persona, declarante y actores de revisión.
// Estado conserva la afirmación aportada; Pendientes conserva sus límites.
type HechoPreparado struct {
	Referencia      string                    `json:"referencia"`
	Version         int                       `json:"version"`
	Tipo            string                    `json:"tipo"`
	ConceptoRef     string                    `json:"concepto_ref"`
	Horas           *int                      `json:"horas,omitempty"`
	Procedencia     domain.Procedencia        `json:"procedencia"`
	Vigencia        domain.Vigencia           `json:"vigencia"`
	Estado          domain.Estado             `json:"estado"`
	Evidencias      []vec.ReferenciaDocumento `json:"evidencias"`
	VigenciaEnCorte string                    `json:"vigencia_en_corte"`
	Pendientes      []string                  `json:"pendientes"`
}

type HechosPreparados struct {
	Alcance        string           `json:"alcance"`
	VersionPaquete string           `json:"version_paquete"`
	Hechos         []HechoPreparado `json:"hechos"`
}

// LectorHechosPreparacion trabaja exclusivamente con paquetes sintéticos.
// Exige versión actual exacta y no distingue hecho ausente, ajeno o versión
// diferente en su error. No sustituye al lector nominal autorizado de RUM05.
type LectorHechosPreparacion interface {
	LeerHechosSinteticos(context.Context, SelectorHechosPreparacion) (HechosPreparados, error)
}
