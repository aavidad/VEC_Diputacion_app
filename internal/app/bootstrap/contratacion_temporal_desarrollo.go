package bootstrap

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"io"
	"log"
	"net/http"
	"os"
	"reflect"
	"strings"
	"time"
	"vec-diputacion-granada/config"
	contratacioncomposicion "vec-diputacion-granada/internal/app/composicion/interna/contrataciontemporal"
	inc "vec-diputacion-granada/internal/app/incorporacionejercicio"
	bolsapersonal "vec-diputacion-granada/internal/modules/bolsa/adapters/httppersonal"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/catalogoalta"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/informejuridico"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/numeracion"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	ctrpt "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/rptpublica"
	plantillasapp "vec-diputacion-granada/internal/modules/contrataciontemporal/application/plantillascatalogo"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	personalrpt "vec-diputacion-granada/internal/modules/personal/adapters/rptpublica"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	docxvec "vec-diputacion-granada/internal/vec/adapters/documentos/docx"
	pdfvec "vec-diputacion-granada/internal/vec/adapters/documentos/pdf"
	"vec-diputacion-granada/internal/vec/adapters/fichero"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	postgresvec "vec-diputacion-granada/internal/vec/adapters/postgres"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	"vec-diputacion-granada/internal/vec/auditoria"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/reglas"
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
	metodo                  string
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
	sello                                            *selloConsultasContratacionTemporalDesarrollo
	resolvedor                                       *resolvedorIdentidadDesarrollo
	noCompuesta                                      *capacidadNoCompuestaContratacionTemporalDesarrollo
	llamamientoCompuesto                             bool
	consultasRRHHCompuestas                          bool
	subsanacionCompuesta                             bool
	fronterasSeguridadComun                          catalogoFronterasComunDesarrollo
	envolverBorradorLlamamiento                      func(http.Handler) http.Handler
	manejadorSituacionParticipacion                  http.Handler
	coleccionesAdicionales                           []vechttp.RutaColeccion
	registradorAuditoriaFronteraRutasExactas         puertosvec.RegistradorAuditoriaFronteraRutaExacta
	materialDietas                                   materialDietasDesdeCTDesarrollo
	materialCronos                                   materialCronosDesdeCTDesarrollo
	materialDocumentos                               *proveedorMaterialAltaContratacionTemporalDesarrollo
	materialInscripcionBolsa                         materialInscripcionRRHHDesarrollo
	materialPersonalFichaPropia                      *proveedorMaterialAltaContratacionTemporalDesarrollo
	materialPersonalExportacionServicios             *proveedorMaterialAltaContratacionTemporalDesarrollo
	materialPersonalHistoriaServicios                *proveedorMaterialAltaContratacionTemporalDesarrollo
	materialPersonalHistoriaRelaciones               *proveedorMaterialAltaContratacionTemporalDesarrollo
	gobiernoUsuariosPreferencias                     *pgxpool.Pool
	materialUsuariosPreferenciasConsultaInterna      *proveedorMaterialAltaContratacionTemporalDesarrollo
	materialUsuariosPreferenciasActualizacionInterna *proveedorMaterialAltaContratacionTemporalDesarrollo
	materialUsuariosPreferenciasConsultaExterna      *proveedorMaterialAltaContratacionTemporalDesarrollo
	materialUsuariosPreferenciasActualizacionExterna *proveedorMaterialAltaContratacionTemporalDesarrollo
	materialUsuariosCorreos                          proveedoresMaterialCorreosUsuarios
	materialUsuariosImagen                           proveedoresMaterialImagenUsuarios
	materialAspirantes                               proveedoresMaterialAspirantes
	plazosOfertasBolsa                               *calculadoraPlazoOfertaDesarrollo
	// presentadorCobertura permite activar después los avisos de la vía de
	// cobertura, cuando Bolsa y las reglas de ejemplo ya están compuestas.
	presentadorCobertura    avisosViaCoberturaConfigurable
	situacionBolsaCobertura *situacionBolsaCoberturaFijable // se fija con las bolsas constituidas
	// personalizacionB7 se enlaza con la fuente de bolsas constituidas cuando
	// la composición raíz la crea; el correo B7 la usa para los marcadores.
	personalizacionB7 *fuentePersonalizacionB7
	// firmaDocumento es nil salvo con VEC_CT_FIRMA_REGISTRO_ENABLED=true.
	firmaDocumento *firmaDocumentoCTDesarrollo
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
	reglasBolsa *reglas.Resolutor,
	incorporacion ...ConfiguracionIncorporacionDesarrollo,
) (
	[]vechttp.RutaExacta,
	*autoridadConsultasContratacionTemporalDesarrollo,
	func(),
	error,
) {
	return nuevasRutasContratacionTemporalConReglasDesarrollo(cfg, reglasEjemploDesarrollo{bolsa: reglasBolsa}, resolvedor, derivador, kms, registro, incorporacion...)
}

// La selección de material y la raíz consultan la misma configuración privada.
// La carga final vuelve a validarla antes de exponer cualquier ruta.
func protocolosIncorporacionConfiguradosDesarrollo(cfg config.Config) (b2, legado bool, err error) {
	cfg = cfg.Normalize()
	if cfg.IncorporacionV2File == "" {
		return false, false, nil
	}
	c, raiz, err := leerConfiguracionIncorporacionV2(cfg.IncorporacionV2File)
	if err != nil {
		return false, false, err
	}
	defer raiz.Close()
	return c.PersonalB2 != nil, c.Planes != "", nil
}

