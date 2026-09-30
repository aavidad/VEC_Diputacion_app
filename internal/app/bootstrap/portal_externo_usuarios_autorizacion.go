package bootstrap

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	vecpg "vec-diputacion-granada/internal/vec/adapters/postgres"
	"vec-diputacion-granada/internal/vec/adapters/seguridad"
	vecapp "vec-diputacion-granada/internal/vec/application"
)

// Los tres grupos AUT-17 son exclusivos de Usuarios externo. La composición
// debe abrir tres LOGIN distintos y cotejar su topología con el preflight.
const (
	rolFuenteUsuariosExterno     = "vec_autorizacion_fuente_usuarios_externa"
	rolRegistroUsuariosExterno   = "vec_autorizacion_registro_usuarios_externo"
	rolMotivosUsuariosExterno    = "vec_autorizacion_motivos_usuarios_externos"
	loginFuenteUsuariosExterno   = "vec_externo_usuarios_v3_fuente_autorizacion_desarrollo"
	loginRegistroUsuariosExterno = "vec_externo_usuarios_v3_registro_autorizacion_desarrollo"
	loginMotivosUsuariosExterno  = "vec_externo_usuarios_v3_motivos_autorizacion_desarrollo"
)

func abrirPoolAutorizacionUsuariosExterno(ctx context.Context, dsn, grupo, loginEsperado string, topologia topologiaPostgreSQLPreferenciasUsuarios) (*pgxpool.Pool, error) {
	logins := map[string]string{rolFuenteUsuariosExterno: loginFuenteUsuariosExterno,
		rolRegistroUsuariosExterno: loginRegistroUsuariosExterno,
		rolMotivosUsuariosExterno:  loginMotivosUsuariosExterno}
	if ctx == nil || dsn == "" || loginEsperado == "" || logins[grupo] != loginEsperado {
		return nil, ErrUsuariosPortalExternoNoDisponible
	}
	c, err := pgxpool.ParseConfig(dsn)
	if err != nil || c.ConnConfig.User != loginEsperado || validarTLSPostgreSQLBorradores(&c.ConnConfig.Config, true) != nil {
		return nil, ErrUsuariosPortalExternoNoDisponible
	}
	c.MaxConns, c.MinConns = 2, 0
	c.ConnConfig.ConnectTimeout = 5 * time.Second
	if c.ConnConfig.RuntimeParams == nil {
		c.ConnConfig.RuntimeParams = map[string]string{}
	}
	for k, v := range map[string]string{"application_name": "vec-usuarios-externo-autorizacion", "timezone": "UTC", "search_path": "pg_catalog", "statement_timeout": "10s", "lock_timeout": "2s", "idle_in_transaction_session_timeout": "15s"} {
		c.ConnConfig.RuntimeParams[k] = v
	}
	pool, err := pgxpool.NewWithConfig(ctx, c)
	if err != nil {
		return nil, ErrUsuariosPortalExternoNoDisponible
	}
	var valido bool
	err = pool.QueryRow(ctx, `SELECT session_user=current_user AND session_user=$1
		AND r.rolcanlogin AND r.rolinherit AND NOT (r.rolsuper OR r.rolcreatedb OR r.rolcreaterole OR r.rolreplication OR r.rolbypassrls)
		AND (SELECT count(*)=1 FROM pg_catalog.pg_auth_members m WHERE m.member=r.oid)
		AND EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m JOIN pg_catalog.pg_roles g ON g.oid=m.roleid
		 WHERE m.member=r.oid AND g.rolname=$2 AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option
		 AND g.rolinherit AND NOT (g.rolcanlogin OR g.rolsuper OR g.rolcreatedb OR g.rolcreaterole OR g.rolreplication OR g.rolbypassrls)
		 AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members superior WHERE superior.member=g.oid))
		FROM pg_catalog.pg_roles r WHERE r.rolname=session_user`, loginEsperado, grupo).Scan(&valido)
	if err != nil || !valido || cotejarTopologiaPostgreSQLPreferenciasUsuarios(ctx, pool, topologia) != nil {
		pool.Close()
		return nil, ErrUsuariosPortalExternoNoDisponible
	}
	return pool, nil
}

