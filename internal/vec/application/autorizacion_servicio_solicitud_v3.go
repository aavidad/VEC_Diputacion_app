package application

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// ServicioAutorizacionSolicitudLigadaV3 usa exclusivamente solicitud, vinculo,
// evidencia, decision, ordenes y confirmacion V3. No implementa los puertos V1
// ni V2 y no convierte sus capacidades nominales a documentos historicos.
type ServicioAutorizacionSolicitudLigadaV3 struct {
	protector            protectorDependenciasAutorizacionLigadaV3
	fuente               ports.FuenteAutorizacion
	registroConcesiones  ports.RegistroConcesionesCandidatasAutorizacionLigadaV3
	registroDenegaciones ports.RegistroDenegacionesAutorizacionLigadaV3
	validadorMotivos     ports.ValidadorReferenciaMotivoAutorizacionV2
	reloj                ports.Reloj
	generador            ports.GeneradorReferenciaDecisionAutorizacion
	vigenciaDecision     time.Duration
}

// protectorDependenciasAutorizacionLigadaV3 evita que una copia accidental
// del servicio parezca una nueva composicion; no contiene datos sensibles.
type protectorDependenciasAutorizacionLigadaV3 struct{}

func NuevoServicioAutorizacionSolicitudLigadaV3(
	fuente ports.FuenteAutorizacion,
	registroConcesiones ports.RegistroConcesionesCandidatasAutorizacionLigadaV3,
	registroDenegaciones ports.RegistroDenegacionesAutorizacionLigadaV3,
	validadorMotivos ports.ValidadorReferenciaMotivoAutorizacionV2,
	reloj ports.Reloj,
	generador ports.GeneradorReferenciaDecisionAutorizacion,
	configuracion ConfiguracionServicioAutorizacion,
) (*ServicioAutorizacionSolicitudLigadaV3, error) {
	if dependenciaAutorizacionNula(fuente) || dependenciaAutorizacionNula(registroConcesiones) ||
		dependenciaAutorizacionNula(registroDenegaciones) || dependenciaAutorizacionNula(validadorMotivos) ||
		dependenciaAutorizacionNula(reloj) || dependenciaAutorizacionNula(generador) {
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
	return &ServicioAutorizacionSolicitudLigadaV3{
		fuente: fuente, registroConcesiones: registroConcesiones,
		registroDenegaciones: registroDenegaciones, validadorMotivos: validadorMotivos,
		reloj: reloj, generador: generador, vigenciaDecision: vigencia,
	}, nil
}

func (s *ServicioAutorizacionSolicitudLigadaV3) ExigirSolicitudLigadaV3(
	ctx context.Context,
	solicitud domain.SolicitudAutorizacionLigadaV3,
	resultadoContexto domain.ResultadoContextoActorRegistradoV2,
) (
	domain.DecisionAutorizacionLigadaV3,
	ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
	error,
) {
	vacia := ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}
	if s == nil || ctx == nil || dependenciaAutorizacionNula(s.fuente) ||
		dependenciaAutorizacionNula(s.registroConcesiones) ||
		dependenciaAutorizacionNula(s.registroDenegaciones) ||
		dependenciaAutorizacionNula(s.validadorMotivos) || dependenciaAutorizacionNula(s.reloj) ||
		dependenciaAutorizacionNula(s.generador) {
		return domain.DecisionAutorizacionLigadaV3{}, vacia, nuevoErrorServicioAutorizacionLigadaV3(
			domain.ErrAutorizacionDenegada, domain.ErrConfiguracionAccesoInvalida,
		)
	}
	if err := ctx.Err(); err != nil {
		return domain.DecisionAutorizacionLigadaV3{}, vacia,
			nuevoErrorServicioAutorizacionLigadaV3(domain.ErrAutorizacionDenegada, err)
	}
	datosSolicitud, err := solicitud.Datos()
	if err != nil {
		return domain.DecisionAutorizacionLigadaV3{}, vacia, nuevoErrorServicioAutorizacionLigadaV3(
			domain.ErrAutorizacionDenegada, domain.ErrSolicitudAutorizacionInvalida, err,
		)
	}
	resultado, err := resultadoContexto.Clonar()
	if err != nil || datosSolicitud.VinculoAutenticacionActor.ValidarPara(resultado) != nil {
		return domain.DecisionAutorizacionLigadaV3{}, vacia, nuevoErrorServicioAutorizacionLigadaV3(
			domain.ErrAutorizacionDenegada, domain.ErrSolicitudAutorizacionInvalida,
			domain.ErrVinculoAutenticacionActorV2Invalido, err,
		)
	}
	instante := s.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if instante.IsZero() || instante.Year() < 1 || instante.Year() > 9999 {
		return domain.DecisionAutorizacionLigadaV3{}, vacia, nuevoErrorServicioAutorizacionLigadaV3(
			domain.ErrAutorizacionDenegada, domain.ErrConfiguracionAccesoInvalida,
		)
	}
	if !datosSolicitud.VinculoAutenticacionActor.VigenteEn(instante, resultado) {
		return domain.DecisionAutorizacionLigadaV3{}, vacia, nuevoErrorServicioAutorizacionLigadaV3(
			domain.ErrAutorizacionDenegada, domain.ErrVinculoAutenticacionActorV2Invalido,
		)
	}
	if err := s.validadorMotivos.ValidarReferenciaMotivoAutorizacionV2(
		ctx, datosSolicitud.ReferenciaMotivo, instante,
	); err != nil {
		return domain.DecisionAutorizacionLigadaV3{}, vacia, nuevoErrorServicioAutorizacionLigadaV3(
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
		return domain.DecisionAutorizacionLigadaV3{}, vacia, nuevoErrorServicioAutorizacionLigadaV3(
			domain.ErrAutorizacionDenegada, domain.ErrSolicitudAutorizacionInvalida, err,
		)
	}
	instantaneaViva, err := s.fuente.ObtenerInstantaneaAutorizacion(
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
		return domain.DecisionAutorizacionLigadaV3{}, vacia, nuevoErrorServicioAutorizacionLigadaV3(
			domain.ErrAutorizacionDenegada, domain.ErrConfiguracionAccesoInvalida, err,
		)
	}
	referenciaDecision, err := s.generador.NuevaReferenciaDecisionAutorizacion()
	if err != nil || referenciaDecision == "" || referenciaDecision != strings.TrimSpace(referenciaDecision) {
		return domain.DecisionAutorizacionLigadaV3{}, vacia, nuevoErrorServicioAutorizacionLigadaV3(
			domain.ErrAutorizacionDenegada, domain.ErrConfiguracionAccesoInvalida,
			sanearErrorDependenciaAutorizacionLigadaV3(err),
		)
	}
	if err := ctx.Err(); err != nil {
		return domain.DecisionAutorizacionLigadaV3{}, vacia,
			nuevoErrorServicioAutorizacionLigadaV3(domain.ErrAutorizacionDenegada, err)
	}
	evidencia, err := domain.NuevaEvidenciaEvaluacionAutorizacionV3(
		solicitud, instantanea, referenciaDecision, instante, instante.Add(s.vigenciaDecision),
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
		orden, errOrden := ports.NuevaOrdenRegistroDenegacionAutorizacionLigadaV3(
			solicitud, decision, datosSolicitud.ReferenciaMotivo, resultado,
		)
		if errOrden != nil {
			return decision, vacia, nuevoErrorServicioAutorizacionLigadaV3(domain.ErrAutorizacionDenegada, errOrden)
		}
		if errContexto := ctx.Err(); errContexto != nil {
			return decision, vacia, nuevoErrorServicioAutorizacionLigadaV3(domain.ErrAutorizacionDenegada, errContexto)
		}
		errRegistro := s.registroDenegaciones.RegistrarDenegacionAutorizacionLigadaV3(ctx, orden)
		if errRegistro != nil {
			return decision, vacia, nuevoErrorServicioAutorizacionLigadaV3(
				domain.ErrAutorizacionDenegada,
				ports.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible,
				sanearErrorDependenciaAutorizacionLigadaV3(errRegistro), ctx.Err(),
			)
		}
		return decision, vacia, domain.ErrAutorizacionDenegada
	}
	orden, err := ports.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(
		solicitud, decision, datosSolicitud.ReferenciaMotivo, resultado,
	)
	if err != nil {
		return decision, vacia, nuevoErrorServicioAutorizacionLigadaV3(domain.ErrAutorizacionDenegada, err)
	}
	if err := ctx.Err(); err != nil {
		return decision, vacia, nuevoErrorServicioAutorizacionLigadaV3(domain.ErrAutorizacionDenegada, err)
	}
	confirmacion, err := ports.
		RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(
			ctx, s.registroConcesiones, orden,
		)
	if err != nil {
		if errors.Is(err, ports.ErrInstantaneaAutorizacionObsoleta) {
			return decision, vacia, nuevoErrorServicioAutorizacionLigadaV3(
				domain.ErrAutorizacionDenegada,
				sanearErrorDependenciaAutorizacionLigadaV3(err), ctx.Err(),
			)
		}
		return decision, vacia, nuevoErrorServicioAutorizacionLigadaV3(
			domain.ErrAutorizacionDenegada,
			ports.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible,
			sanearErrorDependenciaAutorizacionLigadaV3(err), ctx.Err(),
		)
	}
	// El puerto solo devuelve exito despues del COMMIT. Una cancelacion que
	// compite con ese retorno no puede borrar una confirmacion durable ni forzar
	// al llamador a reintentar a ciegas.
	datosConfirmacion, err := confirmacion.Datos()
	if err != nil || confirmacion.ValidarPara(orden) != nil ||
		!confirmacion.DentroDeVentanaEn(datosConfirmacion.RegistradaEn) {
		return decision, vacia, nuevoErrorServicioAutorizacionLigadaV3(
			domain.ErrAutorizacionDenegada,
			ports.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible,
			ports.ErrConfirmacionRegistroConcesionAutorizacionLigadaV3Invalida,
			err,
		)
	}
	return decision, confirmacion, nil
}

// errorDependenciaAutorizacionLigadaV3 conserva errors.Is/As para telemetria y
// control de flujo, pero evita que mensajes arbitrarios de adaptadores (SQL,
// rutas, identificadores o PII) atraviesen la frontera publica de Error().
type errorDependenciaAutorizacionLigadaV3 struct{ causa error }

func (e errorDependenciaAutorizacionLigadaV3) Error() string {
	return "vec: fallo interno de dependencia de autorizacion ligada V3"
}

func (e errorDependenciaAutorizacionLigadaV3) Unwrap() error { return e.causa }

func (e errorDependenciaAutorizacionLigadaV3) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, e.Error())
}

