package bootstrap

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
	"vec-diputacion-granada/config"
	contratacioncomposicion "vec-diputacion-granada/internal/app/composicion/interna/contrataciontemporal"
	inc "vec-diputacion-granada/internal/app/incorporacionejercicio"
	bolsapersonal "vec-diputacion-granada/internal/modules/bolsa/adapters/httppersonal"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/informejuridico"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	docxvec "vec-diputacion-granada/internal/vec/adapters/documentos/docx"
	pdfvec "vec-diputacion-granada/internal/vec/adapters/documentos/pdf"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	expedienteContratacionTemporalDesarrolloRef   = "expediente:ct:demo:0001"
	rolTecnicoRRHHContratacionTemporalDesarrollo  = "tecnico_rrhh"
	rolIntervencionContratacionTemporalDesarrollo = "intervencion"
)

type selloConsultasContratacionTemporalDesarrollo struct{ _ byte }

type claveCapacidadConsultasContratacionTemporalDesarrollo struct{}

type capacidadConsultaContratacionTemporalDesarrollo struct {
	sello                   *selloConsultasContratacionTemporalDesarrollo
	ruta                    string
	principal               vecdomain.Principal
	consultaRRHH            *contextoConsultaRRHHPeticionDesarrollo
	contextoOperacion       *contextoOperacionCTDesarrollo
	vinculoCanalTLS         [32]byte
	certificadoVerificadoEn time.Time
	certificadoValidoHasta  time.Time
}

// autoridadConsultasContratacionTemporalDesarrollo solo reconoce capacidades
// efimeras emitidas tras revalidar el certificado mTLS local. No representa
// autoridad corporativa ni se construye fuera del perfil de desarrollo.
type autoridadConsultasContratacionTemporalDesarrollo struct {
	sello                                    *selloConsultasContratacionTemporalDesarrollo
	resolvedor                               *resolvedorIdentidadDesarrollo
	noCompuesta                              *capacidadNoCompuestaContratacionTemporalDesarrollo
	llamamientoCompuesto                     bool
	consultasRRHHCompuestas                  bool
	subsanacionCompuesta                     bool
	fronterasSeguridadComun                  catalogoFronterasComunDesarrollo
	envolverBorradorLlamamiento              func(http.Handler) http.Handler
	manejadorSituacionParticipacion          http.Handler
	coleccionesAdicionales                   []vechttp.RutaColeccion
	registradorAuditoriaFronteraRutasExactas puertosvec.RegistradorAuditoriaFronteraRutaExacta
	materialDietas                           materialDietasDesdeCTDesarrollo
	materialCronos                           materialCronosDesdeCTDesarrollo
	materialDocumentos                       *proveedorMaterialAltaContratacionTemporalDesarrollo
	materialPersonalFichaPropia              *proveedorMaterialAltaContratacionTemporalDesarrollo
	plazosOfertasBolsa                       *calculadoraPlazoOfertaDesarrollo
	// presentadorCobertura permite activar después los avisos de la vía de
	// cobertura, cuando Bolsa y las reglas de ejemplo ya están compuestas.
	presentadorCobertura avisosViaCoberturaConfigurable
}

type autorizadorLigadoContratacionTemporalDesarrollo interface {
	puertosvec.AutorizadorSolicitudLigadaV3
	puertosvec.PreparadorRegistroCompuestoSolicitudLigadaV3
}

type claveSolicitudAutorizacionContratacionTemporalDesarrollo struct{}

// autorizadorAnalisisContratacionTemporalDesarrollo liga a este contexto
// interno la solicitud ya construida por el caso de uso tras preparar el
// expediente y resolver su politica. soporteAlta no lee cuerpo ni cabeceras
// HTTP para ampliar los ambitos de la instantanea.
type autorizadorAnalisisContratacionTemporalDesarrollo struct {
	delegado  autorizadorLigadoContratacionTemporalDesarrollo
	soporte   *soporteAltaContratacionTemporalDesarrollo
	instalado bool
}

