package bootstrap

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	postgresqlcompartido "vec-diputacion-granada/internal/shared/postgresql"
	"vec-diputacion-granada/internal/shared/telemetria"
)

const (
	rolGuardarPreparacionBasesV3       = "vec_bolsa_convocatorias_ejecutor_preparacion_bases"
	rolConsultarPreparacionBasesV3     = "vec_bolsa_convocatorias_lector_preparacion_bases"
	funcionGuardarPreparacionBasesV3   = "vec_bolsa_convocatorias.guardar_preparacion_bases_v3(text,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)"
	funcionConsultarPreparacionBasesV3 = "vec_bolsa_convocatorias.obtener_preparacion_bases_v3(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)"
)

// La comprobación se repite en cada conexión física del pool. Las cuentas
// nominales no tienen acceso directo a material, historia o concesiones.
const sondaPoolPreparacionBasesV3 = `SELECT session_user::text,
 session_user=current_user AND identidad.rolcanlogin AND identidad.rolinherit
 AND NOT identidad.rolsuper AND NOT identidad.rolcreatedb AND NOT identidad.rolcreaterole
 AND NOT identidad.rolreplication AND NOT identidad.rolbypassrls
 AND pg_catalog.pg_has_role(session_user,$1,'MEMBER')
 AND NOT pg_catalog.pg_has_role(session_user,$2,'MEMBER')
 AND (SELECT count(*)=1 AND bool_and(m.inherit_option AND NOT m.set_option AND NOT m.admin_option
   AND m.roleid=(SELECT oid FROM pg_catalog.pg_roles WHERE rolname=$1))
   FROM pg_catalog.pg_auth_members m WHERE m.member=identidad.oid)
 AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m
   WHERE m.member=(SELECT oid FROM pg_catalog.pg_roles WHERE rolname=$1))
 AND pg_catalog.has_schema_privilege(session_user,'vec_bolsa_convocatorias','USAGE')
 AND coalesce(pg_catalog.has_function_privilege(session_user,pg_catalog.to_regprocedure($3)::oid,'EXECUTE'),false)
 AND NOT coalesce(pg_catalog.has_function_privilege(session_user,pg_catalog.to_regprocedure($4)::oid,'EXECUTE'),false)
 AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_class c
   JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
   WHERE n.nspname IN ('vec_bolsa_convocatorias','vec_autorizacion','vec_autorizacion_atestada_v3')
   AND c.relkind IN ('r','p','v','m')
   AND (pg_catalog.has_table_privilege(session_user,c.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER,MAINTAIN')
     OR pg_catalog.has_any_column_privilege(session_user,c.oid,'SELECT,INSERT,UPDATE,REFERENCES')))
 FROM pg_catalog.pg_roles identidad WHERE identidad.rolname=session_user`

func comprobarPoolPreparacionBasesV3(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, guardar bool) (string, error) {
	if ctx == nil || q == nil {
		return "", errMontajePreparacionBasesV3
	}
	rol, ajeno, funcion, otra := rolConsultarPreparacionBasesV3, rolGuardarPreparacionBasesV3, funcionConsultarPreparacionBasesV3, funcionGuardarPreparacionBasesV3
	if guardar {
		rol, ajeno, funcion, otra = ajeno, rol, otra, funcion
	}
	var usuario string
	var valido bool
	if q.QueryRow(ctx, sondaPoolPreparacionBasesV3, rol, ajeno, funcion, otra).Scan(&usuario, &valido) != nil || !valido || usuario == "" {
		return "", errMontajePreparacionBasesV3
	}
	return usuario, nil
}

func abrirPoolPreparacionBasesV3(ctx context.Context, dsn string, guardar bool) (*pgxpool.Pool, string, error) {
	c, err := pgxpool.ParseConfig(strings.TrimSpace(dsn))
	if err != nil || c == nil || c.ConnConfig == nil || c.ConnConfig.User == "" || len(c.ConnConfig.Fallbacks) != 0 ||
		validarTLSPostgreSQLBorradores(&c.ConnConfig.Config, true) != nil {
		return nil, "", errMontajePreparacionBasesV3
	}
	c.MaxConns, c.MinConns = 4, 0
	c.ConnConfig.ConnectTimeout = 5 * time.Second
	if c.ConnConfig.RuntimeParams == nil {
		c.ConnConfig.RuntimeParams = make(map[string]string)
	}
	p := c.ConnConfig.RuntimeParams
	p["application_name"] = "vec-seleccion-preparacion-bases-consultar-v3"
	if guardar {
		p["application_name"] = "vec-seleccion-preparacion-bases-guardar-v3"
	}
	p["timezone"], p["search_path"] = "UTC", "pg_catalog,pg_temp"
	p["default_transaction_isolation"], p["default_transaction_read_only"] = "serializable", "off"
	p["statement_timeout"], p["lock_timeout"], p["idle_in_transaction_session_timeout"] = "15s", "3s", "20s"
	c.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		_, err := comprobarPoolPreparacionBasesV3(ctx, conn, guardar)
		return err
	}
	telemetria.Instrumentar(c) // consultas por petición en el registro de acceso
	pool, err := postgresqlcompartido.NuevoPoolConPreflightTEMP(ctx, c)
	if err != nil {
		return nil, "", errMontajePreparacionBasesV3
	}
	u, err := comprobarPoolPreparacionBasesV3(ctx, pool, guardar)
	if err != nil {
		pool.Close()
		return nil, "", errMontajePreparacionBasesV3
	}
	return pool, u, nil
}
