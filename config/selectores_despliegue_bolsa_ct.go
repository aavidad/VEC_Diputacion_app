package config

import (
	"errors"
	"fmt"
)

// Selectores de despliegue de capacidades que dependen de migraciones de
// autorización y de dominio que se instalan aparte del binario. Si el binario
// llega antes que ellas, componer la capacidad fallaría al arrancar; por eso
// no basta con que existan sus catálogos: hay que pedirla expresamente. Son
// selectores, no permisos: solo admiten "true" y "false" y la ausencia
// equivale a apagado. Exigen la doble llave de desarrollo.
const (
	// EnvBolsaPortalCandidatoEnabled compone las acciones propias del
	// candidato en «Mi bolsa»: pausa, reactivación y respuesta al
	// llamamiento (AD3-84 y Bolsa 000030), disposición a ofertas publicadas
	// (AD3-84 y Bolsa 000029) y confirmación del contacto propio (AD3-86 y
	// Bolsa 000040). Con el selector encendido el arranque comprueba que
	// existen sus fachadas y se detiene nombrando la primera que falte.
	EnvBolsaPortalCandidatoEnabled = "VEC_BOLSA_PORTAL_CANDIDATO_ENABLED"
	// EnvCTSeguimientoCeseEnabled compone cese, cierre y modificación tras
	// el nombramiento de Contratación temporal; requiere AD3-82, AD3-83,
	// CT115 y CT116 instaladas.
	EnvCTSeguimientoCeseEnabled = "VEC_CT_SEGUIMIENTO_CESE_ENABLED"
	// EnvCTCancelacionEnabled compone la cancelación del expediente antes de
	// la fiscalización (RRHH); requiere AD3-87 y CT122 instaladas.
	EnvCTCancelacionEnabled = "VEC_CT_CANCELACION_ENABLED"
)

var (
	ErrConfiguracionBolsaPortalCandidatoSelector   = errors.New("config: selector del portal del candidato de Bolsa invalido")
	ErrConfiguracionBolsaPortalCandidatoActivacion = errors.New("config: portal del candidato de Bolsa fuera del perfil de desarrollo o sin sus catalogos")
	ErrConfiguracionCTSeguimientoCeseSelector      = errors.New("config: selector del seguimiento de cese de CT invalido")
	ErrConfiguracionCTSeguimientoCeseActivacion    = errors.New("config: seguimiento de cese de CT fuera del perfil de desarrollo o sin sus catalogos")
	ErrConfiguracionCTCancelacionSelector          = errors.New("config: selector de la cancelacion de expedientes de CT invalido")
	ErrConfiguracionCTCancelacionActivacion        = errors.New("config: cancelacion de expedientes de CT fuera del perfil de desarrollo o sin sus catalogos")
)

// BolsaPortalCandidatoDesarrolloActivo valida el selector del portal del
// candidato sin abrir ficheros. Encendido exige también el catálogo de reglas
// de Bolsa: pedirlo sin él detiene el arranque en lugar de dejar las rutas sin
// montar en silencio.
func (c Config) BolsaPortalCandidatoDesarrolloActivo() (bool, error) {
	c = c.Normalize()
	reglas := c.ReglasEjemplo.normalizar()
	return selectorDesarrolloActivo(c, c.BolsaPortalCandidatoEnabled,
		ErrConfiguracionBolsaPortalCandidatoSelector, ErrConfiguracionBolsaPortalCandidatoActivacion,
		catalogoRequerido{EnvBolsaReglasSourcePath, reglas.BolsaSourcePath})
}

// CTSeguimientoCeseDesarrolloActivo valida el selector del seguimiento de
// cese sin abrir ficheros. Encendido exige los tres catálogos que usa (reglas
// de CT, causas de cese y motivos de rectificación): pedirlo sin alguno
// detiene el arranque nombrando la variable que falta.
func (c Config) CTSeguimientoCeseDesarrolloActivo() (bool, error) {
	c = c.Normalize()
	reglas := c.ReglasEjemplo.normalizar()
	return selectorDesarrolloActivo(c, c.CTSeguimientoCeseEnabled,
		ErrConfiguracionCTSeguimientoCeseSelector, ErrConfiguracionCTSeguimientoCeseActivacion,
		catalogoRequerido{EnvCTReglasSourcePath, reglas.CTSourcePath},
		catalogoRequerido{EnvCTCausasCeseSourcePath, reglas.CausasCeseSourcePath},
		catalogoRequerido{EnvCTAnalisisMotivosSourcePath, c.CTAnalisisMotivosSourcePath})
}

// CTCancelacionDesarrolloActivo valida el selector de la cancelación del
// expediente sin abrir ficheros. Encendido exige el catálogo de reglas de CT
// (regla c20: fases admitidas) y el de motivos de cancelación.
func (c Config) CTCancelacionDesarrolloActivo() (bool, error) {
	c = c.Normalize()
	reglas := c.ReglasEjemplo.normalizar()
	return selectorDesarrolloActivo(c, c.CTCancelacionEnabled,
		ErrConfiguracionCTCancelacionSelector, ErrConfiguracionCTCancelacionActivacion,
		catalogoRequerido{EnvCTReglasSourcePath, reglas.CTSourcePath},
		catalogoRequerido{EnvCTMotivosCancelacionSourcePath, reglas.MotivosCancelacionSourcePath})
}

// catalogoRequerido nombra la variable de entorno de un catálogo que una
// capacidad encendida necesita, con la ruta ya normalizada.
type catalogoRequerido struct {
	variable, ruta string
}

func selectorDesarrolloActivo(c Config, valor string, errSelector, errActivacion error, requeridos ...catalogoRequerido) (bool, error) {
	switch valor {
	case "", "false":
		return false, nil
	case "true":
		if !c.DevelopmentEnabledByDoubleKey() {
			return false, errActivacion
		}
		for _, requerido := range requeridos {
			if requerido.ruta == "" {
				return false, fmt.Errorf("%w: falta %s", errActivacion, requerido.variable)
			}
		}
		return true, nil
	default:
		return false, errSelector
	}
}
