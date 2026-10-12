package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// AutoridadVersionarRolBolsaB103 comparte el emisor y la transacción de B1,
// pero acredita un LOGIN propio con EXECUTE exclusivamente sobre AUT75 v2.
type AutoridadVersionarRolBolsaB103 struct{ base *AutoridadVersionarRolBolsa }

var _ ports.AutoridadVersionarRolBolsaB103 = (*AutoridadVersionarRolBolsaB103)(nil)
var _ ports.AutoridadActosAdministracionPerfiles = (*AutoridadVersionarRolBolsaB103)(nil)

const (
	proponerVersionarRolBolsaB103SQL = `SELECT vec_autorizacion.proponer_version_rol_bolsa_v2` + argumentosV3
	cerrarVersionarRolBolsaB103SQL   = `SELECT vec_autorizacion.cerrar_version_rol_bolsa_v2` + argumentosV3
)

const acreditarVersionarRolBolsaB103SQL = `SELECT current_user=session_user AND r.rolcanlogin AND r.rolinherit
 AND NOT (r.rolsuper OR r.rolcreatedb OR r.rolcreaterole OR r.rolreplication OR r.rolbypassrls)
 AND r.rolconfig IS NULL AND (SELECT count(*) FROM pg_catalog.pg_auth_members m WHERE m.member=r.oid)=1
 AND EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.member=r.oid
   AND m.roleid=pg_catalog.to_regrole('vec_admin_version_rol_bolsa_b103_ejecutor')
   AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
 AND NOT pg_catalog.has_database_privilege(current_user,current_database(),'CREATE,TEMP')
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_namespace n WHERE n.nspname LIKE 'vec\_%' ESCAPE '\'
   AND pg_catalog.has_schema_privilege(current_user,n.oid,'CREATE'))
 AND pg_catalog.to_regprocedure('vec_autorizacion.proponer_version_rol_bolsa_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 AND pg_catalog.to_regprocedure('vec_autorizacion.cerrar_version_rol_bolsa_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 AND pg_catalog.has_function_privilege(current_user,
   pg_catalog.to_regprocedure('vec_autorizacion.proponer_version_rol_bolsa_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),'EXECUTE')
 AND pg_catalog.has_function_privilege(current_user,
   pg_catalog.to_regprocedure('vec_autorizacion.cerrar_version_rol_bolsa_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),'EXECUTE')
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
   WHERE n.nspname LIKE 'vec\_%' ESCAPE '\' AND pg_catalog.has_function_privilege(current_user,p.oid,'EXECUTE')
   AND p.oid NOT IN (pg_catalog.to_regprocedure('vec_autorizacion.proponer_version_rol_bolsa_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),
    pg_catalog.to_regprocedure('vec_autorizacion.cerrar_version_rol_bolsa_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')))
 FROM pg_catalog.pg_roles r WHERE r.rolname=session_user`

func NuevaAutoridadVersionarRolBolsaB103(ctx context.Context, pool *pgxpool.Pool,
	catalogo ports.FuenteCatalogoAccionesAdministracionV1, emisor EmisorVersionarRolBolsa,
	reloj ports.Reloj) (*AutoridadVersionarRolBolsaB103, error) {
	return nuevaAutoridadVersionarRolBolsaB103(ctx, pool, catalogo, emisor, reloj)
}

