package composicion

import (
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/personal/domain"
)

func TestMontajeLectorCRN11CierraSinAutoridades(t *testing.T) {
	for _, d := range []DependenciasLectorVinculoCRN11{{}, {Ahora: time.Now}, {Personal: &pgxpool.Pool{}, Ahora: time.Now}} {
		lector, err := ComponerLectorVinculoPropioCRN11(d)
		if lector != nil || !errors.Is(err, domain.ErrVinculoCRN11NoDisponible) {
			t.Fatal("montaje sin autoridades admitido", err)
		}
	}
}
