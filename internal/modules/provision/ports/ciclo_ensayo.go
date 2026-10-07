package ports

import "vec-diputacion-granada/internal/modules/provision/domain"

type PeticionCiclo struct {
	SchemaVersion      string                    `json:"schema_version"`
	Configuracion      domain.Configuracion      `json:"configuracion"`
	Entrada            domain.Entrada            `json:"entrada"`
	CatalogoCausas     domain.CatalogoCausas     `json:"catalogo_causas"`
	RevisionInicialRef string                    `json:"revision_inicial_ref,omitempty"`
	Reclamaciones      []domain.Reclamacion      `json:"reclamaciones"`
	Decisiones         []domain.DecisionRevision `json:"decisiones"`
}

type EnsayadorCiclo interface {
	EnsayarCiclo(PeticionCiclo) (domain.CicloEnsayado, error)
}