func nuevaAutoridadVersionarRolBolsaB103(ctx context.Context, pool conexion,
	catalogo ports.FuenteCatalogoAccionesAdministracionV1, emisor EmisorVersionarRolBolsa,
	reloj ports.Reloj) (*AutoridadVersionarRolBolsaB103, error) {
	if ctx == nil || ausente(pool) || ausente(catalogo) || ausente(emisor) || ausente(reloj) {
		return nil, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var acreditado bool
	if err := pool.QueryRow(ctx, acreditarVersionarRolBolsaB103SQL).Scan(&acreditado); err != nil || !acreditado {
		return nil, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	return &AutoridadVersionarRolBolsaB103{base: &AutoridadVersionarRolBolsa{
		pool: pool, catalogo: catalogo, emisor: emisor, reloj: reloj,
	}}, nil
}

func (a *AutoridadVersionarRolBolsaB103) disponible(ctx context.Context) error {
	if a == nil || a.base == nil {
		return ports.ConClaseVersionBolsa("autoridad_no_disponible", ports.ErrAutoridadAdministracionPerfilesNoDisponible)
	}
	return a.base.disponible(ctx)
}

// Este LOGIN sólo dispone de las fachadas B103. Los actos generales de la
// interfaz de servicio siguen denegados sin consultar otra autoridad SQL.
func (*AutoridadVersionarRolBolsaB103) AplicarActoOrdinario(ctx context.Context,
	_ domain.SolicitudActoAdministracionPerfiles) (domain.ReciboAdministracionPerfiles, error) {
	return domain.ReciboAdministracionPerfiles{}, errorActoAjenoGobierno(ctx)
}

func (*AutoridadVersionarRolBolsaB103) ProponerActoSensible(ctx context.Context,
	_ domain.SolicitudActoAdministracionPerfiles) (ports.PropuestaAdministracionPerfiles, error) {
	return ports.PropuestaAdministracionPerfiles{}, errorActoAjenoGobierno(ctx)
}

func (*AutoridadVersionarRolBolsaB103) CerrarPropuestaSensible(ctx context.Context,
	_ domain.SolicitudCierrePropuestaAdministracionPerfiles) (ports.CierrePropuestaAdministracionPerfiles, error) {
	return ports.CierrePropuestaAdministracionPerfiles{}, errorActoAjenoGobierno(ctx)
}

// ResolverCatalogoVersionarRolBolsaB103 sólo resuelve la fuente de concesiones.
// Los documentos del plan son expectativas CAS y AUT75 los coteja en SQL.
func (a *AutoridadVersionarRolBolsaB103) ResolverCatalogoVersionarRolBolsaB103(ctx context.Context,
	s domain.SolicitudPropuestaVersionarRolBolsaB103) (domain.CatalogoAccionesAdministracionV1, error) {
	var vacio domain.CatalogoAccionesAdministracionV1
	if err := a.disponible(ctx); err != nil {
		return vacio, err
	}
	if s.Validar() != nil {
		return vacio, domain.ErrVersionarRolBolsaB103Invalido
	}
	c, err := a.base.catalogo.ObtenerCatalogoAccionesAdministracionV1(ctx, s.Intencion.CatalogoRef,
		s.Intencion.CatalogoVersion, s.Intencion.CatalogoHuellaSHA256)
	if err != nil {
		return vacio, ports.ConClaseVersionBolsa("catalogo_consulta", err)
	}
	plan, err := domain.PrepararPlanVersionarRolBolsaB103(c, s.Intencion, a.base.reloj.Ahora())
	h, e := plan.HuellaSHA256()
	if err != nil || e != nil || h != s.HuellaPlanEsperada {
		return vacio, ports.ConClaseVersionBolsa("catalogo_plan_huella", domain.ErrVersionarRolBolsaB103Invalido)
	}
	return c, nil
}

func (a *AutoridadVersionarRolBolsaB103) ProponerVersionarRolBolsaB103(ctx context.Context,
	o domain.OrdenPropuestaVersionarRolBolsaB103) (ports.ResultadoPropuestaVersionarRolBolsaB103, error) {
	var vacio ports.ResultadoPropuestaVersionarRolBolsaB103
	if err := a.disponible(ctx); err != nil {
		return vacio, err
	}
	e, err := materialPropuestaVersionarRolBolsaB103(o)
	if err != nil {
		return vacio, ports.ConClaseVersionBolsa("material_propuesta", err)
	}
	var r respuestaPropuestaVersionarRolBolsaB103
	err = a.base.ejecutarConRecurso(ctx, o.Solicitud.Actor, o.Solicitud.Evidencia,
		o.Solicitud.InstantaneaAutorizacion, e, proponerVersionarRolBolsaB103SQL, RecursoVersionarRolBolsaB103, func(b []byte) error {
			if decodificarGobiernoRol(b, &r) != nil || r.Estado != "permitido" || r.HuellaSHA256 == "" ||
				len(r.MaterialCanon) == 0 || len(r.MaterialCanon) > 131072 {
				return ports.ConClaseVersionBolsa("sql_respuesta_invalida", ports.ErrAutoridadAdministracionPerfilesNoDisponible)
			}
			var m domain.MaterialPropuestaVersionarRolBolsaB103
			if decodificarGobiernoRol([]byte(r.MaterialCanon), &m) != nil {
				return ports.ConClaseVersionBolsa("sql_respuesta_invalida", ports.ErrAutoridadAdministracionPerfilesNoDisponible)
			}
			// jsonb serializa timestamptz como «…+00:00»; el dominio exige UTC canónico.
			r.CaducaEn = r.CaducaEn.UTC()
			r.Propuesta = domain.PropuestaVersionarRolBolsaB103{Material: m, HuellaSHA256: r.HuellaSHA256,
				CaducaEn: r.CaducaEn}
			if (ports.ResultadoPropuestaVersionarRolBolsaB103{Propuesta: r.Propuesta,
				Replay: r.Replay, AuditoriaAccesoRef: r.AuditoriaAccesoRef}).ValidarPara(o, a.base.reloj.Ahora()) != nil {
				return ports.ConClaseVersionBolsa("sql_respuesta_invalida", ports.ErrAutoridadAdministracionPerfilesNoDisponible)
			}
			return nil
		})
	if err != nil {
		return vacio, err
	}
	return ports.ResultadoPropuestaVersionarRolBolsaB103{Propuesta: domain.PropuestaVersionarRolBolsaB103{
		Material: r.Propuesta.Material.Copia(), HuellaSHA256: r.HuellaSHA256, CaducaEn: r.CaducaEn},
		Replay: r.Replay, AuditoriaAccesoRef: r.AuditoriaAccesoRef}, nil
}

func (a *AutoridadVersionarRolBolsaB103) CerrarVersionarRolBolsaB103(ctx context.Context,
	s domain.SolicitudCierreVersionarRolBolsaB103) (domain.CierreVersionarRolBolsaB103, error) {
	var vacio domain.CierreVersionarRolBolsaB103
	if err := a.disponible(ctx); err != nil {
		return vacio, err
	}
	e, err := materialCierreVersionarRolBolsaB103(s)
	if err != nil {
		return vacio, err
	}
	var r respuestaCierreVersionarRolBolsaB103
	err = a.base.ejecutarConRecurso(ctx, s.Aprobador, s.Evidencia, s.InstantaneaAutorizacion,
		e, cerrarVersionarRolBolsaB103SQL, RecursoVersionarRolBolsaB103, func(b []byte) error {
			if decodificarGobiernoRol(b, &r) != nil || r.Estado != "permitido" ||
				r.Decision != domain.DecisionAprobarPropuestaPerfil || r.OperacionRef != s.OperacionRef ||
				len(r.MaterialCanon) == 0 || len(r.MaterialCanon) > 131072 {
				return ports.ErrAutoridadAdministracionPerfilesNoDisponible
			}
			var m domain.MaterialPropuestaVersionarRolBolsaB103
			if decodificarGobiernoRol([]byte(r.MaterialCanon), &m) != nil {
				return ports.ErrAutoridadAdministracionPerfilesNoDisponible
			}
			// jsonb serializa timestamptz como «…+00:00»; el dominio exige UTC canónico.
			r.ConfirmadoEn = r.ConfirmadoEn.UTC()
			r.Cierre = domain.CierreVersionarRolBolsaB103{OperacionRef: r.OperacionRef, Material: m,
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

type respuestaPropuestaVersionarRolBolsaB103 struct {
	Estado             string                                `json:"estado"`
	Replay             bool                                  `json:"replay"`
	MaterialCanon      string                                `json:"material_canon"`
	HuellaSHA256       string                                `json:"huella_sha256"`
	CaducaEn           time.Time                             `json:"caduca_en"`
	AuditoriaAccesoRef string                                `json:"auditoria_acceso_ref"`
	Propuesta          domain.PropuestaVersionarRolBolsaB103 `json:"-"`
}

type respuestaCierreVersionarRolBolsaB103 struct {
	Estado                string                                         `json:"estado"`
	Replay                bool                                           `json:"replay"`
	OperacionRef          string                                         `json:"operacion_ref"`
	MaterialCanon         string                                         `json:"material_canon"`
	PropuestaHuellaSHA256 string                                         `json:"propuesta_huella_sha256"`
	Decision              domain.DecisionPropuestaAdministracionPerfiles `json:"decision"`
	ConfirmadoEn          time.Time                                      `json:"confirmado_en"`
	AuditoriaAccesoRef    string                                         `json:"auditoria_acceso_ref"`
	Recibo                domain.ReciboVersionarRolBolsaB103             `json:"recibo"`
	Cierre                domain.CierreVersionarRolBolsaB103             `json:"-"`
}
