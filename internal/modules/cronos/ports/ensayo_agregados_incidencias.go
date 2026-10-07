package ports

import (
	"context"
	"time"
	"vec-diputacion-granada/internal/modules/cronos/domain"
)

// DTOs for the local synthetic CLI only. Selection is not authorization.
type FuenteEnsayoAgregadosIncidencias struct {
	Referencia string `json:"referencia"`
	Version    int64  `json:"version"`
	SHA256     string `json:"sha256"`
	Estado     string `json:"estado"`
}
type CatalogoEnsayoAgregadosIncidencias struct {
	Referencia string                            `json:"referencia"`
	Version    int64                             `json:"version"`
	Codigos    []domain.CodigoIncidenciaAgregada `json:"codigos"`
	Estados    []domain.EstadoIncidenciaAgregada `json:"estados"`
}
type CoberturaEnsayoAgregadosIncidencias struct {
	PersonaRef    string                      `json:"persona_ref"`
	Fecha         string                      `json:"fecha"`
	FuenteRef     string                      `json:"fuente_ref"`
	FuenteVersion int64                       `json:"fuente_version"`
	Estado        domain.CoberturaIncidencias `json:"estado"`
}
type HechoEnsayoAgregadosIncidencias struct {
	Referencia    string                          `json:"referencia"`
	Version       int64                           `json:"version"`
	PersonaRef    string                          `json:"persona_ref"`
	Fecha         string                          `json:"fecha"`
	Codigo        domain.CodigoIncidenciaAgregada `json:"codigo"`
	Estado        domain.EstadoIncidenciaAgregada `json:"estado"`
	FuenteRef     string                          `json:"fuente_ref"`
	FuenteVersion int64                           `json:"fuente_version"`
	RegistradaUTC time.Time                       `json:"registrada_utc"`
}
type SnapshotEnsayoAgregadosIncidencias struct {
	VersionEsquema   int                                   `json:"version_esquema"`
	Demo             bool                                  `json:"demo"`
	Referencia       string                                `json:"referencia"`
	Version          int64                                 `json:"version"`
	OrganizacionRef  string                                `json:"organizacion_ref"`
	Desde            string                                `json:"desde"`
	HastaExclusiva   string                                `json:"hasta_exclusiva"`
	ZonaHoraria      string                                `json:"zona_horaria"`
	InstanteCorteUTC time.Time                             `json:"instante_corte_utc"`
	PersonasRef      []string                              `json:"personas_ref"`
	Catalogo         CatalogoEnsayoAgregadosIncidencias    `json:"catalogo"`
	Fuentes          []FuenteEnsayoAgregadosIncidencias    `json:"fuentes"`
	Cobertura        []CoberturaEnsayoAgregadosIncidencias `json:"cobertura"`
	Incidencias      []HechoEnsayoAgregadosIncidencias     `json:"incidencias"`
	SHA256           string                                `json:"-"`
}
type LectorSnapshotEnsayoAgregadosIncidencias interface {
	LeerSnapshotEnsayoAgregadosIncidencias(context.Context) (SnapshotEnsayoAgregadosIncidencias, error)
}

// The output contains neither individual references nor an individual list.
type ResultadoEnsayoAgregadosIncidencias struct {
	Demo             bool                               `json:"demo"`
	SnapshotRef      string                             `json:"snapshot_ref"`
	SnapshotVersion  int64                              `json:"snapshot_version"`
	SnapshotSHA256   string                             `json:"snapshot_sha256"`
	OrganizacionRef  string                             `json:"organizacion_ref"`
	Desde            string                             `json:"desde"`
	HastaExclusiva   string                             `json:"hasta_exclusiva"`
	ZonaHoraria      string                             `json:"zona_horaria"`
	InstanteCorteUTC time.Time                          `json:"instante_corte_utc"`
	CatalogoRef      string                             `json:"catalogo_ref"`
	CatalogoVersion  int64                              `json:"catalogo_version"`
	CatalogoSHA256   string                             `json:"catalogo_sha256"`
	Fuentes          []FuenteEnsayoAgregadosIncidencias `json:"fuentes"`
	Agregado         domain.AgregadoIncidencias         `json:"agregado"`
}
