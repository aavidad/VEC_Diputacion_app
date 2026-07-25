package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestNormalizarInstantePostgreSQLProduceUTCCanonico(t *testing.T) {
	t.Parallel()
	zonaLocalUTC := time.FixedZone("zona-local-utc-prueba", 0)
	entrada := time.Date(
		2026, time.July, 25, 0, 30, 7, 123456789, zonaLocalUTC,
	)
	resultado := normalizarInstantePostgreSQL(entrada)
	if !domain.InstanteUTCCanonico(resultado) ||
		resultado.Location() != time.UTC ||
		resultado.Nanosecond() != 123456000 {
		t.Fatalf("instante PostgreSQL no canónico: %#v", resultado)
	}
}

func TestTransaccionAnalisisPostgreSQLFallaCerradaSinPool(t *testing.T) {
	t.Parallel()
	adaptador, err := nuevaTransaccionOperacionesAnalisisPostgreSQL(nil)
	if adaptador != nil ||
		!errors.Is(
			err,
			ports.ErrPersistenciaOperacionAnalisisNoDisponible,
		) {
		t.Fatalf("constructor no falló cerrado: adaptador=%v err=%v", adaptador, err)
	}
}

func TestTransaccionAnalisisRechazaOrdenVaciaAntesDePostgreSQL(t *testing.T) {
	t.Parallel()
	iniciador := &iniciadorPreparacionPrueba{}
	adaptador, err := nuevaTransaccionOperacionesAnalisisPostgreSQL(iniciador)
	if err != nil {
		t.Fatal(err)
	}
	recibo, err := adaptador.ConfirmarOperacionAnalisis(
		context.Background(),
		ports.OrdenConfirmarOperacionAnalisis{},
	)
	if recibo != (ports.ReciboOperacionAnalisis{}) ||
		!errors.Is(err, ports.ErrOrdenOperacionAnalisisInvalida) ||
		iniciador.inicios != 0 {
		t.Fatalf(
			"orden vacía cruzó la frontera: recibo=%#v inicios=%d err=%v",
			recibo,
			iniciador.inicios,
			err,
		)
	}
}

func TestErrorConfirmacionAnalisisClasificaConflictoUnico(t *testing.T) {
	t.Parallel()
	err := errorConfirmacionAnalisis(&pgconn.PgError{Code: "23505"})
	if !errors.Is(err, ports.ErrConjuntoFuentesAnalisisYaConsumido) {
		t.Fatalf("23505 no se clasificó como conflicto: %v", err)
	}
	err = errorConfirmacionAnalisis(&pgconn.PgError{Code: "42501"})
	if !errors.Is(
		err,
		ports.ErrPersistenciaOperacionAnalisisNoDisponible,
	) {
		t.Fatalf("fallo de autoridad se expuso: %v", err)
	}
}
