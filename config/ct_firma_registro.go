package config

import "errors"

// EnvCTFirmaRegistroEnabled monta el registro de firmas de prueba de los
// borradores de Contratación temporal (AD3-85 y CT118 instaladas antes del
// binario) y publica su consumidor V3. Es un selector de despliegue, no un
// permiso: sólo admite "true" y "false"; la ausencia equivale a apagado.
// Exige la doble llave de desarrollo y el circuito de firma de ejemplo. La
// verificación de la firma sigue gobernada por VEC_FIRMA_VERIFICACION_ENABLED:
// apagada, toda firma se rechaza.
const EnvCTFirmaRegistroEnabled = "VEC_CT_FIRMA_REGISTRO_ENABLED"

var (
	ErrConfiguracionCTFirmaRegistroSelector   = errors.New("config: selector del registro de firmas de CT invalido")
	ErrConfiguracionCTFirmaRegistroActivacion = errors.New("config: registro de firmas de CT fuera del perfil de desarrollo o sin circuito de firma")
)

// CTFirmaRegistroDesarrolloActivo valida la activación sin abrir ficheros.
func (c Config) CTFirmaRegistroDesarrolloActivo() (bool, error) {
	c = c.Normalize()
	switch c.CTFirmaRegistroEnabled {
	case "", "false":
		return false, nil
	case "true":
		if !c.DevelopmentEnabledByDoubleKey() || c.ReglasEjemplo.normalizar().CTCircuitoFirmaSourcePath == "" {
			return false, ErrConfiguracionCTFirmaRegistroActivacion
		}
		return true, nil
	default:
		return false, ErrConfiguracionCTFirmaRegistroSelector
	}
}
