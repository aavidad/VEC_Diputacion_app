package auditoriacheckpoint

import "vec-diputacion-granada/internal/vec/domain"

// AuditoriaCheckpointOffline se lee de un archivo explícito; no activa servicios.
type AuditoriaCheckpointOffline struct {
	Politica       domain.PoliticaCheckpoint `json:"politica"`
	MaxBytes       int64                     `json:"max_bytes"`
	MaxRegistros   uint64                    `json:"max_registros"`
	VersionBinario string                    `json:"version_binario"`
}

func (c AuditoriaCheckpointOffline) Validar() error {
	if c.Politica.Validar() != nil || c.MaxBytes < 1024 || c.MaxBytes > 64*1024*1024 || c.MaxRegistros == 0 || c.MaxRegistros > 1000000 || c.VersionBinario == "" || len(c.VersionBinario) > 80 {
		return domain.ErrCheckpointInvalido
	}
	return nil
}
