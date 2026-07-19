package bootstrap

import (
	"errors"
	"net/http"

	"vec-diputacion-granada/config"
	httpcartografia "vec-diputacion-granada/internal/modules/dietas/adapters/httpcartografia"
)

var ErrComposicionCartografiaPresentacionInvalida = errors.New("bootstrap: composicion cartografica de presentacion invalida")

// NewHTTPServerCartografiaPresentacionWithConfig construye el proceso
// cartografico aislado. No compone VEC, identidad, persistencia ni datos de
// expedientes; su unica dependencia de red es el OSRM expresamente cercado.
func NewHTTPServerCartografiaPresentacionWithConfig(cfg config.Config) (*http.Server, error) {
	cfg = cfg.Normalize()
	casoUso, err := nuevoCasoUsoCalculoRutas(cfg)
	if err != nil || casoUso == nil {
		return nil, errors.Join(ErrComposicionCartografiaPresentacionInvalida, err)
	}
	superficie, err := httpcartografia.NuevaSuperficiePresentacion(cfg, casoUso)
	if err != nil {
		return nil, errors.Join(ErrComposicionCartografiaPresentacionInvalida, err)
	}
	return &http.Server{
		Addr:              cfg.Address,
		Handler:           superficie,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		MaxHeaderBytes:    cfg.MaxHeaderBytes,
	}, nil
}
