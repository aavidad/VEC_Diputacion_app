package bootstrap

import (
	"context"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	aplicacionvec "vec-diputacion-granada/internal/vec/application"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type dependenciasConsultasRRHHDesarrollo struct {
	emisorCuadro    *emisorMaterialRenovableCTDesarrollo
	materialDetalle *proveedorMaterialAltaContratacionTemporalDesarrollo
	sesion          ports.SesionConsultaRRHH
	// La firma VEC reutiliza la fachada histórica v7 con otra sesión nominal.
	sesionOriginal        ports.SesionConsultaRRHH
	motivos               ports.ResolutorMotivoConsultaRRHH
	identidad             *proveedorSesionConsultaRRHHDesarrollo
	autoridad             *autoridadConsultasRRHHDesarrollo
	cuadro                httpinterno.ConsultorCuadroRRHH
	detalle               httpinterno.ConsultorDetalleRRHH
	originalPropuesta     httpinterno.ConsultorDetalleRRHH
	cuadroHTTP            httpinterno.ConsultorCuadroRRHH
	detalleHTTP           httpinterno.ConsultorDetalleRRHH
	originalPropuestaHTTP httpinterno.ConsultorDetalleRRHH
	preparacionResolucion ports.ConsultorPreparacionResolucionFormalizacion
	estadisticas          ports.ConsultorEstadisticasRRHH
	plazosFase            ports.CalculadoraPlazoFaseRRHH
	// descargas autoriza y registra la descarga de borradores de la ruta de
	// detalle (AD199/CT177); con lectores, la del lector de la petición.
	descargas    ports.RegistradorDescargaBorradorRRHH
	repoDescarga ports.RepositorioDescargaBorradorRRHH
	cerrar       func()
}

func nuevasDependenciasLectoresRRHHDesarrollo(
	ctx context.Context, cfg config.Config, alta *dependenciasAltaContratacionTemporalDesarrollo,
	derivador *derivadorIdentidadOperacionDesarrollo, reloj relojContratacionTemporalDesarrollo,
	lectores []identidadConsultaRRHHDesarrollo, poolConsultas *postgresct.PoolConsultasRRHHPostgreSQL,
	motivos ports.ResolutorMotivoConsultaRRHH, motivoCuadro, motivoDetalle dominiovec.ReferenciaEntradaCatalogo,
	base dependenciasConsultasRRHHDesarrollo,
	fronteras catalogoFronterasComunDesarrollo,
	documentalMontado bool,
) (dependenciasConsultasRRHHDesarrollo, error) {
	vacio := dependenciasConsultasRRHHDesarrollo{}
	if len(lectores) == 0 || poolConsultas == nil || motivos == nil || base.cerrar == nil {
		return vacio, registrarFalloMontajeCTDesarrollo("lectores_rrhh_dependencias", ports.ErrConsultaRRHHNoDisponible)
	}
	mux := nuevoMultiplexoresLectoresRRHHDesarrollo(alta.soporte.sello)
	cierres := []func(){base.cerrar}
	completo := false
	defer func() {
		if !completo {
			for _, cerrar := range cierres {
				cerrar()
			}
		}
	}()
	for _, lector := range lectores {
		// La técnica usa el soporte de Alta y su perfil fijo de lecturas,
		// compuesto en base. El manifiesto liga su certificado al perfil de
		// Alta, pero la ruta selecciona la sesión del perfil de lecturas.
		if lectorTecnicoRRHHDesarrollo(lector) {
			if !lectorTecnicoEsTitularAltaDesarrollo(alta, lector) {
				return vacio, registrarFalloMontajeCTDesarrollo("lector_tecnico_titular_alta", ports.ErrConsultaRRHHNoDisponible)
			}
			if perfilFijoConsultasTecnicoCTDesarrollo(alta.soporte) == nil {
				return vacio, registrarFalloMontajeCTDesarrollo("lector_tecnico_perfil_consultas", ports.ErrConsultaRRHHNoDisponible)
			}
			if !mux.registrar(lector.identidad.principal.ID, alta.soporte, base.cuadroHTTP, base.detalleHTTP, base.originalPropuestaHTTP) {
				return vacio, registrarFalloMontajeCTDesarrollo("lector_tecnico_registrar_mux", ports.ErrConsultaRRHHNoDisponible)
			}
			mux.descargas[lector.identidad.principal.ID] = base.descargas
			continue
		}
		soporte, autorizador, err := nuevoSoporteLectorRRHHDesarrollo(alta, lector, reloj)
		if err != nil {
			return vacio, err
		}
		if err = publicarContextoPostgreSQLContratacionTemporalDesarrollo(ctx, alta.postgresql.gobierno, soporte); err != nil {
			return vacio, err
		}
		local := &dependenciasAltaContratacionTemporalDesarrollo{soporte: soporte, autorizador: autorizador, postgresql: alta.postgresql}
		identidad, cerrarIdentidad, err := nuevasDependenciasIdentidadConsultasDesarrollo(ctx, cfg.ContratacionTemporalPostgreSQL, local, derivador, reloj, soporte, fronteras)
		if err != nil {
			return vacio, err
		}
		cierres = append(cierres, cerrarIdentidad)
		autoridad, err := configurarAutoridadConsultasRRHHDesarrolloConAmbito(local, reloj, motivoCuadro, motivoDetalle, lector.clase, lector.ambitoRef)
		if err != nil {
			return vacio, err
		}
		if err = autoridad.configurarProveedorContextoConsultaRRHHDesarrollo(identidad); err != nil {
			return vacio, err
		}
		if err = componerPerfilFijoLectorRRHHDesarrollo(ctx, alta.postgresql.gobierno, soporte, aprobacionProvisionPerfilesRRHHDesdeConfig(cfg)); err != nil {
			return vacio, err
		}
		material, err := nuevoMaterialAtestacionContratacionTemporalDesarrollo(derivador, reloj.Ahora())
		if err != nil {
			return vacio, err
		}
		material.fuenteConfianza = alta.postgresql.proveedorMaterial.fuenteConfianza
		proveedorCuadro, err := nuevoProveedorMaterialConsumidorDesarrollo(ctx, alta.postgresql.gobierno, material, soporte, reloj, alta.postgresql.catalogoMaterial, ports.AudienciaConsumoConsultaCuadroRRHHV3)
		if err != nil {
			material.borrarCopiasEfimeras()
			return vacio, err
		}
		proveedorDetalle, err := nuevoProveedorMaterialConsumidorDesarrollo(ctx, alta.postgresql.gobierno, material, soporte, reloj, alta.postgresql.catalogoMaterial, ports.AudienciaConsumoConsultaDetalleRRHHV3)
		if err != nil {
			material.borrarCopiasEfimeras()
			return vacio, err
		}
		proveedorDescarga, err := nuevoProveedorMaterialConsumidorDesarrollo(ctx, alta.postgresql.gobierno, material, soporte, reloj, alta.postgresql.catalogoMaterial, ports.AudienciaConsumoDescargaBorradorRRHHV3)
		material.borrarCopiasEfimeras()
		if err != nil {
			return vacio, err
		}
		repoDescarga, err := postgresct.NuevoRepositorioDescargaBorradorRRHHPostgreSQL(poolConsultas)
		if err != nil {
			return vacio, err
		}
		descarga, err := nuevoRegistradorDescargaBorradorRRHHDesarrollo(autoridad, proveedorDescarga, repoDescarga, motivoDetalle,
			alta.auditoriaLecturasCT, alta.procesoAuditoriaLecturasCT)
		if err != nil {
			return vacio, err
		}
		emisorCuadro, err := nuevoEmisorMaterialRenovableCTDesarrollo(autoridad, proveedorCuadro)
		if err != nil {
			return vacio, err
		}
		emisorDetalle, err := nuevoEmisorMaterialRenovableCTDesarrollo(autoridad, proveedorDetalle)
		if err != nil {
			return vacio, err
		}
		emisor, err := ports.NuevoEmisorMaterialConsultaRRHH(motivos, seguridadvec.GeneradorReferenciasCriptograficas{}, reloj, emisorCuadro, emisorDetalle)
		if err != nil {
			return vacio, err
		}
		sesion, err := postgresct.NuevaSesionConsultaRRHHPostgreSQL(poolConsultas)
		if err != nil {
			return vacio, err
		}
		cuadro, err := application.NuevoServicioConsultaCuadroRRHH(autoridad, emisor, sesion, reloj)
		if err != nil {
			return vacio, err
		}
		cuadro.ConfigurarPlazosFase(base.plazosFase)
		continuador, err := nuevoContinuadorSesionCursorRRHHDesarrollo(autoridad, reloj)
		if err != nil {
			return vacio, err
		}
		cuadroHTTP, err := nuevoConsultorCuadroConSesionCursorRRHHDesarrollo(cuadro, continuador)
		if err != nil {
			return vacio, err
		}
		detalle, err := application.NuevoServicioConsultaDetalleRRHH(autoridad, emisor, sesion, reloj)
		if err != nil {
			return vacio, err
		}
		sesionOriginal, err := postgresct.NuevaSesionConsultaOriginalPropuestaRRHHPostgreSQL(poolConsultas)
		if err != nil {
			return vacio, err
		}
		originalPropuesta, err := application.NuevoServicioConsultaDetalleRRHH(autoridad, emisor, sesionOriginal, reloj)
		if err != nil {
			return vacio, registrarFalloMontajeCTDesarrollo("lector_rrhh_original_propuesta", ports.ErrConsultaRRHHNoDisponible)
		}
		if !mux.registrar(lector.identidad.principal.ID, soporte, cuadroHTTP,
			consultorDetalleConDisponibilidadRRHH{lector: detalle, documentalMontado: documentalMontado}, originalPropuesta) {
			return vacio, registrarFalloMontajeCTDesarrollo("lector_rrhh_registrar_mux", ports.ErrConsultaRRHHNoDisponible)
		}
		mux.descargas[lector.identidad.principal.ID] = descarga
	}
	completo = true
	var unaVez sync.Once
	return dependenciasConsultasRRHHDesarrollo{cuadroHTTP: mux, detalleHTTP: multiplexorDetalleLectoresRRHHDesarrollo{mux}, originalPropuestaHTTP: multiplexorOriginalPropuestaLectoresRRHHDesarrollo{mux}, descargas: multiplexorDescargasLectoresRRHHDesarrollo{mux}, cerrar: func() {
		unaVez.Do(func() {
			for _, cerrar := range cierres {
				cerrar()
			}
		})
	}}, nil
}

// perfilesLectoresConsultaRRHHDesarrollo son los perfiles que las fronteras
// de consulta admiten además del perfil propio de la técnica.
func perfilesLectoresConsultaRRHHDesarrollo(lectores []identidadConsultaRRHHDesarrollo) []string {
	perfiles := make([]string, 0, len(lectores))
	for _, lector := range lectores {
		if !lectorTecnicoRRHHDesarrollo(lector) {
			perfiles = append(perfiles, lector.perfilRef)
		}
	}
	return perfiles
}

func lectorTecnicoRRHHDesarrollo(lector identidadConsultaRRHHDesarrollo) bool {
	roles := lector.identidad.principal.Roles
	return len(roles) == 1 && roles[0] == rolTecnicoRRHHContratacionTemporalDesarrollo
}

// lectorTecnicoEsTitularAltaDesarrollo admite reutilizar el circuito del alta
// sólo si la entrada es exactamente la persona y el certificado de ese
// soporte, con el ámbito de organización que ese circuito aplica. Cualquier
// otro ámbito declarado deniega: el circuito base no sabría restringirlo.
func lectorTecnicoEsTitularAltaDesarrollo(alta *dependenciasAltaContratacionTemporalDesarrollo, lector identidadConsultaRRHHDesarrollo) bool {
	if alta == nil || alta.soporte == nil || alta.soporte.lectorConsultasRRHH || !lectorTecnicoRRHHDesarrollo(lector) {
		return false
	}
	principal := lector.identidad.principal
	return principal.ID != "" && principal.ID == alta.soporte.principalID &&
		principal.Attributes["certificate_sha256"] != "" &&
		principal.Attributes["certificate_sha256"] == alta.soporte.certificadoSHA256 &&
		lector.organizacionRef == organizacionAltaContratacionTemporalDesarrollo &&
		lector.clase == ports.AmbitoOrganizacionRRHH && lector.ambitoRef == lector.organizacionRef
}

func nuevoSoporteLectorRRHHDesarrollo(
	alta *dependenciasAltaContratacionTemporalDesarrollo, lector identidadConsultaRRHHDesarrollo,
	reloj relojContratacionTemporalDesarrollo,
) (*soporteAltaContratacionTemporalDesarrollo, autorizadorLigadoContratacionTemporalDesarrollo, error) {
	if alta == nil || alta.soporte == nil || alta.postgresql.gobierno == nil || lector.identidad.principal.ID == "" || len(lector.identidad.principal.Roles) != 1 {
		return nil, nil, ports.ErrConsultaRRHHNoDisponible
	}
	// La técnica consulta con el circuito del alta y no llega aquí; si
	// llegase, se deniega en vez de abrirle otro soporte.
	if lectorTecnicoRRHHDesarrollo(lector) {
		return nil, nil, ports.ErrConsultaRRHHNoDisponible
	}
	contexto, err := nuevoContextoSinteticoContratacionTemporalDesarrollo(lector.identidad.principal, reloj.Ahora())
	if err != nil {
		return nil, nil, err
	}
	datos, err := contexto.Vinculo.Datos()
	if err != nil || datos.PerfilActivoRef != lector.perfilRef {
		return nil, nil, ports.ErrConsultaRRHHNoDisponible
	}
	alta.soporte.mu.Lock()
	registro := alta.soporte.registroDecisionesAnalisis
	alta.soporte.mu.Unlock()
	soporte := &soporteAltaContratacionTemporalDesarrollo{sello: alta.soporte.sello, principalID: lector.identidad.principal.ID, certificadoSHA256: lector.identidad.principal.Attributes["certificate_sha256"], lectorConsultasRRHH: true, organizacionConsultaRRHH: lector.organizacionRef, claseAmbitoConsultaRRHH: lector.clase, ambitoConsultaRRHH: lector.ambitoRef, contexto: contexto, reloj: reloj, concesiones: make(map[string]struct{}), instantaneasPorSolicitud: make(map[string]dominiovec.InstantaneaAutorizacion), registroDecisionesAnalisis: registro}
	soporte.autoridadAsignaciones = &autoridadPostgreSQLContratacionTemporalDesarrollo{pool: alta.postgresql.gobierno, soporte: soporte}
	base, err := aplicacionvec.NuevoServicioAutorizacionSolicitudLigadaV3(soporte, soporte, soporte, soporte, reloj, seguridadvec.GeneradorReferenciasCriptograficas{}, aplicacionvec.ConfiguracionServicioAutorizacion{VigenciaDecision: 90 * time.Second})
	if err != nil {
		return nil, nil, err
	}
	var autorizador autorizadorLigadoContratacionTemporalDesarrollo = base
	return soporte, autorizador, nil
}

// Compone las rutas, el guardián y los dos adaptadores existentes. No aplica
// migraciones ni construye otra publicación o una bandeja en memoria.
func nuevasDependenciasConsultasRRHHDesarrollo(
	dependenciasCT *DependenciasCT,
	alta *dependenciasAltaContratacionTemporalDesarrollo,
	fronteras catalogoFronterasComunDesarrollo,
) (dependenciasConsultasRRHHDesarrollo, error) {
	vacio := dependenciasConsultasRRHHDesarrollo{}
	if dependenciasCT == nil {
		return vacio, ports.ErrConsultaRRHHNoDisponible
	}
	cfg := dependenciasCT.cfg
	documentalMontado, err := plantillasDocumentalCTDesarrolloSolicitado(cfg)
	if err != nil {
		return vacio, err
	}
	resolvedor := dependenciasCT.resolvedor
	derivador := dependenciasCT.derivador
	reloj := dependenciasCT.reloj
	c := cfg.ContratacionTemporalPostgreSQL
	consultaDSN, motivosDSN, err := c.DSNConsultasRRHHSeparados()
	if err != nil || alta == nil || alta.soporte == nil || alta.postgresql.gobierno == nil || resolvedor == nil {
		return vacio, ports.ErrConsultaRRHHNoDisponible
	}
	consultaConfig, errConsulta := pgxpool.ParseConfig(consultaDSN)
	motivosConfig, errMotivos := pgxpool.ParseConfig(motivosDSN)
	if errConsulta != nil || errMotivos != nil || consultaConfig.ConnConfig.User == motivosConfig.ConnConfig.User {
		return vacio, ports.ErrConsultaRRHHNoDisponible
	}
	ejecucion, gobierno, _ := c.DSNSeparados()
	confirmador, lector, _ := c.DSNCoberturaSeparados()
	registro, _ := c.DSNRegistroAutorizacionSeparado()
	anteriores := []string{ejecucion, gobierno, confirmador, lector, registro}
	if c.BolsaLlamamientosConfigurada() {
		bolsa, err := c.DSNBolsaLlamamientosSeparado()
		if err != nil {
			return vacio, ports.ErrConsultaRRHHNoDisponible
		}
		anteriores = append(anteriores, bolsa)
	}
	for _, dsn := range anteriores {
		anterior, err := pgxpool.ParseConfig(dsn)
		if err != nil || anterior.ConnConfig.User == consultaConfig.ConnConfig.User ||
			anterior.ConnConfig.User == motivosConfig.ConnConfig.User {
			return vacio, ports.ErrConsultaRRHHNoDisponible
		}
	}
	ctx, cancelar := context.WithTimeout(context.Background(), plazoarranque.Ampliar(20*time.Second))
	defer cancelar()
	identidad, cerrarIdentidad, err := nuevasDependenciasIdentidadConsultasDesarrollo(ctx, c, alta, derivador, reloj, alta.soporte, fronteras)
	if err != nil {
		return vacio, err
	}
	identidadCompuesta := false
	defer func() {
		if !identidadCompuesta {
			cerrarIdentidad()
		}
	}()
	poolMotivos, err := postgresct.NuevoPoolResolucionMotivosRRHHPostgreSQL(ctx, motivosDSN, motivosConfig.ConnConfig.User)
	if err != nil {
		return vacio, err
	}
	poolConsultas, err := postgresct.NuevoPoolConsultasRRHHPostgreSQL(ctx, consultaDSN, consultaConfig.ConnConfig.User)
	if err != nil {
		poolMotivos.Cerrar()
		return vacio, err
	}
	var unaVez sync.Once
	cerrar := func() { unaVez.Do(func() { poolConsultas.Cerrar(); poolMotivos.Cerrar(); cerrarIdentidad() }) }
	completa := false
	defer func() {
		if !completa {
			cerrar()
		}
	}()
	motivos, err := postgresct.NuevoResolutorMotivoConsultaRRHHPostgreSQL(poolMotivos)
	if err != nil {
		return vacio, err
	}
	motivoCuadro, err := motivos.ResolverMotivoCuadroRRHH(ctx, reloj.Ahora())
	if err != nil {
		return vacio, err
	}
	motivoDetalle, err := motivos.ResolverMotivoDetalleRRHH(ctx, reloj.Ahora())
	if err != nil {
		return vacio, err
	}
	lectores := resolvedor.lectoresConsultaRRHH()
	autoridad, err := configurarAutoridadConsultasRRHHDesarrollo(alta, reloj, motivoCuadro, motivoDetalle)
	if err != nil {
		return vacio, err
	}
	perfilLectura := perfilFijoConsultasTecnicoCTDesarrollo(alta.soporte)
	if perfilLectura == nil {
		return vacio, ports.ErrConsultaRRHHNoDisponible
	}
	if err = ligarAutoridadConsultasTecnicoPerfilFijoDesarrollo(autoridad, perfilLectura); err != nil {
		return vacio, err
	}
	material, err := nuevoMaterialAtestacionContratacionTemporalDesarrollo(derivador, reloj.Ahora())
	if err != nil {
		return vacio, err
	}
	defer material.borrarCopiasEfimeras()
	material.fuenteConfianza = alta.postgresql.proveedorMaterial.fuenteConfianza
	proveedorCuadro, err := nuevoProveedorMaterialConsumidorDesarrollo(ctx, alta.postgresql.gobierno, material, alta.soporte, reloj, alta.postgresql.catalogoMaterial, ports.AudienciaConsumoConsultaCuadroRRHHV3)
	if err != nil {
		return vacio, err
	}
	proveedorDetalle, err := nuevoProveedorMaterialConsumidorDesarrollo(ctx, alta.postgresql.gobierno, material, alta.soporte, reloj, alta.postgresql.catalogoMaterial, ports.AudienciaConsumoConsultaDetalleRRHHV3)
	if err != nil {
		return vacio, err
	}
	proveedorDescarga, err := nuevoProveedorMaterialConsumidorDesarrollo(ctx, alta.postgresql.gobierno, material, alta.soporte, reloj, alta.postgresql.catalogoMaterial, ports.AudienciaConsumoDescargaBorradorRRHHV3)
	if err != nil {
		return vacio, err
	}
	emisorCuadro, err := nuevoEmisorMaterialRenovableCTDesarrollo(autoridad, proveedorCuadro)
	if err != nil {
		return vacio, err
	}
	emisorDetalle, err := nuevoEmisorMaterialRenovableCTDesarrollo(autoridad, proveedorDetalle)
	if err != nil {
		return vacio, err
	}
	emisor, err := ports.NuevoEmisorMaterialConsultaRRHH(motivos, seguridadvec.GeneradorReferenciasCriptograficas{}, reloj, emisorCuadro, emisorDetalle)
	if err != nil {
		return vacio, err
	}
	sesion, err := postgresct.NuevaSesionConsultaRRHHPostgreSQL(poolConsultas)
	if err != nil {
		return vacio, err
	}
	cuadro, err := application.NuevoServicioConsultaCuadroRRHH(autoridad, emisor, sesion, reloj)
	if err != nil {
		return vacio, err
	}
	cuadro.ConfigurarPlazosFase(dependenciasCT.plazosFase)
	continuador, err := nuevoContinuadorSesionCursorRRHHDesarrollo(autoridad, reloj)
	if err != nil {
		return vacio, err
	}
	cuadroHTTP, err := nuevoConsultorCuadroConSesionCursorRRHHDesarrollo(cuadro, continuador)
	if err != nil {
		return vacio, err
	}
	detalle, err := application.NuevoServicioConsultaDetalleRRHH(autoridad, emisor, sesion, reloj)
	if err != nil {
		return vacio, err
	}
	sesionOriginal, err := postgresct.NuevaSesionConsultaOriginalPropuestaRRHHPostgreSQL(poolConsultas)
	if err != nil {
		return vacio, err
	}
	originalPropuesta, err := application.NuevoServicioConsultaDetalleRRHH(autoridad, emisor, sesionOriginal, reloj)
	if err != nil {
		return vacio, err
	}
	preparacion, err := application.NuevoServicioPreparacionResolucionFormalizacion(autoridad, emisor, sesion, reloj)
	if err != nil {
		return vacio, err
	}
	// Estadísticas por periodo (C18): mismo pool acreditado y mismo rol
	// consultor que el cuadro; agregados sin datos personales.
	estadisticas, err := postgresct.NuevoConsultorEstadisticasRRHHPostgreSQL(poolConsultas)
	if err != nil {
		return vacio, err
	}
	repoDescarga, err := postgresct.NuevoRepositorioDescargaBorradorRRHHPostgreSQL(poolConsultas)
	if err != nil {
		return vacio, err
	}
	descargas, err := nuevoRegistradorDescargaBorradorRRHHDesarrollo(autoridad, proveedorDescarga, repoDescarga, motivoDetalle,
		alta.auditoriaLecturasCT, alta.procesoAuditoriaLecturasCT)
	if err != nil {
		return vacio, err
	}
	base := dependenciasConsultasRRHHDesarrollo{descargas: descargas, repoDescarga: repoDescarga, materialDetalle: proveedorDetalle, emisorCuadro: emisorCuadro, sesion: sesion, sesionOriginal: sesionOriginal, motivos: motivos, cuadro: cuadro, detalle: detalle, originalPropuesta: originalPropuesta, cuadroHTTP: cuadroHTTP, detalleHTTP: consultorDetalleConDisponibilidadRRHH{lector: detalle, documentalMontado: documentalMontado}, originalPropuestaHTTP: originalPropuesta, preparacionResolucion: preparacion, identidad: identidad, autoridad: autoridad, estadisticas: estadisticas, plazosFase: dependenciasCT.plazosFase, cerrar: cerrar}
	if len(lectores) > 0 {
		lectoresDependencias, err := nuevasDependenciasLectoresRRHHDesarrollo(ctx, cfg, alta, derivador, reloj, lectores, poolConsultas, motivos, motivoCuadro, motivoDetalle, base, fronteras, documentalMontado)
		if err != nil {
			return vacio, err
		}
		base.cuadroHTTP, base.detalleHTTP, base.originalPropuestaHTTP, base.cerrar = lectoresDependencias.cuadroHTTP, lectoresDependencias.detalleHTTP, lectoresDependencias.originalPropuestaHTTP, lectoresDependencias.cerrar
		base.descargas = lectoresDependencias.descargas
	}
	completa = true
	identidadCompuesta = true
	return base, nil
}
