package usecases

import (
	"context"
	"errors"
)

// ErrContextRequired impide sustituir silenciosamente la cancelacion o la
// ausencia del contexto que delimita una operacion administrativa.
var ErrContextRequired = errors.New("candidate usecase: context is required")

// validateContext aplica cierre por defecto: un contexto ausente, cancelado o
// vencido nunca habilita lecturas ni escrituras en los puertos de salida.
func validateContext(ctx context.Context) error {
	if ctx == nil {
		return ErrContextRequired
	}
	return ctx.Err()
}
