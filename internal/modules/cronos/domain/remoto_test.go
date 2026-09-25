package domain

import "testing"

func TestMovimientosRemotosPermitidosSoloEnumSinDuplicados(t *testing.T) {
	for _, movimientos := range [][]PunchKind{
		{}, {PunchEntry}, {PunchEntry, PunchExit, PunchPauseStart, PunchPauseEnd},
	} {
		if err := ValidarMovimientosRemotosPermitidos(movimientos); err != nil {
			t.Fatal(movimientos, err)
		}
	}
	for _, movimientos := range [][]PunchKind{
		{PunchEntry, PunchEntry}, {PunchKind("otro")},
	} {
		if ValidarMovimientosRemotosPermitidos(movimientos) == nil {
			t.Fatal("secuencia invalida aceptada", movimientos)
		}
	}
}
