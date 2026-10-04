package postgres

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"vec-diputacion-granada/internal/modules/bolsa/domain"
)

func TestCASRevisionObsoletaEsConflicto(t *testing.T) {
	err := errorSituacionParticipacion(&pgconn.PgError{Code: "VBS02"})
	if !errors.Is(err, domain.ErrCambioSituacionParticipacionInvalido) {
		t.Fatalf("CAS obsoleto: %v", err)
	}
}
