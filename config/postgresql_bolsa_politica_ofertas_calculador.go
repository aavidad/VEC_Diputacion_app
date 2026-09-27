package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
)

// EnvBolsaPoliticaOfertasCalculadorDatabaseURL es el LOGIN exclusivo que
// consulta la política vigente para calcular el plazo de una oferta B47.
const EnvBolsaPoliticaOfertasCalculadorDatabaseURL = "VEC_BOLSA_POLITICA_OFERTAS_CALCULADOR_DATABASE_URL"

const configuracionBolsaPoliticaOfertasCalculadorRedactada = "configuracion_postgresql_bolsa_calculador_politica_redactada"

var (
	ErrBolsaPoliticaOfertasCalculadorIncompleto = errors.New("config: falta PostgreSQL del calculador de política de ofertas")
	ErrBolsaPoliticaOfertasCalculadorNoSeparado = errors.New("config: el calculador de política de ofertas comparte LOGIN")
)

type ConfiguracionPostgreSQLBolsaPoliticaOfertasCalculador struct{ dsn string }

func NuevaConfiguracionPostgreSQLBolsaPoliticaOfertasCalculador(dsn string) ConfiguracionPostgreSQLBolsaPoliticaOfertasCalculador {
	return ConfiguracionPostgreSQLBolsaPoliticaOfertasCalculador{dsn: dsn}.normalizar()
}

func (c ConfiguracionPostgreSQLBolsaPoliticaOfertasCalculador) normalizar() ConfiguracionPostgreSQLBolsaPoliticaOfertasCalculador {
	c.dsn = strings.TrimSpace(c.dsn)
	return c
}

func (c Config) DSNBolsaPoliticaOfertasCalculadorSeparado() (string, error) {
	c = c.Normalize()
	dsn := c.BolsaPoliticaOfertasCalculadorPostgreSQL.dsn
	if dsn == "" {
		return "", ErrBolsaPoliticaOfertasCalculadorIncompleto
	}
	previas := append(c.dsnsPostgreSQLConfigurados(), c.BolsaAuditoriaFronteraPostgreSQL.normalizar().dsn,
		c.BolsaRelevoNoIncorporacionPostgreSQL.normalizar().dsn, c.BolsaRelevoCesePostgreSQL.normalizar().dsn)
	for _, previa := range previas {
		if conexionPostgreSQLComparteLogin(dsn, previa) {
			return "", ErrBolsaPoliticaOfertasCalculadorNoSeparado
		}
	}
	return dsn, nil
}

func (ConfiguracionPostgreSQLBolsaPoliticaOfertasCalculador) String() string {
	return configuracionBolsaPoliticaOfertasCalculadorRedactada
}
func (ConfiguracionPostgreSQLBolsaPoliticaOfertasCalculador) GoString() string {
	return configuracionBolsaPoliticaOfertasCalculadorRedactada
}
func (ConfiguracionPostgreSQLBolsaPoliticaOfertasCalculador) Format(s fmt.State, _ rune) {
	_, _ = s.Write([]byte(configuracionBolsaPoliticaOfertasCalculadorRedactada))
}
func (ConfiguracionPostgreSQLBolsaPoliticaOfertasCalculador) MarshalJSON() ([]byte, error) {
	return json.Marshal(configuracionBolsaPoliticaOfertasCalculadorRedactada)
}
func (ConfiguracionPostgreSQLBolsaPoliticaOfertasCalculador) MarshalText() ([]byte, error) {
	return []byte(configuracionBolsaPoliticaOfertasCalculadorRedactada), nil
}
func (ConfiguracionPostgreSQLBolsaPoliticaOfertasCalculador) LogValue() slog.Value {
	return slog.StringValue(configuracionBolsaPoliticaOfertasCalculadorRedactada)
}
