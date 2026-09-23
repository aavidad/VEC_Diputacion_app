package config

import (
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
)

const (
	// EnvDietasBorradoresEnabled es un selector deliberado, no un permiso. Sólo
	// admite los literales canónicos "true" y "false"; la ausencia equivale a
	// apagado. Evitamos que un 1/yes copiado de otro despliegue abra Dietas.
	EnvDietasBorradoresEnabled = "VEC_DIETAS_BORRADORES_ENABLED"
	// EnvDietasIdentidadFile apunta al mapa privado de mTLS a cuentas técnicas.
	// No contiene ni concede datos de persona, empleado o autorización.
	EnvDietasIdentidadFile = "VEC_DIETAS_IDENTIDAD_FILE"

	EnvDietasBorradoresDatabaseURL         = "VEC_DIETAS_BORRADORES_DATABASE_URL"
	EnvDietasPersonalRelacionesDatabaseURL = "VEC_DIETAS_PERSONAL_RELACIONES_DATABASE_URL"
)

var (
	ErrConfiguracionDietasBorradoresIncompleta = errors.New("config: faltan conexiones PostgreSQL separadas para borradores de dietas")
	ErrConfiguracionDietasBorradoresNoSeparada = errors.New("config: las conexiones PostgreSQL de borradores de dietas no estan separadas")
	ErrConfiguracionDietasBorradoresSelector   = errors.New("config: selector de borradores de dietas invalido")
	ErrConfiguracionDietasBorradoresActivacion = errors.New("config: activacion de borradores de dietas incompleta o fuera del perfil de desarrollo")
)

// ConfiguracionDietasBorradores pertenece exclusivamente a la raiz de
// composicion. Cada DSN corresponde a una identidad tecnica nominal: Dietas,
// Personal, registro y revalidacion de sesion, contexto, fuente/registro de
// PDP y catalogo historico de motivos. No incluye persona, empleado, permisos
// ni material de identidad y sus representaciones se redactan siempre.
type ConfiguracionDietasBorradores struct {
	dsnDietas   string
	dsnPersonal string
}

func NuevaConfiguracionDietasBorradores(
	dsnDietas, dsnPersonal string,
) (ConfiguracionDietasBorradores, error) {
	c := ConfiguracionDietasBorradores{
		dsnDietas: dsnDietas, dsnPersonal: dsnPersonal,
	}.normalizar()
	if err := c.Validar(); err != nil {
		return ConfiguracionDietasBorradores{}, err
	}
	return c, nil
}

func (c ConfiguracionDietasBorradores) Validar() error {
	c = c.normalizar()
	dsns := []string{c.dsnDietas, c.dsnPersonal}
	for _, dsn := range dsns {
		if dsn == "" {
			return ErrConfiguracionDietasBorradoresIncompleta
		}
	}
	if conexionPostgreSQLComparteLogin(dsns[0], dsns[1]) {
		return ErrConfiguracionDietasBorradoresNoSeparada
	}
	return nil
}

// DSNSeparados solo expone secretos a la composicion después de comprobar que
// no falta ni se reutiliza una identidad técnica.
func (c ConfiguracionDietasBorradores) DSNSeparados() (dietas, personal string, err error) {
	c = c.normalizar()
	if err = c.Validar(); err != nil {
		return "", "", err
	}
	return c.dsnDietas, c.dsnPersonal, nil
}

func (c ConfiguracionDietasBorradores) Configurada() bool {
	return strings.TrimSpace(c.dsnDietas) != ""
}

// DietasBorradoresDesarrolloActivos valida la activación completa sin abrir
// ficheros ni conexiones. Con selector false (o ausente) no valida fuentes
// incompletas: siguen inertes. Un selector mal escrito siempre falla cerrado.
func (c Config) DietasBorradoresDesarrolloActivos() (bool, error) {
	c = c.Normalize()
	switch c.DietasBorradoresEnabled {
	case "":
		return false, nil
	case "false":
		return false, nil
	case "true":
		if !c.DevelopmentEnabledByDoubleKey() || c.DietasIdentidadFile == "" || !filepath.IsAbs(c.DietasIdentidadFile) {
			return false, ErrConfiguracionDietasBorradoresActivacion
		}
		if err := c.DietasBorradoresPostgreSQL.Validar(); err != nil {
			return false, fmt.Errorf("%w: %v", ErrConfiguracionDietasBorradoresActivacion, err)
		}
		for _, propio := range []string{c.DietasBorradoresPostgreSQL.dsnDietas, c.DietasBorradoresPostgreSQL.dsnPersonal} {
			for _, ajeno := range c.dsnsPostgreSQLConfiguradosSinDietas() {
				if conexionPostgreSQLComparteLogin(propio, ajeno) {
					return false, ErrConfiguracionDietasBorradoresNoSeparada
				}
			}
		}
		return true, nil
	default:
		return false, ErrConfiguracionDietasBorradoresSelector
	}
}

func (c ConfiguracionDietasBorradores) normalizar() ConfiguracionDietasBorradores {
	c.dsnDietas = strings.TrimSpace(c.dsnDietas)
	c.dsnPersonal = strings.TrimSpace(c.dsnPersonal)
	return c
}

const configuracionDietasBorradoresRedactada = "configuracion_dietas_borradores_redactada"

func (ConfiguracionDietasBorradores) String() string   { return configuracionDietasBorradoresRedactada }
func (ConfiguracionDietasBorradores) GoString() string { return configuracionDietasBorradoresRedactada }
func (ConfiguracionDietasBorradores) Format(s fmt.State, _ rune) {
	_, _ = s.Write([]byte(configuracionDietasBorradoresRedactada))
}
func (ConfiguracionDietasBorradores) MarshalJSON() ([]byte, error) {
	return []byte(`"configuracion_dietas_borradores_redactada"`), nil
}
func (ConfiguracionDietasBorradores) MarshalText() ([]byte, error) {
	return []byte(configuracionDietasBorradoresRedactada), nil
}
func (ConfiguracionDietasBorradores) LogValue() slog.Value {
	return slog.StringValue(configuracionDietasBorradoresRedactada)
}
