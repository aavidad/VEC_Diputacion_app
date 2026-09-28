package postgres

import (
	"errors"
	"github.com/jackc/pgx/v5/pgconn"
	"testing"
	puertos "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func TestErrorCatalogoPG(t *testing.T) {
	for _, codigo := range []string{"VBS01", "VBS57", "23505"} {
		if !errors.Is(errorCatalogoPG(&pgconn.PgError{Code: codigo}), puertos.ErrCatalogoCausasParticipacionEnConflicto) {
			t.Fatalf("%s no es conflicto", codigo)
		}
	}
	if !errors.Is(errorCatalogoPG(&pgconn.PgError{Code: "42501"}), dominiovec.ErrAutorizacionDenegada) {
		t.Fatal("denegacion perdida")
	}
}
