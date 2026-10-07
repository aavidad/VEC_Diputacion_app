// Package postgres conecta los contratos ADMIN existentes con las fachadas
// nominales AUT24. No crea perfiles, permisos ni una autoridad alternativa.
package postgres

import (
	"context"
	"reflect"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// Efecto es el material de negocio que el emisor central liga a la capacidad.
// No incluye decisiones, contextos serializados ni datos de autenticación.
type Efecto struct {
	Accion, Audiencia, Referencia string
	Material                      []byte
	// CorrelacionAccesoRef corresponde al acceso actual, también en replay.
	// No forma parte del material semántico ni reescribe el recibo histórico.
	CorrelacionAccesoRef string
}

// Emisor proporciona material V3 nominal desde el contexto acreditado. La
// composición reutiliza el emisor central; HTTP nunca puede suministrarlo.
// Consume el par V2 emitido por las autoridades de sesión/contexto y selecciona
// sus campos nominales para PDP V3; nunca serializa un contexto V2 como V3.
// Cada invocación, incluido replay, revalida y audita el acceso actual; el
// consumo definitivo ocurre en la fachada SQL junto al efecto y su recibo.
type Emisor interface {
	EmitirAdministracionPerfiles(context.Context, domain.ContextoActor, domain.EvidenciaSesionAdministracionPerfiles, domain.InstantaneaAutorizacion, Efecto) (ports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

type conexion interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type Autoridad struct {
	pool   conexion
	emisor Emisor
	reloj  ports.Reloj
}

var _ ports.CatalogoRolesAdministrables = (*Autoridad)(nil)
var _ ports.AutoridadActosAdministracionPerfiles = (*Autoridad)(nil)
var _ ports.AutoridadLotesAdministracionPerfiles = (*Autoridad)(nil)

// Nueva recibe únicamente un pool con LOGIN runtime acotado a AUT24 y el
// emisor central. Las funciones ausentes o no autorizadas fallan cerradas.
// El despliegue debe acreditar sus ACL; el adaptador no concede privilegios.
func Nueva(ctx context.Context, pool *pgxpool.Pool, emisor Emisor, reloj ports.Reloj) (*Autoridad, error) {
	if pool == nil || ausente(emisor) || ausente(reloj) {
		return nil, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	return nueva(ctx, pool, emisor, reloj)
}

func nueva(ctx context.Context, pool conexion, emisor Emisor, reloj ports.Reloj) (*Autoridad, error) {
	if ctx == nil || ausente(pool) || ausente(emisor) || ausente(reloj) {
		return nil, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var acreditado bool
	if err := pool.QueryRow(ctx, acreditarSQL).Scan(&acreditado); err != nil || !acreditado {
		return nil, traducir(ctx, ports.ErrAutoridadAdministracionPerfilesNoDisponible)
	}
	return &Autoridad{pool: pool, emisor: emisor, reloj: reloj}, nil
}

func ausente(v any) bool {
	if v == nil {
		return true
	}
	x := reflect.ValueOf(v)
	switch x.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Func, reflect.Map, reflect.Slice, reflect.Chan:
		return x.IsNil()
	}
	return false
}

const acreditarSQL = `SELECT current_user = session_user AND r.rolcanlogin AND r.rolinherit
 AND NOT (r.rolsuper OR r.rolcreatedb OR r.rolcreaterole OR r.rolreplication OR r.rolbypassrls)
 AND (SELECT count(*) FROM pg_catalog.pg_auth_members m WHERE m.member=r.oid)=1
 AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_database db WHERE db.datname=current_database()
   AND (db.datdba=r.oid OR db.datdba=pg_catalog.to_regrole('vec_admin_perfiles_ejecutor')))
 AND NOT pg_catalog.has_database_privilege(current_user,current_database(),'CREATE')
 AND pg_catalog.to_regprocedure('vec_autorizacion.resolver_rol_administrable_v1(text)') IS NOT NULL
 AND pg_catalog.to_regprocedure('vec_autorizacion.aplicar_acto_ordinario_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 AND pg_catalog.to_regprocedure('vec_autorizacion.proponer_acto_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 AND pg_catalog.to_regprocedure('vec_autorizacion.cerrar_propuesta_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 AND pg_catalog.has_function_privilege(current_user,pg_catalog.to_regprocedure('vec_autorizacion.resolver_rol_administrable_v1(text)'),'EXECUTE')
 AND pg_catalog.has_function_privilege(current_user,pg_catalog.to_regprocedure('vec_autorizacion.aplicar_acto_ordinario_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),'EXECUTE')
 AND pg_catalog.has_function_privilege(current_user,pg_catalog.to_regprocedure('vec_autorizacion.proponer_acto_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),'EXECUTE')
 AND pg_catalog.has_function_privilege(current_user,pg_catalog.to_regprocedure('vec_autorizacion.cerrar_propuesta_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),'EXECUTE')
 AND EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m JOIN pg_catalog.pg_roles g ON g.oid=m.roleid
   WHERE m.member=r.oid AND g.rolname='vec_admin_perfiles_ejecutor'
   AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option
   AND NOT (g.rolsuper OR g.rolcanlogin OR g.rolcreatedb OR g.rolcreaterole OR g.rolreplication OR g.rolbypassrls)
   AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members n WHERE n.member=g.oid))
 AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_namespace n
   WHERE n.nspname LIKE 'vec\_%' ESCAPE '\'
   AND (n.nspowner=r.oid OR n.nspowner=pg_catalog.to_regrole('vec_admin_perfiles_ejecutor')
     OR pg_catalog.has_schema_privilege(current_user,n.oid,'CREATE')))
 AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_type t JOIN pg_catalog.pg_namespace n ON n.oid=t.typnamespace
   WHERE n.nspname LIKE 'vec\_%' ESCAPE '\'
   AND (t.typowner=r.oid OR t.typowner=pg_catalog.to_regrole('vec_admin_perfiles_ejecutor')))
 AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
   WHERE n.nspname LIKE 'vec\_%' ESCAPE '\'
   AND (c.relowner=r.oid OR c.relowner=pg_catalog.to_regrole('vec_admin_perfiles_ejecutor')
    OR (c.relkind IN ('r','p','v','m','f') AND pg_catalog.has_table_privilege(current_user,c.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER'))
    OR (c.relkind IN ('r','p','v','m','f') AND pg_catalog.has_any_column_privilege(current_user,c.oid,'SELECT,INSERT,UPDATE,REFERENCES'))
    OR (c.relkind='S' AND pg_catalog.has_sequence_privilege(current_user,c.oid,'USAGE,SELECT,UPDATE'))))
 AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
   WHERE n.nspname LIKE 'vec\_%' ESCAPE '\'
   AND (p.proowner=r.oid OR p.proowner=pg_catalog.to_regrole('vec_admin_perfiles_ejecutor')
     OR (pg_catalog.has_function_privilege(current_user,p.oid,'EXECUTE')
       AND NOT COALESCE(n.nspname='vec_autorizacion' AND p.oid=ANY(ARRAY[
         pg_catalog.to_regprocedure('vec_autorizacion.resolver_rol_administrable_v1(text)'),
         pg_catalog.to_regprocedure('vec_autorizacion.aplicar_acto_ordinario_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),
         pg_catalog.to_regprocedure('vec_autorizacion.proponer_acto_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),
         pg_catalog.to_regprocedure('vec_autorizacion.cerrar_propuesta_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
       )]::oid[]),false)
       AND (p.prosecdef OR EXISTS (SELECT 1 FROM pg_catalog.aclexplode(COALESCE(p.proacl,pg_catalog.acldefault('f',p.proowner))) acl
         WHERE acl.privilege_type='EXECUTE' AND acl.grantee IN (r.oid,pg_catalog.to_regrole('vec_admin_perfiles_ejecutor')))))))
 FROM pg_catalog.pg_roles r WHERE r.rolname=session_user`

const argumentosV3 = `($1::text,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::numeric,$7::numeric,$8::bytea,$9::bytea,$10::bytea,$11::bytea)`
const aplicarSQL = `SELECT vec_autorizacion.aplicar_acto_ordinario_admin_v1` + argumentosV3
const proponerSQL = `SELECT vec_autorizacion.proponer_acto_admin_v1` + argumentosV3
const cerrarSQL = `SELECT vec_autorizacion.cerrar_propuesta_admin_v1` + argumentosV3
const catalogoSQL = `SELECT vec_autorizacion.resolver_rol_administrable_v1($1::text)`

type rolJSON struct {
	CategoriaAdmin  *string                                   `json:"categoria_admin"`
	VersionRef      string                                    `json:"version_ref"`
	Clase           domain.ClaseControlAdministracionPerfiles `json:"clase"`
	HuellaSHA256    string                                    `json:"huella_sha256"`
	VigenteDesde    time.Time                                 `json:"vigente_desde"`
	VigenteHasta    time.Time                                 `json:"vigente_hasta"`
	UnidadRequerida *bool                                     `json:"unidad_requerida"`
}
