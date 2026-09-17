package bootstrap

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
	"vec-diputacion-granada/config"
)

const rolEjecutorImportacionConvoca = "vec_bolsa_importacion_convoca_ejecutor"

var ErrPoolImportacionConvocaNoDisponible = errors.New("bootstrap: pool de importacion Convoca no disponible")

func abrirPoolImportacionConvoca(ctx context.Context, cfg config.Config) (*pgxpool.Pool, error) {
	if ctx == nil || !cfg.DevelopmentEnabledByDoubleKey() {
		return nil, ErrPoolImportacionConvocaNoDisponible
	}
	dsn, err := cfg.BolsaImportacionConvocaPostgreSQL.DSN()
	if err != nil {
		return nil, ErrPoolImportacionConvocaNoDisponible
	}
	pc, err := pgxpool.ParseConfig(dsn)
	if err != nil || pc == nil || pc.ConnConfig == nil || validarTLSPostgreSQLBorradores(&pc.ConnConfig.Config, true) != nil {
		return nil, ErrPoolImportacionConvocaNoDisponible
	}
	pc.MaxConns = 2
	pc.MinConns = 0
	pc.ConnConfig.ConnectTimeout = 5 * time.Second
	if pc.ConnConfig.RuntimeParams == nil {
		pc.ConnConfig.RuntimeParams = map[string]string{}
	}
	for k, v := range map[string]string{"application_name": "vec-importar-convoca", "timezone": "UTC", "search_path": "pg_catalog,pg_temp", "default_transaction_isolation": "serializable", "default_transaction_read_only": "off", "statement_timeout": "15s", "lock_timeout": "3s", "idle_in_transaction_session_timeout": "15s"} {
		pc.ConnConfig.RuntimeParams[k] = v
	}
	pc.AfterConnect = func(ctx context.Context, c *pgx.Conn) error { return comprobarPoolImportacionConvoca(ctx, c) }
	pool, err := pgxpool.NewWithConfig(ctx, pc)
	if err != nil {
		return nil, ErrPoolImportacionConvocaNoDisponible
	}
	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, ErrPoolImportacionConvocaNoDisponible
	}
	return pool, nil
}
func comprobarPoolImportacionConvoca(ctx context.Context, c interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}) error {
	var ok bool
	err := c.QueryRow(ctx, `SELECT session_user=current_user AND pg_catalog.pg_has_role(session_user,$1,'MEMBER') FROM pg_catalog.pg_roles WHERE rolname=session_user`, rolEjecutorImportacionConvoca).Scan(&ok)
	if err != nil || !ok {
		return ErrPoolImportacionConvocaNoDisponible
	}
	return nil
}
