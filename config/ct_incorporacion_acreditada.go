package config

import (
	"errors"
	"fmt"
)

// EnvCTIncorporacionAcreditadaEnabled compone la incorporación acreditada de
// Contratación temporal (dudas 11 y 12 de RRHH): confirmación de GINPIX por
// RRHH, cierre con el número de esa confirmación y confirmación de la
// incorporación por el centro. Requiere CT 000124 y AD3-88 instaladas y el
// seguimiento de cese encendido; el arranque comprueba las migraciones y se
// detiene nombrando la que falte.
const EnvCTIncorporacionAcreditadaEnabled = "VEC_CT_INCORPORACION_ACREDITADA_ENABLED"

var (
	ErrConfiguracionCTIncorporacionAcreditadaSelector   = errors.New("config: selector de la incorporacion acreditada de CT invalido")
	ErrConfiguracionCTIncorporacionAcreditadaActivacion = errors.New("config: incorporacion acreditada de CT fuera del perfil de desarrollo o sin el seguimiento de cese")
)

// CTIncorporacionAcreditadaDesarrolloActivo valida el selector sin abrir
// ficheros. Encendido exige la doble llave, el catálogo de reglas de CT y el
// seguimiento de cese, sobre el que se compone.
func (c Config) CTIncorporacionAcreditadaDesarrolloActivo() (bool, error) {
	c = c.Normalize()
	reglas := c.ReglasEjemplo.normalizar()
	activo, err := selectorDesarrolloActivo(c, c.CTIncorporacionAcreditadaEnabled,
		ErrConfiguracionCTIncorporacionAcreditadaSelector, ErrConfiguracionCTIncorporacionAcreditadaActivacion,
		catalogoRequerido{EnvCTReglasSourcePath, reglas.CTSourcePath})
	if err != nil || !activo {
		return false, err
	}
	if cese, errCese := c.CTSeguimientoCeseDesarrolloActivo(); errCese != nil || !cese {
		return false, fmt.Errorf("%w: falta %s", ErrConfiguracionCTIncorporacionAcreditadaActivacion, EnvCTSeguimientoCeseEnabled)
	}
	return true, nil
}
