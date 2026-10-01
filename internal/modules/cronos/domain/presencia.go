package domain

import (
	"errors"
	"sort"
	"time"
)

var ErrPresenciaInvalida = errors.New("cronos_presencia_invalida")

type EstadoPresencia string

const (
	PresenciaRegistrada    EstadoPresencia = "entrada_registrada"
	PausaRegistrada        EstadoPresencia = "pausa_registrada"
	SalidaRegistrada       EstadoPresencia = "salida_registrada"
	PresenciaIndeterminada EstadoPresencia = "indeterminado"
)

// EstadoRegistradoAlCorte projects typed punches only. It does not attest physical
// presence, authorize a read, calculate work, or infer absence. Coverage is a fact
// of the synthetic source, never an authorization supplied by a caller.
func EstadoRegistradoAlCorte(hechos []HechoSaldo, inicio, corte time.Time, coberturaCompleta bool) (EstadoPresencia, error) {
	if inicio.IsZero() || corte.IsZero() || inicio.Location() != time.UTC || corte.Location() != time.UTC || corte.Before(inicio) || corte.Sub(inicio) > 25*time.Hour || len(hechos) > 100 || inicio.Nanosecond()%1000 != 0 || corte.Nanosecond()%1000 != 0 {
		return "", ErrPresenciaInvalida
	}
	ordenados := append([]HechoSaldo(nil), hechos...)
	for _, h := range ordenados {
		if h.InstanteUTC.IsZero() || h.InstanteUTC.Location() != time.UTC || h.InstanteUTC.Nanosecond()%1000 != 0 || h.InstanteUTC.Before(inicio) || h.InstanteUTC.After(corte) {
			return "", ErrPresenciaInvalida
		}
		switch h.Movimiento {
		case PunchEntry, PunchExit, PunchPauseStart, PunchPauseEnd:
		default:
			return "", ErrPresenciaInvalida
		}
	}
	if !coberturaCompleta || len(hechos) == 0 {
		return PresenciaIndeterminada, nil
	}
	sort.Slice(ordenados, func(i, j int) bool { return ordenados[i].InstanteUTC.Before(ordenados[j].InstanteUTC) })
	estado := PresenciaIndeterminada
	for i, h := range ordenados {
		if i > 0 && h.InstanteUTC.Equal(ordenados[i-1].InstanteUTC) {
			return PresenciaIndeterminada, nil
		}
		switch h.Movimiento {
		case PunchEntry:
			if estado != PresenciaIndeterminada && estado != SalidaRegistrada {
				return PresenciaIndeterminada, nil
			}
			estado = PresenciaRegistrada
		case PunchPauseStart:
			if estado != PresenciaRegistrada {
				return PresenciaIndeterminada, nil
			}
			estado = PausaRegistrada
		case PunchPauseEnd:
			if estado != PausaRegistrada {
				return PresenciaIndeterminada, nil
			}
			estado = PresenciaRegistrada
		case PunchExit:
			if estado != PresenciaRegistrada {
				return PresenciaIndeterminada, nil
			}
			estado = SalidaRegistrada
		}
	}
	return estado, nil
}