// nuevoServicioAutorizacionUsuariosExterno exige las fachadas AUT-17 y
// construye el PDP V3 sin posibilidad de llamar al almacenamiento interno.
func nuevoServicioAutorizacionUsuariosExterno(ctx context.Context, fuentePool, registroPool, motivosPool *pgxpool.Pool, catalogoID string) (*vecapp.ServicioAutorizacionSolicitudLigadaV3, error) {
	if ctx == nil || fuentePool == nil || registroPool == nil || motivosPool == nil ||
		fuentePool == registroPool || fuentePool == motivosPool || registroPool == motivosPool {
		return nil, ErrUsuariosPortalExternoNoDisponible
	}
	sondas := []struct {
		pool   *pgxpool.Pool
		nombre string
		login  string
	}{
		{fuentePool, "obtener_instantanea_usuarios_externo_v1", loginFuenteUsuariosExterno},
		{registroPool, "registrar_decision_usuarios_externo_v3", loginRegistroUsuariosExterno},
		{motivosPool, "resolver_motivo_usuarios_externo_v1", loginMotivosUsuariosExterno},
	}
	for _, s := range sondas {
		if acreditarFachadaAutorizacionUsuariosExterno(ctx, s.pool, s.nombre, s.login) != nil {
			return nil, ErrUsuariosPortalExternoNoDisponible
		}
	}
	fuente, err := vecpg.NuevoAlmacenAutorizacionUsuariosExterno(fuentePool)
	if err != nil {
		return nil, ErrUsuariosPortalExternoNoDisponible
	}
	registro, err := vecpg.NuevoAlmacenAutorizacionUsuariosExterno(registroPool)
	if err != nil {
		return nil, ErrUsuariosPortalExternoNoDisponible
	}
	motivos, err := vecpg.NuevoValidadorMotivosUsuariosExterno(motivosPool, catalogoID)
	if err != nil {
		return nil, ErrUsuariosPortalExternoNoDisponible
	}
	servicio, err := vecapp.NuevoServicioAutorizacionSolicitudLigadaV3(fuente, registro, registro, motivos,
		relojRutasDietas{}, seguridad.GeneradorReferenciasCriptograficas{},
		vecapp.ConfiguracionServicioAutorizacion{VigenciaDecision: 30 * time.Second})
	if err != nil {
		return nil, ErrUsuariosPortalExternoNoDisponible
	}
	return servicio, nil
}

// La sonda valida la fachada nominal y la configuración final de AUT-21.
// Incluir pg_temp al final evita que sus tipos precedan a pg_catalog.
type consultaFachadaUsuariosExterno interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

const sondaFachadaAutorizacionUsuariosExternoSQL = `WITH candidata AS (
 SELECT p.proconfig FROM pg_catalog.pg_proc p
 JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
 WHERE n.nspname='vec_autorizacion' AND p.proname=$1 AND p.prokind='f'
 AND p.proowner='vec_autorizacion_propietario'::regrole AND p.prosecdef
 AND has_function_privilege(session_user,p.oid,'EXECUTE')
)
 SELECT session_user=current_user AND session_user=$2
 AND has_schema_privilege(session_user,'vec_autorizacion','USAGE')
 AND (SELECT count(*)=1 FROM candidata),
 (SELECT proconfig FROM candidata LIMIT 1)`

func acreditarFachadaAutorizacionUsuariosExterno(ctx context.Context, consulta consultaFachadaUsuariosExterno, nombre, login string) error {
	if ctx == nil || ctx.Err() != nil || consulta == nil {
		return ErrUsuariosPortalExternoNoDisponible
	}
	var nominal bool
	var configuracion []string
	err := consulta.QueryRow(ctx, sondaFachadaAutorizacionUsuariosExternoSQL, nombre, login).Scan(&nominal, &configuracion)
	if err != nil || !nominal || len(configuracion) != 1 || configuracion[0] != "search_path=pg_catalog, pg_temp" || ctx.Err() != nil {
		return ErrUsuariosPortalExternoNoDisponible
	}
	return nil
}
