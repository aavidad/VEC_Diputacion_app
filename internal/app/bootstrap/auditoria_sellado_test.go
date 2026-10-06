package bootstrap

import (
	"context"
	"errors"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	vecpostgres "vec-diputacion-granada/internal/vec/adapters/postgres"
)

func TestMantenerSelladoAuditoriaRepiteLotesLlenosYEsperaConColaVacia(t *testing.T) {
	var llamadas, esperas atomic.Int32
	hecho := make(chan struct{})
	sellar := func(context.Context) (int, int, error) {
		switch llamadas.Add(1) {
		case 1:
			return loteContinuoSelladoAuditoria, 0, nil // grande: sin espera
		case 2:
			return 0, vecpostgres.LoteSelladoAuditoria, nil // lleno: sin espera
		case 3:
			return 0, 0, errors.New("caida") // error: espera
		default:
			select {
			case <-hecho:
			default:
				close(hecho)
			}
			return 3, 0, nil // parcial: espera
		}
	}
	esperar := func(ctx context.Context, _ time.Duration) error {
		esperas.Add(1)
		return ctx.Err()
	}
	detener := mantenerSelladoAuditoria(sellar, time.Millisecond, esperar)
	<-hecho
	detener()
	detener() // idempotente
	if llamadas.Load() < 4 {
		t.Fatalf("llamadas = %d", llamadas.Load())
	}
	// Las dos primeras pasadas grandes no esperan; la de error y la parcial sí.
	if got := esperas.Load(); got < 2 || got > llamadas.Load()-2 {
		t.Fatalf("esperas = %d con %d llamadas", got, llamadas.Load())
	}
}

func TestIniciarSelladoAuditoriaSinConexionNoArrancaNada(t *testing.T) {
	detener, err := iniciarSelladoAuditoria(context.Background(), config.Config{})
	if err != nil || detener == nil {
		t.Fatalf("err = %v", err)
	}
	detener()
}

// TestSelladoAuditoriaLaboratorioPrivado ejecuta el sellador real contra una
// base de laboratorio (AD207 instalada y un LOGIN del grupo encadenador).
func TestSelladoAuditoriaLaboratorioPrivado(t *testing.T) {
	dsn := os.Getenv("VEC_AUDITORIA_SELLADO_LAB_DSN")
	if dsn == "" {
		t.Skip("sin base de laboratorio")
	}
	duracion, err := time.ParseDuration(os.Getenv("VEC_AUDITORIA_SELLADO_LAB_DURACION"))
	if err != nil {
		t.Fatal("duración de laboratorio inválida")
	}
	cfg := config.Config{AuditoriaSelladoPostgreSQL: config.NuevaConfiguracionPostgreSQLAuditoriaSellado(dsn)}
	detener, err := iniciarSelladoAuditoria(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(duracion)
	detener()
}
