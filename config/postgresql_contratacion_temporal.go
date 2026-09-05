package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
)

const (
	EnvContratacionTemporalDatabaseURL                     = "VEC_CT_DATABASE_URL"
	EnvContratacionTemporalGobiernoDatabaseURL             = "VEC_CT_GOBIERNO_DATABASE_URL"
	EnvContratacionTemporalRegistroAutorizacionDatabaseURL = "VEC_CT_REGISTRO_AUTORIZACION_DATABASE_URL"
	EnvContratacionTemporalConfirmadorDatabaseURL          = "VEC_CT_CONFIRMADOR_DATABASE_URL"
	EnvContratacionTemporalLectorResultadoDatabaseURL      = "VEC_CT_LECTOR_RESULTADO_DATABASE_URL"
	EnvBolsaLlamamientosDatabaseURL                        = "VEC_BOLSA_LLAMAMIENTOS_DATABASE_URL"
	configuracionPostgreSQLContratacionTemporalRedactada   = "configuracion_postgresql_contratacion_temporal_redactada"
)

var (
	ErrConfiguracionPostgreSQLContratacionTemporalIncompleta = errors.New(
		"config: faltan conexiones PostgreSQL separadas para contratacion temporal",
	)
	ErrConfiguracionPostgreSQLContratacionTemporalNoSeparada = errors.New(
		"config: las conexiones PostgreSQL de contratacion temporal no estan separadas",
	)
)

// ConfiguracionPostgreSQLContratacionTemporal separa las cinco identidades
// necesarias para ejecutar, gobernar, registrar autorizaciones, confirmar y
// leer resultados históricos.
// Ninguna representación genérica expone los DSN.
type ConfiguracionPostgreSQLContratacionTemporal struct {
	dsnEjecucion            string
	dsnGobierno             string
	dsnRegistroAutorizacion string
	dsnConfirmador          string
	dsnLectorResultado      string
	dsnBolsaLlamamientos    string
}

func NuevaConfiguracionPostgreSQLContratacionTemporal(
	dsnEjecucion string,
	dsnGobierno string,
	dsnRegistroAutorizacion string,
	dsnConfirmador string,
	dsnLectorResultado string,
) (ConfiguracionPostgreSQLContratacionTemporal, error) {
	configuracion := ConfiguracionPostgreSQLContratacionTemporal{
		dsnEjecucion:            dsnEjecucion,
		dsnGobierno:             dsnGobierno,
		dsnRegistroAutorizacion: dsnRegistroAutorizacion,
		dsnConfirmador:          dsnConfirmador,
		dsnLectorResultado:      dsnLectorResultado,
	}.normalizar()
	if err := configuracion.Validar(); err != nil {
		return ConfiguracionPostgreSQLContratacionTemporal{}, err
	}
	return configuracion, nil
}

func (c ConfiguracionPostgreSQLContratacionTemporal) Validar() error {
	c = c.normalizar()
	if c.dsnEjecucion == "" || c.dsnGobierno == "" || c.dsnRegistroAutorizacion == "" ||
		c.dsnConfirmador == "" || c.dsnLectorResultado == "" {
		return ErrConfiguracionPostgreSQLContratacionTemporalIncompleta
	}
	dsnUnicos := map[string]struct{}{
		c.dsnEjecucion:            {},
		c.dsnGobierno:             {},
		c.dsnRegistroAutorizacion: {},
		c.dsnConfirmador:          {},
		c.dsnLectorResultado:      {},
	}
	if len(dsnUnicos) != 5 {
		return ErrConfiguracionPostgreSQLContratacionTemporalNoSeparada
	}
	return nil
}

// DSNRegistroAutorizacionSeparado entrega exclusivamente la identidad que
// confirma decisiones V3; nunca comparte credencial con fuente o consumidor.
func (c ConfiguracionPostgreSQLContratacionTemporal) DSNRegistroAutorizacionSeparado() (
	string,
	error,
) {
	c = c.normalizar()
	if err := c.Validar(); err != nil {
		return "", err
	}
	return c.dsnRegistroAutorizacion, nil
}

