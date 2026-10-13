package main

import (
	"errors"
	"testing"

	contextopg "vec-diputacion-granada/internal/vec/adapters/contextoactor/postgres"
)

func TestPublicadorF1RechazaModoAmbiguoAntesDeConectar(t *testing.T) {
	t.Setenv("VEC_VINCULO_CORPORATIVO_DSN", "host=/ruta-inexistente dbname=postgres")
	for _, args := range [][]string{nil, {"--comprobar", "--aplicar"}, {"--comprobar", "extra"}} {
		var causa *contextopg.ErrorPublicacionVinculoF1
		if err := ejecutar(args); !errors.As(err, &causa) || causa.SQLState != "22023" || causa.Routine != "configuracion" {
			t.Fatalf("modo inválido no se rechazó antes de conectar: %v", err)
		}
	}
}
