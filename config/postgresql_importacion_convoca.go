package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
)

const EnvBolsaImportacionConvocaDatabaseURL = "VEC_BOLSA_IMPORTACION_CONVOCA_DATABASE_URL"
const configuracionPostgreSQLImportacionConvocaRedactada = "configuracion_postgresql_importacion_convoca_redactada"

var ErrConfiguracionPostgreSQLImportacionConvocaIncompleta = errors.New("config: falta la conexion PostgreSQL de importacion Convoca")

type ConfiguracionPostgreSQLImportacionConvoca struct{ dsn string }

func NuevaConfiguracionPostgreSQLImportacionConvoca(dsn string) (ConfiguracionPostgreSQLImportacionConvoca, error) {
	c := ConfiguracionPostgreSQLImportacionConvoca{dsn: strings.TrimSpace(dsn)}
	return c, c.Validar()
}
func (c ConfiguracionPostgreSQLImportacionConvoca) Validar() error {
	if strings.TrimSpace(c.dsn) == "" {
		return ErrConfiguracionPostgreSQLImportacionConvocaIncompleta
	}
	return nil
}
func (c ConfiguracionPostgreSQLImportacionConvoca) DSN() (string, error) {
	if err := c.Validar(); err != nil {
		return "", err
	}
	return strings.TrimSpace(c.dsn), nil
}
func (ConfiguracionPostgreSQLImportacionConvoca) String() string {
	return configuracionPostgreSQLImportacionConvocaRedactada
}
func (ConfiguracionPostgreSQLImportacionConvoca) GoString() string {
	return configuracionPostgreSQLImportacionConvocaRedactada
}
func (ConfiguracionPostgreSQLImportacionConvoca) Format(s fmt.State, _ rune) {
	_, _ = s.Write([]byte(configuracionPostgreSQLImportacionConvocaRedactada))
}
func (ConfiguracionPostgreSQLImportacionConvoca) MarshalJSON() ([]byte, error) {
	return json.Marshal(configuracionPostgreSQLImportacionConvocaRedactada)
}
func (ConfiguracionPostgreSQLImportacionConvoca) MarshalText() ([]byte, error) {
	return []byte(configuracionPostgreSQLImportacionConvocaRedactada), nil
}
func (ConfiguracionPostgreSQLImportacionConvoca) LogValue() slog.Value {
	return slog.StringValue(configuracionPostgreSQLImportacionConvocaRedactada)
}
