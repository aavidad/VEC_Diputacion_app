package postgres

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestConflictoSerializableSoloRepiteChoquesDeTransaccion(t *testing.T) {
	for codigo, repetir := range map[string]bool{"40001": true, "40P01": true, "23505": true, "55P03": true, "42501": false, "VBC01": false, "22023": false} {
		err := fmt.Errorf("envuelto: %w", &pgconn.PgError{Code: codigo})
		if conflictoSerializable(err) != repetir {
			t.Errorf("%s: repetir=%v", codigo, !repetir)
		}
	}
	if conflictoSerializable(errors.New("otro")) {
		t.Error("error sin código repetido")
	}
}
