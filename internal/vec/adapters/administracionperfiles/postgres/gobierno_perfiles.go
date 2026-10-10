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

// EmisorGobiernoRolNuevo sólo emite material de las dos acciones de
// definición. El emisor productivo usa el PDP, contexto y atestación V3
// comunes; un efecto HTTP no puede implementar esta interfaz en composición.
type EmisorGobiernoRolNuevo interface {
	EmitirGobiernoRolNuevo(context.Context, domain.ContextoActor,
		domain.EvidenciaSesionAdministracionPerfiles, domain.InstantaneaAutorizacion,
		Efecto) (ports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

// AutoridadGobiernoRolNuevo usa un LOGIN distinto del de AUT24. El puerto de
// catálogo es sólo lectura exacta: AUT60 verifica de nuevo su cabeza dentro
// de la misma transacción que el consumo V3 y la publicación.
type AutoridadGobiernoRolNuevo struct {
	pool   conexion
	fuente ports.FuenteCatalogoAccionesAdministracionV1
	emisor EmisorGobiernoRolNuevo
	reloj  ports.Reloj
}

var _ ports.AutoridadGobiernoPerfiles = (*AutoridadGobiernoRolNuevo)(nil)

const (
	proponerGobiernoRolSQL = `SELECT vec_autorizacion.proponer_gobierno_rol_nuevo_v1` + argumentosV3
	cerrarGobiernoRolSQL   = `SELECT vec_autorizacion.cerrar_gobierno_rol_nuevo_v1` + argumentosV3
)

const acreditarGobiernoRolSQL = `SELECT current_user=session_user AND r.rolcanlogin AND r.rolinherit
 AND NOT (r.rolsuper OR r.rolcreatedb OR r.rolcreaterole OR r.rolreplication OR r.rolbypassrls)
 AND r.rolconfig IS NULL AND (SELECT count(*) FROM pg_catalog.pg_auth_members m WHERE m.member=r.oid)=1
 AND EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.member=r.oid
   AND m.roleid=pg_catalog.to_regrole('vec_admin_gobierno_roles_ejecutor')
   AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
 AND NOT pg_catalog.has_database_privilege(current_user,current_database(),'CREATE,TEMP')
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_namespace n WHERE n.nspname LIKE 'vec\_%' ESCAPE '\'
   AND pg_catalog.has_schema_privilege(current_user,n.oid,'CREATE'))
 AND pg_catalog.to_regprocedure('vec_autorizacion.proponer_gobierno_rol_nuevo_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 AND pg_catalog.to_regprocedure('vec_autorizacion.cerrar_gobierno_rol_nuevo_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 AND pg_catalog.to_regprocedure('vec_autorizacion.resolver_rol_administrable_v1(text)') IS NOT NULL
 AND pg_catalog.has_function_privilege(current_user,
   pg_catalog.to_regprocedure('vec_autorizacion.proponer_gobierno_rol_nuevo_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),'EXECUTE')
 AND pg_catalog.has_function_privilege(current_user,
   pg_catalog.to_regprocedure('vec_autorizacion.cerrar_gobierno_rol_nuevo_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),'EXECUTE')
 AND pg_catalog.has_function_privilege(current_user,
   pg_catalog.to_regprocedure('vec_autorizacion.resolver_rol_administrable_v1(text)'),'EXECUTE')
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
   WHERE n.nspname LIKE 'vec\_%' ESCAPE '\' AND pg_catalog.has_function_privilege(current_user,p.oid,'EXECUTE')
   AND p.oid NOT IN (pg_catalog.to_regprocedure('vec_autorizacion.proponer_gobierno_rol_nuevo_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),
    pg_catalog.to_regprocedure('vec_autorizacion.cerrar_gobierno_rol_nuevo_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),
    pg_catalog.to_regprocedure('vec_autorizacion.resolver_rol_administrable_v1(text)')))
 FROM pg_catalog.pg_roles r WHERE r.rolname=session_user`

func NuevaAutoridadGobiernoRolNuevo(ctx context.Context, pool *pgxpool.Pool,
	fuente ports.FuenteCatalogoAccionesAdministracionV1, emisor EmisorGobiernoRolNuevo,
	reloj ports.Reloj) (*AutoridadGobiernoRolNuevo, error) {
	if ctx == nil || pool == nil || ausente(fuente) || ausente(emisor) || ausente(reloj) {
		return nil, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	return nuevaAutoridadGobiernoRolNuevo(ctx, pool, fuente, emisor, reloj)
}

func nuevaAutoridadGobiernoRolNuevo(ctx context.Context, pool conexion,
	fuente ports.FuenteCatalogoAccionesAdministracionV1, emisor EmisorGobiernoRolNuevo,
	reloj ports.Reloj) (*AutoridadGobiernoRolNuevo, error) {
	if ctx == nil || ausente(pool) || ausente(fuente) || ausente(emisor) || ausente(reloj) {
		return nil, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var acreditado bool
	if err := pool.QueryRow(ctx, acreditarGobiernoRolSQL).Scan(&acreditado); err != nil || !acreditado {
		return nil, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	return &AutoridadGobiernoRolNuevo{pool: pool, fuente: fuente, emisor: emisor, reloj: reloj}, nil
}

func (a *AutoridadGobiernoRolNuevo) ResolverCatalogoGobiernoPerfil(ctx context.Context,
	s domain.SolicitudPropuestaGobiernoPerfil) (domain.CatalogoAccionesAdministracionV1, error) {
	var vacio domain.CatalogoAccionesAdministracionV1
	if err := a.disponibleGobierno(ctx); err != nil {
		return vacio, err
	}
	if s.Validar() != nil || s.Intencion.Operacion != domain.OperacionCrearPerfilGobernado ||
		s.Intencion.Publicacion == nil || s.Intencion.Deshabilitacion != nil ||
		len(s.Intencion.Publicacion.Selecciones) != 1 || len(s.Intencion.Publicacion.RolPropuesto.Concesiones) != 1 {
		return vacio, domain.ErrPlanGobiernoPerfilInvalido
	}
	p := s.Intencion.Publicacion
	c, err := a.fuente.ObtenerCatalogoAccionesAdministracionV1(ctx, p.CatalogoRef, p.CatalogoVersion, p.CatalogoHuellaSHA256)
	if err != nil {
		return vacio, err
	}
	plan, _, err := domain.PrepararPlanGobiernoRolNuevoDesdeCatalogo(c, s.Intencion,
		a.reloj.Ahora(), s.Actor.PersonaRef)
	if err != nil || plan.Operacion != domain.OperacionCrearPerfilGobernado || plan.Base != nil ||
		plan.DefinicionNueva == nil || len(plan.DefinicionNueva.Concesiones) != 1 {
		return vacio, domain.ErrPlanGobiernoPerfilInvalido
	}
	h, err := plan.HuellaSHA256()
	if err != nil || h != s.HuellaPlanEsperada {
		return vacio, domain.ErrPlanGobiernoPerfilInvalido
	}
	return c, nil
}

func (a *AutoridadGobiernoRolNuevo) ProponerGobiernoPerfil(ctx context.Context,
	o domain.OrdenPropuestaGobiernoPerfil) (domain.PropuestaGobiernoPerfil, error) {
	r, err := a.ProponerGobiernoRolNuevoRecuperable(ctx, o)
	if err != nil {
		return domain.PropuestaGobiernoPerfil{}, err
	}
	return r.Propuesta, nil
}

var _ ports.AutoridadPropuestaGobiernoRolNuevoRecuperable = (*AutoridadGobiernoRolNuevo)(nil)

func (a *AutoridadGobiernoRolNuevo) ProponerGobiernoRolNuevoRecuperable(ctx context.Context,
	o domain.OrdenPropuestaGobiernoPerfil) (ports.ResultadoPropuestaGobiernoRolNuevo, error) {
	var vacia ports.ResultadoPropuestaGobiernoRolNuevo
	if err := a.disponibleGobierno(ctx); err != nil {
		return vacia, err
	}
	e, err := materialPropuestaGobiernoRol(o)
	if err != nil {
		return vacia, err
	}
	var r propuestaGobiernoRolRespuesta
	err = a.ejecutarGobierno(ctx, o.Solicitud.Actor, o.Solicitud.Evidencia,
		o.Solicitud.InstantaneaAutorizacion, e, proponerGobiernoRolSQL, func(b []byte) error {
			if decodificarGobiernoRol(b, &r) != nil || r.Estado != "permitido" ||
				r.HuellaSHA256 == "" {
				return ports.ErrAutoridadAdministracionPerfilesNoDisponible
			}
			var m domain.MaterialPropuestaGobiernoPerfil
			if decodificarGobiernoRol([]byte(r.MaterialCanon), &m) != nil {
				return ports.ErrAutoridadAdministracionPerfilesNoDisponible
			}
			// jsonb serializa timestamptz como «…+00:00»; el dominio exige UTC canónico.
			r.CaducaEn = r.CaducaEn.UTC()
			r.Propuesta = domain.PropuestaGobiernoPerfil{Material: m, HuellaSHA256: r.HuellaSHA256, CaducaEn: r.CaducaEn}
			if (ports.ResultadoPropuestaGobiernoRolNuevo{Propuesta: r.Propuesta,
				Replay: r.Replay, AuditoriaAccesoRef: r.AuditoriaAccesoRef}).ValidarPara(o, a.reloj.Ahora()) != nil {
				return ports.ErrAutoridadAdministracionPerfilesNoDisponible
			}
			return nil
		})
	if err != nil {
		return vacia, err
	}
	r.Propuesta.Material.Plan = r.Propuesta.Material.Plan.Copia()
	return ports.ResultadoPropuestaGobiernoRolNuevo{Propuesta: r.Propuesta,
		Replay: r.Replay, AuditoriaAccesoRef: r.AuditoriaAccesoRef}, nil
}

func (a *AutoridadGobiernoRolNuevo) CerrarGobiernoPerfil(ctx context.Context,
	s domain.SolicitudCierreGobiernoPerfil) (domain.CierreGobiernoPerfil, error) {
	var vacio domain.CierreGobiernoPerfil
	if err := a.disponibleGobierno(ctx); err != nil {
		return vacio, err
	}
	e, err := materialCierreGobiernoRol(s)
	if err != nil {
		return vacio, err
	}
	var r cierreGobiernoRolRespuesta
	err = a.ejecutarGobierno(ctx, s.Aprobador, s.Evidencia, s.InstantaneaAutorizacion,
		e, cerrarGobiernoRolSQL, func(b []byte) error {
			if decodificarGobiernoRol(b, &r) != nil || r.Estado != "permitido" ||
				r.Decision != domain.DecisionAprobarPropuestaPerfil || r.OperacionRef != s.OperacionRef {
				return ports.ErrAutoridadAdministracionPerfilesNoDisponible
			}
			var m domain.MaterialPropuestaGobiernoPerfil
			if decodificarGobiernoRol([]byte(r.MaterialCanon), &m) != nil {
				return ports.ErrAutoridadAdministracionPerfilesNoDisponible
			}
			// jsonb serializa timestamptz como «…+00:00»; el dominio exige UTC canónico.
			r.ConfirmadoEn = r.ConfirmadoEn.UTC()
			r.Cierre = domain.CierreGobiernoPerfil{OperacionRef: r.OperacionRef, Material: m,
				PropuestaHuellaSHA256: r.PropuestaHuellaSHA256, Decision: r.Decision,
				ConfirmadoEn: r.ConfirmadoEn, AuditoriaAccesoRef: r.AuditoriaAccesoRef,
				Recibo: r.Recibo.Dominio()}
			if r.Cierre.ValidarPara(s) != nil || r.Cierre.ConfirmadoEn.After(a.reloj.Ahora()) {
				return ports.ErrAutoridadAdministracionPerfilesNoDisponible
			}
			return nil
		})
	if err != nil {
		return vacio, err
	}
	r.Cierre.Material.Plan = r.Cierre.Material.Plan.Copia()
	if r.Cierre.Recibo != nil {
		copia := r.Cierre.Recibo.Copia()
		r.Cierre.Recibo = &copia
	}
	return r.Cierre, nil
}

func (a *AutoridadGobiernoRolNuevo) disponibleGobierno(ctx context.Context) error {
	if ctx == nil || a == nil || ausente(a.pool) || ausente(a.fuente) || ausente(a.emisor) || ausente(a.reloj) {
		return ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	return ctx.Err()
}

func (a *AutoridadGobiernoRolNuevo) ejecutarGobierno(ctx context.Context, actor domain.ContextoActor,
	evidencia domain.EvidenciaSesionAdministracionPerfiles, instantanea domain.InstantaneaAutorizacion,
	e Efecto, consulta string, validar func([]byte) error) error {
	errNoDisponible := ports.ErrAutoridadAdministracionPerfilesNoDisponible
	if err := a.disponibleGobierno(ctx); err != nil {
		return err
	}
	if validar == nil || evidencia.ValidarEn(actor, a.reloj.Ahora()) != nil ||
		!domain.ReferenciaCorrelacionAutorizacionV2Valida(e.CorrelacionAccesoRef) {
		return errNoDisponible
	}
	recurso, err := RecursoGobiernoRolNuevo(e, instantanea.AsignacionPerfil)
	if err != nil {
		return err
	}
	// El emisor recibe una copia inmutable del material ligado a la fachada.
	entrega := e
	entrega.Material = append([]byte(nil), e.Material...)
	m, err := a.emisor.EmitirGobiernoRolNuevo(ctx, actor, evidencia, instantanea, entrega)
	clear(entrega.Material)
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
	r := m.ResumenCapacidad()
	ahora := a.reloj.Ahora()
	if m.ValidarEstructura() != nil || r.Operacion() != e.Accion || r.AudienciaConsumo() != e.Audiencia ||
		r.EfectoRef() != recurso.Referencia || r.EfectoHuellaSHA256() != h ||
		r.ContextoRef() != evidencia.ResultadoContexto.RegistroContextoRef ||
		r.ContextoHuellaSHA256() != evidencia.ResultadoContexto.HuellaSHA256 ||
		m.PersonaVersion() != actor.Instantanea.PersonaVersion || m.PerfilVersion() != actor.Instantanea.PerfilVersion ||
		ahora.Before(r.EmitidaEn()) || !ahora.Before(r.ExpiraEn()) {
		return errNoDisponible
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
		return errNoDisponible
	}
	defer func() {
		c, cancel := context.WithTimeout(context.Background(), plazoarranque.Ampliar(2*time.Second))
		defer cancel()
		_ = tx.Rollback(c)
	}()
	// El límite debe estar activo ANTES de enviar el SELECT: el SET de la
	// función SQL no rearma el temporizador de la sentencia que ya empezó.
	for _, configuracion := range []string{
		"SET LOCAL statement_timeout = '15s'",
		"SET LOCAL lock_timeout = '2s'",
		"SET LOCAL TimeZone = 'UTC'",
	} {
		if _, err := tx.Exec(ctx, configuracion); err != nil {
			return errNoDisponible
		}
	}
	var b []byte
	if err := tx.QueryRow(ctx, consulta, args...).Scan(&b); err != nil {
		return traducirGobiernoRol(ctx, err)
	}
	// AUT60 devuelve un intento denegado/error sólo después de registrar su
	// auditoría común. Ese resultado se confirma antes de devolver el error;
	// un fallo de auditoría llega como SQL error y revierte toda la transacción.
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
			(intento.Estado == "denegado" && intento.Codigo != "gobierno_rol_nuevo_denegado") ||
			(intento.Estado == "error" && intento.Codigo != "gobierno_rol_nuevo_error") {
			return errNoDisponible
		}
		if intento.Estado == "denegado" {
			errorIntento = domain.ErrAutorizacionDenegada
		} else {
			errorIntento = errNoDisponible
		}
	} else if err := validar(b); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		// Un COMMIT indeterminado se recupera mediante la misma operación y
		// autorización nueva, nunca entregando un recibo provisional.
		return errNoDisponible
	}
	if errorIntento != nil {
		return errors.Join(ports.ErrGobiernoRolIntentoAuditado, errorIntento)
	}
	return nil
}

func traducirGobiernoRol(ctx context.Context, _ error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	return ports.ErrAutoridadAdministracionPerfilesNoDisponible
}

type propuestaGobiernoRolRespuesta struct {
	Estado             string                         `json:"estado,omitempty"`
	Replay             bool                           `json:"replay,omitempty"`
	MaterialCanon      string                         `json:"material_canon"`
	HuellaSHA256       string                         `json:"huella_sha256"`
	CaducaEn           time.Time                      `json:"caduca_en"`
	AuditoriaAccesoRef string                         `json:"auditoria_acceso_ref"`
	Propuesta          domain.PropuestaGobiernoPerfil `json:"-"`
}

type cierreGobiernoRolRespuesta struct {
	Estado                string                                         `json:"estado,omitempty"`
	Replay                bool                                           `json:"replay,omitempty"`
	OperacionRef          string                                         `json:"operacion_ref"`
	MaterialCanon         string                                         `json:"material_canon"`
	PropuestaHuellaSHA256 string                                         `json:"propuesta_huella_sha256"`
	Decision              domain.DecisionPropuestaAdministracionPerfiles `json:"decision"`
	ConfirmadoEn          time.Time                                      `json:"confirmado_en"`
	AuditoriaAccesoRef    string                                         `json:"auditoria_acceso_ref"`
	Recibo                reciboGobiernoRolRespuesta                     `json:"recibo"`
	Cierre                domain.CierreGobiernoPerfil                    `json:"-"`
}

type reciboGobiernoRolRespuesta struct {
	ActoRef             string                           `json:"acto_ref"`
	ReciboRef           string                           `json:"recibo_ref"`
	ActorPersonaRef     string                           `json:"actor_persona_ref"`
	PerfilActivoRef     string                           `json:"perfil_activo_ref"`
	AsignacionPerfilRef string                           `json:"asignacion_perfil_ref"`
	CorrelacionRef      string                           `json:"correlacion_ref"`
	Motivo              domain.ReferenciaEntradaCatalogo `json:"motivo"`
	AuditoriaRef        string                           `json:"auditoria_ref"`
	VersionRol          domain.VersionRol                `json:"version_rol"`
	ControlPosterior    domain.ControlVigenciaVersionRol `json:"control_posterior"`
}

func (r reciboGobiernoRolRespuesta) Dominio() *domain.ReciboGobiernoPerfil {
	return &domain.ReciboGobiernoPerfil{ActoRef: r.ActoRef, ReciboRef: r.ReciboRef,
		ActorPersonaRef: r.ActorPersonaRef, PerfilActivoRef: r.PerfilActivoRef,
		AsignacionPerfilRef: r.AsignacionPerfilRef, CorrelacionRef: r.CorrelacionRef,
		Motivo: r.Motivo, AuditoriaRef: r.AuditoriaRef, VersionRol: r.VersionRol,
		ControlPosterior: r.ControlPosterior}
}
