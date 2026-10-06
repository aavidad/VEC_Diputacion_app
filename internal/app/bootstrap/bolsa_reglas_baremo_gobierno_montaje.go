package bootstrap

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/config"
	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
	bolsapg "vec-diputacion-granada/internal/modules/bolsa/adapters/postgres"
	app "vec-diputacion-granada/internal/modules/bolsa/application/gobiernoreglasbaremo"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	pgvec "vec-diputacion-granada/internal/vec/adapters/postgres"
	seg "vec-diputacion-granada/internal/vec/adapters/seguridad"
	appvec "vec-diputacion-granada/internal/vec/application"
	vecports "vec-diputacion-granada/internal/vec/ports"

	"vec-diputacion-granada/internal/vec/adapters/observabilidad/telemetria/medidorpg"
)

type montajeGobiernoReglasBaremoHTTPV3 struct {
	configuracion *configuracionGobiernoReglasBaremoHTTPV3
	perfil        *PerfilGobiernoReglasBaremoV3
	perfilRef     string
}

func prepararMontajeGobiernoReglasBaremoHTTPV3(cfg config.Config, base *soporteAltaContratacionTemporalDesarrollo, reloj relojContratacionTemporalDesarrollo) (*montajeGobiernoReglasBaremoHTTPV3, error) {
	c, err := leerConfiguracionGobiernoReglasBaremoHTTPV3(cfg)
	if err != nil {
		return nil, err
	}
	m := &montajeGobiernoReglasBaremoHTTPV3{configuracion: c}
	if c == nil {
		// Declara la familia indisponible sin publicar identidad, concesiones,
		// scopes ni material. La referencia sólo describe una frontera.
		_, perfil, err := nuevoSoportePlantillasCTDesdeBaseDesarrollo(base, reloj.Ahora(), discriminadorGobiernoReglasBaremoV3())
		if err != nil {
			return nil, app.ErrGobiernoV3NoDisponible
		}
		m.perfilRef = perfil
		return m, nil
	}
	m.perfil, err = NuevoPerfilGobiernoReglasBaremoV3(base, c.ConvocatoriaRef, c.ExpedienteRef, reloj.Ahora())
	if err != nil {
		return nil, err
	}
	m.perfilRef = m.perfil.PerfilRef()
	return m, nil
}

func rutasHandlerGobiernoReglasBaremoHTTPV3(h http.Handler) []vechttp.RutaExacta {
	var rutas []vechttp.RutaExacta
	for _, p := range paresGobiernoReglasBaremoHTTPV3() {
		rutas = append(rutas, vechttp.RutaExacta{Ruta: p.ruta, Manejador: h})
	}
	return rutas
}

func (m *montajeGobiernoReglasBaremoHTTPV3) indisponibles() []vechttp.RutaExacta {
	return rutasHandlerGobiernoReglasBaremoHTTPV3(&bolsahttp.HandlerGobiernoReglasBaremoV3{})
}

