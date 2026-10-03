package observabilidad

import (
	"encoding/json"
	"io"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

type contadorAlertas struct {
	cfg        ConfiguracionRecolector
	inicio     time.Time
	cantidades map[domain.CodigoIncidenciaTecnica]uint64
	avisados   map[domain.CodigoIncidenciaTecnica]bool
}

func (c *contadorAlertas) registrar(l lineaIncidencia, ahora time.Time, destino io.Writer, m *MetricasRecolector) error {
	if ahora.Sub(c.inicio) >= time.Duration(c.cfg.VentanaSegundos)*time.Second {
		clear(c.cantidades)
		clear(c.avisados)
		c.inicio = ahora
	}
	codigo := domain.CodigoIncidenciaTecnica(l.Codigo)
	if umbral, configurado := c.cfg.Umbrales[codigo]; configurado {
		c.cantidades[codigo] = sumarSaturado(c.cantidades[codigo], uint64(l.Recuento))
		if c.cantidades[codigo] >= umbral && !c.avisados[codigo] {
			// La alerta solo incluye códigos y valores cerrados ya validados.
			if err := json.NewEncoder(destino).Encode(struct {
				Esquema         string                         `json:"esquema"`
				Instante        string                         `json:"instante"`
				Codigo          domain.CodigoIncidenciaTecnica `json:"codigo"`
				Severidad       string                         `json:"severidad"`
				Recuento        uint64                         `json:"recuento"`
				VentanaSegundos int64                          `json:"ventana_segundos"`
			}{"vec.alerta_tecnica.v1", ahora.UTC().Format(formatoInstante), codigo, l.Severidad, c.cantidades[codigo], c.cfg.VentanaSegundos}); err != nil {
				return err
			}
			c.avisados[codigo] = true
			m.Alertas++
		}
	}
	return nil
}
