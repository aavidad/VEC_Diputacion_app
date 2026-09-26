package bootstrap

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// La causa se clasifica sin copiar el texto del error, que puede llevar DSN,
// roles o contenido de filas; el centinela de siempre se conserva.
func TestFalloPostgreSQLCTDesarrolloClasificaSinDatos(t *testing.T) {
	secreto := "postgresql://usuario:clave@servidor/base"
	casos := map[string]error{
		"comprobacion_rechazada":    nil,
		"sqlstate_42501":            &pgconn.PgError{Code: "42501", Message: secreto},
		"sin_filas":                 pgx.ErrNoRows,
		"tiempo_agotado":            context.DeadlineExceeded,
		"gobierno_actual_ajeno":     errGobiernoPostgreSQLContratacionTemporalDesarrolloAjeno,
		"dependencia_no_disponible": errPostgreSQLContratacionTemporalDesarrolloNoDisponible,
		"error_*errors.errorString": errors.New(secreto),
	}
	for esperada, causa := range casos {
		if obtenida := causaFalloPostgreSQLCTDesarrollo(causa); obtenida != esperada || strings.Contains(obtenida, "clave") {
			t.Fatalf("causa %q; se esperaba %q", obtenida, esperada)
		}
	}
	if err := falloPostgreSQLCTDesarrollo(errors.New(secreto)); !errors.Is(err, errPostgreSQLContratacionTemporalDesarrolloNoDisponible) {
		t.Fatalf("el centinela cambió: %v", err)
	}
	if etapa := etapaLlamadorPostgreSQLCTDesarrollo(1); !strings.HasPrefix(etapa, "TestFalloPostgreSQLCTDesarrolloClasificaSinDatos:") {
		t.Fatalf("etapa %q", etapa)
	}
}
