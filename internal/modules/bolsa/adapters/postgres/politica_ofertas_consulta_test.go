package postgres

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

func TestPoliticaOfertasExigePoolsDistintos(t *testing.T) {
	ejecutor, calculador := new(pgxpool.Pool), new(pgxpool.Pool)
	for nombre, pools := range map[string][2]*pgxpool.Pool{
		"sin ejecutor":   {nil, calculador},
		"sin calculador": {ejecutor, nil},
		"mismo pool":     {ejecutor, ejecutor},
	} {
		if _, err := NuevoRepositorioPoliticaOfertasPostgreSQL(pools[0], pools[1]); !errors.Is(err, ports.ErrPoliticaOfertasNoDisponible) {
			t.Errorf("%s: %v", nombre, err)
		}
	}
	if r, err := NuevoRepositorioPoliticaOfertasPostgreSQL(ejecutor, calculador); err != nil || r.pool != ejecutor || r.calculo != calculador {
		t.Fatalf("dos pools separados rechazados: %+v %v", r, err)
	}
}
