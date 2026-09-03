package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
)

const (
	EnvContratacionTemporalDatabaseURL                   = "VEC_CT_DATABASE_URL"
	EnvContratacionTemporalGobiernoDatabaseURL           = "VEC_CT_GOBIERNO_DATABASE_URL"
	configuracionPostgreSQLContratacionTemporalRedactada = "configuracion_postgresql_contratacion_temporal_redactada"
)

var (
	ErrConfiguracionPostgreSQLContratacionTemporalIncompleta = errors.New(
		"config: faltan conexiones PostgreSQL separadas para contratacion temporal",
	)
	ErrConfiguracionPostgreSQLContratacionTemporalNoSeparada = errors.New(
		"config: las conexiones PostgreSQL de contratacion temporal no estan separadas",
	)
)

// ConfiguracionPostgreSQLContratacionTemporal separa la identidad de ejecución
// de la identidad que publica el gobierno sintético de desarrollo. Ninguna
// representación genérica expone los DSN.
type ConfiguracionPostgreSQLContratacionTemporal struct {
	dsnEjecucion string
	dsnGobierno  string
}

func NuevaConfiguracionPostgreSQLContratacionTemporal(
	dsnEjecucion string,
	dsnGobierno string,
) (ConfiguracionPostgreSQLContratacionTemporal, error) {
	configuracion := ConfiguracionPostgreSQLContratacionTemporal{
		dsnEjecucion: dsnEjecucion,
		dsnGobierno:  dsnGobierno,
	}.normalizar()
	if err := configuracion.Validar(); err != nil {
		return ConfiguracionPostgreSQLContratacionTemporal{}, err
	}
	return configuracion, nil
}

func (c ConfiguracionPostgreSQLContratacionTemporal) Validar() error {
	c = c.normalizar()
	if c.dsnEjecucion == "" || c.dsnGobierno == "" {
		return ErrConfiguracionPostgreSQLContratacionTemporalIncompleta
	}
	if c.dsnEjecucion == c.dsnGobierno {
		return ErrConfiguracionPostgreSQLContratacionTemporalNoSeparada
	}
	return nil
}

func (c ConfiguracionPostgreSQLContratacionTemporal) DSNSeparados() (
	ejecucion string,
	gobierno string,
	err error,
) {
	c = c.normalizar()
	if err := c.Validar(); err != nil {
		return "", "", err
	}
	return c.dsnEjecucion, c.dsnGobierno, nil
}

func (c ConfiguracionPostgreSQLContratacionTemporal) normalizar() ConfiguracionPostgreSQLContratacionTemporal {
	c.dsnEjecucion = strings.TrimSpace(c.dsnEjecucion)
	c.dsnGobierno = strings.TrimSpace(c.dsnGobierno)
	return c
}

func (ConfiguracionPostgreSQLContratacionTemporal) String() string {
	return configuracionPostgreSQLContratacionTemporalRedactada
}
func (ConfiguracionPostgreSQLContratacionTemporal) GoString() string {
	return configuracionPostgreSQLContratacionTemporalRedactada
}
func (ConfiguracionPostgreSQLContratacionTemporal) Format(estado fmt.State, _ rune) {
	_, _ = estado.Write([]byte(configuracionPostgreSQLContratacionTemporalRedactada))
}
func (ConfiguracionPostgreSQLContratacionTemporal) MarshalJSON() ([]byte, error) {
	return json.Marshal(configuracionPostgreSQLContratacionTemporalRedactada)
}
func (ConfiguracionPostgreSQLContratacionTemporal) MarshalText() ([]byte, error) {
	return []byte(configuracionPostgreSQLContratacionTemporalRedactada), nil
}
func (ConfiguracionPostgreSQLContratacionTemporal) LogValue() slog.Value {
	return slog.StringValue(configuracionPostgreSQLContratacionTemporalRedactada)
}
