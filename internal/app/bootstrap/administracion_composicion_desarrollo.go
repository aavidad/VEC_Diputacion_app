package bootstrap

import (
	"context"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/config"
	adminpg "vec-diputacion-granada/internal/modules/administracion/adapters/postgres"
	adminapp "vec-diputacion-granada/internal/modules/administracion/application"
	postgrescontexto "vec-diputacion-granada/internal/vec/adapters/contextoactor/postgres"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	postgresidentidad "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	aplicacionvec "vec-diputacion-granada/internal/vec/application"
)

// nuevaRutaAdministracionCorreoDesarrollo compone exclusivamente la vertical
// ADMIN. La ausencia o parcialidad de sus cinco DSN niega la ruta completa.
func nuevaRutaAdministracionCorreoDesarrollo(ctx context.Context, cfg config.Config, c *ComposicionSeguridadDesarrollo, resolvedor *resolvedorIdentidadDesarrollo) (ruta *vechttp.RutaExacta, autoridad *autoridadAdministracionDesarrollo, cerrar func(), err error) {
	if !cfg.AdministracionPostgreSQL.Configurada() && (resolvedor == nil || resolvedor.administracion == nil) {
		return nil, nil, func() {}, nil
	}
	if ctx == nil || ctx.Err() != nil || c == nil || resolvedor == nil || c.protectorCorreoAdministracion == nil || cfg.AdministracionPostgreSQL.Validar() != nil {
		return nil, nil, nil, ErrConfiguracionCorreoAdministracionNoDisponible
	}
	correo, _ := cfg.AdministracionPostgreSQL.DSNCorreo()
	registroDSN, _ := cfg.AdministracionPostgreSQL.DSNRegistroAutorizacion()
	identidadDSN, revalDSN, contextoDSN, _ := cfg.AdministracionPostgreSQL.DSNIdentidad()
	poolCorreo, _, e := abrirPoolCorreoAdministracionDesarrollo(ctx, correo)
	if e != nil {
		return nil, nil, nil, ErrConfiguracionCorreoAdministracionNoDisponible
	}
	pools := []*pgxpool.Pool{poolCorreo}
	var una sync.Once
	cerrarEmisor := func() {}
	liberar := func() {
		una.Do(func() {
			cerrarEmisor()
			for _, p := range pools {
				p.Close()
			}
		})
	}
	ok := false
	defer func() {
		if !ok {
			liberar()
		}
	}()
	abrir := func(dsn, nombre, rol string) (*pgxpool.Pool, error) {
		p, _, x := abrirPoolPostgreSQLContratacionTemporalDesarrollo(ctx, dsn, nombre, rol)
		if x == nil {
			pools = append(pools, p)
		}
		return p, x
	}
	pRegistro, e := abrir(registroDSN, "vec-admin-autorizacion", "vec_autorizacion_registro")
	if e != nil {
		return nil, nil, nil, ErrConfiguracionCorreoAdministracionNoDisponible
	}
	pIdentidad, e := abrir(identidadDSN, "vec-admin-identidad", "vec_identidad_sesiones_v1_registrador")
	if e != nil {
		return nil, nil, nil, ErrConfiguracionCorreoAdministracionNoDisponible
	}
	pReval, e := abrir(revalDSN, "vec-admin-revalidacion", "vec_identidad_sesiones_v1_revalidador")
	if e != nil {
		return nil, nil, nil, ErrConfiguracionCorreoAdministracionNoDisponible
	}
	pContexto, e := abrir(contextoDSN, "vec-admin-contexto", "vec_contexto_actor_v1_runtime")
	if e != nil {
		return nil, nil, nil, ErrConfiguracionCorreoAdministracionNoDisponible
	}
	reloj := relojContratacionTemporalDesarrollo{}
	deps, e := nuevasDependenciasEmisorAdministracionDesarrollo(ctx, cfg, resolvedor.administracion, pRegistro, reloj)
	if e != nil || deps == nil {
		return nil, nil, nil, ErrConfiguracionCorreoAdministracionNoDisponible
	}
	cerrarEmisor = deps.cerrar
	seud := &seudonimizadorSesionDesarrollo{derivador: c.derivadorIdempotencia}
	sesiones, e := postgresidentidad.NuevoRegistroSesionesPostgreSQL(ctx, pIdentidad, pReval, seud, espacioIdentidadSesionDesarrollo, dominioIdentidadSesionDesarrollo)
	if e != nil {
		return nil, nil, nil, ErrConfiguracionCorreoAdministracionNoDisponible
	}
	reval, e := postgresidentidad.NuevoRevalidadorAutenticacionActorPostgreSQL(ctx, pReval)
	if e != nil {
		return nil, nil, nil, ErrConfiguracionCorreoAdministracionNoDisponible
	}
	resolutor, e := postgrescontexto.NuevoResolutorRegistroContextoActorPostgreSQLV2(ctx, pContexto)
	if e != nil {
		return nil, nil, nil, ErrConfiguracionCorreoAdministracionNoDisponible
	}
	servicioContexto, e := aplicacionvec.NuevoServicioContextoActorProductivoV2(resolutor, postgrescontexto.NuevoGeneradorOperacionContextoActorV2Criptografico(), reloj)
	if e != nil {
		return nil, nil, nil, ErrConfiguracionCorreoAdministracionNoDisponible
	}
	contextoActor, e := aplicacionvec.NuevaAutoridadContextoActorRegistradoV2(servicioContexto)
	if e != nil {
		return nil, nil, nil, ErrConfiguracionCorreoAdministracionNoDisponible
	}
	permiso, e := nuevaFuentePermisoAdministracionGobernadaV3(deps.pdp, deps.motivo, seguridadvec.GeneradorReferenciasCriptograficas{}, reloj)
	if e != nil {
		return nil, nil, nil, ErrConfiguracionCorreoAdministracionNoDisponible
	}
	sesion, e := nuevoProveedorSesionDurableAdministracionCorreoV3(sesiones, reval, contextoActor, permiso, reloj)
	if e != nil {
		return nil, nil, nil, ErrConfiguracionCorreoAdministracionNoDisponible
	}
	autoridad, e = nuevaAutoridadAdministracionDesarrollo(cfg, resolvedor, sesion)
	if e != nil {
		return nil, nil, nil, e
	}
	seudonimizadorAuditoria, e := nuevoSeudonimizadorAuditoriaAdministracionDesarrollo(cfg)
	if e != nil {
		return nil, nil, nil, ErrConfiguracionCorreoAdministracionNoDisponible
	}
	preparadorAuditoria, e := nuevaPreparacionAuditoriaAdministracionDesarrollo(sesion, seudonimizadorAuditoria, seguridadvec.GeneradorReferenciasCriptograficas{}, reloj)
	if e != nil {
		return nil, nil, nil, ErrConfiguracionCorreoAdministracionNoDisponible
	}
	autorizador, e := nuevoProveedorAutorizacionConfiguracionCorreoV3(sesion, preparadorAuditoria, deps.emisor, deps.motivo, seguridadvec.GeneradorReferenciasCriptograficas{}, reloj)
	if e != nil {
		return nil, nil, nil, ErrConfiguracionCorreoAdministracionNoDisponible
	}
	registroCorreo, e := adminpg.NuevoRegistroConfiguracionCorreoPostgreSQL(poolCorreo, c.protectorCorreoAdministracion)
	if e != nil {
		return nil, nil, nil, ErrConfiguracionCorreoAdministracionNoDisponible
	}
	servicio, e := adminapp.NuevoServicioConfiguracionCorreo(autoridad, preparadorAuditoria, registroCorreo, autorizador, registroCorreo)
	if e != nil {
		return nil, nil, nil, ErrConfiguracionCorreoAdministracionNoDisponible
	}
	autorizadorConsulta, e := nuevoAutorizadorConsultaConfiguracionCorreoV3(sesion, preparadorAuditoria, deps.emisorConsulta, deps.motivo, seguridadvec.GeneradorReferenciasCriptograficas{}, reloj)
	if e != nil {
		return nil, nil, nil, ErrConfiguracionCorreoAdministracionNoDisponible
	}
	registroConsulta, e := adminpg.NuevaConsultaConfiguracionCorreoPostgreSQL(poolCorreo)
	if e != nil {
		return nil, nil, nil, ErrConfiguracionCorreoAdministracionNoDisponible
	}
	consulta, e := adminapp.NuevoServicioConsultaConfiguracionCorreo(autoridad, preparadorAuditoria, autorizadorConsulta, registroConsulta)
	if e != nil {
		return nil, nil, nil, ErrConfiguracionCorreoAdministracionNoDisponible
	}
	r, e := NuevaRutaConfiguracionCorreoAdministracion(autoridad, &servicioConfiguracionCorreoCompuesto{consulta: consulta, cambio: servicio})
	if e != nil {
		return nil, nil, nil, e
	}
	ok = true
	return &r, autoridad, liberar, nil
}

