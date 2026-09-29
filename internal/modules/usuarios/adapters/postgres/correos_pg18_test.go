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

// La migración de Usuarios y AD3 se instalan antes de esta sonda en la base
// PG18 efímera. El DSN apunta al socket Unix sintético; nunca se imprime.
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
	if pool.Ping(ctx) != nil {
		t.Fatal("socket sintético PG18 no disponible")
	}
	dec := &descifradorCorreoPGPrueba{}
	interno, err := NuevoRegistroCorreosPostgreSQL(ctx, pool, dec, vecdomain.SuperficieAutenticacionInternaCorporativaV1)
	if err != nil || interno == nil {
		t.Fatal("login interno no supera preflight nominal")
	}
	externo, err := NuevoRegistroCorreosPostgreSQL(ctx, pool, dec, vecdomain.SuperficieAutenticacionExternaPersonalV1)
	if externo != nil || !errors.Is(err, ports.ErrCorreosNoDisponible) {
		t.Fatal("login interno fue aceptado como externo")
	}
	lector, err := NuevoLectorCorreoActivoPostgreSQL(ctx, pool, dec)
	if lector != nil || !errors.Is(err, ports.ErrCorreosNoDisponible) {
		t.Fatal("lector técnico publicado antes de AD3 nominal")
	}
}
