package domain

import (
	"errors"
	"sort"
	"time"
)

var ErrSecuenciaSaldoInvalida = errors.New("cronos secuencia de saldo invalida")

// HechoSaldo contains only the immutable time fact needed for calculation.
type HechoSaldo struct {
	Movimiento  PunchKind
	InstanteUTC time.Time
}

type TiempoSaldoDia struct {
	Trabajado  time.Duration
	Pausa      time.Duration
	Incompleto bool
}

// CalcularTiempoSaldo divides closed work and pause intervals at civil midnights.
// The UTC bounds represent the requested civil dates in zona. An open entry
// marks their intersection without extrapolating worked time.
// Elapsed UTC duration is stable across daylight saving changes.
func CalcularTiempoSaldo(hechos []HechoSaldo, zona *time.Location, inicioInclusivoUTC, finExclusivoUTC time.Time) (map[string]TiempoSaldoDia, error) {
	if zona == nil || len(hechos) > 10000 || inicioInclusivoUTC.IsZero() || finExclusivoUTC.IsZero() || !finExclusivoUTC.After(inicioInclusivoUTC) || finExclusivoUTC.Sub(inicioInclusivoUTC) > 369*24*time.Hour || inicioInclusivoUTC.Location() != time.UTC || inicioInclusivoUTC.Nanosecond()%1000 != 0 || finExclusivoUTC.Location() != time.UTC || finExclusivoUTC.Nanosecond()%1000 != 0 {
		return nil, ErrSecuenciaSaldoInvalida
	}
	ordenados := append([]HechoSaldo(nil), hechos...)
	for _, h := range ordenados {
		if h.InstanteUTC.IsZero() || h.InstanteUTC.Nanosecond()%1000 != 0 {
			return nil, ErrSecuenciaSaldoInvalida
		}
		switch h.Movimiento {
		case PunchEntry, PunchExit, PunchPauseStart, PunchPauseEnd:
		default:
			return nil, ErrSecuenciaSaldoInvalida
		}
	}
	sort.SliceStable(ordenados, func(i, j int) bool { return ordenados[i].InstanteUTC.Before(ordenados[j].InstanteUTC) })
	dias := make(map[string]TiempoSaldoDia)
	ventana := ventanaSaldo{dias: dias, zona: zona, inicio: inicioInclusivoUTC, fin: finExclusivoUTC}
	var entrada, tramoTrabajo, inicioPausa time.Time
	for _, h := range ordenados {
		at := h.InstanteUTC
		switch h.Movimiento {
		case PunchEntry:
			if !entrada.IsZero() {
				ventana.marcarTramo(entrada, at)
				continue
			}
			entrada, tramoTrabajo = at, at
		case PunchPauseStart:
			if entrada.IsZero() || !inicioPausa.IsZero() {
				if entrada.IsZero() {
					ventana.marcar(at)
				} else {
					ventana.marcarTramo(entrada, at)
				}
				continue
			}
			ventana.agregarTramo(tramoTrabajo, at, false)
			inicioPausa = at
		case PunchPauseEnd:
			if entrada.IsZero() || inicioPausa.IsZero() {
				if entrada.IsZero() {
					ventana.marcar(at)
				} else {
					ventana.marcarTramo(entrada, at)
				}
				continue
			}
			ventana.agregarTramo(inicioPausa, at, true)
			inicioPausa = time.Time{}
			tramoTrabajo = at
		case PunchExit:
			if entrada.IsZero() {
				ventana.marcar(at)
				continue
			}
			if !inicioPausa.IsZero() {
				ventana.agregarTramo(inicioPausa, at, true)
				ventana.marcarTramo(entrada, at)
			} else {
				ventana.agregarTramo(tramoTrabajo, at, false)
			}
			entrada, tramoTrabajo, inicioPausa = time.Time{}, time.Time{}, time.Time{}
		}
	}
	if !entrada.IsZero() {
		ultimo := finExclusivoUTC.Add(-time.Microsecond)
		ventana.marcarTramo(entrada, ultimo)
	}
	return dias, nil
}

type ventanaSaldo struct {
	dias        map[string]TiempoSaldoDia
	zona        *time.Location
	inicio, fin time.Time
}

func (v ventanaSaldo) marcar(at time.Time) {
	if at.Before(v.inicio) || !at.Before(v.fin) {
		return
	}
	fecha := at.In(v.zona).Format("2006-01-02")
	d := v.dias[fecha]
	d.Incompleto = true
	v.dias[fecha] = d
}

func (v ventanaSaldo) marcarTramo(inicio, fin time.Time) {
	if fin.Before(inicio) || fin.Before(v.inicio) || !inicio.Before(v.fin) {
		return
	}
	if inicio.Before(v.inicio) {
		inicio = v.inicio
	}
	if !fin.Before(v.fin) {
		fin = v.fin.Add(-time.Microsecond)
	}
	local := inicio.In(v.zona)
	dia := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, v.zona)
	ultimo := fin.In(v.zona)
	for !dia.After(ultimo) {
		at := dia
		if at.Before(inicio) {
			at = inicio.In(v.zona)
		}
		v.marcar(at.UTC())
		dia = dia.AddDate(0, 0, 1)
	}
}

func (v ventanaSaldo) agregarTramo(inicio, fin time.Time, pausa bool) {
	if inicio.IsZero() || !fin.After(inicio) {
		return
	}
	if inicio.Before(v.inicio) {
		inicio = v.inicio
	}
	if fin.After(v.fin) {
		fin = v.fin
	}
	if !fin.After(inicio) {
		return
	}
	for actual := inicio; actual.Before(fin); {
		local := actual.In(v.zona)
		siguiente := time.Date(local.Year(), local.Month(), local.Day()+1, 0, 0, 0, 0, v.zona)
		corte := fin
		if siguiente.Before(fin) {
			corte = siguiente
		}
		fecha := local.Format("2006-01-02")
		d := v.dias[fecha]
		if pausa {
			d.Pausa += corte.Sub(actual)
		} else {
			d.Trabajado += corte.Sub(actual)
		}
		v.dias[fecha] = d
		actual = corte
	}
}

// MinutosCompletos makes the precision contract explicit: fractions of a
// minute are retained in elapsed durations until the final presentation.
func MinutosCompletos(d time.Duration) int64 { return int64(d / time.Minute) }
