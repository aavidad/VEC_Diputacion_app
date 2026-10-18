package postgres

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

func TestLectorBolsaRRHHConjuntoFallaCerradoSinBase(t *testing.T) {
	var l *LectorBolsaRRHHConjunto
	if _, _, _, err := l.LeerBolsa(context.Background(), "bolsa:prueba", time.Now()); !errors.Is(err, ports.ErrResumenBolsasNoDisponible) {
		t.Fatalf("lector nulo: %v", err)
	}
	if _, err := NuevoLectorBolsaRRHHConjunto(nil); !errors.Is(err, ports.ErrResumenBolsasNoDisponible) {
		t.Fatalf("pool nulo: %v", err)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	l = &LectorBolsaRRHHConjunto{pool: &pgxpool.Pool{}}
	if _, _, _, err := l.LeerBolsa(ctx, "bolsa:prueba", time.Now()); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelación perdida: %v", err)
	}
}

type contadorConsultasBolsaRRHHConjunto struct{ orden, situaciones, recuentos atomic.Int64 }

func (c *contadorConsultasBolsaRRHHConjunto) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	switch {
	case strings.Contains(data.SQL, "leer_orden_vigente_bolsa_v1"):
		c.orden.Add(1)
	case strings.Contains(data.SQL, "leer_situaciones_bolsa_rrhh_v1"):
		c.situaciones.Add(1)
	case strings.Contains(data.SQL, "leer_llamamientos_en_curso_bolsas_v1"):
		c.recuentos.Add(1)
	}
	return ctx
}

func (*contadorConsultasBolsaRRHHConjunto) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {
}

// Ensayo opcional sobre el clon PostgreSQL 18 del equipo. Comprueba la ruta
// con el LOGIN ejecutor y sus ACL reales, sin crear ni modificar datos.
func TestLectorBolsaRRHHConjuntoEnPostgreSQL(t *testing.T) {
	dsn := os.Getenv("VEC_BOLSA_LOTES_PG_DSN")
	if dsn == "" {
		t.Skip("sin VEC_BOLSA_LOTES_PG_DSN")
	}
	ctx := context.Background()
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	traza := &contadorConsultasBolsaRRHHConjunto{}
	cfg.ConnConfig.Tracer = traza
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var bolsaRef string
	if err := pool.QueryRow(ctx, `SELECT bolsa_ref FROM vec_bolsa_llamamientos.listar_constituciones_v1() ORDER BY bolsa_ref LIMIT 1`).Scan(&bolsaRef); err != nil {
		t.Fatal(err)
	}
	lector, _ := NuevoLectorBolsaRRHHConjunto(pool)
	orden, filas, _, err := lector.LeerBolsa(ctx, bolsaRef, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(filas) == 0 || len(filas) != len(orden.Posiciones) || traza.orden.Load() != 1 ||
		traza.situaciones.Load() != 1 || traza.recuentos.Load() != 1 {
		t.Fatalf("bolsa=%s filas=%d orden=%d lecturas=%d/%d/%d", bolsaRef, len(filas), len(orden.Posiciones),
			traza.orden.Load(), traza.situaciones.Load(), traza.recuentos.Load())
	}
}
