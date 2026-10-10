package adminperfiles

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	api "vec-diputacion-granada/internal/vec/adapters/httpapi/administracionperfiles"
)

func TestClaseSQLSelectorLlevaSoloElSQLSTATE(t *testing.T) {
	pg := fmt.Errorf("consulta: %w", &pgconn.PgError{Code: "42P01", Message: "tabla privada"})
	if c := claseSQLSelector(pg); c != "selector_sql_42P01" {
		t.Fatalf("clase %q", c)
	}
	if c := claseSQLSelector(errors.New("conexión cerrada")); c != "selector_sql_conexion" {
		t.Fatalf("clase %q", c)
	}
	// errorAutoridad devuelve un centinela: la clase de origen se traslada.
	err := conservarClaseSelector(ConClaseSelector(claseSQLSelector(pg), pg), errorAutoridad(pg))
	if ClaseFalloSelector(err) != "selector_sql_42P01" || !errors.Is(err, api.ErrConfiguracionIncompleta) {
		t.Fatalf("clase o causa perdidas: %q %v", ClaseFalloSelector(err), err)
	}
}
