package bootstrap

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/vec/auditoria"
)

const rolConsultaAuditoriaCTDesarrollo = "vec_contratacion_temporal_consultor_rrhh"

const funcionConsultaAuditoriaCTDesarrollo = "vec_contratacion_temporal.consultar_auditoria_ct_atestada_v1(text,text,text,timestamptz,timestamptz,integer,timestamptz,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)"
const funcionConsultaAuditoriaBolsaDesarrollo = "vec_bolsa_llamamientos.consultar_auditoria_participacion_v1(text,text,timestamptz,timestamptz,timestamptz,text,text,integer,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)"

// La consulta CT usa el LOGIN consultor nominal ya configurado. Abre un pool
// exclusivo para la superficie Auditoría y coteja su membresía en cada conexión.
func abrirPoolConsultaAuditoriaCTDesarrollo(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	if ctx == nil || ctx.Err() != nil || dsn == "" {
		return nil, auditoria.ErrNoDisponible
	}
	c, err := pgxpool.ParseConfig(dsn)
	if err != nil || c == nil || c.ConnConfig == nil || c.ConnConfig.User == "" ||
		validarTLSPostgreSQLBorradores(&c.ConnConfig.Config, true) != nil {
		return nil, auditoria.ErrNoDisponible
	}
	c.MaxConns, c.MinConns = 2, 0
	c.ConnConfig.ConnectTimeout = 5 * time.Second
	if c.ConnConfig.RuntimeParams == nil {
		c.ConnConfig.RuntimeParams = make(map[string]string)
	}
	p := c.ConnConfig.RuntimeParams
	p["application_name"] = "vec-auditoria-ct-rrhh"
	p["timezone"] = "UTC"
	p["search_path"] = "pg_catalog,pg_temp"
	p["default_transaction_isolation"] = "serializable"
	p["default_transaction_read_only"] = "off"
	p["statement_timeout"] = "15s"
	p["lock_timeout"] = "2s"
	p["idle_in_transaction_session_timeout"] = "20s"
	login := c.ConnConfig.User
	c.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		return comprobarPoolConsultaAuditoriaCTDesarrollo(ctx, conn, login)
	}
	pool, err := pgxpool.NewWithConfig(ctx, c)
	if err != nil {
		return nil, auditoria.ErrNoDisponible
	}
	if err := comprobarPoolConsultaAuditoriaCTDesarrollo(ctx, pool, login); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

func comprobarPoolConsultaAuditoriaCTDesarrollo(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, login string) error {
	if ctx == nil || ctx.Err() != nil || q == nil || login == "" {
		return auditoria.ErrNoDisponible
	}
	const sonda = `SELECT session_user::text,
	 session_user=current_user AND l.rolcanlogin AND l.rolinherit
	 AND NOT l.rolsuper AND NOT l.rolcreatedb AND NOT l.rolcreaterole
	 AND NOT l.rolreplication AND NOT l.rolbypassrls
	 AND NOT g.rolcanlogin AND NOT g.rolbypassrls
	 AND pg_has_role(session_user,g.oid,'MEMBER')
	 AND pg_has_role(session_user,g.oid,'USAGE')
	 AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=l.oid)=1
	 AND EXISTS(SELECT 1 FROM pg_auth_members m WHERE m.member=l.oid AND m.roleid=g.oid
	            AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
	 AND NOT EXISTS(SELECT 1 FROM pg_auth_members m WHERE m.roleid=l.oid)
	 AND coalesce(has_function_privilege(session_user,to_regprocedure($2)::oid,'EXECUTE'),false)
	 AND NOT coalesce(has_function_privilege(session_user,to_regprocedure($3)::oid,'EXECUTE'),false)
	 FROM pg_roles l JOIN pg_roles g ON g.rolname=$1 WHERE l.rolname=session_user`
	var usuario string
	var valido bool
	sondaCtx, cancelar := context.WithTimeout(ctx, 5*time.Second)
	defer cancelar()
	if err := q.QueryRow(sondaCtx, sonda, rolConsultaAuditoriaCTDesarrollo,
		funcionConsultaAuditoriaCTDesarrollo, funcionConsultaAuditoriaBolsaDesarrollo).Scan(&usuario, &valido); err != nil || !valido || usuario != login {
		return auditoria.ErrNoDisponible
	}
	return nil
}

// Los dos pools de autoridad comparten el protocolo central, con LOGIN y
// membresía nominal distintos del ejecutor CT/Bolsa y entre sí.
func abrirPoolAutoridadAuditoriaDesarrollo(ctx context.Context, dsn, rol, aplicacion string) (*pgxpool.Pool, error) {
	if ctx == nil || ctx.Err() != nil || dsn == "" || rol == "" || aplicacion == "" {
		return nil, auditoria.ErrNoDisponible
	}
	c, err := pgxpool.ParseConfig(dsn)
	if err != nil || c == nil || c.ConnConfig == nil || c.ConnConfig.User == "" ||
		validarTLSPostgreSQLBorradores(&c.ConnConfig.Config, true) != nil {
		return nil, auditoria.ErrNoDisponible
	}
	c.MaxConns, c.MinConns = 2, 0
	c.ConnConfig.ConnectTimeout = 5 * time.Second
	if c.ConnConfig.RuntimeParams == nil {
		c.ConnConfig.RuntimeParams = make(map[string]string)
	}
	p := c.ConnConfig.RuntimeParams
	p["application_name"] = aplicacion
	p["timezone"] = "UTC"
	p["search_path"] = "pg_catalog,pg_temp"
	p["statement_timeout"] = "15s"
	p["lock_timeout"] = "2s"
	p["idle_in_transaction_session_timeout"] = "20s"
	login := c.ConnConfig.User
	c.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		return comprobarPoolAutoridadAuditoriaDesarrollo(ctx, conn, login, rol)
	}
	pool, err := pgxpool.NewWithConfig(ctx, c)
	if err != nil {
		return nil, auditoria.ErrNoDisponible
	}
	if err := comprobarPoolAutoridadAuditoriaDesarrollo(ctx, pool, login, rol); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

func comprobarPoolAutoridadAuditoriaDesarrollo(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, login, rol string) error {
	if ctx == nil || ctx.Err() != nil || q == nil || login == "" ||
		(rol != "vec_autorizacion_fuente" && rol != "vec_autorizacion_motivos_evaluador") {
		return auditoria.ErrNoDisponible
	}
	const sonda = `SELECT session_user::text,
	 session_user=current_user AND l.rolcanlogin AND l.rolinherit
	 AND NOT l.rolsuper AND NOT l.rolcreatedb AND NOT l.rolcreaterole
	 AND NOT l.rolreplication AND NOT l.rolbypassrls
	 AND NOT g.rolcanlogin AND NOT g.rolbypassrls
	 AND pg_has_role(session_user,g.oid,'MEMBER') AND pg_has_role(session_user,g.oid,'USAGE')
	 AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=l.oid)=1
	 AND EXISTS(SELECT 1 FROM pg_auth_members m WHERE m.member=l.oid AND m.roleid=g.oid
	            AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
	 AND NOT EXISTS(SELECT 1 FROM pg_auth_members m WHERE m.roleid=l.oid)
	 FROM pg_roles l JOIN pg_roles g ON g.rolname=$1 WHERE l.rolname=session_user`
	var usuario string
	var valido bool
	sondaCtx, cancelar := context.WithTimeout(ctx, 5*time.Second)
	defer cancelar()
	if err := q.QueryRow(sondaCtx, sonda, rol).Scan(&usuario, &valido); err != nil || !valido || usuario != login {
		return auditoria.ErrNoDisponible
	}
	return nil
}
