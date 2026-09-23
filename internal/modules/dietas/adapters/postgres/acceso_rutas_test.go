package postgres

import "testing"

func TestNuevoRepositorioAccesoRutasFallaCerradoSinPool(t *testing.T) {
	if r, e := NuevoRepositorioAccesoRutas(nil); r != nil || e == nil {
		t.Fatal("acepto pool ausente")
	}
}
