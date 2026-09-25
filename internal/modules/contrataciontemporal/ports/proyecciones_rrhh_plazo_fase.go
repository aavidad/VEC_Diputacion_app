package ports

import (
	"context"
	"regexp"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// EstadoPlazoFaseRRHH resume el vencimiento de la fase actual respecto al
// instante de la consulta, en hora peninsular.
type EstadoPlazoFaseRRHH string

const (
	PlazoFaseEnPlazo  EstadoPlazoFaseRRHH = "en_plazo"
	PlazoFaseVenceHoy EstadoPlazoFaseRRHH = "vence_hoy"
	PlazoFaseVencido  EstadoPlazoFaseRRHH = "vencido"
	// PlazoFaseNoCalculado: no se pudo calcular (catálogo o calendario no
	// disponibles). Se muestra como tal; nunca se supone una fecha.
	PlazoFaseNoCalculado EstadoPlazoFaseRRHH = "no_calculado"
)

var patronDiaCivilPlazoFaseRRHH = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}$`)

// PlazoFaseRRHH es el vencimiento de la fase actual de un expediente. Qué
// fases tienen plazo, su cantidad, unidad y cómputo proceden del catálogo de
// reglas; ReglaRef identifica la entrada exacta (catalogo:version:entrada).
type PlazoFaseRRHH struct {
	UltimoDia    string
	VenceAntesDe time.Time
	Estado       EstadoPlazoFaseRRHH
	ReglaRef     string
	ReglaEjemplo bool
}

// Valido comprueba la forma; la coherencia con el calendario es del cálculo.
func (p PlazoFaseRRHH) Valido() bool {
	if p.Estado == PlazoFaseNoCalculado {
		return p == PlazoFaseRRHH{Estado: PlazoFaseNoCalculado}
	}
	return patronDiaCivilPlazoFaseRRHH.MatchString(p.UltimoDia) &&
		!p.VenceAntesDe.IsZero() && p.ReglaRef != "" && len(p.ReglaRef) <= 400 &&
		(p.Estado == PlazoFaseEnPlazo || p.Estado == PlazoFaseVenceHoy ||
			p.Estado == PlazoFaseVencido)
}

// SolicitudPlazoFaseRRHH pide el vencimiento de una fase en la que el
// expediente entró en Desde, evaluado en Ahora.
type SolicitudPlazoFaseRRHH struct {
	Fase  domain.ClaveFase
	Desde time.Time
	Ahora time.Time
}

// CalculadoraPlazoFaseRRHH resuelve el plazo de una fase con el catálogo de
// reglas. Devuelve aplicable=false si ninguna regla vigente cubre la fase. Un
// error significa que no se pudo calcular: nunca se sustituye por un plazo.
type CalculadoraPlazoFaseRRHH interface {
	CalcularPlazoFase(context.Context, SolicitudPlazoFaseRRHH) (plazo PlazoFaseRRHH, aplicable bool, err error)
}

// fasesDesdeValidas exige que la fecha de entrada en fase (CT-000110), si
// viene, acompañe a cada resumen y no sea anterior a su alta ni posterior a su
// última actualización.
func (p PaginaCuadroRRHH) fasesDesdeValidas() bool {
	if len(p.FasesDesde) == 0 {
		return len(p.Plazos) == 0
	}
	if len(p.FasesDesde) != len(p.Expedientes) ||
		(len(p.Plazos) != 0 && len(p.Plazos) != len(p.Expedientes)) {
		return false
	}
	for indice, desde := range p.FasesDesde {
		resumen := p.Expedientes[indice]
		if !domain.InstanteUTCCanonico(desde) || desde.Before(resumen.CreadoEn) ||
			desde.After(resumen.ActualizadoEn) {
			return false
		}
		if len(p.Plazos) != 0 && p.Plazos[indice] != nil && !p.Plazos[indice].Valido() {
			return false
		}
	}
	return true
}
