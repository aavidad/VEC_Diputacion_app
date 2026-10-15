package separacionportales

import "testing"

func TestMaterialCarteraYAlegacionesPerteneceSoloAlExterno(t *testing.T) {
	for _, archivo := range []string{"identidad/meritos-cartera-externa.json", "identidad/bolsa-alegaciones.json"} {
		t.Run(archivo, func(t *testing.T) {
			externo := materialSintetico(t, PortalExterno)
			escribir(t, externo, archivo, `{"version":1}`)
			if err := ComprobarMaterial(PortalExterno, externo); err != nil {
				t.Fatal(err)
			}
			interno := materialSintetico(t, PortalInterno)
			escribir(t, interno, archivo, `{"version":1}`)
			if err := ComprobarMaterial(PortalInterno, interno); err == nil || motivo(t, err) != archivo {
				t.Fatal("el material del aspirante fue admitido en el portal interno")
			}
		})
	}
	if clasificar("identidad/meritos-cartera-desconocida.json") != pertenenciaInterna {
		t.Fatal("se amplió la lista de material externo por parecido de nombre")
	}
}
