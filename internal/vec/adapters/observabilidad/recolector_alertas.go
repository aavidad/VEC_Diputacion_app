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
	// Dos posiciones fijas: denegado y no_disponible. No hay etiquetas libres.
	cantidadesResultado [2]uint64
	avisadosResultado   [2]bool
}

func (c *contadorAlertas) actualizarVentana(ahora time.Time) {
	if ahora.Sub(c.inicio) >= time.Duration(c.cfg.VentanaSegundos)*time.Second {
		clear(c.cantidades)
		clear(c.avisados)
		c.cantidadesResultado = [2]uint64{}
		c.avisadosResultado = [2]bool{}
		c.inicio = ahora
	}
}

func (c *contadorAlertas) registrar(l lineaIncidencia, ahora time.Time, destino io.Writer, m *MetricasRecolector) error {
	c.actualizarVentana(ahora)
	codigo := domain.CodigoIncidenciaTecnica(l.Codigo)
	if umbral, configurado := c.cfg.Umbrales[codigo]; configurado {
		c.cantidades[codigo] = sumarSaturado(c.cantidades[codigo], uint64(l.Recuento))
		if c.cantidades[codigo] >= umbral && !c.avisados[codigo] {
			// La alerta solo incluye códigos y valores cerrados ya validados.
			if err := escribirAlertaRecolector(destino, struct {
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

// registrarResultado se llama solo tras validar y guardar la proyección. El
// aviso agrega resultados observados; no identifica una persona ni atribuye
// la indisponibilidad a un servicio concreto.
func (c *contadorAlertas) registrarResultado(l lineaResultado, ahora time.Time, destino io.Writer, m *MetricasRecolector) error {
	if len(c.cfg.UmbralesResultado) == 0 {
		return nil
	}
	c.actualizarVentana(ahora)
	resultado := domain.CodigoResultadoTecnico(l.Resultado)
	var posicion int
	switch resultado {
	case domain.ResultadoTecnicoDenegado:
	case domain.ResultadoTecnicoNoDisponible:
		posicion = 1
	default:
		return nil
	}
	umbral, configurado := c.cfg.UmbralesResultado[resultado]
	if !configurado {
		return nil
	}
	c.cantidadesResultado[posicion] = sumarSaturado(c.cantidadesResultado[posicion], 1)
	if c.cantidadesResultado[posicion] < umbral || c.avisadosResultado[posicion] {
		return nil
	}
	// El nivel pertenece al catálogo del resultado ya validado. El aviso no
	// copia componente, etapa, correlación ni referencia de la línea original.
	if err := escribirAlertaRecolector(destino, struct {
		Esquema         string                        `json:"esquema"`
		Instante        string                        `json:"instante"`
		Resultado       domain.CodigoResultadoTecnico `json:"resultado"`
		Nivel           string                        `json:"nivel"`
		Recuento        uint64                        `json:"recuento"`
		VentanaSegundos int64                         `json:"ventana_segundos"`
	}{"vec.alerta_resultado_tecnico.v1", ahora.UTC().Format(formatoInstante), resultado, l.Nivel, c.cantidadesResultado[posicion], c.cfg.VentanaSegundos}); err != nil {
		return err
	}
	c.avisadosResultado[posicion] = true
	m.Alertas++
	return nil
}

// escribirAlertaRecolector confirma solo una línea entregada completa. Un
// destino que devuelve una escritura corta sin error también supone fallo.
func escribirAlertaRecolector(destino io.Writer, aviso any) error {
	datos, err := json.Marshal(aviso)
	if err != nil {
		return err
	}
	datos = append(datos, '\n')
	n, err := destino.Write(datos)
	if err != nil {
		return err
	}
	if n != len(datos) {
		return io.ErrShortWrite
	}
	return nil
}
