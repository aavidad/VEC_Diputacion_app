package config

import (
	"errors"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

// El valor es un DSN privado de un LOGIN miembro únicamente de
// vec_autorizacion_fuente; nunca se incluye en Git ni en mensajes de error.
const EnvBolsaB103FuenteAutorizacionDatabaseURL = "VEC_BOLSA_B103_FUENTE_AUTORIZACION_DATABASE_URL"

var ErrFuenteAutorizacionRespuestasBolsaIncompleta = errors.New("bolsa: fuente nominal de respuestas no configurada")

func (c Config) DSNFuenteAutorizacionRespuestasBolsa() (string, error) {
	c = c.Normalize()
	if !c.DevelopmentEnabledByDoubleKey() {
		return "", ErrFuenteAutorizacionRespuestasBolsaIncompleta
	}
	dsn := strings.TrimSpace(os.Getenv(EnvBolsaB103FuenteAutorizacionDatabaseURL))
	if dsn == "" {
		return "", ErrFuenteAutorizacionRespuestasBolsaIncompleta
	}
	parsed, err := pgconn.ParseConfig(dsn)
	if err != nil || parsed.User == "" {
		return "", ErrFuenteAutorizacionRespuestasBolsaIncompleta
	}
	for _, previo := range c.dsnsPostgreSQLConfigurados() {
		if conexionPostgreSQLComparteLogin(dsn, previo) {
			return "", ErrFuenteAutorizacionRespuestasBolsaIncompleta
		}
	}
	return dsn, nil
}
