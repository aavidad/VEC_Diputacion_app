package config

import (
	"errors"
	"strings"
)

// ConfiguracionPostgreSQLExterna evita mostrar una contraseña al imprimir la
// configuración de un proceso del Área personal.
type ConfiguracionPostgreSQLExterna struct{ dsn string }

var ErrConexionPostgreSQLExternaIncompleta = errors.New("config: falta una conexion PostgreSQL nominal del portal externo")

func (c ConfiguracionPostgreSQLExterna) DSN() (string, error) {
	if strings.TrimSpace(c.dsn) == "" {
		return "", ErrConexionPostgreSQLExternaIncompleta
	}
	return c.dsn, nil
}

func (c ConfiguracionPostgreSQLExterna) normalizar() ConfiguracionPostgreSQLExterna {
	c.dsn = strings.TrimSpace(c.dsn)
	return c
}

func (ConfiguracionPostgreSQLExterna) String() string {
	return "configuracion_postgresql_externa_redactada"
}
func (ConfiguracionPostgreSQLExterna) GoString() string {
	return "configuracion_postgresql_externa_redactada"
}