func (e errorDependenciaAutorizacionLigadaV3) LogValue() slog.Value {
	return slog.StringValue(e.Error())
}

func sanearErrorDependenciaAutorizacionLigadaV3(causa error) error {
	if causa == nil {
		return nil
	}
	return errorDependenciaAutorizacionLigadaV3{causa: causa}
}

// errorServicioAutorizacionLigadaV3 impide que fmt con verbos de depuracion o
// slog recorran por reflexion causas arbitrarias. Unwrap conserva la identidad
// para errors.Is/As sin exponer el texto del adaptador en la salida normal.
type errorServicioAutorizacionLigadaV3 struct{ causas []error }

func (e errorServicioAutorizacionLigadaV3) Error() string {
	var salida strings.Builder
	for _, causa := range e.causas {
		if causa == nil {
			continue
		}
		if salida.Len() > 0 {
			salida.WriteByte('\n')
		}
		salida.WriteString(causa.Error())
	}
	return salida.String()
}

func (e errorServicioAutorizacionLigadaV3) Unwrap() []error {
	return append([]error(nil), e.causas...)
}

func (e errorServicioAutorizacionLigadaV3) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, e.Error())
}

func (e errorServicioAutorizacionLigadaV3) LogValue() slog.Value {
	return slog.StringValue(e.Error())
}

