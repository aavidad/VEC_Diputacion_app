package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
)

// EnvBolsaRelevoCeseDatabaseURL identifica la conexión exclusiva que lleva
// ceses CT acreditados a Bolsa 000045. No es el ejecutor ni el migrador.
const EnvBolsaRelevoCeseDatabaseURL = "VEC_BOLSA_CESE_CT_DATABASE_URL"

const configuracionPostgreSQLBolsaRelevoCeseRedactada = "configuracion_postgresql_bolsa_relevo_cese_redactada"

var (
	ErrConfiguracionPostgreSQLBolsaRelevoCeseIncompleta = errors.New("config: falta la conexion PostgreSQL del relevo de ceses de Bolsa")
	ErrConfiguracionPostgreSQLBolsaRelevoCeseNoSeparada = errors.New("config: la conexion PostgreSQL del relevo de ceses de Bolsa no esta separada")
)

// ConfiguracionPostgreSQLBolsaRelevoCese conserva el DSN sin exponerlo al
// registro, al formato de errores ni a la serialización del servidor.
type ConfiguracionPostgreSQLBolsaRelevoCese struct{ dsn string }

func NuevaConfiguracionPostgreSQLBolsaRelevoCese(dsn string) ConfiguracionPostgreSQLBolsaRelevoCese {
	return ConfiguracionPostgreSQLBolsaRelevoCese{dsn: dsn}.normalizar()
}

func (c ConfiguracionPostgreSQLBolsaRelevoCese) normalizar() ConfiguracionPostgreSQLBolsaRelevoCese {
	c.dsn = strings.TrimSpace(c.dsn)
	return c
}

func (c Config) DSNBolsaRelevoCeseSeparado() (string, error) {
	c = c.Normalize()
	relevo := c.BolsaRelevoCesePostgreSQL.dsn
	if relevo == "" {
		return "", ErrConfiguracionPostgreSQLBolsaRelevoCeseIncompleta
	}
	previas := append(c.dsnsPostgreSQLConfigurados(),
		c.BolsaAuditoriaFronteraPostgreSQL.normalizar().dsn,
		c.BolsaRelevoNoIncorporacionPostgreSQL.normalizar().dsn)
	for _, previa := range previas {
		if conexionPostgreSQLComparteLogin(relevo, previa) {
			return "", ErrConfiguracionPostgreSQLBolsaRelevoCeseNoSeparada
		}
	}
	return relevo, nil
}

func (ConfiguracionPostgreSQLBolsaRelevoCese) String() string {
	return configuracionPostgreSQLBolsaRelevoCeseRedactada
}
func (ConfiguracionPostgreSQLBolsaRelevoCese) GoString() string {
	return configuracionPostgreSQLBolsaRelevoCeseRedactada
}
func (ConfiguracionPostgreSQLBolsaRelevoCese) Format(s fmt.State, _ rune) {
	_, _ = s.Write([]byte(configuracionPostgreSQLBolsaRelevoCeseRedactada))
}
func (ConfiguracionPostgreSQLBolsaRelevoCese) MarshalJSON() ([]byte, error) {
	return json.Marshal(configuracionPostgreSQLBolsaRelevoCeseRedactada)
}
func (ConfiguracionPostgreSQLBolsaRelevoCese) MarshalText() ([]byte, error) {
	return []byte(configuracionPostgreSQLBolsaRelevoCeseRedactada), nil
}
func (ConfiguracionPostgreSQLBolsaRelevoCese) LogValue() slog.Value {
	return slog.StringValue(configuracionPostgreSQLBolsaRelevoCeseRedactada)
}
