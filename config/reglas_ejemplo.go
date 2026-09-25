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
	// EnvCTRetribucionesSourcePath apunta al catálogo ct.retribuciones con el
	// que se estima el coste del nombramiento. Sin él, el coste queda «sin
	// calcular».
	EnvCTRetribucionesSourcePath = "VEC_CT_RETRIBUCIONES_SOURCE_PATH"
	// EnvCTPlantillasSourcePath sustituye el catálogo de plantillas de los
	// documentos de Contratación temporal que trae el repositorio.
	EnvCTPlantillasSourcePath = "VEC_CT_PLANTILLAS_SOURCE_PATH"
	// EnvCTCircuitoFirmaSourcePath declara el circuito de firma de ejemplo de
	// los documentos de Contratación temporal.
	EnvCTCircuitoFirmaSourcePath = "VEC_CT_CIRCUITO_FIRMA_SOURCE_PATH"
)

// ErrConfiguracionReglasEjemploFueraDesarrollo impide arrancar si un
// catálogo de ejemplo se declara fuera del perfil de desarrollo: producción
// no admite reglas sin aprobar, ni siquiera ignorándolas.
var ErrConfiguracionReglasEjemploFueraDesarrollo = errors.New("config: catalogo de reglas de ejemplo fuera del perfil de desarrollo")

// ErrConfiguracionReglasEjemploSinComposicion impide arrancar una raíz que no
// compone reglas de ejemplo cuando se le declara alguna, aunque sea con la
// doble llave de desarrollo: ignorarla ocultaría un error de despliegue.
var ErrConfiguracionReglasEjemploSinComposicion = errors.New("config: catalogo de reglas de ejemplo en una raiz que no lo compone")

// ConfiguracionReglasEjemplo contiene rutas locales de paquetes DEMO.
type ConfiguracionReglasEjemplo struct {
	BolsaSourcePath                 string
	CTSourcePath                    string
	BolsaRolesSegregacionSourcePath string
	CTRetribucionesSourcePath       string
	CTPlantillasSourcePath          string
	CTCircuitoFirmaSourcePath       string
}

func cargarConfiguracionReglasEjemplo() ConfiguracionReglasEjemplo {
	return ConfiguracionReglasEjemplo{
		BolsaSourcePath:                 envFirst(EnvBolsaReglasSourcePath),
		CTSourcePath:                    envFirst(EnvCTReglasSourcePath),
		BolsaRolesSegregacionSourcePath: envFirst(EnvBolsaRolesSegregacionSourcePath),
		CTRetribucionesSourcePath:       envFirst(EnvCTRetribucionesSourcePath),
		CTPlantillasSourcePath:          envFirst(EnvCTPlantillasSourcePath),
		CTCircuitoFirmaSourcePath:       envFirst(EnvCTCircuitoFirmaSourcePath),
	}
}

func (c ConfiguracionReglasEjemplo) normalizar() ConfiguracionReglasEjemplo {
	c.BolsaSourcePath = strings.TrimSpace(c.BolsaSourcePath)
	c.CTSourcePath = strings.TrimSpace(c.CTSourcePath)
	c.BolsaRolesSegregacionSourcePath = strings.TrimSpace(c.BolsaRolesSegregacionSourcePath)
	c.CTRetribucionesSourcePath = strings.TrimSpace(c.CTRetribucionesSourcePath)
	c.CTPlantillasSourcePath = strings.TrimSpace(c.CTPlantillasSourcePath)
	c.CTCircuitoFirmaSourcePath = strings.TrimSpace(c.CTCircuitoFirmaSourcePath)
	return c
}

// Configurada indica si se ha declarado algún catálogo de reglas.
func (c ConfiguracionReglasEjemplo) Configurada() bool {
	c = c.normalizar()
	return c.BolsaSourcePath != "" || c.CTSourcePath != "" || c.BolsaRolesSegregacionSourcePath != "" || c.CTRetribucionesSourcePath != "" || c.CTPlantillasSourcePath != "" || c.CTCircuitoFirmaSourcePath != ""
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

// RechazarReglasEjemploSinComposicion es la comprobación de las raíces que no
// componen reglas de ejemplo (vec-interno y vec-publico). Fuera de la doble
// llave devuelve ErrConfiguracionReglasEjemploFueraDesarrollo; dentro de ella,
// ErrConfiguracionReglasEjemploSinComposicion. Sin catálogos declarados, nil.
func (c Config) RechazarReglasEjemploSinComposicion() error {
	if _, _, err := c.ReglasEjemploDesarrollo(); err != nil {
		return err
	}
	c = c.Normalize()
	if c.ReglasEjemplo.Configurada() || c.CTAnalisisMotivosSourcePath != "" {
		return ErrConfiguracionReglasEjemploSinComposicion
	}
	return nil
}