func nuevoErrorServicioAutorizacionLigadaV3(causas ...error) error {
	filtradas := make([]error, 0, len(causas))
	for _, causa := range causas {
		if causa != nil {
			filtradas = append(filtradas, causa)
		}
	}
	if len(filtradas) == 0 {
		return nil
	}
	return errorServicioAutorizacionLigadaV3{causas: filtradas}
}

func clonarInstantaneaAutorizacionLigadaV3(
	origen domain.InstantaneaAutorizacion,
) (domain.InstantaneaAutorizacion, error) {
	if origen.Validar() != nil {
		return domain.InstantaneaAutorizacion{}, domain.ErrConfiguracionAccesoInvalida
	}
	copia := origen
	copia.AsignacionPerfil.Ambitos = append(
		[]domain.AmbitoPerfil(nil), origen.AsignacionPerfil.Ambitos...,
	)
	for indice := range copia.AsignacionPerfil.Ambitos {
		copia.AsignacionPerfil.Ambitos[indice].Valores = append(
			[]string(nil), origen.AsignacionPerfil.Ambitos[indice].Valores...,
		)
	}
	copia.VersionRol.Concesiones = append([]domain.ConcesionRol(nil), origen.VersionRol.Concesiones...)
	for indice := range copia.VersionRol.Concesiones {
		concesionOrigen := origen.VersionRol.Concesiones[indice]
		concesion := &copia.VersionRol.Concesiones[indice]
		concesion.Finalidades = append([]string(nil), concesionOrigen.Finalidades...)
		concesion.CamposPermitidos = append([]string(nil), concesionOrigen.CamposPermitidos...)
		concesion.Obligaciones = append([]string(nil), concesionOrigen.Obligaciones...)
	}
	copia.Politicas = append([]domain.PoliticaRestrictiva(nil), origen.Politicas...)
	for indice := range copia.Politicas {
		politicaOrigen := origen.Politicas[indice]
		politica := &copia.Politicas[indice]
		politica.Acciones = append([]string(nil), politicaOrigen.Acciones...)
		politica.Modulos = append([]string(nil), politicaOrigen.Modulos...)
		politica.TiposRecurso = append([]string(nil), politicaOrigen.TiposRecurso...)
		politica.FinalidadesPermitidas = append([]string(nil), politicaOrigen.FinalidadesPermitidas...)
		politica.CamposPermitidos = append([]string(nil), politicaOrigen.CamposPermitidos...)
		politica.Obligaciones = append([]string(nil), politicaOrigen.Obligaciones...)
		politica.Restricciones = append(
			[]domain.RestriccionAtributoRecurso(nil), politicaOrigen.Restricciones...,
		)
		for indiceRestriccion := range politica.Restricciones {
			politica.Restricciones[indiceRestriccion].ValoresPermitidos = append(
				[]string(nil), politicaOrigen.Restricciones[indiceRestriccion].ValoresPermitidos...,
			)
		}
	}
	if copia.Validar() != nil {
		return domain.InstantaneaAutorizacion{}, domain.ErrConfiguracionAccesoInvalida
	}
	return copia, nil
}

var _ ports.AutorizadorSolicitudLigadaV3 = (*ServicioAutorizacionSolicitudLigadaV3)(nil)
