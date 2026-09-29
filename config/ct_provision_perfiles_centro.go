package config

import (
	"regexp"
	"strings"
)

// EnvCTProvisionPerfilesCentroAprobacion es la referencia de la aprobación
// con la que el operador autoriza, en un arranque concreto, la provisión del
// perfil general de las personas del centro (peticiones del centro). Solo se
// usa para dejar ese perfil en su asignación del centro cuando el binario
// anterior la había estrechado por petición o el rol del centro ha cambiado;
// nunca actúa sobre una asignación revocada, restringida o retirada. Vacía,
// el arranque solo lee.
const EnvCTProvisionPerfilesCentroAprobacion = "VEC_CT_PROVISION_PERFILES_CENTRO_APROBACION"

// EnvCTProvisionPerfilesCentroPreimagenes son las huellas SHA-256, separadas
// por comas, de las asignaciones vigentes que el operador aprueba sustituir.
// El arranque las muestra en el aviso «pendiente_provision». Ligan la
// aprobación a ese estado exacto: dejar las variables puestas no aprueba
// ningún estado posterior.
const EnvCTProvisionPerfilesCentroPreimagenes = "VEC_CT_PROVISION_PERFILES_CENTRO_PREIMAGENES"

const maximoPreimagenesProvisionPerfilesCentro = 64

var (
	referenciaAprobacionProvisionPerfilesCentro = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{7,127}$`)
	huellaPreimagenProvisionPerfilesCentro      = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

// CTProvisionPerfilesCentroAprobacionRef devuelve la referencia de
// aprobación solo en el perfil de desarrollo, si tiene forma válida y si hay
// al menos una preimagen aprobada bien formada; en otro caso, vacía (el
// arranque solo lee).
func (c Config) CTProvisionPerfilesCentroAprobacionRef() string {
	c = c.Normalize()
	if !c.DevelopmentEnabledByDoubleKey() || !referenciaAprobacionProvisionPerfilesCentro.MatchString(c.CTAprobacionPerfilesCentro) ||
		len(c.CTProvisionPerfilesCentroPreimagenes()) == 0 {
		return ""
	}
	return c.CTAprobacionPerfilesCentro
}

// CTProvisionPerfilesCentroPreimagenes devuelve las huellas aprobadas. Una
// sola entrada mal formada invalida la lista entera.
func (c Config) CTProvisionPerfilesCentroPreimagenes() map[string]bool {
	c = c.Normalize()
	if !c.DevelopmentEnabledByDoubleKey() || c.CTPreimagenesPerfilesCentro == "" {
		return nil
	}
	partes := strings.Split(c.CTPreimagenesPerfilesCentro, ",")
	if len(partes) > maximoPreimagenesProvisionPerfilesCentro {
		return nil
	}
	huellas := make(map[string]bool, len(partes))
	for _, parte := range partes {
		parte = strings.TrimSpace(parte)
		if !huellaPreimagenProvisionPerfilesCentro.MatchString(parte) {
			return nil
		}
		huellas[parte] = true
	}
	return huellas
}
