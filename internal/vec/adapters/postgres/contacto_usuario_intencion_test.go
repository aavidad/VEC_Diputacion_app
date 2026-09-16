package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	vecapp "vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/ports"
)

type iniciadorContactoRecuperablePrueba struct{ llamadas int }

func (i *iniciadorContactoRecuperablePrueba) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	i.llamadas++
	return nil, errors.New("no debe abrir")
}

func TestContactoRecuperableErrorMantieneConflictosCerrados(t *testing.T) {
	if err := contactoRecuperableError(context.Background(), &pgconn.PgError{Code: "P1103"}); !errors.Is(err, vecapp.ErrVersionContactoDivergente) {
		t.Fatal("P1103 debe conservar el conflicto CAS de aplicación")
	}
	if err := contactoRecuperableError(context.Background(), &pgconn.PgError{Code: "P1104"}); !errors.Is(err, vecapp.ErrIntencionContactoDivergente) {
		t.Fatal("P1104 debe conservar la intención ajena de aplicación")
	}
}

func TestRegistroContactoRecuperablePostgreSQLRechazaDependenciaAusente(t *testing.T) {
	if r, err := nuevoRegistroContactoRecuperablePostgreSQL(nil); err == nil || r != nil {
		t.Fatal("el escritor recuperable sin pool debe denegarse")
	}
}

func TestRegistroContactoRecuperableInvalidoNoAbreTransaccion(t *testing.T) {
	iniciador := &iniciadorContactoRecuperablePrueba{}
	r, err := nuevoRegistroContactoRecuperablePostgreSQL(iniciador)
	if err != nil {
		t.Fatal(err)
	}
	_, err = r.GuardarContactoRecuperable(context.Background(), ports.OrdenRegistroContactoRecuperable{})
	if !errors.Is(err, vecapp.ErrContactoUsuarioNoDisponible) || iniciador.llamadas != 0 {
		t.Fatal("una orden incompleta no puede abrir transacción ni alcanzar SQL")
	}
}
