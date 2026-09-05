package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
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
	mu                                sync.Mutex
	sello                             *selloConsultasContratacionTemporalDesarrollo
	principalID                       string
	certificadoSHA256                 string
	contexto                          ports.ContextoAutorizacionAltaV3
	flujo                             ports.ConfiguracionAltaFlujo
	motivo                            dominiovec.ReferenciaEntradaCatalogo
	instantanea                       dominiovec.InstantaneaAutorizacion
	instantaneaAnalisis               dominiovec.InstantaneaAutorizacion
	motivoRegistroAnalisis            dominiovec.ReferenciaEntradaCatalogo
	instantaneaCobertura              dominiovec.InstantaneaAutorizacion
	instantaneaAsignacion             dominiovec.InstantaneaAutorizacion
	instantaneaInformeJuridico        dominiovec.InstantaneaAutorizacion
	instantaneaLlamamiento            dominiovec.InstantaneaAutorizacion
	instantaneaReanudacionLlamamiento dominiovec.InstantaneaAutorizacion
	instantaneaComunicacion           dominiovec.InstantaneaAutorizacion
	instantaneaCuadroRRHH             dominiovec.InstantaneaAutorizacion
	instantaneaDetalleRRHH            dominiovec.InstantaneaAutorizacion
	motivoCuadroRRHH                  dominiovec.ReferenciaEntradaCatalogo
	motivoDetalleRRHH                 dominiovec.ReferenciaEntradaCatalogo
	motivoLlamamiento                 dominiovec.ReferenciaEntradaCatalogo
	motivoComunicacion                dominiovec.ReferenciaEntradaCatalogo
	motivoPropuestaCobertura          dominiovec.ReferenciaEntradaCatalogo
	motivoDecisionCobertura           dominiovec.ReferenciaEntradaCatalogo
	motivoRectificacionCobertura      dominiovec.ReferenciaEntradaCatalogo
	motivoResultadoCobertura          dominiovec.ReferenciaEntradaCatalogo
	motivoAsignacion                  dominiovec.ReferenciaEntradaCatalogo
	motivoInformeJuridico             dominiovec.ReferenciaEntradaCatalogo
	ambitos                           ports.SelladorAmbitoIdempotencia
	reloj                             relojContratacionTemporalDesarrollo
	concesiones                       map[string]struct{}
	autoridadAsignaciones             autoridadAsignacionesContratacionTemporalDesarrollo
	registroDecisionesAnalisis        registroDecisionesAnalisisContratacionTemporalDesarrollo
	instantaneasPorSolicitud          map[string]dominiovec.InstantaneaAutorizacion
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
	cfg config.Config,
	identidad *resolvedorIdentidadDesarrollo,
	derivador *derivadorIdentidadOperacionDesarrollo,
	sello *selloConsultasContratacionTemporalDesarrollo,
	reloj relojContratacionTemporalDesarrollo,
) (dependenciasAltaContratacionTemporalDesarrollo, error) {
	vacias := dependenciasAltaContratacionTemporalDesarrollo{}
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
		datosVinculo.PrincipalID, datosVinculo.PerfilActivoRef, ahora,
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
		motivoAsignacion,
		motivoInformeJuridico,
	} {
		if !dominiovec.ReferenciaMotivoAutorizacionV2Valida(referencia) {
			return vacias, errAltaContratacionTemporalDesarrolloNoDisponible
		}
	}
	soporte := &soporteAltaContratacionTemporalDesarrollo{
		sello: sello, principalID: principal.ID,
		certificadoSHA256: principal.Attributes["certificate_sha256"],
		contexto:          contexto, flujo: flujo, motivo: motivo, instantanea: instantanea,
		instantaneaAnalisis:          instantaneaAnalisis,
		instantaneaAsignacion:        instantaneaAsignacion,
		instantaneaInformeJuridico:   instantaneaInformeJuridico,
		motivoRegistroAnalisis:       motivoRegistroAnalisis,
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
	referencias := seguridadcontratacion.NuevoGeneradorReferenciasAltaCriptografico()
	postgresql, err :=
		nuevasDependenciasPostgreSQLContratacionTemporalDesarrollo(
			cfg, derivador, soporte, reloj,
		)
	if err != nil {
		return vacias, err
	}
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
	dependencias, err := nuevasDependenciasAltaContratacionTemporalDesarrollo(
		cfg,
		identidad,
		derivador,
		sello,
		reloj,
	)
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

func (s *soporteAltaContratacionTemporalDesarrollo) capacidadAltaValida(
	ctx context.Context,
) bool {
	capacidad, valida := s.capacidadValida(ctx)
	return valida && capacidad.ruta == httpinterno.RutaAltaSolicitudes
}

func (s *soporteAltaContratacionTemporalDesarrollo) capacidadValida(
	ctx context.Context,
) (capacidadConsultaContratacionTemporalDesarrollo, bool) {
	if s == nil || contextoInterfazNulo(ctx) || ctx.Err() != nil || s.sello == nil ||
		s.principalID == "" || s.certificadoSHA256 == "" {
		return capacidadConsultaContratacionTemporalDesarrollo{}, false
	}
	capacidad, existe := ctx.Value(
		claveCapacidadConsultasContratacionTemporalDesarrollo{},
	).(capacidadConsultaContratacionTemporalDesarrollo)
	valida := existe && capacidad.sello == s.sello &&
		principalContratacionTemporalDesarrolloValido(capacidad.principal) &&
		capacidad.principal.ID == s.principalID &&
		capacidad.principal.Attributes["certificate_sha256"] == s.certificadoSHA256
	return capacidad, valida
}

func rutaContextoAutorizacionContratacionTemporalDesarrollo(ruta string) bool {
	return ruta == httpinterno.RutaAltaSolicitudes ||
		ruta == httpinterno.RutaPropuestaCobertura ||
		ruta == httpinterno.RutaDecisionCobertura ||
		ruta == httpinterno.RutaRectificacionCobertura ||
		ruta == httpinterno.RutaRegistroAnalisisRRHH ||
		rutaAsignacionContratacionTemporalDesarrollo(ruta) ||
		rutaInformeJuridicoContratacionTemporalDesarrollo(ruta) ||
		rutaLlamamientoContratacionTemporalDesarrollo(ruta) ||
		rutaConsultaRRHHContratacionTemporalDesarrollo(ruta)

}

func rutaAnalisisContratacionTemporalDesarrollo(ruta string) bool {
	return ruta == httpinterno.RutaRegistroAnalisisRRHH

}

func rutaCoberturaContratacionTemporalDesarrollo(ruta string) bool {
	return ruta == httpinterno.RutaPropuestaCobertura ||
		ruta == httpinterno.RutaDecisionCobertura ||
		ruta == httpinterno.RutaRectificacionCobertura ||
		ruta == httpinterno.RutaResultadoCobertura
}

func rutaAsignacionContratacionTemporalDesarrollo(ruta string) bool {
	return ruta == httpinterno.RutaAsignaciones
}

func rutaInformeJuridicoContratacionTemporalDesarrollo(ruta string) bool {
	return ruta == httpinterno.RutaPreparacionesInformeJuridico
}

func (s *soporteAltaContratacionTemporalDesarrollo) ResolverContextoCanalAlta(
	ctx context.Context,
) (application.SolicitudRegistrarExpediente, error) {
	if !s.capacidadAltaValida(ctx) {
		return application.SolicitudRegistrarExpediente{}, ports.ErrAutorizacionDenegada
	}
	vinculo, err := s.contexto.Vinculo.Datos()
	if err != nil {
		return application.SolicitudRegistrarExpediente{}, ports.ErrAutorizacionDenegada
	}
	return application.SolicitudRegistrarExpediente{
		AutenticacionRef: vinculo.AutenticacionRef,
		SesionRef:        vinculo.SesionRef,
		PerfilRef:        vinculo.PerfilActivoRef,
		OrganizacionRef:  organizacionAltaContratacionTemporalDesarrollo,
	}, nil
}

func (s *soporteAltaContratacionTemporalDesarrollo) ResolverContextoCanalAnalisisRRHH(
	ctx context.Context,
) (httpinterno.ContextoCanalAnalisisRRHH, error) {
	capacidad, valida := s.capacidadValida(ctx)
	if !valida || !rutaAnalisisContratacionTemporalDesarrollo(capacidad.ruta) {
		return httpinterno.ContextoCanalAnalisisRRHH{}, ports.ErrAutorizacionDenegada
	}
	vinculo, err := s.contexto.Vinculo.Datos()
	if err != nil {
		return httpinterno.ContextoCanalAnalisisRRHH{}, ports.ErrAutorizacionDenegada
	}
	return httpinterno.ContextoCanalAnalisisRRHH{
		AutenticacionRef: vinculo.AutenticacionRef,
		SesionRef:        vinculo.SesionRef,
		PerfilRef:        vinculo.PerfilActivoRef,
		OrganizacionRef:  organizacionAltaContratacionTemporalDesarrollo,
	}, nil
}

func (s *soporteAltaContratacionTemporalDesarrollo) ResolverContextoCanalAsignacion(
	ctx context.Context,
) (httpinterno.ContextoCanalAsignacion, error) {
	capacidad, valida := s.capacidadValida(ctx)
	if !valida || !rutaAsignacionContratacionTemporalDesarrollo(capacidad.ruta) {
		return httpinterno.ContextoCanalAsignacion{}, ports.ErrAutorizacionDenegada
	}
	vinculo, err := s.contexto.Vinculo.Datos()
	if err != nil {
		return httpinterno.ContextoCanalAsignacion{}, ports.ErrAutorizacionDenegada
	}
	return httpinterno.ContextoCanalAsignacion{
		AutenticacionRef: vinculo.AutenticacionRef,
		SesionRef:        vinculo.SesionRef,
		PerfilRef:        vinculo.PerfilActivoRef,
		OrganizacionRef:  organizacionAltaContratacionTemporalDesarrollo,
	}, nil
}

func (s *soporteAltaContratacionTemporalDesarrollo) ResolverContextoCanalInformeJuridico(
	ctx context.Context,
) (httpinterno.ContextoCanalInformeJuridico, error) {
	capacidad, valida := s.capacidadValida(ctx)
	if !valida || !rutaInformeJuridicoContratacionTemporalDesarrollo(capacidad.ruta) {
		return httpinterno.ContextoCanalInformeJuridico{}, ports.ErrAutorizacionDenegada
	}
	vinculo, err := s.contexto.Vinculo.Datos()
	if err != nil {
		return httpinterno.ContextoCanalInformeJuridico{}, ports.ErrAutorizacionDenegada
	}
	return httpinterno.ContextoCanalInformeJuridico{
		AutenticacionRef: vinculo.AutenticacionRef,
		SesionRef:        vinculo.SesionRef,
		PerfilRef:        vinculo.PerfilActivoRef,
		OrganizacionRef:  organizacionAltaContratacionTemporalDesarrollo,
	}, nil
}

func (s *soporteAltaContratacionTemporalDesarrollo) ResolverContextoAutorizacionAltaV3(
	ctx context.Context,
	solicitud ports.SolicitudResolverContextoAutorizacionAltaV3,
) (ports.ContextoAutorizacionAltaV3, error) {
	capacidad, valida := s.capacidadValida(ctx)
	if !valida || !rutaContextoAutorizacionContratacionTemporalDesarrollo(capacidad.ruta) ||
		solicitud.Validar() != nil {
		return ports.ContextoAutorizacionAltaV3{}, ports.ErrAutorizacionDenegada
	}
	vinculo, err := s.contexto.Vinculo.Datos()
	if err != nil || solicitud.AutenticacionRef != vinculo.AutenticacionRef ||
		solicitud.SesionRef != vinculo.SesionRef ||
		solicitud.PerfilRef != vinculo.PerfilActivoRef {
		return ports.ContextoAutorizacionAltaV3{}, ports.ErrAutorizacionDenegada
	}
	resultado, err := s.contexto.Resultado.Clonar()
	if err != nil {
		return ports.ContextoAutorizacionAltaV3{}, ports.ErrAutorizacionDenegada
	}
	return ports.ContextoAutorizacionAltaV3{
		Vinculo: s.contexto.Vinculo, Resultado: resultado,
	}, nil
}

func (s *soporteAltaContratacionTemporalDesarrollo) ResolverFlujoAlta(
	ctx context.Context,
	solicitud ports.SolicitudResolverFlujo,
) (ports.ConfiguracionAltaFlujo, error) {
	if !s.capacidadAltaValida(ctx) || solicitud.Validar() != nil ||
		solicitud.OrganizacionRef != organizacionAltaContratacionTemporalDesarrollo ||
		solicitud.CentroRef != centroAltaContratacionTemporalDesarrollo ||
		solicitud.CategoriaRef != categoriaAltaContratacionTemporalDesarrollo ||
		solicitud.MotivoClave != motivoAltaContratacionTemporalDesarrollo {
		return ports.ConfiguracionAltaFlujo{}, ports.ErrFlujoNoDisponible
	}
	return s.flujo, nil
}

func (s *soporteAltaContratacionTemporalDesarrollo) ResolverMotivoAutorizacionAltaV3(
	ctx context.Context,
	solicitud ports.SolicitudResolverMotivoAutorizacionAltaV3,
) (dominiovec.ReferenciaEntradaCatalogo, error) {
	if !s.capacidadAltaValida(ctx) || solicitud.Validar() != nil ||
		solicitud.OrganizacionRef != organizacionAltaContratacionTemporalDesarrollo ||
		solicitud.Flujo != s.flujo.Flujo ||
		solicitud.MotivoClave != motivoAltaContratacionTemporalDesarrollo {
		return dominiovec.ReferenciaEntradaCatalogo{}, ports.ErrMotivoAutorizacionNoDisponible
	}
	return s.motivo, nil
}

func (s *soporteAltaContratacionTemporalDesarrollo) ObtenerInstantaneaAutorizacion(
	ctx context.Context,
	principalID string,
	perfilRef string,
) (dominiovec.InstantaneaAutorizacion, error) {
	capacidad, valida := s.capacidadValida(ctx)
	instantanea, instantaneaValida := s.instantaneaParaContexto(ctx, capacidad.ruta)
	if !valida || !instantaneaValida {
		return dominiovec.InstantaneaAutorizacion{}, puertosvec.ErrFuenteAutorizacionNoDisponible
	}
	if principalID != instantanea.AsignacionPerfil.PrincipalID ||
		perfilRef != instantanea.AsignacionPerfil.PerfilActivoRef {
		return dominiovec.InstantaneaAutorizacion{}, puertosvec.ErrFuenteAutorizacionNoDisponible
	}
	return clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(instantanea), nil
}

func (s *soporteAltaContratacionTemporalDesarrollo) ValidarReferenciaMotivoAutorizacionV2(
	ctx context.Context,
	referencia dominiovec.ReferenciaEntradaCatalogo,
	instante time.Time,
) error {
	capacidad, valida := s.capacidadValida(ctx)
	esperada, motivoValido := s.motivoAutorizacionParaRuta(capacidad.ruta)
	if !valida || !motivoValido || referencia != esperada ||
		!domain.InstanteUTCCanonico(instante) {
		return dominiovec.ErrSolicitudAutorizacionInvalida
	}
	return nil
}

func (s *soporteAltaContratacionTemporalDesarrollo) RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(
	ctx context.Context,
	orden puertosvec.OrdenRegistroConcesionCandidataAutorizacionLigadaV3,
) (time.Time, error) {
	capacidad, valida := s.capacidadValida(ctx)
	datos, err := orden.Datos()
	esperada, motivoValido := s.motivoAutorizacionParaRuta(capacidad.ruta)
	if err != nil || !motivoValido || datos.ReferenciaMotivo != esperada ||
		datos.ResultadoContexto.Validar() != nil ||
		datos.Decision.ValidarPara(datos.Solicitud) != nil {
		return time.Time{}, puertosvec.ErrInstantaneaAutorizacionObsoleta
	}
	if !valida {
		return time.Time{}, puertosvec.ErrInstantaneaAutorizacionObsoleta
	}
	huella, err := dominiovec.HuellaSHA256DecisionAutorizacionV3(datos.Decision)
	desde, hasta, errVentana := datos.Decision.VentanaValidez()
	ahora := s.reloj.Ahora()
	if err != nil || errVentana != nil || ahora.Before(desde) || !ahora.Before(hasta) {
		return time.Time{}, puertosvec.ErrInstantaneaAutorizacionObsoleta
	}
	var instantanea dominiovec.InstantaneaAutorizacion
	var registroAnalisis registroDecisionesAnalisisContratacionTemporalDesarrollo
	if capacidad.ruta == httpinterno.RutaAltaSolicitudes ||
		rutaMutacionDurableContratacionTemporalDesarrollo(capacidad.ruta) ||
		rutaConsultaRRHHContratacionTemporalDesarrollo(capacidad.ruta) {
		clave, claveValida := claveInstantaneaContratacionTemporalDesarrollo(
			datos.Solicitud,
		)
		s.mu.Lock()
		instantanea, valida = s.instantaneasPorSolicitud[clave]
		autoridad := s.autoridadAsignaciones
		if rutaMutacionDurableContratacionTemporalDesarrollo(capacidad.ruta) ||
			rutaConsultaRRHHContratacionTemporalDesarrollo(capacidad.ruta) {
			registroAnalisis = s.registroDecisionesAnalisis
		}
		s.mu.Unlock()
		if !claveValida || !valida || autoridad == nil ||
			instantanea.Validar() != nil ||
			autoridad.PublicarInstantanea(ctx, instantanea) != nil {
			return time.Time{}, puertosvec.ErrInstantaneaAutorizacionObsoleta
		}
		if (rutaMutacionDurableContratacionTemporalDesarrollo(capacidad.ruta) ||
			rutaConsultaRRHHContratacionTemporalDesarrollo(capacidad.ruta)) &&
			registroAnalisis == nil {
			return time.Time{}, puertosvec.ErrInstantaneaAutorizacionObsoleta
		}
	} else {
		var instantaneaValida bool
		instantanea, instantaneaValida = s.instantaneaParaContexto(ctx, capacidad.ruta)
		if !instantaneaValida || instantanea.Validar() != nil {
			return time.Time{}, puertosvec.ErrInstantaneaAutorizacionObsoleta
		}
	}
	registradaEn := ahora
	if registroAnalisis != nil {
		registradaEn, err = registroAnalisis.
			RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(ctx, orden)
		if err != nil {
			return time.Time{}, err
		}
		return registradaEn, nil
	}
	s.mu.Lock()
	s.concesiones[huella] = struct{}{}
	s.mu.Unlock()
	return registradaEn, nil
}

func (s *soporteAltaContratacionTemporalDesarrollo) RegistrarDenegacionAutorizacionLigadaV3(
	ctx context.Context,
	orden puertosvec.OrdenRegistroDenegacionAutorizacionLigadaV3,
) error {
	capacidad, valida := s.capacidadValida(ctx)
	if !valida {
		return puertosvec.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible
	}
	datos, err := orden.Datos()
	esperada, motivoValido := s.motivoAutorizacionParaRuta(capacidad.ruta)
	if err != nil || !motivoValido || datos.ReferenciaMotivo != esperada {
		return puertosvec.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible
	}
	if rutaMutacionDurableContratacionTemporalDesarrollo(capacidad.ruta) ||
		rutaConsultaRRHHContratacionTemporalDesarrollo(capacidad.ruta) {
		clave, claveValida := claveInstantaneaContratacionTemporalDesarrollo(
			datos.Solicitud,
		)
		s.mu.Lock()
		instantanea, existe := s.instantaneasPorSolicitud[clave]
		autoridad := s.autoridadAsignaciones
		registro := s.registroDecisionesAnalisis
		s.mu.Unlock()
		if !claveValida || !existe || autoridad == nil || registro == nil ||
			instantanea.Validar() != nil ||
			autoridad.PublicarInstantanea(ctx, instantanea) != nil {
			return puertosvec.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible
		}
		if err := registro.RegistrarDenegacionAutorizacionLigadaV3(ctx, orden); err != nil {
			return err
		}
	}
	return nil
}

func (s *soporteAltaContratacionTemporalDesarrollo) motivoAutorizacionParaRuta(
	ruta string,
) (dominiovec.ReferenciaEntradaCatalogo, bool) {
	if s == nil {
		return dominiovec.ReferenciaEntradaCatalogo{}, false
	}
	switch ruta {
	case httpinterno.RutaConsultaCuadroRRHH:
		return s.motivoCuadroRRHH, dominiovec.ReferenciaMotivoAutorizacionV2Valida(s.motivoCuadroRRHH)
	case httpinterno.RutaConsultaDetalleRRHH:
		return s.motivoDetalleRRHH, dominiovec.ReferenciaMotivoAutorizacionV2Valida(s.motivoDetalleRRHH)
	case httpinterno.RutaAltaSolicitudes:
		return s.motivo, true
	case httpinterno.RutaPropuestaCobertura:
		return s.motivoPropuestaCobertura, true
	case httpinterno.RutaDecisionCobertura:
		return s.motivoDecisionCobertura, true
	case httpinterno.RutaRectificacionCobertura:
		return s.motivoRectificacionCobertura, true
	case httpinterno.RutaRegistroAnalisisRRHH:
		return s.motivoRegistroAnalisis, true
	case httpinterno.RutaResultadoCobertura:
		return s.motivoResultadoCobertura, true
	case httpinterno.RutaAsignaciones:
		return s.motivoAsignacion, true
	case httpinterno.RutaPreparacionesInformeJuridico:
		return s.motivoInformeJuridico, true
	case httpinterno.RutaSeleccionLlamamiento:
		return s.motivoLlamamiento, true
	case httpinterno.RutaRegistroComunicacionLlamamiento, httpinterno.RutaResolucionComunicacionLlamamiento:
		return s.motivoComunicacion, true
	default:
		return dominiovec.ReferenciaEntradaCatalogo{}, false
	}
}

func (s *soporteAltaContratacionTemporalDesarrollo) instantaneaParaRuta(
	ruta string,
) (dominiovec.InstantaneaAutorizacion, bool) {
	if s == nil {
		return dominiovec.InstantaneaAutorizacion{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if ruta == httpinterno.RutaConsultaCuadroRRHH {
		return clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(s.instantaneaCuadroRRHH), s.instantaneaCuadroRRHH.Validar() == nil
	}
	if ruta == httpinterno.RutaConsultaDetalleRRHH {
		return clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(s.instantaneaDetalleRRHH), s.instantaneaDetalleRRHH.Validar() == nil
	}
	if ruta == httpinterno.RutaAltaSolicitudes {
		return clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(s.instantanea), true
	}
	if rutaCoberturaContratacionTemporalDesarrollo(ruta) {
		return clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(s.instantaneaCobertura), true
	}
	if rutaAnalisisContratacionTemporalDesarrollo(ruta) {
		return clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(s.instantaneaAnalisis), true
	}
	if rutaAsignacionContratacionTemporalDesarrollo(ruta) {
		return clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(s.instantaneaAsignacion), true
	}
	if rutaInformeJuridicoContratacionTemporalDesarrollo(ruta) {
		return clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(s.instantaneaInformeJuridico), true
	}
	if ruta == httpinterno.RutaSeleccionLlamamiento {
		return clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(s.instantaneaLlamamiento), true
	}
	if rutaLlamamientoContratacionTemporalDesarrollo(ruta) {
		return clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(s.instantaneaComunicacion), true
	}
	return dominiovec.InstantaneaAutorizacion{}, false
}

func (s *soporteAltaContratacionTemporalDesarrollo) instantaneaParaContexto(
	ctx context.Context,
	ruta string,
) (dominiovec.InstantaneaAutorizacion, bool) {
	instantanea, valida := s.instantaneaParaRuta(ruta)
	dinamica := ruta == httpinterno.RutaAltaSolicitudes ||
		rutaMutacionDurableContratacionTemporalDesarrollo(ruta) ||
		rutaConsultaRRHHContratacionTemporalDesarrollo(ruta) ||
		ruta == httpinterno.RutaDecisionCobertura ||
		ruta == httpinterno.RutaRectificacionCobertura
	if !valida || !dinamica {
		return instantanea, valida
	}
	if ctx == nil {
		return dominiovec.InstantaneaAutorizacion{}, false
	}
	datos, existe := ctx.Value(
		claveSolicitudAutorizacionContratacionTemporalDesarrollo{},
	).(dominiovec.DatosSolicitudAutorizacionLigadaV3)
	if !existe {
		if ruta == httpinterno.RutaAltaSolicitudes {
			return instantanea, true
		}
		return dominiovec.InstantaneaAutorizacion{}, false
	}
	if rutaConsultaRRHHContratacionTemporalDesarrollo(ruta) {
		if !s.solicitudAutorizacionConsultaRRHHDesarrolloValida(ruta, datos) {
			return dominiovec.InstantaneaAutorizacion{}, false
		}
		// El ámbito de organización es fijo; nunca se amplía desde la petición.
	} else if rutaAnalisisContratacionTemporalDesarrollo(ruta) {
		if !solicitudAutorizacionAnalisisContratacionTemporalDesarrolloValida(ruta, datos) {
			return dominiovec.InstantaneaAutorizacion{}, false
		}
		instantanea.AsignacionPerfil.Ambitos = []dominiovec.AmbitoPerfil{
			{Clave: "organizacion_ref", Valores: []string{datos.Recurso.Ambitos["organizacion_ref"]}},
			{Clave: "expediente_ref", Valores: []string{datos.Recurso.Ambitos["expediente_ref"]}},
			{Clave: "fase_previa", Valores: []string{datos.Recurso.Ambitos["fase_previa"]}},
			{Clave: "estado_previo", Valores: []string{datos.Recurso.Ambitos["estado_previo"]}},
		}
	} else if rutaAsignacionContratacionTemporalDesarrollo(ruta) {
		if !solicitudAutorizacionAsignacionContratacionTemporalDesarrolloValida(
			ruta,
			datos,
		) {
			return dominiovec.InstantaneaAutorizacion{}, false
		}
		instantanea.AsignacionPerfil.Ambitos = []dominiovec.AmbitoPerfil{
			{Clave: "organizacion_ref", Valores: []string{datos.Recurso.Ambitos["organizacion_ref"]}},
			{Clave: "expediente_ref", Valores: []string{datos.Recurso.Ambitos["expediente_ref"]}},
			{Clave: "fase_previa", Valores: []string{datos.Recurso.Ambitos["fase_previa"]}},
			{Clave: "estado_previo", Valores: []string{datos.Recurso.Ambitos["estado_previo"]}},
			{Clave: "unidad_destino_ref", Valores: []string{datos.Recurso.Ambitos["unidad_destino_ref"]}},
		}
	} else if rutaInformeJuridicoContratacionTemporalDesarrollo(ruta) {
		if !solicitudAutorizacionInformeJuridicoContratacionTemporalDesarrolloValida(
			ruta,
			datos,
		) {
			return dominiovec.InstantaneaAutorizacion{}, false
		}
		instantanea.AsignacionPerfil.Ambitos = []dominiovec.AmbitoPerfil{
			{Clave: "organizacion_ref", Valores: []string{datos.Recurso.Ambitos["organizacion_ref"]}},
			{Clave: "expediente_ref", Valores: []string{datos.Recurso.Ambitos["expediente_ref"]}},
			{Clave: "fase_previa", Valores: []string{datos.Recurso.Ambitos["fase_previa"]}},
			{Clave: "estado_previo", Valores: []string{datos.Recurso.Ambitos["estado_previo"]}},
		}
	} else if rutaLlamamientoContratacionTemporalDesarrollo(ruta) {
		if !solicitudAutorizacionLlamamientoDesarrolloValida(ctx, ruta, datos) {
			return dominiovec.InstantaneaAutorizacion{}, false
		}
		if datos.Accion == ports.AccionReanudacionSeleccionLlamamiento {
			s.mu.Lock()
			instantanea = clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(s.instantaneaReanudacionLlamamiento)
			s.mu.Unlock()
			if instantanea.Validar() != nil {
				return dominiovec.InstantaneaAutorizacion{}, false
			}
		}
		instantanea.AsignacionPerfil.Ambitos = ambitosLlamamientoDesarrollo(datos.Recurso)
	} else if ruta == httpinterno.RutaDecisionCobertura ||
		ruta == httpinterno.RutaRectificacionCobertura {
		if !solicitudAutorizacionDecisionCoberturaDesarrolloValida(ruta, datos) {
			return dominiovec.InstantaneaAutorizacion{}, false
		}
	} else if !solicitudAutorizacionAltaContratacionTemporalDesarrolloValida(ruta, datos) {
		return dominiovec.InstantaneaAutorizacion{}, false
	}
	clave, claveValida := claveInstantaneaContratacionTemporalDesarrolloDesdeDatos(
		datos,
	)
	if !claveValida {
		return dominiovec.InstantaneaAutorizacion{}, false
	}
	s.mu.Lock()
	preparada, existe := s.instantaneasPorSolicitud[clave]
	autoridad := s.autoridadAsignaciones
	s.mu.Unlock()
	if existe {
		return clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(preparada),
			preparada.Validar() == nil
	}
	if autoridad == nil {
		return dominiovec.InstantaneaAutorizacion{}, false
	}
	preparada, err := autoridad.PrepararInstantanea(ctx, instantanea)
	if err != nil || preparada.Validar() != nil {
		return dominiovec.InstantaneaAutorizacion{}, false
	}
	s.mu.Lock()
	if existente, yaExiste := s.instantaneasPorSolicitud[clave]; yaExiste {
		preparada = existente
	} else {
		s.instantaneasPorSolicitud[clave] = preparada
	}
	s.mu.Unlock()
	return clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(preparada),
		preparada.Validar() == nil
}

func (s *soporteAltaContratacionTemporalDesarrollo) publicarInstantaneaDecisionCobertura(
	ctx context.Context,
	ruta string,
) error {
	capacidad, valida := s.capacidadValida(ctx)
	if !valida || capacidad.ruta != ruta ||
		(ruta != httpinterno.RutaDecisionCobertura &&
			ruta != httpinterno.RutaRectificacionCobertura) {
		return errAltaContratacionTemporalDesarrolloNoDisponible
	}
	instantanea, valida := s.instantaneaParaContexto(ctx, ruta)
	if !valida || instantanea.Validar() != nil {
		return errAltaContratacionTemporalDesarrolloNoDisponible
	}
	s.mu.Lock()
	autoridad := s.autoridadAsignaciones
	s.mu.Unlock()
	if autoridad == nil || autoridad.PublicarInstantanea(ctx, instantanea) != nil {
		return errAltaContratacionTemporalDesarrolloNoDisponible
	}
	return nil
}

func claveInstantaneaContratacionTemporalDesarrollo(
	solicitud dominiovec.SolicitudAutorizacionLigadaV3,
) (string, bool) {
	huella, err := dominiovec.HuellaSHA256SolicitudAutorizacionV3(solicitud)
	return huella, err == nil && huella != ""
}

func claveInstantaneaContratacionTemporalDesarrolloDesdeDatos(
	datos dominiovec.DatosSolicitudAutorizacionLigadaV3,
) (string, bool) {
	solicitud, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(datos)
	if err != nil {
		return "", false
	}
	return claveInstantaneaContratacionTemporalDesarrollo(solicitud)
}

func solicitudAutorizacionAltaContratacionTemporalDesarrolloValida(
	ruta string,
	datos dominiovec.DatosSolicitudAutorizacionLigadaV3,
) bool {
	return ruta == httpinterno.RutaAltaSolicitudes &&
		datos.Accion == ports.AccionCrearSolicitud &&
		datos.Recurso.ModuloID == ports.ModuloContratacion &&
		datos.Recurso.Tipo == ports.TipoRecursoExpediente &&
		datos.Finalidad == ports.FinalidadCrearSolicitud &&
		len(datos.Recurso.Ambitos) == 3 &&
		datos.Recurso.Ambitos["organizacion_ref"] == organizacionAltaContratacionTemporalDesarrollo &&
		datos.Recurso.Ambitos["centro_ref"] == centroAltaContratacionTemporalDesarrollo &&
		datos.Recurso.Ambitos["categoria_ref"] == categoriaAltaContratacionTemporalDesarrollo
}

func solicitudAutorizacionAnalisisContratacionTemporalDesarrolloValida(
	ruta string,
	datos dominiovec.DatosSolicitudAutorizacionLigadaV3,
) bool {
	accionValida := ruta == httpinterno.RutaRegistroAnalisisRRHH &&
		datos.Accion == ports.AccionRegistrarAnalisis
	return accionValida && datos.Recurso.ModuloID == ports.ModuloContratacion &&
		datos.Recurso.Tipo == ports.TipoRecursoAnalisis &&
		datos.Finalidad == finalidadAnalisisContratacionTemporalDesarrollo &&
		len(datos.Recurso.Ambitos) == 4 &&
		datos.Recurso.Ambitos["organizacion_ref"] == organizacionAltaContratacionTemporalDesarrollo &&
		datos.Recurso.Ambitos["expediente_ref"] == datos.Recurso.Referencia &&
		datos.Recurso.Atributos[ports.AtributoUnidadPoliticaRef] == unidadCoberturaContratacionTemporalDesarrollo
}

func solicitudAutorizacionDecisionCoberturaDesarrolloValida(
	ruta string,
	datos dominiovec.DatosSolicitudAutorizacionLigadaV3,
) bool {
	accion := string(domain.AccionDecidirCoberturaGobernada)
	if ruta == httpinterno.RutaRectificacionCobertura {
		accion = string(domain.AccionRectificarCoberturaGobernada)
	} else if ruta != httpinterno.RutaDecisionCobertura {
		return false
	}
	return datos.Accion == accion &&
		datos.Recurso.ModuloID == ports.ModuloContratacion &&
		datos.Recurso.Tipo == tipoRecursoDecisionCoberturaDesarrollo &&
		datos.Finalidad == finalidadDecisionCoberturaDesarrollo &&
		len(datos.Recurso.Ambitos) == 2 &&
		datos.Recurso.Ambitos["organizacion_ref"] ==
			organizacionAltaContratacionTemporalDesarrollo &&
		datos.Recurso.Ambitos["unidad_ejecutora_ref"] ==
			unidadCoberturaContratacionTemporalDesarrollo
}

func nuevaInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(
	principalID string,
	perfilRef string,
	ahora time.Time,
) (dominiovec.InstantaneaAutorizacion, error) {
	return nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(
		principalID,
		perfilRef,
		ahora,
		"tecnico_rrhh_desarrollo",
		"Tecnico RRHH de desarrollo",
		"asignacion-rrhh-desarrollo-no-autoritativa",
		[]dominiovec.ConcesionRol{{
			Accion:         ports.AccionCrearSolicitud,
			ModuloID:       ports.ModuloContratacion,
			TipoRecurso:    ports.TipoRecursoExpediente,
			Finalidades:    []string{ports.FinalidadCrearSolicitud},
			GarantiaMinima: dominiovec.AuthAssuranceHigh,
		}},
		[]dominiovec.AmbitoPerfil{
			{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}},
			{Clave: "centro_ref", Valores: []string{centroAltaContratacionTemporalDesarrollo}},
			{Clave: "categoria_ref", Valores: []string{categoriaAltaContratacionTemporalDesarrollo}},
		},
	)
}

func nuevaInstantaneaAutorizacionCoberturaContratacionTemporalDesarrollo(
	principalID string,
	perfilRef string,
	ahora time.Time,
) (dominiovec.InstantaneaAutorizacion, error) {
	concesion := func(accion, finalidad, tipoRecurso string) dominiovec.ConcesionRol {
		return dominiovec.ConcesionRol{
			Accion: accion, ModuloID: ports.ModuloContratacion,
			TipoRecurso:    tipoRecurso,
			Finalidades:    []string{finalidad},
			GarantiaMinima: dominiovec.AuthAssuranceHigh,
		}
	}
	return nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(
		principalID,
		perfilRef,
		ahora,
		"tecnico_rrhh_cobertura_desarrollo",
		"Tecnico RRHH de cobertura de desarrollo",
		"asignacion-rrhh-cobertura-desarrollo-no-autoritativa",
		[]dominiovec.ConcesionRol{
			concesion(accionPropuestaCoberturaDesarrollo, finalidadPropuestaCoberturaDesarrollo, ports.TipoRecursoExpediente),
			concesion(string(domain.AccionDecidirCoberturaGobernada), finalidadDecisionCoberturaDesarrollo, tipoRecursoDecisionCoberturaDesarrollo),
			concesion(string(domain.AccionRectificarCoberturaGobernada), finalidadDecisionCoberturaDesarrollo, tipoRecursoDecisionCoberturaDesarrollo),
			concesion(
				string(ports.AccionConsultarResultadoCobertura),
				string(ports.FinalidadRecuperarResultadoCobertura),
				ports.TipoRecursoExpediente,
			),
		},
		[]dominiovec.AmbitoPerfil{
			{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}},
			{Clave: "unidad_ejecutora_ref", Valores: []string{unidadCoberturaContratacionTemporalDesarrollo}},
		},
	)
}

func nuevaInstantaneaAutorizacionAnalisisContratacionTemporalDesarrollo(
	principalID string,
	perfilRef string,
	ahora time.Time,
) (dominiovec.InstantaneaAutorizacion, error) {
	concesion := func(accion string) dominiovec.ConcesionRol {
		return dominiovec.ConcesionRol{
			Accion: accion, ModuloID: ports.ModuloContratacion,
			TipoRecurso:    ports.TipoRecursoAnalisis,
			Finalidades:    []string{finalidadAnalisisContratacionTemporalDesarrollo},
			GarantiaMinima: dominiovec.AuthAssuranceHigh,
		}
	}
	instantanea, err := nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(
		principalID,
		perfilRef,
		ahora,
		"tecnico_rrhh_analisis_desarrollo",
		"Tecnico RRHH de analisis de desarrollo",
		"asignacion-rrhh-analisis-desarrollo-no-autoritativa",
		[]dominiovec.ConcesionRol{
			concesion(ports.AccionRegistrarAnalisis),
		},
		[]dominiovec.AmbitoPerfil{
			{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}},
			{Clave: "expediente_ref", Valores: []string{expedienteContratacionTemporalDesarrolloRef}},
			{Clave: "fase_previa", Valores: []string{"solicitud"}},
			{Clave: "estado_previo", Valores: []string{string(domain.EstadoEnCurso)}},
		},
	)
	if err != nil {
		return dominiovec.InstantaneaAutorizacion{}, err
	}
	if instantanea.Validar() != nil {
		return dominiovec.InstantaneaAutorizacion{}, errAltaContratacionTemporalDesarrolloNoDisponible
	}
	return instantanea, nil
}

func nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(
	principalID string,
	perfilRef string,
	ahora time.Time,
	rolID string,
	nombreRol string,
	asignacionID string,
	concesiones []dominiovec.ConcesionRol,
	ambitos []dominiovec.AmbitoPerfil,
) (dominiovec.InstantaneaAutorizacion, error) {
	desde, hasta, vigente := ventanaAutoridadSinteticaContratacionTemporalDesarrollo(ahora)
	if !vigente {
		return dominiovec.InstantaneaAutorizacion{},
			errAltaContratacionTemporalDesarrolloNoDisponible
	}
	publicadaEn := desde
	asignacionID = referenciaAltaContratacionTemporalDesarrollo(
		"asg_", principalID+"\x00"+perfilRef+"\x00"+asignacionID,
	)
	version := dominiovec.VersionRol{
		RolID: rolID, Version: 1, Nombre: nombreRol,
		Estado:       dominiovec.EstadoVersionRolPublicada,
		Concesiones:  concesiones,
		PublicadaPor: "seguridad:desarrollo:no-autoritativa",
		PublicadaEn:  publicadaEn,
	}
	asignacion := dominiovec.AsignacionPerfil{
		AsignacionID: asignacionID,
		Version:      1, PerfilActivoRef: perfilRef, PrincipalID: principalID,
		VersionRolRef: version.Referencia(),
		Estado:        dominiovec.EstadoAsignacionPerfilActiva,
		Ambitos:       ambitos,
		VigenteDesde:  desde,
		VigenteHasta:  hasta,
		EmitidaPor:    "identidad:desarrollo:no-autoritativa",
		EmitidaEn:     desde,
	}
	huellaPoliticas, err := dominiovec.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		return dominiovec.InstantaneaAutorizacion{}, err
	}
	instantanea := dominiovec.InstantaneaAutorizacion{
		AsignacionPerfil: asignacion,
		VersionRol:       version,
		ControlVigenciaVersionRol: dominiovec.ControlVigenciaVersionRol{
			VersionRolRef: version.Referencia(), Revision: 1,
			Estado:         dominiovec.EstadoControlVigenciaVersionRolHabilitada,
			ActualizadoPor: version.PublicadaPor, ActualizadoEn: publicadaEn,
		},
		RevisionCatalogoPoliticas:     1,
		CatalogoPoliticasHuellaSHA256: huellaPoliticas,
	}
	if instantanea.Validar() != nil {
		return dominiovec.InstantaneaAutorizacion{}, errAltaContratacionTemporalDesarrolloNoDisponible
	}
	return instantanea, nil
}

