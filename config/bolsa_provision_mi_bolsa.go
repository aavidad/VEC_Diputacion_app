package config

import "regexp"

// EnvBolsaProvisionMiBolsaAprobacion es la referencia de la aprobación con la
// que el operador autoriza, en un arranque concreto, que Mi Bolsa vuelva a
// dejar el permiso del candidato en la forma que compone este binario. Solo
// sirve cuando el permiso vigente es uno que puso el propio arranque de Mi
// Bolsa con otra forma de rol (por ejemplo, otro binario con otras acciones
// del portal); nunca actúa sobre un permiso revocado, con los ámbitos o la
// vigencia recortados, con el rol retirado o puesto por otro acto. Un rol de
// Mi Bolsa con menos acciones puesto con los actos de este mismo circuito no
// se distingue de «otra forma»: por eso la aprobación va ligada a la huella
// exacta que el operador ha revisado.
// Vacía, el arranque solo publica el permiso inicial o comprueba el vigente.
const EnvBolsaProvisionMiBolsaAprobacion = "VEC_BOLSA_PROVISION_MI_BOLSA_APROBACION"

// EnvBolsaProvisionMiBolsaPreimagen es la huella SHA-256 de la asignación
// vigente que el operador aprueba sustituir. El arranque la muestra en el
// aviso «pendiente_provision». Liga la aprobación a ese estado exacto:
// dejar las variables puestas no aprueba ningún estado posterior.
const EnvBolsaProvisionMiBolsaPreimagen = "VEC_BOLSA_PROVISION_MI_BOLSA_PREIMAGEN"

var (
	referenciaAprobacionProvisionMiBolsa = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{7,127}$`)
	huellaPreimagenProvisionMiBolsa      = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

// BolsaProvisionMiBolsa devuelve la referencia de aprobación y la huella
// aprobada solo en el perfil de desarrollo con doble llave y si ambas tienen
// forma válida; en otro caso, las dos vacías (el arranque no restaura nada).
func (c Config) BolsaProvisionMiBolsa() (aprobacion, preimagen string) {
	c = c.Normalize()
	if !c.DevelopmentEnabledByDoubleKey() ||
		!referenciaAprobacionProvisionMiBolsa.MatchString(c.BolsaAprobacionProvisionMiBolsa) ||
		!huellaPreimagenProvisionMiBolsa.MatchString(c.BolsaPreimagenProvisionMiBolsa) {
		return "", ""
	}
	return c.BolsaAprobacionProvisionMiBolsa, c.BolsaPreimagenProvisionMiBolsa
}
