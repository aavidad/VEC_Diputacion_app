package confianzaatestacion

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

var errEmisionMaterialAutorizacionAtestadaV3NoDisponible = errors.New(
	"vec: emision de material de autorizacion atestada V3 no disponible",
)

type atestadorAutorizacionLigadaV3 interface {
	Atestar(
		context.Context,
		domain.DecisionAutorizacionLigadaV3,
		domain.ReferenciaEntradaCatalogo,
		domain.ResultadoContextoActorRegistradoV2,
	) (ports.AtestacionAutorizacionV3, error)
}

// EmisorMaterialAutorizacionAtestadaV3 encadena las autoridades VEC-AD-3 sin
// permitir que el llamador aporte motivo, raíces, audiencias ni claves.
type EmisorMaterialAutorizacionAtestadaV3 struct {
	bloqueoSerializacionV3
	autorizador ports.AutorizadorSolicitudLigadaV3
	atestador   atestadorAutorizacionLigadaV3
	confianza   *ServicioConfianzaAtestacionAutorizacionV3
	emisor      *EmisorCapacidadesAtestacionAutorizacionV3
}

func NuevoEmisorMaterialAutorizacionAtestadaV3(
	autorizador ports.AutorizadorSolicitudLigadaV3,
	atestador atestadorAutorizacionLigadaV3,
	confianza *ServicioConfianzaAtestacionAutorizacionV3,
	emisor *EmisorCapacidadesAtestacionAutorizacionV3,
) (*EmisorMaterialAutorizacionAtestadaV3, error) {
	if dependenciaConfianzaAtestacionNula(autorizador) ||
		dependenciaConfianzaAtestacionNula(atestador) ||
		dependenciaConfianzaAtestacionNula(confianza) ||
		dependenciaConfianzaAtestacionNula(emisor) {
		return nil, nuevoErrorEmisionMaterialAutorizacionAtestadaV3(nil)
	}
	return &EmisorMaterialAutorizacionAtestadaV3{
		autorizador: autorizador,
		atestador:   atestador,
		confianza:   confianza,
		emisor:      emisor,
	}, nil
}