// El pool ADMIN verifica TLS y roles técnicos antes de entregar conexiones.
func abrirPoolCorreoAdministracionDesarrollo(ctx context.Context, dsn string) (*pgxpool.Pool, string, error) {
	fallo := ErrConfiguracionCorreoAdministracionNoDisponible
	if ctx == nil || ctx.Err() != nil {
		return nil, "", fallo
	}
	pc, e := pgxpool.ParseConfig(dsn)
	if e != nil || pc == nil || pc.ConnConfig == nil || validarTLSPostgreSQLBorradores(&pc.ConnConfig.Config, true) != nil {
		return nil, "", fallo
	}
	pc.MaxConns = 4
	pc.MinConns = 0
	pc.ConnConfig.ConnectTimeout = 5 * time.Second
	if pc.ConnConfig.RuntimeParams == nil {
		pc.ConnConfig.RuntimeParams = map[string]string{}
	}
	for k, v := range map[string]string{"application_name": "vec-admin-correo", "timezone": "UTC", "search_path": "pg_catalog", "default_transaction_isolation": "serializable", "default_transaction_read_only": "off", "statement_timeout": "15s", "lock_timeout": "3s", "idle_in_transaction_session_timeout": "20s"} {
		pc.ConnConfig.RuntimeParams[k] = v
	}
	pool, e := pgxpool.NewWithConfig(ctx, pc)
	if e != nil {
		return nil, "", fallo
	}
	var usuario string
	var valido bool
	e = pool.QueryRow(ctx, `SELECT session_user::text, session_user=current_user AND r.rolcanlogin AND r.rolinherit AND NOT r.rolsuper AND NOT r.rolcreatedb AND NOT r.rolcreaterole AND NOT r.rolreplication AND NOT r.rolbypassrls AND pg_has_role(session_user,'vec_administracion_ejecutor','MEMBER') AND NOT EXISTS (SELECT 1 FROM pg_roles x WHERE x.rolname IN ('vec_administracion_propietario','vec_administracion_migrador','vec_contratacion_temporal_ejecutor','vec_contratacion_temporal_propietario','vec_contratacion_temporal_migrador') AND pg_has_role(session_user,x.oid,'MEMBER')) FROM pg_roles r WHERE r.rolname=session_user`).Scan(&usuario, &valido)
	if e != nil || !valido || usuario == "" {
		pool.Close()
		return nil, "", fallo
	}
	return pool, usuario, nil
}
