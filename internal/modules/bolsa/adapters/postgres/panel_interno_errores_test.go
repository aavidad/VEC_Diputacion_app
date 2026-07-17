package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

func TestErroresPanelInternoPostgreSQLSeClasificanSinFiltrarDetalle(t *testing.T) {
	casos := []struct {
		nombre   string
		codigo   string
		esperado error
	}{
		{"serializacion", "40001", ErrConsultaPanelInternoPostgreSQLEnCurso},
		{"interbloqueo", "40P01", ErrConsultaPanelInternoPostgreSQLEnCurso},
		{"respuesta incoherente", "23514", puertosbolsa.ErrResultadoPanelInternoInvalido},
		{"infraestructura", "08006", ErrFuentePanelInternoPostgreSQLNoDisponible},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			err := errorPostgreSQLPanelInterno(
				context.Background(),
				&pgconn.PgError{Code: caso.codigo, Message: "detalle interno no publicable"},
			)
			if !errors.Is(err, caso.esperado) || stringsContieneDetalleInterno(err.Error()) {
				t.Fatalf("clasificacion inesperada: %v", err)
			}
		})
	}
}

func TestErrorPanelInternoPostgreSQLPriorizaCancelacion(t *testing.T) {
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	err := errorPostgreSQLPanelInterno(ctx, &pgconn.PgError{Code: "08006"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelacion ocultada: %v", err)
	}
}

func stringsContieneDetalleInterno(texto string) bool {
	return strings.Contains(texto, "detalle interno no publicable")
}
