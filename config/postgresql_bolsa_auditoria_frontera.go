package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

const (
	EnvBolsaAuditoriaFronteraDatabaseURL                   = "VEC_BOLSA_AUDITORIA_FRONTERA_DATABASE_URL"
	configuracionPostgreSQLBolsaAuditoriaFronteraRedactada = "configuracion_postgresql_bolsa_auditoria_frontera_redactada"
)

var (
	ErrConfiguracionPostgreSQLBolsaAuditoriaFronteraIncompleta = errors.New("config: falta la conexion PostgreSQL de auditoria de frontera de Bolsa")
	ErrConfiguracionPostgreSQLBolsaAuditoriaFronteraNoSeparada = errors.New("config: la conexion PostgreSQL de auditoria de frontera de Bolsa no esta separada")
)

// ConfiguracionPostgreSQLBolsaAuditoriaFrontera conserva solo el LOGIN
// nominal del registrador de rechazos B-BACK. Nunca comparte la conexion del
// ejecutor de Bolsa ni las de Contratacion temporal.
type ConfiguracionPostgreSQLBolsaAuditoriaFrontera struct{ dsn string }

func (c ConfiguracionPostgreSQLBolsaAuditoriaFrontera) normalizar() ConfiguracionPostgreSQLBolsaAuditoriaFrontera {
	c.dsn = strings.TrimSpace(c.dsn)
	return c
}

func (c ConfiguracionPostgreSQLBolsaAuditoriaFrontera) DSNSeparadoDeConfiguracion(
	configuracion Config,
) (string, error) {
	c = c.normalizar()
	if c.dsn == "" {
		return "", ErrConfiguracionPostgreSQLBolsaAuditoriaFronteraIncompleta
	}
	for _, previo := range configuracion.dsnsPostgreSQLConfigurados() {
		if conexionPostgreSQLComparteLogin(c.dsn, previo) {
			return "", ErrConfiguracionPostgreSQLBolsaAuditoriaFronteraNoSeparada
		}
	}
	return c.dsn, nil
}

// DSNBolsaAuditoriaFronteraSeparado solo exige la credencial cuando el paso
// Bolsa se ha solicitado expresamente. Su ausencia conserva CT sin simular
// B-BACK; una configuracion parcial de B-BACK falla cerrada.
func (c Config) DSNBolsaAuditoriaFronteraSeparado() (string, error) {
	c = c.Normalize()
	if !c.ContratacionTemporalPostgreSQL.BolsaLlamamientosConfigurada() {
		return "", nil
	}
	return c.BolsaAuditoriaFronteraPostgreSQL.DSNSeparadoDeConfiguracion(c)
}

// dsnsPostgreSQLConfigurados mantiene el inventario privado de credenciales
// que ya tienen autoridad en VEC. Incluir configuraciones parciales evita que
// una URL añadida para Bolsa se convierta en vía de reutilización.
func (c Config) dsnsPostgreSQLConfigurados() []string {
	c = c.Normalize()
	resultado := c.ContratacionTemporalPostgreSQL.dsnsConfigurados()
	for _, dsn := range []string{
		c.BolsaPublicaPostgreSQL.dsn,
		c.BolsaImportacionConvocaPostgreSQL.dsn,
		c.BolsaBorradoresPostgreSQL.dsnEjecutorConsulta,
		c.BolsaBorradoresPostgreSQL.dsnProyectorGobierno,
		c.BolsaBorradoresPostgreSQL.dsnVerificadorRecibo,
	} {
		if dsn = strings.TrimSpace(dsn); dsn != "" {
			resultado = append(resultado, dsn)
		}
	}
	return resultado
}

func conexionPostgreSQLComparteLogin(izquierda, derecha string) bool {
	izquierda, derecha = strings.TrimSpace(izquierda), strings.TrimSpace(derecha)
	if izquierda == "" || derecha == "" {
		return false
	}
	if izquierda == derecha {
		return true
	}
	configuracionIzquierda, errIzquierda := pgconn.ParseConfig(izquierda)
	configuracionDerecha, errDerecha := pgconn.ParseConfig(derecha)
	return errIzquierda == nil && errDerecha == nil && configuracionIzquierda.User != "" && configuracionIzquierda.User == configuracionDerecha.User
}

func (ConfiguracionPostgreSQLBolsaAuditoriaFrontera) String() string {
	return configuracionPostgreSQLBolsaAuditoriaFronteraRedactada
}
func (ConfiguracionPostgreSQLBolsaAuditoriaFrontera) GoString() string {
	return configuracionPostgreSQLBolsaAuditoriaFronteraRedactada
}
func (ConfiguracionPostgreSQLBolsaAuditoriaFrontera) Format(s fmt.State, _ rune) {
	_, _ = s.Write([]byte(configuracionPostgreSQLBolsaAuditoriaFronteraRedactada))
}
func (ConfiguracionPostgreSQLBolsaAuditoriaFrontera) MarshalJSON() ([]byte, error) {
	return json.Marshal(configuracionPostgreSQLBolsaAuditoriaFronteraRedactada)
}
func (ConfiguracionPostgreSQLBolsaAuditoriaFrontera) MarshalText() ([]byte, error) {
	return []byte(configuracionPostgreSQLBolsaAuditoriaFronteraRedactada), nil
}
func (ConfiguracionPostgreSQLBolsaAuditoriaFrontera) LogValue() slog.Value {
	return slog.StringValue(configuracionPostgreSQLBolsaAuditoriaFronteraRedactada)
}
