package config

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
)

const (
	EnvBolsaBorradoresEjecutorConsultaDatabaseURL  = "VEC_BOLSA_BORRADORES_EJECUTOR_CONSULTA_DATABASE_URL"
	EnvBolsaBorradoresProyectorGobiernoDatabaseURL = "VEC_BOLSA_BORRADORES_PROYECTOR_GOBIERNO_DATABASE_URL"
	EnvBolsaBorradoresVerificadorReciboDatabaseURL = "VEC_BOLSA_BORRADORES_VERIFICADOR_RECIBO_DATABASE_URL"

	configuracionPostgreSQLBorradoresRedactada = "configuracion_postgresql_borradores_redactada"
)

var (
	ErrConfiguracionPostgreSQLBorradoresIncompleta = errors.New(
		"config: faltan conexiones PostgreSQL separadas para los borradores de bolsa",
	)
	ErrConfiguracionPostgreSQLBorradoresNoSeparada = errors.New(
		"config: las conexiones PostgreSQL de borradores de bolsa no estan separadas",
	)
)

// ConfiguracionPostgreSQLBorradores conserva tres credenciales de minimo
// privilegio. Sus campos permanecen privados y todas sus representaciones
// genericas se redactan para impedir que una URL con contrasena llegue a logs,
// JSON, diagnosticos o mensajes de error.
type ConfiguracionPostgreSQLBorradores struct {
	dsnEjecutorConsulta  string
	dsnProyectorGobierno string
	dsnVerificadorRecibo string
}

// NuevaConfiguracionPostgreSQLBorradores construye una configuracion completa.
// No analiza ni abre las URL: esa validacion pertenece a la fabrica de pools.
func NuevaConfiguracionPostgreSQLBorradores(
	dsnEjecutorConsulta string,
	dsnProyectorGobierno string,
	dsnVerificadorRecibo string,
) (ConfiguracionPostgreSQLBorradores, error) {
	configuracion := ConfiguracionPostgreSQLBorradores{
		dsnEjecutorConsulta:  dsnEjecutorConsulta,
		dsnProyectorGobierno: dsnProyectorGobierno,
		dsnVerificadorRecibo: dsnVerificadorRecibo,
	}.normalizar()
	if err := configuracion.Validar(); err != nil {
		return ConfiguracionPostgreSQLBorradores{}, err
	}
	return configuracion, nil
}

func (c ConfiguracionPostgreSQLBorradores) Validar() error {
	c = c.normalizar()
	if c.dsnEjecutorConsulta == "" || c.dsnProyectorGobierno == "" ||
		c.dsnVerificadorRecibo == "" {
		return ErrConfiguracionPostgreSQLBorradoresIncompleta
	}
	if c.dsnEjecutorConsulta == c.dsnProyectorGobierno ||
		c.dsnEjecutorConsulta == c.dsnVerificadorRecibo ||
		c.dsnProyectorGobierno == c.dsnVerificadorRecibo {
		return ErrConfiguracionPostgreSQLBorradoresNoSeparada
	}
	return nil
}

// DSNSeparados entrega las URL solo a la raiz de composicion, despues de
// revalidar que no falta ninguna y que no se ha reutilizado el mismo valor.
func (c ConfiguracionPostgreSQLBorradores) DSNSeparados() (
	ejecutorConsulta string,
	proyectorGobierno string,
	verificadorRecibo string,
	err error,
) {
	c = c.normalizar()
	if err := c.Validar(); err != nil {
		return "", "", "", err
	}
	return c.dsnEjecutorConsulta, c.dsnProyectorGobierno, c.dsnVerificadorRecibo, nil
}

func (c ConfiguracionPostgreSQLBorradores) normalizar() ConfiguracionPostgreSQLBorradores {
	c.dsnEjecutorConsulta = strings.TrimSpace(c.dsnEjecutorConsulta)
	c.dsnProyectorGobierno = strings.TrimSpace(c.dsnProyectorGobierno)
	c.dsnVerificadorRecibo = strings.TrimSpace(c.dsnVerificadorRecibo)
	return c
}

func (ConfiguracionPostgreSQLBorradores) String() string {
	return configuracionPostgreSQLBorradoresRedactada
}

func (ConfiguracionPostgreSQLBorradores) GoString() string {
	return configuracionPostgreSQLBorradoresRedactada
}

func (ConfiguracionPostgreSQLBorradores) Format(estado fmt.State, _ rune) {
	_, _ = estado.Write([]byte(configuracionPostgreSQLBorradoresRedactada))
}

func (ConfiguracionPostgreSQLBorradores) MarshalJSON() ([]byte, error) {
	return []byte(`"configuracion_postgresql_borradores_redactada"`), nil
}

func (ConfiguracionPostgreSQLBorradores) MarshalText() ([]byte, error) {
	return []byte(configuracionPostgreSQLBorradoresRedactada), nil
}

func (ConfiguracionPostgreSQLBorradores) LogValue() slog.Value {
	return slog.StringValue(configuracionPostgreSQLBorradoresRedactada)
}
