package application

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// ServicioAutorizacionSolicitudLigadaV2 comparte el evaluador RBAC/ABAC, pero
// usa puertos de registro distintos y solo expone el metodo V2. No implementa
// ports.Autorizador ni puede registrar una decision historica por accidente.
type ServicioAutorizacionSolicitudLigadaV2 struct {
	servicio *ServicioAutorizacion
}

func NuevoServicioAutorizacionSolicitudLigadaV2(
	fuente ports.FuenteAutorizacion,
	registroConcesiones ports.RegistroDecisionesAutorizacionSolicitudLigadaV2,
	registroDenegaciones ports.RegistroDenegacionesAutorizacionSolicitudLigadaV2,
	validadorMotivos ports.ValidadorReferenciaMotivoAutorizacionV2,
	reloj ports.Reloj,
	generador ports.GeneradorReferenciaDecisionAutorizacion,
	configuracion ConfiguracionServicioAutorizacion,
) (*ServicioAutorizacionSolicitudLigadaV2, error) {
	if dependenciaAutorizacionNula(fuente) || dependenciaAutorizacionNula(registroConcesiones) ||
		dependenciaAutorizacionNula(registroDenegaciones) || dependenciaAutorizacionNula(validadorMotivos) ||
		dependenciaAutorizacionNula(reloj) ||
		dependenciaAutorizacionNula(generador) {
		return nil, domain.ErrConfiguracionAccesoInvalida
	}
	vigencia := configuracion.VigenciaDecision
	if vigencia == 0 {
		vigencia = vigenciaDecisionPredeterminada
	}
	if vigencia < 0 || vigencia > domain.VigenciaMaximaDecisionAutorizacion ||
		vigencia%time.Microsecond != 0 {
		return nil, domain.ErrConfiguracionAccesoInvalida
	}
	return &ServicioAutorizacionSolicitudLigadaV2{servicio: &ServicioAutorizacion{
		fuente: fuente, registroConcesionesV2: registroConcesiones,
		registroDenegacionesV2: registroDenegaciones, validadorMotivosV2: validadorMotivos, reloj: reloj,
		generador: generador, vigenciaDecision: vigencia,
	}}, nil
}

func (s *ServicioAutorizacionSolicitudLigadaV2) ExigirSolicitudLigadaV2(
	ctx context.Context,
	solicitud domain.SolicitudAutorizacionLigadaV2,
) (domain.DecisionAutorizacion, error) {
	if s == nil || s.servicio == nil {
		return domain.DecisionAutorizacion{}, errors.Join(
			domain.ErrAutorizacionDenegada,
			domain.ErrConfiguracionAccesoInvalida,
		)
	}
	return s.servicio.exigir(ctx, domain.SolicitudAutorizacion{}, &solicitud)
}

var _ ports.AutorizadorSolicitudLigadaV2 = (*ServicioAutorizacionSolicitudLigadaV2)(nil)
