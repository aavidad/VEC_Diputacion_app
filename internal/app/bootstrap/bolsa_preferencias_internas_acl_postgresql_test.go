package bootstrap

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Ejecutar con el LOGIN nominal de una copia local que tenga AD279 y B116.
func TestPreferenciasBolsasInternasMetadatosSinAccesoAAutorizacion(t *testing.T) {
	dsn := os.Getenv("VEC_TEST_BOLSA_PREFERENCIAS_INTERNA_DSN")
	if dsn == "" {
		t.Skip("requiere la conexión nominal de una copia local")
	}
	ctx, cancelar := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancelar()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("no se pudo preparar la conexión de ensayo: %T", err)
	}
	defer pool.Close()
	var autorizacion, atestada bool
	if err := pool.QueryRow(ctx, `SELECT
 has_schema_privilege(session_user,'vec_autorizacion','USAGE'),
 has_schema_privilege(session_user,'vec_autorizacion_atestada_v3','USAGE')`).Scan(&autorizacion, &atestada); err != nil {
		t.Fatalf("no se pudieron comprobar los permisos de ensayo: %T", err)
	}
	if autorizacion || atestada {
		t.Fatal("el ensayo exige la separación de esquemas del LOGIN de Bolsa")
	}
	if err := comprobarACLPreferenciasBolsasInternas(ctx, pool); err != nil {
		t.Fatalf("el LOGIN nominal no pudo comprobar los metadatos: %v", err)
	}
}
