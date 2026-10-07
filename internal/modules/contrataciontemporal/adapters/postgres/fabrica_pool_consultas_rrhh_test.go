package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestFabricaPoolConsultasRRHHLigaLoginNominalExclusivo(t *testing.T) {
	t.Parallel()
	const login = "vec_ct_rrhh_prueba_01"
	cadena := "postgres:///" +
		"?host=/tmp/vec-ct46-socket-inexistente" +
		"&port=5432&user=" + login + "&sslmode=disable"

	configuracion, err := pgxpool.ParseConfig(cadena)
	if err != nil || configuracion.ConnConfig.User != login || !loginNominalConsultaRRHHValido(login) {
		t.Fatalf("configuración no ligada al LOGIN nominal: %v", err)
	}
	pool, err := nuevoPoolConsultasRRHHPostgreSQL(
		context.Background(),
		cadena,
		login,
		modoTLSAcreditacionPoolO405SocketUnixPrueba,
	)
	if pool != nil || !errors.Is(err, ports.ErrConsultaRRHHNoDisponible) {
		t.Fatalf("pool nominal sin conexión aceptado: pool=%v err=%v", pool, err)
	}

	if _, err := nuevoPoolConsultasRRHHPostgreSQL(
		context.Background(),
		cadena,
		"vec_ct_rrhh_otro",
		modoTLSAcreditacionPoolO405SocketUnixPrueba,
	); !errors.Is(err, ports.ErrConsultaRRHHNoDisponible) {
		t.Fatalf("se aceptó LOGIN distinto del DSN: %v", err)
	}
	if _, err := nuevoPoolConsultasRRHHPostgreSQL(
		context.Background(),
		"postgres:///?host=/tmp/vec-ct46-socket-inexistente"+
			"&port=5432&user="+rolConsultorRRHHPostgreSQL+
			"&sslmode=disable",
		rolConsultorRRHHPostgreSQL,
		modoTLSAcreditacionPoolO405SocketUnixPrueba,
	); !errors.Is(err, ports.ErrConsultaRRHHNoDisponible) {
		t.Fatalf("se aceptó el grupo NOLOGIN como identidad: %v", err)
	}
	if _, err := NuevoPoolConsultasRRHHPostgreSQL(
		context.Background(),
		cadena,
		login,
	); !errors.Is(err, ports.ErrConsultaRRHHNoDisponible) {
		t.Fatalf("producción aceptó transporte sin TLS: %v", err)
	}
}
