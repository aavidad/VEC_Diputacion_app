package application

import (
	"context"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func (s *ServicioConfirmacionDecisionCobertura) ejecutarValidada(
	ctx context.Context,
	solicitud datosSolicitudConfirmacionDecisionCobertura,
) (cobertura.ReciboOperacionDecisionCobertura, error) {
	solicitudContexto := ports.SolicitudResolverContextoAutorizacionAltaV3{
		AutenticacionRef: solicitud.autenticacionRef,
		SesionRef:        solicitud.sesionRef, PerfilRef: solicitud.perfilRef,
	}
	contexto, err := s.contextos.ResolverContextoAutorizacionAltaV3(
		ctx,
		solicitudContexto,
	)
	if err != nil {
		return cobertura.ReciboOperacionDecisionCobertura{},
			s.errorContexto(ctx, err)
	}
	instanteContexto, err := s.ahoraConfirmacion(ctx)
	if err != nil ||
		contexto.ValidarPara(solicitudContexto, instanteContexto) != nil {
		return cobertura.ReciboOperacionDecisionCobertura{},
			s.errorDenegacion(ctx)
	}
	motivoInicial, motivoNominal, err := s.resolverMotivoConfirmacion(
		ctx,
		solicitud.motivoClave,
		instanteContexto,
	)
	if err != nil {
		return cobertura.ReciboOperacionDecisionCobertura{}, err
	}
	identidad, err := cobertura.NuevaIdentidadOperacionDecisionCobertura(
		solicitud.claveIdempotencia,
		solicitud.tipo,
		solicitud.organizacionRef,
		solicitud.expedienteRef,
		solicitud.versionEsperada,
		contexto,
		solicitudContexto,
		instanteContexto,
		accionConfirmacionDecisionCobertura(solicitud.tipo),
		solicitud.viaElegida,
		solicitud.identidadSemantica,
		motivoNominal,
		solicitud.predecesoraRef,
		solicitud.predecesoraHuella,
	)
	if err != nil {
		return cobertura.ReciboOperacionDecisionCobertura{},
			ErrSolicitudConfirmacionDecisionCoberturaInvalida
	}
	preimagenes, err :=
		cobertura.NuevasPreimagenesOperacionDecisionCobertura(identidad)
	if err != nil {
		return cobertura.ReciboOperacionDecisionCobertura{},
			ErrConfirmacionDecisionCoberturaNoConfiable
	}
	sellos, err := s.sellador.SellarOperacionDecisionCobertura(
		ctx,
		preimagenes,
	)
	if err != nil {
		return cobertura.ReciboOperacionDecisionCobertura{},
			s.errorDependencia(ctx, err)
	}
	if sellos.Validar() != nil {
		return cobertura.ReciboOperacionDecisionCobertura{},
			ErrConfirmacionDecisionCoberturaNoConfiable
	}
	consulta, err :=
		cobertura.NuevaSolicitudConsultarOperacionDecisionCoberturaConfirmada(
			identidad,
			sellos,
		)
	if err != nil {
		return cobertura.ReciboOperacionDecisionCobertura{},
			ErrConfirmacionDecisionCoberturaNoConfiable
	}
	replay, encontrado, err :=
		s.idempotencia.ConsultarOperacionDecisionCoberturaConfirmada(
			ctx,
			consulta,
		)
	if err != nil {
		return cobertura.ReciboOperacionDecisionCobertura{},
			s.errorPersistencia(ctx, err)
	}
	if encontrado {
		recibo, err := replay.ReciboConfirmadoPara(consulta)
		if err != nil {
			return cobertura.ReciboOperacionDecisionCobertura{},
				ErrConfirmacionDecisionCoberturaNoConfiable
		}
		return recibo, nil
	}
	token, err :=
		cobertura.GenerarTokenPropietarioOperacionDecisionCobertura()
	if err != nil {
		return cobertura.ReciboOperacionDecisionCobertura{},
			ErrConfirmacionDecisionCoberturaNoDisponible
	}
	solicitudReserva, err :=
		cobertura.NuevaSolicitudReservarOperacionDecisionCobertura(
			consulta,
			token,
		)
	if err != nil {
		return cobertura.ReciboOperacionDecisionCobertura{},
			ErrConfirmacionDecisionCoberturaNoConfiable
	}
	preparacionReserva, err :=
		s.idempotencia.ReservarOReapropiarOperacionDecisionCobertura(
			ctx,
			solicitudReserva,
		)
	if err != nil {
		return cobertura.ReciboOperacionDecisionCobertura{},
			s.errorPersistencia(ctx, err)
	}
	estado, err := preparacionReserva.EstadoPara(solicitudReserva)
	if err != nil {
		return cobertura.ReciboOperacionDecisionCobertura{},
			ErrConfirmacionDecisionCoberturaNoConfiable
	}
	switch estado {
	case cobertura.PreparacionOperacionDecisionCoberturaConfirmada:
		recibo, err := preparacionReserva.ReciboConfirmadoPara(consulta)
		if err != nil {
			return cobertura.ReciboOperacionDecisionCobertura{},
				ErrConfirmacionDecisionCoberturaNoConfiable
		}
		return recibo, nil
	case cobertura.PreparacionOperacionDecisionCoberturaOcupada:
		return cobertura.ReciboOperacionDecisionCobertura{},
			ErrConfirmacionDecisionCoberturaOcupada
	case cobertura.PreparacionOperacionDecisionCoberturaPropietaria:
		return s.ejecutarComoPropietario(
			ctx,
			solicitud,
			solicitudContexto,
			contexto,
			motivoInicial,
			solicitudReserva,
			preparacionReserva,
		)
	default:
		return cobertura.ReciboOperacionDecisionCobertura{},
			ErrConfirmacionDecisionCoberturaNoConfiable
	}
}

func (s *ServicioConfirmacionDecisionCobertura) ejecutarComoPropietario(
	ctx context.Context,
	solicitud datosSolicitudConfirmacionDecisionCobertura,
	solicitudContexto ports.SolicitudResolverContextoAutorizacionAltaV3,
	contexto ports.ContextoAutorizacionAltaV3,
	motivoInicial cobertura.ResolucionMotivoDecisionCobertura,
	solicitudReserva cobertura.SolicitudReservarOperacionDecisionCobertura,
	preparacionReserva cobertura.PreparacionOperacionDecisionCobertura,
) (cobertura.ReciboOperacionDecisionCobertura, error) {
	datosReserva, err :=
		preparacionReserva.DatosPropietariaPara(solicitudReserva)
	if err != nil {
		return cobertura.ReciboOperacionDecisionCobertura{},
			ErrConfirmacionDecisionCoberturaNoConfiable
	}
	solicitudAnalisis, err :=
		cobertura.NuevaSolicitudInstantaneaAnalisisDurableO3(
			solicitud.organizacionRef,
			solicitud.expedienteRef,
			solicitud.versionEsperada,
		)
	if err != nil {
		return cobertura.ReciboOperacionDecisionCobertura{},
			ErrSolicitudConfirmacionDecisionCoberturaInvalida
	}
	instantanea, err := cobertura.ObtenerInstantaneaAnalisisDurableO3(
		ctx,
		s.analisis,
		solicitudAnalisis,
	)
	if err != nil {
		return cobertura.ReciboOperacionDecisionCobertura{},
			s.errorDependencia(ctx, err)
	}
	expediente, analisisRef, analisisHuella, err :=
		instantanea.DesplegarPara(solicitudAnalisis)
	if err != nil {
		return cobertura.ReciboOperacionDecisionCobertura{},
			ErrConfirmacionDecisionCoberturaNoConfiable
	}
	solicitudGobierno, err :=
		solicitudGobiernoConfirmacionCobertura(expediente, solicitud.tipo)
	if err != nil {
		return cobertura.ReciboOperacionDecisionCobertura{}, err
	}
	gobierno, err := cobertura.ObtenerGobiernoOperacionCobertura(
		ctx,
		s.reloj,
		s.gobierno,
		solicitudGobierno,
	)
	if err != nil {
		return cobertura.ReciboOperacionDecisionCobertura{},
			s.errorDependencia(ctx, err)
	}
	datosGobierno, err := gobierno.DesplegarPara(
		ctx,
		s.reloj,
		solicitudGobierno,
	)
	if err != nil {
		return cobertura.ReciboOperacionDecisionCobertura{},
			s.errorDependencia(ctx, err)
	}
	if expediente.Analisis == nil {
		return cobertura.ReciboOperacionDecisionCobertura{},
			ErrConfirmacionDecisionCoberturaNoConfiable
	}
	datosGlobales, err := nuevosDatosPreparacionGlobalCobertura(
		analisisRef,
		analisisHuella,
		datosGobierno.Catalogo,
		datosGobierno.Politica,
		expediente.OrganizacionRef,
		expediente.Referencia,
		expediente.Version,
		expediente.Analisis.CategoriaRef,
		expediente.Analisis.Periodo,
	)
	if err != nil {
		return cobertura.ReciboOperacionDecisionCobertura{},
			ErrConfirmacionDecisionCoberturaNoConfiable
	}
	preparacionC1, err := s.coberturas.Preparar(ctx, datosGlobales)
	if err != nil {
		return cobertura.ReciboOperacionDecisionCobertura{},
			s.errorDependencia(ctx, err)
	}
	instantePropuesta, err := s.ahoraConfirmacion(ctx)
	if err != nil {
		return cobertura.ReciboOperacionDecisionCobertura{}, err
	}
	datosPropuesta, err := preparacionC1.DatosCrearPropuestaEn(
		instantePropuesta,
	)
	if err != nil {
		return cobertura.ReciboOperacionDecisionCobertura{},
			ErrConfirmacionDecisionCoberturaNoConfiable
	}
	propuesta, err := domain.CrearPropuestaDecisionCobertura(datosPropuesta)
	if err != nil {
		return cobertura.ReciboOperacionDecisionCobertura{},
			ErrConfirmacionDecisionCoberturaNoConfiable
	}
	semanticaFresca, err := propuesta.IdentidadSemantica()
	if err != nil ||
		!semanticaFresca.CoincideExactamente(
			solicitud.identidadSemantica,
		) {
		return cobertura.ReciboOperacionDecisionCobertura{},
			ErrConfirmacionDecisionCoberturaEnConflicto
	}
	motivoFinal, motivoNominalFinal, err := s.resolverMotivoConfirmacion(
		ctx,
		solicitud.motivoClave,
		instantePropuesta,
	)
	if err != nil {
		return cobertura.ReciboOperacionDecisionCobertura{}, err
	}
	if !motivosConfirmacionCoberturaIguales(
		motivoInicial,
		motivoFinal,
	) {
		return cobertura.ReciboOperacionDecisionCobertura{},
			ErrConfirmacionDecisionCoberturaEnConflicto
	}
	_ = motivoNominalFinal
	preparacionOrden, err :=
		cobertura.PrepararOrdenOperacionDecisionCobertura(
			ctx,
			s.reloj,
			solicitudReserva,
			preparacionReserva,
			solicitudGobierno,
			gobierno,
			preparacionC1,
			propuesta,
			motivoFinal,
		)
	if err != nil {
		return cobertura.ReciboOperacionDecisionCobertura{},
			ErrConfirmacionDecisionCoberturaEnConflicto
	}
	recurso, err := preparacionOrden.RecursoAutorizableVEC()
	if err != nil {
		return cobertura.ReciboOperacionDecisionCobertura{},
			ErrConfirmacionDecisionCoberturaNoConfiable
	}
	instanteVEC, err := s.ahoraConfirmacion(ctx)
	if err != nil ||
		contexto.ValidarPara(solicitudContexto, instanteVEC) != nil {
		return cobertura.ReciboOperacionDecisionCobertura{},
			s.errorDenegacion(ctx)
	}
	solicitudVEC, err := nuevaSolicitudVECConfirmacionCobertura(
		ctx,
		contexto,
		datosReserva,
		datosGobierno,
		recurso,
	)
	if err != nil {
		return cobertura.ReciboOperacionDecisionCobertura{},
			s.errorDependencia(ctx, err)
	}
	generadorDecision, err :=
		nuevoGeneradorDecisionVECReservada(datosReserva.DecisionVECRef)
	if err != nil {
		return cobertura.ReciboOperacionDecisionCobertura{},
			ErrConfirmacionDecisionCoberturaNoConfiable
	}
	_, candidata, err :=
		s.autorizaciones.PrepararRegistroCompuestoSolicitudLigadaV3(
			ctx,
			solicitudVEC,
			contexto.Resultado,
			generadorDecision,
		)
	if err != nil {
		return cobertura.ReciboOperacionDecisionCobertura{},
			s.errorDependencia(ctx, err)
	}
	orden, err := cobertura.NuevaOrdenOperacionDecisionCobertura(
		ctx,
		s.reloj,
		preparacionOrden,
		candidata,
	)
	if err != nil {
		return cobertura.ReciboOperacionDecisionCobertura{},
			ErrConfirmacionDecisionCoberturaNoConfiable
	}
	return s.confirmarOrden(ctx, orden)
}
