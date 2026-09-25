package bootstrap

import (
	"context"
	"errors"
	"sync"
	"time"
	"vec-diputacion-granada/config"
	contratacioncomposicion "vec-diputacion-granada/internal/app/composicion/interna/contrataciontemporal"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgrescontratacion "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	seguridadcontratacion "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/seguridad"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	aplicacionvec "vec-diputacion-granada/internal/vec/application"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	organizacionAltaContratacionTemporalDesarrollo  = "organizacion:desarrollo:dipgra"
	centroAltaContratacionTemporalDesarrollo        = "centro:desarrollo:001"
	categoriaAltaContratacionTemporalDesarrollo     = "categoria:desarrollo:c2"
	unidadCoberturaContratacionTemporalDesarrollo   = "unidad:desarrollo:rrhh"
	motivoAltaContratacionTemporalDesarrollo        = domain.ClaveCatalogo("sustitucion")
	accionPropuestaCoberturaDesarrollo              = "contratacion_temporal.cobertura.propuesta.consultar"
	tipoRecursoDecisionCoberturaDesarrollo          = "decision_cobertura_gobernada"
	finalidadPropuestaCoberturaDesarrollo           = "presentar_propuesta_cobertura"
	finalidadDecisionCoberturaDesarrollo            = "tramitar_cobertura_temporal"
	finalidadAnalisisContratacionTemporalDesarrollo = "analisis.tramitar"
)

var errAltaContratacionTemporalDesarrolloNoDisponible = errors.New(
	"contratacion temporal: alta efimera de desarrollo no disponible",
)

type relojContratacionTemporalDesarrollo struct{}

func (relojContratacionTemporalDesarrollo) Ahora() time.Time {
	return time.Now().UTC().Truncate(time.Microsecond)
}

func ventanaAutoridadSinteticaContratacionTemporalDesarrollo(
	ahora time.Time,
) (time.Time, time.Time, bool) {
	desde := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	hasta := time.Date(2036, 1, 1, 0, 0, 0, 0, time.UTC)
	ahora = ahora.UTC().Truncate(time.Microsecond)
	return desde, hasta, !ahora.Before(desde) && ahora.Before(hasta)
}

type autoridadAsignacionesContratacionTemporalDesarrollo interface {
	PrepararInstantanea(
		context.Context,
		dominiovec.InstantaneaAutorizacion,
	) (dominiovec.InstantaneaAutorizacion, error)
	PublicarInstantanea(
		context.Context,
		dominiovec.InstantaneaAutorizacion,
	) error
}

type registroDecisionesAnalisisContratacionTemporalDesarrollo interface {
	puertosvec.RegistroConcesionesCandidatasAutorizacionLigadaV3
	puertosvec.RegistroDenegacionesAutorizacionLigadaV3
}