func referenciaMotivoAutorizacionCoberturaDesarrollo(
	operacion string,
) dominiovec.ReferenciaEntradaCatalogo {
	return dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID:           "motivos_autorizacion_cobertura",
		CatalogoVersion:      1,
		CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("catalogo-motivos-cobertura"),
		EntradaClave: referenciaAltaContratacionTemporalDesarrollo(
			"motivo_",
			"cobertura-"+operacion,
		),
	}
}

func referenciaMotivoAutorizacionAnalisisDesarrollo(
	operacion string,
) dominiovec.ReferenciaEntradaCatalogo {
	return dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID:           "motivos_autorizacion_analisis",
		CatalogoVersion:      1,
		CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("catalogo-motivos-analisis"),
		EntradaClave: referenciaAltaContratacionTemporalDesarrollo(
			"motivo_",
			"analisis-"+operacion,
		),
	}
}

func clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(
	instantanea dominiovec.InstantaneaAutorizacion,
) dominiovec.InstantaneaAutorizacion {
	copia := instantanea
	copia.VersionRol.Concesiones = append(
		[]dominiovec.ConcesionRol(nil), instantanea.VersionRol.Concesiones...,
	)
	for indice := range copia.VersionRol.Concesiones {
		copia.VersionRol.Concesiones[indice].Finalidades = append(
			[]string(nil), instantanea.VersionRol.Concesiones[indice].Finalidades...,
		)
	}
	copia.AsignacionPerfil.Ambitos = append(
		[]dominiovec.AmbitoPerfil(nil), instantanea.AsignacionPerfil.Ambitos...,
	)
	for indice := range copia.AsignacionPerfil.Ambitos {
		copia.AsignacionPerfil.Ambitos[indice].Valores = append(
			[]string(nil), instantanea.AsignacionPerfil.Ambitos[indice].Valores...,
		)
	}
	copia.Politicas = append([]dominiovec.PoliticaRestrictiva(nil), instantanea.Politicas...)
	for indice := range copia.Politicas {
		politicaOrigen := instantanea.Politicas[indice]
		politica := &copia.Politicas[indice]
		politica.Acciones = append([]string(nil), politicaOrigen.Acciones...)
		politica.Modulos = append([]string(nil), politicaOrigen.Modulos...)
		politica.TiposRecurso = append([]string(nil), politicaOrigen.TiposRecurso...)
		politica.FinalidadesPermitidas = append(
			[]string(nil), politicaOrigen.FinalidadesPermitidas...,
		)
		politica.Restricciones = append(
			[]dominiovec.RestriccionAtributoRecurso(nil), politicaOrigen.Restricciones...,
		)
		for indiceRestriccion := range politica.Restricciones {
			politica.Restricciones[indiceRestriccion].ValoresPermitidos = append(
				[]string(nil), politicaOrigen.Restricciones[indiceRestriccion].ValoresPermitidos...,
			)
		}
	}
	return copia
}

