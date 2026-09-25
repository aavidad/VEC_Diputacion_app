package config

import "errors"

// Selectores de despliegue de capacidades que dependen de migraciones de
// autorización y de dominio que se instalan aparte del binario. Si el binario
// llega antes que ellas, componer la capacidad fallaría al arrancar; por eso
// no basta con que existan sus catálogos: hay que pedirla expresamente. Son
// selectores, no permisos: solo admiten "true" y "false" y la ausencia
// equivale a apagado. Exigen la doble llave de desarrollo.
const (
	// EnvBolsaPortalCandidatoEnabled compone las acciones propias del
	// candidato en «Mi bolsa» (pausa, reactivación y respuesta al
	// llamamiento); requiere AD3-84 y Bolsa 000030 instaladas.
	EnvBolsaPortalCandidatoEnabled = "VEC_BOLSA_PORTAL_CANDIDATO_ENABLED"
	// EnvCTSeguimientoCeseEnabled compone cese, cierre y modificación tras
	// el nombramiento de Contratación temporal; requiere AD3-82, AD3-83,
	// CT115 y CT116 instaladas.
	EnvCTSeguimientoCeseEnabled = "VEC_CT_SEGUIMIENTO_CESE_ENABLED"
)

var (
	ErrConfiguracionBolsaPortalCandidatoSelector   = errors.New("config: selector del portal del candidato de Bolsa invalido")
	ErrConfiguracionBolsaPortalCandidatoActivacion = errors.New("config: portal del candidato de Bolsa fuera del perfil de desarrollo")
	ErrConfiguracionCTSeguimientoCeseSelector      = errors.New("config: selector del seguimiento de cese de CT invalido")
	ErrConfiguracionCTSeguimientoCeseActivacion    = errors.New("config: seguimiento de cese de CT fuera del perfil de desarrollo")
)

// BolsaPortalCandidatoDesarrolloActivo valida el selector del portal del
// candidato sin abrir ficheros.
func (c Config) BolsaPortalCandidatoDesarrolloActivo() (bool, error) {
	c = c.Normalize()
	return selectorDesarrolloActivo(c, c.BolsaPortalCandidatoEnabled,
		ErrConfiguracionBolsaPortalCandidatoSelector, ErrConfiguracionBolsaPortalCandidatoActivacion)
}

// CTSeguimientoCeseDesarrolloActivo valida el selector del seguimiento de
// cese sin abrir ficheros.
func (c Config) CTSeguimientoCeseDesarrolloActivo() (bool, error) {
	c = c.Normalize()
	return selectorDesarrolloActivo(c, c.CTSeguimientoCeseEnabled,
		ErrConfiguracionCTSeguimientoCeseSelector, ErrConfiguracionCTSeguimientoCeseActivacion)
}

func selectorDesarrolloActivo(c Config, valor string, errSelector, errActivacion error) (bool, error) {
	switch valor {
	case "", "false":
		return false, nil
	case "true":
		if !c.DevelopmentEnabledByDoubleKey() {
			return false, errActivacion
		}
		return true, nil
	default:
		return false, errSelector
	}
}
