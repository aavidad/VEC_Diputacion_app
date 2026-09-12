package bootstrap

import (
	"context"
	"net/http"
	"time"
	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/app/server"
)

// NewHTTPServerAdministracionDesarrolloWithConfig compone una superficie
// administrativa propia, sin registrar sus rutas en el servidor de RRHH.
// El cierre devuelto pertenece al ciclo conjunto y es idempotente.
func NewHTTPServerAdministracionDesarrolloWithConfig(cfg config.Config, composicion *ComposicionSeguridadDesarrollo) (*http.Server, func(), error) {
	cerrarVacio := func() {}
	if !cfg.TransporteAdministracion.Configurada() {
		if cfg.AdministracionPostgreSQL.Configurada() || (composicion != nil && composicion.identidadAdministracion != nil) {
			return nil, cerrarVacio, ErrConfiguracionCorreoAdministracionNoDisponible
		}
		return nil, cerrarVacio, nil
	}
	if composicion == nil || composicion.identidadAdministracion == nil {
		return nil, cerrarVacio, ErrConfiguracionCorreoAdministracionNoDisponible
	}
	cfgAdministracion, err := cfg.TransporteAdministracion.ConfiguracionServidor(cfg)
	if err != nil {
		return nil, cerrarVacio, ErrConfiguracionCorreoAdministracionNoDisponible
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 45*time.Second)
	ruta, autoridad, cerrar, err := nuevaRutaAdministracionCorreoDesarrollo(ctx, cfgAdministracion, composicion, composicion.identidadAdministracion)
	cancelar()
	if err != nil || ruta == nil || autoridad == nil || cerrar == nil {
		return nil, cerrarVacio, ErrConfiguracionCorreoAdministracionNoDisponible
	}
	completa := false
	defer func() {
		if !completa {
			cerrar()
		}
	}()
	servidor, err := server.NewHTTPServerAdministracion(cfg, ruta.Manejador)
	if err != nil {
		return nil, cerrarVacio, ErrConfiguracionCorreoAdministracionNoDisponible
	}
	if err = autoridad.vincularServidor(servidor); err != nil {
		return nil, cerrarVacio, ErrConfiguracionCorreoAdministracionNoDisponible
	}
	completa = true
	return servidor, cerrar, nil
}
