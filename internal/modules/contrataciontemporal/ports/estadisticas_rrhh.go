package ports

import (
	"context"
	"errors"
	"regexp"
	"time"
)

// Estadísticas de contratación por periodo (C18): agregados sin datos
// personales sobre las versiones publicadas para RRHH hasta el último corte
// global, con el mismo alcance (organización, centro o unidad) que el cuadro.

var (
	ErrEstadisticasRRHHInvalida     = errors.New("estadisticas rrhh: consulta invalida")
	ErrEstadisticasRRHHNoDisponible = errors.New("estadisticas rrhh: no disponibles")
)

const (
	PeriodoEstadisticasRRHHAnual   = "anual"
	PeriodoEstadisticasRRHHMensual = "mensual"
	PeriodoEstadisticasRRHHSemanal = "semanal"

	// Tope de puntos de una serie; alineado con la función SQL.
	maximoPeriodosEstadisticasRRHH = 400
	formatoFechaEstadisticasRRHH   = "2006-01-02"
)

var patronReferenciaAlcanceRRHH = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$`)

// AlcanceEstadisticasRRHH es el ámbito del lector de RRHH que consulta; no
// procede nunca de la petición sino de su identidad acreditada.
type AlcanceEstadisticasRRHH struct {
	OrganizacionRef string
	ClaseAmbito     ClaseAmbitoConsultaRRHH
	AmbitoRef       string
}

func (a AlcanceEstadisticasRRHH) Validar() error {
	if !patronReferenciaAlcanceRRHH.MatchString(a.OrganizacionRef) || !a.ClaseAmbito.valida() ||
		!patronReferenciaAlcanceRRHH.MatchString(a.AmbitoRef) ||
		(a.ClaseAmbito == AmbitoOrganizacionRRHH && a.AmbitoRef != a.OrganizacionRef) {
		return ErrEstadisticasRRHHInvalida
	}
	return nil
}

// ConsultaEstadisticasRRHH: periodo y rango de fechas (ambos inclusive, en
// días naturales de Europe/Madrid).
type ConsultaEstadisticasRRHH struct {
	Periodo string
	Desde   time.Time
	Hasta   time.Time
}

func (c ConsultaEstadisticasRRHH) Validar() error {
	if c.Periodo != PeriodoEstadisticasRRHHAnual && c.Periodo != PeriodoEstadisticasRRHHMensual && c.Periodo != PeriodoEstadisticasRRHHSemanal {
		return ErrEstadisticasRRHHInvalida
	}
	if c.Desde.IsZero() || c.Hasta.IsZero() || c.Desde.After(c.Hasta) ||
		c.Desde.Year() < 2000 || c.Hasta.Year() > 2100 {
		return ErrEstadisticasRRHHInvalida
	}
	dias := int(c.Hasta.Sub(c.Desde).Hours() / 24)
	porPeriodo := map[string]int{PeriodoEstadisticasRRHHAnual: 365, PeriodoEstadisticasRRHHMensual: 28, PeriodoEstadisticasRRHHSemanal: 7}[c.Periodo]
	if dias/porPeriodo > maximoPeriodosEstadisticasRRHH {
		return ErrEstadisticasRRHHInvalida
	}
	return nil
}

// FechaEstadisticasRRHH interpreta una fecha AAAA-MM-DD sin zona.
func FechaEstadisticasRRHH(valor string) (time.Time, error) {
	fecha, err := time.Parse(formatoFechaEstadisticasRRHH, valor)
	if err != nil || fecha.Format(formatoFechaEstadisticasRRHH) != valor {
		return time.Time{}, ErrEstadisticasRRHHInvalida
	}
	return fecha, nil
}

// SerieEstadisticasRRHH es un periodo con sus recuentos. Cada expediente
// cuenta una vez por concepto (en el periodo de su primera versión en esa
// situación); las incidencias cuentan cada entrada en ese estado.
type SerieEstadisticasRRHH struct {
	Inicio          time.Time
	Altas           uint64
	Llamamientos    uint64
	Formalizaciones uint64
	Cierres         uint64
	Incidencias     uint64
}

type EstadisticasRRHH struct {
	CorteGlobal uint64
	Series      []SerieEstadisticasRRHH
}

// Totales suma las series.
func (e EstadisticasRRHH) Totales() SerieEstadisticasRRHH {
	var total SerieEstadisticasRRHH
	for _, serie := range e.Series {
		total.Altas += serie.Altas
		total.Llamamientos += serie.Llamamientos
		total.Formalizaciones += serie.Formalizaciones
		total.Cierres += serie.Cierres
		total.Incidencias += serie.Incidencias
	}
	return total
}

type ConsultorEstadisticasRRHH interface {
	ConsultarEstadisticasRRHH(context.Context, AlcanceEstadisticasRRHH, ConsultaEstadisticasRRHH) (EstadisticasRRHH, error)
}
