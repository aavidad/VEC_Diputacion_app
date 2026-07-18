package config

// Este perfil nunca se selecciona por omision y no comparte la composicion de
// desarrollo. Su unico fin es servir el artefacto desechable para la revision
// funcional de RRHH con datos sinteticos.
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
