package bootstrap

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/shared/postgresql"
	"vec-diputacion-granada/internal/shared/telemetria"
)

var errMontajeInscripcionBolsa = errors.New("bootstrap: montaje de inscripcion Bolsa no disponible")

// Cada proceso abre sólo el lector de su superficie. Los tres LOGIN y sus
// roles NOLOGIN pertenecen a B96; no se crean ni se conceden al arrancar.
type claseLectorInscripcionBolsa uint8

const (
	lectorInscripcionExterno claseLectorInscripcionBolsa = iota + 1
	lectorInscripcionEmpleado
	lectorInscripcionRRHH
)

func (c claseLectorInscripcionBolsa) identidad() (login, rol, aplicacion string, valida bool) {
	switch c {
	case lectorInscripcionExterno:
		return "vec_bolsa_inscripciones_lector", "vec_bolsa_llamamientos_lector_inscripciones", "vec-bolsa-inscripcion-lector-externo", true
	case lectorInscripcionEmpleado:
		return "vec_bolsa_inscripciones_empleado_lector", "vec_bolsa_llamamientos_lector_inscripciones_empleado", "vec-bolsa-inscripcion-lector-empleado", true
	case lectorInscripcionRRHH:
		return "vec_bolsa_inscripciones_rrhh_lector", "vec_bolsa_llamamientos_lector_inscripciones_rrhh", "vec-bolsa-inscripcion-lector-rrhh", true
	default:
		return "", "", "", false
	}
}

func abrirLectorInscripcionBolsa(ctx context.Context, dsn string, clase claseLectorInscripcionBolsa) (*pgxpool.Pool, error) {
	login, rol, aplicacion, valida := clase.identidad()
	if ctx == nil || ctx.Err() != nil || dsn == "" || !valida {
		return nil, errMontajeInscripcionBolsa
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil || cfg == nil || cfg.ConnConfig == nil || cfg.ConnConfig.User != login ||
		validarTLSPostgreSQLBorradores(&cfg.ConnConfig.Config, true) != nil {
		return nil, errMontajeInscripcionBolsa
	}
	postgresql.FijarTamanoPool(cfg, dsn, 4)
	cfg.MinConns = 0
	cfg.ConnConfig.ConnectTimeout = 5 * time.Second
	if cfg.ConnConfig.RuntimeParams == nil {
		cfg.ConnConfig.RuntimeParams = make(map[string]string)
	}
	for k, v := range map[string]string{
		"application_name": aplicacion, "timezone": "UTC", "search_path": "pg_catalog",
		"statement_timeout": "15s", "lock_timeout": "2s", "idle_in_transaction_session_timeout": "20s",
	} {
		cfg.ConnConfig.RuntimeParams[k] = v
	}
	telemetria.Instrumentar(cfg)
	pool, err := postgresql.NuevoPoolConPreflightTEMP(ctx, cfg)
	if err != nil {
		return nil, errMontajeInscripcionBolsa
	}
	var permitido bool
	err = pool.QueryRow(ctx, `WITH login AS (
 SELECT oid,rolcanlogin,rolinherit,rolsuper,rolcreatedb,rolcreaterole,rolreplication,rolbypassrls
 FROM pg_catalog.pg_roles WHERE rolname=session_user
), membresia AS (
 SELECT m.member,m.inherit_option,m.set_option,m.admin_option,g.rolname AS nombre,
        g.rolcanlogin,g.rolsuper,g.rolcreatedb,g.rolcreaterole,g.rolreplication,g.rolbypassrls,g.oid AS grupo_oid
 FROM pg_catalog.pg_auth_members m JOIN pg_catalog.pg_roles g ON g.oid=m.roleid
 WHERE m.member=(SELECT oid FROM login)
), funciones AS (
 SELECT p.oid,p.proname,p.prosecdef FROM pg_catalog.pg_proc p
 JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
 WHERE n.nspname='vec_bolsa_llamamientos' AND pg_catalog.has_function_privilege(session_user,p.oid,'EXECUTE')
), tablas AS (
 SELECT c.oid FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
 WHERE n.nspname='vec_bolsa_llamamientos' AND c.relkind IN ('r','p','v','m','f')
), secuencias AS (
 SELECT c.oid FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
 WHERE n.nspname='vec_bolsa_llamamientos' AND c.relkind='S'
)
SELECT session_user=$1
 AND (SELECT rolcanlogin AND rolinherit AND NOT(rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls) FROM login)
 AND (SELECT count(*)=1 AND bool_and(nombre=$2 AND inherit_option AND NOT set_option AND NOT admin_option
     AND NOT(rolcanlogin OR rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls)
     AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members superior WHERE superior.member=grupo_oid)) FROM membresia)
 AND (SELECT count(*)=1 AND bool_and(proname='consultar_inscripcion_v1' AND prosecdef) FROM funciones)
 AND NOT EXISTS(SELECT 1 FROM tablas t WHERE pg_catalog.has_table_privilege(session_user,t.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER')
  OR pg_catalog.has_any_column_privilege(session_user,t.oid,'SELECT,INSERT,UPDATE,REFERENCES'))
 AND NOT EXISTS(SELECT 1 FROM secuencias s WHERE pg_catalog.has_sequence_privilege(session_user,s.oid,'USAGE,SELECT,UPDATE'))
 AND NOT pg_catalog.has_schema_privilege(session_user,'vec_bolsa_llamamientos','CREATE')`, login, rol).Scan(&permitido)
	if err != nil || !permitido {
		pool.Close()
		return nil, errMontajeInscripcionBolsa
	}
	return pool, nil
}