// soporteAltaContratacionTemporalDesarrollo simula las fuentes corporativas
// solo para ejercitar los casos de uso reales. Todo su estado es efimero,
// no_autoritativo y queda aislado por la composicion de doble llave.
type soporteAltaContratacionTemporalDesarrollo struct {
	origen                            *origenConsultasContratacionTemporalDesarrollo
	peticionesCentro                  bool
	candidatoBolsa                    bool
	mu                                sync.Mutex
	sello                             *selloConsultasContratacionTemporalDesarrollo
	principalID                       string
	certificadoSHA256                 string
	lectorConsultasRRHH               bool
	tecnicoConsultaRRHH               bool
	organizacionConsultaRRHH          string
	claseAmbitoConsultaRRHH           ports.ClaseAmbitoConsultaRRHH
	ambitoConsultaRRHH                string
	contexto                          ports.ContextoAutorizacionAltaV3
	contextoEsperadoRegistrado        dominiovec.ResultadoContextoActorRegistradoV2
	sesionOperativa                   proveedorSesionOperativaCTDesarrollo
	flujo                             ports.ConfiguracionAltaFlujo
	motivo                            dominiovec.ReferenciaEntradaCatalogo
	instantanea                       dominiovec.InstantaneaAutorizacion
	instantaneaAnalisis               dominiovec.InstantaneaAutorizacion
	motivoRegistroAnalisis            dominiovec.ReferenciaEntradaCatalogo
	motivoRectificacionAnalisis       dominiovec.ReferenciaEntradaCatalogo
	instantaneaCobertura              dominiovec.InstantaneaAutorizacion
	instantaneaAsignacion             dominiovec.InstantaneaAutorizacion
	instantaneaInformeJuridico        dominiovec.InstantaneaAutorizacion
	instantaneaLlamamiento            dominiovec.InstantaneaAutorizacion
	instantaneaReanudacionLlamamiento dominiovec.InstantaneaAutorizacion
	instantaneaComunicacion           dominiovec.InstantaneaAutorizacion
	instantaneaCorreo                 dominiovec.InstantaneaAutorizacion
	instantaneaRespuestaRecibida      dominiovec.InstantaneaAutorizacion
	instantaneaConsultaJustificante   dominiovec.InstantaneaAutorizacion
	instantaneaResolucionManual       dominiovec.InstantaneaAutorizacion
	// reglasPlazo se fija al componer, antes de servir, y no cambia después.
	reglasPlazo                        ports.ReglasPlazoRespuestaLlamamiento
	instantaneaAceptacionBolsa         dominiovec.InstantaneaAutorizacion
	instantaneaRenunciaBolsa           dominiovec.InstantaneaAutorizacion
	instantaneaContinuacionCT          dominiovec.InstantaneaAutorizacion
	instantaneaSiguienteBolsa          dominiovec.InstantaneaAutorizacion
	instantaneaPropuestaFormalizacion  dominiovec.InstantaneaAutorizacion
	instantaneaResolucionFormalizacion dominiovec.InstantaneaAutorizacion
	instantaneaOrganizacion            dominiovec.InstantaneaAutorizacion
	instantaneaEntregaPeticion         dominiovec.InstantaneaAutorizacion
	instantaneaCuadroRRHH              dominiovec.InstantaneaAutorizacion
	instantaneaDetalleRRHH             dominiovec.InstantaneaAutorizacion
	instantaneaSubsanacion             dominiovec.InstantaneaAutorizacion
	instantaneaFirmaDocumento          dominiovec.InstantaneaAutorizacion
	motivoCuadroRRHH                   dominiovec.ReferenciaEntradaCatalogo
	motivoDetalleRRHH                  dominiovec.ReferenciaEntradaCatalogo
	motivoLlamamiento                  dominiovec.ReferenciaEntradaCatalogo
	motivoComunicacion                 dominiovec.ReferenciaEntradaCatalogo
	motivoDespachoCorreo               dominiovec.ReferenciaEntradaCatalogo
	motivoResultadoCorreo              dominiovec.ReferenciaEntradaCatalogo
	motivoRespuestaRecibida            dominiovec.ReferenciaEntradaCatalogo
	motivoConsultaJustificante         dominiovec.ReferenciaEntradaCatalogo
	motivoPropuestaCobertura           dominiovec.ReferenciaEntradaCatalogo
	motivoDecisionCobertura            dominiovec.ReferenciaEntradaCatalogo
	motivoRectificacionCobertura       dominiovec.ReferenciaEntradaCatalogo
	motivoResultadoCobertura           dominiovec.ReferenciaEntradaCatalogo
	motivoAsignacion                   dominiovec.ReferenciaEntradaCatalogo
	motivoInformeJuridico              dominiovec.ReferenciaEntradaCatalogo
	motivoSubsanacion                  dominiovec.ReferenciaEntradaCatalogo
	motivoFirmaDocumento               dominiovec.ReferenciaEntradaCatalogo
	ambitos                            ports.SelladorAmbitoIdempotencia
	reloj                              relojContratacionTemporalDesarrollo
	concesiones                        map[string]struct{}
	autoridadAsignaciones              autoridadAsignacionesContratacionTemporalDesarrollo
	registroDecisionesAnalisis         registroDecisionesAnalisisContratacionTemporalDesarrollo
	instantaneasPorSolicitud           map[string]dominiovec.InstantaneaAutorizacion
}

var _ httpinterno.AutoridadContextoCanalAnalisisRRHH = (*soporteAltaContratacionTemporalDesarrollo)(nil)
var _ httpinterno.AutoridadContextoCanalAsignacion = (*soporteAltaContratacionTemporalDesarrollo)(nil)
var _ httpinterno.AutoridadContextoCanalInformeJuridico = (*soporteAltaContratacionTemporalDesarrollo)(nil)

type dependenciasAltaContratacionTemporalDesarrollo struct {
	soporte     *soporteAltaContratacionTemporalDesarrollo
	servicio    *application.ServicioRegistroSolicitud
	autorizador autorizadorLigadoContratacionTemporalDesarrollo
	postgresql  dependenciasPostgreSQLContratacionTemporalDesarrollo
}

func (d *dependenciasAltaContratacionTemporalDesarrollo) cerrar() {
	if d != nil {
		d.postgresql.cerrar()
	}
}

