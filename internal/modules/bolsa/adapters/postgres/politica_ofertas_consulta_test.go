package postgres

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

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
		if _, err := NuevoRepositorioPoliticaOfertasPostgreSQL(t.Context(), pools[0], pools[1]); !errors.Is(err, ports.ErrPoliticaOfertasNoDisponible) {
			t.Errorf("%s: %v", nombre, err)
		}
	}
	if _, err := NuevoRepositorioPoliticaOfertasPostgreSQL(nil, ejecutor, calculador); !errors.Is(err, ports.ErrPoliticaOfertasNoDisponible) {
		t.Fatalf("constructor sin contexto: %v", err)
	}
}

// El script PG18 efímero aporta tres LOGIN sintéticos por socket Unix. La
// prueba compara sesiones reales, no solo punteros a pools diferentes.
func TestAcreditarPoolsPoliticaPG18(t *testing.T) {
	ejecutorDSN, calculadorDSN, dualDSN := os.Getenv("VEC_B51_TEST_EJECUTOR_DSN"), os.Getenv("VEC_B51_TEST_CALCULADOR_DSN"), os.Getenv("VEC_B51_TEST_DUAL_DSN")
	fase := os.Getenv("VEC_B51_TEST_FASE")
	if ejecutorDSN == "" || (fase != "exclusivo" && fase != "dual") {
		t.Skip("requiere PG18 efímero")
	}
	ctx, cancelar := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancelar()
	abrir := func(dsn string) *pgxpool.Pool {
		p, err := pgxpool.New(ctx, dsn)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(p.Close)
		return p
	}
	ejecutor := abrir(ejecutorDSN)
	if fase == "exclusivo" {
		if calculadorDSN == "" {
			t.Fatal("falta calculador")
		}
		calculador := abrir(calculadorDSN)
		if _, err := NuevoRepositorioPoliticaOfertasPostgreSQL(ctx, ejecutor, calculador); err != nil {
			t.Fatalf("LOGIN nominales exclusivos rechazados: %v", err)
		}
		otroEjecutor := abrir(ejecutorDSN)
		if _, err := NuevoRepositorioPoliticaOfertasPostgreSQL(ctx, ejecutor, otroEjecutor); !errors.Is(err, ports.ErrPoliticaOfertasNoDisponible) {
			t.Fatalf("dos pools con mismo LOGIN aceptados: %v", err)
		}
	} else {
		if dualDSN == "" {
			t.Fatal("falta LOGIN dual")
		}
		dual := abrir(dualDSN)
		if _, err := NuevoRepositorioPoliticaOfertasPostgreSQL(ctx, ejecutor, dual); !errors.Is(err, ports.ErrPoliticaOfertasNoDisponible) {
			t.Fatalf("LOGIN dual aceptado: %v", err)
		}
	}
}
