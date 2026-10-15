package bootstrap

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Esta prueba se ejecuta con el login CT de una copia privada que ya tiene
// CT220, B120/B121 y AD278 instalados. No publica ni consume material V3.
func TestPreflightSeleccionOfertaCTSinUsageBolsa(t *testing.T) {
	dsn := os.Getenv("VEC_TEST_C2_PREFLIGHT_DSN")
	if dsn == "" {
		t.Skip("requiere login CT de clon local preparado")
	}
	ctx, cancelar := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancelar()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal("pool de prueba no disponible")
	}
	defer pool.Close()
	var usuario string
	var usoBolsa bool
	if err := pool.QueryRow(ctx, `SELECT session_user,
		pg_catalog.has_schema_privilege(session_user,'vec_bolsa_llamamientos','USAGE')`).Scan(&usuario, &usoBolsa); err != nil {
		t.Fatal("no se pudo verificar el login de la prueba")
	}
	if usuario != "vec_ct_o207_runtime" || usoBolsa {
		t.Fatal("la prueba requiere el login CT sin USAGE de Bolsa")
	}
	if !detectarSeleccionOfertaCTDesarrollo(ctx, pool) {
		t.Fatal("las once firmas y sus ACL publicadas no fueron reconocidas")
	}
	const original = "verificar_cierre_intencion_oferta_ct_v1"
	consultaIncompleta := strings.Replace(consultaPreflightSeleccionOfertaCTDesarrollo,
		original, original+"_ausente", 1)
	if consultaIncompleta == consultaPreflightSeleccionOfertaCTDesarrollo {
		t.Fatal("no se alteró la función elegida para la prueba negativa")
	}
	var disponible bool
	if err := pool.QueryRow(ctx, consultaIncompleta).Scan(&disponible); err != nil || disponible {
		t.Fatal("la falta de una función no cerró la capacidad")
	}
}