type selladorHMACAltaContratacionTemporalDesarrollo struct {
	derivador *derivadorIdentidadOperacionDesarrollo
	indice    int
	ambito    bool
	dominio   string
}

func (s *selladorHMACAltaContratacionTemporalDesarrollo) SellarDatos(
	ctx context.Context,
	datos []byte,
) (string, error) {
	if s == nil || ctx == nil || ctx.Err() != nil || len(datos) == 0 ||
		s.derivador == nil || !s.derivador.valido() {
		return "", seguridadcontratacion.ErrSelladoAltaNoDisponible
	}
	resultados, err := s.derivador.calcularHMAC(datos, datos)
	if err != nil {
		return "", err
	}
	defer borrarResultadosHMACIdempotenciaDesarrollo(resultados)
	if s.indice < 0 || s.indice >= len(resultados) {
		return "", seguridadcontratacion.ErrSelladoAltaNoDisponible
	}
	resultado := resultados[s.indice]
	valor := resultado.huellaSolicitud[:]
	if s.ambito {
		valor = resultado.localizador[:]
	}
	return fmt.Sprintf(
		"hmac-sha256:%s/v%d:%s",
		s.dominio, resultado.generacion, hex.EncodeToString(valor),
	), nil
}

func nuevasCapacidadesHMACAltaContratacionTemporalDesarrollo(
	derivador *derivadorIdentidadOperacionDesarrollo,
) (
	ports.DerivadorHuellaAlta,
	ports.SelladorAmbitoIdempotencia,
	error,
) {
	huellaActiva, huellasRetenidas, err := configuracionesHMACAltaContratacionTemporalDesarrollo(
		derivador, "vec.contratacion-temporal.huella-peticion", false,
	)
	if err != nil {
		return nil, nil, err
	}
	ambitoActivo, ambitosRetenidos, err := configuracionesHMACAltaContratacionTemporalDesarrollo(
		derivador, "vec.contratacion-temporal.ambito-idempotencia", true,
	)
	if err != nil {
		return nil, nil, err
	}
	huellas, err := seguridadcontratacion.NuevoDerivadorHuellaAltaHMACRotable(
		huellaActiva, huellasRetenidas,
	)
	if err != nil {
		return nil, nil, err
	}
	ambitos, err := seguridadcontratacion.NuevoSelladorAmbitoIdempotenciaHMACRotable(
		ambitoActivo, ambitosRetenidos,
	)
	if err != nil {
		return nil, nil, err
	}
	return huellas, ambitos, nil
}