// EmitirMaterialAutorizacionAtestadaV3 devuelve decisión y confirmación
// durables incluso si una fase criptográfica posterior falla. Solo entrega
// material cuando toda la cadena nominal termina dentro del contexto vivo.
func (e *EmisorMaterialAutorizacionAtestadaV3) EmitirMaterialAutorizacionAtestadaV3(
	ctx context.Context,
	solicitud domain.SolicitudAutorizacionLigadaV3,
	resultado domain.ResultadoContextoActorRegistradoV2,
) (
	domain.DecisionAutorizacionLigadaV3,
	ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
	ports.ExportadorMaterialConsumoAutorizacionAtestadaV3,
	error,
) {
	decisionVacia := domain.DecisionAutorizacionLigadaV3{}
	confirmacionVacia := ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}
	if ctx == nil || e == nil ||
		dependenciaConfianzaAtestacionNula(e.autorizador) ||
		dependenciaConfianzaAtestacionNula(e.atestador) ||
		dependenciaConfianzaAtestacionNula(e.confianza) ||
		dependenciaConfianzaAtestacionNula(e.emisor) {
		return decisionVacia, confirmacionVacia, nil, nuevoErrorEmisionMaterialAutorizacionAtestadaV3(nil)
	}
	if err := ctx.Err(); err != nil {
		return decisionVacia, confirmacionVacia, nil, nuevoErrorEmisionMaterialAutorizacionAtestadaV3(err)
	}
	datosSolicitud, errSolicitud := solicitud.Datos()
	resultadoExacto, errResultado := resultado.Clonar()
	if errSolicitud != nil || errResultado != nil ||
		resultadoExacto.Validar() != nil ||
		datosSolicitud.VinculoAutenticacionActor.
			ValidarPara(resultadoExacto) != nil {
		return decisionVacia, confirmacionVacia, nil, nuevoErrorEmisionMaterialAutorizacionAtestadaV3(nil)
	}
	motivo := datosSolicitud.ReferenciaMotivo
	decision, confirmacion, err := e.autorizador.ExigirSolicitudLigadaV3(
		ctx,
		solicitud,
		resultadoExacto,
	)
	if validarConcesionEmisionMaterialV3(
		solicitud,
		decision,
		confirmacion,
		motivo,
		resultadoExacto,
	) != nil {
		return decisionVacia, confirmacionVacia, nil, nuevoErrorEmisionMaterialAutorizacionAtestadaV3(err, ctx.Err())
	}
	if err != nil {
		return decision, confirmacion, nil, nuevoErrorEmisionMaterialAutorizacionAtestadaV3(err, ctx.Err())
	}
	if err := ctx.Err(); err != nil {
		return decision, confirmacion, nil, nuevoErrorEmisionMaterialAutorizacionAtestadaV3(err)
	}
	atestacion, err := e.atestador.Atestar(
		ctx,
		decision,
		motivo,
		resultadoExacto,
	)
	if err != nil {
		return decision, confirmacion, nil, nuevoErrorEmisionMaterialAutorizacionAtestadaV3(err, ctx.Err())
	}
	if err := ctx.Err(); err != nil {
		return decision, confirmacion, nil, nuevoErrorEmisionMaterialAutorizacionAtestadaV3(err)
	}
	prueba, err := e.confianza.Verificar(
		ctx, solicitud, decision, motivo, resultadoExacto, atestacion,
	)
	if err != nil {
		return decision, confirmacion, nil, nuevoErrorEmisionMaterialAutorizacionAtestadaV3(err, ctx.Err())
	}
	if err := ctx.Err(); err != nil {
		return decision, confirmacion, nil, nuevoErrorEmisionMaterialAutorizacionAtestadaV3(err)
	}
	capacidad, err := e.emisor.Emitir(
		ctx, solicitud, decision, motivo, resultadoExacto, atestacion, prueba,
	)
	if err != nil {
		return decision, confirmacion, nil, nuevoErrorEmisionMaterialAutorizacionAtestadaV3(err, ctx.Err())
	}
	if err := ctx.Err(); err != nil {
		return decision, confirmacion, nil, nuevoErrorEmisionMaterialAutorizacionAtestadaV3(err)
	}
	raiz, err := e.confianza.raizPublicaParaPruebaV3(prueba)
	if err != nil {
		return decision, confirmacion, nil, nuevoErrorEmisionMaterialAutorizacionAtestadaV3(err)
	}
	if err := ctx.Err(); err != nil {
		return decision, confirmacion, nil, nuevoErrorEmisionMaterialAutorizacionAtestadaV3(err)
	}
	material, err := NuevoMaterialConsumoAutorizacionAtestadaV3(
		solicitud, decision, motivo, resultadoExacto,
		atestacion, prueba, capacidad, raiz,
	)
	if err != nil {
		return decision, confirmacion, nil, nuevoErrorEmisionMaterialAutorizacionAtestadaV3(err)
	}
	if err := ctx.Err(); err != nil {
		return decision, confirmacion, nil, nuevoErrorEmisionMaterialAutorizacionAtestadaV3(err)
	}
	return decision, confirmacion, material, nil
}

func validarConcesionEmisionMaterialV3(
	solicitud domain.SolicitudAutorizacionLigadaV3,
	decision domain.DecisionAutorizacionLigadaV3,
	confirmacion ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
	motivo domain.ReferenciaEntradaCatalogo,
	resultado domain.ResultadoContextoActorRegistradoV2,
) error {
	concedida, codigo, errResultado := decision.Resultado()
	orden, errOrden := ports.
		NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(
			solicitud, decision, motivo, resultado,
		)
	datosConfirmacion, errConfirmacion := confirmacion.Datos()
	if errResultado != nil || errOrden != nil || errConfirmacion != nil ||
		!concedida || codigo != "concedida" ||
		decision.ValidarPara(solicitud) != nil ||
		confirmacion.ValidarPara(orden) != nil ||
		!confirmacion.DentroDeVentanaEn(datosConfirmacion.RegistradaEn) {
		return errEmisionMaterialAutorizacionAtestadaV3NoDisponible
	}
	return nil
}

func nuevoErrorEmisionMaterialAutorizacionAtestadaV3(
	causas ...error,
) error {
	filtradas := []error{errEmisionMaterialAutorizacionAtestadaV3NoDisponible}
	for _, causa := range causas {
		switch {
		case errors.Is(causa, context.Canceled):
			filtradas = append(filtradas, context.Canceled)
		case errors.Is(causa, context.DeadlineExceeded):
			filtradas = append(filtradas, context.DeadlineExceeded)
		}
	}
	return errors.Join(filtradas...)
}
