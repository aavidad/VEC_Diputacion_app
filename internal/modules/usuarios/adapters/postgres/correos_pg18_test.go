package postgres

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/usuarios/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

// Sonda opcional contra un PostgreSQL 18 efímero con Usuarios 000004 ya
// instalado y un LOGIN ejecutor interno. El DSN nunca se imprime.
func TestCorreosPG18PoolNominal(t *testing.T) {
	dsn := os.Getenv("VEC_USUARIOS_CORREOS_PG18_DSN")
	if dsn == "" {
		t.Skip("PG18 efímero no configurado")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal("DSN sintético PG18 inválido")
	}
	defer pool.Close()
	dec := &descifradorCorreoPGPrueba{}
	interno, err := NuevoRegistroCorreosPostgreSQL(ctx, pool, dec, vecdomain.SuperficieAutenticacionInternaCorporativaV1)
	if err != nil || interno == nil {
		t.Fatal("login interno no supera la sonda nominal")
	}
	externo, err := NuevoRegistroCorreosPostgreSQL(ctx, pool, dec, vecdomain.SuperficieAutenticacionExternaPersonalV1)
	if externo != nil || !errors.Is(err, ports.ErrCorreosNoDisponible) {
		t.Fatal("login interno aceptado como externo")
	}
}
