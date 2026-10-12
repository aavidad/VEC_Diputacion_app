package bootstrap

import (
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/reglas"
)

func TestComposicionReglasProvisionablesSinMarcaDeDemostracion(t *testing.T) {
	reloj := relojReglasEjemploPrueba{ahora: time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)}
	casos := []struct {
		ruta, id, modulo string
		entradas         int
	}{
		{"../../../data/catalogos/contratacion-temporal/reglas.v2.json", reglas.CatalogoContratacionTemporal, reglas.ModuloContratacionTemporal, 36},
		{"../../../data/catalogos/bolsa/reglas.v6.json", reglas.CatalogoBolsa, reglas.ModuloBolsa, 50},
		{"../../../data/catalogos/contratacion-temporal/circuito_firma.v2.json", reglas.CatalogoCircuitoFirmaCT, reglas.ModuloContratacionTemporal, 11},
	}
	for _, tc := range casos {
		t.Run(tc.id, func(t *testing.T) {
			resolutor, err := nuevoResolutorReglasEjemplo(tc.ruta, tc.id, tc.modulo, nil, reloj)
			if err != nil {
				t.Fatalf("composición: %v", err)
			}
			vigentes, err := resolutor.Reglas(t.Context())
			if err != nil || len(vigentes) != tc.entradas {
				t.Fatalf("reglas vigentes: %d/%d, %v", len(vigentes), tc.entradas, err)
			}
			for _, regla := range vigentes {
				if regla.PaqueteEjemplo || (regla.Origen == reglas.OrigenConfiguracion && regla.EsEjemplo()) {
					t.Fatalf("marca de demostración en %s", regla.Clave)
				}
			}
		})
	}
}
