package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
)

const (
	EnvAdministracionCorreoDatabaseURL                = "VEC_ADMIN_CORREO_DATABASE_URL"
	EnvAdministracionRegistroAutorizacionDatabaseURL  = "VEC_ADMIN_REGISTRO_AUTORIZACION_DATABASE_URL"
	EnvAdministracionRegistroIdentidadDatabaseURL     = "VEC_ADMIN_REGISTRO_IDENTIDAD_DATABASE_URL"
	EnvAdministracionRevalidacionIdentidadDatabaseURL = "VEC_ADMIN_REVALIDACION_IDENTIDAD_DATABASE_URL"
	EnvAdministracionContextoActorDatabaseURL         = "VEC_ADMIN_CONTEXTO_ACTOR_DATABASE_URL"
	configuracionPostgreSQLAdministracionRedactada    = "configuracion_postgresql_administracion_redactada"
)

var (
	ErrConfiguracionPostgreSQLAdministracionIncompleta = errors.New("config: faltan conexiones PostgreSQL separadas para administracion")
	ErrConfiguracionPostgreSQLAdministracionNoSeparada = errors.New("config: las conexiones PostgreSQL de administracion no estan separadas")
)

// ConfiguracionPostgreSQLAdministracion separa la escritura SMTP, el
// registro de autorización y las tres identidades de la sesión ADMIN.
// Ningún DSN de Contratación temporal es un valor por defecto admisible.
type ConfiguracionPostgreSQLAdministracion struct {
	dsnCorreo               string
	dsnRegistroAutorizacion string
	dsnRegistroIdentidad    string
	dsnRevalidacion         string
	dsnContextoActor        string
}

func NuevaConfiguracionPostgreSQLAdministracion(correo, autorizacion, registroIdentidad, revalidacion, contexto string) (ConfiguracionPostgreSQLAdministracion, error) {
	c := ConfiguracionPostgreSQLAdministracion{correo, autorizacion, registroIdentidad, revalidacion, contexto}.normalizar()
	if err := c.Validar(); err != nil {
		return ConfiguracionPostgreSQLAdministracion{}, err
	}
	return c, nil
}

func (c ConfiguracionPostgreSQLAdministracion) Configurada() bool {
	c = c.normalizar()
	return c.dsnCorreo != "" || c.dsnRegistroAutorizacion != "" || c.dsnRegistroIdentidad != "" || c.dsnRevalidacion != "" || c.dsnContextoActor != ""
}

func (c ConfiguracionPostgreSQLAdministracion) Validar() error {
	c = c.normalizar()
	dsns := []string{c.dsnCorreo, c.dsnRegistroAutorizacion, c.dsnRegistroIdentidad, c.dsnRevalidacion, c.dsnContextoActor}
	for _, dsn := range dsns {
		if dsn == "" {
			return ErrConfiguracionPostgreSQLAdministracionIncompleta
		}
	}
	vistos := make(map[string]struct{}, len(dsns))
	for _, dsn := range dsns {
		vistos[dsn] = struct{}{}
	}
	if len(vistos) != len(dsns) {
		return ErrConfiguracionPostgreSQLAdministracionNoSeparada
	}
	return nil
}

func (c ConfiguracionPostgreSQLAdministracion) DSNCorreo() (string, error) {
	if err := c.Validar(); err != nil {
		return "", err
	}
	return c.normalizar().dsnCorreo, nil
}
func (c ConfiguracionPostgreSQLAdministracion) DSNRegistroAutorizacion() (string, error) {
	if err := c.Validar(); err != nil {
		return "", err
	}
	return c.normalizar().dsnRegistroAutorizacion, nil
}
func (c ConfiguracionPostgreSQLAdministracion) DSNIdentidad() (registro, revalidacion, contexto string, err error) {
	if err = c.Validar(); err != nil {
		return "", "", "", err
	}
	c = c.normalizar()
	return c.dsnRegistroIdentidad, c.dsnRevalidacion, c.dsnContextoActor, nil
}
func (c ConfiguracionPostgreSQLAdministracion) normalizar() ConfiguracionPostgreSQLAdministracion {
	c.dsnCorreo = strings.TrimSpace(c.dsnCorreo)
	c.dsnRegistroAutorizacion = strings.TrimSpace(c.dsnRegistroAutorizacion)
	c.dsnRegistroIdentidad = strings.TrimSpace(c.dsnRegistroIdentidad)
	c.dsnRevalidacion = strings.TrimSpace(c.dsnRevalidacion)
	c.dsnContextoActor = strings.TrimSpace(c.dsnContextoActor)
	return c
}
func (ConfiguracionPostgreSQLAdministracion) String() string {
	return configuracionPostgreSQLAdministracionRedactada
}
func (ConfiguracionPostgreSQLAdministracion) GoString() string {
	return configuracionPostgreSQLAdministracionRedactada
}
func (ConfiguracionPostgreSQLAdministracion) Format(s fmt.State, _ rune) {
	_, _ = s.Write([]byte(configuracionPostgreSQLAdministracionRedactada))
}
func (ConfiguracionPostgreSQLAdministracion) MarshalJSON() ([]byte, error) {
	return json.Marshal(configuracionPostgreSQLAdministracionRedactada)
}
func (ConfiguracionPostgreSQLAdministracion) MarshalText() ([]byte, error) {
	return []byte(configuracionPostgreSQLAdministracionRedactada), nil
}
func (ConfiguracionPostgreSQLAdministracion) LogValue() slog.Value {
	return slog.StringValue(configuracionPostgreSQLAdministracionRedactada)
}
