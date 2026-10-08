package bootstrap

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestContextoCentroPublicaSoloSeccionesCompuestas(t *testing.T) {
	id, _ := escenarioIdentidadCentroPrueba(t, "capacidades-centro")
	p := &proveedorPeticionCentroDesarrollo{actores: map[string]*identidadPeticionCentroDesarrollo{id.principal.ID: id}, catalogos: catalogoCentroPrueba{}, reloj: id.soporte.reloj}
	m := &manejadorPeticionCentroDesarrollo{proveedor: p, catalogo: &catalogosAltaContratacionTemporalDesarrollo{}}
	for _, caso := range []struct {
		nombre                             string
		incorporacion, cancelacion, perfil bool
		esperaI, esperaC                   bool
	}{
		{"sin_montaje", false, false, false, false, false},
		{"solo_incorporaciones", true, false, false, true, false},
		{"cancelacion_sin_perfil", true, true, false, true, false},
		{"ambas", true, true, true, true, true},
		{"sin_bandeja", false, true, true, false, false},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			p.incorporacionesCompuestas = caso.incorporacion
			p.cancelacionesCompuestas = caso.cancelacion
			id.cancelacion = nil
			if caso.perfil {
				id.cancelacion = &perfilCentroDesarrollo{}
			}
			r := httptest.NewRequest(http.MethodGet, rutaContextoPeticionCentro, nil).WithContext(contextoRutaCentroPrueba(id, rutaContextoPeticionCentro))
			data, err := m.contexto(r, id)
			if err != nil {
				t.Fatal(err)
			}
			caps := data.(map[string]any)["capacidades"].(map[string]bool)
			if len(caps) != 2 || caps["incorporaciones"] != caso.esperaI || caps["cancelaciones"] != caso.esperaC {
				t.Fatalf("capacidades=%v, esperadas incorporaciones=%v cancelaciones=%v", caps, caso.esperaI, caso.esperaC)
			}
		})
	}
}
