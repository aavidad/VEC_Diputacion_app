package ports

import "testing"

func coberturaRepresentacionesSinMatrizBaremacionPrueba(cobertura uint16) bool {
	return cobertura&coberturaRepresentacionesIntencionBaremacion != 0 &&
		cobertura&coberturaMatrizBaremacion == 0
}

func TestCoberturaVistaBaremacionExigeMatrizOriginal(t *testing.T) {
	casos := []struct {
		nombre    string
		cobertura uint16
		valida    bool
	}{
		{
			nombre:    "producto base completo",
			cobertura: coberturaObligatoriaVistaBaremacion,
			valida:    true,
		},
		{
			nombre: "producto completo y representaciones derivadas",
			cobertura: coberturaObligatoriaVistaBaremacion |
				coberturaRepresentacionesIntencionBaremacion,
			valida: true,
		},
		{
			nombre: "representaciones no sustituyen la matriz original",
			cobertura: coberturaObligatoriaVistaBaremacion&^coberturaMatrizBaremacion |
				coberturaRepresentacionesIntencionBaremacion,
			valida: false,
		},
		{
			nombre:    "marca desconocida",
			cobertura: coberturaObligatoriaVistaBaremacion | 1<<15,
			valida:    false,
		},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if obtenida := coberturaVistaBaremacionValida(caso.cobertura); obtenida != caso.valida {
				t.Fatalf("validez obtenida=%t, esperada=%t, cobertura=%016b",
					obtenida, caso.valida, caso.cobertura)
			}
		})
	}
}
