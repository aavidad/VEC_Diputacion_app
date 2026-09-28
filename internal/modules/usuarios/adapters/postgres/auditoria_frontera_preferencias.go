package postgres

import (
	"context"
	"errors"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var ErrAuditoriaFronteraPreferenciasNoDisponible = errors.New("usuarios: auditoria de frontera no disponible")

const (
	superficieAuditoriaPreferencias = "api.usuarios.preferencias.ruta_exacta"
	rutaAuditoriaPreferencias       = "/api/vec/usuarios/mis-preferencias"
	registrarDenegacionSQL          = `SELECT vec_usuarios.registrar_denegacion_preferencias_v1($1::text,$2::text,$3::text,$4::text,NULLIF($5::text,''))`
)

// La sonda comprueba permisos efectivos, no solo una membresía nominal. La
// función SQL vuelve a comprobar esta frontera en cada registro.
const preflightDenegacionSQL = `SELECT session_user=current_user
 AND l.rolcanlogin AND l.rolinherit AND NOT l.rolsuper AND NOT l.rolcreatedb
 AND NOT l.rolcreaterole AND NOT l.rolreplication AND NOT l.rolbypassrls
 AND NOT g.rolcanlogin AND g.rolinherit AND NOT g.rolsuper AND NOT g.rolcreatedb
 AND NOT g.rolcreaterole AND NOT g.rolreplication AND NOT g.rolbypassrls
 AND pg_catalog.pg_has_role(session_user,g.oid,'MEMBER') AND pg_catalog.pg_has_role(session_user,g.oid,'USAGE')
 AND (SELECT count(*) FROM pg_catalog.pg_auth_members m WHERE m.member=l.oid)=1
 AND EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.member=l.oid AND m.roleid=g.oid
   AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
 AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.roleid=l.oid OR m.member=g.oid)
 AND pg_catalog.has_schema_privilege(session_user,'vec_usuarios','USAGE')
 AND NOT pg_catalog.has_schema_privilege(session_user,'vec_usuarios','CREATE')
 AND pg_catalog.has_function_privilege(session_user,
   'vec_usuarios.registrar_denegacion_preferencias_v1(text,text,text,text,text)','EXECUTE')
 AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_proc p WHERE p.pronamespace='vec_usuarios'::pg_catalog.regnamespace
   AND p.oid<>'vec_usuarios.registrar_denegacion_preferencias_v1(text,text,text,text,text)'::pg_catalog.regprocedure
   AND pg_catalog.has_function_privilege(session_user,p.oid,'EXECUTE'))
 AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_class c WHERE c.relnamespace='vec_usuarios'::pg_catalog.regnamespace
   AND c.relkind IN ('r','p','v','m') AND (pg_catalog.has_table_privilege(session_user,c.oid,'SELECT')
   OR pg_catalog.has_table_privilege(session_user,c.oid,'INSERT') OR pg_catalog.has_table_privilege(session_user,c.oid,'UPDATE')
   OR pg_catalog.has_table_privilege(session_user,c.oid,'DELETE') OR pg_catalog.has_table_privilege(session_user,c.oid,'TRUNCATE')
   OR pg_catalog.has_table_privilege(session_user,c.oid,'REFERENCES') OR pg_catalog.has_table_privilege(session_user,c.oid,'TRIGGER')
   OR pg_catalog.has_table_privilege(session_user,c.oid,'MAINTAIN')
   OR pg_catalog.has_any_column_privilege(session_user,c.oid,'SELECT')
   OR pg_catalog.has_any_column_privilege(session_user,c.oid,'INSERT')
   OR pg_catalog.has_any_column_privilege(session_user,c.oid,'UPDATE')
   OR pg_catalog.has_any_column_privilege(session_user,c.oid,'REFERENCES')))
 AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_class c WHERE c.relnamespace='vec_usuarios'::pg_catalog.regnamespace
   AND c.relkind='S' AND (pg_catalog.has_sequence_privilege(session_user,c.oid,'USAGE')
   OR pg_catalog.has_sequence_privilege(session_user,c.oid,'SELECT')
   OR pg_catalog.has_sequence_privilege(session_user,c.oid,'UPDATE')))
 AND NOT pg_catalog.pg_is_in_recovery()
 FROM pg_catalog.pg_roles l JOIN pg_catalog.pg_roles g ON g.rolname='vec_usuarios_registrador_frontera'
 WHERE l.rolname=session_user`

var (
	correlacionPreferencias = regexp.MustCompile(`^corr_[0-9a-f]{32}$`)
	actorPreferencias       = regexp.MustCompile(`^per_[A-Za-z0-9_-]{22,128}$`)
)

type consultorDenegaciones interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

var _ vecports.RegistradorAuditoriaFronteraRutaExacta = (*RegistradorDenegacionPreferenciasPostgreSQL)(nil)

// El pool pertenece a bootstrap y usa un LOGIN privado distinto del ejecutor.
type RegistradorDenegacionPreferenciasPostgreSQL struct {
	consultor consultorDenegaciones
}

func NuevoRegistradorDenegacionPreferenciasPostgreSQL(ctx context.Context, pool *pgxpool.Pool) (*RegistradorDenegacionPreferenciasPostgreSQL, error) {
	if pool == nil {
		return nil, ErrAuditoriaFronteraPreferenciasNoDisponible
	}
	return nuevoRegistradorDenegacionPreferenciasPostgreSQL(ctx, pool)
}

func nuevoRegistradorDenegacionPreferenciasPostgreSQL(ctx context.Context, consultor consultorDenegaciones) (*RegistradorDenegacionPreferenciasPostgreSQL, error) {
	if ctx == nil || consultor == nil {
		return nil, ErrAuditoriaFronteraPreferenciasNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	sonda, cancelar := context.WithTimeout(ctx, 3*time.Second)
	defer cancelar()
	var valido bool
	if err := consultor.QueryRow(sonda, preflightDenegacionSQL).Scan(&valido); err != nil || !valido {
		return nil, ErrAuditoriaFronteraPreferenciasNoDisponible
	}
	return &RegistradorDenegacionPreferenciasPostgreSQL{consultor: consultor}, nil
}

func (r *RegistradorDenegacionPreferenciasPostgreSQL) RegistrarAuditoriaFronteraRutaExacta(ctx context.Context, orden vecports.OrdenAuditoriaFronteraRutaExacta) error {
	if r == nil || r.consultor == nil || ctx == nil {
		return ErrAuditoriaFronteraPreferenciasNoDisponible
	}
	if !ordenDenegacionPreferenciasValida(orden) {
		return vecports.ErrOrdenAuditoriaFronteraRutaExactaInvalida
	}
	// La petición ya fue denegada. Su cancelación no debe impedir registrar el
	// hecho; se conservan solo valores y trazas del contexto, nunca autoridad.
	registro, cancelar := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancelar()
	var guardado bool
	if err := r.consultor.QueryRow(registro, registrarDenegacionSQL,
		orden.CorrelacionRef, string(orden.Motivo), orden.Superficie, orden.Ruta, orden.ActorRef,
	).Scan(&guardado); err != nil || !guardado {
		return ErrAuditoriaFronteraPreferenciasNoDisponible
	}
	return nil
}

func ordenDenegacionPreferenciasValida(o vecports.OrdenAuditoriaFronteraRutaExacta) bool {
	if o.Superficie != superficieAuditoriaPreferencias || o.Ruta != rutaAuditoriaPreferencias ||
		(o.CorrelacionRef != "corr_no_disponible" && !correlacionPreferencias.MatchString(o.CorrelacionRef)) {
		return false
	}
	switch o.Motivo {
	case vecports.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida:
		return o.ActorRef == ""
	case vecports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado:
		return actorPreferencias.MatchString(o.ActorRef)
	default:
		return false
	}
}
