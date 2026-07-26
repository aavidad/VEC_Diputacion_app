package cobertura_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	rutaFixturePostgreSQLO405 = "/run/vec-o405/fixture.json"
	maximoBytesFixtureO405    = 8 * 1024
)

type fixturePostgreSQLO405 struct {
	OrganizacionRef string   `json:"organizacion_ref"`
	ExpedienteRef   string   `json:"expediente_ref"`
	AmbitosHMAC     []string `json:"ambitos_idempotencia_hmac"`
}

func TestIntegracionPostgreSQLO405LectorNominativo(
	t *testing.T,
) {
	exigirIntegracionPostgreSQLO405(t)
	fixture := cargarFixturePostgreSQLO405(t)
	solicitud := solicitudFixturePostgreSQLO405(
		t,
		fixture.OrganizacionRef,
		fixture.ExpedienteRef,
		fixture.AmbitosHMAC,
	)
	pool := nuevoPoolPostgreSQLO405(t, "vec_o405_lector")
	lector := nuevoLectorPostgreSQLO405(t, pool)

	ctx, cancelar := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelar()
	resultado, err :=
		lector.LeerResultadoHistoricoOperacionDecisionCobertura(ctx, solicitud)
	if err != nil {
		t.Fatalf("lectura confirmada O4-05: %v", err)
	}
	if _, confirmado := resultado.ReciboConfirmadoPara(solicitud); !confirmado {
		t.Fatal("el lector TCB no construyó la rama confirmado")
	}

	solicitudAusente := solicitudFixturePostgreSQLO405(
		t,
		fixture.OrganizacionRef,
		"expediente:o405:ausente",
		fixture.AmbitosHMAC,
	)
	resultado, err =
		lector.LeerResultadoHistoricoOperacionDecisionCobertura(
			ctx,
			solicitudAusente,
		)
	if err != nil {
		t.Fatalf("lectura no observable O4-05: %v", err)
	}
	if !resultado.NoObservablePara(solicitudAusente) {
		t.Fatal("el lector TCB no construyó la rama no_observable")
	}
}

func TestIntegracionPostgreSQLO405LoginSinGrantO405FallaCerrado(
	t *testing.T,
) {
	exigirIntegracionPostgreSQLO405(t)
	fixture := cargarFixturePostgreSQLO405(t)
	solicitud := solicitudFixturePostgreSQLO405(
		t,
		fixture.OrganizacionRef,
		fixture.ExpedienteRef,
		fixture.AmbitosHMAC,
	)
	pool := nuevoPoolPostgreSQLO405(t, "vec_o404e_tcb")
	lector := nuevoLectorPostgreSQLO405(t, pool)

	ctx, cancelar := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelar()
	resultado, err :=
		lector.LeerResultadoHistoricoOperacionDecisionCobertura(ctx, solicitud)
	if !errors.Is(
		err,
		cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoConfiable,
	) {
		t.Fatalf("LOGIN sin grant O4-05 no falló cerrado: %v", err)
	}
	if _, confirmado := resultado.ReciboConfirmadoPara(solicitud); confirmado ||
		resultado.NoObservablePara(solicitud) {
		t.Fatal("LOGIN sin grant O4-05 elevó una rama terminal")
	}
}

func exigirIntegracionPostgreSQLO405(t *testing.T) {
	t.Helper()
	if os.Getenv("VEC_O405_INTEGRACION_POSTGRES") != "1" {
		t.Skip("integración PostgreSQL O4-05 no habilitada")
	}
}

func cargarFixturePostgreSQLO405(t *testing.T) fixturePostgreSQLO405 {
	t.Helper()
	archivo, err := os.Open(rutaFixturePostgreSQLO405)
	if err != nil {
		t.Fatal("fixture PostgreSQL O4-05 no disponible")
	}
	defer archivo.Close()
	contenido, err := io.ReadAll(io.LimitReader(
		archivo,
		maximoBytesFixtureO405+1,
	))
	if err != nil || len(contenido) == 0 ||
		len(contenido) > maximoBytesFixtureO405 {
		t.Fatal("fixture PostgreSQL O4-05 fuera de límite")
	}
	var fixture fixturePostgreSQLO405
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(&fixture); err != nil {
		t.Fatal("fixture PostgreSQL O4-05 inválido")
	}
	if err := decodificador.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		t.Fatal("fixture PostgreSQL O4-05 contiene datos adicionales")
	}
	return fixture
}

func solicitudFixturePostgreSQLO405(
	t *testing.T,
	organizacionRef string,
	expedienteRef string,
	ambitos []string,
) cobertura.SolicitudRecuperacionResultadoOperacionDecisionCobertura {
	t.Helper()
	if len(ambitos) == 0 {
		t.Fatal("fixture PostgreSQL O4-05 sin ámbitos")
	}
	coleccion, err := ports.NuevaColeccionSellosHMAC(
		ambitos[0],
		ambitos[1:],
	)
	if err != nil {
		t.Fatal("fixture PostgreSQL O4-05 con ámbitos inválidos")
	}
	solicitud, err :=
		cobertura.NuevaSolicitudRecuperacionResultadoOperacionDecisionCoberturaIntegracionPrueba(
			organizacionRef,
			expedienteRef,
			coleccion,
		)
	if err != nil {
		t.Fatal("fixture PostgreSQL O4-05 no forma una solicitud nominal")
	}
	return solicitud
}

func nuevoPoolPostgreSQLO405(
	t *testing.T,
	usuarioEsperado string,
) *pgxpool.Pool {
	t.Helper()
	ctx, cancelar := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelar()
	pool, err := pgxpool.New(ctx, "")
	if err != nil {
		t.Fatal("configuración PostgreSQL O4-05 inválida")
	}
	t.Cleanup(pool.Close)
	var usuario string
	if err := pool.QueryRow(ctx, "SELECT session_user").Scan(&usuario); err != nil {
		t.Fatal("LOGIN PostgreSQL O4-05 no disponible")
	}
	if usuario != usuarioEsperado {
		t.Fatalf("LOGIN PostgreSQL inesperado: %q", usuario)
	}
	return pool
}

func nuevoLectorPostgreSQLO405(
	t *testing.T,
	pool *pgxpool.Pool,
) cobertura.LectorResultadoHistoricoOperacionDecisionCobertura {
	t.Helper()
	ejecutor, err :=
		postgres.NuevoEjecutorLecturaResultadoHistoricoOperacionDecisionCoberturaPostgreSQL(
			pool,
		)
	if err != nil {
		t.Fatal(err)
	}
	lector, err :=
		cobertura.NuevoLectorResultadoHistoricoOperacionDecisionCoberturaTCB(
			ejecutor,
		)
	if err != nil {
		t.Fatal(err)
	}
	return lector
}
