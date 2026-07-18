package bootstrap

import (
	"errors"
	"net/http"
	"path/filepath"
	"strings"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/app/server"
)

var (
	ErrPresentacionRRHHEnComposicionNormal = errors.New("bootstrap: selectores de presentacion RRHH prohibidos en composicion normal")
	ErrComposicionPresentacionRRHHInvalida = errors.New("bootstrap: composicion de presentacion RRHH incompleta o insegura")
)

// NewHTTPServerPresentacionWithConfig es la raiz minima y exclusiva del
// artefacto de presentacion. Solo compone la consulta publica desde JSON
// sintetico y el servidor de estaticos allowlisted; no crea identidad,
// PostgreSQL, S3, firma, registro, pagos, comunicaciones ni clientes de red.
func NewHTTPServerPresentacionWithConfig(cfg config.Config) (*http.Server, error) {
	cfg = cfg.Normalize()
	if !cfg.RRHHPresentationEnabledByDoubleGuard() ||
		cfg.AuthMode != config.AuthModeDisabled || cfg.StorageMode != config.StorageModeMemory ||
		cfg.FakeCredentialsPath != "" || cfg.PersonalCatalogPath != "" || !cfg.PersonalCatalogInMemory ||
		cfg.OSRMBaseURL != "" || len(cfg.OSRMAllowedCIDRs) != 0 ||
		!rutaSinteticaPresentacion(cfg.BolsaPublicSourcePath) ||
		!rutaSinteticaPresentacion(cfg.BolsaCategoriesSourcePath) {
		return nil, ErrComposicionPresentacionRRHHInvalida
	}
	apiPublica, err := NewAPIPublicaBolsaWithConfig(cfg)
	if err != nil {
		return nil, errors.Join(ErrComposicionPresentacionRRHHInvalida, err)
	}
	return server.NewHTTPServerPresentacion(cfg, apiPublica)
}

func rechazarSelectoresPresentacionEnComposicionNormal(cfg config.Config) error {
	if cfg.HasRRHHPresentationSelectors() {
		return ErrPresentacionRRHHEnComposicionNormal
	}
	return nil
}

func rutaSinteticaPresentacion(ruta string) bool {
	base := strings.ToLower(filepath.Base(strings.TrimSpace(ruta)))
	return strings.HasSuffix(base, ".demo.json") && base != ".demo.json"
}