func (m *montajeGobiernoReglasBaremoHTTPV3) rutas(ctx context.Context, cfg config.Config, alta *dependenciasAltaContratacionTemporalDesarrollo,
	identidadBase *proveedorSesionConsultaRRHHDesarrollo, fronteras catalogoFronterasComunDesarrollo,
	derivador *derivadorIdentidadOperacionDesarrollo, reloj relojContratacionTemporalDesarrollo,
) ([]vechttp.RutaExacta, func(), error) {
	if m == nil {
		return nil, nil, app.ErrGobiernoV3NoDisponible
	}
	if m.configuracion == nil {
		return m.indisponibles(), func() {}, nil
	}
	if ctx == nil || ctx.Err() != nil || alta == nil || alta.postgresql.gobierno == nil ||
		alta.postgresql.registroAutorizacion == nil || alta.postgresql.proveedorMaterial == nil ||
		dependenciaEsNulaContratacionTemporalDesarrollo(alta.postgresql.registradorAuditoriaFrontera) ||
		m.perfil == nil || identidadBase == nil || fronteras.identidad == nil ||
		!identidadBase.fronteras.mismaInstancia(fronteras) {
		return nil, nil, app.ErrGobiernoV3NoDisponible
	}
	for _, par := range paresGobiernoReglasBaremoHTTPV3() {
		f, ok := fronteras.resolver(http.MethodPost, par.ruta)
		if !ok || f.Clave != par.frontera || f.ClaveCapacidad != par.accion || !f.admitePerfil(m.perfilRef) {
			return nil, nil, app.ErrGobiernoV3NoDisponible
		}
	}
	pools, cerrar, err := abrirPoolsGobiernoReglasBaremoHTTPV3(ctx, cfg, m.configuracion)
	if err != nil {
		return nil, nil, err
	}
	completo := false
	defer func() {
		if !completo {
			cerrar()
		}
	}()
	repo, err := bolsapg.NuevoRepositorioGobiernoReglasBaremoV3PostgreSQL(ctx, pools["runtime"], "vec_bolsa_reglas_baremo_ejecutor_gobierno")
	if err != nil {
		return nil, nil, app.ErrGobiernoV3NoDisponible
	}
	soporte := m.perfil.soporte
	operacion := referenciaAltaContratacionTemporalDesarrollo("oca_", soporte.principalID+"\x00"+soporte.certificadoSHA256+"\x00registro-contexto-bolsa-gobierno-reglas-baremo-v3")
	if publicarResultadoContextoPostgreSQLDesarrollo(ctx, alta.postgresql.gobierno, soporte.contexto.Resultado, operacion) != nil {
		return nil, nil, app.ErrGobiernoV3NoDisponible
	}
	esperado, err := contextoEsperadoRegistradoDesarrollo(ctx, identidadBase.resolutor, soporte)
	if err != nil {
		return nil, nil, app.ErrGobiernoV3NoDisponible
	}
	soporte.contextoEsperadoRegistrado = esperado
	// La provisión sólo se selecciona expresamente en configuración privada
	// de arranque. Sin ella, una ausencia no se convierte en una concesión.
	if m.configuracion.ProvisionarPerfil {
		err = AsegurarPerfilGobiernoReglasBaremoV3(ctx, alta.postgresql.gobierno, m.perfil,
			m.configuracion.AprobacionRef, m.configuracion.PreimagenPerfilSHA256, reloj.Ahora())
	} else {
		_, existe, e := leerInstantaneaPublicadaPostgreSQLDesarrollo(ctx, alta.postgresql.gobierno, m.perfilRef)
		if e != nil || !existe {
			return nil, nil, app.ErrGobiernoV3NoDisponible
		}
	}
	if err != nil {
		return nil, nil, app.ErrGobiernoV3NoDisponible
	}
	sesion, err := nuevoProveedorSesionConsultaRRHHConCatalogoDesarrollo(soporte, identidadBase.registro,
		identidadBase.revalidador, reloj, identidadBase.resolutor, fronteras)
	if err != nil {
		return nil, nil, app.ErrGobiernoV3NoDisponible
	}
	soporte.sesionOperativa = sesion
	fuente, err := pgvec.NuevoAlmacenAutorizacion(pools["fuente_autorizacion"])
	if err != nil {
		return nil, nil, app.ErrGobiernoV3NoDisponible
	}
	registro, err := pgvec.NuevoAlmacenAutorizacion(alta.postgresql.registroAutorizacion)
	if err != nil {
		return nil, nil, app.ErrGobiernoV3NoDisponible
	}
	motivos, err := pgvec.NuevoValidadorReferenciaMotivoPostgreSQLV2(pools["motivos_autorizacion"], m.configuracion.CatalogoMotivosID)
	if err != nil {
		return nil, nil, app.ErrGobiernoV3NoDisponible
	}
	if motivos.ValidarReferenciaMotivoAutorizacionV2(ctx, m.configuracion.MotivoIntentoDenegado, reloj.Ahora()) != nil || motivos.ValidarReferenciaMotivoAutorizacionV2(ctx, m.configuracion.MotivoIntentoError, reloj.Ahora()) != nil {
		return nil, nil, app.ErrGobiernoV3NoDisponible
	}
	politica, err := nuevaPoliticaAutorizacionSolicitudLigadaV3Desarrollo(fuente, registro, registro, motivos)
	if err != nil {
		return nil, nil, app.ErrGobiernoV3NoDisponible
	}
	descriptores, err := autorizacionesGobiernoReglasBaremoHTTPV3(politica)
	if err != nil {
		return nil, nil, err
	}
	catalogo, err := nuevoCatalogoAutorizacionComunDesarrollo(fronteras, descriptores)
	if err != nil {
		return nil, nil, err
	}
	pdp, err := nuevoAutorizadorComunDesarrollo(catalogo, reloj, seg.GeneradorReferenciasCriptograficas{}, appvec.ConfiguracionServicioAutorizacion{VigenciaDecision: 90 * time.Second})
	if err != nil {
		return nil, nil, app.ErrGobiernoV3NoDisponible
	}
	material, err := materialGobiernoReglasBaremoHTTPV3(ctx, alta, soporte, derivador, reloj)
	if err != nil {
		return nil, nil, err
	}
	proveedor, err := NuevoProveedorGobiernoReglasBaremoV3(m.perfil, sesion, pdp, material, alta.postgresql.gobierno,
		RutasGobiernoReglasBaremoV3{bolsahttp.RutaAltaGobiernoReglasBaremoV3, bolsahttp.RutaConsultaGobiernoReglasBaremoV3, bolsahttp.RutaRecuperarGobiernoReglasBaremoV3}, reloj)
	if err != nil {
		return nil, nil, app.ErrGobiernoV3NoDisponible
	}
	logins := []string{}
	for _, pool := range pools {
		logins = append(logins, pool.Config().ConnConfig.User)
	}
	registradorIntentos, procesoIntentos, cerrarIntentos, err := AbrirRegistradorIntentosAuditoriaDesarrollo(ctx, cfg, pools["runtime"], logins)
	if err != nil {
		return nil, nil, app.ErrGobiernoV3NoDisponible
	}
	cerrarPools := cerrar
	cerrar = func() { cerrarIntentos(); cerrarPools() }
	auditor := &auditorGobiernoBaremoHTTPV3{sesion: sesion, registrador: registradorIntentos, proceso: procesoIntentos, motivoDenegado: m.configuracion.MotivoIntentoDenegado, motivoError: m.configuracion.MotivoIntentoError, recursoRef: m.configuracion.ExpedienteRef}
	proveedor.auditarAntesPDP = auditorDenegacionAntesPDPGobiernoReglasBaremoHTTPV3(sesion, alta.postgresql.registradorAuditoriaFrontera, auditor)
	servicio, err := app.NuevoServicioGobiernoV3(repo, repo, proveedor, reloj.Ahora)
	if err != nil {
		return nil, nil, err
	}
	h, err := bolsahttp.NuevoHandlerGobiernoReglasBaremoV3(proveedor, &operadorAuditadoGobiernoBaremoHTTPV3{proveedor: proveedor, operador: servicio, auditor: auditor},
		func(ctx context.Context, ruta string, fallo error) error {
			operativo, err := auditor.contextoHistorico(ctx)
			if err == nil {
				return auditor.registrar(ctx, operativo, fallo)
			}
			return auditorRechazoSesionGobiernoReglasBaremoHTTPV3(sesion, alta.postgresql.registradorAuditoriaFrontera)(ctx, ruta, fallo)
		},
		func(ctx context.Context, fallo error) error {
			var auditado errorAuditadoGobiernoBaremoHTTPV3
			if errors.As(fallo, &auditado) {
				if !auditado.confirmada {
					return app.ErrGobiernoV3NoDisponible
				}
				return nil
			}
			operativo, err := auditor.contextoHistorico(ctx)
			if err != nil {
				return app.ErrGobiernoV3NoDisponible
			}
			return auditor.registrar(ctx, operativo, fallo)
		})
	if err != nil {
		return nil, nil, err
	}
	completo = true
	return rutasHandlerGobiernoReglasBaremoHTTPV3(handlerIntentoGobiernoBaremoHTTPV3(h, m.configuracion.ExpedienteRef)), cerrar, nil
}

