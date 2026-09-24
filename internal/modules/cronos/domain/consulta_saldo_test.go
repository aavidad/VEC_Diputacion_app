package domain

import (
	"testing"
	"time"
)

func TestCalcularTiempoSaldoDivideTurnoNocturnoEnCambioHorario(t *testing.T) {
	zona, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Fatal(err)
	}
	// On 25 October 2026 Madrid repeats 02:00. Elapsed time is four hours.
	hechos := []HechoSaldo{
		{PunchEntry, time.Date(2026, 10, 24, 23, 0, 0, 0, time.UTC)},
		{PunchPauseStart, time.Date(2026, 10, 25, 1, 0, 0, 0, time.UTC)},
		{PunchPauseEnd, time.Date(2026, 10, 25, 1, 30, 0, 0, time.UTC)},
		{PunchExit, time.Date(2026, 10, 25, 3, 0, 0, 0, time.UTC)},
	}
	dias, err := CalcularTiempoSaldo(hechos, zona, time.Date(2026, 10, 25, 0, 0, 0, 0, zona).UTC(), time.Date(2026, 10, 26, 0, 0, 0, 0, zona).UTC())
	if err != nil {
		t.Fatal(err)
	}
	if got := MinutosCompletos(dias["2026-10-25"].Trabajado); got != 210 {
		t.Fatalf("trabajado=%d", got)
	}
	if got := MinutosCompletos(dias["2026-10-25"].Pausa); got != 30 {
		t.Fatalf("pausa=%d", got)
	}
}

func TestCalcularTiempoSaldoNoExtrapolaEntradaAbierta(t *testing.T) {
	dias, err := CalcularTiempoSaldo([]HechoSaldo{{PunchEntry, time.Date(2026, 9, 24, 23, 0, 0, 0, time.UTC)}}, time.UTC, time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC), time.Date(2026, 9, 26, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	for _, fecha := range []string{"2026-09-24", "2026-09-25"} {
		if d := dias[fecha]; !d.Incompleto || d.Trabajado != 0 {
			t.Fatalf("%s=%+v", fecha, d)
		}
	}
}

func TestCalcularTiempoSaldoEntradaAbiertaCruzaCambioHorarioHastaFinCivil(t *testing.T) {
	zona, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Fatal(err)
	}
	// Entry is 24 Oct 23:00 CEST. 25 Oct repeats 02:00; the bound is
	// 26 Oct 00:00 CET, not a fixed number of UTC hours after the entry.
	dias, err := CalcularTiempoSaldo([]HechoSaldo{{PunchEntry, time.Date(2026, 10, 24, 21, 0, 0, 0, time.UTC)}}, zona, time.Date(2026, 10, 24, 0, 0, 0, 0, zona).UTC(), time.Date(2026, 10, 26, 0, 0, 0, 0, zona).UTC())
	if err != nil {
		t.Fatal(err)
	}
	for _, fecha := range []string{"2026-10-24", "2026-10-25"} {
		if d := dias[fecha]; !d.Incompleto || d.Trabajado != 0 {
			t.Fatalf("%s=%+v", fecha, d)
		}
	}
}

func TestCalcularTiempoSaldoMarcaAmbosDiasDePausaAbiertaNocturnaConCambioHorario(t *testing.T) {
	zona, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Fatal(err)
	}
	// Entry: 24 Oct 23:00 CEST; pause: 25 Oct 01:00 CEST;
	// exit: 25 Oct 02:30 CET. The repeated hour is measured in UTC.
	dias, err := CalcularTiempoSaldo([]HechoSaldo{
		{PunchEntry, time.Date(2026, 10, 24, 21, 0, 0, 0, time.UTC)},
		{PunchPauseStart, time.Date(2026, 10, 24, 23, 0, 0, 0, time.UTC)},
		{PunchExit, time.Date(2026, 10, 25, 1, 30, 0, 0, time.UTC)},
	}, zona, time.Date(2026, 10, 24, 0, 0, 0, 0, zona).UTC(), time.Date(2026, 10, 26, 0, 0, 0, 0, zona).UTC())
	if err != nil {
		t.Fatal(err)
	}
	for _, fecha := range []string{"2026-10-24", "2026-10-25"} {
		if !dias[fecha].Incompleto {
			t.Fatalf("%s no marcado: %+v", fecha, dias[fecha])
		}
	}
	if got := MinutosCompletos(dias["2026-10-25"].Trabajado); got != 60 {
		t.Fatalf("trabajo día B=%d", got)
	}
	if got := MinutosCompletos(dias["2026-10-25"].Pausa); got != 150 {
		t.Fatalf("pausa día B=%d", got)
	}
}

func TestMinutosCompletosNoRedondeaMilisegundos(t *testing.T) {
	if got := MinutosCompletos(59*time.Minute + 999*time.Millisecond); got != 59 {
		t.Fatalf("got=%d", got)
	}
}
