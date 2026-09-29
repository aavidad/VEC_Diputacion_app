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
	// El contador es una fila por año: con altas simultáneas una sentencia
	// puede perder una carrera serializable (40001). Se repite con la política
	// común; el aborto no consume número. Otro fallo es de la base, no de la
	// petición.
	var numero string
	err := ejecutarConReintentoSerializable(ctx, func() error {
		return c.pool.QueryRow(ctx, "SELECT vec_contratacion_temporal.siguiente_numero_visible_v1($1)", anio).Scan(&numero)
	})
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", ports.ErrPersistenciaNoDisponible
	}
	return numero, nil
}

var _ ports.ContadorNumeroVisible = (*ContadorNumeroVisiblePostgreSQL)(nil)
