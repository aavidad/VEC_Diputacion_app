package ports

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestCanonCierreIntencionOfertaCTConservaOrigenCongelado(t *testing.T) {
	t.Parallel()
	comando := json.RawMessage(`{"origen_ct":{"lectura":{"corte":"2026-10-15T12:00:00.000000Z","entradas":[{"participacion_ref":"participacion:uno"}]}},"numero_plazas":2}`)
	actor := "per_0123456789abcdefghijkl"
	canon, err := CanonCierreIntencionOfertaCT(actor, comando)
	if err != nil {
		t.Fatal(err)
	}
	esperado := `{"esquema":"vec.bolsa.oferta-ct.cierre-intencion.v1","actor_ref":"` + actor + `","comando":` + string(comando) + `}`
	if canon != esperado {
		t.Fatalf("canon alteró origen o campo: %s", canon)
	}
	if _, err := CanonCierreIntencionOfertaCT(actor, json.RawMessage(`{"origen_ct":`)); !errors.Is(err, ErrCierreIntencionOfertaCTInvalido) {
		t.Fatalf("comando incompleto aceptado: %v", err)
	}
	if _, err := CanonCierreIntencionOfertaCT(strings.Repeat("a", 25), comando); !errors.Is(err, ErrCierreIntencionOfertaCTInvalido) {
		t.Fatalf("actor sin identidad aceptado: %v", err)
	}
}
