package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
)

const (
	EnvBolsaConstitucionDatabaseURL                   = "VEC_BOLSA_CONSTITUCION_DATABASE_URL"
	configuracionPostgreSQLBolsaConstitucionRedactada = "configuracion_postgresql_bolsa_constitucion_redactada"
)

var (
	ErrConfiguracionPostgreSQLBolsaConstitucionIncompleta = errors.New("config: falta la conexion PostgreSQL de constitucion de Bolsa")
	ErrConfiguracionPostgreSQLBolsaConstitucionNoSeparada = errors.New("config: la conexion PostgreSQL de constitucion de Bolsa no esta separada")
)

// ConfiguracionPostgreSQLBolsaConstitucion pertenece exclusivamente al
// subcomando constituir-bolsa. El servidor HTTP no abre esta conexion.
type ConfiguracionPostgreSQLBolsaConstitucion struct{ dsn string }

func (c ConfiguracionPostgreSQLBolsaConstitucion) normalizar() ConfiguracionPostgreSQLBolsaConstitucion {
	c.dsn = strings.TrimSpace(c.dsn)
	return c
}

// DSNBolsaConstitucionSeparado exige una identidad distinta de todos los
// LOGIN PostgreSQL configurados para la web, importacion y otras operaciones.
func (c Config) DSNBolsaConstitucionSeparado() (string, error) {
	c = c.Normalize()
	dsn := c.BolsaConstitucionPostgreSQL.dsn
	if dsn == "" {
		return "", ErrConfiguracionPostgreSQLBolsaConstitucionIncompleta
	}
	for _, otro := range c.dsnsPostgreSQLConfigurados() {
		if conexionPostgreSQLComparteLogin(dsn, otro) {
			return "", ErrConfiguracionPostgreSQLBolsaConstitucionNoSeparada
		}
	}
	if conexionPostgreSQLComparteLogin(dsn, c.BolsaAuditoriaFronteraPostgreSQL.dsn) {
		return "", ErrConfiguracionPostgreSQLBolsaConstitucionNoSeparada
	}
	return dsn, nil
}

func (ConfiguracionPostgreSQLBolsaConstitucion) String() string {
	return configuracionPostgreSQLBolsaConstitucionRedactada
}
func (ConfiguracionPostgreSQLBolsaConstitucion) GoString() string {
	return configuracionPostgreSQLBolsaConstitucionRedactada
}
func (ConfiguracionPostgreSQLBolsaConstitucion) Format(s fmt.State, _ rune) {
	_, _ = s.Write([]byte(configuracionPostgreSQLBolsaConstitucionRedactada))
}
func (ConfiguracionPostgreSQLBolsaConstitucion) MarshalJSON() ([]byte, error) {
	return json.Marshal(configuracionPostgreSQLBolsaConstitucionRedactada)
}
func (ConfiguracionPostgreSQLBolsaConstitucion) MarshalText() ([]byte, error) {
	return []byte(configuracionPostgreSQLBolsaConstitucionRedactada), nil
}
func (ConfiguracionPostgreSQLBolsaConstitucion) LogValue() slog.Value {
	return slog.StringValue(configuracionPostgreSQLBolsaConstitucionRedactada)
}
