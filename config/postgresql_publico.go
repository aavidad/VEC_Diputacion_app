package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
)

const (
	// EnvBolsaPublicaDatabaseURL identifica exclusivamente la base de datos de
	// proyeccion publica. No debe apuntar a la base interna de RRHH ni compartir
	// su cuenta de servicio.
	EnvBolsaPublicaDatabaseURL = "VEC_BOLSA_PUBLICA_DATABASE_URL"
	// EnvBolsaPublicaManifiestoSHA256 es el testigo externo obligatorio. No se
	// obtiene de PostgreSQL ni dispone de valor por defecto.
	EnvBolsaPublicaManifiestoSHA256 = "VEC_BOLSA_PUBLICA_MANIFIESTO_SHA256"

	configuracionPostgreSQLPublicaRedactada = "configuracion_postgresql_publica_redactada"
)

var ErrConfiguracionPostgreSQLPublicaIncompleta = errors.New(
	"config: falta la conexion PostgreSQL exclusiva de la bolsa publica",
)

var (
	ErrHuellaManifiestoPublicoInvalida = errors.New("config: huella externa del manifiesto publico invalida")
	patronHuellaManifiestoPublico      = regexp.MustCompile(`^[a-f0-9]{64}$`)
)

// ValidarHuellaManifiestoPublico rechaza tambien el testigo de invalidacion
// reservado por PostgreSQL. Nunca puede configurarse como ancla legitima.
func ValidarHuellaManifiestoPublico(valor string) error {
	huella := strings.TrimSpace(valor)
	if !patronHuellaManifiestoPublico.MatchString(huella) || huella == strings.Repeat("0", 64) {
		return ErrHuellaManifiestoPublicoInvalida
	}
	return nil
}

// ConfiguracionPostgreSQLPublica encapsula la credencial de solo lectura para
// que un volcado accidental de Config, un log estructurado o JSON no revele el
// DSN. El valor solo se entrega a la raiz publica mediante DSN().
type ConfiguracionPostgreSQLPublica struct {
	dsn string
}

func NuevaConfiguracionPostgreSQLPublica(dsn string) (ConfiguracionPostgreSQLPublica, error) {
	configuracion := ConfiguracionPostgreSQLPublica{dsn: strings.TrimSpace(dsn)}
	if err := configuracion.Validar(); err != nil {
		return ConfiguracionPostgreSQLPublica{}, err
	}
	return configuracion, nil
}

func (c ConfiguracionPostgreSQLPublica) Validar() error {
	if strings.TrimSpace(c.dsn) == "" {
		return ErrConfiguracionPostgreSQLPublicaIncompleta
	}
	return nil
}

func (c ConfiguracionPostgreSQLPublica) DSN() (string, error) {
	if err := c.Validar(); err != nil {
		return "", err
	}
	return c.dsn, nil
}

func (c ConfiguracionPostgreSQLPublica) normalizar() ConfiguracionPostgreSQLPublica {
	c.dsn = strings.TrimSpace(c.dsn)
	return c
}

func (ConfiguracionPostgreSQLPublica) String() string { return configuracionPostgreSQLPublicaRedactada }
func (ConfiguracionPostgreSQLPublica) GoString() string {
	return configuracionPostgreSQLPublicaRedactada
}

func (ConfiguracionPostgreSQLPublica) Format(estado fmt.State, _ rune) {
	_, _ = estado.Write([]byte(configuracionPostgreSQLPublicaRedactada))
}

func (ConfiguracionPostgreSQLPublica) MarshalJSON() ([]byte, error) {
	return json.Marshal(configuracionPostgreSQLPublicaRedactada)
}

func (ConfiguracionPostgreSQLPublica) MarshalText() ([]byte, error) {
	return []byte(configuracionPostgreSQLPublicaRedactada), nil
}

func (ConfiguracionPostgreSQLPublica) LogValue() slog.Value {
	return slog.StringValue(configuracionPostgreSQLPublicaRedactada)
}
