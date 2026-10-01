package domain

import (
	"reflect"
	"testing"
	"time"
)

func TestEstadoRegistradoNoExtrapolaHorasNiRecuperaAmbiguedad(t *testing.T) {
	inicio := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	corte := inicio.Add(9 * time.Hour)
	marca := func(k PunchKind, h int) HechoSaldo {
		return HechoSaldo{Movimiento: k, InstanteUTC: inicio.Add(time.Duration(h) * time.Hour)}
	}
	for _, tc := range []struct {
		nombre   string
		marcas   []HechoSaldo
		completo bool
		estado   EstadoPresencia
		causa    CausaPresencia
	}{
		{"abierta", []HechoSaldo{marca(PunchEntry, 6)}, true, PresenciaRegistrada, ""},
		{"pausa", []HechoSaldo{marca(PunchEntry, 6), marca(PunchPauseStart, 7)}, true, PausaRegistrada, ""},
		{"fin pausa", []HechoSaldo{marca(PunchEntry, 6), marca(PunchPauseStart, 7), marca(PunchPauseEnd, 8)}, true, PresenciaRegistrada, ""},
		{"salida", []HechoSaldo{marca(PunchEntry, 6), marca(PunchExit, 8)}, true, SalidaRegistrada, ""},
		{"reentrada", []HechoSaldo{marca(PunchEntry, 4), marca(PunchExit, 5), marca(PunchEntry, 6)}, true, PresenciaRegistrada, ""},
		{"sin marcas", []HechoSaldo{}, true, PresenciaIndeterminada, SinMarcajes},
		{"sin marcas parcial", []HechoSaldo{}, false, PresenciaIndeterminada, CoberturaIncompleta},
		{"ambigua parcial", []HechoSaldo{marca(PunchExit, 7)}, false, PresenciaIndeterminada, CoberturaIncompleta},
		{"parcial", []HechoSaldo{marca(PunchEntry, 6)}, false, PresenciaIndeterminada, CoberturaIncompleta},
		{"entrada duplicada", []HechoSaldo{marca(PunchEntry, 6), marca(PunchEntry, 7), marca(PunchExit, 8)}, true, PresenciaIndeterminada, SecuenciaAmbigua},
		{"salida sin entrada", []HechoSaldo{marca(PunchExit, 7)}, true, PresenciaIndeterminada, SecuenciaAmbigua},
		{"pausa sin entrada", []HechoSaldo{marca(PunchPauseStart, 7)}, true, PresenciaIndeterminada, SecuenciaAmbigua},
		{"fin sin pausa", []HechoSaldo{marca(PunchEntry, 6), marca(PunchPauseEnd, 7)}, true, PresenciaIndeterminada, SecuenciaAmbigua},
		{"salida en pausa", []HechoSaldo{marca(PunchEntry, 6), marca(PunchPauseStart, 7), marca(PunchExit, 8)}, true, PresenciaIndeterminada, SecuenciaAmbigua},
		{"simultaneas", []HechoSaldo{marca(PunchEntry, 6), marca(PunchExit, 6)}, true, PresenciaIndeterminada, SecuenciaAmbigua},
		{"orden entrada", []HechoSaldo{marca(PunchExit, 8), marca(PunchEntry, 6)}, true, SalidaRegistrada, ""},
		{"exacto corte", []HechoSaldo{marca(PunchEntry, 9)}, true, PresenciaRegistrada, ""},
	} {
		t.Run(tc.nombre, func(t *testing.T) {
			original := append([]HechoSaldo{}, tc.marcas...)
			obtenido, err := EvaluarPresenciaAlCorte(tc.marcas, inicio, corte, tc.completo)
			if err != nil || obtenido.Estado != tc.estado || obtenido.Causa != tc.causa {
				t.Fatalf("%+v %v", obtenido, err)
			}
			estado, err := EstadoRegistradoAlCorte(tc.marcas, inicio, corte, tc.completo)
			if err != nil || estado != obtenido.Estado {
				t.Fatalf("state-only contract changed: %s %v", estado, err)
			}
			if len(original) > 0 && !reflect.DeepEqual(original, tc.marcas) {
				t.Fatal("mutated source")
			}
		})
	}
	for _, hechos := range [][]HechoSaldo{{marca(PunchEntry, 10)}, {marca(PunchEntry, -1)}, {{Movimiento: PunchKind("desconocido"), InstanteUTC: inicio}}, {{Movimiento: PunchEntry, InstanteUTC: inicio.Add(time.Nanosecond)}}} {
		if r, err := EvaluarPresenciaAlCorte(hechos, inicio, corte, false); err != ErrPresenciaInvalida || r != (ResultadoPresencia{}) {
			t.Fatal("invalid source accepted")
		}
	}
}
