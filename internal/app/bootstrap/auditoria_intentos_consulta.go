package bootstrap

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/config"
	postgresvec "vec-diputacion-granada/internal/vec/adapters/postgres"
	"vec-diputacion-granada/internal/vec/auditoria"
)

// El pool exclusivo solo registra intentos. AD169 coteja el LOGIN, su única
// membresía y el proceso/canal DBA en cada conexión y al escribir el asiento.
func nuevoRegistradorIntentosConsulta(ctx context.Context, cfg config.Config) (*postgresvec.RegistradorIntentosAuditoriaPostgreSQL, auditoria.ConfiguracionIntentosConsulta, func(), error) {
	fallo := func() (*postgresvec.RegistradorIntentosAuditoriaPostgreSQL, auditoria.ConfiguracionIntentosConsulta, func(), error) {
		return nil, auditoria.ConfiguracionIntentosConsulta{}, nil, auditoria.ErrNoDisponible
	}
	if ctx == nil || ctx.Err() != nil {
		return fallo()
	}
	dsn, proceso, canal, plazo, err := cfg.ConexionIntentosAuditoriaConsulta()
	if err != nil {
		return fallo()
	}
	c, err := pgxpool.ParseConfig(dsn)
	if err != nil || c == nil || c.ConnConfig == nil || validarTLSPostgreSQLBorradores(&c.ConnConfig.Config, true) != nil {
		return fallo()
	}
	c.MaxConns, c.MinConns = 2, 0
	c.ConnConfig.ConnectTimeout = plazo
	if c.ConnConfig.RuntimeParams == nil {
		c.ConnConfig.RuntimeParams = make(map[string]string)
	}
	c.ConnConfig.RuntimeParams["application_name"] = proceso
	c.ConnConfig.RuntimeParams["timezone"] = "UTC"
	c.ConnConfig.RuntimeParams["search_path"] = "pg_catalog"
	c.ConnConfig.RuntimeParams["default_transaction_read_only"] = "off"
	c.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		sonda, cancelar := context.WithTimeout(ctx, plazo)
		defer cancelar()
		var admitida bool
		if err := conn.QueryRow(sonda, `SELECT vec_autorizacion_atestada_v3.preflight_registrador_intentos_v1($1::text,$2::text)`, proceso, canal).Scan(&admitida); err != nil || !admitida {
			return auditoria.ErrNoDisponible
		}
		return nil
	}
	pool, err := pgxpool.NewWithConfig(ctx, c)
	if err != nil {
		return fallo()
	}
	registrador, err := postgresvec.NuevoRegistradorIntentosAuditoriaPostgreSQL(pool, proceso, canal, plazo)
	if err != nil || registrador.PreflightIntentoAuditoria(ctx) != nil {
		pool.Close()
		return fallo()
	}
	return registrador, auditoria.ConfiguracionIntentosConsulta{Proceso: proceso, Canal: canal, Plazo: plazo}, pool.Close, nil
}
