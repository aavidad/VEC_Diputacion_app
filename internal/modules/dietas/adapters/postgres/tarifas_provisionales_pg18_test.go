package postgres

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/dietas/domain"
)

// Ensayo de solo lectura sobre un PostgreSQL 18 desechable. La variable de
// entorno se proporciona fuera de Git; el test no crea ni modifica tarifas.
func TestConsultarTarifaPG18LeeFuentesYDeniegaVersionOVigenciaAjena(t *testing.T) {
	dsn := os.Getenv("VEC_DIETAS_TARIFAS_PG18_DSN")
	if dsn == "" {
		t.Skip("ensayo PostgreSQL 18 no configurado")
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal("configuración PostgreSQL 18 inválida")
	}
	cfg.MaxConns = 1
	cfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		_, err := conn.Exec(ctx, "SET ROLE vec_dietas_ejecutor")
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal("no se pudo abrir el pool PostgreSQL 18")
	}
	defer pool.Close()
	repo, err := NuevoRepositorioTarifasProvisionales(pool)
	if err != nil {
		t.Fatal(err)
	}
	version := "provisional:rd462:20260923"
	fecha := time.Date(2026, 9, 23, 0, 0, 0, 0, time.UTC)
	leida, err := repo.Consultar(ctx, version, 2, "automovil", fecha)
	if err != nil || !leida.ReferenciasNormativasValidas() || leida.Dieta.Rotulo != domain.RotuloTarifaProvisional ||
		leida.Dieta.ManutencionCentimos != 3740 || leida.Dieta.AlojamientoTopeCentimos != 6597 || leida.EURPorKM != "0.2600" {
		t.Fatalf("tarifa original no disponible o modificada: %+v error=%v", leida, err)
	}
	for _, caso := range []struct {
		version string
		fecha   time.Time
	}{
		{"provisional:rd462:20260924", fecha},
		{version, fecha.AddDate(0, 0, -1)},
	} {
		if _, err := repo.Consultar(ctx, caso.version, 2, "automovil", caso.fecha); !errors.Is(err, ErrTarifaProvisionalNoDisponible) {
			t.Fatalf("versión o vigencia ajena aceptada: %v", err)
		}
	}
}
