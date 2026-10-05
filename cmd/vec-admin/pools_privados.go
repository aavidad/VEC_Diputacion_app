package main

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
)

// limitesSesionADMIN son los límites que exige el núcleo VEC-AD-3 al consumir
// una decisión (statement_timeout entre 1 y 15 s e
// idle_in_transaction_session_timeout entre 1 y 20 s); sin ellos responde
// «límites VEC-AD-3 ausentes». Los mismos valores usan los demás procesos.
var limitesSesionADMIN = map[string]string{"statement_timeout": "10s", "lock_timeout": "2s", "idle_in_transaction_session_timeout": "15s"}

// configurarPoolADMIN interpreta el DSN privado y fija los límites de sesión
// en todos los pools del proceso. El DSN privado no admite options, así que
// no puede aportarlos ni sustituirlos.
func configurarPoolADMIN(dsn string) (*pgxpool.Config, error) {
	pc, err := pgxpool.ParseConfig(dsn)
	if err != nil || pc == nil || pc.ConnConfig == nil {
		return nil, errConfiguracionPrivadaPerfiles
	}
	if pc.ConnConfig.RuntimeParams == nil {
		pc.ConnConfig.RuntimeParams = map[string]string{}
	}
	for clave, valor := range limitesSesionADMIN {
		pc.ConnConfig.RuntimeParams[clave] = valor
	}
	return pc, nil
}

// zonaHorariaCargos la exige Personal28 a la sesión que publica cargos
// (current_setting('TimeZone')='UTC'); el DSN privado no puede aportarla.
const zonaHorariaCargos = "UTC"

// configurarPoolCargosADMIN es configurarPoolADMIN más la zona horaria UTC.
func configurarPoolCargosADMIN(dsn string) (*pgxpool.Config, error) {
	pc, err := configurarPoolADMIN(dsn)
	if err != nil {
		return nil, err
	}
	pc.ConnConfig.RuntimeParams["timezone"] = zonaHorariaCargos
	return pc, nil
}

// acreditarZonaHorariaUTC comprueba al arrancar que la sesión del pool queda
// en UTC; si no, la publicación de cargos se denegaría en cada petición.
func acreditarZonaHorariaUTC(ctx context.Context, pool *pgxpool.Pool) error {
	var zona string
	if pool == nil || pool.QueryRow(ctx, `SELECT pg_catalog.current_setting('TimeZone')`).Scan(&zona) != nil || zona != zonaHorariaCargos {
		return errConfiguracionPrivadaPerfiles
	}
	return nil
}

// acreditarPoolCentral acota los cuatro pools cuyos adaptadores reciben
// infraestructura ya acreditada: fuente, registro, motivos y frontera.
func acreditarPoolCentral(ctx context.Context, pool *pgxpool.Pool, grupo string) error {
	var permitido bool
	if pool == nil || pool.QueryRow(ctx, acreditarPoolCentralSQL, grupo).Scan(&permitido) != nil || !permitido {
		return errConfiguracionPrivadaPerfiles
	}
	return nil
}

const acreditarPoolCentralSQL = `SELECT current_user=session_user AND r.rolcanlogin AND r.rolinherit
 AND NOT(r.rolsuper OR r.rolcreaterole OR r.rolcreatedb OR r.rolreplication OR r.rolbypassrls)
 AND (SELECT count(*) FROM pg_catalog.pg_auth_members WHERE member=r.oid)=1
 AND EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members m JOIN pg_catalog.pg_roles g ON g.oid=m.roleid
   WHERE m.member=r.oid AND g.rolname=$1 AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option
   AND NOT(g.rolsuper OR g.rolcanlogin OR g.rolcreaterole OR g.rolcreatedb OR g.rolreplication OR g.rolbypassrls)
   AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members WHERE member=g.oid))
 AND NOT pg_catalog.has_database_privilege(current_user,current_database(),'CREATE')
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_database d WHERE d.datname=current_database()
   AND d.datdba IN(r.oid,pg_catalog.to_regrole($1)))
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_namespace n WHERE n.nspname LIKE 'vec\_%' ESCAPE '\'
   AND(n.nspowner IN(r.oid,pg_catalog.to_regrole($1)) OR pg_catalog.has_schema_privilege(current_user,n.oid,'CREATE')))
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
   WHERE n.nspname LIKE 'vec\_%' ESCAPE '\' AND(c.relowner IN(r.oid,pg_catalog.to_regrole($1))
   OR(c.relkind IN('r','p','v','m','f') AND(pg_catalog.has_table_privilege(current_user,c.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER')
     OR pg_catalog.has_any_column_privilege(current_user,c.oid,'SELECT,INSERT,UPDATE,REFERENCES')))
   OR(c.relkind='S' AND pg_catalog.has_sequence_privilege(current_user,c.oid,'USAGE,SELECT,UPDATE'))))
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_type t JOIN pg_catalog.pg_namespace n ON n.oid=t.typnamespace
   WHERE n.nspname LIKE 'vec\_%' ESCAPE '\' AND t.typowner IN(r.oid,pg_catalog.to_regrole($1)))
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
   WHERE n.nspname LIKE 'vec\_%' ESCAPE '\' AND(p.proowner IN(r.oid,pg_catalog.to_regrole($1))
     OR EXISTS(SELECT 1 FROM pg_catalog.aclexplode(COALESCE(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
        WHERE a.grantee=r.oid AND a.privilege_type='EXECUTE')
     OR(p.prosecdef AND pg_catalog.has_function_privilege(current_user,p.oid,'EXECUTE')
       AND NOT EXISTS(SELECT 1 FROM pg_catalog.aclexplode(COALESCE(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
         WHERE a.grantee=pg_catalog.to_regrole($1) AND a.privilege_type='EXECUTE'))))
 FROM pg_catalog.pg_roles r WHERE r.rolname=session_user`
