package config

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
)

const (
	// EnvDietasBorradoresEnabled es un selector deliberado, no un permiso. Sólo
	// admite los literales canónicos "true" y "false"; la ausencia equivale a
	// apagado. Evitamos que un 1/yes copiado de otro despliegue abra Dietas.
	EnvDietasBorradoresEnabled                    = "VEC_DIETAS_BORRADORES_ENABLED"
	EnvDietasBorradoresDatabaseURL                = "VEC_DIETAS_BORRADORES_DATABASE_URL"
	EnvDietasPersonalRelacionesDatabaseURL        = "VEC_DIETAS_PERSONAL_RELACIONES_DATABASE_URL"
	EnvDietasPersonalAsignacionDatabaseURL        = "VEC_DIETAS_PERSONAL_ASIGNACION_DATABASE_URL"
	EnvDietasPersonalAuditoriaFronteraDatabaseURL = "VEC_DIETAS_PERSONAL_AUDITORIA_FRONTERA_DATABASE_URL"
)

var (
	ErrConfiguracionDietasBorradoresIncompleta = errors.New("config: faltan conexiones PostgreSQL separadas para borradores de dietas")
	ErrConfiguracionDietasBorradoresNoSeparada = errors.New("config: las conexiones PostgreSQL de borradores de dietas no estan separadas")
	ErrConfiguracionDietasBorradoresSelector   = errors.New("config: selector de borradores de dietas invalido")
	ErrConfiguracionDietasBorradoresActivacion = errors.New("config: activacion de borradores de dietas incompleta o fuera del perfil de desarrollo")
)

// ConfiguracionDietasBorradores pertenece exclusivamente a la raíz de
// composición. Separa el ejecutor Dietas, la lectura de relaciones Personal,
// el ejecutor de asignaciones Personal y su auditoría de frontera. Las demás
// autoridades de sesión y V3 se configuran fuera de estos DSN. No contiene
// personas ni material de identidad y siempre redacta sus representaciones.
type ConfiguracionDietasBorradores struct {
	dsnDietas             string
	dsnPersonal           string
	dsnAsignacionPersonal string
	dsnAuditoriaPersonal  string
}

func NuevaConfiguracionDietasBorradores(
	dsnDietas, dsnPersonal string,
	dsnPersonalAdicional ...string,
) (ConfiguracionDietasBorradores, error) {
	if len(dsnPersonalAdicional) > 2 {
		return ConfiguracionDietasBorradores{}, ErrConfiguracionDietasBorradoresIncompleta
	}
	c := ConfiguracionDietasBorradores{
		dsnDietas: dsnDietas, dsnPersonal: dsnPersonal,
	}.normalizar()
	if len(dsnPersonalAdicional) > 0 {
		c.dsnAsignacionPersonal = strings.TrimSpace(dsnPersonalAdicional[0])
	}
	if len(dsnPersonalAdicional) > 1 {
		c.dsnAuditoriaPersonal = strings.TrimSpace(dsnPersonalAdicional[1])
	}
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
	if c.dsnAsignacionPersonal != "" && (conexionPostgreSQLComparteLogin(c.dsnAsignacionPersonal, dsns[0]) || conexionPostgreSQLComparteLogin(c.dsnAsignacionPersonal, dsns[1])) {
		return ErrConfiguracionDietasBorradoresNoSeparada
	}
	for _, previo := range []string{dsns[0], dsns[1], c.dsnAsignacionPersonal} {
		if c.dsnAuditoriaPersonal != "" && conexionPostgreSQLComparteLogin(c.dsnAuditoriaPersonal, previo) {
			return ErrConfiguracionDietasBorradoresNoSeparada
		}
	}
	return nil
}

func (c ConfiguracionDietasBorradores) validarCompleta() error {
	c = c.normalizar()
	if err := c.Validar(); err != nil {
		return err
	}
	if c.dsnAsignacionPersonal == "" || c.dsnAuditoriaPersonal == "" {
		return ErrConfiguracionDietasBorradoresIncompleta
	}
	return nil
}

// DSNAsignacionPersonal entrega solo la identidad técnica propia de Personal.
// La activación completa exige este tercer login, separado de Dietas y de la
// consulta de relaciones.
func (c ConfiguracionDietasBorradores) DSNAsignacionPersonal() (string, error) {
	c = c.normalizar()
	if err := c.Validar(); err != nil {
		return "", err
	}
	if c.dsnAsignacionPersonal == "" {
		return "", ErrConfiguracionDietasBorradoresIncompleta
	}
	return c.dsnAsignacionPersonal, nil
}

func (c ConfiguracionDietasBorradores) DSNAuditoriaPersonal() (string, error) {
	c = c.normalizar()
	if err := c.validarCompleta(); err != nil {
		return "", err
	}
	return c.dsnAuditoriaPersonal, nil
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
		if !c.DevelopmentEnabledByDoubleKey() {
			return false, ErrConfiguracionDietasBorradoresActivacion
		}
		if err := c.DietasBorradoresPostgreSQL.validarCompleta(); err != nil {
			return false, fmt.Errorf("%w: %v", ErrConfiguracionDietasBorradoresActivacion, err)
		}
		for _, propio := range []string{c.DietasBorradoresPostgreSQL.dsnDietas, c.DietasBorradoresPostgreSQL.dsnPersonal, c.DietasBorradoresPostgreSQL.dsnAsignacionPersonal, c.DietasBorradoresPostgreSQL.dsnAuditoriaPersonal} {
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
	c.dsnAsignacionPersonal = strings.TrimSpace(c.dsnAsignacionPersonal)
	c.dsnAuditoriaPersonal = strings.TrimSpace(c.dsnAuditoriaPersonal)
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
