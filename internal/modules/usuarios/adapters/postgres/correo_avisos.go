package postgres

import (
	"context"
	"encoding/json"
	"regexp"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/usuarios/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// La lectura vive en una fachada propia del esquema de correos del Área
// personal (Usuarios 000010): el LOGIN interno solo alcanza esa función.
const leerCorreoAvisosSQL = `SELECT vec_usuarios_correos_avisos.correo_activo_avisos_llamamiento_v1($1::text,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::numeric,$7::numeric,$8::bytea,$9::bytea,$10::bytea,$11::bytea)`

// Los seis ajustes mantienen los valores y el alcance de SET LOCAL.
const ajustesTransaccionCorreoAvisosSQL = `SELECT pg_catalog.set_config('search_path','pg_catalog',true),
 pg_catalog.set_config('row_security','on',true),
 pg_catalog.set_config('timezone','UTC',true),
 pg_catalog.set_config('lock_timeout','2s',true),
 pg_catalog.set_config('statement_timeout','5s',true),
 pg_catalog.set_config('idle_in_transaction_session_timeout','20s',true)`

// El pool de avisos usa el LOGIN ejecutor interno de Usuarios: sólo hereda ese
// grupo, puede ejecutar la lectura de avisos y no alcanza ni las tablas ni el
// esquema de correos del Área personal.
const acreditarEjecutorCorreoAvisosSQL = `SELECT session_user=current_user
 AND l.rolcanlogin AND l.rolinherit AND NOT l.rolsuper AND NOT l.rolcreatedb
 AND NOT l.rolcreaterole AND NOT l.rolreplication AND NOT l.rolbypassrls
 AND g.rolname='vec_usuarios_ejecutor_interno' AND NOT g.rolcanlogin AND g.rolinherit AND NOT g.rolbypassrls
 AND pg_catalog.pg_has_role(session_user,g.oid,'MEMBER')
 AND EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.member=l.oid AND m.roleid=g.oid
   AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
 AND (SELECT count(*) FROM pg_catalog.pg_auth_members m WHERE m.member=l.oid)=1
 AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.member=g.oid)
 AND pg_catalog.has_function_privilege(session_user,'vec_usuarios_correos_avisos.correo_activo_avisos_llamamiento_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 AND NOT pg_catalog.has_table_privilege(session_user,'vec_usuarios_correos_externo.correos_direccion','SELECT')
 AND NOT pg_catalog.has_schema_privilege(session_user,'vec_usuarios_correos_externo','USAGE')
 FROM pg_catalog.pg_roles l JOIN pg_catalog.pg_roles g ON g.rolname='vec_usuarios_ejecutor_interno'
 WHERE l.rolname=session_user`

var (
	patronAuditoriaAvisos = regexp.MustCompile(`^[A-Za-z0-9:._-]{1,256}$`)
	patronPersonaAvisos   = regexp.MustCompile(`^per_[A-Za-z0-9_-]{22,128}$`)
)

// RegistroCorreoAvisosPostgreSQL lee el correo activo para un aviso de
// llamamiento. No posee el pool y nunca devuelve la dirección en claro.
type RegistroCorreoAvisosPostgreSQL struct {
	iniciar func(context.Context) (transaccionCorreos, error)
}

var _ ports.RegistroCorreoAvisos = (*RegistroCorreoAvisosPostgreSQL)(nil)

// NuevoRegistroCorreoAvisosPostgreSQL acredita el LOGIN y la función antes
// de admitir lecturas; sin la migración Usuarios 000008 no se compone.
func NuevoRegistroCorreoAvisosPostgreSQL(ctx context.Context, pool *pgxpool.Pool) (*RegistroCorreoAvisosPostgreSQL, error) {
	if ctx == nil || pool == nil || ctx.Err() != nil {
		return nil, ports.ErrCorreosNoDisponible
	}
	r := &RegistroCorreoAvisosPostgreSQL{iniciar: func(ctx context.Context) (transaccionCorreos, error) {
		tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
		if err != nil {
			return nil, err
		}
		return transaccionCorreosPGX{tx}, nil
	}}
	tx, err := r.abrir(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(context.Background())
	if err := tx.Commit(ctx); err != nil {
		return nil, errorCorreosSeguro(ctx, err)
	}
	return r, nil
}

func (r *RegistroCorreoAvisosPostgreSQL) abrir(ctx context.Context) (transaccionCorreos, error) {
	if ctx == nil || r == nil || r.iniciar == nil {
		return nil, ports.ErrCorreosNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	tx, err := r.iniciar(ctx)
	if err != nil {
		return nil, errorCorreosSeguro(ctx, err)
	}
	if tx == nil {
		return nil, ports.ErrCorreosNoDisponible
	}
	fallar := func(err error) (transaccionCorreos, error) {
		_ = tx.Rollback(context.Background())
		return nil, errorCorreosSeguro(ctx, err)
	}
	if _, err := tx.Exec(ctx, ajustesTransaccionCorreoAvisosSQL); err != nil {
		return fallar(err)
	}
	var valido bool
	if err := tx.QueryRow(ctx, acreditarEjecutorCorreoAvisosSQL).Scan(&valido); err != nil {
		return fallar(err)
	}
	if !valido {
		_ = tx.Rollback(context.Background())
		return nil, ports.ErrCorreosNoDisponible
	}
	return tx, nil
}

type lecturaCorreoAvisosSQL struct {
	Encontrado   bool            `json:"encontrado"`
	AuditoriaRef string          `json:"auditoria_ref"`
	PersonaRef   string          `json:"persona_ref,omitempty"`
	CorreoRef    string          `json:"correo_ref,omitempty"`
	Sobre        *sobreCorreoSQL `json:"sobre,omitempty"`
}

// LeerCorreoActivoAvisos consume la V3 y lee en una sola transacción. Una
// carrera SERIALIZABLE o cualquier otro fallo se devuelven como
// indisponibilidad: quien avisa usa entonces el correo del alta.
func (r *RegistroCorreoAvisosPostgreSQL) LeerCorreoActivoAvisos(ctx context.Context, material []byte, v3 vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.LecturaCorreoAvisos, error) {
	vacia := ports.LecturaCorreoAvisos{}
	if len(material) < 2 || len(material) > 4096 || !json.Valid(material) || v3.ValidarEstructura() != nil ||
		v3.ResumenCapacidad().AudienciaConsumo() != ports.AudienciaCorreoAvisosLlamamientoInterna {
		return vacia, ports.ErrCorreosProhibido
	}
	tx, err := r.abrir(ctx)
	if err != nil {
		return vacia, err
	}
	defer tx.Rollback(context.Background())
	var bruto []byte
	if err := tx.QueryRow(ctx, leerCorreoAvisosSQL, argumentosCorreosV3(material, v3)...).Scan(&bruto); err != nil {
		return vacia, errorCorreosSeguro(ctx, err)
	}
	var l lecturaCorreoAvisosSQL
	if decodificarCorreosEstricto(bruto, &l) != nil || !patronAuditoriaAvisos.MatchString(l.AuditoriaRef) {
		return vacia, ports.ErrCorreosNoDisponible
	}
	lectura := ports.LecturaCorreoAvisos{Encontrado: l.Encontrado, AuditoriaRef: l.AuditoriaRef}
	if l.Encontrado {
		if l.Sobre == nil || !patronCorreoRef.MatchString(l.CorreoRef) || !patronPersonaAvisos.MatchString(l.PersonaRef) {
			return vacia, ports.ErrCorreosNoDisponible
		}
		sobre, err := l.Sobre.decodificar(l.PersonaRef, l.CorreoRef)
		if err != nil {
			return vacia, err
		}
		lectura.PersonaRef, lectura.CorreoRef, lectura.Sobre = l.PersonaRef, l.CorreoRef, sobre
	} else if l.PersonaRef != "" || l.CorreoRef != "" || l.Sobre != nil {
		return vacia, ports.ErrCorreosNoDisponible
	}
	if err := tx.Commit(ctx); err != nil {
		return vacia, errorCorreosSeguro(ctx, err)
	}
	return lectura, nil
}
