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

// CausaPresencia explains an undetermined projection, not a reason for absence.
type CausaPresencia string

const (
	CoberturaIncompleta CausaPresencia = "cobertura_incompleta"
	SinMarcajes         CausaPresencia = "sin_marcajes"
	SecuenciaAmbigua    CausaPresencia = "secuencia_ambigua"
)

type ResultadoPresencia struct {
	Estado EstadoPresencia
	Causa  CausaPresencia
}

// EstadoRegistradoAlCorte preserves the state-only contract.
func EstadoRegistradoAlCorte(hechos []HechoSaldo, inicio, corte time.Time, coberturaCompleta bool) (EstadoPresencia, error) {
	r, err := EvaluarPresenciaAlCorte(hechos, inicio, corte, coberturaCompleta)
	return r.Estado, err
}

// EvaluarPresenciaAlCorte projects typed punches only. It does not attest physical
// presence, authorize a read, calculate work, or infer absence. Coverage is a fact
// of the synthetic source, never an authorization supplied by a caller.
func EvaluarPresenciaAlCorte(hechos []HechoSaldo, inicio, corte time.Time, coberturaCompleta bool) (ResultadoPresencia, error) {
	if inicio.IsZero() || corte.IsZero() || inicio.Location() != time.UTC || corte.Location() != time.UTC || corte.Before(inicio) || corte.Sub(inicio) > 25*time.Hour || len(hechos) > 100 || inicio.Nanosecond()%1000 != 0 || corte.Nanosecond()%1000 != 0 {
		return ResultadoPresencia{}, ErrPresenciaInvalida
	}
	ordenados := append([]HechoSaldo(nil), hechos...)
	for _, h := range ordenados {
		if h.InstanteUTC.IsZero() || h.InstanteUTC.Location() != time.UTC || h.InstanteUTC.Nanosecond()%1000 != 0 || h.InstanteUTC.Before(inicio) || h.InstanteUTC.After(corte) {
			return ResultadoPresencia{}, ErrPresenciaInvalida
		}
		switch h.Movimiento {
		case PunchEntry, PunchExit, PunchPauseStart, PunchPauseEnd:
		default:
			return ResultadoPresencia{}, ErrPresenciaInvalida
		}
	}
	if !coberturaCompleta {
		return ResultadoPresencia{Estado: PresenciaIndeterminada, Causa: CoberturaIncompleta}, nil
	}
	if len(hechos) == 0 {
		return ResultadoPresencia{Estado: PresenciaIndeterminada, Causa: SinMarcajes}, nil
	}
	sort.Slice(ordenados, func(i, j int) bool { return ordenados[i].InstanteUTC.Before(ordenados[j].InstanteUTC) })
	estado := PresenciaIndeterminada
	for i, h := range ordenados {
		if i > 0 && h.InstanteUTC.Equal(ordenados[i-1].InstanteUTC) {
			return ResultadoPresencia{Estado: PresenciaIndeterminada, Causa: SecuenciaAmbigua}, nil
		}
		switch h.Movimiento {
		case PunchEntry:
			if estado != PresenciaIndeterminada && estado != SalidaRegistrada {
				return ResultadoPresencia{Estado: PresenciaIndeterminada, Causa: SecuenciaAmbigua}, nil
			}
			estado = PresenciaRegistrada
		case PunchPauseStart:
			if estado != PresenciaRegistrada {
				return ResultadoPresencia{Estado: PresenciaIndeterminada, Causa: SecuenciaAmbigua}, nil
			}
			estado = PausaRegistrada
		case PunchPauseEnd:
			if estado != PausaRegistrada {
				return ResultadoPresencia{Estado: PresenciaIndeterminada, Causa: SecuenciaAmbigua}, nil
			}
			estado = PresenciaRegistrada
		case PunchExit:
			if estado != PresenciaRegistrada {
				return ResultadoPresencia{Estado: PresenciaIndeterminada, Causa: SecuenciaAmbigua}, nil
			}
			estado = SalidaRegistrada
		}
	}
	return ResultadoPresencia{Estado: estado}, nil
}
