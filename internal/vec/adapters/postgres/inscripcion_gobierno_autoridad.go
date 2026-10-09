package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// EmisorGobiernoInscripcionV3 usa el PDP y la atestación centrales. Sólo
// acepta el efecto construido por el servidor; el cuerpo HTTP no transporta
// una decisión o una capacidad V3.
type EmisorGobiernoInscripcionV3 interface {
	EmitirGobiernoInscripcion(context.Context, domain.ContextoActor,
		domain.EvidenciaSesionAdministracionPerfiles, domain.InstantaneaAutorizacion,
		EfectoGobiernoInscripcion) (ports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

type poolGobiernoInscripcion interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type AutoridadGobiernoInscripcionPostgreSQL struct {
	pool   poolGobiernoInscripcion
	fuente ports.FuenteCatalogoAccionesAdministracionV1
	emisor EmisorGobiernoInscripcionV3
	reloj  ports.Reloj
}

var _ ports.AutoridadVersionInscripcion = (*AutoridadGobiernoInscripcionPostgreSQL)(nil)

const preflightGobiernoInscripcionSQL = `SELECT current_user=session_user AND r.rolcanlogin AND r.rolinherit
 AND NOT (r.rolsuper OR r.rolcreatedb OR r.rolcreaterole OR r.rolreplication OR r.rolbypassrls)
 AND r.rolconfig IS NULL
 AND (SELECT count(*) FROM pg_catalog.pg_auth_members m WHERE m.member=r.oid)=1
 AND EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.member=r.oid
   AND m.roleid=pg_catalog.to_regrole('vec_admin_version_inscripcion_ejecutor')
   AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
 AND NOT pg_catalog.has_database_privilege(current_user,current_database(),'CREATE,TEMP')
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_namespace n WHERE n.nspname LIKE 'vec\_%' ESCAPE '\'
   AND pg_catalog.has_schema_privilege(current_user,n.oid,'CREATE'))
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
   WHERE n.nspname LIKE 'vec\_%' ESCAPE '\' AND c.relkind IN('r','p','v','m','f')
   AND pg_catalog.has_table_privilege(current_user,c.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER'))
 AND pg_catalog.to_regprocedure('vec_autorizacion.proponer_version_inscripcion_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 AND pg_catalog.to_regprocedure('vec_autorizacion.cerrar_version_inscripcion_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 AND pg_catalog.has_function_privilege(current_user,
   pg_catalog.to_regprocedure('vec_autorizacion.proponer_version_inscripcion_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),'EXECUTE')
 AND pg_catalog.has_function_privilege(current_user,
   pg_catalog.to_regprocedure('vec_autorizacion.cerrar_version_inscripcion_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),'EXECUTE')
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
   WHERE n.nspname LIKE 'vec\_%' ESCAPE '\' AND pg_catalog.has_function_privilege(current_user,p.oid,'EXECUTE')
   AND p.oid NOT IN (pg_catalog.to_regprocedure('vec_autorizacion.proponer_version_inscripcion_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),
    pg_catalog.to_regprocedure('vec_autorizacion.cerrar_version_inscripcion_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')))
 FROM pg_catalog.pg_roles r WHERE r.rolname=session_user`

const propuestaGobiernoInscripcionSQL = `SELECT vec_autorizacion.proponer_version_inscripcion_v1($1::text,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::numeric,$7::numeric,$8::bytea,$9::bytea,$10::bytea,$11::bytea)`
const cierreGobiernoInscripcionSQL = `SELECT vec_autorizacion.cerrar_version_inscripcion_v1($1::text,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::numeric,$7::numeric,$8::bytea,$9::bytea,$10::bytea,$11::bytea)`

func NuevaAutoridadGobiernoInscripcionPostgreSQL(ctx context.Context, pool *pgxpool.Pool,
	fuente ports.FuenteCatalogoAccionesAdministracionV1, emisor EmisorGobiernoInscripcionV3,
	reloj ports.Reloj) (*AutoridadGobiernoInscripcionPostgreSQL, error) {
	return nuevaAutoridadGobiernoInscripcionPostgreSQL(ctx, pool, fuente, emisor, reloj)
}

func nuevaAutoridadGobiernoInscripcionPostgreSQL(ctx context.Context, pool poolGobiernoInscripcion,
	fuente ports.FuenteCatalogoAccionesAdministracionV1, emisor EmisorGobiernoInscripcionV3,
	reloj ports.Reloj) (*AutoridadGobiernoInscripcionPostgreSQL, error) {
	if ctx == nil || interfazGobiernoInscripcionPostgreSQLNula(pool) ||
		interfazGobiernoInscripcionPostgreSQLNula(fuente) ||
		interfazGobiernoInscripcionPostgreSQLNula(emisor) ||
		interfazGobiernoInscripcionPostgreSQLNula(reloj) {
		return nil, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var acreditado bool
	if err := pool.QueryRow(ctx, preflightGobiernoInscripcionSQL).Scan(&acreditado); err != nil || !acreditado {
		return nil, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	return &AutoridadGobiernoInscripcionPostgreSQL{pool: pool, fuente: fuente, emisor: emisor, reloj: reloj}, nil
}

func interfazGobiernoInscripcionPostgreSQLNula(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return r.IsNil()
	default:
		return false
	}
}

func (a *AutoridadGobiernoInscripcionPostgreSQL) ResolverCatalogoVersionInscripcion(ctx context.Context,
	p domain.PlanVersionInscripcion) (domain.CatalogoAccionesAdministracionV1, error) {
	var vacio domain.CatalogoAccionesAdministracionV1
	if a == nil || ctx == nil || interfazGobiernoInscripcionPostgreSQLNula(a.fuente) ||
		interfazGobiernoInscripcionPostgreSQLNula(a.reloj) || p.ValidarEstructura() != nil {
		return vacio, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	c, err := a.fuente.ObtenerCatalogoAccionesAdministracionV1(ctx, p.CatalogoRef, p.CatalogoVersion, p.CatalogoHuellaSHA256)
	if err != nil || p.ValidarFuenteHistorica(c, a.reloj.Ahora()) != nil {
		return vacio, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	return c, nil
}

func (a *AutoridadGobiernoInscripcionPostgreSQL) ProponerVersionInscripcion(ctx context.Context,
	o domain.OrdenPropuestaVersionInscripcion) (domain.PropuestaVersionInscripcion, bool, error) {
	var vacia domain.PropuestaVersionInscripcion
	if a == nil || o.Validar() != nil {
		return vacia, false, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	e, err := MaterialPropuestaGobiernoInscripcion(o)
	if err != nil {
		return vacia, false, err
	}
	var propuesta domain.PropuestaVersionInscripcion
	var replay bool
	err = a.ejecutar(ctx, o.Solicitud.Actor, o.Solicitud.Evidencia,
		o.Solicitud.InstantaneaAutorizacion, e, propuestaGobiernoInscripcionSQL, func(b []byte) error {
			var r struct {
				Estado             string    `json:"estado"`
				Replay             bool      `json:"replay"`
				MaterialCanon      string    `json:"material_canon"`
				HuellaSHA256       string    `json:"huella_sha256"`
				CaducaEn           time.Time `json:"caduca_en"`
				AuditoriaAccesoRef string    `json:"auditoria_acceso_ref"`
			}
			if json.Unmarshal(b, &r) != nil || r.Estado != "permitido" {
				return ports.ErrAutoridadAdministracionPerfilesNoDisponible
			}
			var m domain.MaterialPropuestaVersionInscripcion
			if decodificarGobiernoInscripcion([]byte(r.MaterialCanon), &m) != nil {
				return ports.ErrAutoridadAdministracionPerfilesNoDisponible
			}
			p := domain.PropuestaVersionInscripcion{Material: m, HuellaSHA256: r.HuellaSHA256,
				CaducaEn: r.CaducaEn, AuditoriaAccesoRef: r.AuditoriaAccesoRef}
			if p.ValidarPara(o) != nil {
				return ports.ErrAutoridadAdministracionPerfilesNoDisponible
			}
			propuesta, replay = p, r.Replay
			return nil
		})
	if err != nil {
		return vacia, false, err
	}
	return propuesta, replay, nil
}

func (a *AutoridadGobiernoInscripcionPostgreSQL) CerrarVersionInscripcion(ctx context.Context,
	s domain.SolicitudCierreVersionInscripcion) (domain.CierreVersionInscripcion, bool, error) {
	var vacio domain.CierreVersionInscripcion
	if a == nil || s.Validar() != nil {
		return vacio, false, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	e, err := MaterialCierreGobiernoInscripcion(s)
	if err != nil {
		return vacio, false, err
	}
	var cierre domain.CierreVersionInscripcion
	var replay bool
	err = a.ejecutar(ctx, s.Aprobador, s.Evidencia, s.InstantaneaAutorizacion,
		e, cierreGobiernoInscripcionSQL, func(b []byte) error {
			var r struct {
				Estado                string                                         `json:"estado"`
				Replay                bool                                           `json:"replay"`
				OperacionRef          string                                         `json:"operacion_ref"`
				MaterialCanon         string                                         `json:"material_canon"`
				PropuestaHuellaSHA256 string                                         `json:"propuesta_huella_sha256"`
				Decision              domain.DecisionPropuestaAdministracionPerfiles `json:"decision"`
				ConfirmadoEn          time.Time                                      `json:"confirmado_en"`
				AuditoriaAccesoRef    string                                         `json:"auditoria_acceso_ref"`
				Recibo                domain.ReciboVersionInscripcion                `json:"recibo"`
			}
			if json.Unmarshal(b, &r) != nil || r.Estado != "permitido" || r.OperacionRef != s.OperacionRef {
				return ports.ErrAutoridadAdministracionPerfilesNoDisponible
			}
			var m domain.MaterialPropuestaVersionInscripcion
			if decodificarGobiernoInscripcion([]byte(r.MaterialCanon), &m) != nil {
				return ports.ErrAutoridadAdministracionPerfilesNoDisponible
			}
			c := domain.CierreVersionInscripcion{OperacionRef: r.OperacionRef, Material: m,
				PropuestaHuellaSHA256: r.PropuestaHuellaSHA256, Decision: r.Decision,
				ConfirmadoEn: r.ConfirmadoEn, AuditoriaAccesoRef: r.AuditoriaAccesoRef, Recibo: &r.Recibo}
			if c.ValidarPara(s) != nil {
				return ports.ErrAutoridadAdministracionPerfilesNoDisponible
			}
			cierre, replay = c, r.Replay
			return nil
		})
	if err != nil {
		return vacio, false, err
	}
	return cierre, replay, nil
}

func (a *AutoridadGobiernoInscripcionPostgreSQL) ejecutar(ctx context.Context,
	actor domain.ContextoActor, evidencia domain.EvidenciaSesionAdministracionPerfiles,
	instantanea domain.InstantaneaAutorizacion, e EfectoGobiernoInscripcion, consulta string, validar func([]byte) error) error {
	errNoDisponible := ports.ErrAutoridadAdministracionPerfilesNoDisponible
	if a == nil || ctx == nil || ctx.Err() != nil || interfazGobiernoInscripcionPostgreSQLNula(a.pool) ||
		interfazGobiernoInscripcionPostgreSQLNula(a.emisor) || interfazGobiernoInscripcionPostgreSQLNula(a.reloj) ||
		validar == nil || evidencia.ValidarEn(actor, a.reloj.Ahora()) != nil {
		return errNoDisponible
	}
	recurso, err := RecursoGobiernoInscripcion(e, instantanea.AsignacionPerfil)
	if err != nil {
		return errNoDisponible
	}
	material, err := a.emisor.EmitirGobiernoInscripcion(ctx, actor, evidencia, instantanea, e)
	if err != nil {
		if errors.Is(err, domain.ErrAutorizacionDenegada) {
			return err
		}
		return errNoDisponible
	}
	h, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		return errNoDisponible
	}
	r := material.ResumenCapacidad()
	ahora := a.reloj.Ahora()
	if material.ValidarEstructura() != nil || r.Operacion() != e.Accion || r.AudienciaConsumo() != e.Audiencia ||
		r.EfectoRef() != recurso.Referencia || r.EfectoHuellaSHA256() != h ||
		r.ContextoRef() != evidencia.ResultadoContexto.RegistroContextoRef ||
		r.ContextoHuellaSHA256() != evidencia.ResultadoContexto.HuellaSHA256 ||
		!bytes.Equal(material.ContextoActorCanonico(), evidencia.ResultadoContexto.RepresentacionCanonica) ||
		material.PersonaVersion() != actor.Instantanea.PersonaVersion ||
		material.PerfilVersion() != actor.Instantanea.PerfilVersion ||
		ahora.Before(r.EmitidaEn()) || !ahora.Before(r.ExpiraEn()) {
		return errNoDisponible
	}
	args := []any{string(e.Material), material.CapacidadCanonica(), material.DecisionCanonica(),
		material.MotivoCanonico(), material.ContextoActorCanonico(),
		strconv.FormatUint(material.PersonaVersion(), 10), strconv.FormatUint(material.PerfilVersion(), 10),
		material.PayloadVECAD3(), material.SobreCOSESign1(), material.EvidenciaVerificacion(), material.RaizPublicaSPKI()}
	defer func() {
		for _, arg := range args {
			if b, ok := arg.([]byte); ok {
				clear(b)
			}
		}
	}()
	tx, err := a.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil || tx == nil {
		return errNoDisponible
	}
	defer func() {
		rollbackCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = tx.Rollback(rollbackCtx)
	}()
	for _, configuracion := range []string{
		"SET LOCAL search_path = pg_catalog, pg_temp",
		"SET LOCAL row_security = on",
		"SET LOCAL TimeZone = 'UTC'",
		"SET LOCAL lock_timeout = '2s'",
		"SET LOCAL statement_timeout = '15s'",
		"SET LOCAL idle_in_transaction_session_timeout = '20s'",
	} {
		if _, err := tx.Exec(ctx, configuracion); err != nil {
			return errNoDisponible
		}
	}
	var b []byte
	if err := tx.QueryRow(ctx, consulta, args...).Scan(&b); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return errNoDisponible
	}
	defer clear(b)
	var intento struct {
		Estado           string `json:"estado"`
		Codigo           string `json:"codigo"`
		AuditoriaIntento struct {
			AuditoriaRef string `json:"auditoria_ref"`
		} `json:"auditoria_intento"`
	}
	if json.Unmarshal(b, &intento) == nil && (intento.Estado == "denegado" || intento.Estado == "error") {
		if intento.AuditoriaIntento.AuditoriaRef == "" ||
			(intento.Estado == "denegado" && intento.Codigo != "version_inscripcion_denegado") ||
			(intento.Estado == "error" && intento.Codigo != "version_inscripcion_error") {
			return errNoDisponible
		}
		if err := tx.Commit(ctx); err != nil {
			return errNoDisponible
		}
		clear(b)
		if intento.Estado == "denegado" {
			return errors.Join(ports.ErrGobiernoInscripcionIntentoAuditado, domain.ErrAutorizacionDenegada)
		}
		return errors.Join(ports.ErrGobiernoInscripcionIntentoAuditado, errNoDisponible)
	}
	if err := validar(b); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return errNoDisponible
	}
	return nil
}