// instalarDelegadoComun se invoca exclusivamente durante la composición, antes
// de publicar el servidor. Conserva el mismo wrapper que ya recibieron los
// servicios de este wrapper CT, de modo que estos consumidores y B-BACK
// quedan detrás de la misma instancia V3. Otras autoridades CT conservan sus
// composiciones nominales propias.
func (a *autorizadorAnalisisContratacionTemporalDesarrollo) instalarDelegadoComun(
	delegado autorizadorLigadoContratacionTemporalDesarrollo,
) error {
	if a == nil || dependenciaEsNulaContratacionTemporalDesarrollo(delegado) || a.instalado {
		return errAltaContratacionTemporalDesarrolloNoDisponible
	}
	a.delegado = delegado
	a.instalado = true
	return nil
}

func (a *autorizadorAnalisisContratacionTemporalDesarrollo) ExigirSolicitudLigadaV3(
	ctx context.Context,
	solicitud vecdomain.SolicitudAutorizacionLigadaV3,
	resultado vecdomain.ResultadoContextoActorRegistradoV2,
) (
	vecdomain.DecisionAutorizacionLigadaV3,
	puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
	error,
) {
	if a == nil || dependenciaEsNulaContratacionTemporalDesarrollo(a.delegado) {
		return vecdomain.DecisionAutorizacionLigadaV3{},
			puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3{},
			errAltaContratacionTemporalDesarrolloNoDisponible
	}
	datos, err := solicitud.Datos()
	if err != nil {
		return vecdomain.DecisionAutorizacionLigadaV3{},
			puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3{},
			errAltaContratacionTemporalDesarrolloNoDisponible
	}
	if datos.Accion == ports.AccionRegistrarAnalisis ||
		datos.Accion == ports.AccionRectificarAnalisis ||
		datos.Accion == ports.AccionCrearSolicitud ||
		datos.Accion == ports.AccionRegistrarAsignacion ||
		datos.Accion == ports.AccionEmitirInformeJuridico ||
		datos.Accion == string(domain.AccionRegistrarSubsanacionReparo) {
		if ctx == nil {
			return vecdomain.DecisionAutorizacionLigadaV3{},
				puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3{},
				errAltaContratacionTemporalDesarrolloNoDisponible
		}
		ctx = context.WithValue(
			ctx,
			claveSolicitudAutorizacionContratacionTemporalDesarrollo{},
			datos,
		)
	}
	return a.delegado.ExigirSolicitudLigadaV3(ctx, solicitud, resultado)
}

func (a *autorizadorAnalisisContratacionTemporalDesarrollo) PrepararRegistroCompuestoSolicitudLigadaV3(
	ctx context.Context,
	solicitud vecdomain.SolicitudAutorizacionLigadaV3,
	resultado vecdomain.ResultadoContextoActorRegistradoV2,
	generador puertosvec.GeneradorReferenciaDecisionAutorizacion,
) (
	vecdomain.DecisionAutorizacionLigadaV3,
	puertosvec.CandidataRegistroDecisionAutorizacionLigadaV3,
	error,
) {
	if a == nil || dependenciaEsNulaContratacionTemporalDesarrollo(a.delegado) {
		return vecdomain.DecisionAutorizacionLigadaV3{},
			puertosvec.CandidataRegistroDecisionAutorizacionLigadaV3{},
			errAltaContratacionTemporalDesarrolloNoDisponible
	}
	datos, err := solicitud.Datos()
	if err != nil {
		return vecdomain.DecisionAutorizacionLigadaV3{},
			puertosvec.CandidataRegistroDecisionAutorizacionLigadaV3{},
			errAltaContratacionTemporalDesarrolloNoDisponible
	}
	ruta := ""
	switch datos.Accion {
	case string(domain.AccionDecidirCoberturaGobernada):
		ruta = httpinterno.RutaDecisionCobertura
	case string(domain.AccionRectificarCoberturaGobernada):
		ruta = httpinterno.RutaRectificacionCobertura
	}
	if ruta != "" {
		if a.soporte == nil || ctx == nil {
			return vecdomain.DecisionAutorizacionLigadaV3{},
				puertosvec.CandidataRegistroDecisionAutorizacionLigadaV3{},
				errAltaContratacionTemporalDesarrolloNoDisponible
		}
		ctx = context.WithValue(
			ctx,
			claveSolicitudAutorizacionContratacionTemporalDesarrollo{},
			datos,
		)
		if err := a.soporte.publicarInstantaneaDecisionCobertura(
			ctx,
			ruta,
		); err != nil {
			return vecdomain.DecisionAutorizacionLigadaV3{},
				puertosvec.CandidataRegistroDecisionAutorizacionLigadaV3{},
				err
		}
	}
	return a.delegado.PrepararRegistroCompuestoSolicitudLigadaV3(
		ctx, solicitud, resultado, generador,
	)
}

