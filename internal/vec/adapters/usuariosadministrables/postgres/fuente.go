// Package postgres consulta los metadatos nominales de AUT43 con una decisión
// V3 nueva por lectura. El montaje y el emisor pertenecen a la composición.
package postgres

import (
	"context"
	"errors"
	"maps"
	"reflect"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const listarSQL = `SELECT vec_autorizacion.listar_usuarios_admin_v1($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`
const consultarSQL = `SELECT vec_autorizacion.consultar_usuario_admin_v1($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`

type conexion interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type Configuracion struct {
	OrganizacionRef string
	UnidadRef       string
	Proceso         string
	Canal           string
	MotivoDenegado  domain.ReferenciaEntradaCatalogo
	MotivoError     domain.ReferenciaEntradaCatalogo
}

type Fuente struct {
	pool     conexion
	emisor   ports.EmisorLecturaUsuariosAdministrables
	fuente   ports.FuenteAutorizacion
	intentos ports.RegistradorIntentosAuditoria
	reloj    ports.Reloj
	config   Configuracion
	ambito   ambito
}

var _ ports.FuenteUsuariosAdministrables = (*Fuente)(nil)

func Nueva(ctx context.Context, pool *pgxpool.Pool, emisor ports.EmisorLecturaUsuariosAdministrables,
	fuente ports.FuenteAutorizacion, intentos ports.RegistradorIntentosAuditoria, reloj ports.Reloj, config Configuracion,
) (*Fuente, error) {
	return nueva(ctx, pool, emisor, fuente, intentos, reloj, config)
}

func nueva(ctx context.Context, pool conexion, emisor ports.EmisorLecturaUsuariosAdministrables,
	fuente ports.FuenteAutorizacion, intentos ports.RegistradorIntentosAuditoria, reloj ports.Reloj, config Configuracion,
) (*Fuente, error) {
	if ctx == nil || ausente(pool) || ausente(emisor) || ausente(fuente) || ausente(intentos) || ausente(reloj) {
		return nil, ports.ErrLecturaUsuariosAdministrablesNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	a := ambito{config.OrganizacionRef, config.UnidadRef}
	if a.validar() != nil || config.Proceso == "" || config.Canal != "administracion_privilegiada" ||
		config.MotivoDenegado.Validar() != nil || config.MotivoError.Validar() != nil {
		return nil, ports.ErrLecturaUsuariosAdministrablesNoDisponible
	}
	var acreditado bool
	if err := pool.QueryRow(ctx, acreditarSQL).Scan(&acreditado); err != nil || !acreditado {
		return nil, ports.ErrLecturaUsuariosAdministrablesNoDisponible
	}
	return &Fuente{pool: pool, emisor: emisor, fuente: fuente, intentos: intentos, reloj: reloj, config: config, ambito: a}, nil
}

// acreditarSQL sólo lee el catálogo. El lector no tiene ni debe tener USAGE
// sobre vec_autorizacion_atestada_v3, y to_regprocedure sobre un esquema sin
// USAGE falla con 42501 en vez de devolver NULL; por eso la función atestada
// se localiza en pg_proc por esquema, nombre y tipos (LATERAL atestada).
const acreditarSQL = `SELECT COALESCE(
 current_user=session_user AND current_setting('role')='none' AND l.rolcanlogin AND l.rolinherit
 AND NOT(l.rolsuper OR l.rolcreatedb OR l.rolcreaterole OR l.rolreplication OR l.rolbypassrls) AND l.rolconfig IS NULL
 AND g.rolname='vec_admin_usuarios_lector' AND NOT(g.rolcanlogin OR g.rolinherit OR g.rolsuper OR g.rolcreatedb OR g.rolcreaterole OR g.rolreplication OR g.rolbypassrls) AND g.rolconfig IS NULL
 AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option
 AND (SELECT count(*) FROM pg_catalog.pg_auth_members x WHERE x.member=l.oid)=1
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members x WHERE x.member=g.oid)
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_db_role_setting x WHERE x.setrole IN(l.oid,g.oid))
 AND pg_catalog.has_database_privilege(current_user,current_database(),'CONNECT')
 AND NOT pg_catalog.has_database_privilege(current_user,current_database(),'CREATE')
 AND NOT pg_catalog.has_database_privilege(current_user,current_database(),'TEMPORARY')
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_database d CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(d.datacl,pg_catalog.acldefault('d',d.datdba))) a
  WHERE d.datname=current_database() AND a.grantee IN(l.oid,g.oid) AND (a.privilege_type<>'CONNECT' OR a.is_grantable))
 AND EXISTS(SELECT 1 FROM pg_catalog.pg_namespace n CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(n.nspacl,pg_catalog.acldefault('n',n.nspowner))) a
  WHERE n.nspname='vec_autorizacion' AND n.nspowner=pg_catalog.to_regrole('vec_autorizacion_propietario')
   AND a.grantee=g.oid AND a.privilege_type='USAGE' AND NOT a.is_grantable)
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_namespace n CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(n.nspacl,pg_catalog.acldefault('n',n.nspowner))) a
  WHERE n.nspname IN('vec_autorizacion','vec_autorizacion_atestada_v3','vec_contexto_actor_v1')
   AND (a.grantee=l.oid OR (n.nspname<>'vec_autorizacion' AND a.grantee=g.oid)))
 AND pg_catalog.has_schema_privilege(current_user,'vec_autorizacion','USAGE')
 AND NOT pg_catalog.has_schema_privilege(current_user,'vec_autorizacion','CREATE')
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
  CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
  WHERE n.nspname IN('vec_autorizacion','vec_autorizacion_atestada_v3','vec_contexto_actor_v1')
   AND (a.grantee=l.oid OR a.grantee=g.oid AND (n.nspname<>'vec_autorizacion' OR p.oid NOT IN(
    pg_catalog.to_regprocedure('vec_autorizacion.listar_usuarios_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),
    pg_catalog.to_regprocedure('vec_autorizacion.consultar_usuario_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')))))
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
  WHERE n.nspname IN('vec_autorizacion','vec_autorizacion_atestada_v3','vec_contexto_actor_v1')
   AND c.relkind IN('r','p','v','m','f','S')
   AND (c.relowner IS DISTINCT FROM CASE n.nspname
    WHEN 'vec_autorizacion' THEN pg_catalog.to_regrole('vec_autorizacion_propietario')
    WHEN 'vec_autorizacion_atestada_v3' THEN pg_catalog.to_regrole('vec_autorizacion_atestada_v3_propietario')
    ELSE pg_catalog.to_regrole('vec_contexto_actor_v1_propietario') END
    OR EXISTS(SELECT 1 FROM pg_catalog.aclexplode(COALESCE(c.relacl,pg_catalog.acldefault(
      CASE WHEN c.relkind='S' THEN 'S'::"char" ELSE 'r'::"char" END,c.relowner))) a
      WHERE a.grantee IN(l.oid,g.oid,0::oid))
    OR EXISTS(SELECT 1 FROM pg_catalog.pg_attribute at CROSS JOIN LATERAL pg_catalog.aclexplode(at.attacl) a
      WHERE at.attrelid=c.oid AND at.attnum>0 AND NOT at.attisdropped AND at.attacl IS NOT NULL
       AND a.grantee IN(l.oid,g.oid,0::oid))))
 AND pg_catalog.to_regprocedure('vec_autorizacion.listar_usuarios_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 AND pg_catalog.to_regprocedure('vec_autorizacion.consultar_usuario_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 AND atestada.oid IS NOT NULL
 AND pg_catalog.has_function_privilege(current_user,pg_catalog.to_regprocedure('vec_autorizacion.listar_usuarios_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),'EXECUTE')
 AND pg_catalog.has_function_privilege(current_user,pg_catalog.to_regprocedure('vec_autorizacion.consultar_usuario_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),'EXECUTE')
 AND NOT pg_catalog.has_function_privilege(current_user,atestada.oid,'EXECUTE')
 AND (SELECT count(*)=2 FROM pg_catalog.pg_proc p WHERE p.oid IN (
  pg_catalog.to_regprocedure('vec_autorizacion.listar_usuarios_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),
  pg_catalog.to_regprocedure('vec_autorizacion.consultar_usuario_admin_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'))
  AND p.proowner=pg_catalog.to_regrole('vec_autorizacion_propietario') AND p.prosecdef
  AND EXISTS(SELECT 1 FROM pg_catalog.aclexplode(COALESCE(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
   WHERE a.grantee=g.oid AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
  AND NOT EXISTS(SELECT 1 FROM pg_catalog.aclexplode(COALESCE(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
   WHERE a.grantee NOT IN(p.proowner,g.oid) OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
 AND EXISTS(SELECT 1 FROM pg_catalog.pg_proc p WHERE p.oid=atestada.oid
  AND p.proowner=pg_catalog.to_regrole('vec_autorizacion_atestada_v3_propietario') AND p.prosecdef
  AND NOT EXISTS(SELECT 1 FROM pg_catalog.aclexplode(COALESCE(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
   WHERE a.grantee NOT IN(p.proowner,pg_catalog.to_regrole('vec_autorizacion_propietario')) OR a.privilege_type<>'EXECUTE' OR a.is_grantable)),false)
 FROM pg_catalog.pg_roles l JOIN pg_catalog.pg_auth_members m ON m.member=l.oid JOIN pg_catalog.pg_roles g ON g.oid=m.roleid
 LEFT JOIN LATERAL (SELECT p.oid FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
  WHERE n.nspname='vec_autorizacion_atestada_v3' AND p.proname='registrar_y_consumir_usuarios_admin_v3_atestada'
   AND pg_catalog.oidvectortypes(p.proargtypes)='text, bytea, bytea, bytea, bytea, numeric, numeric, bytea, bytea, bytea, bytea') atestada ON true
 WHERE l.rolname=session_user`

func ausente(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return r.IsNil()
	}
	return false
}

func (f *Fuente) ListarUsuarios(ctx context.Context, actor domain.ContextoActor, evidencia domain.EvidenciaSesionAdministracionPerfiles, filtros ports.FiltrosUsuariosAdministrables) (ports.PaginaUsuariosAdministrables, error) {
	if f == nil {
		return ports.PaginaUsuariosAdministrables{}, ports.ErrLecturaUsuariosAdministrablesNoDisponible
	}
	p, err := materialListar(f.ambito, filtros)
	if err != nil {
		return ports.PaginaUsuariosAdministrables{}, f.rechazarListaInvalida(ctx, actor, evidencia, err)
	}
	bruto, err := f.ejecutar(ctx, actor, evidencia, &p, listarSQL)
	if err != nil {
		return ports.PaginaUsuariosAdministrables{}, err
	}
	return listaRespuesta(bruto, f.ambito, filtros, p.decisionRef, p.recurso.Referencia, p.contextoSHA)
}

func (f *Fuente) ConsultarUsuario(ctx context.Context, actor domain.ContextoActor, evidencia domain.EvidenciaSesionAdministracionPerfiles, persona string) (*ports.UsuarioAdministrable, error) {
	if f == nil {
		return nil, ports.ErrLecturaUsuariosAdministrablesNoDisponible
	}
	p, err := materialConsultar(f.ambito, persona)
	if err != nil {
		return nil, err
	}
	bruto, err := f.ejecutar(ctx, actor, evidencia, &p, consultarSQL)
	if err != nil {
		return nil, err
	}
	return fichaRespuesta(bruto, f.ambito, persona, p.decisionRef, p.recurso.Referencia, p.contextoSHA)
}

func (f *Fuente) ejecutar(ctx context.Context, actor domain.ContextoActor, evidencia domain.EvidenciaSesionAdministracionPerfiles, p *peticion, consulta string) ([]byte, error) {
	if ctx == nil {
		return nil, ports.ErrLecturaUsuariosAdministrablesNoDisponible
	}
	actorOriginal, evidenciaOriginal, err := clonarIdentidadLectura(actor, evidencia)
	if err != nil {
		return nil, err
	}
	correlacion, err := ports.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
	if err != nil {
		return nil, ports.ErrLecturaUsuariosAdministrablesNoDisponible
	}
	ref, err := correlacion.ValorCanonico()
	if err != nil {
		return nil, ports.ErrLecturaUsuariosAdministrablesNoDisponible
	}
	p.correlacion = ref
	bruto, err := f.consumir(ctx, actorOriginal, evidenciaOriginal, p, consulta, correlacion)
	if err == nil {
		return bruto, nil
	}
	if evidenciaOriginal.ValidarPara(actorOriginal) != nil {
		return nil, ports.ErrLecturaUsuariosAdministrablesNoDisponible
	}
	if f.registrarFallo(ctx, evidenciaOriginal, p.accion, p.recurso.Referencia, p.correlacion, err) != nil {
		return nil, ports.ErrLecturaUsuariosAdministrablesNoDisponible
	}
	return nil, err
}

// Un cursor o filtro incompatible no llega a la fachada. La referencia de
// auditoría es el conjunto real del ámbito privado; no se guarda el valor
// recibido ni se construye una decisión para justificar su rechazo.
func (f *Fuente) rechazarListaInvalida(ctx context.Context, actor domain.ContextoActor, evidencia domain.EvidenciaSesionAdministracionPerfiles, causa error) error {
	if ctx == nil || actor.Validar() != nil || evidencia.ValidarPara(actor) != nil {
		return causa
	}
	vinculo, err := evidencia.Vinculo.Datos()
	if err != nil {
		return falloValidacionRedactado(err)
	}
	if string(vinculo.Superficie) != f.config.Canal || !vinculo.CuentaPrivilegiada {
		return causa
	}
	correlacion, err := ports.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
	if err != nil {
		return ports.ErrLecturaUsuariosAdministrablesNoDisponible
	}
	ref, err := correlacion.ValorCanonico()
	if err != nil {
		return ports.ErrLecturaUsuariosAdministrablesNoDisponible
	}
	conjunto, err := conjuntoUsuarios(f.ambito)
	if err != nil {
		return ports.ErrLecturaUsuariosAdministrablesNoDisponible
	}
	if f.registrarFallo(ctx, evidencia, accionListar, conjunto, ref, domain.ErrAutorizacionDenegada) != nil {
		return ports.ErrLecturaUsuariosAdministrablesNoDisponible
	}
	return causa
}

func (f *Fuente) consumir(ctx context.Context, actor domain.ContextoActor, evidencia domain.EvidenciaSesionAdministracionPerfiles, p *peticion, consulta string, correlacion domain.ReferenciaCorrelacionAutorizacionV2) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ahora := f.reloj.Ahora()
	if actor.Validar() != nil || evidencia.ValidarEn(actor, ahora) != nil || !actor.Instantanea.VigenteEn(ahora) {
		return nil, domain.ErrAutorizacionDenegada
	}
	vinculo, err := evidencia.Vinculo.Datos()
	if err != nil || string(vinculo.Superficie) != f.config.Canal {
		return nil, domain.ErrAutorizacionDenegada
	}
	snapshot, err := f.fuente.ObtenerInstantaneaAutorizacion(ctx, actor.PersonaRef, actor.PerfilActivoRef)
	if err != nil {
		return nil, clasificarDependencia(err)
	}
	if snapshot.Validar() != nil || snapshot.AsignacionPerfil.PrincipalID != actor.PersonaRef || snapshot.AsignacionPerfil.PerfilActivoRef != actor.PerfilActivoRef ||
		!snapshot.AsignacionPerfil.VigenteEn(ahora) || !snapshot.AsignacionPerfil.Cubre(p.recurso) ||
		!versionRolUsuariosAdmitida(snapshot.VersionRol.Referencia()) || snapshot.VersionRol.Estado != domain.EstadoVersionRolPublicada ||
		snapshot.ControlVigenciaVersionRol.Estado != domain.EstadoControlVigenciaVersionRolHabilitada {
		return nil, domain.ErrAutorizacionDenegada
	}
	recursoEmisor := p.recurso
	recursoEmisor.Ambitos = maps.Clone(p.recurso.Ambitos)
	recursoEmisor.Atributos = maps.Clone(p.recurso.Atributos)
	emision := ports.EmisionUsuariosAdministrables{Material: append([]byte(nil), p.material...), Recurso: recursoEmisor, Accion: p.accion, Audiencia: p.audiencia, Correlacion: correlacion}
	actorEmisor, evidenciaEmisor, err := clonarIdentidadLectura(actor, evidencia)
	if err != nil {
		clear(emision.Material)
		return nil, err
	}
	snapshotEmisor, err := clonarInstantaneaLectura(snapshot)
	if err != nil {
		clear(emision.Material)
		return nil, err
	}
	m, err := f.emisor.EmitirLecturaUsuariosAdministrables(ctx, actorEmisor, evidenciaEmisor, snapshotEmisor, emision)
	clear(emision.Material)
	if err != nil {
		return nil, clasificarDependencia(err)
	}
	ahora = f.reloj.Ahora()
	if evidencia.ValidarEn(actor, ahora) != nil || !actor.Instantanea.VigenteEn(ahora) {
		return nil, domain.ErrAutorizacionDenegada
	}
	decision, huella, err := validarExportacion(m, actor, evidencia, p, ahora)
	if err != nil {
		return nil, err
	}
	p.decisionRef, p.contextoSHA = decision, huella
	args := []any{string(p.material), m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(), strconv.FormatUint(m.PersonaVersion(), 10), strconv.FormatUint(m.PerfilVersion(), 10), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI()}
	defer func() {
		for _, a := range args {
			if b, ok := a.([]byte); ok {
				clear(b)
			}
		}
	}()
	return consultarTransaccion(ctx, f.pool, args, p, consulta)
}

func consultarTransaccion(ctx context.Context, pool conexion, args []any, p *peticion, consulta string) ([]byte, error) {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		// El lector no ha ejecutado SQL nominal; tampoco hay denegación del gate.
		return nil, ports.ErrLecturaUsuariosAdministrablesNoDisponible
	}
	defer func() {
		c, cancel := context.WithTimeout(context.Background(), plazoarranque.Ampliar(2*time.Second))
		defer cancel()
		_ = tx.Rollback(c)
	}()
	var bruto []byte
	if err := tx.QueryRow(ctx, consulta, args...).Scan(&bruto); err != nil {
		return nil, traducir(err)
	}
	if err := validarRespuesta(bruto, p, consulta); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, ports.ErrLecturaUsuariosAdministrablesNoDisponible
	}
	return bruto, nil
}

func traducir(err error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "42501" {
		return domain.ErrAutorizacionDenegada
	}
	return ports.ErrLecturaUsuariosAdministrablesNoDisponible
}

func clasificarDependencia(err error) error {
	if errors.Is(err, domain.ErrAutorizacionDenegada) {
		return domain.ErrAutorizacionDenegada
	}
	return ports.ErrLecturaUsuariosAdministrablesNoDisponible
}
