package bootstrap

import (
	"context"
	"os"
	"testing"
	"time"

	"vec-diputacion-granada/config"
)

// Requiere PostgreSQL sintético y aislado. El DSN base apunta a loopback;
// cada pool sigue autenticándose con un login nominal distinto.
func TestPoolsPostgreSQLDietasDesarrolloPGReal(t *testing.T) {
	base := os.Getenv("VEC_DIETAS_TEST_PG_URL")
	if base == "" {
		t.Skip("PostgreSQL aislado no configurado")
	}
	dietas, personal := base+"&user=login_dietas_prueba", base+"&user=login_personal_prueba"
	c, err := config.NuevaConfiguracionDietasBorradores(dietas, personal)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	pools, err := nuevosPoolsPostgreSQLDietasDesarrollo(ctx, config.Config{DietasBorradoresPostgreSQL: c})
	if err != nil {
		t.Fatal(err)
	}
	defer pools.Cerrar()
	if pools.Dietas() == nil || pools.Personal() == nil {
		t.Fatal("faltan pools nominales")
	}
}
