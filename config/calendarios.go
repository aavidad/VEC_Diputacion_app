package config

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
)

// EnvCalendariosDatabaseURL es la conexión de un login miembro del rol
// vec_calendarios_lector. Sin ella las rutas de Calendarios responden 503.
const EnvCalendariosDatabaseURL = "VEC_CALENDARIOS_DATABASE_URL"

var ErrConfiguracionCalendariosNoSeparada = errors.New("config: la conexion de Calendarios comparte login con otro modulo")

// ConfiguracionCalendarios pertenece a la raíz de composición. Su única
// conexión es de solo lectura y sus representaciones se redactan siempre.
type ConfiguracionCalendarios struct {
	dsn string
}

func NuevaConfiguracionCalendarios(dsn string) ConfiguracionCalendarios {
	return ConfiguracionCalendarios{dsn: strings.TrimSpace(dsn)}
}

func (c ConfiguracionCalendarios) Configurada() bool { return strings.TrimSpace(c.dsn) != "" }

// DSN solo se entrega si no reutiliza el login de otra capacidad configurada.
func (c Config) DSNCalendarios() (string, error) {
	dsn := strings.TrimSpace(c.CalendariosPostgreSQL.dsn)
	if dsn == "" {
		return "", nil
	}
	ajenos := append(c.dsnsPostgreSQLConfiguradosSinDietas(),
		c.DietasBorradoresPostgreSQL.dsnDietas, c.DietasBorradoresPostgreSQL.dsnPersonal)
	for _, ajeno := range ajenos {
		if conexionPostgreSQLComparteLogin(dsn, ajeno) {
			return "", ErrConfiguracionCalendariosNoSeparada
		}
	}
	return dsn, nil
}

const configuracionCalendariosRedactada = "configuracion_calendarios_redactada"

func (ConfiguracionCalendarios) String() string   { return configuracionCalendariosRedactada }
func (ConfiguracionCalendarios) GoString() string { return configuracionCalendariosRedactada }
func (ConfiguracionCalendarios) Format(s fmt.State, _ rune) {
	_, _ = s.Write([]byte(configuracionCalendariosRedactada))
}
func (ConfiguracionCalendarios) MarshalJSON() ([]byte, error) {
	return []byte(`"` + configuracionCalendariosRedactada + `"`), nil
}
func (ConfiguracionCalendarios) MarshalText() ([]byte, error) {
	return []byte(configuracionCalendariosRedactada), nil
}
func (ConfiguracionCalendarios) LogValue() slog.Value {
	return slog.StringValue(configuracionCalendariosRedactada)
}
