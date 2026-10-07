package domain

import (
	"errors"
	"time"

	cal "vec-diputacion-granada/internal/modules/calendarios/domain"
)

var ErrIncidenciasPeriodoInvalido = errors.New("cronos.incidencias_periodo.entrada_invalida")

const (
	CoberturaRegistroCompleta     = "complete"
	CoberturaRegistroIncompleta   = "incomplete"
	CoberturaRegistroNoDisponible = "not_available"
)

type CoberturaRegistro struct {
	Fecha  string `json:"fecha"`
	Estado string `json:"estado"`
}

type HechoRegistroIncidencia struct {
	Ref         string    `json:"ref"`
	Version     int64     `json:"version"`
	Movimiento  PunchKind `json:"movimiento"`
	InstanteUTC time.Time `json:"instante_utc"`
}

type ConsultaIncidenciasPeriodo struct {
	Desde                 string                    `json:"desde"`
	Hasta                 string                    `json:"hasta"`
	Zona                  string                    `json:"zona"`
	CorteUTC              time.Time                 `json:"corte_utc"`
	AntecedentesCompletos bool                      `json:"antecedentes_completos"`
	Cobertura             []CoberturaRegistro       `json:"cobertura"`
	Hechos                []HechoRegistroIncidencia `json:"hechos"`
}

type RegistroDiaIncidencias struct {
	Fecha             string     `json:"fecha"`
	Cobertura         string     `json:"cobertura"`
	Estado            string     `json:"estado"`
	HechosRegistrados *int       `json:"hechos_registrados"`
	EvaluadoHastaUTC  *time.Time `json:"evaluado_hasta_utc"`
}

// ConsultarIncidenciasPeriodo reports source coverage independently of record
// sequence. It does not infer absence, attendance, expected hours or a cause.
// The only sequence classification is the existing saldo engine's Incompleto.
func ConsultarIncidenciasPeriodo(q ConsultaIncidenciasPeriodo) ([]RegistroDiaIncidencias, error) {
	inicio, e1 := time.Parse("2006-01-02", q.Desde)
	fin, e2 := time.Parse("2006-01-02", q.Hasta)
	if e1 != nil || e2 != nil || inicio.Format("2006-01-02") != q.Desde || fin.Format("2006-01-02") != q.Hasta || !fin.After(inicio) || fin.Sub(inicio) > 366*24*time.Hour || (q.Zona == "" || q.Zona == "Local" || len(q.Zona) > 128) || !instanteIncidenciasValido(q.CorteUTC) || len(q.Hechos) > 10000 || len(q.Cobertura) > 366 {
		return nil, ErrIncidenciasPeriodoInvalido
	}
	zona, err := time.LoadLocation(q.Zona)
	if err != nil {
		return nil, err
	}
	inicioUTC := time.Date(inicio.Year(), inicio.Month(), inicio.Day(), 0, 0, 0, 0, zona).UTC()
	finUTC := time.Date(fin.Year(), fin.Month(), fin.Day(), 0, 0, 0, 0, zona).UTC()
	cobertura := map[string]string{}
	for _, c := range q.Cobertura {
		if c.Fecha < q.Desde || c.Fecha >= q.Hasta || cobertura[c.Fecha] != "" {
			return nil, ErrIncidenciasPeriodoInvalido
		}
		fecha, err := time.Parse("2006-01-02", c.Fecha)
		if err != nil || fecha.Format("2006-01-02") != c.Fecha {
			return nil, ErrIncidenciasPeriodoInvalido
		}
		switch c.Estado {
		case CoberturaRegistroCompleta, CoberturaRegistroIncompleta, CoberturaRegistroNoDisponible:
		default:
			return nil, ErrIncidenciasPeriodoInvalido
		}
		cobertura[c.Fecha] = c.Estado
	}
	hechos := make([]HechoSaldo, 0, len(q.Hechos))
	referencias := map[string]bool{}
	instantes := map[time.Time]bool{}
	recuentos := map[string]int{}
	for _, h := range q.Hechos {
		if !cal.ReferenciaValida(h.Ref) || referencias[h.Ref] || h.Version < 1 || !instanteIncidenciasValido(h.InstanteUTC) || (!h.InstanteUTC.Before(q.CorteUTC) || !h.InstanteUTC.Before(finUTC)) || instantes[h.InstanteUTC] {
			return nil, ErrIncidenciasPeriodoInvalido
		}
		switch h.Movimiento {
		case PunchEntry, PunchExit, PunchPauseStart, PunchPauseEnd:
		default:
			return nil, ErrIncidenciasPeriodoInvalido
		}
		referencias[h.Ref] = true
		instantes[h.InstanteUTC] = true
		hechos = append(hechos, HechoSaldo{Movimiento: h.Movimiento, InstanteUTC: h.InstanteUTC})
		recuentos[h.InstanteUTC.In(zona).Format("2006-01-02")]++
	}
	evaluadoHasta := finUTC
	if q.CorteUTC.Before(evaluadoHasta) {
		evaluadoHasta = q.CorteUTC
	}
	tiempos := map[string]TiempoSaldoDia{}
	if evaluadoHasta.After(inicioUTC) {
		tiempos, err = CalcularTiempoSaldo(hechos, zona, inicioUTC, evaluadoHasta)
		if err != nil {
			return nil, err
		}
	}
	dias := []RegistroDiaIncidencias{}
	confiable := q.AntecedentesCompletos
	// A gap anywhere in the evaluated source can change earlier sequence states
	// (for example, a later exit can close an earlier entry). V1 conservatively
	// leaves every evaluated day unknown rather than segmenting partial sources.
	for fecha := inicio; fecha.Before(fin); fecha = fecha.AddDate(0, 0, 1) {
		diaUTC := time.Date(fecha.Year(), fecha.Month(), fecha.Day(), 0, 0, 0, 0, zona).UTC()
		if diaUTC.Before(evaluadoHasta) && cobertura[fecha.Format("2006-01-02")] != CoberturaRegistroCompleta {
			confiable = false
		}
	}
	for fecha := inicio; fecha.Before(fin); fecha = fecha.AddDate(0, 0, 1) {
		clave := fecha.Format("2006-01-02")
		c := cobertura[clave]
		if c == "" {
			c = CoberturaRegistroIncompleta
		}
		d := RegistroDiaIncidencias{Fecha: clave, Cobertura: c, Estado: "indeterminado"}
		diaUTC := time.Date(fecha.Year(), fecha.Month(), fecha.Day(), 0, 0, 0, 0, zona).UTC()
		siguienteUTC := time.Date(fecha.Year(), fecha.Month(), fecha.Day()+1, 0, 0, 0, 0, zona).UTC()
		if !diaUTC.Before(evaluadoHasta) {
			d.Estado = "no_evaluado"
		} else {
			limite := evaluadoHasta
			if siguienteUTC.Before(limite) {
				limite = siguienteUTC
			}
			d.EvaluadoHastaUTC = &limite
			if c == CoberturaRegistroCompleta && confiable {
				n := recuentos[clave]
				d.HechosRegistrados = &n
				_, tieneTramo := tiempos[clave]
				switch {
				case tiempos[clave].Incompleto:
					d.Estado = "registro_incompleto"
				case n > 0 || tieneTramo:
					d.Estado = "registrado"
				default:
					d.Estado = "sin_registros"
				}
			}
		}
		dias = append(dias, d)
	}
	return dias, nil
}

func instanteIncidenciasValido(t time.Time) bool {
	return !t.IsZero() && t.Location() == time.UTC && t.Nanosecond()%1000 == 0
}
