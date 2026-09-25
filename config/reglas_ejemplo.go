package config

import (
	"errors"
	"strings"
)

// Las reglas de ejemplo son catálogos inventados para desarrollo y para la
// presentación a RRHH; se borran al pasar a producción, como los datos de
// ejemplo. Solo se componen con el perfil de desarrollo y su doble llave.
// Los motivos de rectificación del análisis ya tienen su propia variable,
// EnvCTAnalisisMotivosSourcePath, sujeta a la misma restricción.
const (
	EnvBolsaReglasSourcePath = "VEC_BOLSA_REGLAS_SOURCE_PATH"
	EnvCTReglasSourcePath    = "VEC_CT_REGLAS_SOURCE_PATH"
	// EnvBolsaRolesSegregacionSourcePath declara qué operaciones de Bolsa
	// exigen una segunda persona; sin él rige la exclusión como hasta ahora.
	EnvBolsaRolesSegregacionSourcePath = "VEC_BOLSA_ROLES_SEGREGACION_SOURCE_PATH"
)

// ErrConfiguracionReglasEjemploFueraDesarrollo impide arrancar si un
// catálogo de ejemplo se declara fuera del perfil de desarrollo: producción
// no admite reglas sin aprobar, ni siquiera ignorándolas.
var ErrConfiguracionReglasEjemploFueraDesarrollo = errors.New("config: catalogo de reglas de ejemplo fuera del perfil de desarrollo")

// ConfiguracionReglasEjemplo contiene rutas locales de paquetes DEMO.
type ConfiguracionReglasEjemplo struct {
	BolsaSourcePath                 string
	CTSourcePath                    string
	BolsaRolesSegregacionSourcePath string
}

func cargarConfiguracionReglasEjemplo() ConfiguracionReglasEjemplo {
	return ConfiguracionReglasEjemplo{
		BolsaSourcePath:                 envFirst(EnvBolsaReglasSourcePath),
		CTSourcePath:                    envFirst(EnvCTReglasSourcePath),
		BolsaRolesSegregacionSourcePath: envFirst(EnvBolsaRolesSegregacionSourcePath),
	}
}

func (c ConfiguracionReglasEjemplo) normalizar() ConfiguracionReglasEjemplo {
	c.BolsaSourcePath = strings.TrimSpace(c.BolsaSourcePath)
	c.CTSourcePath = strings.TrimSpace(c.CTSourcePath)
	c.BolsaRolesSegregacionSourcePath = strings.TrimSpace(c.BolsaRolesSegregacionSourcePath)
	return c
}

// Configurada indica si se ha declarado algún catálogo de reglas.
func (c ConfiguracionReglasEjemplo) Configurada() bool {
	c = c.normalizar()
	return c.BolsaSourcePath != "" || c.CTSourcePath != "" || c.BolsaRolesSegregacionSourcePath != ""
}

// ReglasEjemploDesarrollo valida la activación sin abrir ficheros. Devuelve
// la configuración normalizada y si debe componerse. Cualquier catálogo de
// ejemplo, incluidos los motivos de rectificación, fuera de la doble llave de
// desarrollo impide el arranque.
func (c Config) ReglasEjemploDesarrollo() (ConfiguracionReglasEjemplo, bool, error) {
	c = c.Normalize()
	reglas := c.ReglasEjemplo.normalizar()
	if !reglas.Configurada() && c.CTAnalisisMotivosSourcePath == "" {
		return ConfiguracionReglasEjemplo{}, false, nil
	}
	if !c.DevelopmentEnabledByDoubleKey() {
		return ConfiguracionReglasEjemplo{}, false, ErrConfiguracionReglasEjemploFueraDesarrollo
	}
	return reglas, reglas.Configurada(), nil
}
