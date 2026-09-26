package nodisponible

import (
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/seleccion/ports"
)

func TestNuncaInformaExitoAparente(t *testing.T) {
	s, ctx, p := Servicios{}, context.Background(), ports.PresentacionExterna{}
	for _, err := range []error{s.FirmarPresentacion(ctx, p), s.RegistrarPresentacion(ctx, p), s.ComprobarTasa(ctx, p), s.NotificarPresentacion(ctx, p)} {
		if !errors.Is(err, ports.ErrServicioExternoNoDisponible) {
			t.Fatal("un servicio sin proveedor debe responder no disponible")
		}
	}
}
