package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

// Entrega del histórico de contratos (B13): Contratación temporal publica las
// incorporaciones y un relevo las lleva al inbox de Bolsa. Cadencia y lote
// son decisiones de operación, no reglas de negocio: se fijan aquí y se
// pueden cambiar sin tocar código. Ya no hay ventana de relectura: CT publica
// con marca de agua (000113) y ningún evento confirma por detrás del cursor;
// VEC_BOLSA_CONTRATOS_CT_RELECTURA, si sigue en un entorno, se ignora.
const (
	EnvBolsaContratosCTIntervalo = "VEC_BOLSA_CONTRATOS_CT_INTERVALO"
	EnvBolsaContratosCTLote      = "VEC_BOLSA_CONTRATOS_CT_LOTE"

	DefaultBolsaContratosCTIntervalo = 30 * time.Second
	DefaultBolsaContratosCTLote      = 50
)

var ErrEntregaContratosCTBolsaInvalida = errors.New("config: entrega de contratos CT a Bolsa invalida")

// ConfiguracionEntregaContratosCTBolsa conserva los valores tal como llegan
// del entorno; Resolver los valida al componer.
type ConfiguracionEntregaContratosCTBolsa struct {
	intervalo, lote string
}

// EntregaContratosCTBolsa es la configuración ya validada del relevo.
type EntregaContratosCTBolsa struct {
	// Activa es false solo si el intervalo se fija expresamente a "0".
	Activa    bool
	Intervalo time.Duration
	Lote      int
}

func cargarEntregaContratosCTBolsa() ConfiguracionEntregaContratosCTBolsa {
	return ConfiguracionEntregaContratosCTBolsa{
		intervalo: strings.TrimSpace(os.Getenv(EnvBolsaContratosCTIntervalo)),
		lote:      strings.TrimSpace(os.Getenv(EnvBolsaContratosCTLote)),
	}
}

// NuevaConfiguracionEntregaContratosCTBolsa permite fijar los valores en
// pruebas y composiciones sin variables de entorno.
func NuevaConfiguracionEntregaContratosCTBolsa(intervalo, lote string) ConfiguracionEntregaContratosCTBolsa {
	return ConfiguracionEntregaContratosCTBolsa{intervalo: intervalo, lote: lote}
}

// Resolver aplica valores por defecto y rangos cerrados. Un valor mal escrito
// es un error: nunca se sustituye en silencio por otro.
func (c ConfiguracionEntregaContratosCTBolsa) Resolver() (EntregaContratosCTBolsa, error) {
	r := EntregaContratosCTBolsa{Activa: true, Intervalo: DefaultBolsaContratosCTIntervalo, Lote: DefaultBolsaContratosCTLote}
	if c.intervalo == "0" {
		return EntregaContratosCTBolsa{}, nil
	}
	if c.intervalo != "" {
		d, err := time.ParseDuration(c.intervalo)
		if err != nil || d < time.Second || d > time.Hour {
			return EntregaContratosCTBolsa{}, ErrEntregaContratosCTBolsaInvalida
		}
		r.Intervalo = d
	}
	if c.lote != "" {
		n, err := strconv.Atoi(c.lote)
		if err != nil || n < 1 || n > 100 {
			return EntregaContratosCTBolsa{}, ErrEntregaContratosCTBolsaInvalida
		}
		r.Lote = n
	}
	return r, nil
}
