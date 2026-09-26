package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
)

const (
	// EnvBolsaRelevoNoIncorporacionDatabaseURL es la conexión del relevo que
	// entrega a la bandeja de Bolsa 000042 las no incorporaciones de CT. Su
	// LOGIN solo pertenece al grupo vec_bolsa_llamamientos_relevo_no_incorporacion.
	EnvBolsaRelevoNoIncorporacionDatabaseURL                   = "VEC_BOLSA_RELEVO_NO_INCORPORACION_DATABASE_URL"
	configuracionPostgreSQLBolsaRelevoNoIncorporacionRedactada = "configuracion_postgresql_bolsa_relevo_no_incorporacion_redactada"
)

var (
	ErrConfiguracionPostgreSQLBolsaRelevoNoIncorporacionIncompleta = errors.New("config: falta la conexion PostgreSQL del relevo de no incorporaciones de Bolsa")
	ErrConfiguracionPostgreSQLBolsaRelevoNoIncorporacionNoSeparada = errors.New("config: la conexion PostgreSQL del relevo de no incorporaciones de Bolsa no esta separada")
)

// ConfiguracionPostgreSQLBolsaRelevoNoIncorporacion conserva solo el LOGIN
// nominal del relevo de no incorporaciones. Nunca comparte la conexión del
// ejecutor de Bolsa, de la auditoría de frontera ni las de Contratación
// temporal: la bandeja solo acepta entregas de ese rol.
type ConfiguracionPostgreSQLBolsaRelevoNoIncorporacion struct{ dsn string }

// NuevaConfiguracionPostgreSQLBolsaRelevoNoIncorporacion sirve a pruebas y
// composiciones que no leen el entorno.
func NuevaConfiguracionPostgreSQLBolsaRelevoNoIncorporacion(dsn string) ConfiguracionPostgreSQLBolsaRelevoNoIncorporacion {
	return ConfiguracionPostgreSQLBolsaRelevoNoIncorporacion{dsn: dsn}.normalizar()
}

func (c ConfiguracionPostgreSQLBolsaRelevoNoIncorporacion) normalizar() ConfiguracionPostgreSQLBolsaRelevoNoIncorporacion {
	c.dsn = strings.TrimSpace(c.dsn)
	return c
}

// DSNBolsaRelevoNoIncorporacionSeparado exige la credencial propia del relevo
// y que no reutilice ningún LOGIN ya configurado.
func (c Config) DSNBolsaRelevoNoIncorporacionSeparado() (string, error) {
	c = c.Normalize()
	relevo := c.BolsaRelevoNoIncorporacionPostgreSQL.normalizar().dsn
	if relevo == "" {
		return "", ErrConfiguracionPostgreSQLBolsaRelevoNoIncorporacionIncompleta
	}
	previas := append(c.dsnsPostgreSQLConfigurados(), c.BolsaAuditoriaFronteraPostgreSQL.normalizar().dsn)
	for _, previa := range previas {
		if conexionPostgreSQLComparteLogin(relevo, previa) {
			return "", ErrConfiguracionPostgreSQLBolsaRelevoNoIncorporacionNoSeparada
		}
	}
	return relevo, nil
}

func (ConfiguracionPostgreSQLBolsaRelevoNoIncorporacion) String() string {
	return configuracionPostgreSQLBolsaRelevoNoIncorporacionRedactada
}
func (ConfiguracionPostgreSQLBolsaRelevoNoIncorporacion) GoString() string {
	return configuracionPostgreSQLBolsaRelevoNoIncorporacionRedactada
}
func (ConfiguracionPostgreSQLBolsaRelevoNoIncorporacion) Format(s fmt.State, _ rune) {
	_, _ = s.Write([]byte(configuracionPostgreSQLBolsaRelevoNoIncorporacionRedactada))
}
func (ConfiguracionPostgreSQLBolsaRelevoNoIncorporacion) MarshalJSON() ([]byte, error) {
	return json.Marshal(configuracionPostgreSQLBolsaRelevoNoIncorporacionRedactada)
}
func (ConfiguracionPostgreSQLBolsaRelevoNoIncorporacion) MarshalText() ([]byte, error) {
	return []byte(configuracionPostgreSQLBolsaRelevoNoIncorporacionRedactada), nil
}
func (ConfiguracionPostgreSQLBolsaRelevoNoIncorporacion) LogValue() slog.Value {
	return slog.StringValue(configuracionPostgreSQLBolsaRelevoNoIncorporacionRedactada)
}
