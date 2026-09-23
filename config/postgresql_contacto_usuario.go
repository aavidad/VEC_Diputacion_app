package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
)

const (
	EnvContactoUsuarioWriterDatabaseURL             = "VEC_CONTACTO_USUARIO_WRITER_DATABASE_URL"
	EnvContactoUsuarioReaderDatabaseURL             = "VEC_CONTACTO_USUARIO_READER_DATABASE_URL"
	EnvContactoUsuarioMotivosDatabaseURL            = "VEC_CONTACTO_USUARIO_MOTIVOS_DATABASE_URL"
	EnvContactoUsuarioFuenteDatabaseURL             = "VEC_CONTACTO_USUARIO_FUENTE_DATABASE_URL"
	configuracionPostgreSQLContactoUsuarioRedactada = "configuracion_postgresql_contacto_usuario_redactada"
)

var ErrConfiguracionPostgreSQLContactoUsuarioIncompleta = errors.New("config: contacto propio requiere cuatro conexiones nominales")
var ErrConfiguracionPostgreSQLContactoUsuarioNoSeparada = errors.New("config: contacto propio debe usar logins separados")

type ConfiguracionPostgreSQLContactoUsuario struct {
	dsnWriter  string
	dsnReader  string
	dsnFuente  string
	dsnMotivos string
}

func NuevaConfiguracionPostgreSQLContactoUsuario(writer, reader, fuente, motivos string) ConfiguracionPostgreSQLContactoUsuario {
	return ConfiguracionPostgreSQLContactoUsuario{strings.TrimSpace(writer), strings.TrimSpace(reader), strings.TrimSpace(fuente), strings.TrimSpace(motivos)}
}

func (c ConfiguracionPostgreSQLContactoUsuario) Configurada() bool {
	return c.dsnWriter != "" || c.dsnReader != "" || c.dsnFuente != "" || c.dsnMotivos != ""
}

func (c ConfiguracionPostgreSQLContactoUsuario) DSNSeparados(configuracion Config) (string, string, string, string, error) {
	c = NuevaConfiguracionPostgreSQLContactoUsuario(c.dsnWriter, c.dsnReader, c.dsnFuente, c.dsnMotivos)
	if c.dsnWriter == "" || c.dsnReader == "" || c.dsnFuente == "" || c.dsnMotivos == "" {
		return "", "", "", "", ErrConfiguracionPostgreSQLContactoUsuarioIncompleta
	}
	todos := []string{c.dsnWriter, c.dsnReader, c.dsnFuente, c.dsnMotivos}
	for i, dsn := range todos {
		for _, previo := range configuracion.dsnsPostgreSQLConfiguradosSinContacto() {
			if conexionPostgreSQLComparteLogin(dsn, previo) {
				return "", "", "", "", ErrConfiguracionPostgreSQLContactoUsuarioNoSeparada
			}
		}
		for _, anterior := range todos[:i] {
			if conexionPostgreSQLComparteLogin(dsn, anterior) {
				return "", "", "", "", ErrConfiguracionPostgreSQLContactoUsuarioNoSeparada
			}
		}
	}
	return todos[0], todos[1], todos[2], todos[3], nil
}

func (ConfiguracionPostgreSQLContactoUsuario) String() string {
	return configuracionPostgreSQLContactoUsuarioRedactada
}
func (ConfiguracionPostgreSQLContactoUsuario) GoString() string {
	return configuracionPostgreSQLContactoUsuarioRedactada
}
func (ConfiguracionPostgreSQLContactoUsuario) Format(s fmt.State, _ rune) {
	_, _ = s.Write([]byte(configuracionPostgreSQLContactoUsuarioRedactada))
}
func (ConfiguracionPostgreSQLContactoUsuario) MarshalJSON() ([]byte, error) {
	return json.Marshal(configuracionPostgreSQLContactoUsuarioRedactada)
}
func (ConfiguracionPostgreSQLContactoUsuario) MarshalText() ([]byte, error) {
	return []byte(configuracionPostgreSQLContactoUsuarioRedactada), nil
}
func (ConfiguracionPostgreSQLContactoUsuario) LogValue() slog.Value {
	return slog.StringValue(configuracionPostgreSQLContactoUsuarioRedactada)
}
