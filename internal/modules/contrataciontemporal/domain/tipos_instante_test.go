package domain

import (
	"testing"
	"time"
)

func TestInstanteUTCCanonicoRespetaLimitesSerializables(t *testing.T) {
	t.Parallel()

	maximo := time.Date(
		9999, time.December, 31, 23, 59, 59, 999999000, time.UTC,
	)
	if !InstanteUTCCanonico(maximo) {
		t.Fatal("el último instante serializable debe ser canónico")
	}

	casosInvalidos := map[string]time.Time{
		"año cero":            time.Date(0, time.January, 1, 0, 0, 0, 0, time.UTC),
		"año diez mil":        time.Date(10000, time.January, 1, 0, 0, 0, 0, time.UTC),
		"nanosegundos":        maximo.Add(time.Nanosecond),
		"zona no UTC":         time.Date(2026, time.July, 28, 12, 0, 0, 0, time.FixedZone("CET", 3600)),
		"valor temporal cero": {},
	}
	for nombre, instante := range casosInvalidos {
		instante := instante
		t.Run(nombre, func(t *testing.T) {
			t.Parallel()
			if InstanteUTCCanonico(instante) {
				t.Fatalf("el instante %s no debe ser canónico", nombre)
			}
		})
	}
}
