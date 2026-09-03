package bootstrap

import (
	"context"
	"errors"
	"strconv"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var errAutorizacionCoberturaDesarrolloNoDisponible = errors.New(
	"contratacion temporal: autorizacion de cobertura de desarrollo no disponible",
)

type generadorCorrelacionCoberturaDesarrollo interface {
	NuevaReferenciaCorrelacionAutorizacionV2(context.Context) (string, error)
}

type autorizadorConsultasCoberturaDesarrollo struct {
	soporte            *soporteAltaContratacionTemporalDesarrollo
	autorizador        puertosvec.AutorizadorSolicitudLigadaV3
	generador          generadorCorrelacionCoberturaDesarrollo
	motivoPropuesta    dominiovec.ReferenciaEntradaCatalogo
	motivoRecuperacion dominiovec.ReferenciaEntradaCatalogo
}

var (
	_ application.AutorizadorPresentacionPropuestaCobertura = (*autorizadorConsultasCoberturaDesarrollo)(nil)
	_ ports.AutorizadorLecturaResultadoCobertura            = (*autorizadorConsultasCoberturaDesarrollo)(nil)
	_ httpinterno.AutoridadContextoCanalCobertura           = (*soporteAltaContratacionTemporalDesarrollo)(nil)
	_ ports.ResolutorContextoRecuperacionResultadoCobertura = (*soporteAltaContratacionTemporalDesarrollo)(nil)
)

func nuevoAutorizadorConsultasCoberturaDesarrollo(
	soporte *soporteAltaContratacionTemporalDesarrollo,
	autorizador puertosvec.AutorizadorSolicitudLigadaV3,
	generador generadorCorrelacionCoberturaDesarrollo,
) (*autorizadorConsultasCoberturaDesarrollo, error) {
	if soporte == nil || dependenciaEsNulaContratacionTemporalDesarrollo(autorizador) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(generador) ||
		!dominiovec.ReferenciaMotivoAutorizacionV2Valida(
			soporte.motivoPropuestaCobertura,
		) || !dominiovec.ReferenciaMotivoAutorizacionV2Valida(
		soporte.motivoResultadoCobertura,
	) || soporte.motivoPropuestaCobertura == soporte.motivoResultadoCobertura {
		return nil, errAutorizacionCoberturaDesarrolloNoDisponible
	}
	return &autorizadorConsultasCoberturaDesarrollo{
		soporte:            soporte,
		autorizador:        autorizador,
		generador:          generador,
		motivoPropuesta:    soporte.motivoPropuestaCobertura,
		motivoRecuperacion: soporte.motivoResultadoCobertura,
	}, nil
}

func (s *soporteAltaContratacionTemporalDesarrollo) ResolverContextoCanalCobertura(
	ctx context.Context,
) (httpinterno.ContextoCanalCobertura, error) {
	capacidad, valida := s.capacidadValida(ctx)
	if !valida || (capacidad.ruta != httpinterno.RutaPropuestaCobertura &&
		capacidad.ruta != httpinterno.RutaDecisionCobertura &&
		capacidad.ruta != httpinterno.RutaRectificacionCobertura) {
		return httpinterno.ContextoCanalCobertura{}, ports.ErrAutorizacionDenegada
	}
	vinculo, err := s.contexto.Vinculo.Datos()
	if err != nil {
		return httpinterno.ContextoCanalCobertura{}, ports.ErrAutorizacionDenegada
	}
	return httpinterno.ContextoCanalCobertura{
		AutenticacionRef: vinculo.AutenticacionRef,
		SesionRef:        vinculo.SesionRef,
		PerfilRef:        vinculo.PerfilActivoRef,
		OrganizacionRef:  organizacionAltaContratacionTemporalDesarrollo,
	}, nil
}

func (s *soporteAltaContratacionTemporalDesarrollo) ResolverContextoRecuperacionResultadoCobertura(
	ctx context.Context,
) (ports.ContextoRecuperacionResultadoCobertura, error) {
	capacidad, valida := s.capacidadValida(ctx)
	if !valida || capacidad.ruta != httpinterno.RutaResultadoCobertura {
		return ports.ContextoRecuperacionResultadoCobertura{},
			ports.ErrAutorizacionDenegada
	}
	vinculo, err := s.contexto.Vinculo.Datos()
	if err != nil {
		return ports.ContextoRecuperacionResultadoCobertura{},
			ports.ErrAutorizacionDenegada
	}
	solicitud := ports.SolicitudResolverContextoAutorizacionAltaV3{
		AutenticacionRef: vinculo.AutenticacionRef,
		SesionRef:        vinculo.SesionRef,
		PerfilRef:        vinculo.PerfilActivoRef,
	}
	return ports.NuevoContextoRecuperacionResultadoCobertura(
		solicitud,
		s.contexto,
		organizacionAltaContratacionTemporalDesarrollo,
		s.reloj.Ahora(),
	)
}

func (a *autorizadorConsultasCoberturaDesarrollo) AutorizarPresentacionPropuestaCobertura(
	ctx context.Context,
	solicitudContexto ports.SolicitudResolverContextoAutorizacionAltaV3,
	contexto ports.ContextoAutorizacionAltaV3,
	analisis cobertura.SolicitudInstantaneaAnalisisDurableO3,
	instante time.Time,
) error {
	if a == nil || a.soporte == nil || contextoInterfazNulo(ctx) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(a.autorizador) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(a.generador) ||
		contexto.ValidarPara(solicitudContexto, instante) != nil {
		return application.ErrPresentacionPropuestaCoberturaDenegada
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	capacidad, valida := a.soporte.capacidadValida(ctx)
	if !valida || capacidad.ruta != httpinterno.RutaPropuestaCobertura {
		return application.ErrPresentacionPropuestaCoberturaDenegada
	}
	organizacionRef, expedienteRef, versionEsperada, err := analisis.Coordenadas()
	if err != nil || organizacionRef != organizacionAltaContratacionTemporalDesarrollo {
		return application.ErrPresentacionPropuestaCoberturaDenegada
	}
	solicitud, err := a.nuevaSolicitud(
		ctx,
		contexto,
		a.motivoPropuesta,
		accionPropuestaCoberturaDesarrollo,
		finalidadPropuestaCoberturaDesarrollo,
		organizacionRef,
		expedienteRef,
		map[string]string{"version_esperada": strconv.FormatUint(versionEsperada, 10)},
	)
	if err != nil {
		return errAutorizacionCoberturaDesarrolloNoDisponible
	}
	concedida, err := a.exigir(ctx, solicitud, contexto)
	if errContexto := ctx.Err(); errContexto != nil {
		return errContexto
	}
	if falloInfraestructuraAutorizacionCoberturaDesarrollo(err) {
		return application.ErrPresentacionPropuestaCoberturaNoDisponible
	}
	if err != nil || !concedida {
		return application.ErrPresentacionPropuestaCoberturaDenegada
	}
	return nil
}

func (a *autorizadorConsultasCoberturaDesarrollo) AutorizarLecturaResultadoCobertura(
	ctx context.Context,
	solicitud ports.SolicitudLecturaResultadoCobertura,
) (ports.ResultadoAutorizacionLecturaResultadoCobertura, error) {
	if a == nil || a.soporte == nil || contextoInterfazNulo(ctx) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(a.autorizador) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(a.generador) {
		return ports.AutorizacionLecturaResultadoCoberturaDenegada,
			errAutorizacionCoberturaDesarrolloNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return ports.AutorizacionLecturaResultadoCoberturaDenegada, err
	}
	capacidad, valida := a.soporte.capacidadValida(ctx)
	if !valida || capacidad.ruta != httpinterno.RutaResultadoCobertura {
		return ports.AutorizacionLecturaResultadoCoberturaDenegada, nil
	}
	datos, err := solicitud.Datos()
	if err != nil || datos.OrganizacionRef != organizacionAltaContratacionTemporalDesarrollo {
		return ports.AutorizacionLecturaResultadoCoberturaDenegada, nil
	}
	peticion, err := a.nuevaSolicitud(
		ctx,
		datos.Contexto,
		a.motivoRecuperacion,
		string(datos.Accion),
		string(datos.Finalidad),
		datos.OrganizacionRef,
		datos.ExpedienteRef,
		map[string]string{},
	)
	if err != nil {
		return ports.AutorizacionLecturaResultadoCoberturaDenegada,
			errAutorizacionCoberturaDesarrolloNoDisponible
	}
	concedida, err := a.exigir(ctx, peticion, datos.Contexto)
	if errContexto := ctx.Err(); errContexto != nil {
		return ports.AutorizacionLecturaResultadoCoberturaDenegada, errContexto
	}
	if falloInfraestructuraAutorizacionCoberturaDesarrollo(err) {
		return ports.AutorizacionLecturaResultadoCoberturaDenegada,
			errAutorizacionCoberturaDesarrolloNoDisponible
	}
	if err != nil || !concedida {
		return ports.AutorizacionLecturaResultadoCoberturaDenegada, nil
	}
	return ports.AutorizacionLecturaResultadoCoberturaConcedida, nil
}

func (a *autorizadorConsultasCoberturaDesarrollo) nuevaSolicitud(
	ctx context.Context,
	contexto ports.ContextoAutorizacionAltaV3,
	motivo dominiovec.ReferenciaEntradaCatalogo,
	accion string,
	finalidad string,
	organizacionRef string,
	expedienteRef string,
	atributos map[string]string,
) (dominiovec.SolicitudAutorizacionLigadaV3, error) {
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(
		ctx,
		a.generador,
	)
	if err != nil {
		return dominiovec.SolicitudAutorizacionLigadaV3{}, err
	}
	return dominiovec.NuevaSolicitudAutorizacionLigadaV3(
		dominiovec.DatosSolicitudAutorizacionLigadaV3{
			VinculoAutenticacionActor: contexto.Vinculo,
			ReferenciaMotivo:          motivo,
			Accion:                    accion,
			Recurso: dominiovec.RecursoAutorizable{
				Referencia: expedienteRef,
				ModuloID:   ports.ModuloContratacion,
				Tipo:       ports.TipoRecursoExpediente,
				Ambitos: map[string]string{
					"organizacion_ref":     organizacionRef,
					"unidad_ejecutora_ref": unidadCoberturaContratacionTemporalDesarrollo,
				},
				Atributos: atributos,
			},
			Finalidad:   finalidad,
			Correlacion: correlacion,
		},
	)
}

func (a *autorizadorConsultasCoberturaDesarrollo) exigir(
	ctx context.Context,
	solicitud dominiovec.SolicitudAutorizacionLigadaV3,
	contexto ports.ContextoAutorizacionAltaV3,
) (bool, error) {
	decision, confirmacion, err := a.autorizador.ExigirSolicitudLigadaV3(
		ctx,
		solicitud,
		contexto.Resultado,
	)
	if err != nil {
		return false, err
	}
	concedida, _, err := decision.Resultado()
	if err != nil || !concedida || decision.ValidarPara(solicitud) != nil {
		return false, dominiovec.ErrAutorizacionDenegada
	}
	if _, err := confirmacion.Datos(); err != nil {
		return false, dominiovec.ErrAutorizacionDenegada
	}
	return true, nil
}

func falloInfraestructuraAutorizacionCoberturaDesarrollo(err error) bool {
	return err != nil && (errors.Is(err, puertosvec.ErrFuenteAutorizacionNoDisponible) ||
		errors.Is(err, puertosvec.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible) ||
		errors.Is(err, puertosvec.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible) ||
		errors.Is(err, puertosvec.ErrConfirmacionRegistroConcesionAutorizacionLigadaV3Invalida))
}
