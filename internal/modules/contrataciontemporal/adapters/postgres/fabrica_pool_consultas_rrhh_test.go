package postgres

import (
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestFabricaPoolConsultasRRHHLigaLoginNominalExclusivo(t *testing.T) {
	t.Parallel()
	const login = "vec_ct_rrhh_prueba_01"
	cadena := "postgres:///" +
		"?host=/tmp/vec-ct46-socket-inexistente" +
		"&port=5432&user=" + login + "&sslmode=disable"

	pool, err := nuevoPoolConsultasRRHHPostgreSQL(
		context.Background(),
		cadena,
		login,
		modoTLSAcreditacionPoolO405SocketUnixPrueba,
	)
	if err != nil {
		t.Fatalf("crear pool nominal de prueba: %v", err)
	}
	defer pool.Cerrar()
	if pool.iniciador == nil || pool.iniciador.loginNominal != login ||
		pool.pool.Config().ConnConfig.User != login {
		t.Fatal("el pool no quedó ligado al LOGIN nominal")
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
