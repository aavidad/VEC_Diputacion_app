package ports

import (
	"context"
	"errors"
	"reflect"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type generadorCorrelacionConsultaRRHH interface {
	NuevaReferenciaCorrelacionAutorizacionV2(context.Context) (string, error)
}

// EmisorMaterialConsultaRRHH fija la semántica antes de delegar en VEC-AD-3.
type EmisorMaterialConsultaRRHH struct {
	bloqueoSerializacionConsultaRRHH
	motivos       ResolutorMotivoConsultaRRHH
	correlaciones generadorCorrelacionConsultaRRHH
	reloj         Reloj
	emisores      emisoresMaterialConsultaRRHH
}

// NuevoEmisorMaterialConsultaRRHH exige autoridades completas y segregadas.
func NuevoEmisorMaterialConsultaRRHH(
	motivos ResolutorMotivoConsultaRRHH,
	correlaciones generadorCorrelacionConsultaRRHH,
	reloj Reloj,
	emisorCuadro emisorMaterialAutorizacionAtestadaV3,
	emisorDetalle emisorMaterialAutorizacionAtestadaV3,
) (*EmisorMaterialConsultaRRHH, error) {
	if dependenciaGuardianConsultaRRHHNula(motivos) ||
		dependenciaGuardianConsultaRRHHNula(correlaciones) ||
		dependenciaGuardianConsultaRRHHNula(reloj) {
		return nil, ErrCapacidadConsultaRRHHInvalida
	}
	emisores, err := nuevosEmisoresMaterialConsultaRRHH(
		emisorCuadro,
		emisorDetalle,
	)
	if err != nil {
		return nil, ErrCapacidadConsultaRRHHInvalida
	}
	return &EmisorMaterialConsultaRRHH{
		motivos: motivos, correlaciones: correlaciones,
		reloj: reloj, emisores: emisores,
	}, nil
}

// EmitirMaterialCuadroRRHH deriva recursos y semántica cerrados de cuadro.
func (e *EmisorMaterialConsultaRRHH) EmitirMaterialCuadroRRHH(
	ctx context.Context,
	contexto ContextoConsultaRRHH,
	solicitud SolicitudCuadroRRHH,
) (MaterialAutorizacionConsultaRRHH, error) {
	instante, err := e.iniciar(ctx)
	if err != nil {
		return MaterialAutorizacionConsultaRRHH{}, err
	}
	motivo, err := e.motivos.ResolverMotivoCuadroRRHH(ctx, instante)
	if err != nil || !dominiovec.ReferenciaMotivoAutorizacionV2Valida(motivo) {
		return MaterialAutorizacionConsultaRRHH{},
			errorEmisionMaterialConsultaRRHH(err, ctx.Err())
	}
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(
		ctx,
		e.correlaciones,
	)
	if err != nil {
		return MaterialAutorizacionConsultaRRHH{},
			errorEmisionMaterialConsultaRRHH(err, ctx.Err())
	}
	preparacion, err := prepararAutorizacionCuadroRRHH(
		contexto, solicitud, motivo, correlacion, instante,
	)
	if err != nil || ctx.Err() != nil {
		return MaterialAutorizacionConsultaRRHH{},
			errorEmisionMaterialConsultaRRHH(err, ctx.Err())
	}
	return e.emitir(ctx, preparacion, e.emisores.cuadro.emisor, instante)
}

// EmitirMaterialDetalleRRHH nunca reutiliza el emisor nominal de cuadro.
func (e *EmisorMaterialConsultaRRHH) EmitirMaterialDetalleRRHH(
	ctx context.Context,
	contexto ContextoConsultaRRHH,
	solicitud SolicitudDetalleRRHH,
) (MaterialAutorizacionConsultaRRHH, error) {
	instante, err := e.iniciar(ctx)
	if err != nil {
		return MaterialAutorizacionConsultaRRHH{}, err
	}
	motivo, err := e.motivos.ResolverMotivoDetalleRRHH(ctx, instante)
	if err != nil || !dominiovec.ReferenciaMotivoAutorizacionV2Valida(motivo) {
		return MaterialAutorizacionConsultaRRHH{},
			errorEmisionMaterialConsultaRRHH(err, ctx.Err())
	}
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(
		ctx,
		e.correlaciones,
	)
	if err != nil {
		return MaterialAutorizacionConsultaRRHH{},
			errorEmisionMaterialConsultaRRHH(err, ctx.Err())
	}
	preparacion, err := prepararAutorizacionDetalleRRHH(
		contexto, solicitud, motivo, correlacion, instante,
	)
	if err != nil || ctx.Err() != nil {
		return MaterialAutorizacionConsultaRRHH{},
			errorEmisionMaterialConsultaRRHH(err, ctx.Err())
	}
	return e.emitir(ctx, preparacion, e.emisores.detalle.emisor, instante)
}

func (e *EmisorMaterialConsultaRRHH) iniciar(ctx context.Context) (time.Time, error) {
	if ctx == nil || e == nil ||
		dependenciaGuardianConsultaRRHHNula(e.motivos) ||
		dependenciaGuardianConsultaRRHHNula(e.correlaciones) ||
		dependenciaGuardianConsultaRRHHNula(e.reloj) {
		return time.Time{}, errorEmisionMaterialConsultaRRHH(nil)
	}
	if err := ctx.Err(); err != nil {
		return time.Time{}, errorEmisionMaterialConsultaRRHH(err)
	}
	instante := e.reloj.Ahora()
	if !domain.InstanteUTCCanonico(instante) {
		return time.Time{}, errorEmisionMaterialConsultaRRHH(nil)
	}
	if err := ctx.Err(); err != nil {
		return time.Time{}, errorEmisionMaterialConsultaRRHH(err)
	}
	return instante, nil
}

func (e *EmisorMaterialConsultaRRHH) emitir(
	ctx context.Context,
	preparacion preparacionAutorizacionConsultaRRHH,
	emisor emisorMaterialAutorizacionAtestadaV3,
	instanteInicial time.Time,
) (MaterialAutorizacionConsultaRRHH, error) {
	decision, confirmacion, exportador, errEmision :=
		emisor.EmitirMaterialAutorizacionAtestadaV3(
			ctx, preparacion.solicitudVEC, preparacion.resultado,
		)
	if err := ctx.Err(); err != nil {
		return MaterialAutorizacionConsultaRRHH{},
			errorEmisionMaterialConsultaRRHH(errEmision, err)
	}
	instanteFinal := e.reloj.Ahora()
	if err := ctx.Err(); err != nil {
		return MaterialAutorizacionConsultaRRHH{},
			errorEmisionMaterialConsultaRRHH(errEmision, err)
	}
	if errEmision != nil || !domain.InstanteUTCCanonico(instanteFinal) ||
		instanteFinal.Before(instanteInicial) ||
		preparacion.validarEn(instanteFinal) != nil {
		return MaterialAutorizacionConsultaRRHH{},
			errorEmisionMaterialConsultaRRHH(errEmision)
	}
	material, err := nuevoMaterialAutorizacionConsultaRRHH(
		preparacion.contexto, preparacion.solicitudVEC,
		decision, confirmacion, preparacion.resultado,
		exportador, instanteFinal,
	)
	if err != nil || ctx.Err() != nil {
		return MaterialAutorizacionConsultaRRHH{},
			errorEmisionMaterialConsultaRRHH(err, ctx.Err())
	}
	return material, nil
}

func dependenciaGuardianConsultaRRHHNula(dependencia any) bool {
	if dependencia == nil {
		return true
	}
	valor := reflect.ValueOf(dependencia)
	switch valor.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
		reflect.Pointer, reflect.Slice:
		return valor.IsNil()
	default:
		return false
	}
}

func errorEmisionMaterialConsultaRRHH(causas ...error) error {
	publicos := []error{ErrConsultaRRHHNoDisponible}
	for _, causa := range causas {
		switch {
		case errors.Is(causa, context.Canceled):
			publicos = append(publicos, context.Canceled)
		case errors.Is(causa, context.DeadlineExceeded):
			publicos = append(publicos, context.DeadlineExceeded)
		}
	}
	return errors.Join(publicos...)
}
