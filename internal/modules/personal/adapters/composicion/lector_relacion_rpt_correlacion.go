package composicion

import (
	"context"

	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

type claveCorrelacionLectorRelacionRPT struct{}

// LectorRelacionSeleccionadaRPT conserva la única operación que consume la
// preparación RPT. El decorador abarca también los rechazos anteriores al servicio.
type LectorRelacionSeleccionadaRPT interface {
	ConsultarSeleccionada(context.Context, vecdomain.ContextoActor, personaldomain.PreparacionRelacionParaRPT, string, string) (personalports.ResultadoRelacionParaRPTV1, error)
}

type lectorRelacionSeleccionadaCorrelacion struct {
	siguiente LectorRelacionSeleccionadaRPT
}

func (l lectorRelacionSeleccionadaCorrelacion) ConsultarSeleccionada(ctx context.Context, actor vecdomain.ContextoActor, preparacion personaldomain.PreparacionRelacionParaRPT, organismo, seleccion string) (personalports.ResultadoRelacionParaRPTV1, error) {
	var vacio personalports.ResultadoRelacionParaRPTV1
	if ctx == nil || dependenciaNula(l.siguiente) {
		return vacio, personaldomain.ErrLectorRelacionRPTNoDisponible
	}
	// La referencia se acuña aquí, antes de cualquier validación del lector.
	// WithoutCancel permite conservar la auditoría de un contexto cancelado.
	ref, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(context.WithoutCancel(ctx), seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return vacio, personaldomain.ErrLectorRelacionRPTNoDisponible
	}
	return l.siguiente.ConsultarSeleccionada(context.WithValue(ctx, claveCorrelacionLectorRelacionRPT{}, ref), actor, preparacion, organismo, seleccion)
}

// La clave privada impide aceptar una correlación libre del cliente. Una llamada
// sin la marca creada por el decorador falla cerrada, sin crear otra referencia.
func correlacionLectorRelacionRPT(ctx context.Context) (vecdomain.ReferenciaCorrelacionAutorizacionV2, error) {
	if ctx == nil {
		return vecdomain.ReferenciaCorrelacionAutorizacionV2{}, personaldomain.ErrLectorRelacionRPTNoDisponible
	}
	ref, ok := ctx.Value(claveCorrelacionLectorRelacionRPT{}).(vecdomain.ReferenciaCorrelacionAutorizacionV2)
	if !ok || ref.Validar() != nil {
		return vecdomain.ReferenciaCorrelacionAutorizacionV2{}, personaldomain.ErrLectorRelacionRPTNoDisponible
	}
	return ref, nil
}

var _ LectorRelacionSeleccionadaRPT = lectorRelacionSeleccionadaCorrelacion{}
