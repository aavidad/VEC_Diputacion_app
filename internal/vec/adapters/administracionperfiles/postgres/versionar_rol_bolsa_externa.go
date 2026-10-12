package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// AutoridadVersionarRolBolsaExterna comparte el emisor y la transacción de B1,
// pero acredita un LOGIN propio con EXECUTE exclusivamente sobre AUT77 v1.
type AutoridadVersionarRolBolsaExterna struct{ base *AutoridadVersionarRolBolsa }

var _ ports.AutoridadVersionarRolBolsaExterna = (*AutoridadVersionarRolBolsaExterna)(nil)
var _ ports.AutoridadCompletaVersionarRolBolsaExterna = (*AutoridadVersionarRolBolsaExterna)(nil)
var _ ports.AutoridadActosAdministracionPerfiles = (*AutoridadVersionarRolBolsaExterna)(nil)

const (
	proponerVersionarRolBolsaExternaSQL = `SELECT vec_autorizacion.proponer_version_rol_bolsa_externa_v1` + argumentosV3
	cerrarVersionarRolBolsaExternaSQL   = `SELECT vec_autorizacion.cerrar_version_rol_bolsa_externa_v1` + argumentosV3
)

const acreditarVersionarRolBolsaExternaSQL = `SELECT current_user=session_user AND r.rolcanlogin AND r.rolinherit
 AND NOT (r.rolsuper OR r.rolcreatedb OR r.rolcreaterole OR r.rolreplication OR r.rolbypassrls)
 AND r.rolconfig IS NULL AND (SELECT count(*) FROM pg_catalog.pg_auth_members m WHERE m.member=r.oid)=1
 AND EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.member=r.oid
   AND m.roleid=pg_catalog.to_regrole('vec_admin_version_rol_bolsa_externa_ejecutor')
   AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
 AND NOT pg_catalog.has_database_privilege(current_user,current_database(),'CREATE,TEMP')
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_namespace n WHERE n.nspname LIKE 'vec\_%' ESCAPE '\'
   AND pg_catalog.has_schema_privilege(current_user,n.oid,'CREATE'))
 AND pg_catalog.to_regprocedure('vec_autorizacion.proponer_version_rol_bolsa_externa_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 AND pg_catalog.to_regprocedure('vec_autorizacion.cerrar_version_rol_bolsa_externa_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 AND pg_catalog.to_regprocedure('vec_autorizacion.resolver_rol_administrable_v1(text)') IS NOT NULL
 AND pg_catalog.to_regprocedure('vec_autorizacion.resolver_base_version_rol_bolsa_externa_v1()') IS NOT NULL
 AND pg_catalog.has_function_privilege(current_user,
   pg_catalog.to_regprocedure('vec_autorizacion.proponer_version_rol_bolsa_externa_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),'EXECUTE')
 AND pg_catalog.has_function_privilege(current_user,
   pg_catalog.to_regprocedure('vec_autorizacion.cerrar_version_rol_bolsa_externa_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),'EXECUTE')
 AND pg_catalog.has_function_privilege(current_user,
   pg_catalog.to_regprocedure('vec_autorizacion.resolver_rol_administrable_v1(text)'),'EXECUTE')
 AND pg_catalog.has_function_privilege(current_user,
   pg_catalog.to_regprocedure('vec_autorizacion.resolver_base_version_rol_bolsa_externa_v1()'),'EXECUTE')
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
   WHERE n.nspname LIKE 'vec\_%' ESCAPE '\' AND pg_catalog.has_function_privilege(current_user,p.oid,'EXECUTE')
   AND p.oid NOT IN (pg_catalog.to_regprocedure('vec_autorizacion.proponer_version_rol_bolsa_externa_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),
    pg_catalog.to_regprocedure('vec_autorizacion.cerrar_version_rol_bolsa_externa_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),
    pg_catalog.to_regprocedure('vec_autorizacion.resolver_rol_administrable_v1(text)'),
    pg_catalog.to_regprocedure('vec_autorizacion.resolver_base_version_rol_bolsa_externa_v1()')))
 FROM pg_catalog.pg_roles r WHERE r.rolname=session_user`

