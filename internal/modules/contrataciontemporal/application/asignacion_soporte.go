package application

import (
	"context"
	"errors"
	"strconv"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func (s *ServicioAsignacion) dependenciasValidas() bool {
	return s != nil && !dependenciaNula(s.contextos) &&
		!dependenciaNula(s.ambitos) && !dependenciaNula(s.huellas) &&
		!dependenciaNula(s.consultas) &&
		!dependenciaNula(s.preparaciones) &&
		!dependenciaNula(s.destinos) && !dependenciaNula(s.politicas) &&
		!dependenciaNula(s.correlaciones) &&
		!dependenciaNula(s.autorizador) && !dependenciaNula(s.reloj) &&
		!dependenciaNula(s.transaccion)
}

func (s datosSolicitudAsignacion) validar(instante time.Time) error {
	contexto := ports.SolicitudResolverContextoAutorizacionAltaV3{
		AutenticacionRef: s.autenticacionRef,
		SesionRef:        s.sesionRef,
		PerfilRef:        s.perfilRef,
	}
	if !s.operacion.Valida() || contexto.Validar() != nil ||
		!domain.ReferenciaOpacaValida(s.organizacionRef) ||
		!domain.ReferenciaOpacaValida(s.expedienteRef) ||
		!ports.VersionOperacionAnalisisConIncrementoValida(
			s.versionEsperada,
		) ||
		!ports.ClaveIdempotenciaValida(s.claveIdempotencia) ||
		!domain.ReferenciaOpacaValida(s.unidadRef) ||
		!domain.ReferenciaOpacaValida(s.responsableRef) ||
		!domain.InstanteUTCCanonico(instante) {
		return ErrSolicitudAsignacionInvalida
	}
	material := s.material(
		"persona:validacion:0123456789abcdef",
		"perfil:validacion:0123456789abcdef",
	)
	return material.Validar()
}

func (s datosSolicitudAsignacion) material(
	actorRef string,
	perfilRef string,
) ports.MaterialHuellaAsignacion {
	return ports.MaterialHuellaAsignacion{
		Operacion:               s.operacion,
		OrganizacionRef:         s.organizacionRef,
		ExpedienteRef:           s.expedienteRef,
		VersionExpediente:       s.versionEsperada,
		ActorRef:                actorRef,
		PerfilRef:               perfilRef,
		UnidadRef:               s.unidadRef,
		ResponsableRef:          s.responsableRef,
		MotivoReasignacionClave: s.motivoReasignacion,
		Observaciones:           s.observaciones,
	}
}

func solicitudPrepararAsignacion(
	solicitud datosSolicitudAsignacion,
	material ports.MaterialHuellaAsignacion,
	ambitos ports.ColeccionSellosHMAC,
	huellas ports.ColeccionSellosHMAC,
) ports.SolicitudPrepararAsignacion {
	return ports.SolicitudPrepararAsignacion{
		ClaveIdempotencia:   solicitud.claveIdempotencia,
		AmbitosHMAC:         ambitos,
		HuellasPeticionHMAC: huellas,
		Operacion:           material.Operacion,
		OrganizacionRef:     material.OrganizacionRef,
		ExpedienteRef:       material.ExpedienteRef,
		VersionExpediente:   material.VersionExpediente,
		ActorRef:            material.ActorRef,
		PerfilRef:           material.PerfilRef,
		UnidadRef:           material.UnidadRef,
		ResponsableRef:      material.ResponsableRef,
	}
}

func (s *ServicioAsignacion) confirmar(
	ctx context.Context,
	solicitud datosSolicitudAsignacion,
	material ports.MaterialHuellaAsignacion,
	contextoSolicitud ports.SolicitudResolverContextoAutorizacionAltaV3,
	contexto ports.ContextoAutorizacionAltaV3,
	preparar ports.SolicitudPrepararAsignacion,
	preparacion ports.PreparacionAsignacion,
) (ports.ReciboAsignacion, error) {
	vista, err := ports.NuevaVistaAutoridadesAsignacion(
		preparacion.Expediente,
	)
	if err != nil {
		return ports.ReciboAsignacion{}, ErrResultadoAsignacionNoConfiable
	}
	evidencia, err := s.resolverAutoridadesAsignacion(
		ctx,
		solicitud,
		material,
		contextoSolicitud,
		contexto,
		preparar,
		vista,
	)
	if err != nil {
		return ports.ReciboAsignacion{}, err
	}
	siguiente, err := aplicarAsignacion(
		material,
		preparacion,
		evidencia.Politica,
		evidencia.Instante,
	)
	if err != nil {
		return ports.ReciboAsignacion{}, ErrSolicitudAsignacionInvalida
	}
	orden, err := ports.NuevaOrdenConfirmarAsignacion(
		ports.DatosOrdenConfirmarAsignacion{
			SolicitudContexto:    evidencia.SolicitudContexto,
			ContextoAutorizacion: evidencia.ContextoAutorizacion,
			Material:             evidencia.Material,
			SolicitudPreparacion: evidencia.SolicitudPreparacion,
			Preparacion:          preparacion,
			SolicitudDestino:     evidencia.SolicitudDestino,
			Destino:              evidencia.Destino,
			SolicitudPolitica:    evidencia.SolicitudPolitica,
			Politica:             evidencia.Politica,
			SolicitudV3:          evidencia.SolicitudV3,
			DecisionV3:           evidencia.DecisionV3,
			ConfirmacionV3:       evidencia.ConfirmacionV3,
			InstanteEfecto:       evidencia.Instante,
			ExpedienteSiguiente:  siguiente,
		},
	)
	if err != nil {
		return ports.ReciboAsignacion{}, ErrResultadoAsignacionNoConfiable
	}
	recibo, err := s.transaccion.ConfirmarAsignacion(ctx, orden)
	if err != nil {
		return ports.ReciboAsignacion{},
			clasificarFalloAsignacion(ctx, err)
	}
	if recibo.ValidarParaOrden(orden) != nil {
		return ports.ReciboAsignacion{},
			ErrResultadoAsignacionNoConfiable
	}
	return recibo, nil
}

func (s *ServicioAsignacion) reconciliar(
	ctx context.Context,
	solicitud datosSolicitudAsignacion,
	material ports.MaterialHuellaAsignacion,
	contextoSolicitud ports.SolicitudResolverContextoAutorizacionAltaV3,
	contexto ports.ContextoAutorizacionAltaV3,
	preparar ports.SolicitudPrepararAsignacion,
	vista ports.VistaAutoridadesAsignacion,
	estado ports.EstadoCandidatoAsignacionIdempotente,
) (ports.ReciboAsignacion, error) {
	evidencia, err := s.resolverAutoridadesAsignacion(
		ctx,
		solicitud,
		material,
		contextoSolicitud,
		contexto,
		preparar,
		vista,
	)
	if err != nil {
		return ports.ReciboAsignacion{}, err
	}
	recibo, err := estado.Reconciliar(evidencia)
	if err != nil {
		return ports.ReciboAsignacion{},
			ErrResultadoAsignacionNoConfiable
	}
	return recibo, nil
}

func (s *ServicioAsignacion) resolverAutoridadesAsignacion(
	ctx context.Context,
	solicitud datosSolicitudAsignacion,
	material ports.MaterialHuellaAsignacion,
	contextoSolicitud ports.SolicitudResolverContextoAutorizacionAltaV3,
	contexto ports.ContextoAutorizacionAltaV3,
	preparar ports.SolicitudPrepararAsignacion,
	vista ports.VistaAutoridadesAsignacion,
) (ports.EvidenciaReconciliacionAsignacion, error) {
	vacia := ports.EvidenciaReconciliacionAsignacion{}
	if err := ctx.Err(); err != nil {
		return vacia, err
	}
	consulta, err := ports.NuevaSolicitudConsultarAsignacionIdempotente(
		preparar,
	)
	if err != nil || vista.ValidarPara(consulta) != nil {
		return vacia, ErrResultadoAsignacionNoConfiable
	}
	instanteResolucion := instanteCanonico(s.reloj.Ahora())
	destinoSolicitud := ports.SolicitudResolverDestinoAsignacion{
		OrganizacionRef:   material.OrganizacionRef,
		ExpedienteRef:     material.ExpedienteRef,
		VersionExpediente: material.VersionExpediente,
		ActorRef:          material.ActorRef,
		UnidadRef:         material.UnidadRef,
		ResponsableRef:    material.ResponsableRef,
		Instante:          instanteResolucion,
	}
	destino, err := s.destinos.ResolverDestinoAsignacion(
		ctx,
		destinoSolicitud,
	)
	if err != nil || destino.ValidarPara(
		destinoSolicitud,
		instanteResolucion,
	) != nil {
		return vacia,
			clasificarFalloAsignacion(ctx, ErrAsignacionDenegada)
	}
	if err := ctx.Err(); err != nil {
		return vacia, err
	}
	politicaSolicitud := solicitudPoliticaAsignacion(
		solicitud,
		vista,
		material,
		destino,
		instanteResolucion,
	)
	politica, err := s.politicas.ResolverPoliticaAsignacion(
		ctx,
		politicaSolicitud,
	)
	if err != nil || politica.ValidarPara(
		politicaSolicitud,
		instanteResolucion,
	) != nil {
		return vacia,
			clasificarFalloAsignacion(ctx, ErrAsignacionDenegada)
	}
	if err := ctx.Err(); err != nil {
		return vacia, err
	}
	solicitudV3, err := s.nuevaSolicitudAutorizacionAsignacion(
		ctx,
		contexto,
		preparar,
		vista,
		material,
		destino,
		politica,
	)
	if err != nil {
		return vacia, ErrAsignacionDenegada
	}
	decisionV3, confirmacionV3, err := s.autorizador.ExigirSolicitudLigadaV3(
		ctx,
		solicitudV3,
		contexto.Resultado,
	)
	if err != nil {
		return vacia,
			clasificarFalloAsignacion(ctx, ErrAsignacionDenegada)
	}
	if err := ctx.Err(); err != nil {
		return vacia, err
	}
	instanteEfecto := instanteCanonico(s.reloj.Ahora())
	if contexto.ValidarPara(contextoSolicitud, instanteEfecto) != nil ||
		destino.ValidarPara(destinoSolicitud, instanteEfecto) != nil ||
		politica.ValidarPara(politicaSolicitud, instanteEfecto) != nil ||
		!autorizacionV3ValidaEn(
			solicitudV3,
			decisionV3,
			confirmacionV3,
			instanteEfecto,
		) {
		return vacia, ErrAsignacionDenegada
	}
	return ports.EvidenciaReconciliacionAsignacion{
		SolicitudContexto:    contextoSolicitud,
		ContextoAutorizacion: contexto,
		Material:             material,
		SolicitudPreparacion: preparar,
		SolicitudDestino:     destinoSolicitud,
		Destino:              destino,
		SolicitudPolitica:    politicaSolicitud,
		Politica:             politica,
		SolicitudV3:          solicitudV3,
		DecisionV3:           decisionV3,
		ConfirmacionV3:       confirmacionV3,
		Instante:             instanteEfecto,
	}, nil
}

func solicitudPoliticaAsignacion(
	solicitud datosSolicitudAsignacion,
	vista ports.VistaAutoridadesAsignacion,
	material ports.MaterialHuellaAsignacion,
	destino ports.DestinoAsignacionResuelto,
	instante time.Time,
) ports.SolicitudResolverPoliticaAsignacion {
	resultado := ports.SolicitudResolverPoliticaAsignacion{
		Operacion:               material.Operacion,
		OrganizacionRef:         material.OrganizacionRef,
		ExpedienteRef:           material.ExpedienteRef,
		VersionExpediente:       material.VersionExpediente,
		Flujo:                   vista.Flujo,
		FasePrevia:              vista.FaseActual,
		EstadoPrevio:            vista.EstadoActual,
		ActorRef:                material.ActorRef,
		PerfilRef:               material.PerfilRef,
		Destino:                 destino,
		MotivoReasignacionClave: solicitud.motivoReasignacion,
		Instante:                instante,
	}
	if vista.TieneAsignacion {
		resultado.UnidadAnteriorRef = vista.UnidadAnteriorRef
		resultado.ResponsableAnteriorRef = vista.ResponsableAnteriorRef
	}
	return resultado
}

func (s *ServicioAsignacion) nuevaSolicitudAutorizacionAsignacion(
	ctx context.Context,
	contexto ports.ContextoAutorizacionAltaV3,
	preparar ports.SolicitudPrepararAsignacion,
	vista ports.VistaAutoridadesAsignacion,
	material ports.MaterialHuellaAsignacion,
	destino ports.DestinoAsignacionResuelto,
	politica ports.PoliticaAsignacion,
) (dominiovec.SolicitudAutorizacionLigadaV3, error) {
	ambitoActivo, huellaActiva, err := ports.ParActivoColeccionesHMAC(
		preparar.AmbitosHMAC,
		ports.DominioAmbitoIdempotenciaAsignacion,
		preparar.HuellasPeticionHMAC,
		ports.DominioHuellaPeticionAsignacion,
	)
	if err != nil {
		return dominiovec.SolicitudAutorizacionLigadaV3{}, err
	}
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(
		ctx,
		s.correlaciones,
	)
	if err != nil {
		return dominiovec.SolicitudAutorizacionLigadaV3{}, err
	}
	return dominiovec.NuevaSolicitudAutorizacionLigadaV3(
		dominiovec.DatosSolicitudAutorizacionLigadaV3{
			VinculoAutenticacionActor: contexto.Vinculo,
			ReferenciaMotivo:          politica.MotivoAutorizacion,
			Accion:                    string(politica.Accion),
			Recurso: dominiovec.RecursoAutorizable{
				Referencia: material.ExpedienteRef,
				ModuloID:   ports.ModuloContratacion,
				Tipo:       ports.TipoRecursoAsignacion,
				Ambitos: map[string]string{
					"organizacion_ref":   material.OrganizacionRef,
					"expediente_ref":     material.ExpedienteRef,
					"fase_previa":        string(vista.FaseActual),
					"estado_previo":      string(vista.EstadoActual),
					"unidad_destino_ref": material.UnidadRef,
				},
				Atributos: map[string]string{
					ports.AtributoOperacionAsignacion: string(material.Operacion),
					ports.AtributoVersionAsignacion: strconv.FormatUint(
						material.VersionExpediente,
						10,
					),
					ports.AtributoPoliticaAsignacionRef: politica.DefinicionRef,
					ports.AtributoPoliticaAsignacionVersion: strconv.FormatUint(
						politica.DefinicionVersion,
						10,
					),
					ports.AtributoPoliticaAsignacionHuella: politica.DefinicionHuellaSHA256,
					ports.AtributoEvidenciaDestinoRef:      destino.EvidenciaRef,
					ports.AtributoEvidenciaDestinoHuella:   destino.EvidenciaHuellaSHA256,
					ports.AtributoUnidadDestino:            material.UnidadRef,
					ports.AtributoResponsableDestino:       material.ResponsableRef,
					ports.AtributoAmbitoIdempotenciaActivo: ambitoActivo,
					ports.AtributoHuellaPeticionAsignacion: huellaActiva,
					ports.AtributoSegregacionAsignacion: strconv.FormatBool(
						politica.ExigeActorDistintoResponsable,
					),
				},
			},
			Finalidad:   string(politica.Finalidad),
			Correlacion: correlacion,
		},
	)
}

func aplicarAsignacion(
	material ports.MaterialHuellaAsignacion,
	preparacion ports.PreparacionAsignacion,
	politica ports.PoliticaAsignacion,
	instante time.Time,
) (domain.Expediente, error) {
	asignacion := domain.AsignacionUnidad{
		UnidadRef:       material.UnidadRef,
		ResponsableRef:  material.ResponsableRef,
		NotificacionRef: preparacion.Referencias.NotificacionRef,
		AsignadaEn:      instante,
		Observaciones:   material.Observaciones,
	}
	actuacion := domain.DatosActuacion{
		AccionClave:   politica.Accion,
		ActorRef:      material.ActorRef,
		UnidadRef:     politica.UnidadEjecutoraRef,
		ReciboRef:     preparacion.Referencias.ReciboRef,
		RealizadaEn:   instante,
		FaseDestino:   preparacion.Expediente.FaseActual,
		EstadoDestino: preparacion.Expediente.EstadoActual,
		Observaciones: material.Observaciones,
	}
	if material.Operacion == ports.OperacionRegistrarAsignacion {
		return preparacion.Expediente.RegistrarAsignacion(
			material.VersionExpediente,
			asignacion,
			actuacion,
		)
	}
	asignacion.MotivoClave = material.MotivoReasignacionClave
	return preparacion.Expediente.ReasignarUnidad(
		material.VersionExpediente,
		asignacion,
		actuacion,
	)
}

func clasificarFalloAsignacion(ctx context.Context, seguro error) error {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	if errors.Is(seguro, ports.ErrClaveIdempotenciaUsada) {
		return ports.ErrClaveIdempotenciaUsada
	}
	if errors.Is(seguro, ErrAsignacionDenegada) {
		return ErrAsignacionDenegada
	}
	return ports.ErrPersistenciaAsignacionNoDisponible
}
