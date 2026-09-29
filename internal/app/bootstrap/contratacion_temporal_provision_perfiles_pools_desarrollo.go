package bootstrap

import (
	"context"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
	postgrescontexto "vec-diputacion-granada/internal/vec/adapters/contextoactor/postgres"
	postgresvec "vec-diputacion-granada/internal/vec/adapters/postgres"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const rolProvisionadorPerfilesCT = "vec_autorizacion_migrador"
const rolLectorHistoricoPerfilesCT = "vec_contexto_actor_v1_lector_historico"

func abrirPoolContextoHistoricoProvisionCT(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	if ctx == nil || ctx.Err() != nil || dsn == "" {
		return nil, errProvisionPerfilesCTNoDisponible
	}
	c, err := pgxpool.ParseConfig(dsn)
	if err != nil || c == nil || c.ConnConfig == nil || c.ConnConfig.User == "" ||
		validarTLSPostgreSQLBorradores(&c.ConnConfig.Config, true) != nil {
		return nil, errProvisionPerfilesCTNoDisponible
	}
	c.MaxConns, c.MinConns = 2, 0
	c.ConnConfig.ConnectTimeout = 5 * time.Second
	if c.ConnConfig.RuntimeParams == nil {
		c.ConnConfig.RuntimeParams = make(map[string]string)
	}
	c.ConnConfig.RuntimeParams["application_name"] = "vec-ct-perfiles-contexto-historico"
	c.ConnConfig.RuntimeParams["timezone"] = "UTC"
	c.ConnConfig.RuntimeParams["search_path"] = "pg_catalog,pg_temp"
	c.ConnConfig.RuntimeParams["default_transaction_read_only"] = "on"
	c.ConnConfig.RuntimeParams["statement_timeout"] = "15s"
	login := c.ConnConfig.User
	c.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		conn.TypeMap().RegisterType(&pgtype.Type{Name: "timestamptz", OID: pgtype.TimestamptzOID,
			Codec: &pgtype.TimestamptzCodec{ScanLocation: time.UTC}})
		return acreditarLectorHistoricoProvisionCT(ctx, conn, login)
	}
	pool, err := pgxpool.NewWithConfig(ctx, c)
	if err != nil || acreditarLectorHistoricoProvisionCT(ctx, pool, login) != nil {
		if pool != nil {
			pool.Close()
		}
		return nil, errProvisionPerfilesCTNoDisponible
	}
	return pool, nil
}

func acreditarLectorHistoricoProvisionCT(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, login string) error {
	if ctx == nil || ctx.Err() != nil || q == nil || login == "" {
		return errProvisionPerfilesCTNoDisponible
	}
	var usuario string
	var acreditado bool
	err := q.QueryRow(ctx, `
		SELECT session_user::text,
		       session_user=current_user AND l.rolcanlogin AND l.rolinherit
		       AND NOT l.rolsuper AND NOT l.rolcreatedb AND NOT l.rolcreaterole
		       AND NOT l.rolreplication AND NOT l.rolbypassrls
		       AND NOT g.rolcanlogin AND NOT g.rolinherit
		       AND NOT g.rolsuper AND NOT g.rolcreatedb AND NOT g.rolcreaterole
		       AND NOT g.rolreplication AND NOT g.rolbypassrls
		       AND (SELECT count(*) FROM pg_catalog.pg_auth_members m WHERE m.member=l.oid)=1
		       AND EXISTS (
		         SELECT 1 FROM pg_catalog.pg_auth_members m
		          WHERE m.member=l.oid AND m.roleid=g.oid
		            AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
		       AND pg_catalog.pg_has_role(session_user,g.oid,'USAGE')
		  FROM pg_catalog.pg_roles l
		  JOIN pg_catalog.pg_roles g ON g.rolname=$1
		 WHERE l.rolname=session_user`, rolLectorHistoricoPerfilesCT).Scan(&usuario, &acreditado)
	if err != nil || !acreditado || usuario != login {
		return errProvisionPerfilesCTNoDisponible
	}
	return nil
}

