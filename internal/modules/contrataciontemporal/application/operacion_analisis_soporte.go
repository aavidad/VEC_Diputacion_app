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

func (s *ServicioOperacionAnalisis) dependenciasValidas() bool {
	return s != nil && !dependenciaNula(s.contextos) &&
		!dependenciaNula(s.artefactos) && !dependenciaNula(s.sellador) &&
		!dependenciaNula(s.preparaciones) &&
		!dependenciaNula(s.politicas) &&
		!dependenciaNula(s.correlaciones) &&
		!dependenciaNula(s.autorizador) && !dependenciaNula(s.reloj) &&
		!dependenciaNula(s.transaccion)
}

func (s datosSolicitudOperacionAnalisis) validar(instante time.Time) error {
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
		!domain.ReferenciaOpacaValida(s.artefactoRef) ||
		s.datosFuncionales.Validar() != nil ||
		!domain.InstanteUTCCanonico(instante) {
		return ErrSolicitudOperacionAnalisisInvalida
	}
	if s.operacion == ports.OperacionRegistrarAnalisis {
		if s.motivoRectificacion != "" {
			return ErrSolicitudOperacionAnalisisInvalida
		}
		return nil
	}
	if !s.motivoRectificacion.Valida() {
		return ErrSolicitudOperacionAnalisisInvalida
	}
	return nil
}

func actorAnalisisAnterior(
	expediente domain.Expediente,
	operacion ports.TipoOperacionAnalisis,
) (string, error) {
	if expediente.Validar() != nil ||
		!ports.VersionOperacionAnalisisConIncrementoValida(
			expediente.Version,
		) {
		return "", ErrSolicitudOperacionAnalisisInvalida
	}
	if operacion == ports.OperacionRegistrarAnalisis {
		if expediente.Analisis != nil {
			return "", ErrSolicitudOperacionAnalisisInvalida
		}
		return "", nil
	}
	if expediente.Analisis == nil ||
		expediente.Analisis.ActuacionRegistro == nil {
		return "", ErrSolicitudOperacionAnalisisInvalida
	}
	secuencia := expediente.Analisis.ActuacionRegistro.Secuencia
	if secuencia == 0 || secuencia > uint64(len(expediente.Actuaciones)) {
		return "", ErrSolicitudOperacionAnalisisInvalida
	}
	actuacion := expediente.Actuaciones[secuencia-1]
	if actuacion.Secuencia != secuencia ||
		actuacion.VersionExpediente !=
			expediente.Analisis.ActuacionRegistro.VersionExpediente ||
		!domain.ReferenciaOpacaValida(actuacion.ActorRef) {
		return "", ErrSolicitudOperacionAnalisisInvalida
	}
	return actuacion.ActorRef, nil
}

func (s *ServicioOperacionAnalisis) nuevaSolicitudAutorizacion(
	ctx context.Context,
	contexto ports.ContextoAutorizacionAltaV3,
	preparacion ports.DatosPreparacionOperacionAnalisis,
	politica ports.PoliticaOperacionAnalisis,
) (dominiovec.SolicitudAutorizacionLigadaV3, error) {
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
				Referencia: preparacion.ExpedienteRef,
				ModuloID:   ports.ModuloContratacion,
				Tipo:       ports.TipoRecursoAnalisis,
				Ambitos: map[string]string{
					"organizacion_ref": preparacion.OrganizacionRef,
					"expediente_ref":   preparacion.ExpedienteRef,
					"fase_previa":      string(politica.FasePrevia),
					"estado_previo":    string(politica.EstadoPrevio),
				},
				Atributos: map[string]string{
					ports.AtributoOperacionAnalisis:       string(preparacion.Operacion),
					ports.AtributoVersionAnalisis:         strconv.FormatUint(preparacion.VersionExpediente, 10),
					ports.AtributoPoliticaAnalisisRef:     politica.DefinicionRef,
					ports.AtributoPoliticaAnalisisVersion: strconv.FormatUint(politica.Version, 10),
					ports.AtributoPoliticaAnalisisHuella:  politica.HuellaSHA256,
					ports.AtributoArtefactoAnalisisRef:    preparacion.ArtefactoRef,
					ports.AtributoArtefactoAnalisisHuella: preparacion.ArtefactoHuellaSHA256,
					ports.AtributoHuellaSemanticaAnalisis: preparacion.HuellaSemanticaHMAC,
					ports.AtributoSegregacionAnalisis:     strconv.FormatBool(politica.ExigeActorDistinto),
				},
			},
			Finalidad:   string(politica.Finalidad),
			Correlacion: correlacion,
		},
	)
}

