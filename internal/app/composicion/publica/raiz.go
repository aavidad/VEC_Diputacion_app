package publica

import (
	"errors"
	"net/http"

	"vec-diputacion-granada/config"
)

var (
	ErrComposicionProductivaNoDisponible = errors.New("composicion publica: proveedor autoritativo no disponible")
	ErrPerfilEjecucionDesconocido        = errors.New("composicion publica: perfil de ejecucion desconocido")
	ErrModoAutenticacionDesconocido      = errors.New("composicion publica: modo de autenticacion desconocido")
	ErrAutenticacionNoAdmitida           = errors.New("composicion publica: la superficie anonima exige autenticacion deshabilitada")
	ErrActivacionDesarrolloInvalida      = errors.New("composicion publica: selectores de desarrollo prohibidos")
	ErrSelectoresPresentacionProhibidos  = errors.New("composicion publica: selectores de presentacion prohibidos")
)

// NuevoServidor construira la superficie publica productiva cuando exista un
// adaptador publico autoritativo. C1 establece la frontera fisica, pero no
// convierte los ficheros de presentacion en autoridad: hasta C3 falla antes de
// construir una API o abrir un socket.
func NuevoServidor(cfg Configuracion) (*http.Server, error) {
	cfg = cfg.normalizar()
	if err := validarSelectoresConocidos(cfg); err != nil {
		return nil, err
	}
	if cfg.AutenticacionSolicitada != config.AuthModeDisabled {
		return nil, ErrAutenticacionNoAdmitida
	}
	if tieneSelectoresPresentacion(cfg) {
		return nil, ErrSelectoresPresentacionProhibidos
	}
	if cfg.PerfilEjecucion == config.ExecutionProfileDevelopment ||
		cfg.GuardaDesarrollo != "" || cfg.MaterialDesarrollo != "" {
		return nil, ErrActivacionDesarrolloInvalida
	}
	return nil, ErrComposicionProductivaNoDisponible
}

func validarSelectoresConocidos(cfg Configuracion) error {
	switch cfg.PerfilEjecucion {
	case config.ExecutionProfileProduction, config.ExecutionProfileDevelopment, config.ExecutionProfileRRHHPresentation:
	default:
		return ErrPerfilEjecucionDesconocido
	}
	switch cfg.AutenticacionSolicitada {
	case config.AuthModeDisabled, config.AuthModeFake, config.AuthModeTrustedHeaders, config.AuthModeDevelopment:
	default:
		return ErrModoAutenticacionDesconocido
	}
	return nil
}

func tieneSelectoresPresentacion(cfg Configuracion) bool {
	return cfg.PerfilEjecucion == config.ExecutionProfileRRHHPresentation ||
		cfg.PresentacionHabilitada || cfg.GuardaPresentacionUno != "" ||
		cfg.GuardaPresentacionDos != ""
}
