package config

// Perfil residual de la presentación RRHH de julio. Su binario y su artefacto se
// retiraron el 06/10/2026; solo lo usan las pruebas de la API heredada de
// demostración (nuevaAPIDemo), que no tiene otro perfil no productivo. Se
// retirará junto con esa API. Nunca se selecciona por omision.
const (
	ExecutionProfileRRHHPresentation = "presentacion_rrhh"

	RRHHPresentationGuardOneAcknowledgement = "ACEPTO_MODO_PRESENTACION_RRHH_NO_AUTORITATIVO"
	RRHHPresentationGuardTwoAcknowledgement = "CONFIRMO_DATOS_SINTETICOS_SIN_VALIDEZ_ADMINISTRATIVA"
)

// RRHHPresentationEnabledByDoubleGuard exige el perfil, el selector y dos
// reconocimientos literales independientes. No se usa un build tag como
// barrera: la raiz de composicion vuelve a validar todas las condiciones.
func (c Config) RRHHPresentationEnabledByDoubleGuard() bool {
	c = c.Normalize()
	return c.ExecutionProfile == ExecutionProfileRRHHPresentation &&
		c.RRHHPresentationEnabled &&
		c.RRHHPresentationGuardOne == RRHHPresentationGuardOneAcknowledgement &&
		c.RRHHPresentationGuardTwo == RRHHPresentationGuardTwoAcknowledgement
}

// HasRRHHPresentationSelectors permite que los binarios normales fallen
// cerrados aunque solo se haya configurado una de las llaves por error.
func (c Config) HasRRHHPresentationSelectors() bool {
	c = c.Normalize()
	return c.ExecutionProfile == ExecutionProfileRRHHPresentation ||
		c.RRHHPresentationEnabled || c.RRHHPresentationGuardOne != "" ||
		c.RRHHPresentationGuardTwo != ""
}
