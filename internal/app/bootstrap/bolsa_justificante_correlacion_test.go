package bootstrap

import (
	"context"
	"testing"

	docapp "vec-diputacion-granada/internal/vec/documentos/application"
)

func TestRecuperacionJustificanteLigadaAPersonaBolsaYClave(t *testing.T) {
	base, err := docapp.NuevoDerivadorCorrelacionSubidaLocalHMAC(make([]byte, 32))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	d := correlacionJustificanteBolsa{base, "bolsa:uno"}
	primera, err := d.CorrelacionSubida(ctx, "persona:uno", "clave:uno")
	if err != nil {
		t.Fatal(err)
	}
	repetida, err := d.CorrelacionSubida(ctx, "persona:uno", "clave:uno")
	if err != nil || primera != repetida {
		t.Fatal("la recuperación perdió su clave estable")
	}
	for _, caso := range []struct{ persona, bolsa, clave string }{
		{"persona:dos", "bolsa:uno", "clave:uno"},
		{"persona:uno", "bolsa:dos", "clave:uno"},
		{"persona:uno", "bolsa:uno", "clave:dos"},
	} {
		otra, err := (correlacionJustificanteBolsa{base, caso.bolsa}).CorrelacionSubida(ctx, caso.persona, caso.clave)
		if err != nil || otra == primera {
			t.Fatal("recuperación compartida entre ámbitos distintos")
		}
	}
}
