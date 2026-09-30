package bootstrap

import (
	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/app/separacionportales"
)

// comprobarSeparacionPortalConEntorno permite comprobar el fallo del lector
// sin cambiar la instantánea global ni la autoridad de separación. La raíz
// real siempre aporta entornoProcesoActual.
func comprobarSeparacionPortalConEntorno(cfg config.Config, resolver func() (separacionportales.Entorno, error)) (separacionportales.Portal, error) {
	entorno, err := resolver()
	if err != nil && portalProcesoSeparado(cfg) {
		return separacionportales.PortalCombinado, err
	}
	return comprobarSeparacionPortalProceso(cfg, entorno)
}
