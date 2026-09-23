package application

import "testing"

func TestNuevoServicioAutorizacionRutasDietasFallaCerradoSinDependencias(t *testing.T) {
	if s, e := NuevoServicioAutorizacionRutasDietas(nil, nil, nil); s != nil || e == nil {
		t.Fatal("acepto dependencias ausentes")
	}
}