func NuevaAutoridadVersionarRolBolsaExterna(ctx context.Context, pool *pgxpool.Pool,
	catalogo ports.FuenteCatalogoAccionesAdministracionV1, emisor EmisorVersionarRolBolsa,
	reloj ports.Reloj) (*AutoridadVersionarRolBolsaExterna, error) {
	return nuevaAutoridadVersionarRolBolsaExterna(ctx, pool, catalogo, emisor, reloj)
}

func nuevaAutoridadVersionarRolBolsaExterna(ctx context.Context, pool conexion,
	catalogo ports.FuenteCatalogoAccionesAdministracionV1, emisor EmisorVersionarRolBolsa,
	reloj ports.Reloj) (*AutoridadVersionarRolBolsaExterna, error) {
	if ctx == nil || ausente(pool) || ausente(catalogo) || ausente(emisor) || ausente(reloj) {
		return nil, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var acreditado bool
	if err := pool.QueryRow(ctx, acreditarVersionarRolBolsaExternaSQL).Scan(&acreditado); err != nil || !acreditado {
		return nil, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	return &AutoridadVersionarRolBolsaExterna{base: &AutoridadVersionarRolBolsa{
		pool: pool, catalogo: catalogo, emisor: emisor, reloj: reloj,
	}}, nil
}

func (a *AutoridadVersionarRolBolsaExterna) disponible(ctx context.Context) error {
	if a == nil || a.base == nil {
		return ports.ConClaseVersionBolsa("autoridad_no_disponible", ports.ErrAutoridadAdministracionPerfilesNoDisponible)
	}
	return a.base.disponible(ctx)
}

// Este LOGIN sólo dispone de las fachadas AUT77. Los actos generales de la
// interfaz de servicio siguen denegados sin consultar otra autoridad SQL.
func (*AutoridadVersionarRolBolsaExterna) AplicarActoOrdinario(ctx context.Context,
	_ domain.SolicitudActoAdministracionPerfiles) (domain.ReciboAdministracionPerfiles, error) {
	return domain.ReciboAdministracionPerfiles{}, errorActoAjenoGobierno(ctx)
}

func (*AutoridadVersionarRolBolsaExterna) ProponerActoSensible(ctx context.Context,
	_ domain.SolicitudActoAdministracionPerfiles) (ports.PropuestaAdministracionPerfiles, error) {
	return ports.PropuestaAdministracionPerfiles{}, errorActoAjenoGobierno(ctx)
}

func (*AutoridadVersionarRolBolsaExterna) CerrarPropuestaSensible(ctx context.Context,
	_ domain.SolicitudCierrePropuestaAdministracionPerfiles) (ports.CierrePropuestaAdministracionPerfiles, error) {
	return ports.CierrePropuestaAdministracionPerfiles{}, errorActoAjenoGobierno(ctx)
}

// ResolverFuentesVersionarRolBolsaExterna toma las dos entradas del catálogo
// gobernado y la preimagen externa del resolver nominal AUT77.
func (a *AutoridadVersionarRolBolsaExterna) ResolverFuentesVersionarRolBolsaExterna(ctx context.Context,
	s domain.SolicitudPropuestaVersionarRolBolsaExterna) (ports.FuentesVersionarRolBolsaExterna, error) {
	var vacio ports.FuentesVersionarRolBolsaExterna
	if err := a.disponible(ctx); err != nil {
		return vacio, err
	}
	if s.Validar() != nil {
		return vacio, domain.ErrVersionarRolBolsaExternaInvalido
	}
	c, err := a.base.catalogo.ObtenerCatalogoAccionesAdministracionV1(ctx, s.Intencion.CatalogoRef,
		s.Intencion.CatalogoVersion, s.Intencion.CatalogoHuellaSHA256)
	if err != nil {
		return vacio, ports.ConClaseVersionBolsa("catalogo_consulta", err)
	}
	var b []byte
	if err := a.base.pool.QueryRow(ctx, "SELECT vec_autorizacion.resolver_base_version_rol_bolsa_externa_v1()").Scan(&b); err != nil {
		return vacio, ports.ConClaseVersionBolsa("base_externa_consulta", traducirGobiernoRol(ctx, err))
	}
	var base domain.PerfilPublicadoAdministracionV1
	if len(b) == 0 || len(b) > 131072 || decodificarGobiernoRol(b, &base) != nil {
		return vacio, ports.ConClaseVersionBolsa("base_externa_respuesta", ports.ErrAutoridadAdministracionPerfilesNoDisponible)
	}
	plan, err := domain.PrepararPlanVersionarRolBolsaExterna(c, base, s.Intencion, a.base.reloj.Ahora())
	h, e := plan.HuellaSHA256()
	if err != nil || e != nil || h != s.HuellaPlanEsperada {
		return vacio, ports.ConClaseVersionBolsa("catalogo_plan_huella", domain.ErrVersionarRolBolsaExternaInvalido)
	}
	return ports.FuentesVersionarRolBolsaExterna{Catalogo: c, Base: base}, nil
}

func (a *AutoridadVersionarRolBolsaExterna) ProponerVersionarRolBolsaExterna(ctx context.Context,
	o domain.OrdenPropuestaVersionarRolBolsaExterna) (ports.ResultadoPropuestaVersionarRolBolsaExterna, error) {
	var vacio ports.ResultadoPropuestaVersionarRolBolsaExterna
	if err := a.disponible(ctx); err != nil {
		return vacio, err
	}
	e, err := materialPropuestaVersionarRolBolsaExterna(o)
	if err != nil {
		return vacio, ports.ConClaseVersionBolsa("material_propuesta", err)
	}
	var r respuestaPropuestaVersionarRolBolsaExterna
	err = a.base.ejecutarConRecurso(ctx, o.Solicitud.Actor, o.Solicitud.Evidencia,
		o.Solicitud.InstantaneaAutorizacion, e, proponerVersionarRolBolsaExternaSQL, RecursoVersionarRolBolsaExterna, func(b []byte) error {
			if decodificarGobiernoRol(b, &r) != nil || r.Estado != "permitido" || r.HuellaSHA256 == "" ||
				len(r.MaterialCanon) == 0 || len(r.MaterialCanon) > 131072 {
				return ports.ConClaseVersionBolsa("sql_respuesta_invalida", ports.ErrAutoridadAdministracionPerfilesNoDisponible)
			}
			var m domain.MaterialPropuestaVersionarRolBolsaExterna
			if decodificarGobiernoRol([]byte(r.MaterialCanon), &m) != nil {
				return ports.ConClaseVersionBolsa("sql_respuesta_invalida", ports.ErrAutoridadAdministracionPerfilesNoDisponible)
			}
			// jsonb serializa timestamptz como «…+00:00»; el dominio exige UTC canónico.
			r.CaducaEn = r.CaducaEn.UTC()
			r.Propuesta = domain.PropuestaVersionarRolBolsaExterna{Material: m, HuellaSHA256: r.HuellaSHA256,
				CaducaEn: r.CaducaEn}
			if (ports.ResultadoPropuestaVersionarRolBolsaExterna{Propuesta: r.Propuesta,
				Replay: r.Replay, AuditoriaAccesoRef: r.AuditoriaAccesoRef}).ValidarPara(o, a.base.reloj.Ahora()) != nil {
				return ports.ConClaseVersionBolsa("sql_respuesta_invalida", ports.ErrAutoridadAdministracionPerfilesNoDisponible)
			}
			return nil
		})
	if err != nil {
		return vacio, err
	}
	return ports.ResultadoPropuestaVersionarRolBolsaExterna{Propuesta: domain.PropuestaVersionarRolBolsaExterna{
		Material: r.Propuesta.Material.Copia(), HuellaSHA256: r.HuellaSHA256, CaducaEn: r.CaducaEn},
		Replay: r.Replay, AuditoriaAccesoRef: r.AuditoriaAccesoRef}, nil
}

func (a *AutoridadVersionarRolBolsaExterna) CerrarVersionarRolBolsaExterna(ctx context.Context,
	s domain.SolicitudCierreVersionarRolBolsaExterna) (domain.CierreVersionarRolBolsaExterna, error) {
	var vacio domain.CierreVersionarRolBolsaExterna
	if err := a.disponible(ctx); err != nil {
		return vacio, err
	}
	e, err := materialCierreVersionarRolBolsaExterna(s)
	if err != nil {
		return vacio, err
	}
	var r respuestaCierreVersionarRolBolsaExterna
	err = a.base.ejecutarConRecurso(ctx, s.Aprobador, s.Evidencia, s.InstantaneaAutorizacion,
		e, cerrarVersionarRolBolsaExternaSQL, RecursoVersionarRolBolsaExterna, func(b []byte) error {
			if decodificarGobiernoRol(b, &r) != nil || r.Estado != "permitido" ||
				r.Decision != domain.DecisionAprobarPropuestaPerfil || r.OperacionRef != s.OperacionRef ||
				len(r.MaterialCanon) == 0 || len(r.MaterialCanon) > 131072 {
				return ports.ErrAutoridadAdministracionPerfilesNoDisponible
			}
			var m domain.MaterialPropuestaVersionarRolBolsaExterna
			if decodificarGobiernoRol([]byte(r.MaterialCanon), &m) != nil {
				return ports.ErrAutoridadAdministracionPerfilesNoDisponible
			}
			// jsonb serializa timestamptz como «…+00:00»; el dominio exige UTC canónico.
			r.ConfirmadoEn = r.ConfirmadoEn.UTC()
			r.Cierre = domain.CierreVersionarRolBolsaExterna{OperacionRef: r.OperacionRef, Material: m,
				PropuestaHuellaSHA256: r.PropuestaHuellaSHA256, Decision: r.Decision,
				ConfirmadoEn: r.ConfirmadoEn, AuditoriaAccesoRef: r.AuditoriaAccesoRef, Recibo: &r.Recibo}
			if r.Cierre.ValidarPara(s) != nil || r.Cierre.ConfirmadoEn.After(a.base.reloj.Ahora()) {
				return ports.ErrAutoridadAdministracionPerfilesNoDisponible
			}
			return nil
		})
	if err != nil {
		return vacio, err
	}
	return r.Cierre.Copia(), nil
}

type respuestaPropuestaVersionarRolBolsaExterna struct {
	Estado             string                                   `json:"estado"`
	Replay             bool                                     `json:"replay"`
	MaterialCanon      string                                   `json:"material_canon"`
	HuellaSHA256       string                                   `json:"huella_sha256"`
	CaducaEn           time.Time                                `json:"caduca_en"`
	AuditoriaAccesoRef string                                   `json:"auditoria_acceso_ref"`
	Propuesta          domain.PropuestaVersionarRolBolsaExterna `json:"-"`
}

type respuestaCierreVersionarRolBolsaExterna struct {
	Estado                string                                         `json:"estado"`
	Replay                bool                                           `json:"replay"`
	OperacionRef          string                                         `json:"operacion_ref"`
	MaterialCanon         string                                         `json:"material_canon"`
	PropuestaHuellaSHA256 string                                         `json:"propuesta_huella_sha256"`
	Decision              domain.DecisionPropuestaAdministracionPerfiles `json:"decision"`
	ConfirmadoEn          time.Time                                      `json:"confirmado_en"`
	AuditoriaAccesoRef    string                                         `json:"auditoria_acceso_ref"`
	Recibo                domain.ReciboVersionarRolBolsaExterna          `json:"recibo"`
	Cierre                domain.CierreVersionarRolBolsaExterna          `json:"-"`
}
