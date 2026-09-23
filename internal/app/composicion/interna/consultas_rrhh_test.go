package interna

import (
	"errors"
	"testing"
)

func TestLecturasRRHHSinPoolNominalNoPublicanRutas(t *testing.T) {
	rutas, err := nuevasRutasConsultasRRHH(dependenciasLecturasRRHH{})
	if rutas != nil || !errors.Is(err, ErrDependenciasProductivasNoDisponibles) {
		t.Fatalf("lecturas CT sin pool = (%v, %v)", rutas, err)
	}
}
