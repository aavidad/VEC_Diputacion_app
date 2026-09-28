package postgres

import (
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func TestErrorReincorporacionTitularNoFiltraDetallePostgreSQL(t *testing.T) {
	for _, caso := range []struct {
		entrada  error
		esperado error
	}{
		{&pgconn.PgError{Code: "42501", Detail: "referencia privada"}, dominiovec.ErrAutorizacionDenegada},
		{&pgconn.PgError{Code: "23514", Detail: "referencia privada"}, ports.ErrReincorporacionTitularNoDisponible},
		{errors.New("dsn privado"), ports.ErrReincorporacionTitularNoDisponible},
	} {
		got := errorReincorporacionTitular(caso.entrada)
		if !errors.Is(got, caso.esperado) || strings.Contains(got.Error(), "privad") ||
			strings.Contains(got.Error(), "23514") || strings.Contains(got.Error(), "42501") {
			t.Fatalf("error no minimizado: %v", got)
		}
	}
}
