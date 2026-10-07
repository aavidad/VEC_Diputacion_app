package contrastecopias

import (
	"context"
	"errors"
	"testing"
	d "vec-diputacion-granada/internal/modules/administracion/domain/contrastecopias"
)

type lectorFallido struct{}

func (lectorFallido) Capturar(context.Context) (d.Snapshot, error) {
	return d.Snapshot{}, errors.New("postgres://secreto@ruta_privada")
}

func TestErroresProveedorSeOcultan(t *testing.T) {
	for _, s := range []Servicio{{}, {Lector: lectorFallido{}}} {
		_, err := s.Capturar(context.Background())
		if err != ErrCaptura {
			t.Fatalf("error no nominal: %v", err)
		}
	}
}