func (c ConfiguracionPostgreSQLContratacionTemporal) DSNSeparados() (
	ejecucion string,
	gobierno string,
	err error,
) {
	c = c.normalizar()
	if err := c.Validar(); err != nil {
		return "", "", err
	}
	return c.dsnEjecucion, c.dsnGobierno, nil
}

// DSNCoberturaSeparados entrega exclusivamente las conexiones adicionales de
// cobertura a la raíz de composición, después de validar las cinco.
func (c ConfiguracionPostgreSQLContratacionTemporal) DSNCoberturaSeparados() (
	confirmador string,
	lectorResultado string,
	err error,
) {
	c = c.normalizar()
	if err := c.Validar(); err != nil {
		return "", "", err
	}
	return c.dsnConfirmador, c.dsnLectorResultado, nil
}

func (c ConfiguracionPostgreSQLContratacionTemporal) normalizar() ConfiguracionPostgreSQLContratacionTemporal {
	c.dsnEjecucion = strings.TrimSpace(c.dsnEjecucion)
	c.dsnGobierno = strings.TrimSpace(c.dsnGobierno)
	c.dsnRegistroAutorizacion = strings.TrimSpace(c.dsnRegistroAutorizacion)
	c.dsnConfirmador = strings.TrimSpace(c.dsnConfirmador)
	c.dsnLectorResultado = strings.TrimSpace(c.dsnLectorResultado)
	c.dsnBolsaLlamamientos = strings.TrimSpace(c.dsnBolsaLlamamientos)
	return c
}

// BolsaLlamamientosConfigurada indica si se ha solicitado componer el paso de
// llamamiento. Su ausencia conserva los pasos anteriores; no simula Bolsa.
func (c ConfiguracionPostgreSQLContratacionTemporal) BolsaLlamamientosConfigurada() bool {
	return strings.TrimSpace(c.dsnBolsaLlamamientos) != ""
}

// DSNBolsaLlamamientosSeparado entrega la conexión del módulo propietario de
// Bolsa. La raíz comprueba además que su identidad real no mezcla roles de CT.
func (c ConfiguracionPostgreSQLContratacionTemporal) DSNBolsaLlamamientosSeparado() (string, error) {
	c = c.normalizar()
	if err := c.Validar(); err != nil {
		return "", err
	}
	if c.dsnBolsaLlamamientos == "" {
		return "", ErrConfiguracionPostgreSQLContratacionTemporalIncompleta
	}
	for _, dsn := range []string{c.dsnEjecucion, c.dsnGobierno, c.dsnRegistroAutorizacion, c.dsnConfirmador, c.dsnLectorResultado} {
		if c.dsnBolsaLlamamientos == dsn {
			return "", ErrConfiguracionPostgreSQLContratacionTemporalNoSeparada
		}
	}
	return c.dsnBolsaLlamamientos, nil
}

func (ConfiguracionPostgreSQLContratacionTemporal) String() string {
	return configuracionPostgreSQLContratacionTemporalRedactada
}
func (ConfiguracionPostgreSQLContratacionTemporal) GoString() string {
	return configuracionPostgreSQLContratacionTemporalRedactada
}
func (ConfiguracionPostgreSQLContratacionTemporal) Format(estado fmt.State, _ rune) {
	_, _ = estado.Write([]byte(configuracionPostgreSQLContratacionTemporalRedactada))
}
func (ConfiguracionPostgreSQLContratacionTemporal) MarshalJSON() ([]byte, error) {
	return json.Marshal(configuracionPostgreSQLContratacionTemporalRedactada)
}
func (ConfiguracionPostgreSQLContratacionTemporal) MarshalText() ([]byte, error) {
	return []byte(configuracionPostgreSQLContratacionTemporalRedactada), nil
}
func (ConfiguracionPostgreSQLContratacionTemporal) LogValue() slog.Value {
	return slog.StringValue(configuracionPostgreSQLContratacionTemporalRedactada)
}
