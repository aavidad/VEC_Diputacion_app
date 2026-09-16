package postgres

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type ContadorNumeroVisiblePostgreSQL struct{ pool *pgxpool.Pool }

func NuevoContadorNumeroVisiblePostgreSQL(pool *pgxpool.Pool) (*ContadorNumeroVisiblePostgreSQL, error) {
	if pool == nil {
		return nil, ports.ErrPreparacionAltaInvalida
	}
	return &ContadorNumeroVisiblePostgreSQL{pool}, nil
}
func (c *ContadorNumeroVisiblePostgreSQL) SiguienteNumeroVisible(ctx context.Context, anio int) (string, error) {
	if c == nil || c.pool == nil || ctx == nil || anio < 1 || anio > 9999 {
		return "", ports.ErrPreparacionAltaInvalida
	}
	var numero string
	if err := c.pool.QueryRow(ctx, "SELECT vec_contratacion_temporal.siguiente_numero_visible_v1($1)", anio).Scan(&numero); err != nil {
		return "", ports.ErrPreparacionAltaInvalida
	}
	return numero, nil
}

var _ ports.ContadorNumeroVisible = (*ContadorNumeroVisiblePostgreSQL)(nil)