func auditorRechazoSesionGobiernoReglasBaremoHTTPV3(sesion *proveedorSesionConsultaRRHHDesarrollo, registrador vecports.RegistradorAuditoriaFronteraRutaExacta) bolsahttp.AuditarRechazoSesionGobiernoReglasV3 {
	return func(ctx context.Context, ruta string, err error) error {
		// El export común sólo registra una observación. La composición
		// conserva aquí la prueba opaca del perímetro que emitió la raíz.
		if !sesion.sesionGobiernoReglasBaremoHTTPV3(ctx, ruta) {
			return app.ErrGobiernoV3NoDisponible
		}
		var motivo vecports.MotivoAuditoriaFronteraRutaExacta
		switch {
		case errors.Is(err, app.ErrGobiernoV3NoAutenticado):
			motivo = vecports.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida
		case errors.Is(err, app.ErrGobiernoV3Prohibido):
			motivo = vecports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado
		default:
			return app.ErrGobiernoV3NoDisponible
		}
		return vechttp.RegistrarDenegacionFronteraPreparacion(ctx, registrador, http.MethodPost, ruta, motivo)
	}
}

func abrirPoolsGobiernoReglasBaremoHTTPV3(ctx context.Context, cfg config.Config, c *configuracionGobiernoReglasBaremoHTTPV3) (map[string]*pgxpool.Pool, func(), error) {
	pools := map[string]*pgxpool.Pool{}
	cerrar := func() {
		for _, p := range pools {
			p.Close()
		}
	}
	raiz, err := os.OpenRoot(cfg.DevelopmentMaterialDir)
	if err != nil {
		return nil, nil, app.ErrGobiernoV3NoDisponible
	}
	defer raiz.Close()
	usuarios := map[string]bool{}
	for _, nombre := range []string{"runtime", "fuente_autorizacion", "motivos_autorizacion"} {
		b, err := leerArchivoIncorporacionV2(raiz, c.DSNFiles[nombre], 16<<10)
		if err != nil {
			cerrar()
			return nil, nil, app.ErrGobiernoV3NoDisponible
		}
		dsn := strings.TrimSpace(string(b))
		borrarBytes(b)
		var p *pgxpool.Pool
		if nombre == "runtime" {
			p, err = abrirPoolRuntimeGobiernoReglasBaremoHTTPV3(ctx, dsn)
		} else {
			rol := config.RolAutorizacionFuenteRRHH
			if nombre == "motivos_autorizacion" {
				rol = config.RolAutorizacionMotivosEvaluadorRRHH
			}
			p, err = abrirPoolAutorizacionRRHHDesarrollo(ctx, dsn, rol, "vec-bolsa-baremo-"+nombre)
		}
		if err != nil {
			cerrar()
			return nil, nil, app.ErrGobiernoV3NoDisponible
		}
		pools[nombre] = p
		usuario := p.Config().ConnConfig.User
		if usuario == "" || usuarios[usuario] {
			cerrar()
			return nil, nil, app.ErrGobiernoV3NoDisponible
		}
		usuarios[usuario] = true
	}
	if preflightAutoridadesPlantillasCT(ctx, pools["fuente_autorizacion"], pools["motivos_autorizacion"]) != nil {
		cerrar()
		return nil, nil, app.ErrGobiernoV3NoDisponible
	}
	return pools, cerrar, nil
}

