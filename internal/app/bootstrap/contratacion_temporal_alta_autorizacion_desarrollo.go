package bootstrap

import (
	"context"
	"time"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

func (s *soporteAltaContratacionTemporalDesarrollo) capacidadAltaValida(
	ctx context.Context,
) bool {
	capacidad, valida := s.capacidadValida(ctx)
	_, desdePeticion := altaDePeticionConfiable(ctx)
	return valida && (capacidad.ruta == httpinterno.RutaAltaSolicitudes || capacidad.ruta == rutaEntregaPeticionCentro && desdePeticion)
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
	principalValido := principalContratacionTemporalDesarrolloValido(capacidad.principal)
	if s.lectorConsultasRRHH {
		principalValido = rutaConsultaRRHHContratacionTemporalDesarrollo(capacidad.ruta) &&
			principalSinteticoContratacionTemporalDesarrolloValido(capacidad.principal)
	}
	if s.peticionesCentro {
		principalValido = rutaPeticionCentroDesarrollo(capacidad.ruta) && principalPeticionCentroDesarrolloValido(capacidad.principal)
	}
	if s.candidatoBolsa {
		principalValido = capacidad.ruta == "/api/vec/bolsa/mi-bolsa" &&
			principalSinteticoContratacionTemporalDesarrolloValido(capacidad.principal) &&
			len(capacidad.principal.Roles) == 1 && capacidad.principal.Roles[0] == "candidato_bolsa"
	}
	if capacidad.ruta == httpinterno.RutaSubsanacionReparos {
		ahora := s.reloj.Ahora()
		if !domain.InstanteUTCCanonico(ahora) ||
			!domain.InstanteUTCCanonico(capacidad.certificadoVerificadoEn) ||
			!domain.InstanteUTCCanonico(capacidad.certificadoValidoHasta) ||
			capacidad.certificadoVerificadoEn.After(ahora) ||
			!ahora.Before(capacidad.certificadoValidoHasta) {
			principalValido = false
		}
	}
	valida := existe && capacidad.sello == s.sello && principalValido &&
		capacidad.principal.ID == s.principalID &&
		capacidad.principal.Attributes["certificate_sha256"] == s.certificadoSHA256
	return capacidad, valida
}

func (s *soporteAltaContratacionTemporalDesarrollo) capacidadSubsanacionVigente(ctx context.Context) bool {
	capacidad, valida := s.capacidadValida(ctx)
	return valida && capacidad.ruta == httpinterno.RutaSubsanacionReparos
}

func rutaContextoAutorizacionContratacionTemporalDesarrollo(ruta string) bool {
	return ruta == rutaEntregaPeticionCentro || rutaPeticionCentroDesarrollo(ruta) || ruta == rutaCambiosOrganizacionContratacionTemporalDesarrollo ||
		ruta == httpinterno.RutaAltaSolicitudes ||
		ruta == httpinterno.RutaPropuestaCobertura ||
		ruta == httpinterno.RutaDecisionCobertura ||
		ruta == httpinterno.RutaRectificacionCobertura ||
		rutaAnalisisContratacionTemporalDesarrollo(ruta) ||
		rutaAsignacionContratacionTemporalDesarrollo(ruta) ||
		rutaInformeJuridicoContratacionTemporalDesarrollo(ruta) ||
		ruta == httpinterno.RutaSubsanacionReparos ||
		rutaFirmaDocumentoCTDesarrollo(ruta) ||
		rutaSeguimientoCeseDesarrollo(ruta) || rutaCancelacionCTDesarrollo(ruta) ||
		rutaLlamamientoContratacionTemporalDesarrollo(ruta) ||
		rutaConsultaRRHHContratacionTemporalDesarrollo(ruta)

}

func rutaAnalisisContratacionTemporalDesarrollo(ruta string) bool {
	return ruta == httpinterno.RutaRegistroAnalisisRRHH || ruta == httpinterno.RutaRectificacionAnalisisRRHH

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
	operativo, err := s.contextoOperativoDesarrollo(ctx)
	if err != nil {
		return application.SolicitudRegistrarExpediente{}, ports.ErrAutorizacionDenegada
	}
	vinculo, err := operativo.Vinculo.Datos()
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
	operativo, err := s.contextoOperativoDesarrollo(ctx)
	if err != nil {
		return httpinterno.ContextoCanalAnalisisRRHH{}, ports.ErrAutorizacionDenegada
	}
	vinculo, err := operativo.Vinculo.Datos()
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
	operativo, err := s.contextoOperativoDesarrollo(ctx)
	if err != nil {
		return httpinterno.ContextoCanalAsignacion{}, ports.ErrAutorizacionDenegada
	}
	vinculo, err := operativo.Vinculo.Datos()
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
	operativo, err := s.contextoOperativoDesarrollo(ctx)
	if err != nil {
		return httpinterno.ContextoCanalInformeJuridico{}, ports.ErrAutorizacionDenegada
	}
	vinculo, err := operativo.Vinculo.Datos()
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
	operativo, err := s.contextoOperativoDesarrollo(ctx)
	if err != nil {
		return ports.ContextoAutorizacionAltaV3{}, ports.ErrAutorizacionDenegada
	}
	vinculo, err := operativo.Vinculo.Datos()
	if err != nil || solicitud.AutenticacionRef != vinculo.AutenticacionRef ||
		solicitud.SesionRef != vinculo.SesionRef ||
		solicitud.PerfilRef != vinculo.PerfilActivoRef {
		return ports.ContextoAutorizacionAltaV3{}, ports.ErrAutorizacionDenegada
	}
	return operativo, nil
}

func (s *soporteAltaContratacionTemporalDesarrollo) centroDeCatalogo(ref string) bool {
	if s == nil || s.origen == nil {
		return ref == centroAltaContratacionTemporalDesarrollo
	}
	return s.origen.centroDeCatalogo(ref)
}

func (s *soporteAltaContratacionTemporalDesarrollo) categoriaDeCatalogo(ref string) bool {
	if s == nil || s.origen == nil {
		return ref == categoriaAltaContratacionTemporalDesarrollo
	}
	return s.origen.categoriaDeCatalogo(ref)
}

func (s *soporteAltaContratacionTemporalDesarrollo) ResolverFlujoAlta(
	ctx context.Context,
	solicitud ports.SolicitudResolverFlujo,
) (ports.ConfiguracionAltaFlujo, error) {
	centroValido := s.centroDeCatalogo(solicitud.CentroRef)
	if e, ok := altaDePeticionConfiable(ctx); ok {
		centroValido = solicitud.CentroRef == e.Peticion.Solicitud.CentroRef && centroValido
	}
	if !s.capacidadAltaValida(ctx) || solicitud.Validar() != nil ||
		solicitud.OrganizacionRef != organizacionAltaContratacionTemporalDesarrollo ||
		!centroValido ||
		!s.categoriaDeCatalogo(solicitud.CategoriaRef) ||
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
	esperada, motivoValido := s.motivoAutorizacionParaContexto(ctx, capacidad.ruta)
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
	esperada, motivoValido := s.motivoAutorizacionParaContexto(ctx, capacidad.ruta)
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
	esperada, motivoValido := s.motivoAutorizacionParaContexto(ctx, capacidad.ruta)
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
	if s.peticionesCentro && rutaPeticionCentroDesarrollo(ruta) {
		return motivoPeticionCentroDesarrollo(), true
	}
	switch ruta {
	case rutaEntregaPeticionCentro:
		return motivoEntregaPeticionDesarrollo(), true
	case rutaCambiosOrganizacionContratacionTemporalDesarrollo:
		return motivoOrganizacionDesarrollo(), true
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
	case httpinterno.RutaRectificacionAnalisisRRHH:
		return s.motivoRectificacionAnalisis, dominiovec.ReferenciaMotivoAutorizacionV2Valida(s.motivoRectificacionAnalisis)
	case httpinterno.RutaResultadoCobertura:
		return s.motivoResultadoCobertura, true
	case httpinterno.RutaAsignaciones:
		return s.motivoAsignacion, true
	case httpinterno.RutaPreparacionesInformeJuridico:
		return s.motivoInformeJuridico, true
	case httpinterno.RutaSeleccionLlamamiento:
		return s.motivoLlamamiento, true
	case httpinterno.RutaRegistroComunicacionLlamamiento:
		return s.motivoComunicacion, true
	case httpinterno.RutaResolucionComunicacionLlamamiento:
		return s.motivoConsultaJustificante, dominiovec.ReferenciaMotivoAutorizacionV2Valida(s.motivoConsultaJustificante)
	case httpinterno.RutaContinuacionLlamamiento:
		return motivoContinuacionDesarrollo(false), true
	case httpinterno.RutaPropuestaFormalizacion:
		return motivoPropuestaFormalizacionDesarrollo(), true
	case httpinterno.RutaResolucionFormalizacion:
		return motivoResolucionFormalizacionDesarrollo(), true
	case httpinterno.RutaRegistroRespuestaRecibida:
		return s.motivoRespuestaRecibida, dominiovec.ReferenciaMotivoAutorizacionV2Valida(s.motivoRespuestaRecibida)
	case httpinterno.RutaEventoPlazoLlamamiento:
		return motivoResolucionManualDesarrollo(false), true
	case httpinterno.RutaSubsanacionReparos:
		return s.motivoSubsanacion, dominiovec.ReferenciaMotivoAutorizacionV2Valida(s.motivoSubsanacion)
	case httpinterno.RutaFirmaDocumento:
		return s.motivoFirmaDocumento, dominiovec.ReferenciaMotivoAutorizacionV2Valida(s.motivoFirmaDocumento)
	case httpinterno.RutaCesesNombramiento, httpinterno.RutaCierresExpediente, httpinterno.RutaModificacionesNombramiento, httpinterno.RutaSeguimientoCese,
		httpinterno.RutaConfirmacionesGINPIX:
		return motivoSeguimientoCeseDesarrollo(ruta), s.seguimientoCese != nil
	case httpinterno.RutaCancelacionesExpediente, httpinterno.RutaCancelacionExpediente:
		return motivoCancelacionCTDesarrollo(ruta), s.cancelacion != nil
	default:
		return dominiovec.ReferenciaEntradaCatalogo{}, false
	}
}

func (s *soporteAltaContratacionTemporalDesarrollo) instantaneaParaRuta(
	ruta string,
) (dominiovec.InstantaneaAutorizacion, bool) {
	if s != nil && s.peticionesCentro && rutaPeticionCentroDesarrollo(ruta) {
		return clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(s.instantanea), s.instantanea.Validar() == nil
	}
	if s == nil {
		return dominiovec.InstantaneaAutorizacion{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if ruta == rutaEntregaPeticionCentro {
		return clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(s.instantaneaEntregaPeticion), s.instantaneaEntregaPeticion.Validar() == nil
	}
	if ruta == rutaCambiosOrganizacionContratacionTemporalDesarrollo {
		return clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(s.instantaneaOrganizacion), s.instantaneaOrganizacion.Validar() == nil
	}
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
	if ruta == httpinterno.RutaContinuacionLlamamiento {
		return clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(s.instantaneaContinuacionCT), s.instantaneaContinuacionCT.Validar() == nil
	}
	if ruta == httpinterno.RutaPropuestaFormalizacion {
		return clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(s.instantaneaPropuestaFormalizacion), s.instantaneaPropuestaFormalizacion.Validar() == nil
	}
	if ruta == httpinterno.RutaResolucionFormalizacion {
		return clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(s.instantaneaResolucionFormalizacion), s.instantaneaResolucionFormalizacion.Validar() == nil
	}
	if ruta == httpinterno.RutaRegistroRespuestaRecibida {
		return clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(s.instantaneaRespuestaRecibida), s.instantaneaRespuestaRecibida.Validar() == nil
	}
	if ruta == httpinterno.RutaEventoPlazoLlamamiento {
		return clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(s.instantaneaResolucionManual), s.instantaneaResolucionManual.Validar() == nil
	}
	if ruta == httpinterno.RutaSubsanacionReparos {
		return clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(s.instantaneaSubsanacion), s.instantaneaSubsanacion.Validar() == nil
	}
	if ruta == httpinterno.RutaFirmaDocumento {
		return clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(s.instantaneaFirmaDocumento), s.instantaneaFirmaDocumento.Validar() == nil
	}
	if rutaSeguimientoCeseDesarrollo(ruta) {
		return s.instantaneaSeguimientoCese()
	}
	if rutaCancelacionCTDesarrollo(ruta) {
		return s.instantaneaCancelacionCT()
	}
	if ruta == httpinterno.RutaResolucionComunicacionLlamamiento {
		return clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(s.instantaneaConsultaJustificante), s.instantaneaConsultaJustificante.Validar() == nil
	}
	if rutaLlamamientoContratacionTemporalDesarrollo(ruta) {
		return clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(s.instantaneaComunicacion), true
	}
	return dominiovec.InstantaneaAutorizacion{}, false
}
