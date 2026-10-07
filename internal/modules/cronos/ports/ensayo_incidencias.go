package ports

import (
	"context"
	"vec-diputacion-granada/internal/modules/cronos/domain"
)

type FuenteEnsayoIncidencias struct {
	Referencia string `json:"referencia"`
	Version    int64  `json:"version"`
	SHA256     string `json:"sha256"`
}

type SnapshotEnsayoIncidencias struct {
	VersionEsquema int                               `json:"version_esquema"`
	Demostracion   bool                              `json:"demostracion"`
	Ref            string                            `json:"ref"`
	Version        int64                             `json:"version"`
	PersonaRef     string                            `json:"persona_ref"`
	Fuente         FuenteEnsayoIncidencias           `json:"fuente"`
	Consulta       domain.ConsultaIncidenciasPeriodo `json:"consulta"`
	SHA256         string                            `json:"-"`
}

type LectorEnsayoIncidencias interface {
	LeerSnapshotEnsayoIncidencias(context.Context) (SnapshotEnsayoIncidencias, error)
}

type ResultadoEnsayoIncidencias struct {
	Demostracion          bool                            `json:"demostracion"`
	SnapshotRef           string                          `json:"snapshot_ref"`
	SnapshotVersion       int64                           `json:"snapshot_version"`
	SnapshotSHA256        string                          `json:"snapshot_sha256"`
	PersonaRef            string                          `json:"persona_ref"`
	Fuente                FuenteEnsayoIncidencias         `json:"fuente"`
	Desde                 string                          `json:"desde"`
	Hasta                 string                          `json:"hasta"`
	Zona                  string                          `json:"zona"`
	CorteUTC              string                          `json:"corte_utc"`
	AntecedentesCompletos bool                            `json:"antecedentes_completos"`
	Dias                  []domain.RegistroDiaIncidencias `json:"dias"`
}