func configuracionesHMACAltaContratacionTemporalDesarrollo(
	derivador *derivadorIdentidadOperacionDesarrollo,
	dominio string,
	ambito bool,
) (
	seguridadcontratacion.ConfiguracionSelladorHMAC,
	[]seguridadcontratacion.ConfiguracionSelladorHMAC,
	error,
) {
	if derivador == nil || !derivador.valido() {
		return seguridadcontratacion.ConfiguracionSelladorHMAC{}, nil,
			seguridadcontratacion.ErrSelladoAltaNoDisponible
	}
	configuraciones := make(
		[]seguridadcontratacion.ConfiguracionSelladorHMAC,
		len(derivador.generaciones),
	)
	for indice, generacion := range derivador.generaciones {
		sellador := &selladorHMACAltaContratacionTemporalDesarrollo{
			derivador: derivador, indice: indice, ambito: ambito, dominio: dominio,
		}
		configuracion, err := seguridadcontratacion.NuevaConfiguracionSelladorHMAC(
			fmt.Sprintf("%s/v%d", dominio, generacion.generacion),
			sellador,
		)
		if err != nil {
			return seguridadcontratacion.ConfiguracionSelladorHMAC{}, nil, err
		}
		configuraciones[indice] = configuracion
	}
	return configuraciones[0], configuraciones[1:], nil
}

