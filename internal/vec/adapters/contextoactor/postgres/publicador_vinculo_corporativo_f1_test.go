package postgres

import (
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestErrorPublicacionF1MinimizaLaCausaPostgreSQL(t *testing.T) {
	const privado = "contenido reservado de la prueba"
	err := errorPublicacionVinculoF1(&pgconn.PgError{Code: "42501", Routine: "exec_stmt_raise", Message: privado})
	var causa *ErrorPublicacionVinculoF1
	if !errors.As(err, &causa) || causa.SQLState != "42501" || causa.Routine != "exec_stmt_raise" ||
		strings.Contains(err.Error(), privado) || causa.Clave != "" {
		t.Fatalf("causa sin minimizar: %v", err)
	}
	err = errorPublicacionVinculoF1(&pgconn.PgError{Code: "42501", Routine: "exec_stmt_raise",
		Message: "CA42: PARO clave=fuente actual=no_acreditada esperado=exacta"})
	if !errors.As(err, &causa) || causa.Clave != "fuente" || causa.Observado != "no_acreditada" || causa.Esperado != "exacta" {
		t.Fatalf("guarda sin clave y valores: %v", err)
	}
}
