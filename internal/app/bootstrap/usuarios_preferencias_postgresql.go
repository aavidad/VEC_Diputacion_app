package bootstrap

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/adapters/observabilidad/telemetria/medidorpg"
)

// Usuarios usa un LOGIN propio, con membresía única en el rol ejecutor de
// AD3-106. La composición conserva el pool y lo cierra al apagar el servidor.
func rolEjecutorPreferencias(superficie string) string {
	if superficie == "interna_corporativa" {
		return "vec_usuarios_ejecutor_interno"
	}
	if superficie == "externa_personal" {
		return "vec_usuarios_ejecutor_externo"
	}
	return ""
}
func rolRegistradorPreferencias(superficie string) string {
	if superficie == "interna_corporativa" {
		return "vec_usuarios_registrador_frontera_interno"
	}
	if superficie == "externa_personal" {
		return "vec_usuarios_registrador_frontera_externo"
	}
	return ""
}

func abrirPoolUsuariosPreferencias(ctx context.Context, dsn, rol string) (*pgxpool.Pool, string, error) {
	if dsn == "" || (rol != "vec_usuarios_ejecutor_interno" && rol != "vec_usuarios_ejecutor_externo" && rol != "vec_usuarios_registrador_frontera_interno" && rol != "vec_usuarios_registrador_frontera_externo") {
		return nil, "", errComposicionUsuariosPreferencias
	}
	c, err := pgxpool.ParseConfig(dsn)
	if err != nil || c.ConnConfig.User == "" || validarTLSPostgreSQLBorradores(&c.ConnConfig.Config, true) != nil {
		return nil, "", errComposicionUsuariosPreferencias
	}
	c.MaxConns = 4
	c.MinConns = 0
	c.ConnConfig.ConnectTimeout = 5 * time.Second
	if c.ConnConfig.RuntimeParams == nil {
		c.ConnConfig.RuntimeParams = map[string]string{}
	}
	for k, v := range map[string]string{"application_name": "vec-usuarios-preferencias-desarrollo", "timezone": "UTC", "search_path": "pg_catalog", "statement_timeout": "10s", "lock_timeout": "2s", "idle_in_transaction_session_timeout": "15s"} {
		c.ConnConfig.RuntimeParams[k] = v
	}
	// Mide consultas y esperas de conexión por petición (registro técnico).
	medidorpg.Instrumentar(c)
	pool, err := pgxpool.NewWithConfig(ctx, c)
	if err != nil {
		return nil, "", errComposicionUsuariosPreferencias
	}
	var login string
	var valido bool
	err = pool.QueryRow(ctx, `SELECT session_user::text,
 session_user=current_user AND r.rolcanlogin AND r.rolinherit AND NOT(r.rolsuper OR r.rolcreatedb OR r.rolcreaterole OR r.rolreplication OR r.rolbypassrls)
 AND (SELECT count(*)=1 FROM pg_catalog.pg_auth_members m WHERE m.member=r.oid)
 AND EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members m JOIN pg_catalog.pg_roles g ON g.oid=m.roleid WHERE m.member=r.oid AND g.rolname=$1 AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option AND NOT(g.rolcanlogin OR g.rolsuper OR g.rolcreatedb OR g.rolcreaterole OR g.rolreplication OR g.rolbypassrls)
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members superior WHERE superior.member=g.oid))
 FROM pg_catalog.pg_roles r WHERE r.rolname=session_user`, rol).Scan(&login, &valido)
	if err != nil || !valido {
		pool.Close()
		return nil, "", errComposicionUsuariosPreferencias
	}
	return pool, login, nil
}