func descriptoresFronterasIncorporacionB2Desarrollo() []descriptorFronteraComunDesarrollo {
	planGET := fronteraContratacionTemporalDesarrollo("ct-incorporacion-b2-plan-consultar", ports.AccionLeerPlanNominalB2, httpinterno.RutaPlanB2, nil)
	planGET.Metodo = http.MethodGet
	return []descriptorFronteraComunDesarrollo{
		planGET,
		fronteraContratacionTemporalDesarrollo("ct-incorporacion-b2-plan-registrar", ports.AccionRegistrarPlanNominalB2, httpinterno.RutaPlanB2, nil),
		fronteraContratacionTemporalDesarrollo("ct-incorporacion-b2-origen-confirmar", ports.AccionConfirmarOrigenB2, httpinterno.RutaConfirmacionB2, nil),
	}
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
	b2Configurada, legadoConfigurado, err := protocolosIncorporacionConfiguradosDesarrollo(cfg)
	if err != nil {
		return nil, nil, nil, err
	}
	dependencias, err := nuevasDependenciasCT(cfg, resolvedor, derivador, kms, registro)
	if err != nil {
		return nil, nil, nil, err
	}
	dependencias.reglasEjemplo = reglasEjemplo
	reglasBolsa := reglasEjemplo.bolsa
	// El plazo de respuesta del llamamiento lo rige el Reglamento de bolsas:
	// se gobierna con el catálogo de reglas de Bolsa.
	reglasLlamamiento := reglasEjemplo.bolsa
	dependencias.plazosFase = nuevaCalculadoraPlazoFaseCT(reglasEjemplo.contratacionTemporal)
	if err := dependencias.cfg.ContratacionTemporalPostgreSQL.ValidarIdentidadOperativa(); err != nil {
		return nil, nil, nil, err
	}
	cfg, resolvedorDesarrollo, derivador, registro := dependencias.cfg, dependencias.resolvedor, dependencias.derivador, dependencias.registro
	plantillasActivas, err := plantillasCatalogoCTDesarrolloSolicitado(cfg)
	ajustesCTActivos, errAjustes := ajustesReglasCTSolicitados(cfg)
	if errAjustes != nil || ajustesCTActivos && !cfg.ContratacionTemporalPostgreSQL.ConsultasRRHHConfiguradas() {
		return nil, nil, nil, errMontajeAjustesReglasCT
	}
	if err != nil {
		return nil, nil, nil, err
	}
	documentalActiva, err := plantillasDocumentalCTDesarrolloSolicitado(cfg)
	if err != nil {
		return nil, nil, nil, err
	}
	configuracionPreparacionBases, preparacionBasesActiva, err := leerConfiguracionPreparacionBasesV3(cfg)
	if err != nil || preparacionBasesActiva && (!cfg.BolsaBorradoresEnabled || !cfg.ContratacionTemporalPostgreSQL.ConsultasRRHHConfiguradas()) {
		return nil, nil, nil, errMontajePreparacionBasesV3
	}
	var fuenteAutorizacionPlantillas, motivosEvaluadorPlantillas *pgxpool.Pool
	var autoridadesCargaConvoca *autoridadesCargaConvocaPostgreSQL
	cerrarAutoridadesPlantillas := func() {
		if motivosEvaluadorPlantillas != nil {
			motivosEvaluadorPlantillas.Close()
		}
		if fuenteAutorizacionPlantillas != nil {
			fuenteAutorizacionPlantillas.Close()
		}
	}
	cerrarAutoridadesPlantillasPendiente := true
	defer func() {
		if cerrarAutoridadesPlantillasPendiente {
			cerrarAutoridadesPlantillas()
		}
	}()
	autoridadesBolsaConfiguradas := debeComponerBorradorLlamamientoDesarrollo(cfg) &&
		(strings.TrimSpace(os.Getenv(config.EnvAutorizacionFuenteDatabaseURL)) != "" ||
			strings.TrimSpace(os.Getenv(config.EnvAutorizacionMotivosEvaluadorDatabaseURL)) != "")
	if plantillasActivas || documentalActiva || preparacionBasesActiva || ajustesCTActivos || autoridadesBolsaConfiguradas {
		if !cfg.ContratacionTemporalPostgreSQL.ConsultasRRHHConfiguradas() {
			return nil, nil, nil, plantillasapp.ErrNoDisponible
		}
		dsnFuente, dsnMotivos, err := cfg.DSNAutoridadesAutorizacionRRHH()
		if err != nil {
			return nil, nil, nil, err
		}
		sonda, cancelar := context.WithTimeout(context.Background(), plazoarranque.Ampliar(15*time.Second))
		defer cancelar()
		fuenteAutorizacionPlantillas, err = abrirPoolAutorizacionRRHHDesarrollo(
			sonda, dsnFuente, config.RolAutorizacionFuenteRRHH, "vec-ct-plantillas-fuente")
		if err != nil {
			return nil, nil, nil, plantillasapp.ErrNoDisponible
		}
		motivosEvaluadorPlantillas, err = abrirPoolAutorizacionRRHHDesarrollo(
			sonda, dsnMotivos, config.RolAutorizacionMotivosEvaluadorRRHH, "vec-ct-plantillas-motivos")
		if err != nil || preflightAutoridadesPlantillasCT(sonda, fuenteAutorizacionPlantillas, motivosEvaluadorPlantillas) != nil {
			return nil, nil, nil, plantillasapp.ErrNoDisponible
		}
		if debeComponerBorradorLlamamientoDesarrollo(cfg) {
			autoridadesCargaConvoca = &autoridadesCargaConvocaPostgreSQL{
				fuente: fuenteAutorizacionPlantillas, motivos: motivosEvaluadorPlantillas,
			}
		}
	}
	noCompuesta, err := nuevaCapacidadNoCompuestaContratacionTemporalDesarrollo(registro)
	if err != nil {
		return nil, nil, nil, err
	}
	catalogoDesarrollo, err := nuevoCatalogoDesarrollo(cfg.PersonalOrganizacionSourcePath, cfg.RPTCatalogoPath, cfg.CTNecesidadesAltaSourcePath)
	if err != nil {
		return nil, nil, nil, err
	}
	politicaNumero, err := numeracion.Cargar(cfg.CTNumeroExpedienteSourcePath)
	if err != nil {
		return nil, nil, nil, err
	}
	catalogoDesarrollo.NumeroExpedienteMOAD = &politicaNumero
	origen := nuevoOrigenConsultasConCatalogoDesarrollo(catalogoDesarrollo)
	sello := dependencias.sello
	reloj := dependencias.reloj
	// Las reglas del análisis se resuelven antes del alta: al abrir
	// PostgreSQL se publican sus vías de cobertura y su numeración.
	reglasAnalisis, err := nuevasFuentesReglasAnalisisDesarrollo(cfg, reloj)
	if err != nil {
		return nil, nil, nil, err
	}
	dependencias.opcionesCatalogoCT = reglasAnalisis.opciones
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
	if ajustesCTActivos {
		if err := conectarAjustesReglasCTAlResolutor(cfg, reglasEjemplo.contratacionTemporal,
			alta.postgresql.ejecucion, reglasEjemplo.calendarios, reloj); err != nil {
			return nil, nil, nil, err
		}
	}
	// El alta de necesidad queda cerrada hasta declarar una publicación propia
	// y disponer del confirmador CT193 y de la lectura pública exacta de RPT.
	// La relectura de la instantánea usa la conexión del ejecutor: es el único
	// rol al que CT193 concede leer_instantanea_necesidad_alta_v3.
	if cfg.CTNecesidadesAltaSourcePath != "" {
		recuperador, err := postgresct.NuevoRecuperadorNecesidadAltaPostgreSQL(alta.postgresql.ejecucion)
		if err != nil {
			return nil, nil, nil, err
		}
		fuenteNecesidades, err := catalogoalta.NuevaFuente(cfg.CTNecesidadesAltaSourcePath, recuperador)
		if err != nil {
			return nil, nil, nil, err
		}
		fuenteRPT, err := personalrpt.NuevaFuente(cfg.RPTCatalogoPath)
		if err != nil {
			return nil, nil, nil, err
		}
		if _, err := fuenteRPT.ObtenerRPTPublica(context.Background()); err != nil {
			return nil, nil, nil, err
		}
		verificadorRPT, err := ctrpt.NuevoVerificadorAlta(fuenteRPT)
		if err != nil {
			return nil, nil, nil, err
		}
		if err := alta.servicio.ConfigurarFuenteNecesidadesAlta(fuenteNecesidades); err != nil {
			return nil, nil, nil, err
		}
		if err := alta.servicio.ConfigurarVerificadorPuestoRPTAlta(verificadorRPT); err != nil {
			return nil, nil, nil, err
		}
	}
	alta.soporte.reglasPlazo = reglasPlazoLlamamientoDesarrollo{resolutor: reglasLlamamiento}
	montajePreparacionBases, existePreparacionBases, err := NuevoMontajePreparacionBasesV3(cfg, alta.soporte, reloj)
	if err != nil || existePreparacionBases != preparacionBasesActiva || preparacionBasesActiva && montajePreparacionBases.configuracion != configuracionPreparacionBases {
		return nil, nil, nil, errMontajePreparacionBasesV3
	}
	montajeBaremo, err := prepararMontajeGobiernoReglasBaremoHTTPV3(cfg, alta.soporte, reloj)
	if err != nil {
		return nil, nil, nil, err
	}
	var consultaCircuitoRRHH http.Handler
	if cfg.CTCircuitoRRHHSourcePath != "" {
		if err := preflightCircuitoRRHHDesarrollo(alta.postgresql.ejecucion); err != nil {
			return nil, nil, nil, err
		}
		consultaCircuitoRRHH, err = nuevoManejadorConsultaCircuitoRRHHDesarrollo(cfg, &alta, derivador, reloj)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	firmasR5V2, err := nuevasRutasFirmasR5V2CTDesarrollo(cfg, &alta, derivador, reloj)
	if err != nil {
		return nil, nil, nil, err
	}
	if firmasR5V2 != nil {
		alta.postgresql.cerrarFirmasR5V2 = firmasR5V2.cerrar
	}
	for _, descriptor := range descriptoresMaterialIncorporacionB2() {
		_, seleccionada := alta.postgresql.catalogoMaterial.descriptorPara(descriptor.Audiencia)
		if seleccionada != b2Configurada {
			return nil, nil, nil, ErrComposicionDesarrolloIncompleta
		}
	}
	var soportePlantillas *soporteAltaContratacionTemporalDesarrollo
	var perfilPlantillas string
	if plantillasActivas {
		soportePlantillas, perfilPlantillas, err = nuevoSoportePlantillasCatalogoCTDesdeBaseDesarrollo(alta.soporte, reloj.Ahora())
		if err != nil {
			return nil, nil, nil, err
		}
	}
	var soporteDocumental *soporteAltaContratacionTemporalDesarrollo
	var perfilDocumental string
	if documentalActiva {
		soporteDocumental, perfilDocumental, err = nuevoSoportePlantillasDocumentalCTDesdeBaseDesarrollo(alta.soporte, reloj.Ahora())
		if err != nil {
			return nil, nil, nil, err
		}
	}
	fuenteMotivosRectificacion, err := nuevaFuenteMotivosRectificacionAnalisisDesarrolloConfigurada(cfg, reloj)
	if err != nil {
		return nil, nil, nil, err
	}
	dependencias.retribucionesCT = reglasAnalisis.retribuciones
	catalogoDesarrollo.componerOpcionesAnalisis(reglasAnalisis.opciones)
	if alta.postgresql.ejecucion != nil {
		ctxMigracion, cancelarMigracion := context.WithTimeout(context.Background(), plazoarranque.Ampliar(5*time.Second))
		err = comprobarMigracionUrgenciaAnalisis(ctxMigracion, alta.postgresql.ejecucion)
		cancelarMigracion()
		if err != nil {
			return nil, nil, nil, err
		}
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
		aprobacionProvisionPerfilesRRHHDesdeConfig(cfg),
	)
	if err != nil {
		return nil, nil, nil, err
	}
	if err := gobernarResultadosFiscalizacionDesarrollo(fiscalizacionReal.servicio, reglasEjemplo.contratacionTemporal); err != nil {
		return nil, nil, nil, err
	}
	fuenteInformeNuevo, err := gobernarInformeTrasSubsanacionDesarrollo(
		reglasEjemplo.contratacionTemporal, alta.postgresql.ejecucion, alta.postgresql.gobierno,
		fiscalizacionReal.servicio, informeJuridicoReal,
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
			if fuente.configurar(&alta, aprobacionProvisionPerfilesRRHHDesdeConfig(cfg)) != nil {
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
	firmaDocumento, err := nuevaFirmaDocumentoCTDesarrollo(cfg, &alta, reloj, fiscalizacionReal.servicio)
	if err != nil {
		return nil, nil, nil, err
	}
	if firmaDocumento != nil {
		firmaDocumento.informeTrasSubsanacion = fuenteInformeNuevo
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
	var consultaReciboRespuestaReal http.Handler
	var consultaComunicacionesReal http.Handler
	var eventoPlazoReal http.Handler
	if alta.postgresql.bolsa != nil {
		if alta.postgresql.gobierno == nil || alta.postgresql.registroAutorizacion == nil || alta.postgresql.ejecucion == nil {
			return nil, nil, nil, ports.ErrConsultaRRHHNoDisponible
		}
		ctxIntentos, cancelarIntentos := context.WithTimeout(context.Background(), plazoarranque.Ampliar(30*time.Second))
		reservadosIntentos := []string{
			alta.postgresql.gobierno.Config().ConnConfig.User,
			alta.postgresql.registroAutorizacion.Config().ConnConfig.User,
			alta.postgresql.bolsa.Config().ConnConfig.User,
		}
		alta.auditoriaLecturasCT, alta.procesoAuditoriaLecturasCT, alta.cerrarAuditoriaLecturasCT, err =
			AbrirRegistradorIntentosAuditoriaDesarrollo(ctxIntentos, cfg, alta.postgresql.ejecucion, reservadosIntentos)
		cancelarIntentos()
		if err != nil {
			return nil, nil, nil, err
		}
		seleccionReal, comunicacionReal, err = nuevasDependenciasLlamamientoContratacionTemporalDesarrollo(cfg, &alta, derivador, reloj, origen.etiquetasReferenciasCatalogosAlta())
		if err != nil {
			return nil, nil, nil, err
		}
		respuestaRecibidaReal, err = nuevoManejadorRespuestaRecibidaDesarrollo(&alta, reloj)
		if err != nil {
			return nil, nil, nil, err
		}
		consultaReciboRespuestaReal, err = nuevoManejadorConsultaReciboRespuestaDesarrollo(&alta, reloj)
		if err != nil || alta.postgresql.registradorAuditoriaFrontera == nil {
			return nil, nil, nil, ports.ErrConsultaReciboRespuestaFallo
		}
		consultaReciboRespuestaReal = auditorConsultaReciboRespuestaDenegada{
			siguiente: consultaReciboRespuestaReal, registrador: alta.postgresql.registradorAuditoriaFrontera, soporte: alta.soporte,
		}
		consultaComunicacionesReal, err = nuevoManejadorConsultaComunicacionesExpedienteDesarrollo(&alta, derivador, reloj)
		if err != nil {
			return nil, nil, nil, err
		}
		consultaComunicacionesReal = auditorConsultaCTDenegada{
			siguiente: consultaComunicacionesReal, registrador: alta.postgresql.registradorAuditoriaFrontera,
			soporte: alta.soporte, ruta: httpinterno.RutaConsultaComunicacionesExpediente,
		}
		eventoPlazoReal, err = nuevoManejadorEventoPlazoDesarrollo(&alta, reloj)
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
	var borradorRRHH, borradorFormalizacion ports.RenderizadorBorradorRRHH
	var borradorRRHHDOCX httpinterno.RenderizadorBorradorRRHHDOCX
	alta.soporte.mu.Lock()
	perfilCTCatalogo := alta.soporte.contexto.Resultado.Contexto.PerfilActivoRef
	alta.soporte.mu.Unlock()
	auditoriaActiva, err := selectorCapacidadRRHHDesarrollo(cfg, envRRHHAuditoriaEnabled)
	if err != nil || (auditoriaActiva && (!cfg.BolsaBorradoresEnabled || !cfg.ContratacionTemporalPostgreSQL.ConsultasRRHHConfiguradas())) {
		return nil, nil, nil, ErrActivacionDesarrolloInvalida
	}
	lectoresDeclarados := resolvedorDesarrollo.lectoresConsultaRRHH()
	lectoresConsulta := make([]string, 0, len(lectoresDeclarados))
	for _, lector := range lectoresDeclarados {
		lectoresConsulta = append(lectoresConsulta, lector.perfilRef)
	}
	perfilesConsulta := perfilesConsultaContratacionTemporalDesarrollo(perfilCTCatalogo, lectoresConsulta)
	reincorporacionTitular, err := selectorCapacidadRRHHDesarrollo(cfg, envCTReincorporacionTitularEnabled)
	if err != nil || (reincorporacionTitular && (!seguimientoCeseSolicitado(cfg) || !cfg.BolsaBorradoresEnabled)) {
		return nil, nil, nil, ErrActivacionDesarrolloInvalida
	}
	perfilReincorporacionTitular := ""
	if reincorporacionTitular {
		contexto, err := nuevoContextoReincorporacionTitularDesarrollo(alta.soporte, reloj.Ahora())
		if err != nil {
			return nil, nil, nil, ErrActivacionDesarrolloInvalida
		}
		perfilReincorporacionTitular = contexto.Resultado.Contexto.PerfilActivoRef
		alta.soporte.reincorporacionTitular = &soporteSeguimientoCeseDesarrollo{contexto: contexto}
	}
	politicaOfertasActiva, err := selectorCapacidadRRHHDesarrollo(cfg, envBolsaPoliticaOfertasEnabled)
	if err != nil || (politicaOfertasActiva && !cfg.BolsaBorradoresEnabled) {
		return nil, nil, nil, ErrActivacionDesarrolloInvalida
	}
	declaracionesFrontera, err := descriptoresFronterasContratacionTemporalConPlantillasDesarrollo(
		perfilCTCatalogo, perfilesConsulta, firmaDocumento != nil, plantillasActivas, perfilPlantillas)
	if err != nil {
		return nil, nil, nil, err
	}
	if ajustesCTActivos {
		declaracionesFrontera = append(declaracionesFrontera, descriptoresFronteraAjustesReglasCT(perfilCTCatalogo)...)
	}
	if consultaCircuitoRRHH != nil {
		declaracionesFrontera = append(declaracionesFrontera,
			fronteraContratacionTemporalDesarrollo("ct-circuito-rrhh-consultar",
				postgresct.AccionConsultaCircuitoRRHH, httpinterno.RutaConsultaCircuitoRRHH,
				[]string{perfilCTCatalogo}))
	}
	declaracionesFrontera, err = asignarPerfilesFijosEnFronterasCTDesarrollo(alta.soporte, perfilCTCatalogo, declaracionesFrontera)
	if err != nil {
		return nil, nil, nil, err
	}
	if legadoConfigurado {
		declaracionesFrontera, err = asignarPerfilesNominalesIncorporacionEnFronteras(alta.soporte, declaracionesFrontera)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	if b2Configurada {
		declaracionesFrontera = append(declaracionesFrontera, descriptoresFronterasIncorporacionB2Desarrollo()...)
		declaracionesFrontera, err = asignarPerfilesNominalesB2EnFronteras(alta.soporte, declaracionesFrontera)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	if documentalActiva {
		declaracionesFrontera, err = anexarFronterasPlantillasDocumentalCTDesarrollo(
			declaracionesFrontera, perfilCTCatalogo, perfilPlantillas, perfilDocumental, perfilesConsulta)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	if reincorporacionTitular {
		declaracionesFrontera = append(declaracionesFrontera,
			descriptoresFronterasReincorporacionTitularDesarrollo(perfilReincorporacionTitular)...)
	}
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
		declaracionesFrontera = append(declaracionesFrontera, descriptorFronteraComunDesarrollo{
			Clave:      "bolsa-mi-bolsa-historial-consultar",
			Superficie: superficieExternaPersonalSeguridadComunDesarrollo,
			Metodo:     http.MethodGet, Ruta: bolsapersonal.RutaMiBolsaHistorial,
			PerfilesActivosRef: []string{candidato.perfilRef},
			ClavePolitica:      "politica-bolsa-mi-bolsa", ClaveCapacidad: "capacidad-bolsa-mi-bolsa-historial-consultar",
		})
		if debeComponerPortalCandidatoDesarrollo(cfg) {
			for clave, ruta := range map[string]string{
				"bolsa-mi-bolsa-solicitar":            bolsapersonal.RutaMiBolsaSolicitudes,
				"bolsa-mi-bolsa-solicitud-documental": bolsapersonal.RutaMiBolsaSolicitudesDocumentales,
				"bolsa-mi-bolsa-responder":            bolsapersonal.RutaMiBolsaRespuestas,
				"bolsa-mi-bolsa-disposicion":          bolsapersonal.RutaMiBolsaDisposiciones,
				"bolsa-mi-bolsa-contacto":             bolsapersonal.RutaMiBolsaContacto,
			} {
				declaracionesFrontera = append(declaracionesFrontera, descriptorFronteraComunDesarrollo{
					Clave: clave, Superficie: superficieExternaPersonalSeguridadComunDesarrollo,
					Metodo: http.MethodPost, Ruta: ruta, PerfilesActivosRef: []string{candidato.perfilRef},
					ClavePolitica: "politica-bolsa-mi-bolsa", ClaveCapacidad: "capacidad-" + clave,
				})
			}
		}
	}
	var soporteBolsaCatalogo *soporteSesionBorradorBolsaDesarrollo
	if debeComponerBorradorLlamamientoDesarrollo(cfg) {
		soporteBolsaCatalogo, err = nuevoSoporteSesionBorradorBolsaDesarrollo(cfg.DevelopmentMaterialDir, alta.soporte, reloj.Ahora())
		if err != nil {
			return nil, nil, nil, fmt.Errorf("%w: %w", errBorradorNoDisponibleEn(), err)
		}
		perfilBolsa := soporteBolsaCatalogo.soporteCanal.contexto.Resultado.Contexto.PerfilActivoRef
		cargaConvocaGobernada := false
		if autoridadesCargaConvoca != nil {
			versionBase := 5
			if politicaOfertasActiva {
				versionBase++
			}
			if reincorporacionTitular {
				versionBase += 2
			}
			sondaCarga, cancelarCarga := context.WithTimeout(context.Background(), plazoarranque.Ampliar(15*time.Second))
			cargaConvocaGobernada, err = permiteMontarCargaConvocaPostgreSQL(sondaCarga,
				autoridadesCargaConvoca.fuente, soporteBolsaCatalogo, reloj.Ahora(), versionBase)
			cancelarCarga()
			if err != nil {
				log.Printf("bolsa CONVOCA: no se pudo comprobar la concesión de carga (%T)", err)
				cargaConvocaGobernada = false
			}
		}
		bolsaFronteras, e := descriptoresFronterasBorradorLlamamientoBolsaDesarrollo(perfilBolsa, politicaOfertasActiva, reincorporacionTitular, cargaConvocaGobernada)
		if e != nil {
			return nil, nil, nil, errBorradorNoDisponibleEn()
		}
		declaracionesFrontera = append(declaracionesFrontera, bolsaFronteras...)
	}
	var soportesAuditoria soportesAuditoriaConsultaDesarrollo
	if auditoriaActiva {
		soportesAuditoria, err = nuevosSoportesAuditoriaConsultaDesarrollo(cfg, &alta, soporteBolsaCatalogo, reloj)
		if err != nil {
			return nil, nil, nil, err
		}
		fronterasAuditoria, err := descriptoresFronterasAuditoriaRRHHDesarrollo(soportesAuditoria.PerfilCT, soportesAuditoria.PerfilBolsa)
		if err != nil {
			return nil, nil, nil, err
		}
		declaracionesFrontera = append(declaracionesFrontera, fronterasAuditoria...)
	}
	if preparacionBasesActiva {
		fronteras, err := montajePreparacionBases.Fronteras()
		if err != nil {
			return nil, nil, nil, err
		}
		declaracionesFrontera = append(declaracionesFrontera, fronteras...)
	}
	fronterasBaremo, err := fronterasGobiernoReglasBaremoHTTPV3(montajeBaremo.perfilRef)
	if err != nil {
		return nil, nil, nil, err
	}
	declaracionesFrontera = append(declaracionesFrontera, fronterasBaremo...)
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
		// Descargas con logotipo; la formalización guarda la huella de su PDF y sigue sin él.
		borradorRRHH = informejuridico.RenderizadorBorradorDesarrollo{PDF: pdfvec.Renderizador{Membrete: true}, Etiquetas: etiquetas, Plantillas: plantillas}
		borradorFormalizacion = informejuridico.RenderizadorBorradorDesarrollo{PDF: pdfvec.Renderizador{}, Etiquetas: etiquetas, Plantillas: plantillas}
		borradorRRHHDOCX = informejuridico.RenderizadorBorradorDOCXDesarrollo{DOCX: docxvec.Renderizador{Membrete: true}, Etiquetas: etiquetas, Plantillas: plantillas}
	}
	defer func() {
		if cerrarAlta {
			consultasRRHH.cerrar()
		}
	}()
	if firmaDocumento != nil {
		ctxFirma, cancelarFirma := context.WithTimeout(context.Background(), plazoarranque.Ampliar(15*time.Second))
		err := prepararCuentaNominalFirmasIntervencionDesarrollo(ctxFirma, alta.postgresql.gobierno, fiscalizacionReal.soporte, derivador)
		if err == nil {
			err = firmaDocumento.configurarLecturaIntervencion(ctxFirma, fiscalizacionReal.soporte,
				consultasRRHH.identidad, aprobacionProvisionPerfilesRRHHDesdeConfig(cfg))
		}
		cancelarFirma()
		if err != nil {
			return nil, nil, nil, err
		}
	}
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
		if cfg.IncorporacionV2File == "" && !incorporacion[0].legadoCompuesto {
			return nil, nil, nil, ErrComposicionDesarrolloIncompleta
		}
		if (incorporacion[0].nominales != nil && incorporacion[0].nominales.montajeB2 != nil) != b2Configurada {
			return nil, nil, nil, ErrComposicionDesarrolloIncompleta
		}
		if cfg.IncorporacionV2File != "" && incorporacion[0].legadoCompuesto != legadoConfigurado {
			return nil, nil, nil, ErrComposicionDesarrolloIncompleta
		}
		if incorporacion[0].legadoCompuesto {
			incorporacionV2, err = nuevasDependenciasIncorporacionV2Desarrollo(incorporacion[0], &alta, consultasRRHH, reloj)
			if err != nil {
				return nil, nil, nil, err
			}
		}
	}
	lectorVisualLegado, err := LeerLectorFlujoVisualRRHH(
		strings.NewReader(config.PresentacionFlujoRRHHDesarrollo()),
	)
	if err != nil {
		return nil, nil, nil, err
	}
	lectoresVisuales := []*LectorFlujoVisualRRHH{lectorVisualLegado}
	if cfg.CTCircuitoRRHHSourcePath != "" {
		lectorVisualCircuito, err := LeerLectorFlujoVisualRRHH(
			strings.NewReader(config.PresentacionCircuitoRRHH()),
		)
		if err != nil || lectorVisualCircuito.origen != alta.soporte.flujo.Flujo {
			return nil, nil, nil, ErrManifestFlujoVisualRRHHInvalido
		}
		lectoresVisuales = append(lectoresVisuales, lectorVisualCircuito)
	}
	presentacionFlujoRRHH, err := NuevoLectorFlujosVisualesRRHH(lectoresVisuales...)
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
			DescargaBorradorRRHH:            consultasRRHH.descargas,
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
	if consultaCircuitoRRHH != nil {
		if dependenciaEsNulaContratacionTemporalDesarrollo(alta.postgresql.registradorAuditoriaFrontera) {
			return nil, nil, nil, ports.ErrConsultaCircuitoRRHHNoDisponible
		}
		consultaCircuitoRRHH = auditorConsultaCircuitoRRHHDenegada{
			siguiente: consultaCircuitoRRHH, registrador: alta.postgresql.registradorAuditoriaFrontera,
			soporte: alta.soporte, reloj: reloj,
		}
		rutas = append(rutas, vechttp.RutaExacta{Ruta: httpinterno.RutaConsultaCircuitoRRHH, Manejador: consultaCircuitoRRHH})
	}
	if firmasR5V2 != nil {
		rutas = append(rutas, firmasR5V2.rutas...)
	}
	if len(incorporacion) == 1 && incorporacion[0].nominales != nil && incorporacion[0].nominales.montajeB2 != nil {
		rutasB2, err := incorporacion[0].nominales.montajeB2.rutas(alta.soporte, catalogoFronteras)
		if err != nil {
			return nil, nil, nil, err
		}
		rutas = append(rutas, rutasB2...)
	}
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
		incorporacion[0].continuidad.admisionSinCese = admisionCierreSinCeseDesarrollo(reglasEjemplo.contratacionTemporal)
		continuidad, err := incorporacion[0].continuidad.rutas(derivador, catalogoFronteras)
		if err != nil {
			return nil, nil, nil, err
		}
		rutas = append(rutas, continuidad...)
	}
	if comunicacionReal != nil && consultasRRHH.detalle != nil && borradorFormalizacion != nil {
		resolucion, err := nuevasDependenciasResolucionFormalizacionDesarrollo(&alta, reloj, consultasRRHH.detalle, borradorFormalizacion, consultasRRHH.preparacionResolucion, catalogoFronteras)
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
	rutasSeguimientoCese, err := nuevasRutasSeguimientoCeseDesarrollo(dependencias, &alta, finCesePersonalB2(incorporacion, alta.soporte, catalogoFronteras))
	if err != nil {
		return nil, nil, nil, err
	}
	rutas = append(rutas, rutasSeguimientoCese...)
	rutasCancelacion, err := nuevasRutasCancelacionCTDesarrollo(dependencias, &alta)
	if err != nil {
		return nil, nil, nil, err
	}
	rutas = append(rutas, rutasCancelacion...)
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
	if consultaReciboRespuestaReal != nil {
		rutas = append(rutas, vechttp.RutaExacta{Ruta: httpinterno.RutaConsultaReciboRespuesta, Manejador: consultaReciboRespuestaReal})
	}
	if consultaComunicacionesReal != nil {
		rutas = append(rutas, vechttp.RutaExacta{Ruta: httpinterno.RutaConsultaComunicacionesExpediente, Manejador: consultaComunicacionesReal})
	}
	if eventoPlazoReal != nil {
		rutas = append(rutas, vechttp.RutaExacta{Ruta: httpinterno.RutaEventoPlazoLlamamiento, Manejador: eventoPlazoReal})
	}
	rutasBorrador := []vechttp.RutaExacta(nil)
	coleccionesBorrador := []vechttp.RutaColeccion(nil)
	// Instancia única construida antes de las sesiones CT y Bolsa.
	seguridadBorrador := catalogoFronteras
	var envolverBorrador func(http.Handler) http.Handler
	var manejadorSituacion http.Handler
	cerrarBorrador := func() {}
	personalizacionB7 := &fuentePersonalizacionB7{}
	var autorizacionesPreparacionBases []descriptorAutorizacionComunDesarrollo
	if preparacionBasesActiva {
		autorizacionesPreparacionBases, err = montajePreparacionBases.autorizacionesPostgreSQL(
			fuenteAutorizacionPlantillas, alta.postgresql.registroAutorizacion, motivosEvaluadorPlantillas)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	if debeComponerBorradorLlamamientoDesarrollo(cfg) {
		if consultasRRHH.identidad == nil {
			return nil, nil, nil, errBorradorNoDisponibleEn()
		}
		var errBorrador error
		rutasBorrador, coleccionesBorrador, manejadorSituacion, seguridadBorrador, envolverBorrador, cerrarBorrador, errBorrador = nuevasDependenciasBorradorLlamamientoDesarrollo(
			context.Background(), cfg, dependencias, &alta, soporteBolsaCatalogo, catalogoFronteras, consultasRRHH.identidad, personalizacionB7,
			autorizacionesPreparacionBases,
			autoridadesCargaConvoca,
			alta.postgresql.proveedorMaterialConsultaReincorporacionTitular,
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
	cerrarPreparacionBases := func() {}
	if preparacionBasesActiva {
		var rutasPreparacionBases []vechttp.RutaExacta
		rutasPreparacionBases, cerrarPreparacionBases, err = montajePreparacionBases.rutasDesdeRaiz(context.Background(), &alta, consultasRRHH.identidad, catalogoFronteras,
			fuenteAutorizacionPlantillas, motivosEvaluadorPlantillas)
		if err != nil {
			return nil, nil, nil, err
		}
		rutas = append(rutas, rutasPreparacionBases...)
	}
	defer func() {
		if cerrarAlta {
			cerrarPreparacionBases()
		}
	}()
	sondaBaremo, cancelarBaremo := context.WithTimeout(context.Background(), plazoarranque.Ampliar(60*time.Second))
	rutasBaremo, cerrarBaremo, errBaremo := montajeBaremo.rutas(sondaBaremo, cfg, &alta,
		consultasRRHH.identidad, seguridadBorrador, derivador, reloj)
	cancelarBaremo()
	if errBaremo != nil {
		log.Print("bolsa: gobierno de baremo no disponible; etapa=composicion; causa=dependencia_no_disponible")
		rutasBaremo, cerrarBaremo = montajeBaremo.indisponibles(), func() {}
	}
	defer func() {
		if cerrarAlta {
			cerrarBaremo()
		}
	}()
	rutas = append(rutas, rutasBaremo...)
	if plantillasActivas {
		if consultasRRHH.identidad == nil || alta.postgresql.proveedorMaterialPlantillasCatalogo == nil {
			return nil, nil, nil, plantillasapp.ErrNoDisponible
		}
		sondaPlantillas, cancelarPlantillas := context.WithTimeout(context.Background(), plazoarranque.Ampliar(60*time.Second))
		defer cancelarPlantillas()
		rutasPlantillas, err := nuevasRutasPlantillasCTDesarrollo(
			sondaPlantillas, cfg, &alta, soportePlantillas, consultasRRHH.identidad,
			seguridadBorrador, fuenteAutorizacionPlantillas, motivosEvaluadorPlantillas,
			reloj, alta.postgresql.proveedorMaterialPlantillasCatalogo)
		if err != nil {
			return nil, nil, nil, err
		}
		rutas = append(rutas, rutasPlantillas...)
	}
	if ajustesCTActivos {
		sondaAjustes, cancelarAjustes := sondaAjustesReglasCT()
		rutaAjustes, err := nuevaRutaAjustesReglasCT(sondaAjustes, cfg, &alta,
			seguridadBorrador, fuenteAutorizacionPlantillas,
			motivosEvaluadorPlantillas, alta.postgresql.proveedorMaterialAjustesReglasCT,
			reglasEjemplo.contratacionTemporal, reloj)
		cancelarAjustes()
		if err != nil {
			return nil, nil, nil, err
		}
		rutas = append(rutas, rutaAjustes)
	}
	if documentalActiva {
		if consultasRRHH.identidad == nil || alta.postgresql.proveedorMaterialPlantillasDocumental == nil {
			return nil, nil, nil, plantillasapp.ErrNoDisponible
		}
		sondaDocumental, cancelarDocumental := context.WithTimeout(context.Background(), plazoarranque.Ampliar(60*time.Second))
		defer cancelarDocumental()
		rutasDocumentales, err := nuevasRutasPlantillasDocumentalCTDesarrollo(
			sondaDocumental, cfg, &alta, soporteDocumental, consultasRRHH.identidad,
			seguridadBorrador, fuenteAutorizacionPlantillas, motivosEvaluadorPlantillas,
			reloj, consultasRRHH, origen.etiquetasReferenciasCatalogosAlta())
		if err != nil {
			return nil, nil, nil, err
		}
		rutas = append(rutas, rutasDocumentales...)
	}
	cerrarAuditoria := func() {}
	cerrarAuditoriaPendiente := true
	defer func() {
		if cerrarAuditoriaPendiente {
			cerrarAuditoria()
		}
	}()
	if auditoriaActiva {
		if consultasRRHH.identidad == nil {
			return nil, nil, nil, errAutoridadesAuditoriaConsultaDesarrollo
		}
		rutasAuditoria, detener, err := nuevasRutasAuditoriaConsultaDesarrollo(
			context.Background(), cfg, &alta, soportesAuditoria, consultasRRHH.identidad, seguridadBorrador, reloj)
		if err != nil {
			return nil, nil, nil, err
		}
		cerrarAuditoria = detener
		rutas = append(rutas, rutasAuditoria...)
	}
	if resolvedorDesarrollo.candidatoBolsa != nil {
		if !debeComponerMiBolsaDesarrollo(cfg) || consultasRRHH.identidad == nil {
			return nil, nil, nil, errMiBolsaNoDisponible
		}
		camposMiBolsa, err := camposPortalMiBolsaDesarrollo(reglasBolsa)
		if err != nil {
			return nil, nil, nil, err
		}
		var portal puertosbolsa.ReglasPortalCandidato
		if debeComponerPortalCandidatoDesarrollo(cfg) {
			if reglasBolsa == nil {
				return nil, nil, nil, errMiBolsaNoDisponible
			}
			portal = reglasPortalCandidatoDesarrollo{resolutor: reglasBolsa}
		}
		if alta.postgresql.gobierno == nil || alta.postgresql.registroAutorizacion == nil || alta.postgresql.ejecucion == nil || alta.postgresql.bolsa == nil {
			return nil, nil, nil, errMiBolsaNoDisponible
		}
		ctxIntentosBolsa, cancelarIntentosBolsa := context.WithTimeout(context.Background(), plazoarranque.Ampliar(30*time.Second))
		reservadosIntentosBolsa := []string{
			alta.postgresql.gobierno.Config().ConnConfig.User,
			alta.postgresql.registroAutorizacion.Config().ConnConfig.User,
			alta.postgresql.ejecucion.Config().ConnConfig.User,
		}
		alta.postgresql.auditoriaLecturasBolsa, alta.postgresql.procesoAuditoriaLecturasBolsa, alta.postgresql.cerrarAuditoriaLecturasBolsa, err =
			AbrirRegistradorIntentosAuditoriaExternaDesarrollo(ctxIntentosBolsa, cfg, alta.postgresql.bolsa, reservadosIntentosBolsa)
		cancelarIntentosBolsa()
		if err != nil {
			return nil, nil, nil, err
		}
		rutasMiBolsa, err := nuevaRutaMiBolsaDesarrollo(
			context.Background(), resolvedorDesarrollo.candidatoBolsa, sello, &alta,
			consultasRRHH.identidad, catalogoFronteras, derivador, reloj, camposMiBolsa, portal,
			aprobacionProvisionMiBolsaDesdeConfig(cfg), personalizacionB7,
		)
		if err != nil {
			return nil, nil, nil, err
		}
		rutas = append(rutas, rutasMiBolsa...)
	}
	registradorFrontera := puertosvec.RegistradorAuditoriaFronteraRutaExacta(alta.postgresql.registradorAuditoriaFrontera)
	cerrarFronteraAuditoria := func() {}
	cerrarFronteraAuditoriaPendiente := true
	defer func() {
		if cerrarFronteraAuditoriaPendiente {
			cerrarFronteraAuditoria()
		}
	}()
	if auditoriaActiva {
		registrador, detener, err := nuevoRegistradorFronteraAuditoriaConsultaDesarrollo(context.Background(), cfg)
		if err != nil {
			return nil, nil, nil, err
		}
		cerrarFronteraAuditoria = detener
		registradorFrontera = registradorFronterasPorSuperficieDesarrollo{
			ct: alta.postgresql.registradorAuditoriaFrontera, auditoria: registrador}
		for i := range rutas {
			if rutas[i].Ruta == auditoria.RutaOpciones || rutas[i].Ruta == auditoria.RutaConsulta {
				rutas[i].Manejador = manejadorAuditoriaDenegacionesLocales{
					siguiente: rutas[i].Manejador, registrador: registrador, soporte: alta.soporte}
			}
		}
	}
	autoridad := &autoridadConsultasContratacionTemporalDesarrollo{
		sello:                                            sello,
		resolvedor:                                       resolvedorDesarrollo,
		noCompuesta:                                      noCompuesta,
		llamamientoCompuesto:                             comunicacionReal != nil,
		consultasRRHHCompuestas:                          consultasRRHH.cuadro != nil && consultasRRHH.detalle != nil,
		subsanacionCompuesta:                             subsanacionReal.servicio != nil,
		fronterasSeguridadComun:                          seguridadBorrador,
		envolverBorradorLlamamiento:                      envolverBorrador,
		manejadorSituacionParticipacion:                  manejadorSituacion,
		plazosOfertasBolsa:                               dependencias.plazosOfertasBolsa,
		personalizacionB7:                                personalizacionB7,
		coleccionesAdicionales:                           coleccionesBorrador,
		registradorAuditoriaFronteraRutasExactas:         registradorFrontera,
		materialDietas:                                   alta.postgresql.materialDietas,
		materialCronos:                                   alta.postgresql.materialCronos,
		materialDocumentos:                               alta.postgresql.materialDocumentos,
		materialInscripcionBolsa:                         alta.postgresql.materialInscripcionBolsa,
		materialPersonalFichaPropia:                      alta.postgresql.materialPersonalFichaPropia,
		materialPersonalExportacionServicios:             alta.postgresql.materialPersonalExportacionServicios,
		materialPersonalHistoriaServicios:                alta.postgresql.materialPersonalHistoriaServicios,
		materialPersonalHistoriaRelaciones:               alta.postgresql.materialPersonalHistoriaRelaciones,
		gobiernoUsuariosPreferencias:                     alta.postgresql.gobierno,
		materialUsuariosPreferenciasConsultaInterna:      alta.postgresql.materialUsuariosPreferenciasConsultaInterna,
		materialUsuariosPreferenciasActualizacionInterna: alta.postgresql.materialUsuariosPreferenciasActualizacionInterna,
		materialUsuariosPreferenciasConsultaExterna:      alta.postgresql.materialUsuariosPreferenciasConsultaExterna,
		materialUsuariosPreferenciasActualizacionExterna: alta.postgresql.materialUsuariosPreferenciasActualizacionExterna,
		materialUsuariosCorreos:                          alta.postgresql.materialUsuariosCorreos,
		materialUsuariosImagen:                           alta.postgresql.materialUsuariosImagen,
		materialAspirantes:                               alta.postgresql.materialAspirantes,
		presentadorCobertura:                             coberturaReal.presentador,
		situacionBolsaCobertura:                          coberturaReal.situacionBolsa,
		firmaDocumento:                                   firmaDocumento,
	}
	if autoridad.registradorAuditoriaFronteraRutasExactas == nil {
		return nil, nil, nil, falloPostgreSQLCTDesarrollo(nil)
	}
	dependencias.cerrar = func() {
		cerrarPreparacionBases()
		cerrarBaremo()
		cerrarAutoridadesPlantillas()
		cerrarFronteraAuditoria()
		cerrarAuditoria()
		cerrarBorrador()
		cerrarIncorporacion()
		consultasRRHH.cerrar()
		coberturaReal.cerrar()
		alta.cerrar()
	}
	cerrarCobertura = false
	cerrarAlta = false
	cerrarBorradorPendiente = false
	cerrarAuditoriaPendiente = false
	cerrarFronteraAuditoriaPendiente = false
	cerrarAutoridadesPlantillasPendiente = false
	return rutas, autoridad, dependencias.Cerrar, nil
}

func debeComponerBorradorLlamamientoDesarrollo(cfg config.Config) bool {
	return cfg.BolsaBorradoresEnabled
}

type identidadSesionAuditoriaConsultaDesarrollo struct {
	proveedor *proveedorSesionConsultaRRHHDesarrollo
	fuente    auditoria.FuenteConsulta
	ruta      string
	metodo    string
}

func (i identidadSesionAuditoriaConsultaDesarrollo) ResolverIdentidadConsulta(ctx context.Context, r *http.Request, fuente auditoria.FuenteConsulta) (auditoria.IdentidadResuelta, error) {
	if ctx == nil || ctx.Err() != nil || r == nil || r.URL == nil || i.proveedor == nil ||
		fuente != i.fuente || r.URL.Path != i.ruta || r.Method != i.metodo {
		return auditoria.IdentidadResuelta{}, auditoria.ErrDenegada
	}
	resuelta, err := i.proveedor.ResolverContexto(ctx)
	if err != nil || resuelta.Resultado.Validar() != nil || resuelta.Vinculo.ValidarPara(resuelta.Resultado) != nil {
		return auditoria.IdentidadResuelta{}, auditoria.ErrDenegada
	}
	if holder, ok := ctx.Value(claveActorAuditoriaLocal{}).(*actorAuditoriaLocal); ok {
		holder.fijar(resuelta.Resultado.Contexto.Principal.ID)
	}
	correlacion, ok := ctx.Value(claveCorrelacionAuditoriaLocal{}).(vecdomain.ReferenciaCorrelacionAutorizacionV2)
	if !ok || correlacion.Validar() != nil {
		correlacion, err = vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridadvec.GeneradorReferenciasCriptograficas{})
		if err != nil {
			return auditoria.IdentidadResuelta{}, auditoria.ErrNoDisponible
		}
	}
	return auditoria.IdentidadResuelta{Vinculo: resuelta.Vinculo, Resultado: resuelta.Resultado, Correlacion: correlacion}, nil
}

func nuevasRutasAuditoriaConsultaDesarrollo(
	ctx context.Context, cfg config.Config, alta *dependenciasAltaContratacionTemporalDesarrollo,
	soportes soportesAuditoriaConsultaDesarrollo, identidadBase *proveedorSesionConsultaRRHHDesarrollo,
	fronteras catalogoFronterasComunDesarrollo, reloj relojContratacionTemporalDesarrollo,
) ([]vechttp.RutaExacta, func(), error) {
	fallo := func(cerrar func()) ([]vechttp.RutaExacta, func(), error) {
		if cerrar != nil {
			cerrar()
		}
		return nil, nil, errAutoridadesAuditoriaConsultaDesarrollo
	}
	if ctx == nil || ctx.Err() != nil || alta == nil || alta.postgresql.gobierno == nil ||
		alta.postgresql.registroAutorizacion == nil || alta.postgresql.bolsa == nil ||
		alta.postgresql.proveedorMaterialAuditoriaCT == nil || alta.postgresql.proveedorMaterialAuditoriaBolsa == nil ||
		soportes.CT == nil || soportes.Bolsa == nil || identidadBase == nil || fronteras.identidad == nil {
		return fallo(nil)
	}
	ctRef, bolsaRef, err := cfg.ExpedientesAuditoriaConsultaDesarrollo()
	if err != nil {
		return fallo(nil)
	}
	dsnCT, _, err := cfg.ContratacionTemporalPostgreSQL.DSNConsultasRRHHSeparados()
	if err != nil {
		return fallo(nil)
	}
	dsnMotivos, err := cfg.DSNMotivosAuditoriaDesarrollo()
	if err != nil {
		return fallo(nil)
	}
	dsnFuente, err := cfg.DSNFuenteAutorizacionAuditoriaDesarrollo()
	if err != nil {
		return fallo(nil)
	}
	sonda, cancelar := context.WithTimeout(ctx, plazoarranque.Ampliar(60*time.Second))
	defer cancelar()
	poolCT, err := abrirPoolConsultaAuditoriaCTDesarrollo(sonda, dsnCT)
	if err != nil {
		return fallo(nil)
	}
	poolFuente, err := abrirPoolAutoridadAuditoriaDesarrollo(sonda, dsnFuente, config.RolFuenteAutorizacionAuditoriaDesarrollo, "vec-auditoria-fuente")
	if err != nil {
		poolCT.Close()
		return fallo(nil)
	}
	poolMotivos, err := abrirPoolAutoridadAuditoriaDesarrollo(sonda, dsnMotivos, config.RolMotivosAuditoriaDesarrollo, "vec-auditoria-motivos")
	if err != nil {
		poolFuente.Close()
		poolCT.Close()
		return fallo(nil)
	}
	cerrar := func() { poolMotivos.Close(); poolFuente.Close(); poolCT.Close() }
	if preflightAuditoriaConsultaDesarrollo(sonda, poolFuente, poolMotivos, alta.postgresql.bolsa) != nil {
		return fallo(cerrar)
	}
	for _, soporte := range []struct {
		base      *soporteAltaContratacionTemporalDesarrollo
		operacion string
	}{
		{soportes.CT, soportes.OperacionContextoCT}, {soportes.Bolsa, soportes.OperacionContextoBolsa},
	} {
		if publicarResultadoContextoPostgreSQLDesarrollo(sonda, alta.postgresql.gobierno, soporte.base.contexto.Resultado, soporte.operacion) != nil {
			return fallo(cerrar)
		}
		esperado, err := contextoEsperadoRegistradoDesarrollo(sonda, identidadBase.resolutor, soporte.base)
		if err != nil {
			return fallo(cerrar)
		}
		soporte.base.contextoEsperadoRegistrado = esperado
	}
	validador, err := postgresvec.NuevoValidadorReferenciaMotivoPostgreSQLV2(poolMotivos, catalogoMotivosAuditoriaConsultaDesarrollo)
	if err != nil {
		return fallo(cerrar)
	}
	opciones, err := configurarOpcionesAuditoriaConsultaDesarrollo(sonda, cfg, alta.postgresql.gobierno, validador, reloj)
	if err != nil {
		return fallo(cerrar)
	}
	for _, s := range []struct {
		base                       *soporteAltaContratacionTemporalDesarrollo
		fuente, perfil, expediente string
	}{
		{soportes.CT, "ct", soportes.PerfilCT, ctRef},
		{soportes.Bolsa, "bolsa", soportes.PerfilBolsa, bolsaRef},
	} {
		semilla, err := instantaneaAuditoriaConsultaNominalDesarrollo(s.base.principalID, s.perfil, s.fuente, s.expediente, opciones.FinalidadRef, reloj.Ahora())
		if err != nil || publicarInstantaneaAuditoriaConsultaDesarrollo(sonda, alta.postgresql.gobierno, poolFuente, s.base, semilla) != nil {
			return fallo(cerrar)
		}
	}
	identidadCT, err := nuevoProveedorSesionConsultaRRHHConCatalogoDesarrollo(soportes.CT,
		identidadBase.registro, identidadBase.revalidador, reloj, identidadBase.resolutor, fronteras)
	if err != nil {
		return fallo(cerrar)
	}
	identidadBolsa, err := nuevoProveedorSesionConsultaRRHHConCatalogoDesarrollo(soportes.Bolsa,
		identidadBase.registro, identidadBase.revalidador, reloj, identidadBase.resolutor, fronteras)
	if err != nil {
		return fallo(cerrar)
	}
	emisorCT, emisorBolsa, err := nuevosEmisoresAuditoriaConsultaDesarrollo(dependenciasAutoridadesAuditoriaConsultaDesarrollo{
		FuenteCT: poolFuente, FuenteBolsa: poolFuente,
		RegistroCT: alta.postgresql.registroAutorizacion, RegistroBolsa: alta.postgresql.registroAutorizacion,
		MotivosCT: poolMotivos, MotivosBolsa: poolMotivos,
		MaterialCT: alta.postgresql.proveedorMaterialAuditoriaCT, MaterialBolsa: alta.postgresql.proveedorMaterialAuditoriaBolsa,
		PerfilCT: soportes.PerfilCT, PerfilBolsa: soportes.PerfilBolsa,
		ExpedienteCT: ctRef, ExpedienteBolsa: bolsaRef, Opciones: opciones, Reloj: reloj,
	})
	if err != nil {
		return fallo(cerrar)
	}
	proveedorOpciones, _, err := nuevoProveedorOpcionesAuditoriaConsultaDesarrollo(sonda, cfg, reloj, validador)
	if err != nil {
		return fallo(cerrar)
	}
	rutas, err := nuevasRutasAuditoriaConsultaConIdentidadesRRHH(dependenciasIdentidadAuditoriaConsultaRRHH{
		PoolCT: poolCT, PoolBolsa: alta.postgresql.bolsa,
		EmisorCT: emisorCT, EmisorBolsa: emisorBolsa,
		IdentidadOpciones: identidadSesionAuditoriaConsultaDesarrollo{identidadCT, auditoria.FuenteConsultaGeneral, auditoria.RutaOpciones, http.MethodGet},
		IdentidadCT:       identidadSesionAuditoriaConsultaDesarrollo{identidadCT, auditoria.FuenteConsultaCT, auditoria.RutaConsulta, http.MethodPost},
		IdentidadBolsa:    identidadSesionAuditoriaConsultaDesarrollo{identidadBolsa, auditoria.FuenteConsultaBolsa, auditoria.RutaConsulta, http.MethodPost},
		Opciones:          proveedorOpciones,
		Intentos: auditoria.ConfiguracionIntentos{
			Registrador: alta.auditoriaLecturasCT, Proceso: alta.procesoAuditoriaLecturasCT,
			Canal:     string(vecdomain.SuperficieAutenticacionInternaCorporativaV1),
			Finalidad: opciones.FinalidadRef, Motivo: opciones.Motivo,
		},
	})
	if err != nil {
		return fallo(cerrar)
	}
	return rutas, cerrar, nil
}

func configurarOpcionesAuditoriaConsultaDesarrollo(ctx context.Context, cfg config.Config, gobierno *pgxpool.Pool,
	validador *postgresvec.ValidadorReferenciaMotivoPostgreSQLV2, reloj relojContratacionTemporalDesarrollo) (auditoria.Opciones, error) {
	ruta, err := cfg.RutaCatalogoAuditoriaConsultaDesarrollo()
	if err != nil || gobierno == nil || validador == nil {
		return auditoria.Opciones{}, errAutoridadesAuditoriaConsultaDesarrollo
	}
	consulta, err := fichero.NuevaConsultaCatalogos(ruta)
	if err != nil {
		return auditoria.Opciones{}, errAutoridadesAuditoriaConsultaDesarrollo
	}
	lector, err := reglas.NuevoResolutor(reglas.Configuracion{Consulta: consulta, Metadatos: consulta,
		CatalogoID: catalogoMotivosAuditoriaConsultaDesarrollo, ModuloID: auditoria.ModuloAutorizacion, Reloj: reloj})
	if err != nil {
		return auditoria.Opciones{}, errAutoridadesAuditoriaConsultaDesarrollo
	}
	p, err := auditoria.NuevoProveedorOpcionesCatalogo(lector, validador)
	if err != nil {
		return auditoria.Opciones{}, errAutoridadesAuditoriaConsultaDesarrollo
	}
	declaradas, err := p.Configuradas(ctx)
	if err != nil || !declaradas.EsEjemplo ||
		publicarCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(ctx, gobierno,
			[]vecdomain.ReferenciaEntradaCatalogo{declaradas.Motivo}, reloj.Ahora()) != nil {
		return auditoria.Opciones{}, errAutoridadesAuditoriaConsultaDesarrollo
	}
	_, actuales, err := nuevoProveedorOpcionesAuditoriaConsultaDesarrollo(ctx, cfg, reloj, validador)
	if err != nil || actuales.Motivo != declaradas.Motivo || actuales.FinalidadRef != declaradas.FinalidadRef {
		return auditoria.Opciones{}, errAutoridadesAuditoriaConsultaDesarrollo
	}
	return actuales, nil
}

func publicarInstantaneaAuditoriaConsultaDesarrollo(ctx context.Context, gobierno, fuente *pgxpool.Pool,
	soporte *soporteAltaContratacionTemporalDesarrollo, semilla vecdomain.InstantaneaAutorizacion) error {
	if gobierno == nil || fuente == nil || soporte == nil || semilla.Validar() != nil {
		return errAutoridadesAuditoriaConsultaDesarrollo
	}
	actualExiste, err := existeAsignacionAuditoriaConsultaDesarrollo(ctx, gobierno, semilla.AsignacionPerfil.PerfilActivoRef)
	if err != nil {
		return errAutoridadesAuditoriaConsultaDesarrollo
	}
	if actualExiste {
		almacen, err := postgresvec.NuevoAlmacenAutorizacion(fuente)
		if err != nil {
			return errAutoridadesAuditoriaConsultaDesarrollo
		}
		actual, err := almacen.ObtenerInstantaneaAutorizacion(ctx, semilla.AsignacionPerfil.PrincipalID, semilla.AsignacionPerfil.PerfilActivoRef)
		if err != nil || !instantaneaAuditoriaConsultaVigenteExacta(actual, semilla, time.Now().UTC().Truncate(time.Microsecond)) {
			return errAutoridadesAuditoriaConsultaDesarrollo
		}
		return nil
	}
	a := autoridadPostgreSQLDesarrollo{pool: gobierno, vinculo: soporte.contexto.Vinculo,
		prefijoBloqueo: "vec:auditoria:desarrollo:autorizacion:",
		actoControlRol: "acto:auditoria:desarrollo:control-rol:v1",
		actoAsignacion: "acto:auditoria:desarrollo:asignacion:v1",
		actoSesion:     "acto:auditoria:desarrollo:sesion:v1", soloInicial: true}
	preparada, err := a.prepararInstantanea(ctx, semilla, true)
	if err != nil || !instantaneaInicialAuditoriaConsultaExacta(preparada, semilla) {
		return errAutoridadesAuditoriaConsultaDesarrollo
	}
	if err := a.publicarInstantanea(ctx, preparada); err != nil {
		return errAutoridadesAuditoriaConsultaDesarrollo
	}
	return nil
}

func instantaneaInicialAuditoriaConsultaExacta(preparada, semilla vecdomain.InstantaneaAutorizacion) bool {
	return preparada.Validar() == nil && semilla.Validar() == nil &&
		preparada.AsignacionPerfil.Version == 1 &&
		preparada.AsignacionPerfil.PrincipalID == semilla.AsignacionPerfil.PrincipalID &&
		preparada.AsignacionPerfil.PerfilActivoRef == semilla.AsignacionPerfil.PerfilActivoRef &&
		preparada.VersionRol.RolID == semilla.VersionRol.RolID &&
		reflect.DeepEqual(preparada.VersionRol.Concesiones, semilla.VersionRol.Concesiones) &&
		reflect.DeepEqual(preparada.AsignacionPerfil.Ambitos, semilla.AsignacionPerfil.Ambitos)
}

func instantaneaAuditoriaConsultaVigenteExacta(actual, semilla vecdomain.InstantaneaAutorizacion, ahora time.Time) bool {
	return actual.Validar() == nil && semilla.Validar() == nil &&
		actual.AsignacionPerfil.Estado == vecdomain.EstadoAsignacionPerfilActiva &&
		actual.AsignacionPerfil.VigenteEn(ahora) &&
		actual.ControlVigenciaVersionRol.Estado == vecdomain.EstadoControlVigenciaVersionRolHabilitada &&
		actual.AsignacionPerfil.PrincipalID == semilla.AsignacionPerfil.PrincipalID &&
		actual.AsignacionPerfil.PerfilActivoRef == semilla.AsignacionPerfil.PerfilActivoRef &&
		actual.VersionRol.RolID == semilla.VersionRol.RolID &&
		reflect.DeepEqual(actual.VersionRol.Concesiones, semilla.VersionRol.Concesiones) &&
		reflect.DeepEqual(actual.AsignacionPerfil.Ambitos, semilla.AsignacionPerfil.Ambitos)
}

func existeAsignacionAuditoriaConsultaDesarrollo(ctx context.Context, gobierno *pgxpool.Pool, perfil string) (bool, error) {
	if ctx == nil || ctx.Err() != nil || gobierno == nil || !perfilActivoSeguridadComunValido(perfil) {
		return false, errAutoridadesAuditoriaConsultaDesarrollo
	}
	tx, err := gobierno.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return false, errAutoridadesAuditoriaConsultaDesarrollo
	}
	defer tx.Rollback(context.Background())
	if _, err := tx.Exec(ctx, `SET LOCAL ROLE vec_autorizacion_propietario`); err != nil {
		return false, errAutoridadesAuditoriaConsultaDesarrollo
	}
	_, encontrada, err := leerAsignacionActualPostgreSQLDesarrollo(ctx, tx, perfil)
	if err != nil || tx.Commit(ctx) != nil {
		return false, errAutoridadesAuditoriaConsultaDesarrollo
	}
	return encontrada, nil
}

func preflightAuditoriaConsultaDesarrollo(ctx context.Context, fuente, motivos, bolsa *pgxpool.Pool) error {
	if ctx == nil || fuente == nil || motivos == nil || bolsa == nil {
		return errAutoridadesAuditoriaConsultaDesarrollo
	}
	const sonda = `SELECT
	 to_regprocedure($1) IS NOT NULL AND to_regprocedure($2) IS NOT NULL
	 AND coalesce(has_function_privilege(session_user,to_regprocedure($1)::oid,'EXECUTE'),false)
	 AND NOT coalesce(has_function_privilege(session_user,to_regprocedure($2)::oid,'EXECUTE'),false)`
	const funcionFuente = "vec_autorizacion.obtener_instantanea(text,text)"
	const funcionMotivos = "vec_autorizacion.resolver_motivo_autorizacion_v2_historico(text,integer,text,text,timestamptz)"
	for _, caso := range []struct {
		pool          *pgxpool.Pool
		propia, ajena string
	}{
		{fuente, funcionFuente, funcionMotivos},
		{motivos, funcionMotivos, funcionFuente},
		{bolsa, funcionConsultaAuditoriaBolsaDesarrollo, funcionConsultaAuditoriaCTDesarrollo},
	} {
		var valida bool
		if err := caso.pool.QueryRow(ctx, sonda, caso.propia, caso.ajena).Scan(&valida); err != nil || !valida {
			return errAutoridadesAuditoriaConsultaDesarrollo
		}
	}
	return nil
}
