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
}

// Emisor proporciona material V3 nominal desde el contexto acreditado. La
// composición reutiliza el emisor central; HTTP nunca puede suministrarlo.
type Emisor interface {
	EmitirAdministracionPerfiles(context.Context, domain.ContextoActor, domain.InstantaneaAutorizacion, Efecto) (ports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
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
 AND EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m JOIN pg_catalog.pg_roles g ON g.oid=m.roleid
   WHERE m.member=r.oid AND g.rolname='vec_admin_perfiles_ejecutor'
   AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option
   AND NOT (g.rolsuper OR g.rolcanlogin OR g.rolcreatedb OR g.rolcreaterole OR g.rolreplication OR g.rolbypassrls)
   AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members n WHERE n.member=g.oid))
 FROM pg_catalog.pg_roles r WHERE r.rolname=session_user`

const argumentosV3 = `($1::text,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::numeric,$7::numeric,$8::bytea,$9::bytea,$10::bytea,$11::bytea)`
const aplicarSQL = `SELECT vec_autorizacion.aplicar_acto_ordinario_admin_v1` + argumentosV3
const proponerSQL = `SELECT vec_autorizacion.proponer_acto_admin_v1` + argumentosV3
const cerrarSQL = `SELECT vec_autorizacion.cerrar_propuesta_admin_v1` + argumentosV3
const catalogoSQL = `SELECT vec_autorizacion.resolver_rol_administrable_v1($1::text)`

type rolJSON struct {
	VersionRef      string                                    `json:"version_ref"`
	Clase           domain.ClaseControlAdministracionPerfiles `json:"clase"`
	HuellaSHA256    string                                    `json:"huella_sha256"`
	VigenteDesde    time.Time                                 `json:"vigente_desde"`
	VigenteHasta    time.Time                                 `json:"vigente_hasta"`
	UnidadRequerida bool                                      `json:"unidad_requerida"`
}
