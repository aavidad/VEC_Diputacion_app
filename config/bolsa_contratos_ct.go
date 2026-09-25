package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

// Entrega del histórico de contratos (B13): Contratación temporal publica las
// incorporaciones y un relevo las lleva al inbox de Bolsa. Cadencia, lote y
// ventana de relectura son decisiones de operación, no reglas de negocio:
// se fijan aquí y se pueden cambiar sin tocar código.
const (
	EnvBolsaContratosCTIntervalo = "VEC_BOLSA_CONTRATOS_CT_INTERVALO"
	EnvBolsaContratosCTLote      = "VEC_BOLSA_CONTRATOS_CT_LOTE"
	EnvBolsaContratosCTRelectura = "VEC_BOLSA_CONTRATOS_CT_RELECTURA"

	DefaultBolsaContratosCTIntervalo = 30 * time.Second
	DefaultBolsaContratosCTLote      = 50
	DefaultBolsaContratosCTRelectura = 10 * time.Minute
)

var ErrEntregaContratosCTBolsaInvalida = errors.New("config: entrega de contratos CT a Bolsa invalida")

// ConfiguracionEntregaContratosCTBolsa conserva los valores tal como llegan
// del entorno; Resolver los valida al componer.
type ConfiguracionEntregaContratosCTBolsa struct {
	intervalo, lote, relectura string
}

// EntregaContratosCTBolsa es la configuración ya validada del relevo.
type EntregaContratosCTBolsa struct {
	// Activa es false solo si el intervalo se fija expresamente a "0".
	Activa    bool
	Intervalo time.Duration
	Lote      int
	// Relectura es la ventana hacia atrás desde el último evento recibido.
	// Cubre confirmaciones CT fuera de orden; el inbox absorbe duplicados.
	Relectura time.Duration
}

func cargarEntregaContratosCTBolsa() ConfiguracionEntregaContratosCTBolsa {
	return ConfiguracionEntregaContratosCTBolsa{
		intervalo: strings.TrimSpace(os.Getenv(EnvBolsaContratosCTIntervalo)),
		lote:      strings.TrimSpace(os.Getenv(EnvBolsaContratosCTLote)),
		relectura: strings.TrimSpace(os.Getenv(EnvBolsaContratosCTRelectura)),
	}
}

// NuevaConfiguracionEntregaContratosCTBolsa permite fijar los valores en
// pruebas y composiciones sin variables de entorno.
func NuevaConfiguracionEntregaContratosCTBolsa(intervalo, lote, relectura string) ConfiguracionEntregaContratosCTBolsa {
	return ConfiguracionEntregaContratosCTBolsa{intervalo: intervalo, lote: lote, relectura: relectura}
}

// Resolver aplica valores por defecto y rangos cerrados. Un valor mal escrito
// es un error: nunca se sustituye en silencio por otro.
func (c ConfiguracionEntregaContratosCTBolsa) Resolver() (EntregaContratosCTBolsa, error) {
	r := EntregaContratosCTBolsa{Activa: true, Intervalo: DefaultBolsaContratosCTIntervalo, Lote: DefaultBolsaContratosCTLote, Relectura: DefaultBolsaContratosCTRelectura}
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
	if c.relectura != "" {
		d, err := time.ParseDuration(c.relectura)
		if err != nil || d < 0 || d > 24*time.Hour {
			return EntregaContratosCTBolsa{}, ErrEntregaContratosCTBolsaInvalida
		}
		r.Relectura = d
	}
	return r, nil
}
