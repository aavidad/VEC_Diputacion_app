package postgres

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
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

func TestErrorContactoSeparaClaveDivergenteDeContactoInvalido(t *testing.T) {
	if err := errorContactoParticipacion(&pgconn.PgError{Code: "VBC01"}); !errors.Is(err, dominiobolsa.ErrContactoClaveDivergente) {
		t.Fatalf("VBC01: %v", err)
	}
	for _, codigo := range []string{"22023", "23514"} {
		err := errorContactoParticipacion(&pgconn.PgError{Code: codigo})
		if !errors.Is(err, dominiobolsa.ErrContactoParticipacionInvalido) || errors.Is(err, dominiobolsa.ErrContactoClaveDivergente) {
			t.Fatalf("%s: %v", codigo, err)
		}
	}
}
