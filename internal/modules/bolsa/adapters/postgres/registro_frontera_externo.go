package postgres

import (
	"context"
	"errors"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httppersonal"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var ErrRegistroFronteraBolsaExternaNoDisponible = errors.New("bolsa: registro de frontera exterior no disponible")

const (
	rolRegistroFronteraBolsaExterna  = "vec_bolsa_llamamientos_registrador_portal_externo"
	registrarFronteraBolsaExternaSQL = `SELECT vec_bolsa_llamamientos.registrar_denegacion_portal_externo_v1($1::text,$2::text,$3::text,$4::text,NULLIF($5::text,''))`
)

// La sonda comprueba privilegios efectivos; la función repite las guardas en
// cada escritura. Un permiso nuevo detiene el montaje o el registro.
const preflightFronteraBolsaExternaSQL = `SELECT session_user=current_user
 AND l.rolcanlogin AND l.rolinherit AND NOT(l.rolsuper OR l.rolcreatedb OR l.rolcreaterole OR l.rolreplication OR l.rolbypassrls)
 AND NOT g.rolcanlogin AND g.rolinherit AND NOT(g.rolsuper OR g.rolcreatedb OR g.rolcreaterole OR g.rolreplication OR g.rolbypassrls)
 AND (SELECT count(*)=1 FROM pg_catalog.pg_auth_members m WHERE m.member=l.oid)
 AND EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.member=l.oid AND m.roleid=g.oid
   AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.roleid=l.oid OR m.member=g.oid)
 AND pg_catalog.has_schema_privilege(session_user,'vec_bolsa_llamamientos','USAGE')
 AND NOT pg_catalog.has_schema_privilege(session_user,'vec_bolsa_llamamientos','CREATE')
 AND pg_catalog.has_function_privilege(session_user,
  'vec_bolsa_llamamientos.registrar_denegacion_portal_externo_v1(text,text,text,text,text)','EXECUTE')
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_proc p,
   LATERAL pg_catalog.aclexplode(p.proacl) a
   WHERE p.oid='vec_bolsa_llamamientos.registrar_denegacion_portal_externo_v1(text,text,text,text,text)'::pg_catalog.regprocedure
    AND a.privilege_type='EXECUTE' AND (a.grantee=0 OR (a.grantee=g.oid AND a.is_grantable)))
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_namespace n WHERE n.nspname LIKE 'vec_%'
   AND n.nspname<>'vec_bolsa_llamamientos' AND pg_catalog.has_schema_privilege(session_user,n.oid,'USAGE,CREATE'))
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_namespace n WHERE n.nspname LIKE 'vec_%'
   AND (n.nspowner IN (l.oid,g.oid) OR EXISTS(SELECT 1 FROM pg_catalog.aclexplode(n.nspacl) a
    WHERE a.grantee=l.oid OR (a.grantee=g.oid AND (n.nspname<>'vec_bolsa_llamamientos' OR a.privilege_type<>'USAGE')))))
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
   WHERE n.nspname LIKE 'vec_%' AND (c.relowner IN (l.oid,g.oid)
     OR EXISTS(SELECT 1 FROM pg_catalog.aclexplode(c.relacl) a WHERE a.grantee IN (l.oid,g.oid))))
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_attribute a JOIN pg_catalog.pg_class c ON c.oid=a.attrelid
   JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname LIKE 'vec_%'
   AND EXISTS(SELECT 1 FROM pg_catalog.aclexplode(a.attacl) x WHERE x.grantee IN (l.oid,g.oid)))
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
   WHERE n.nspname LIKE 'vec_%' AND (p.proowner IN (l.oid,g.oid)
    OR EXISTS(SELECT 1 FROM pg_catalog.aclexplode(p.proacl) a WHERE a.grantee=l.oid
      OR (a.grantee=g.oid AND (p.oid<>'vec_bolsa_llamamientos.registrar_denegacion_portal_externo_v1(text,text,text,text,text)'::pg_catalog.regprocedure
       OR a.privilege_type<>'EXECUTE')))))
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_type t JOIN pg_catalog.pg_namespace n ON n.oid=t.typnamespace
   WHERE n.nspname LIKE 'vec_%' AND (t.typowner IN (l.oid,g.oid)
    OR EXISTS(SELECT 1 FROM pg_catalog.aclexplode(t.typacl) a WHERE a.grantee IN (l.oid,g.oid))))
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
   WHERE n.nspname LIKE 'vec_%' AND p.oid<>'vec_bolsa_llamamientos.registrar_denegacion_portal_externo_v1(text,text,text,text,text)'::pg_catalog.regprocedure
     AND pg_catalog.has_function_privilege(session_user,p.oid,'EXECUTE'))
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
   WHERE n.nspname LIKE 'vec_%' AND CASE WHEN c.relkind IN ('r','p','v','m','f') THEN
    pg_catalog.has_table_privilege(session_user,c.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER,MAINTAIN')
      OR pg_catalog.has_any_column_privilege(session_user,c.oid,'SELECT,INSERT,UPDATE,REFERENCES') ELSE false END)
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
   WHERE n.nspname LIKE 'vec_%' AND CASE WHEN c.relkind='S' THEN
     pg_catalog.has_sequence_privilege(session_user,c.oid,'USAGE,SELECT,UPDATE') ELSE false END)
 AND NOT pg_catalog.pg_is_in_recovery()
 FROM pg_catalog.pg_roles l JOIN pg_catalog.pg_roles g ON g.rolname=$1
 WHERE l.rolname=session_user`

var (
	correlacionRegistroBolsaExterna = regexp.MustCompile(`^corr_[0-9a-f]{32}$`)
	actorRegistroBolsaExterna       = regexp.MustCompile(`^per_[A-Za-z0-9_-]{22,128}$`)
)

type consultorFronteraBolsaExterna interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

// El pool lo abre bootstrap con el LOGIN privado del registrador.
type RegistradorFronteraBolsaExternaPostgreSQL struct {
	consultor consultorFronteraBolsaExterna
}

var _ vecports.RegistradorAuditoriaFronteraRutaExacta = (*RegistradorFronteraBolsaExternaPostgreSQL)(nil)

func NuevoRegistradorFronteraBolsaExternaPostgreSQL(ctx context.Context, pool *pgxpool.Pool) (*RegistradorFronteraBolsaExternaPostgreSQL, error) {
	if pool == nil {
		return nil, ErrRegistroFronteraBolsaExternaNoDisponible
	}
	return nuevoRegistradorFronteraBolsaExternaPostgreSQL(ctx, pool)
}

func nuevoRegistradorFronteraBolsaExternaPostgreSQL(ctx context.Context, consultor consultorFronteraBolsaExterna) (*RegistradorFronteraBolsaExternaPostgreSQL, error) {
	if ctx == nil || consultor == nil || ctx.Err() != nil {
		return nil, ErrRegistroFronteraBolsaExternaNoDisponible
	}
	sonda, cancelar := context.WithTimeout(ctx, plazoarranque.Ampliar(3*time.Second))
	defer cancelar()
	var valido bool
	if err := consultor.QueryRow(sonda, preflightFronteraBolsaExternaSQL, rolRegistroFronteraBolsaExterna).Scan(&valido); err != nil || !valido {
		return nil, ErrRegistroFronteraBolsaExternaNoDisponible
	}
	return &RegistradorFronteraBolsaExternaPostgreSQL{consultor: consultor}, nil
}

func ordenFronteraBolsaExternaValida(o vecports.OrdenAuditoriaFronteraRutaExacta) bool {
	if o.Superficie != vecports.SuperficieAuditoriaFronteraRutaExactaBolsaCandidato ||
		(o.CorrelacionRef != "corr_no_disponible" && !correlacionRegistroBolsaExterna.MatchString(o.CorrelacionRef)) {
		return false
	}
	switch o.Ruta {
	case bolsahttp.RutaMiBolsa, bolsahttp.RutaMiBolsaHistorial,
		bolsahttp.RutaMiBolsaSolicitudes, bolsahttp.RutaMiBolsaRespuestas,
		bolsahttp.RutaMiBolsaDisposiciones, bolsahttp.RutaMiBolsaContacto:
	default:
		return false
	}
	switch o.Motivo {
	case vecports.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida:
		return o.ActorRef == ""
	case vecports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado:
		return actorRegistroBolsaExterna.MatchString(o.ActorRef)
	default:
		return false
	}
}

func (r *RegistradorFronteraBolsaExternaPostgreSQL) RegistrarAuditoriaFronteraRutaExacta(ctx context.Context, orden vecports.OrdenAuditoriaFronteraRutaExacta) error {
	if r == nil || r.consultor == nil || ctx == nil {
		return ErrRegistroFronteraBolsaExternaNoDisponible
	}
	if !ordenFronteraBolsaExternaValida(orden) {
		return vecports.ErrOrdenAuditoriaFronteraRutaExactaInvalida
	}
	registro, cancelar := context.WithTimeout(context.WithoutCancel(ctx), plazoarranque.Ampliar(3*time.Second))
	defer cancelar()
	var guardado bool
	if err := r.consultor.QueryRow(registro, registrarFronteraBolsaExternaSQL,
		orden.CorrelacionRef, string(orden.Motivo), orden.Superficie, orden.Ruta, orden.ActorRef,
	).Scan(&guardado); err != nil || !guardado {
		return ErrRegistroFronteraBolsaExternaNoDisponible
	}
	return nil
}