func abrirPoolProvisionadorPerfilesCT(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	if ctx == nil || ctx.Err() != nil || dsn == "" {
		return nil, errProvisionPerfilesCTNoDisponible
	}
	c, err := pgxpool.ParseConfig(dsn)
	if err != nil || c == nil || c.ConnConfig == nil || c.ConnConfig.User == "" ||
		validarTLSPostgreSQLBorradores(&c.ConnConfig.Config, true) != nil {
		return nil, errProvisionPerfilesCTNoDisponible
	}
	c.MaxConns, c.MinConns = 2, 0
	c.ConnConfig.ConnectTimeout = 5 * time.Second
	if c.ConnConfig.RuntimeParams == nil {
		c.ConnConfig.RuntimeParams = make(map[string]string)
	}
	c.ConnConfig.RuntimeParams["application_name"] = "vec-ct-provisionar-perfiles"
	c.ConnConfig.RuntimeParams["timezone"] = "UTC"
	c.ConnConfig.RuntimeParams["search_path"] = "pg_catalog,pg_temp"
	c.ConnConfig.RuntimeParams["default_transaction_isolation"] = "serializable"
	c.ConnConfig.RuntimeParams["statement_timeout"] = "15s"
	c.ConnConfig.RuntimeParams["lock_timeout"] = "3s"
	usuario := c.ConnConfig.User
	c.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		return acreditarProvisionadorPerfilesCT(ctx, conn, usuario)
	}
	pool, err := pgxpool.NewWithConfig(ctx, c)
	if err != nil || acreditarProvisionadorPerfilesCT(ctx, pool, usuario) != nil {
		if pool != nil {
			pool.Close()
		}
		return nil, errProvisionPerfilesCTNoDisponible
	}
	return pool, nil
}

// El LOGIN de provisión no se hereda del servidor. Su única membresía directa
// es el migrador VEC, que puede SET ROLE propietario dentro de la transacción.
func acreditarProvisionadorPerfilesCT(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, login string) error {
	if ctx == nil || ctx.Err() != nil || q == nil || login == "" {
		return errProvisionPerfilesCTNoDisponible
	}
	var usuario string
	var acreditado bool
	err := q.QueryRow(ctx, `
		SELECT session_user::text,
		       session_user=current_user AND l.rolcanlogin AND NOT l.rolinherit
		       AND NOT l.rolsuper AND NOT l.rolcreatedb AND NOT l.rolcreaterole
		       AND NOT l.rolreplication AND NOT l.rolbypassrls
		       AND NOT g.rolcanlogin AND NOT g.rolinherit
		       AND NOT g.rolsuper AND NOT g.rolcreatedb AND NOT g.rolcreaterole
		       AND NOT g.rolreplication AND NOT g.rolbypassrls
		       AND (SELECT count(*) FROM pg_catalog.pg_auth_members m WHERE m.member=l.oid)=1
		       AND EXISTS (
		         SELECT 1 FROM pg_catalog.pg_auth_members m
		          WHERE m.member=l.oid AND m.roleid=g.oid
		            AND NOT m.admin_option AND NOT m.inherit_option AND m.set_option)
		       AND pg_catalog.pg_has_role(session_user,g.oid,'SET')
		       AND pg_catalog.pg_has_role(session_user,'vec_autorizacion_propietario','SET')
		  FROM pg_catalog.pg_roles l
		  JOIN pg_catalog.pg_roles g ON g.rolname=$1
		 WHERE l.rolname=session_user`, rolProvisionadorPerfilesCT).Scan(&usuario, &acreditado)
	if err != nil || !acreditado || usuario != login {
		return errProvisionPerfilesCTNoDisponible
	}
	return nil
}

type conexionesProvisionPerfilesCT struct {
	provisionador     *pgxpool.Pool
	fuente            *pgxpool.Pool
	contextoRuntime   *pgxpool.Pool
	contextoHistorico *pgxpool.Pool
	almacen           puertosvec.FuenteAutorizacion
	lector            puertosvec.LectorContextoOriginalV2
}