func aplicarOperacionAnalisis(
	solicitud datosSolicitudOperacionAnalisis,
	anterior domain.Expediente,
	solicitudArtefacto ports.SolicitudPrepararArtefactoAnalisis,
	artefacto ports.ArtefactoAnalisisPreparado,
	politica ports.PoliticaOperacionAnalisis,
	reciboRef string,
	instante time.Time,
) (domain.Expediente, error) {
	analisis, err := ports.DerivarAnalisisDesdeArtefacto(
		solicitudArtefacto,
		artefacto,
	)
	if err != nil {
		return domain.Expediente{}, err
	}
	actuacion := domain.DatosActuacion{
		AccionClave:   politica.Accion,
		ActorRef:      politica.ActorRef,
		UnidadRef:     politica.UnidadRef,
		ReciboRef:     reciboRef,
		RealizadaEn:   instante,
		FaseDestino:   anterior.FaseActual,
		EstadoDestino: anterior.EstadoActual,
	}
	if solicitud.operacion == ports.OperacionRegistrarAnalisis {
		return anterior.RegistrarAnalisis(
			solicitud.versionEsperada,
			analisis,
			actuacion,
		)
	}
	actuacion.Observaciones =
		string(politica.MotivoRectificacion.ClaveMensajeI18N)
	return anterior.RectificarAnalisis(
		solicitud.versionEsperada,
		analisis,
		actuacion,
	)
}

func clasificarResultadoAutorizador(
	ctx context.Context,
	solicitud dominiovec.SolicitudAutorizacionLigadaV3,
	decision dominiovec.DecisionAutorizacionLigadaV3,
) error {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return nuevoErrorOperacionAnalisis(tipoErrorDependencia, err)
		}
	}
	concedida, _, err := decision.Resultado()
	if err == nil && decision.ValidarPara(solicitud) == nil && !concedida {
		return nuevoErrorOperacionAnalisis(tipoErrorDenegacion, nil)
	}
	return nuevoErrorOperacionAnalisis(tipoErrorDependencia, nil)
}

func clasificarFalloAutorizacion(ctx context.Context, causa error) error {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return nuevoErrorOperacionAnalisis(tipoErrorDependencia, err)
		}
	}
	if errors.Is(causa, ports.ErrAutorizacionDenegada) ||
		errors.Is(causa, dominiovec.ErrAutorizacionDenegada) {
		return nuevoErrorOperacionAnalisis(tipoErrorDenegacion, nil)
	}
	return nuevoErrorOperacionAnalisis(tipoErrorDependencia, nil)
}

func clasificarFalloPersistencia(ctx context.Context, causa error) error {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return nuevoErrorOperacionAnalisis(tipoErrorDependencia, err)
		}
	}
	if errors.Is(
		causa,
		ports.ErrClaveIdempotenciaOperacionAnalisisUsada,
	) || errors.Is(causa, ports.ErrConjuntoFuentesAnalisisYaConsumido) ||
		errors.Is(causa, domain.ErrVersionEnConflicto) {
		return nuevoErrorOperacionAnalisis(tipoErrorConflicto, nil)
	}
	return nuevoErrorOperacionAnalisis(tipoErrorDependencia, nil)
}

func errorDependenciaOperacionAnalisis(ctx context.Context) error {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return nuevoErrorOperacionAnalisis(tipoErrorDependencia, err)
		}
	}
	return nuevoErrorOperacionAnalisis(tipoErrorDependencia, nil)
}
