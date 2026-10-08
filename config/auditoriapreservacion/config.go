package auditoriapreservacion

import (
	"regexp"
	"time"
	"vec-diputacion-granada/internal/vec/domain"
)

// Ejecutor contiene límites técnicos, no plazos o políticas de conservación.
type Ejecutor struct {
	Version         uint64 `json:"version"`
	TimeoutSegundos int64  `json:"timeout_segundos"`
	VersionBinario  string `json:"version_binario"`
}

var versionBinarioValida = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,79}$`)

func (c Ejecutor) Validar() error {
	if c.Version != 1 || c.TimeoutSegundos < 1 || c.TimeoutSegundos > 300 || !versionBinarioValida.MatchString(c.VersionBinario) {
		return domain.ErrPreservacionAuditoriaInvalida
	}
	return nil
}
func (c Ejecutor) Timeout() time.Duration {
	if c.Validar() != nil {
		return 0
	}
	return time.Duration(c.TimeoutSegundos) * time.Second
}