func nuevasRutasContratacionTemporalDesarrollo(
	cfg config.Config,
	resolvedor vechttp.DemoIdentityResolver,
	derivador *derivadorIdentidadOperacionDesarrollo,
	kms *emisorKMSDesarrollo,
	registro io.Writer,
	incorporacion ...ConfiguracionIncorporacionDesarrollo,
) (
	[]vechttp.RutaExacta,
	*autoridadConsultasContratacionTemporalDesarrollo,
	func(),
	error,
) {
	return nuevasRutasContratacionTemporalConReglasDesarrollo(cfg, reglasEjemploDesarrollo{}, resolvedor, derivador, kms, registro, incorporacion...)
}

// nuevasRutasContratacionTemporalConReglasDesarrollo recibe además las
// reglas de ejemplo ya validadas; sin ellas cada consumidor conserva su
// conducta sin catálogo.
func nuevasRutasContratacionTemporalConReglasDesarrollo(
	cfg config.Config,
	reglasEjemplo reglasEjemploDesarrollo,
	resolvedor vechttp.DemoIdentityResolver,
	derivador *derivadorIdentidadOperacionDesarrollo,
	kms *emisorKMSDesarrollo,
	registro io.Writer,
	incorporacion ...ConfiguracionIncorporacionDesarrollo,
) (
	[]vechttp.RutaExacta,
	*autoridadConsultasContratacionTemporalDesarrollo,
	func(),
	error,
) {
	if len(incorporacion) > 1 || (len(incorporacion) != 0 && cfg.IncorporacionV2File != "") {
		return nil, nil, nil, ErrComposicionDesarrolloIncompleta
	}
	dependencias, err := nuevasDependenciasCT(cfg, resolvedor, derivador, kms, registro)
	if err != nil {
		return nil, nil, nil, err
	}
	dependencias.reglasEjemplo = reglasEjemplo
	dependencias.plazosFase = nuevaCalculadoraPlazoFaseCT(reglasEjemplo.contratacionTemporal)
	if err := dependencias.cfg.ContratacionTemporalPostgreSQL.ValidarIdentidadOperativa(); err != nil {
		return nil, nil, nil, err
	}
	cfg, resolvedorDesarrollo, derivador, registro := dependencias.cfg, dependencias.resolvedor, dependencias.derivador, dependencias.registro
	noCompuesta, err := nuevaCapacidadNoCompuestaContratacionTemporalDesarrollo(registro)
	if err != nil {
		return nil, nil, nil, err
	}
	catalogoDesarrollo, err := nuevoCatalogoDesarrollo(cfg.PersonalOrganizacionSourcePath, cfg.RPTCatalogoPath)
	if err != nil {
		return nil, nil, nil, err
	}
	origen := nuevoOrigenConsultasConCatalogoDesarrollo(catalogoDesarrollo)
	sello := dependencias.sello
	reloj := dependencias.reloj
	alta, err := nuevasDependenciasAltaContratacionTemporalDesarrollo(
		dependencias, origen,
	)
	if err != nil {
		return nil, nil, nil, err
	}
	cerrarAlta := true
	defer func() {
		if cerrarAlta {
			alta.cerrar()
		}
	}()
	fuenteMotivosRectificacion, err := nuevaFuenteMotivosRectificacionAnalisisDesarrolloConfigurada(cfg, reloj)
	if err != nil {
		return nil, nil, nil, err
	}
	reglasAnalisis, err := nuevasFuentesReglasAnalisisDesarrollo(cfg, reloj)
	if err != nil {
		return nil, nil, nil, err
	}
	servicioAnalisis, err := nuevasDependenciasAnalisisContratacionTemporalDesarrollo(
		dependencias,
		&alta,
		fuenteMotivosRectificacion,
		reglasAnalisis.retribuciones,
		catalogoDesarrollo,
	)
	if err != nil {
		return nil, nil, nil, err
	}
	coberturaReal, err := nuevasDependenciasCoberturaContratacionTemporalDesarrollo(
		dependencias,
		&alta,
		catalogoDesarrollo,
	)
	if err != nil {
		return nil, nil, nil, err
	}
	asignacionReal, err := nuevasDependenciasAsignacionContratacionTemporalDesarrollo(
		derivador,
		&alta,
		reloj,
	)
	if err != nil {
		return nil, nil, nil, err
	}
	informeJuridicoReal, err := nuevasDependenciasInformeJuridicoContratacionTemporalDesarrollo(
		derivador,
		&alta,
		reloj,
	)
	if err != nil {
		return nil, nil, nil, err
	}
	fiscalizacionReal, err := nuevasDependenciasFiscalizacionContratacionTemporalDesarrollo(
		resolvedorDesarrollo,
		derivador,
		&alta,
		sello,
		reloj,
	)
	if err != nil {
		return nil, nil, nil, err
	}
	var subsanacionReal dependenciasSubsanacionReparosContratacionTemporalDesarrollo
	if strings.TrimSpace(cfg.ContratacionTemporalSubsanacionPoliticaFile) != "" {
		politica, causa := cargarConfiguracionPoliticaSubsanacionReparosDesarrollo(cfg)
		if causa != nil {
			log.Print("contratacion temporal: subsanacion no disponible; etapa=configuracion")
		} else {
			fuente := fuentePoliticaSubsanacionReparosDesarrollo{soporte: alta.soporte, configuracion: politica}
			if fuente.configurar(&alta) != nil {
				log.Print("contratacion temporal: subsanacion no disponible; etapa=fuente")
			} else {
				var causaDependencias error
				subsanacionReal, causaDependencias = nuevasDependenciasSubsanacionReparosContratacionTemporalDesarrollo(derivador, &alta, fuente, reloj)
				if causaDependencias != nil {
					log.Print("contratacion temporal: subsanacion no disponible; etapa=dependencias")
				}
			}
		}
	}
	cerrarCobertura := true
	defer func() {
		if cerrarCobertura {
			coberturaReal.cerrar()
		}
	}()
	rutaCatalogosAlta, err := nuevaRutaCatalogosAltaContratacionTemporalDesarrollo(origen)
	if err != nil {
		return nil, nil, nil, err
	}
	rutaConfiguracionAnalisis, err := nuevaRutaConfiguracionAnalisisConReglasDesarrollo(
		subsanacionReal.servicio != nil,
		fuenteMotivosRectificacion,
		reglasAnalisis.jornada,
		catalogoDesarrollo,
	)
	if err != nil {
		return nil, nil, nil, err
	}
	rutasOrganizacion, err := nuevasRutasOrganizacionContratacionTemporalDesarrollo(cfg, &alta, reloj)
	if err != nil {
		return nil, nil, nil, err
	}
	var seleccionReal httpinterno.EjecutorSeleccionLlamamiento = noCompuesta
	var autoridadPropuestaReal httpinterno.AutoridadServidorPropuestaFormalizacion = noCompuesta
	var propuestaReal httpinterno.EjecutorPropuestaFormalizacion = noCompuesta
	var comunicacionReal http.Handler
	var respuestaRecibidaReal http.Handler
	if alta.postgresql.bolsa != nil {
		seleccionReal, comunicacionReal, err = nuevasDependenciasLlamamientoContratacionTemporalDesarrollo(cfg, &alta, derivador, reloj, origen.etiquetasReferenciasCatalogosAlta())
		if err != nil {
			return nil, nil, nil, err
		}
		respuestaRecibidaReal, err = nuevoManejadorRespuestaRecibidaDesarrollo(&alta, reloj)
		if err != nil {
			return nil, nil, nil, err
		}
		propuesta, err := nuevasDependenciasPropuestaFormalizacionDesarrollo(&alta, reloj)
		if err != nil {
			return nil, nil, nil, err
		}
		autoridadPropuestaReal, propuestaReal = propuesta, propuesta
	}
	var cuadroReal httpinterno.ConsultorCuadroRRHH = &consultorCuadroNoCompuestoContratacionTemporalDesarrollo{noCompuesta}
	var detalleReal httpinterno.ConsultorDetalleRRHH = &consultorDetalleNoCompuestoContratacionTemporalDesarrollo{noCompuesta}
	var originalPropuestaReal httpinterno.ConsultorDetalleRRHH = &consultorDetalleNoCompuestoContratacionTemporalDesarrollo{noCompuesta}
	consultasRRHH := dependenciasConsultasRRHHDesarrollo{cerrar: func() {}}
	var borradorRRHH ports.RenderizadorBorradorRRHH
	var borradorRRHHDOCX httpinterno.RenderizadorBorradorRRHHDOCX
	alta.soporte.mu.Lock()
	perfilCTCatalogo := alta.soporte.contexto.Resultado.Contexto.PerfilActivoRef
	alta.soporte.mu.Unlock()
	lectoresDeclarados := resolvedorDesarrollo.lectoresConsultaRRHH()
	lectoresConsulta := make([]string, 0, len(lectoresDeclarados))
	for _, lector := range lectoresDeclarados {
		lectoresConsulta = append(lectoresConsulta, lector.perfilRef)
	}
	perfilesConsulta := perfilesConsultaContratacionTemporalDesarrollo(perfilCTCatalogo, lectoresConsulta)
	declaracionesFrontera := descriptoresFronterasContratacionTemporalDesarrollo(perfilCTCatalogo, perfilesConsulta)
	if candidato := resolvedorDesarrollo.candidatoBolsa; candidato != nil {
		if !debeComponerMiBolsaDesarrollo(cfg) || !perfilActivoSeguridadComunValido(candidato.perfilRef) {
			return nil, nil, nil, errMiBolsaNoDisponible
		}
		declaracionesFrontera = append(declaracionesFrontera, descriptorFronteraComunDesarrollo{
			Clave:      "bolsa-mi-bolsa-consultar",
			Superficie: superficieExternaPersonalSeguridadComunDesarrollo,
			Metodo:     http.MethodGet, Ruta: bolsapersonal.RutaMiBolsa,
			PerfilesActivosRef: []string{candidato.perfilRef},
			ClavePolitica:      "politica-bolsa-mi-bolsa", ClaveCapacidad: "capacidad-bolsa-mi-bolsa-consultar",
		})
	}
	var soporteBolsaCatalogo *soporteSesionBorradorBolsaDesarrollo
	if debeComponerBorradorLlamamientoDesarrollo(cfg) {
		soporteBolsaCatalogo, err = nuevoSoporteSesionBorradorBolsaDesarrollo(cfg.DevelopmentMaterialDir, alta.soporte, reloj.Ahora())
		if err != nil {
			return nil, nil, nil, fmt.Errorf("%w: %w", errBorradorNoDisponibleEn(), err)
		}
		perfilBolsa := soporteBolsaCatalogo.soporteCanal.contexto.Resultado.Contexto.PerfilActivoRef
		bolsaFronteras, e := descriptoresFronterasBorradorLlamamientoBolsaDesarrollo(perfilBolsa)
		if e != nil {
			return nil, nil, nil, errBorradorNoDisponibleEn()
		}
		declaracionesFrontera = append(declaracionesFrontera, bolsaFronteras...)
	}
	catalogoFronteras, err := nuevoCatalogoFronterasComunDesarrollo(declaracionesFrontera)
	if err != nil {
		return nil, nil, nil, errBorradorNoDisponibleEn()
	}
	if cfg.ContratacionTemporalPostgreSQL.ConsultasRRHHConfiguradas() {
		plantillas, err := cargarPlantillasBorradorCTDesarrollo(cfg, reloj.Ahora())
		if err != nil {
			return nil, nil, nil, err
		}
		consultasRRHH, err = nuevasDependenciasConsultasRRHHDesarrollo(dependencias, &alta, catalogoFronteras)
		if err != nil {
			return nil, nil, nil, err
		}
		cuadroReal, detalleReal, originalPropuestaReal = consultasRRHH.cuadroHTTP, consultasRRHH.detalleHTTP, consultasRRHH.originalPropuestaHTTP
		etiquetas := origen.etiquetasReferenciasCatalogosAlta()
		borradorRRHH = informejuridico.RenderizadorBorradorDesarrollo{PDF: pdfvec.Renderizador{}, Etiquetas: etiquetas, Plantillas: plantillas}
		borradorRRHHDOCX = informejuridico.RenderizadorBorradorDOCXDesarrollo{DOCX: docxvec.Renderizador{}, Etiquetas: etiquetas, Plantillas: plantillas}
	}
	defer func() {
		if cerrarAlta {
			consultasRRHH.cerrar()
		}
	}()
	var incorporacionV2 *inc.ServidorV2PostgreSQL
	cerrarIncorporacion := func() {}
	if cfg.IncorporacionV2File != "" {
		var preparada ConfiguracionIncorporacionDesarrollo
		preparada, cerrarIncorporacion, err = cargarIncorporacionV2Desarrollo(cfg, &alta, consultasRRHH, reloj, catalogoFronteras)
		if err != nil {
			return nil, nil, nil, err
		}
		incorporacion = []ConfiguracionIncorporacionDesarrollo{preparada}
	}
	defer func() {
		if cerrarAlta {
			cerrarIncorporacion()
		}
	}()
	if len(incorporacion) == 1 {
		incorporacionV2, err = nuevasDependenciasIncorporacionV2Desarrollo(incorporacion[0], &alta, consultasRRHH, reloj)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	presentacionFlujoRRHH, err := LeerLectorFlujoVisualRRHH(
		strings.NewReader(config.PresentacionFlujoRRHHDesarrollo()),
	)
	if err != nil {
		return nil, nil, nil, err
	}
	var autoridadSubsanacion httpinterno.AutoridadContextoCanalSubsanacionReparos
	var ejecutorSubsanacion httpinterno.EjecutorSubsanacionReparos
	if subsanacionReal.servicio != nil && subsanacionReal.autoridad != nil {
		autoridadSubsanacion = subsanacionReal.autoridad
		ejecutorSubsanacion = subsanacionReal.servicio
	}
	rutas, err := contratacioncomposicion.NuevasRutas(
		contratacioncomposicion.DependenciasRutas{
			PresentacionFlujoRRHH:           presentacionFlujoRRHH,
			IncorporacionV2:                 incorporacionV2,
			AutoridadAlta:                   alta.soporte,
			EjecutorAlta:                    alta.servicio,
			Reloj:                           reloj,
			AutoridadCobertura:              alta.soporte,
			Presentador:                     coberturaReal.presentador,
			Decisor:                         coberturaReal.decisor,
			ConsultorResultado:              coberturaReal.consultor,
			AutoridadAnalisis:               alta.soporte,
			EjecutorAnalisis:                servicioAnalisis,
			ConsultorCuadroRRHH:             cuadroReal,
			ConsultorDetalleRRHH:            detalleReal,
			ConsultorOriginalPropuestaRRHH:  originalPropuestaReal,
			BorradorRRHH:                    borradorRRHH,
			BorradorRRHHDOCX:                borradorRRHHDOCX,
			EjecutorSeleccion:               seleccionReal,
			AutoridadPropuestaFormalizacion: autoridadPropuestaReal,
			EjecutorPropuestaFormalizacion:  propuestaReal,
			AutoridadCierreAdministrativo:   noCompuesta,
			EjecutorCierreAdministrativo:    noCompuesta,
			AutoridadAsignacion:             alta.soporte,
			EjecutorAsignacion:              asignacionReal,
			AutoridadInformeJuridico:        alta.soporte,
			EjecutorInformeJuridico:         informeJuridicoReal,
			AutoridadFiscalizacion:          fiscalizacionReal.soporte,
			EjecutorFiscalizacion:           fiscalizacionReal.servicio,
			AutoridadSubsanacionReparos:     autoridadSubsanacion,
			EjecutorSubsanacionReparos:      ejecutorSubsanacion,
		},
	)
	if err != nil {
		return nil, nil, nil, err
	}
	rutas = append(rutas, rutaCatalogosAlta, rutaConfiguracionAnalisis)
	if incorporacionV2 != nil {
		mapeo, err := nuevoMapeoFichaGINPIXDesarrollo()
		if err != nil {
			return nil, nil, nil, err
		}
		ficha, err := contratacioncomposicion.NuevaRutaFichaGINPIXV2(incorporacionV2, mapeo)
		if err != nil {
			return nil, nil, nil, err
		}
		rutas = append(rutas, ficha)
		seguimiento, err := contratacioncomposicion.NuevaRutaConsultaSeguimientoV2(incorporacionV2)
		if err != nil {
			return nil, nil, nil, err
		}
		rutas = append(rutas, seguimiento)
		for i := range rutas {
			if rutas[i].Ruta == httpinterno.RutaIncorporacionEjercicioV2 || rutas[i].Ruta == httpinterno.RutaFichaGINPIXV2 || rutas[i].Ruta == httpinterno.RutaConsultaSeguimientoV2 {
				rutas[i].Manejador = ligarContextoIncorporacionV2Desarrollo(rutas[i].Manejador, alta.soporte, catalogoFronteras)
			}
		}
	}
	if len(incorporacion) == 1 && incorporacion[0].continuidad != nil {
		continuidad, err := incorporacion[0].continuidad.rutas(derivador, catalogoFronteras)
		if err != nil {
			return nil, nil, nil, err
		}
		rutas = append(rutas, continuidad...)
	}
	if comunicacionReal != nil && consultasRRHH.detalle != nil && borradorRRHH != nil {
		resolucion, err := nuevasDependenciasResolucionFormalizacionDesarrollo(&alta, reloj, consultasRRHH.detalle, borradorRRHH, consultasRRHH.preparacionResolucion, catalogoFronteras)
		if err != nil {
			return nil, nil, nil, err
		}
		h, err := httpinterno.NuevoManejadorResolucionFormalizacion(resolucion, resolucion)
		if err != nil {
			return nil, nil, nil, err
		}
		rutas = append(rutas, vechttp.RutaExacta{Ruta: httpinterno.RutaResolucionFormalizacion, Manejador: h})
	}
	rutas = append(rutas, rutasOrganizacion...)
	if consultasRRHH.estadisticas != nil {
		h, err := httpinterno.NuevoManejadorEstadisticasRRHH(consultasRRHH.estadisticas,
			&resolutorAlcanceEstadisticasRRHHDesarrollo{sello: sello, resolvedor: resolvedorDesarrollo}, reloj.Ahora)
		if err != nil {
			return nil, nil, nil, err
		}
		rutas = append(rutas, vechttp.RutaExacta{Ruta: httpinterno.RutaEstadisticasRRHH, Manejador: h})
	}
	rutasPeticionesCentro, err := nuevasRutasPeticionCentroDesarrollo(cfg, resolvedorDesarrollo, &alta, reloj, catalogoDesarrollo)
	if err != nil {
		return nil, nil, nil, err
	}
	rutas = append(rutas, rutasPeticionesCentro...)
	if len(rutasPeticionesCentro) > 0 {
		rutaEntrega, err := nuevaRutaEntregaPeticionDesarrollo(&alta, reloj)
		if err != nil {
			return nil, nil, nil, err
		}
		rutas = append(rutas, rutaEntrega)
	}
	if comunicacionReal != nil {
		// La continuación confirma solo después de abrir en Bolsa; no implica
		// envío de aviso ni aplicación de un plazo legal.
		rutas = append(rutas, vechttp.RutaExacta{Ruta: httpinterno.RutaRegistroComunicacionLlamamiento, Manejador: comunicacionReal})
		rutas = append(rutas, vechttp.RutaExacta{Ruta: httpinterno.RutaResolucionComunicacionLlamamiento, Manejador: comunicacionReal})
		rutas = append(rutas, vechttp.RutaExacta{Ruta: httpinterno.RutaContinuacionLlamamiento, Manejador: comunicacionReal})
	}
	if respuestaRecibidaReal != nil {
		rutas = append(rutas, vechttp.RutaExacta{Ruta: httpinterno.RutaRegistroRespuestaRecibida, Manejador: respuestaRecibidaReal})
	}
	rutasBorrador := []vechttp.RutaExacta(nil)
	coleccionesBorrador := []vechttp.RutaColeccion(nil)
	// Instancia única construida antes de las sesiones CT y Bolsa.
	seguridadBorrador := catalogoFronteras
	var envolverBorrador func(http.Handler) http.Handler
	var manejadorSituacion http.Handler
	cerrarBorrador := func() {}
	if debeComponerBorradorLlamamientoDesarrollo(cfg) {
		if consultasRRHH.identidad == nil {
			return nil, nil, nil, errBorradorNoDisponibleEn()
		}
		var errBorrador error
		rutasBorrador, coleccionesBorrador, manejadorSituacion, seguridadBorrador, envolverBorrador, cerrarBorrador, errBorrador = nuevasDependenciasBorradorLlamamientoDesarrollo(
			context.Background(), cfg, dependencias, &alta, soporteBolsaCatalogo, catalogoFronteras, consultasRRHH.identidad,
		)
		if errBorrador != nil {
			return nil, nil, nil, errBorrador
		}
	}
	cerrarBorradorPendiente := true
	defer func() {
		if cerrarBorradorPendiente {
			cerrarBorrador()
		}
	}()
	rutas = append(rutas, rutasBorrador...)
	if resolvedorDesarrollo.candidatoBolsa != nil {
		if !debeComponerMiBolsaDesarrollo(cfg) || consultasRRHH.identidad == nil {
			return nil, nil, nil, errMiBolsaNoDisponible
		}
		miBolsa, err := nuevaRutaMiBolsaDesarrollo(
			context.Background(), resolvedorDesarrollo.candidatoBolsa, sello, &alta,
			consultasRRHH.identidad, catalogoFronteras, derivador, reloj,
		)
		if err != nil {
			return nil, nil, nil, err
		}
		rutas = append(rutas, vechttp.RutaExacta{Ruta: bolsapersonal.RutaMiBolsa, Manejador: miBolsa})
	}
	autoridad := &autoridadConsultasContratacionTemporalDesarrollo{
		sello:                                    sello,
		resolvedor:                               resolvedorDesarrollo,
		noCompuesta:                              noCompuesta,
		llamamientoCompuesto:                     comunicacionReal != nil,
		consultasRRHHCompuestas:                  consultasRRHH.cuadro != nil && consultasRRHH.detalle != nil,
		subsanacionCompuesta:                     subsanacionReal.servicio != nil,
		fronterasSeguridadComun:                  seguridadBorrador,
		envolverBorradorLlamamiento:              envolverBorrador,
		manejadorSituacionParticipacion:          manejadorSituacion,
		plazosOfertasBolsa:                       dependencias.plazosOfertasBolsa,
		coleccionesAdicionales:                   coleccionesBorrador,
		registradorAuditoriaFronteraRutasExactas: alta.postgresql.registradorAuditoriaFrontera,
		materialDietas:                           alta.postgresql.materialDietas,
		materialCronos:                           alta.postgresql.materialCronos,
		materialDocumentos:                       alta.postgresql.materialDocumentos,
		materialPersonalFichaPropia:              alta.postgresql.materialPersonalFichaPropia,
		presentadorCobertura:                     coberturaReal.presentador,
	}
	if autoridad.registradorAuditoriaFronteraRutasExactas == nil {
		return nil, nil, nil, errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	dependencias.cerrar = func() {
		cerrarBorrador()
		cerrarIncorporacion()
		consultasRRHH.cerrar()
		coberturaReal.cerrar()
		alta.cerrar()
	}
	cerrarCobertura = false
	cerrarAlta = false
	cerrarBorradorPendiente = false
	return rutas, autoridad, dependencias.Cerrar, nil
}

func debeComponerBorradorLlamamientoDesarrollo(cfg config.Config) bool {
	return cfg.BolsaBorradoresEnabled
}