func nuevasDependenciasAltaContratacionTemporalDesarrollo(
	dependenciasCT *DependenciasCT,
	origenOpcional ...*origenConsultasContratacionTemporalDesarrollo,
) (dependenciasAltaContratacionTemporalDesarrollo, error) {
	vacias := dependenciasAltaContratacionTemporalDesarrollo{}
	if dependenciasCT == nil {
		return vacias, ErrActivacionDesarrolloInvalida
	}
	cfg := dependenciasCT.cfg
	identidad := dependenciasCT.resolvedor
	derivador := dependenciasCT.derivador
	sello := dependenciasCT.sello
	reloj := dependenciasCT.reloj
	var origen *origenConsultasContratacionTemporalDesarrollo
	if len(origenOpcional) > 0 && origenOpcional[0] != nil {
		origen = origenOpcional[0]
	} else if cfg.PersonalOrganizacionSourcePath != "" {
		origen = nuevoOrigenConsultasContratacionTemporalDesarrollo(cfg.PersonalOrganizacionSourcePath)
	}
	if identidad == nil || derivador == nil || !derivador.valido() || sello == nil {
		return vacias, ErrActivacionDesarrolloInvalida
	}
	principal, identidadRRHHValida := identidad.principalConRolUnico(
		rolTecnicoRRHHContratacionTemporalDesarrollo,
	)
	if !identidadRRHHValida || !principalContratacionTemporalDesarrolloValido(principal) {
		return vacias, ErrActivacionDesarrolloInvalida
	}
	ahora := reloj.Ahora()
	contexto, err := nuevoContextoAltaContratacionTemporalDesarrollo(
		principal, ahora,
	)
	if err != nil {
		return vacias, err
	}
	datosVinculo, err := contexto.Vinculo.Datos()
	if err != nil {
		return vacias, err
	}
	huellas, ambitos, err := nuevasCapacidadesHMACAltaContratacionTemporalDesarrollo(
		derivador,
	)
	if err != nil {
		return vacias, err
	}
	flujo := ports.ConfiguracionAltaFlujo{
		Flujo: domain.ReferenciaFlujo{
			DefinicionRef: "flujo:ct:desarrollo",
			Version:       1,
			HuellaSHA256:  huellaAltaContratacionTemporalDesarrollo("flujo"),
		},
		FaseInicial:      domain.ClaveFase("solicitud"),
		UnidadInicialRef: "unidad:desarrollo:rrhh",
		AccionInicial:    domain.ClaveCatalogo("alta"),
	}
	motivo := dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID:           "motivos_autorizacion",
		CatalogoVersion:      1,
		CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("catalogo-motivos"),
		EntradaClave: referenciaAltaContratacionTemporalDesarrollo(
			"motivo_", "crear-solicitud",
		),
	}
	instantanea, err := nuevaInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(
		datosVinculo.PrincipalID, datosVinculo.PerfilActivoRef, ahora, origen,
	)
	instantaneaCobertura, errCobertura :=
		nuevaInstantaneaAutorizacionCoberturaContratacionTemporalDesarrollo(
			datosVinculo.PrincipalID,
			datosVinculo.PerfilActivoRef,
			ahora,
		)
	instantaneaAnalisis, errAnalisis :=
		nuevaInstantaneaAutorizacionAnalisisContratacionTemporalDesarrollo(
			datosVinculo.PrincipalID,
			datosVinculo.PerfilActivoRef,
			ahora,
		)
	instantaneaAsignacion, errAsignacion :=
		nuevaInstantaneaAutorizacionAsignacionContratacionTemporalDesarrollo(
			datosVinculo.PrincipalID,
			datosVinculo.PerfilActivoRef,
			ahora,
		)
	instantaneaInformeJuridico, errInformeJuridico :=
		nuevaInstantaneaAutorizacionInformeJuridicoContratacionTemporalDesarrollo(
			datosVinculo.PrincipalID,
			datosVinculo.PerfilActivoRef,
			ahora,
		)
	motivoPropuesta := referenciaMotivoAutorizacionCoberturaDesarrollo("propuesta")
	motivoDecision := referenciaMotivoAutorizacionCoberturaDesarrollo("decision")
	motivoRectificacion := referenciaMotivoAutorizacionCoberturaDesarrollo("rectificacion")
	motivoResultado := referenciaMotivoAutorizacionCoberturaDesarrollo("resultado")
	motivoRegistroAnalisis := referenciaMotivoAutorizacionAnalisisDesarrollo("registro")
	motivoRectificacionAnalisis := referenciaMotivoAutorizacionAnalisisDesarrollo("rectificacion")
	motivoAsignacion := referenciaMotivoAutorizacionAsignacionDesarrollo()
	motivoInformeJuridico := referenciaMotivoAutorizacionInformeJuridicoDesarrollo()
	if err != nil || errCobertura != nil || errAnalisis != nil ||
		errAsignacion != nil || errInformeJuridico != nil || flujo.Validar() != nil ||
		!dominiovec.ReferenciaMotivoAutorizacionV2Valida(motivo) {
		return vacias, errAltaContratacionTemporalDesarrolloNoDisponible
	}
	for _, referencia := range []dominiovec.ReferenciaEntradaCatalogo{
		motivoPropuesta,
		motivoDecision,
		motivoRectificacion,
		motivoResultado,
		motivoRegistroAnalisis,
		motivoRectificacionAnalisis,
		motivoAsignacion,
		motivoInformeJuridico,
	} {
		if !dominiovec.ReferenciaMotivoAutorizacionV2Valida(referencia) {
			return vacias, errAltaContratacionTemporalDesarrolloNoDisponible
		}
	}
	soporte := &soporteAltaContratacionTemporalDesarrollo{
		origen: origen,
		sello:  sello, principalID: principal.ID,
		certificadoSHA256: principal.Attributes["certificate_sha256"],
		contexto:          contexto, flujo: flujo, motivo: motivo, instantanea: instantanea,
		instantaneaAnalisis:          instantaneaAnalisis,
		instantaneaAsignacion:        instantaneaAsignacion,
		instantaneaInformeJuridico:   instantaneaInformeJuridico,
		motivoRegistroAnalisis:       motivoRegistroAnalisis,
		motivoRectificacionAnalisis:  motivoRectificacionAnalisis,
		instantaneaCobertura:         instantaneaCobertura,
		motivoPropuestaCobertura:     motivoPropuesta,
		motivoDecisionCobertura:      motivoDecision,
		motivoRectificacionCobertura: motivoRectificacion,
		motivoResultadoCobertura:     motivoResultado,
		motivoAsignacion:             motivoAsignacion,
		motivoInformeJuridico:        motivoInformeJuridico,
		ambitos:                      ambitos, reloj: reloj,
		concesiones:              make(map[string]struct{}),
		instantaneasPorSolicitud: make(map[string]dominiovec.InstantaneaAutorizacion),
	}
	generador := seguridadvec.GeneradorReferenciasCriptograficas{}
	autorizadorBase, err := aplicacionvec.NuevoServicioAutorizacionSolicitudLigadaV3(
		soporte, soporte, soporte, soporte, reloj, generador,
		aplicacionvec.ConfiguracionServicioAutorizacion{VigenciaDecision: 90 * time.Second},
	)
	if err != nil {
		return vacias, err
	}
	autorizador := &autorizadorAnalisisContratacionTemporalDesarrollo{
		delegado: autorizadorBase,
		soporte:  soporte,
	}
	postgresql, err :=
		nuevasDependenciasPostgreSQLContratacionTemporalDesarrollo(
			cfg, derivador, soporte, reloj,
		)
	if err != nil {
		return vacias, err
	}
	contador, err := postgrescontratacion.NuevoContadorNumeroVisiblePostgreSQL(postgresql.ejecucion)
	if err != nil {
		postgresql.cerrar()
		return vacias, err
	}
	referencias := seguridadcontratacion.NuevoGeneradorReferenciasAltaCriptograficoConContador(contador)
	servicio, err := application.NuevoServicioRegistroSolicitud(
		soporte, soporte, huellas, ambitos, soporte, generador,
		referencias, postgresql.candidaturas,
		postgrescontratacion.NuevoDerivadorHuellaEfectoAltaCanonico(),
		autorizador, reloj, postgresql.transaccionAlta,
	)
	if err != nil {
		postgresql.cerrar()
		return vacias, err
	}
	return dependenciasAltaContratacionTemporalDesarrollo{
		soporte:     soporte,
		servicio:    servicio,
		autorizador: autorizador,
		postgresql:  postgresql,
	}, nil
}

func nuevaRutaAltaContratacionTemporalDesarrollo(
	cfg config.Config,
	identidad *resolvedorIdentidadDesarrollo,
	derivador *derivadorIdentidadOperacionDesarrollo,
	sello *selloConsultasContratacionTemporalDesarrollo,
	reloj relojContratacionTemporalDesarrollo,
) (vechttp.RutaExacta, func(), error) {
	origen := nuevoOrigenConsultasContratacionTemporalDesarrollo(cfg.PersonalOrganizacionSourcePath)
	dependenciasCT, err := nuevasDependenciasCT(cfg, identidad, derivador, nil, nil)
	if err != nil {
		return vechttp.RutaExacta{}, nil, err
	}
	dependencias, err := nuevasDependenciasAltaContratacionTemporalDesarrollo(dependenciasCT, origen)
	if err != nil {
		return vechttp.RutaExacta{}, nil, err
	}
	ruta, err := contratacioncomposicion.NuevaRutaAlta(
		dependencias.soporte,
		dependencias.servicio,
		reloj,
	)
	if err != nil {
		dependencias.cerrar()
		return vechttp.RutaExacta{}, nil, err
	}
	return ruta, dependencias.cerrar, nil
}
