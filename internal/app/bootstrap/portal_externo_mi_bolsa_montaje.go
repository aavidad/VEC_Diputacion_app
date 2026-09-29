package bootstrap

import (
	"context"
	"net/http"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/config"
	contextopg "vec-diputacion-granada/internal/vec/adapters/contextoactor/postgres"
	identidadpg "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
	vecpg "vec-diputacion-granada/internal/vec/adapters/postgres"
	vecapp "vec-diputacion-granada/internal/vec/application"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// La presencia del DSN nominal activa la consulta. Las acciones necesitan
// además el selector y el catálogo gobernado; ningún montaje publica permisos.
func nuevaMiBolsaPortalExterno(ctx context.Context, cfg config.Config,
	identidad *resolvedorIdentidadDesarrollo, derivador *derivadorIdentidadOperacionDesarrollo,
	preflight *pgxpool.Pool, incidencias vecports.EmisorIncidenciasTecnicas,
) (http.Handler, func(), error) {
	nada := func() {}
	portal, err := cfg.BolsaPortalCandidatoDesarrolloActivo()
	bolsaDSN, sinBolsa := cfg.ExternoBolsaPostgreSQL.DSN()
	if err != nil {
		return nil, nada, err
	}
	if sinBolsa != nil {
		if portal {
			return nil, nada, errMiBolsaNoDisponible
		}
		return nil, nada, nil
	}
	if ctx == nil || identidad == nil || identidad.candidatoBolsa == nil ||
		derivador == nil || !derivador.valido() || preflight == nil || incidencias == nil {
		return nil, nada, errMiBolsaNoDisponible
	}
	topologia, err := acreditarTopologiaPostgreSQLPreferenciasUsuarios(ctx, preflight)
	if err != nil {
		return nil, nada, errMiBolsaNoDisponible
	}
	entradas := []struct {
		configuracion config.ConfiguracionPostgreSQLExterna
		rol           string
	}{
		{cfg.ExternoIdentidadRegistroPostgreSQL, "vec_identidad_externa_v1_registrador"},
		{cfg.ExternoIdentidadRevalidacionPostgreSQL, "vec_identidad_externa_v1_revalidador"},
		{cfg.ExternoContextoPostgreSQL, "vec_contexto_actor_v1_candidato_externo"},
		{cfg.ExternoAutorizacionFuentePostgreSQL, "vec_autorizacion_fuente_externa"},
		{cfg.ExternoAutorizacionRegistroPostgreSQL, "vec_autorizacion_registro_externo"},
		{cfg.ExternoAutorizacionMotivosPostgreSQL, "vec_autorizacion_motivos_externos"},
	}
	var pools []*pgxpool.Pool
	var unaVez sync.Once
	cerrar := func() {
		unaVez.Do(func() {
			for _, p := range pools {
				p.Close()
			}
		})
	}
	completa := false
	defer func() {
		if !completa {
			cerrar()
		}
	}()
	logins := map[string]bool{}
	for _, entrada := range entradas {
		dsn, err := entrada.configuracion.DSN()
		if err != nil {
			return nil, nada, errMiBolsaNoDisponible
		}
		pool, login, err := abrirPoolMiBolsaPortalExterno(ctx, dsn, entrada.rol)
		if err != nil {
			return nil, nada, errMiBolsaNoDisponible
		}
		pools = append(pools, pool)
		if logins[login] || cotejarTopologiaPostgreSQLPreferenciasUsuarios(ctx, pool, topologia) != nil {
			return nil, nada, errMiBolsaNoDisponible
		}
		logins[login] = true
	}
	bolsa, login, err := abrirBolsaMiBolsaPortalExterno(ctx, bolsaDSN)
	if err != nil {
		return nil, nada, errMiBolsaNoDisponible
	}
	pools = append(pools, bolsa)
	if logins[login] || cotejarTopologiaPostgreSQLPreferenciasUsuarios(ctx, bolsa, topologia) != nil {
		return nil, nada, errMiBolsaNoDisponible
	}
	logins[login] = true
	dsnFrontera, err := cfg.ExternoBolsaFronteraPostgreSQL.DSN()
	if err != nil {
		return nil, nada, errMiBolsaNoDisponible
	}
	frontera, auditoria, err := abrirAuditoriaMiBolsaPortalExterno(ctx, dsnFrontera)
	if err != nil {
		return nil, nada, errMiBolsaNoDisponible
	}
	pools = append(pools, frontera)
	if logins[frontera.Config().ConnConfig.User] || cotejarTopologiaPostgreSQLPreferenciasUsuarios(ctx, frontera, topologia) != nil {
		return nil, nada, errMiBolsaNoDisponible
	}
	registro, err := identidadpg.NuevoRegistroSesionesExternoPostgreSQL(ctx, pools[0], pools[1],
		&seudonimizadorSesionDesarrollo{derivador: derivador}, espacioIdentidadSesionDesarrollo, dominioIdentidadSesionDesarrollo)
	if err != nil {
		return nil, nada, errMiBolsaNoDisponible
	}
	revalidador, err := identidadpg.NuevoRevalidadorAutenticacionActorExternoPostgreSQL(ctx, pools[1])
	if err != nil {
		return nil, nada, errMiBolsaNoDisponible
	}
	resolutor, err := contextopg.NuevoResolutorRegistroContextoActorExternoPostgreSQLV1(ctx, pools[2])
	if err != nil {
		return nil, nada, errMiBolsaNoDisponible
	}
	servicioContexto, err := vecapp.NuevoServicioContextoActorProductivoV2(resolutor,
		contextopg.NuevoGeneradorOperacionContextoActorV2Criptografico(), relojRutasDietas{})
	if err != nil {
		return nil, nada, errMiBolsaNoDisponible
	}
	contextos, err := vecapp.NuevaAutoridadContextoActorRegistradoV2(servicioContexto)
	if err != nil {
		return nil, nada, errMiBolsaNoDisponible
	}
	sesion, err := nuevaSesionMiBolsaPortalExterno(identidad, registro, revalidador, contextos)
	if err != nil {
		return nil, nada, errMiBolsaNoDisponible
	}
	fuente, err := vecpg.NuevoAlmacenAutorizacionExterna(pools[3])
	if err != nil {
		return nil, nada, errMiBolsaNoDisponible
	}
	decisiones, err := vecpg.NuevoAlmacenAutorizacionExterna(pools[4])
	if err != nil {
		return nil, nada, errMiBolsaNoDisponible
	}
	motivos, err := nuevosMotivosMiBolsaPortalExterno(pools[5])
	if err != nil {
		return nil, nada, errMiBolsaNoDisponible
	}
	autorizador, err := nuevaAutorizacionMiBolsaPortalExterno(fuente, decisiones, decisiones, motivos)
	if err != nil {
		return nil, nada, errMiBolsaNoDisponible
	}
	proveedores, err := nuevosProveedoresV3PortalExterno(ctx, cfg.DevelopmentMaterialDir, preflight, "mi_bolsa", relojContratacionTemporalDesarrollo{})
	if err != nil {
		return nil, nada, errMiBolsaNoDisponible
	}
	d := dependenciasMiBolsaPortalExterno{preparador: &preparadorMiBolsaPortalExterno{preferencias: sesion, identidad: identidad.candidatoBolsa},
		autorizador: autorizador, bolsa: bolsa, proveedores: proveedores, reloj: relojContratacionTemporalDesarrollo{}}
	if portal {
		if comprobarMigracionesPortalCandidatoDesarrollo(ctx, bolsa) != nil {
			return nil, nada, errMiBolsaNoDisponible
		}
		dsnCalendarios, err := cfg.ExternoCalendariosPostgreSQL.DSN()
		if err != nil {
			return nil, nada, errMiBolsaNoDisponible
		}
		calendarioPool, calendarios, err := nuevaConsultaCalendariosMiBolsaPortalExterno(ctx, dsnCalendarios, topologia)
		if err != nil {
			return nil, nada, errMiBolsaNoDisponible
		}
		pools = append(pools, calendarioPool)
		if logins[calendarioPool.Config().ConnConfig.User] {
			return nil, nada, errMiBolsaNoDisponible
		}
		reglas, err := nuevasReglasEjemploDesarrollo(cfg, calendarios, relojCalendariosDesarrollo{})
		if err != nil || reglas.bolsa == nil {
			return nil, nada, errMiBolsaNoDisponible
		}
		d.reglas = reglasPortalCandidatoDesarrollo{resolutor: reglas.bolsa}
		d.campos, err = camposPortalMiBolsaDesarrollo(reglas.bolsa)
		if err != nil {
			return nil, nada, errMiBolsaNoDisponible
		}
		acciones, err := nuevosProveedoresV3PortalExterno(ctx, cfg.DevelopmentMaterialDir, preflight, "portal_candidato", relojContratacionTemporalDesarrollo{})
		if err != nil {
			return nil, nada, errMiBolsaNoDisponible
		}
		for audiencia, proveedor := range acciones {
			d.proveedores[audiencia] = proveedor
		}
	}
	manejador, err := nuevaAPIMiBolsaPortalExterno(identidad, incidencias, auditoria, d)
	if err != nil {
		return nil, nada, errMiBolsaNoDisponible
	}
	completa = true
	return manejador, cerrar, nil
}
