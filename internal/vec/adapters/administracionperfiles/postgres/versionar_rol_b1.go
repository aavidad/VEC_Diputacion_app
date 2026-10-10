package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/shared/plazoarranque"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// EmisorVersionarRolBolsa emplea la autoridad PDP/V3 ADMIN con las audiencias
// propias de B1. El material de la petición nunca contiene capacidad V3.
type EmisorVersionarRolBolsa interface {
	EmitirVersionarRolBolsa(context.Context, domain.ContextoActor,
		domain.EvidenciaSesionAdministracionPerfiles, domain.InstantaneaAutorizacion,
		Efecto) (ports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

type AutoridadVersionarRolBolsa struct {
	pool     conexion
	catalogo ports.FuenteCatalogoAccionesAdministracionV1
	emisor   EmisorVersionarRolBolsa
	reloj    ports.Reloj
}

var _ ports.AutoridadVersionarRolBolsa = (*AutoridadVersionarRolBolsa)(nil)

const (
	proponerVersionarRolBolsaSQL = `SELECT vec_autorizacion.proponer_version_rol_bolsa_v1` + argumentosV3
	cerrarVersionarRolBolsaSQL   = `SELECT vec_autorizacion.cerrar_version_rol_bolsa_v1` + argumentosV3
)

const acreditarVersionarRolBolsaSQL = `SELECT current_user=session_user AND r.rolcanlogin AND r.rolinherit
 AND NOT (r.rolsuper OR r.rolcreatedb OR r.rolcreaterole OR r.rolreplication OR r.rolbypassrls)
 AND r.rolconfig IS NULL AND (SELECT count(*) FROM pg_catalog.pg_auth_members m WHERE m.member=r.oid)=1
 AND EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.member=r.oid
   AND m.roleid=pg_catalog.to_regrole('vec_admin_version_rol_bolsa_ejecutor')
   AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
 AND NOT pg_catalog.has_database_privilege(current_user,current_database(),'CREATE,TEMP')
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_namespace n WHERE n.nspname LIKE 'vec\_%' ESCAPE '\'
   AND pg_catalog.has_schema_privilege(current_user,n.oid,'CREATE'))
 AND pg_catalog.to_regprocedure('vec_autorizacion.proponer_version_rol_bolsa_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 AND pg_catalog.to_regprocedure('vec_autorizacion.cerrar_version_rol_bolsa_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 AND pg_catalog.to_regprocedure('vec_autorizacion.resolver_rol_administrable_v1(text)') IS NOT NULL
 AND pg_catalog.has_function_privilege(current_user,
   pg_catalog.to_regprocedure('vec_autorizacion.proponer_version_rol_bolsa_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),'EXECUTE')
 AND pg_catalog.has_function_privilege(current_user,
   pg_catalog.to_regprocedure('vec_autorizacion.cerrar_version_rol_bolsa_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),'EXECUTE')
 AND pg_catalog.has_function_privilege(current_user,
   pg_catalog.to_regprocedure('vec_autorizacion.resolver_rol_administrable_v1(text)'),'EXECUTE')
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
   WHERE n.nspname LIKE 'vec\_%' ESCAPE '\' AND pg_catalog.has_function_privilege(current_user,p.oid,'EXECUTE')
   AND p.oid NOT IN (pg_catalog.to_regprocedure('vec_autorizacion.proponer_version_rol_bolsa_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),
    pg_catalog.to_regprocedure('vec_autorizacion.cerrar_version_rol_bolsa_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),
    pg_catalog.to_regprocedure('vec_autorizacion.resolver_rol_administrable_v1(text)')))
 FROM pg_catalog.pg_roles r WHERE r.rolname=session_user`

func NuevaAutoridadVersionarRolBolsa(ctx context.Context, pool *pgxpool.Pool,
	catalogo ports.FuenteCatalogoAccionesAdministracionV1,
	emisor EmisorVersionarRolBolsa,
	reloj ports.Reloj) (*AutoridadVersionarRolBolsa, error) {
	return nuevaAutoridadVersionarRolBolsa(ctx, pool, catalogo, emisor, reloj)
}

func nuevaAutoridadVersionarRolBolsa(ctx context.Context, pool conexion,
	catalogo ports.FuenteCatalogoAccionesAdministracionV1,
	emisor EmisorVersionarRolBolsa,
	reloj ports.Reloj) (*AutoridadVersionarRolBolsa, error) {
	if ctx == nil || ausente(pool) || ausente(catalogo) || ausente(emisor) || ausente(reloj) {
		return nil, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var acreditado bool
	if err := pool.QueryRow(ctx, acreditarVersionarRolBolsaSQL).Scan(&acreditado); err != nil || !acreditado {
		return nil, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	return &AutoridadVersionarRolBolsa{pool: pool, catalogo: catalogo, emisor: emisor, reloj: reloj}, nil
}

func (a *AutoridadVersionarRolBolsa) disponible(ctx context.Context) error {
	if ctx == nil || a == nil || ausente(a.pool) || ausente(a.catalogo) ||
		ausente(a.emisor) || ausente(a.reloj) {
		return ports.ConClaseVersionBolsa("autoridad_no_disponible", ports.ErrAutoridadAdministracionPerfilesNoDisponible)
	}
	return ctx.Err()
}

// ResolverCatalogoVersionarRolBolsa sólo resuelve la fuente de concesiones.
// Los documentos del plan son expectativas CAS y AUT63 los coteja en SQL.
func (a *AutoridadVersionarRolBolsa) ResolverCatalogoVersionarRolBolsa(ctx context.Context,
	s domain.SolicitudPropuestaVersionarRolBolsa) (domain.CatalogoAccionesAdministracionV1, error) {
	var vacio domain.CatalogoAccionesAdministracionV1
	if err := a.disponible(ctx); err != nil {
		return vacio, err
	}
	if s.Validar() != nil {
		return vacio, domain.ErrVersionarRolBolsaInvalido
	}
	c, err := a.catalogo.ObtenerCatalogoAccionesAdministracionV1(ctx, s.Intencion.CatalogoRef,
		s.Intencion.CatalogoVersion, s.Intencion.CatalogoHuellaSHA256)
	if err != nil {
		return vacio, ports.ConClaseVersionBolsa("catalogo_consulta", err)
	}
	plan, err := domain.PrepararPlanVersionarRolBolsa(c, s.Intencion, a.reloj.Ahora())
	h, e := plan.HuellaSHA256()
	if err != nil || e != nil || h != s.HuellaPlanEsperada {
		return vacio, ports.ConClaseVersionBolsa("catalogo_plan_huella", domain.ErrVersionarRolBolsaInvalido)
	}
	return c, nil
}

func (a *AutoridadVersionarRolBolsa) ProponerVersionarRolBolsa(ctx context.Context,
	o domain.OrdenPropuestaVersionarRolBolsa) (ports.ResultadoPropuestaVersionarRolBolsa, error) {
	var vacio ports.ResultadoPropuestaVersionarRolBolsa
	if err := a.disponible(ctx); err != nil {
		return vacio, err
	}
	e, err := materialPropuestaVersionarRolBolsa(o)
	if err != nil {
		return vacio, ports.ConClaseVersionBolsa("material_propuesta", err)
	}
	var r respuestaPropuestaVersionarRolBolsa
	err = a.ejecutar(ctx, o.Solicitud.Actor, o.Solicitud.Evidencia,
		o.Solicitud.InstantaneaAutorizacion, e, proponerVersionarRolBolsaSQL, func(b []byte) error {
			if decodificarGobiernoRol(b, &r) != nil || r.Estado != "permitido" || r.HuellaSHA256 == "" {
				return ports.ConClaseVersionBolsa("sql_respuesta_invalida", ports.ErrAutoridadAdministracionPerfilesNoDisponible)
			}
			var m domain.MaterialPropuestaVersionarRolBolsa
			if decodificarGobiernoRol([]byte(r.MaterialCanon), &m) != nil {
				return ports.ConClaseVersionBolsa("sql_respuesta_invalida", ports.ErrAutoridadAdministracionPerfilesNoDisponible)
			}
			r.Propuesta = domain.PropuestaVersionarRolBolsa{Material: m, HuellaSHA256: r.HuellaSHA256,
				CaducaEn: r.CaducaEn}
			if (ports.ResultadoPropuestaVersionarRolBolsa{Propuesta: r.Propuesta,
				Replay: r.Replay, AuditoriaAccesoRef: r.AuditoriaAccesoRef}).ValidarPara(o, a.reloj.Ahora()) != nil {
				return ports.ConClaseVersionBolsa("sql_respuesta_invalida", ports.ErrAutoridadAdministracionPerfilesNoDisponible)
			}
			return nil
		})
	if err != nil {
		return vacio, err
	}
	return ports.ResultadoPropuestaVersionarRolBolsa{Propuesta: domain.PropuestaVersionarRolBolsa{
		Material: r.Propuesta.Material.Copia(), HuellaSHA256: r.HuellaSHA256, CaducaEn: r.CaducaEn},
		Replay: r.Replay, AuditoriaAccesoRef: r.AuditoriaAccesoRef}, nil
}

func (a *AutoridadVersionarRolBolsa) CerrarVersionarRolBolsa(ctx context.Context,
	s domain.SolicitudCierreVersionarRolBolsa) (domain.CierreVersionarRolBolsa, error) {
	var vacio domain.CierreVersionarRolBolsa
	if err := a.disponible(ctx); err != nil {
		return vacio, err
	}
	e, err := materialCierreVersionarRolBolsa(s)
	if err != nil {
		return vacio, err
	}
	var r respuestaCierreVersionarRolBolsa
	err = a.ejecutar(ctx, s.Aprobador, s.Evidencia, s.InstantaneaAutorizacion,
		e, cerrarVersionarRolBolsaSQL, func(b []byte) error {
			if decodificarGobiernoRol(b, &r) != nil || r.Estado != "permitido" ||
				r.Decision != domain.DecisionAprobarPropuestaPerfil || r.OperacionRef != s.OperacionRef {
				return ports.ErrAutoridadAdministracionPerfilesNoDisponible
			}
			var m domain.MaterialPropuestaVersionarRolBolsa
			if decodificarGobiernoRol([]byte(r.MaterialCanon), &m) != nil {
				return ports.ErrAutoridadAdministracionPerfilesNoDisponible
			}
			r.Cierre = domain.CierreVersionarRolBolsa{OperacionRef: r.OperacionRef, Material: m,
				PropuestaHuellaSHA256: r.PropuestaHuellaSHA256, Decision: r.Decision,
				ConfirmadoEn: r.ConfirmadoEn, AuditoriaAccesoRef: r.AuditoriaAccesoRef, Recibo: &r.Recibo}
			if r.Cierre.ValidarPara(s) != nil || r.Cierre.ConfirmadoEn.After(a.reloj.Ahora()) {
				return ports.ErrAutoridadAdministracionPerfilesNoDisponible
			}
			return nil
		})
	if err != nil {
		return vacio, err
	}
	return r.Cierre.Copia(), nil
}

func (a *AutoridadVersionarRolBolsa) ejecutar(ctx context.Context, actor domain.ContextoActor,
	evidencia domain.EvidenciaSesionAdministracionPerfiles, instantanea domain.InstantaneaAutorizacion,
	e Efecto, consulta string, validar func([]byte) error) error {
	errNoDisponible := ports.ErrAutoridadAdministracionPerfilesNoDisponible
	if err := a.disponible(ctx); err != nil {
		return err
	}
	if validar == nil || evidencia.ValidarEn(actor, a.reloj.Ahora()) != nil ||
		!domain.ReferenciaCorrelacionAutorizacionV2Valida(e.CorrelacionAccesoRef) {
		return ports.ConClaseVersionBolsa("evidencia_o_correlacion", errNoDisponible)
	}
	recurso, err := RecursoVersionarRolBolsa(e, instantanea.AsignacionPerfil)
	if err != nil {
		return ports.ConClaseVersionBolsa("recurso_asignacion", err)
	}
	entrega := e
	entrega.Material = append([]byte(nil), e.Material...)
	m, err := a.emisor.EmitirVersionarRolBolsa(ctx, actor, evidencia, instantanea, entrega)
	clear(entrega.Material)
	if err != nil {
		if errors.Is(err, domain.ErrAutorizacionDenegada) {
			return err
		}
		// Sólo viaja la clase del emisor, nunca su causa.
		clase := ports.ClaseFalloVersionBolsa(err)
		if clase == "" {
			clase = "v3_emision"
		}
		return ports.ConClaseVersionBolsa(clase, errNoDisponible)
	}
	h, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		return ports.ConClaseVersionBolsa("v3_material_incoherente", errNoDisponible)
	}
	r := m.ResumenCapacidad()
	ahora := a.reloj.Ahora()
	if m.ValidarEstructura() != nil || r.Operacion() != e.Accion || r.AudienciaConsumo() != e.Audiencia ||
		r.EfectoRef() != recurso.Referencia || r.EfectoHuellaSHA256() != h ||
		r.ContextoRef() != evidencia.ResultadoContexto.RegistroContextoRef ||
		r.ContextoHuellaSHA256() != evidencia.ResultadoContexto.HuellaSHA256 ||
		m.PersonaVersion() != actor.Instantanea.PersonaVersion || m.PerfilVersion() != actor.Instantanea.PerfilVersion ||
		ahora.Before(r.EmitidaEn()) || !ahora.Before(r.ExpiraEn()) {
		return ports.ConClaseVersionBolsa("v3_material_incoherente", errNoDisponible)
	}
	args := []any{string(e.Material), m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(),
		m.ContextoActorCanonico(), strconv.FormatUint(m.PersonaVersion(), 10), strconv.FormatUint(m.PerfilVersion(), 10),
		m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI()}
	defer func() {
		for _, arg := range args {
			if b, ok := arg.([]byte); ok {
				clear(b)
			}
		}
	}()
	tx, err := a.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil || ausente(tx) {
		return ports.ConClaseVersionBolsa("sql_transaccion", errNoDisponible)
	}
	defer func() {
		c, cancel := context.WithTimeout(context.Background(), plazoarranque.Ampliar(2*time.Second))
		defer cancel()
		_ = tx.Rollback(c)
	}()
	for _, configuracion := range []string{
		"SET LOCAL statement_timeout = '15s'",
		"SET LOCAL lock_timeout = '2s'",
		"SET LOCAL TimeZone = 'UTC'",
	} {
		if _, err := tx.Exec(ctx, configuracion); err != nil {
			return ports.ConClaseVersionBolsa("sql_configuracion", errNoDisponible)
		}
	}
	var b []byte
	if err := tx.QueryRow(ctx, consulta, args...).Scan(&b); err != nil {
		return ports.ConClaseVersionBolsa("sql_consulta", traducirGobiernoRol(ctx, err))
	}
	var intento struct {
		Estado           string `json:"estado"`
		Codigo           string `json:"codigo"`
		AuditoriaIntento struct {
			AuditoriaRef string `json:"auditoria_ref"`
		} `json:"auditoria_intento"`
	}
	var errorIntento error
	if json.Unmarshal(b, &intento) == nil && (intento.Estado == "denegado" || intento.Estado == "error") {
		if intento.AuditoriaIntento.AuditoriaRef == "" ||
			(intento.Estado == "denegado" && intento.Codigo != "version_rol_bolsa_denegado") ||
			(intento.Estado == "error" && intento.Codigo != "version_rol_bolsa_error") {
			return ports.ConClaseVersionBolsa("sql_intento_incoherente", errNoDisponible)
		}
		if intento.Estado == "denegado" {
			errorIntento = domain.ErrAutorizacionDenegada
		} else {
			errorIntento = ports.ConClaseVersionBolsa("sql_intento_error", errNoDisponible)
		}
	} else if err := validar(b); err != nil {
		return ports.ConClaseVersionBolsa("sql_respuesta_invalida", err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return ports.ConClaseVersionBolsa("sql_commit", errNoDisponible)
	}
	if errorIntento != nil {
		return errors.Join(ports.ErrGobiernoRolIntentoAuditado, errorIntento)
	}
	return nil
}

type respuestaPropuestaVersionarRolBolsa struct {
	Estado             string                            `json:"estado"`
	Replay             bool                              `json:"replay"`
	MaterialCanon      string                            `json:"material_canon"`
	HuellaSHA256       string                            `json:"huella_sha256"`
	CaducaEn           time.Time                         `json:"caduca_en"`
	AuditoriaAccesoRef string                            `json:"auditoria_acceso_ref"`
	Propuesta          domain.PropuestaVersionarRolBolsa `json:"-"`
}

type respuestaCierreVersionarRolBolsa struct {
	Estado                string                                         `json:"estado"`
	Replay                bool                                           `json:"replay"`
	OperacionRef          string                                         `json:"operacion_ref"`
	MaterialCanon         string                                         `json:"material_canon"`
	PropuestaHuellaSHA256 string                                         `json:"propuesta_huella_sha256"`
	Decision              domain.DecisionPropuestaAdministracionPerfiles `json:"decision"`
	ConfirmadoEn          time.Time                                      `json:"confirmado_en"`
	AuditoriaAccesoRef    string                                         `json:"auditoria_acceso_ref"`
	Recibo                domain.ReciboVersionarRolBolsa                 `json:"recibo"`
	Cierre                domain.CierreVersionarRolBolsa                 `json:"-"`
}
