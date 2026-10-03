package contrastecopias

import (
	"context"
	"errors"
	d "vec-diputacion-granada/internal/modules/administracion/domain/contrastecopias"
	p "vec-diputacion-granada/internal/modules/administracion/ports/contrastecopias"
)

type Servicio struct{ Lector p.Lector }

var ErrCaptura = errors.New("copias_contraste_error_captura")

// Capturar evita propagar errores del proveedor que puedan contener secretos.
func (s Servicio) Capturar(ctx context.Context) (d.Snapshot, error) {
	if s.Lector == nil {
		return d.Snapshot{}, ErrCaptura
	}
	i, err := s.Lector.Capturar(ctx)
	if err != nil {
		return d.Snapshot{}, ErrCaptura
	}
	return i, nil
}

func (Servicio) Comparar(a, b d.Snapshot) d.Resultado { return d.Comparar(a, b) }
