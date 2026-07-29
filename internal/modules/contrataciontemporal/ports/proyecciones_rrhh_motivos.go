package ports

import (
	"context"
	"errors"
	"time"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// ErrMotivoConsultaRRHHNoDisponible oculta si falló la entrada, el catálogo,
// su vigencia o la fuente gobernada. El diagnóstico privado queda en el
// adaptador y nunca cruza este puerto.
var ErrMotivoConsultaRRHHNoDisponible = errors.New(
	"contratacion temporal: motivo de consulta RRHH no disponible",
)

// ResolutorMotivoConsultaRRHH obtiene del gobierno VEC la referencia exacta
// publicada para cada consulta. Los métodos son nominales para impedir que el
// llamador aporte una acción, finalidad, clave de entrada, organización o
// cualquier otro selector libre.
//
// Cada implementación debe rechazar contexto nulo o cancelado e instante no
// canónico en UTC. Ante cualquier error debe devolver la referencia cero y un
// error que conserve ErrMotivoConsultaRRHHNoDisponible. La referencia positiva
// debe ser una ReferenciaMotivoAutorizacionV2 válida y resuelta contra la
// publicación vigente, no una coordenada construida desde configuración libre.
//
// Una interfaz Go no puede imponer esas precondiciones ni detectar por sí sola
// una implementación nula tipada. La futura composición y el orquestador que
// consuma este puerto deberán cerrar ambas fronteras antes de usar el resultado.
type ResolutorMotivoConsultaRRHH interface {
	ResolverMotivoCuadroRRHH(
		context.Context,
		time.Time,
	) (dominiovec.ReferenciaEntradaCatalogo, error)
	ResolverMotivoDetalleRRHH(
		context.Context,
		time.Time,
	) (dominiovec.ReferenciaEntradaCatalogo, error)
}
