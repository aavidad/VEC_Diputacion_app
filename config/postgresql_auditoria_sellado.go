package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
)

// EnvAuditoriaSelladoDatabaseURL identifica la conexión del sellado diferido
// de la cadena de auditoría (AD207). Su LOGIN solo pertenece al grupo
// vec_auditoria_encadenador y no comparte usuario con ninguna otra conexión.
const EnvAuditoriaSelladoDatabaseURL = "VEC_AUDITORIA_SELLADO_DATABASE_URL"

const configuracionPostgreSQLAuditoriaSelladoRedactada = "configuracion_postgresql_auditoria_sellado_redactada"

var ErrConfiguracionPostgreSQLAuditoriaSelladoNoSeparada = errors.New("config: la conexion PostgreSQL del sellado de auditoria no esta separada")

// ConfiguracionPostgreSQLAuditoriaSellado conserva el DSN sin exponerlo al
// registro, al formato de errores ni a la serialización del servidor.
type ConfiguracionPostgreSQLAuditoriaSellado struct{ dsn string }

func NuevaConfiguracionPostgreSQLAuditoriaSellado(dsn string) ConfiguracionPostgreSQLAuditoriaSellado {
	return ConfiguracionPostgreSQLAuditoriaSellado{dsn: dsn}.normalizar()
}

func (c ConfiguracionPostgreSQLAuditoriaSellado) normalizar() ConfiguracionPostgreSQLAuditoriaSellado {
	c.dsn = strings.TrimSpace(c.dsn)
	return c
}

// DSNAuditoriaSelladoSeparado devuelve "" sin error si no se configuró.
func (c Config) DSNAuditoriaSelladoSeparado() (string, error) {
	c = c.Normalize()
	sellado := c.AuditoriaSelladoPostgreSQL.dsn
	if sellado == "" {
		return "", nil
	}
	previas := append(c.dsnsPostgreSQLConfigurados(),
		c.BolsaAuditoriaFronteraPostgreSQL.normalizar().dsn,
		c.BolsaRelevoNoIncorporacionPostgreSQL.normalizar().dsn,
		c.BolsaRelevoCesePostgreSQL.normalizar().dsn)
	for _, previa := range previas {
		if conexionPostgreSQLComparteLogin(sellado, previa) {
			return "", ErrConfiguracionPostgreSQLAuditoriaSelladoNoSeparada
		}
	}
	return sellado, nil
}

func (ConfiguracionPostgreSQLAuditoriaSellado) String() string {
	return configuracionPostgreSQLAuditoriaSelladoRedactada
}
func (ConfiguracionPostgreSQLAuditoriaSellado) GoString() string {
	return configuracionPostgreSQLAuditoriaSelladoRedactada
}
func (ConfiguracionPostgreSQLAuditoriaSellado) Format(s fmt.State, _ rune) {
	_, _ = s.Write([]byte(configuracionPostgreSQLAuditoriaSelladoRedactada))
}
func (ConfiguracionPostgreSQLAuditoriaSellado) MarshalJSON() ([]byte, error) {
	return json.Marshal(configuracionPostgreSQLAuditoriaSelladoRedactada)
}
func (ConfiguracionPostgreSQLAuditoriaSellado) MarshalText() ([]byte, error) {
	return []byte(configuracionPostgreSQLAuditoriaSelladoRedactada), nil
}
func (ConfiguracionPostgreSQLAuditoriaSellado) LogValue() slog.Value {
	return slog.StringValue(configuracionPostgreSQLAuditoriaSelladoRedactada)
}