func (c *conexionesProvisionPerfilesCT) cerrar() {
	if c == nil {
		return
	}
	if c.contextoHistorico != nil {
		c.contextoHistorico.Close()
	}
	if c.contextoRuntime != nil {
		c.contextoRuntime.Close()
	}
	if c.fuente != nil {
		c.fuente.Close()
	}
	if c.provisionador != nil {
		c.provisionador.Close()
	}
}

func abrirConexionesProvisionPerfilesCT(ctx context.Context, cfg config.Config) (_ conexionesProvisionPerfilesCT, errFinal error) {
	vacias := conexionesProvisionPerfilesCT{}
	dsnProvisionador := os.Getenv(EnvCTPerfilesProvisionadorDatabaseURL)
	dsnFuente := os.Getenv(config.EnvAutorizacionFuenteDatabaseURL)
	dsnContextoRuntime, err := cfg.Normalize().ContratacionTemporalPostgreSQL.DSNContextoActorConsultasSeparado()
	dsnContextoHistorico := os.Getenv(EnvCTPerfilesContextoHistoricoDatabaseURL)
	if err != nil || dsnProvisionador == "" || dsnFuente == "" || dsnContextoHistorico == "" {
		return vacias, errProvisionPerfilesCTNoDisponible
	}
	provisionador, err := abrirPoolProvisionadorPerfilesCT(ctx, dsnProvisionador)
	if err != nil {
		return vacias, errProvisionPerfilesCTNoDisponible
	}
	c := conexionesProvisionPerfilesCT{provisionador: provisionador}
	defer func() {
		if errFinal != nil {
			c.cerrar()
		}
	}()
	c.fuente, err = abrirPoolAutorizacionRRHHDesarrollo(ctx, dsnFuente,
		config.RolAutorizacionFuenteRRHH, "vec-ct-perfiles-fuente")
	if err != nil {
		return vacias, errProvisionPerfilesCTNoDisponible
	}
	c.contextoRuntime, _, err = abrirPoolPostgreSQLContratacionTemporalDesarrollo(ctx,
		dsnContextoRuntime, "vec-ct-perfiles-contexto-runtime", rolContextoActorConsultasDesarrollo)
	if err != nil {
		return vacias, errProvisionPerfilesCTNoDisponible
	}
	c.contextoHistorico, err = abrirPoolContextoHistoricoProvisionCT(ctx, dsnContextoHistorico)
	if err != nil {
		return vacias, errProvisionPerfilesCTNoDisponible
	}
	usuarios := map[string]struct{}{}
	for _, p := range []*pgxpool.Pool{c.provisionador, c.fuente, c.contextoRuntime, c.contextoHistorico} {
		u := p.Config().ConnConfig.User
		if _, existe := usuarios[u]; u == "" || existe {
			return vacias, errProvisionPerfilesCTNoDisponible
		}
		usuarios[u] = struct{}{}
	}
	_, dsnGobierno, err := cfg.Normalize().ContratacionTemporalPostgreSQL.DSNSeparados()
	if err != nil {
		return vacias, errProvisionPerfilesCTNoDisponible
	}
	cfgGobierno, err := pgxpool.ParseConfig(dsnGobierno)
	if err != nil || cfgGobierno == nil || cfgGobierno.ConnConfig == nil ||
		cfgGobierno.ConnConfig.User == c.provisionador.Config().ConnConfig.User {
		return vacias, errProvisionPerfilesCTNoDisponible
	}
	c.almacen, err = postgresvec.NuevoAlmacenAutorizacion(c.fuente)
	if err != nil {
		return vacias, errProvisionPerfilesCTNoDisponible
	}
	c.lector, err = postgrescontexto.NuevoLectorContextoOriginalPostgreSQLV2(
		c.contextoHistorico, relojContratacionTemporalDesarrollo{})
	if err != nil {
		return vacias, errProvisionPerfilesCTNoDisponible
	}
	return c, nil
}
