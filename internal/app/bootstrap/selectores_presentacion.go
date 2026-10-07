package bootstrap

import (
	"errors"

	"vec-diputacion-granada/config"
)

// ErrPresentacionRRHHEnComposicionNormal se conserva mientras exista el perfil
// presentacion_rrhh en config: el binario de presentación ya no existe, pero las
// composiciones normales siguen rechazando sus selectores en lugar de ignorarlos.
var ErrPresentacionRRHHEnComposicionNormal = errors.New("bootstrap: selectores de presentacion RRHH prohibidos en composicion normal")

func rechazarSelectoresPresentacionEnComposicionNormal(cfg config.Config) error {
	if cfg.HasRRHHPresentationSelectors() {
		return ErrPresentacionRRHHEnComposicionNormal
	}
	return nil
}
