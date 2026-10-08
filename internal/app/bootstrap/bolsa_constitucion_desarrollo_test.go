package bootstrap

import (
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/config"
)

func TestPoolConstitucionBolsaNoReutilizaLoginWeb(t *testing.T) {
	t.Setenv(config.EnvBolsaConstitucionDatabaseURL, "")
	t.Setenv(config.EnvBolsaLlamamientosDatabaseURL, "postgres://web:secreto@localhost/vec?sslmode=require")
	pool, err := abrirPoolConstitucionBolsaDesarrollo(context.Background(), config.Load())
	if pool != nil || !errors.Is(err, ErrConstitucionBolsaNoDisponible) {
		t.Fatalf("pool CLI ausente: pool=%v error=%v", pool, err)
	}
	t.Setenv(config.EnvBolsaConstitucionDatabaseURL, "postgres://web:otro-secreto@localhost/vec?sslmode=require")
	pool, err = abrirPoolConstitucionBolsaDesarrollo(context.Background(), config.Load())
	if pool != nil || !errors.Is(err, ErrConstitucionBolsaNoDisponible) {
		t.Fatalf("LOGIN web repetido: pool=%v error=%v", pool, err)
	}
}
