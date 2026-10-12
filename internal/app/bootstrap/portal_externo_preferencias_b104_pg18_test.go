package bootstrap

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// La fixture usa un LOGIN sintético de una sola membresía en un PostgreSQL 18
// desechable con B104 instalada. No necesita certificado ni datos de persona.
func TestPreflightPreferenciasB104PostgreSQL18(t *testing.T) {
	dsn := os.Getenv("VEC_B104_PG18_DSN")
	if dsn == "" {
		t.Skip("requiere la copia PostgreSQL 18 sintética de B104")
	}
	ctx, cancelar := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancelar()
	inicio := time.Now()
	// El clon usa socket Unix sin TLS; solo se ensaya la ACL. El constructor
	// productivo sigue exigiendo su canal PostgreSQL con TLS acreditado.
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("conexión local del LOGIN sintético: %v", err)
	}
	defer pool.Close()
	var login string
	if err := pool.QueryRow(ctx, `SELECT session_user::text`).Scan(&login); err != nil {
		t.Fatal(err)
	}
	if login != "vec_externo_bolsa_desarrollo" {
		t.Fatalf("LOGIN no coincide con B104: %q", login)
	}
	if !comprobarACLMiBolsaPortalExterno(ctx, pool, true) {
		t.Fatal("ACL B104 de Mi Bolsa e inscripción no coincide con las tres funciones nuevas")
	}
	var directo bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_catalog.pg_proc p
	 JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
	 WHERE n.nspname='vec_autorizacion_atestada_v3'
	  AND p.proname='registrar_intento_lectura_preferencias_bolsas_v1'
	  AND pg_catalog.has_function_privilege(session_user,p.oid,'EXECUTE'))`).Scan(&directo); err != nil || directo {
		t.Fatalf("LOGIN obtuvo función de auditoría común: directo=%v err=%v", directo, err)
	}
	t.Logf("preflight ACL Bolsa exterior B104: %s", time.Since(inicio))
}