func abrirPoolRuntimeGobiernoReglasBaremoHTTPV3(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	c, err := pgxpool.ParseConfig(dsn)
	if err != nil || c == nil || c.ConnConfig == nil || validarTLSPostgreSQLBorradores(&c.ConnConfig.Config, true) != nil {
		return nil, app.ErrGobiernoV3NoDisponible
	}
	c.MaxConns, c.MinConns = 4, 0
	c.ConnConfig.ConnectTimeout = 5 * time.Second
	if c.ConnConfig.RuntimeParams == nil {
		c.ConnConfig.RuntimeParams = map[string]string{}
	}
	for k, v := range map[string]string{"application_name": "vec-bolsa-gobierno-reglas-baremo-v3", "search_path": "pg_catalog", "timezone": "UTC", "statement_timeout": "15s", "lock_timeout": "2s", "idle_in_transaction_session_timeout": "20s"} {
		c.ConnConfig.RuntimeParams[k] = v
	}
	// Mide consultas y esperas de conexión por petición (registro técnico).
	medidorpg.Instrumentar(c)
	p, err := pgxpool.NewWithConfig(ctx, c)
	if err != nil {
		return nil, app.ErrGobiernoV3NoDisponible
	}
	return p, nil
}

func materialGobiernoReglasBaremoHTTPV3(ctx context.Context, alta *dependenciasAltaContratacionTemporalDesarrollo, soporte *soporteAltaContratacionTemporalDesarrollo, derivador *derivadorIdentidadOperacionDesarrollo, reloj relojContratacionTemporalDesarrollo) (*proveedorMaterialAltaContratacionTemporalDesarrollo, error) {
	fuente := alta.postgresql.proveedorMaterial.fuenteConfianza
	if fuente == nil {
		return nil, app.ErrGobiernoV3NoDisponible
	}
	if _, err := fuente.instantanea(ctx); err != nil {
		return nil, app.ErrGobiernoV3NoDisponible
	}
	m, err := nuevoMaterialAtestacionContratacionTemporalDesarrollo(derivador, reloj.Ahora())
	if err != nil {
		return nil, app.ErrGobiernoV3NoDisponible
	}
	defer borrarBytes(m.privada)
	defer borrarBytes(m.claveHMAC)
	fuente.mu.Lock()
	actual := fuente.material
	fuente.mu.Unlock()
	if actual.claveID != m.claveID || !bytes.Equal(actual.spki, m.spki) {
		return nil, app.ErrGobiernoV3NoDisponible
	}
	// Conserva la raíz y metadatos realmente publicados por la única fuente;
	// sólo deriva y gobierna la clave de este consumidor nominal al arrancar.
	m.claveVersion, m.raiz = actual.claveVersion, actual.raiz
	m.configuracion, m.configuracionRef, m.configuracionOrden, m.configuracionHuella = actual.configuracion, actual.configuracionRef, actual.configuracionOrden, actual.configuracionHuella
	m.publicadaEn, m.expiraEn, m.validaDesde, m.validaHasta = actual.publicadaEn, actual.expiraEn, actual.validaDesde, actual.validaHasta
	m.fuenteConfianza = fuente
	return nuevoProveedorMaterialConsumidorConDescriptorDesarrollo(ctx, alta.postgresql.gobierno, m, soporte, reloj, DescriptorMaterialGobiernoReglasBaremoV3())
}
