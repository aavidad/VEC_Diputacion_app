package application

import (
	"context"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// ServicioPreparacionSolicitudLigadaV3 evalua una solicitud V3 y prepara una
// orden candidata opaca. Nunca registra ni confirma una concesion. Las
// denegaciones sí se registran para no perder su traza obligatoria.
type ServicioPreparacionSolicitudLigadaV3 struct {
	protector            protectorDependenciasAutorizacionLigadaV3
	fuente               ports.FuenteAutorizacion
	registroDenegaciones ports.RegistroDenegacionesAutorizacionLigadaV3
	validadorMotivos     ports.ValidadorReferenciaMotivoAutorizacionV2
	reloj                ports.Reloj
	generador            ports.GeneradorReferenciaDecisionAutorizacion
	vigenciaDecision     time.Duration
}

func NuevoServicioPreparacionSolicitudLigadaV3(
	fuente ports.FuenteAutorizacion,
	registroDenegaciones ports.RegistroDenegacionesAutorizacionLigadaV3,
	validadorMotivos ports.ValidadorReferenciaMotivoAutorizacionV2,
	reloj ports.Reloj,
	generador ports.GeneradorReferenciaDecisionAutorizacion,
	configuracion ConfiguracionServicioAutorizacion,
) (*ServicioPreparacionSolicitudLigadaV3, error) {
	if dependenciaAutorizacionNula(fuente) ||
		dependenciaAutorizacionNula(registroDenegaciones) ||
		dependenciaAutorizacionNula(validadorMotivos) ||
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
	return &ServicioPreparacionSolicitudLigadaV3{
		fuente: fuente, registroDenegaciones: registroDenegaciones,
		validadorMotivos: validadorMotivos, reloj: reloj,
		generador: generador, vigenciaDecision: vigencia,
	}, nil
}

func (s *ServicioPreparacionSolicitudLigadaV3) PrepararSolicitudLigadaV3(
	ctx context.Context,
	solicitud domain.SolicitudAutorizacionLigadaV3,
	resultadoContexto domain.ResultadoContextoActorRegistradoV2,
) (
	domain.DecisionAutorizacionLigadaV3,
	ports.OrdenRegistroConcesionCandidataAutorizacionLigadaV3,
	error,
) {
	if s == nil {
		return preparacionSolicitudLigadaV3Invalida()
	}
	return prepararSolicitudLigadaV3(
		ctx, solicitud, resultadoContexto,
		dependenciasPreparacionSolicitudLigadaV3{
			fuente: s.fuente, registroDenegaciones: s.registroDenegaciones,
			validadorMotivos: s.validadorMotivos, reloj: s.reloj,
			generador: s.generador, vigenciaDecision: s.vigenciaDecision,
		},
	)
}

// PrepararSolicitudLigadaV3 permite reutilizar el mismo evaluador del servicio
// completo sin ejecutar el registro durable de la concesion.
func (s *ServicioAutorizacionSolicitudLigadaV3) PrepararSolicitudLigadaV3(
	ctx context.Context,
	solicitud domain.SolicitudAutorizacionLigadaV3,
	resultadoContexto domain.ResultadoContextoActorRegistradoV2,
) (
	domain.DecisionAutorizacionLigadaV3,
	ports.OrdenRegistroConcesionCandidataAutorizacionLigadaV3,
	error,
) {
	if s == nil {
		return preparacionSolicitudLigadaV3Invalida()
	}
	return prepararSolicitudLigadaV3(
		ctx, solicitud, resultadoContexto,
		dependenciasPreparacionSolicitudLigadaV3{
			fuente: s.fuente, registroDenegaciones: s.registroDenegaciones,
			validadorMotivos: s.validadorMotivos, reloj: s.reloj,
			generador: s.generador, vigenciaDecision: s.vigenciaDecision,
		},
	)
}

type dependenciasPreparacionSolicitudLigadaV3 struct {
	fuente               ports.FuenteAutorizacion
	registroDenegaciones ports.RegistroDenegacionesAutorizacionLigadaV3
	validadorMotivos     ports.ValidadorReferenciaMotivoAutorizacionV2
	reloj                ports.Reloj
	generador            ports.GeneradorReferenciaDecisionAutorizacion
	vigenciaDecision     time.Duration
}

func prepararSolicitudLigadaV3(
	ctx context.Context,
	solicitud domain.SolicitudAutorizacionLigadaV3,
	resultadoContexto domain.ResultadoContextoActorRegistradoV2,
	dependencias dependenciasPreparacionSolicitudLigadaV3,
) (
	domain.DecisionAutorizacionLigadaV3,
	ports.OrdenRegistroConcesionCandidataAutorizacionLigadaV3,
	error,
) {
	vacia := ports.OrdenRegistroConcesionCandidataAutorizacionLigadaV3{}
	if ctx == nil || !dependencias.validas() {
		return domain.DecisionAutorizacionLigadaV3{}, vacia,
			nuevoErrorServicioAutorizacionLigadaV3(
				domain.ErrAutorizacionDenegada, domain.ErrConfiguracionAccesoInvalida,
			)
	}
	if err := ctx.Err(); err != nil {
		return domain.DecisionAutorizacionLigadaV3{}, vacia,
			nuevoErrorServicioAutorizacionLigadaV3(domain.ErrAutorizacionDenegada, err)
	}
	datosSolicitud, err := solicitud.Datos()
	if err != nil {
		return domain.DecisionAutorizacionLigadaV3{}, vacia,
			nuevoErrorServicioAutorizacionLigadaV3(
				domain.ErrAutorizacionDenegada, domain.ErrSolicitudAutorizacionInvalida, err,
			)
	}
	resultado, err := resultadoContexto.Clonar()
	if err != nil || datosSolicitud.VinculoAutenticacionActor.ValidarPara(resultado) != nil {
		return domain.DecisionAutorizacionLigadaV3{}, vacia,
			nuevoErrorServicioAutorizacionLigadaV3(
				domain.ErrAutorizacionDenegada, domain.ErrSolicitudAutorizacionInvalida,
				domain.ErrVinculoAutenticacionActorV2Invalido, err,
			)
	}
	instante := dependencias.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if instante.IsZero() || instante.Year() < 1 || instante.Year() > 9999 {
		return domain.DecisionAutorizacionLigadaV3{}, vacia,
			nuevoErrorServicioAutorizacionLigadaV3(
				domain.ErrAutorizacionDenegada, domain.ErrConfiguracionAccesoInvalida,
			)
	}
	if !datosSolicitud.VinculoAutenticacionActor.VigenteEn(instante, resultado) {
		return domain.DecisionAutorizacionLigadaV3{}, vacia,
			nuevoErrorServicioAutorizacionLigadaV3(
				domain.ErrAutorizacionDenegada, domain.ErrVinculoAutenticacionActorV2Invalido,
			)
	}
	if err := dependencias.validadorMotivos.ValidarReferenciaMotivoAutorizacionV2(
		ctx, datosSolicitud.ReferenciaMotivo, instante,
	); err != nil {
		return domain.DecisionAutorizacionLigadaV3{}, vacia,
			nuevoErrorServicioAutorizacionLigadaV3(
				domain.ErrAutorizacionDenegada, domain.ErrSolicitudAutorizacionInvalida,
				sanearErrorDependenciaAutorizacionLigadaV3(err), ctx.Err(),
			)
	}
	if err := ctx.Err(); err != nil {
		return domain.DecisionAutorizacionLigadaV3{}, vacia,
			nuevoErrorServicioAutorizacionLigadaV3(domain.ErrAutorizacionDenegada, err)
	}
	vinculo, err := datosSolicitud.VinculoAutenticacionActor.Datos()
	if err != nil {
		return domain.DecisionAutorizacionLigadaV3{}, vacia,
			nuevoErrorServicioAutorizacionLigadaV3(
				domain.ErrAutorizacionDenegada, domain.ErrSolicitudAutorizacionInvalida, err,
			)
	}
	instantaneaViva, err := dependencias.fuente.ObtenerInstantaneaAutorizacion(
		ctx, vinculo.PrincipalID, vinculo.PerfilActivoRef,
	)
	if err != nil {
		return domain.DecisionAutorizacionLigadaV3{}, vacia,
			nuevoErrorServicioAutorizacionLigadaV3(
				domain.ErrAutorizacionDenegada, ports.ErrFuenteAutorizacionNoDisponible,
				sanearErrorDependenciaAutorizacionLigadaV3(err), ctx.Err(),
			)
	}
	if err := ctx.Err(); err != nil {
		return domain.DecisionAutorizacionLigadaV3{}, vacia,
			nuevoErrorServicioAutorizacionLigadaV3(domain.ErrAutorizacionDenegada, err)
	}
	instantanea, err := clonarInstantaneaAutorizacionLigadaV3(instantaneaViva)
	if err != nil || instantanea.AsignacionPerfil.PrincipalID != vinculo.PrincipalID ||
		instantanea.AsignacionPerfil.PerfilActivoRef != vinculo.PerfilActivoRef {
		return domain.DecisionAutorizacionLigadaV3{}, vacia,
			nuevoErrorServicioAutorizacionLigadaV3(
				domain.ErrAutorizacionDenegada, domain.ErrConfiguracionAccesoInvalida, err,
			)
	}
	referenciaDecision, err := dependencias.generador.NuevaReferenciaDecisionAutorizacion()
	if err != nil || referenciaDecision == "" ||
		referenciaDecision != strings.TrimSpace(referenciaDecision) {
		return domain.DecisionAutorizacionLigadaV3{}, vacia,
			nuevoErrorServicioAutorizacionLigadaV3(
				domain.ErrAutorizacionDenegada, domain.ErrConfiguracionAccesoInvalida,
				sanearErrorDependenciaAutorizacionLigadaV3(err),
			)
	}
	if err := ctx.Err(); err != nil {
		return domain.DecisionAutorizacionLigadaV3{}, vacia,
			nuevoErrorServicioAutorizacionLigadaV3(domain.ErrAutorizacionDenegada, err)
	}
	evidencia, err := domain.NuevaEvidenciaEvaluacionAutorizacionV3(
		solicitud, instantanea, referenciaDecision, instante,
		instante.Add(dependencias.vigenciaDecision),
	)
	if err != nil {
		return domain.DecisionAutorizacionLigadaV3{}, vacia,
			nuevoErrorServicioAutorizacionLigadaV3(domain.ErrAutorizacionDenegada, err)
	}
	decision, err := domain.NuevaDecisionAutorizacionLigadaV3(solicitud, evidencia)
	if err != nil {
		return domain.DecisionAutorizacionLigadaV3{}, vacia,
			nuevoErrorServicioAutorizacionLigadaV3(domain.ErrAutorizacionDenegada, err)
	}
	concedida, _, err := decision.Resultado()
	if err != nil {
		return domain.DecisionAutorizacionLigadaV3{}, vacia,
			nuevoErrorServicioAutorizacionLigadaV3(domain.ErrAutorizacionDenegada, err)
	}
	if !concedida {
		return registrarDenegacionSolicitudLigadaV3(
			ctx, solicitud, decision, datosSolicitud.ReferenciaMotivo, resultado,
			dependencias.registroDenegaciones,
		)
	}
	orden, err := ports.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(
		solicitud, decision, datosSolicitud.ReferenciaMotivo, resultado,
	)
	if err != nil {
		return decision, vacia,
			nuevoErrorServicioAutorizacionLigadaV3(domain.ErrAutorizacionDenegada, err)
	}
	if err := ctx.Err(); err != nil {
		return decision, vacia,
			nuevoErrorServicioAutorizacionLigadaV3(domain.ErrAutorizacionDenegada, err)
	}
	return decision, orden, nil
}

func (d dependenciasPreparacionSolicitudLigadaV3) validas() bool {
	return !dependenciaAutorizacionNula(d.fuente) &&
		!dependenciaAutorizacionNula(d.registroDenegaciones) &&
		!dependenciaAutorizacionNula(d.validadorMotivos) &&
		!dependenciaAutorizacionNula(d.reloj) &&
		!dependenciaAutorizacionNula(d.generador) &&
		d.vigenciaDecision > 0 &&
		d.vigenciaDecision <= domain.VigenciaMaximaDecisionAutorizacion &&
		d.vigenciaDecision%time.Microsecond == 0
}

func registrarDenegacionSolicitudLigadaV3(
	ctx context.Context,
	solicitud domain.SolicitudAutorizacionLigadaV3,
	decision domain.DecisionAutorizacionLigadaV3,
	referenciaMotivo domain.ReferenciaEntradaCatalogo,
	resultado domain.ResultadoContextoActorRegistradoV2,
	registro ports.RegistroDenegacionesAutorizacionLigadaV3,
) (
	domain.DecisionAutorizacionLigadaV3,
	ports.OrdenRegistroConcesionCandidataAutorizacionLigadaV3,
	error,
) {
	vacia := ports.OrdenRegistroConcesionCandidataAutorizacionLigadaV3{}
	orden, err := ports.NuevaOrdenRegistroDenegacionAutorizacionLigadaV3(
		solicitud, decision, referenciaMotivo, resultado,
	)
	if err != nil {
		return decision, vacia,
			nuevoErrorServicioAutorizacionLigadaV3(domain.ErrAutorizacionDenegada, err)
	}
	if err := ctx.Err(); err != nil {
		return decision, vacia,
			nuevoErrorServicioAutorizacionLigadaV3(domain.ErrAutorizacionDenegada, err)
	}
	if err := registro.RegistrarDenegacionAutorizacionLigadaV3(ctx, orden); err != nil {
		return decision, vacia, nuevoErrorServicioAutorizacionLigadaV3(
			domain.ErrAutorizacionDenegada,
			ports.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible,
			sanearErrorDependenciaAutorizacionLigadaV3(err), ctx.Err(),
		)
	}
	return decision, vacia, domain.ErrAutorizacionDenegada
}

func preparacionSolicitudLigadaV3Invalida() (
	domain.DecisionAutorizacionLigadaV3,
	ports.OrdenRegistroConcesionCandidataAutorizacionLigadaV3,
	error,
) {
	return domain.DecisionAutorizacionLigadaV3{},
		ports.OrdenRegistroConcesionCandidataAutorizacionLigadaV3{},
		nuevoErrorServicioAutorizacionLigadaV3(
			domain.ErrAutorizacionDenegada, domain.ErrConfiguracionAccesoInvalida,
		)
}

var _ ports.PreparadorSolicitudLigadaV3 = (*ServicioPreparacionSolicitudLigadaV3)(nil)
var _ ports.PreparadorSolicitudLigadaV3 = (*ServicioAutorizacionSolicitudLigadaV3)(nil)
