package application

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestCursorCarteraNoPermiteCambiarPersonaFiltroCorteNiCaducidad(t *testing.T) {
	s, err := NuevoSelladorCursorCartera()
	if err != nil {
		t.Fatal(err)
	}
	ahora := time.Date(2026, 10, 12, 14, 30, 0, 0, time.UTC)
	corte := ahora.Add(-time.Minute)
	token, err := s.Sellar("persona:uno", "disponibles", "es", 25, corte, "hecho:ultimo", ahora)
	if err != nil {
		t.Fatal(err)
	}
	recuperado, ultimo, err := s.Abrir(token, "persona:uno", "disponibles", "es", 25, ahora.Add(time.Second))
	if err != nil || !recuperado.Equal(corte) || ultimo != "hecho:ultimo" {
		t.Fatalf("cursor propio válido: corte=%v ultimo=%q err=%v", recuperado, ultimo, err)
	}
	otro, err := NuevoSelladorCursorCartera()
	if err != nil {
		t.Fatal(err)
	}
	mutado := []byte(token)
	if mutado[5] == 'A' {
		mutado[5] = 'B'
	} else {
		mutado[5] = 'A'
	}
	casos := []struct {
		nombre, cursor, persona, filtro, idioma string
		limite                                  int
		instante                                time.Time
		sellador                                *SelladorCursorCartera
	}{
		{"persona ajena", token, "persona:dos", "disponibles", "es", 25, ahora, s},
		{"filtro cambiado", token, "persona:uno", "retirados", "es", 25, ahora, s},
		{"idioma cambiado", token, "persona:uno", "disponibles", "en", 25, ahora, s},
		{"limite cambiado", token, "persona:uno", "disponibles", "es", 50, ahora, s},
		{"firma mutada", string(mutado), "persona:uno", "disponibles", "es", 25, ahora, s},
		{"proceso reiniciado", token, "persona:uno", "disponibles", "es", 25, ahora, otro},
		{"caducado", token, "persona:uno", "disponibles", "es", 25, corte.Add(vigenciaCursorCartera), s},
		{"sobredimensionado", strings.Repeat("x", 2049), "persona:uno", "disponibles", "es", 25, ahora, s},
	}
	for _, tc := range casos {
		t.Run(tc.nombre, func(t *testing.T) {
			_, _, err := tc.sellador.Abrir(tc.cursor, tc.persona, tc.filtro, tc.idioma, tc.limite, tc.instante)
			if !errors.Is(err, ErrCursorCartera) {
				t.Fatalf("cursor no denegado: %v", err)
			}
		})
	}
}
