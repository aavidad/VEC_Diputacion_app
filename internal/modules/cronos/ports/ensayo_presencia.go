package ports

import (
	"context"
	"time"
	"vec-diputacion-granada/internal/modules/cronos/domain"
)

// These DTOs belong exclusively to the synthetic CLI. A selection of references
// is not a grant and cannot be used to query Personal or the runtime repository.
type MarcajeEnsayoPresencia struct {
	Referencia  string           `json:"referencia"`
	Version     int64            `json:"version"`
	FuenteRef   string           `json:"fuente_ref"`
	Movimiento  domain.PunchKind `json:"movimiento"`
	InstanteUTC time.Time        `json:"instante_utc"`
}
type PersonaEnsayoPresencia struct {
	PersonaRef         string                   `json:"persona_ref"`
	CompletaHastaCorte *bool                    `json:"completa_hasta_corte"`
	Marcajes           []MarcajeEnsayoPresencia `json:"marcajes"`
}
type SnapshotEnsayoPresencia struct {
	VersionEsquema   int                      `json:"version_esquema"`
	Demostracion     bool                     `json:"demostracion"`
	Referencia       string                   `json:"referencia"`
	Version          int64                    `json:"version"`
	FuenteRef        string                   `json:"fuente_ref"`
	FuenteVersion    int64                    `json:"fuente_version"`
	OrganizacionRef  string                   `json:"organizacion_ref"`
	Fecha            string                   `json:"fecha"`
	ZonaHoraria      string                   `json:"zona_horaria"`
	InstanteCorteUTC time.Time                `json:"instante_corte_utc"`
	PersonasRef      []string                 `json:"personas_ref"`
	Personas         []PersonaEnsayoPresencia `json:"personas"`
	SHA256           string                   `json:"-"`
}

// LectorSnapshotEnsayoPresencia is a single immutable synthetic snapshot read.
type LectorSnapshotEnsayoPresencia interface {
	LeerSnapshotEnsayoPresencia(context.Context) (SnapshotEnsayoPresencia, error)
}
type EstadoPersonaEnsayoPresencia struct {
	PersonaRef        string                 `json:"persona_ref"`
	Estado            domain.EstadoPresencia `json:"estado"`
	Motivo            domain.CausaPresencia  `json:"motivo,omitempty"`
	CoberturaCompleta bool                   `json:"cobertura_completa"`
}
type AgregadoEnsayoPresencia struct {
	Total               int `json:"total"`
	EntradasRegistradas int `json:"entradas_registradas"`
	PausasRegistradas   int `json:"pausas_registradas"`
	SalidasRegistradas  int `json:"salidas_registradas"`
	Indeterminado       int `json:"indeterminado"`
}
type ResultadoEnsayoPresencia struct {
	Demostracion     bool                           `json:"demostracion"`
	SnapshotRef      string                         `json:"snapshot_ref"`
	SnapshotVersion  int64                          `json:"snapshot_version"`
	SnapshotSHA256   string                         `json:"snapshot_sha256"`
	FuenteRef        string                         `json:"fuente_ref"`
	FuenteVersion    int64                          `json:"fuente_version"`
	OrganizacionRef  string                         `json:"organizacion_ref"`
	Fecha            string                         `json:"fecha"`
	ZonaHoraria      string                         `json:"zona_horaria"`
	InstanteCorteUTC time.Time                      `json:"instante_corte_utc"`
	Personas         []EstadoPersonaEnsayoPresencia `json:"personas"`
	Agregado         AgregadoEnsayoPresencia        `json:"agregado"`
}