func nuevoContextoAltaContratacionTemporalDesarrollo(
	principal dominiovec.Principal,
	ahora time.Time,
) (ports.ContextoAutorizacionAltaV3, error) {
	if !principalContratacionTemporalDesarrolloValido(principal) {
		return ports.ContextoAutorizacionAltaV3{},
			errAltaContratacionTemporalDesarrolloNoDisponible
	}
	return nuevoContextoSinteticoContratacionTemporalDesarrollo(principal, ahora)
}

func nuevoContextoSinteticoContratacionTemporalDesarrollo(
	principal dominiovec.Principal,
	ahora time.Time,
) (ports.ContextoAutorizacionAltaV3, error) {
	if !principalSinteticoContratacionTemporalDesarrolloValido(principal) ||
		!domain.InstanteUTCCanonico(ahora) {
		return ports.ContextoAutorizacionAltaV3{},
			errAltaContratacionTemporalDesarrolloNoDisponible
	}
	desde, hasta, vigente := ventanaAutoridadSinteticaContratacionTemporalDesarrollo(ahora)
	if !vigente {
		return ports.ContextoAutorizacionAltaV3{},
			errAltaContratacionTemporalDesarrolloNoDisponible
	}
	resueltoEn := desde.Add(3 * time.Minute)
	base := principal.ID + "\x00" + principal.Attributes["certificate_sha256"]
	cuentaRef := referenciaAltaContratacionTemporalDesarrollo("cta_", base+"\x00cuenta")
	personaRef := referenciaAltaContratacionTemporalDesarrollo("per_", base+"\x00persona")
	perfilRef := referenciaAltaContratacionTemporalDesarrollo("prf_", base+"\x00perfil")
	vinculoRef := referenciaAltaContratacionTemporalDesarrollo("vca_", base+"\x00vinculo")
	cuenta := dominiovec.CuentaAutenticadaContextoActor{
		CuentaRef: cuentaRef,
		Metodo:    dominiovec.AuthMethodCertificate,
		Garantia:  dominiovec.AuthAssuranceHigh,
	}
	instantanea := dominiovec.InstantaneaContextoActor{
		VinculoRef: vinculoRef, VinculoVersion: 1,
		CuentaRef: cuentaRef, CuentaVersion: 1,
		PersonaRef: personaRef, PersonaVersion: 1,
		PerfilActivoRef: perfilRef, PerfilVersion: 1,
		Estado:       dominiovec.EstadoVinculoContextoActorActivo,
		VigenteDesde: desde,
		VigenteHasta: hasta,
		Vinculos:     []dominiovec.VinculoReferenciaContextoActor{},
	}
	actor, err := dominiovec.NuevoContextoActor(cuenta, instantanea, resueltoEn)
	if err != nil {
		return ports.ContextoAutorizacionAltaV3{}, err
	}
	canon, err := actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		return ports.ContextoAutorizacionAltaV3{}, err
	}
	huellaContexto, err := actor.HuellaSHA256VinculadaV2()
	if err != nil {
		return ports.ContextoAutorizacionAltaV3{}, err
	}
	// La etiqueta de autoridad es una precondicion estructural del contrato V3;
	// las referencias que la acompañan siguen marcadas como desarrollo efimero
	// y nunca salen de la composicion protegida por doble llave y mTLS.
	acreditacion := dominiovec.AcreditacionProcedenciaComponenteContextoActorV1{
		ProcedenciaRef: referenciaAltaContratacionTemporalDesarrollo(
			"prc_", base+"\x00procedencia",
		),
		ProcedenciaVersion: 1,
		ProcedenciaHuellaSHA256: huellaAltaContratacionTemporalDesarrollo(
			base + "\x00procedencia",
		),
		ProcedenciaAutoridad: dominiovec.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
	}
	manifiesto := dominiovec.ManifiestoProcedenciaContextoActorV1{
		Esquema:           dominiovec.EsquemaManifiestoProcedenciaContextoActorV1,
		AutoridadEfectiva: dominiovec.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
		Cuenta: dominiovec.ProcedenciaCuentaContextoActorV1{
			CuentaRef: cuentaRef, Version: 1,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Persona: dominiovec.ProcedenciaPersonaContextoActorV1{
			PersonaRef: personaRef, Version: 1,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Perfil: dominiovec.ProcedenciaPerfilContextoActorV1{
			PerfilRef: perfilRef, Version: 1,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Contexto: dominiovec.ProcedenciaVinculoContextoActorV1{
			VinculoRef: vinculoRef, Version: 1,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Vinculos: []dominiovec.ProcedenciaVinculoReferenciaContextoActorV1{},
	}
	manifiestoCanon, err := manifiesto.RepresentacionCanonicaV1()
	if err != nil {
		return ports.ContextoAutorizacionAltaV3{}, err
	}
	manifiestoHuella, err := dominiovec.HuellaSHA256ManifiestoProcedenciaContextoActorV1(
		manifiestoCanon,
	)
	if err != nil {
		return ports.ContextoAutorizacionAltaV3{}, err
	}
	resultado := dominiovec.ResultadoContextoActorRegistradoV2{
		RegistroContextoRef: referenciaAltaContratacionTemporalDesarrollo(
			"rca_", base+"\x00registro-contexto",
		),
		Contexto: actor, RepresentacionCanonica: canon,
		HuellaSHA256:                      huellaContexto,
		ManifiestoProcedenciaCanonico:     manifiestoCanon,
		ManifiestoProcedenciaHuellaSHA256: manifiestoHuella,
		AutoridadEfectiva:                 dominiovec.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
		ResueltoEnAutoritativo:            resueltoEn,
	}
	autenticacion := dominiovec.AutenticacionRevalidadaV1{
		AutenticacionRef: referenciaAltaContratacionTemporalDesarrollo(
			"aut_", base+"\x00autenticacion",
		),
		AutenticacionHuellaSHA256: huellaAltaContratacionTemporalDesarrollo(
			base + "\x00autenticacion",
		),
		AsercionRef: referenciaAltaContratacionTemporalDesarrollo(
			"ase_", base+"\x00asercion",
		),
		SesionRef: referenciaAltaContratacionTemporalDesarrollo(
			"ses_", base+"\x00sesion",
		),
		ControlSesionRef: referenciaAltaContratacionTemporalDesarrollo(
			"cse_", base+"\x00control-sesion",
		),
		ControlSesionRevision: 1,
		ControlSesionHuellaSHA256: huellaAltaContratacionTemporalDesarrollo(
			base + "\x00control-sesion",
		),
		CuentaRef: cuentaRef, CuentaOrdinariaRef: cuentaRef,
		Superficie:        dominiovec.SuperficieAutenticacionInternaCorporativaV1,
		MetodoObservado:   dominiovec.AuthMethodCertificate,
		GarantiaObservada: dominiovec.AuthAssuranceHigh,
		PoliticaGarantiaRef: referenciaAltaContratacionTemporalDesarrollo(
			"pga_", base+"\x00politica-garantia",
		),
		PoliticaGarantiaHuellaSHA256: huellaAltaContratacionTemporalDesarrollo(
			base + "\x00politica-garantia",
		),
		AutenticacionVerificadaEn: desde,
		SesionEmitidaEn:           desde.Add(time.Minute),
		SesionValidaHasta:         hasta,
		SesionRevalidadaEn:        desde.Add(2 * time.Minute),
	}
	solicitudAutenticacion := dominiovec.SolicitudRevalidacionAutenticacionActorV1{
		AutenticacionRef: autenticacion.AutenticacionRef,
		SesionRef:        autenticacion.SesionRef,
	}
	solicitudContexto := dominiovec.SolicitudContextoActor{
		Cuenta: cuenta, PerfilActivoRef: perfilRef,
	}
	vinculo, resultadoClonado, err := dominiovec.CrearVinculoAutenticacionActorV2ConResultado(
		context.Background(),
		revalidadorAutenticacionAltaContratacionTemporalDesarrollo{valor: autenticacion},
		solicitudAutenticacion,
		resolutorContextoAltaContratacionTemporalDesarrollo{valor: resultado},
		solicitudContexto,
		relojFijoAltaContratacionTemporalDesarrollo{ahora: ahora},
	)
	if err != nil {
		return ports.ContextoAutorizacionAltaV3{}, err
	}
	return ports.ContextoAutorizacionAltaV3{
		Vinculo: vinculo, Resultado: resultadoClonado,
	}, nil
}

type revalidadorAutenticacionAltaContratacionTemporalDesarrollo struct {
	valor dominiovec.AutenticacionRevalidadaV1
}

func (r revalidadorAutenticacionAltaContratacionTemporalDesarrollo) RevalidarAutenticacionActorV1(
	ctx context.Context,
	solicitud dominiovec.SolicitudRevalidacionAutenticacionActorV1,
) (dominiovec.AutenticacionRevalidadaV1, error) {
	if ctx == nil || ctx.Err() != nil ||
		solicitud.AutenticacionRef != r.valor.AutenticacionRef ||
		solicitud.SesionRef != r.valor.SesionRef {
		return dominiovec.AutenticacionRevalidadaV1{},
			dominiovec.ErrAutenticacionRevalidadaInvalida
	}
	return r.valor, nil
}

type resolutorContextoAltaContratacionTemporalDesarrollo struct {
	valor dominiovec.ResultadoContextoActorRegistradoV2
}

func (r resolutorContextoAltaContratacionTemporalDesarrollo) ResolverContextoActorRegistradoV2(
	ctx context.Context,
	solicitud dominiovec.SolicitudContextoActor,
) (dominiovec.ResultadoContextoActorRegistradoV2, error) {
	if ctx == nil || ctx.Err() != nil ||
		solicitud.Cuenta.CuentaRef != r.valor.Contexto.Instantanea.CuentaRef ||
		solicitud.PerfilActivoRef != r.valor.Contexto.PerfilActivoRef {
		return dominiovec.ResultadoContextoActorRegistradoV2{},
			dominiovec.ErrVinculoAutenticacionActorV2Invalido
	}
	return r.valor.Clonar()
}

type relojFijoAltaContratacionTemporalDesarrollo struct {
	ahora time.Time
}

func (r relojFijoAltaContratacionTemporalDesarrollo) Ahora() time.Time {
	return r.ahora
}

func referenciaAltaContratacionTemporalDesarrollo(prefijo, material string) string {
	suma := sha256.Sum256([]byte("vec.ct.alta.desarrollo.v1\x00" + material))
	return prefijo + hex.EncodeToString(suma[:16])
}

func huellaAltaContratacionTemporalDesarrollo(material string) string {
	suma := sha256.Sum256([]byte("vec.ct.alta.desarrollo.v1\x00" + material))
	return hex.EncodeToString(suma[:])
}
