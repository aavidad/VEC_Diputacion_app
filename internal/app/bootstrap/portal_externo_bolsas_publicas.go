package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"vec-diputacion-granada/config"
	bolsapostgrespublico "vec-diputacion-granada/internal/modules/bolsa/adapters/postgrespublico"
	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/publico/httpapi"
)

// ErrBolsasPublicasPortalExternoNoDisponibles no expone la conexión ni el
// manifiesto de la proyección pública en los errores de arranque.
var ErrBolsasPublicasPortalExternoNoDisponibles = errors.New("bootstrap: bolsas publicas del portal externo no disponibles")

// nuevasBolsasPublicasPortalExterno monta B10 exclusivamente desde la base de
// proyección pública de Bolsa. El llamante aporta el DSN del LOGIN externo
// dedicado y registra el cierre con el servidor. Sin activación explícita no
// instala la ruta; una configuración parcial impide el arranque.
func nuevasBolsasPublicasPortalExterno(ctx context.Context, cfg config.Config, dsnExterno string) (http.Handler, func(), error) {
	nada := func() {}
	if ctx == nil || ctx.Err() != nil {
		return nil, nada, ErrBolsasPublicasPortalExternoNoDisponibles
	}
	cfg = cfg.Normalize()
	dsnExterno = strings.TrimSpace(dsnExterno)
	if cfg.BolsaPublicaPostgreSQL.Validar() == nil {
		return nil, nada, ErrBolsasPublicasPortalExternoNoDisponibles
	}
	if dsnExterno == "" && cfg.BolsaPublicaManifiestoSHA256 == "" {
		return nil, nada, nil
	}
	if dsnExterno == "" ||
		config.ValidarHuellaManifiestoPublico(cfg.BolsaPublicaManifiestoSHA256) != nil ||
		cfg.BolsaCategoriesCatalogID == "" || cfg.BolsaCategoriesVersion < 1 ||
		cfg.BolsaCategoriesSHA256 == "" || cfg.BolsaCategoriesPublicProjectionSHA256 == "" {
		return nil, nada, ErrBolsasPublicasPortalExternoNoDisponibles
	}
	fuente, err := bolsapostgrespublico.Abrir(ctx, dsnExterno,
		cfg.BolsaCategoriesCatalogID, cfg.BolsaCategoriesVersion,
		cfg.BolsaCategoriesSHA256, cfg.BolsaCategoriesPublicProjectionSHA256,
		cfg.BolsaPublicaManifiestoSHA256)
	if err != nil {
		return nil, nada, ErrBolsasPublicasPortalExternoNoDisponibles
	}
	if err := fuente.ValidarConfiguracionPublica(ctx, time.Now().UTC()); err != nil {
		fuente.Cerrar()
		return nil, nada, ErrBolsasPublicasPortalExternoNoDisponibles
	}
	manejador, err := bolsahttp.NuevoManejadorBolsasPublicas(fuente)
	if err != nil {
		fuente.Cerrar()
		return nil, nada, ErrBolsasPublicasPortalExternoNoDisponibles
	}
	return manejador, fuente.Cerrar, nil
}
