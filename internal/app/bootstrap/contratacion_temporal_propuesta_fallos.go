package bootstrap

import (
	"context"
	"errors"
	"log/slog"

	bp "vec-diputacion-granada/internal/modules/bolsa/ports"
	pgct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// La etapa y la causa pertenecen a listas cerradas. No se registra Error()
// del error recibido ni su envoltorio, material, actor o solicitud.
func registrarFalloPropuestaCT(ctx context.Context, etapa string, err error) {
	switch etapa {
	case "dependencias", "preimagen", "antecedente", "fuentes_nativas",
		"resultado_emision", "actor_anclaje", "consulta_anclaje", "confirmacion":
	default:
		etapa = "desconocida"
	}
	causa := "no_clasificado"
	if err == nil {
		causa = "comprobacion_rechazada"
	}
	for _, centinela := range []error{
		context.Canceled, context.DeadlineExceeded,
		ct.ErrAutorizacionDenegada, ct.ErrConsultaRRHHNoDisponible,
		ct.ErrResultadoPropuestaFormalizacionNoConfiable,
		ct.ErrOperacionPropuestaFormalizacionDenegada,
		ct.ErrResolucionLlamamientoNoAceptada,
		pgct.ErrPersistenciaPropuestaFormalizacionNoDisponible,
		application.ErrPropuestaFormalizacionNoDisponible,
		application.ErrResultadoPropuestaFormalizacionNoConfiable,
		application.ErrPropuestaFormalizacionDenegada,
		bp.ErrResultadoEmisionCTNoDisponible, bp.ErrResultadoEmisionCTNoEncontrado,
		bp.ErrResultadoEmisionCTDivergente, bp.ErrResultadoEmisionCTDenegado,
		bp.ErrAceptacionNativaCTInvalida, bp.ErrAceptacionNativaCTDenegada,
		bp.ErrAceptacionNativaCTNoDisponible,
	} {
		if errors.Is(err, centinela) {
			causa = centinela.Error()
			break
		}
	}
	slog.ErrorContext(ctx, "contratación: propuesta no completada",
		"operacion", "propuesta_formalizacion", "etapa", etapa, "causa_interna", causa)
}
