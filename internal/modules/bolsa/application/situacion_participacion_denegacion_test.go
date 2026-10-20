package application

import (
	"context"
	"errors"
	"testing"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// F1-60: la denegación que el emisor atestado ya registró llega unida a «no
// disponible». Es una denegación (403) y no una caída de la dependencia (503);
// cualquier otro fallo sigue siendo indisponibilidad.
func TestErrorDependenciaSituacionReconoceLaDenegacionRegistrada(t *testing.T) {
	registrada := errors.Join(errors.New("emisión no disponible"), vecports.ErrDenegacionExplicitaAutorizacionLigadaV3)
	if err := errorDependenciaSituacion(registrada); !errors.Is(err, dominiovec.ErrAutorizacionDenegada) {
		t.Fatalf("denegación registrada: %v", err)
	}
	for _, err := range []error{nil, errors.New("caída"), context.Canceled} {
		if got := errorDependenciaSituacion(err); !errors.Is(got, ErrCambioSituacionParticipacionNoDisponible) {
			t.Fatalf("%v: %v", err, got)
		}
	}
}
