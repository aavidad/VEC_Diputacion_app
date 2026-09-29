package bootstrap

import (
	"errors"
	"os"
	"os/user"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/app/separacionportales"
)

var (
	// ErrPortalSeparadoFueraDesarrollo: la separación de procesos solo se
	// admite en la composición que hoy arranca (doble llave de desarrollo).
	ErrPortalSeparadoFueraDesarrollo = errors.New("bootstrap: la separacion de portales solo se admite en la composicion de desarrollo")
	// ErrComposicionPortalExternoPendiente: el proceso externo ya tiene sus
	// comprobaciones, pero su composición propia (Área personal sin el
	// material de RRHH) es la siguiente minitarea. Hasta entonces no arranca
	// en lugar de componer rutas internas.
	ErrComposicionPortalExternoPendiente = errors.New("bootstrap: la composicion propia del portal externo aun no existe")
)

// portalProcesoConfigurado interpreta VEC_PORTAL_PROCESO sin tocar nada más.
func portalProcesoConfigurado(cfg config.Config) (separacionportales.Portal, error) {
	return separacionportales.Parsear(cfg.PortalProceso)
}

// portalProcesoSeparado indica si la configuración pide un proceso por
// portal. Un valor no válido cuenta como separado para que ninguna
// relajación dependa de una errata: la comprobación posterior lo rechaza.
func portalProcesoSeparado(cfg config.Config) bool {
	portal, err := portalProcesoConfigurado(cfg)
	return err != nil || portal.Separado()
}

// rechazarPortalSeparadoFueraDesarrollo se aplica antes de elegir la
// composición, para que un portal mal escrito o fuera de desarrollo no llegue
// a la composición heredada.
func rechazarPortalSeparadoFueraDesarrollo(cfg config.Config) error {
	portal, err := portalProcesoConfigurado(cfg)
	if err != nil {
		return err
	}
	if portal.Separado() && !cfg.DevelopmentEnabledByDoubleKey() {
		return ErrPortalSeparadoFueraDesarrollo
	}
	return nil
}

// comprobarSeparacionPortalProceso se ejecuta antes de leer material o abrir
// conexiones. Con un portal separado exige que el entorno y el material del
// proceso no contengan credenciales ni claves del otro portal. En el portal
// combinado solo impide usar un material ya separado.
func comprobarSeparacionPortalProceso(cfg config.Config, entorno separacionportales.Entorno) (separacionportales.Portal, error) {
	if err := rechazarPortalSeparadoFueraDesarrollo(cfg); err != nil {
		return separacionportales.PortalCombinado, err
	}
	portal, _ := portalProcesoConfigurado(cfg)
	if portal.Separado() {
		entorno.DirectorioMaterial = cfg.DevelopmentMaterialDir
		if err := separacionportales.ComprobarEntorno(portal, entorno); err != nil {
			return portal, err
		}
	}
	if err := separacionportales.ComprobarMaterial(portal, cfg.DevelopmentMaterialDir); err != nil {
		return portal, err
	}
	return portal, nil
}

// entornoProcesoActual es la instantánea real del proceso: sus variables y el
// directorio personal en el que pgx busca credenciales por defecto.
func entornoProcesoActual() separacionportales.Entorno {
	return separacionportales.Entorno{Variables: os.Environ(), DirectorioPersonal: directorioPersonalPostgreSQL()}
}

// directorioPersonalPostgreSQL devuelve el mismo directorio en el que pgx
// busca .pgpass y la clave de cliente: el del usuario del sistema, no $HOME.
func directorioPersonalPostgreSQL() string {
	actual, err := user.Current()
	if err != nil {
		return ""
	}
	return actual.HomeDir
}
