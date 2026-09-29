package config

import "regexp"

// EnvCTProvisionPerfilesCentroAprobacion es la referencia de la aprobación
// con la que el operador autoriza, en un arranque concreto, la provisión del
// perfil general de las personas del centro (peticiones del centro). Solo se
// usa para dejar ese perfil en su asignación del centro cuando el binario
// anterior la había estrechado por petición; nunca actúa sobre una asignación
// revocada, restringida o retirada. Vacía, el arranque solo lee.
const EnvCTProvisionPerfilesCentroAprobacion = "VEC_CT_PROVISION_PERFILES_CENTRO_APROBACION"

var referenciaAprobacionProvisionPerfilesCentro = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{7,127}$`)

// CTProvisionPerfilesCentroAprobacionRef devuelve la referencia de
// aprobación solo en el perfil de desarrollo y si tiene forma válida; en
// cualquier otro caso, vacía (el arranque solo lee).
func (c Config) CTProvisionPerfilesCentroAprobacionRef() string {
	c = c.Normalize()
	if !c.DevelopmentEnabledByDoubleKey() || !referenciaAprobacionProvisionPerfilesCentro.MatchString(c.CTAprobacionPerfilesCentro) {
		return ""
	}
	return c.CTAprobacionPerfilesCentro
}
