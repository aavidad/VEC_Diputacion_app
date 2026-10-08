package bootstrap

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"vec-diputacion-granada/config"
)

type filaIdentidadConError struct{}

func (filaIdentidadConError) Scan(...any) error {
	return errors.New("SQL falló con password sintético")
}

type consultaIdentidadConError struct{}

func (consultaIdentidadConError) QueryRow(context.Context, string, ...any) pgx.Row {
	return filaIdentidadConError{}
}

func TestPreflightConstitucionConservaCausaSinMensajeSQL(t *testing.T) {
	err := comprobarIdentidadConstitucionBolsaDesarrollo(context.Background(), consultaIdentidadConError{})
	var causa CausaConstitucionBolsa
	if !errors.Is(err, ErrConstitucionBolsaNoDisponible) || !errors.As(err, &causa) || causa.Codigo != "consulta_identidad_fallida" || strings.Contains(err.Error(), "password") {
		t.Fatalf("fallo de identidad no minimizado: %v", err)
	}
}

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
