package main

import (
	"context"
	"testing"

	"vec-diputacion-granada/internal/vec/adapters/fichero"
	"vec-diputacion-granada/internal/vec/reglas"
)

// El publicador CT190 y el arranque han de poder leer exactamente los mismos
// tres paquetes. Esta prueba usa el lector de producción y el parser tipado.
func TestCatalogosDeOperacionSonProvisionables(t *testing.T) {
	casos := []struct {
		ruta, id          string
		version, entradas int
	}{
		{"../../../data/catalogos/contratacion-temporal/reglas.v2.json", reglas.CatalogoContratacionTemporal, 2, 36},
		{"../../../data/catalogos/bolsa/reglas.v6.json", reglas.CatalogoBolsa, 6, 50},
		{"../../../data/catalogos/contratacion-temporal/circuito_firma.v2.json", reglas.CatalogoCircuitoFirmaCT, 2, 11},
	}
	for _, tc := range casos {
		t.Run(tc.id, func(t *testing.T) {
			fuente, err := fichero.NuevaConsultaCatalogoProvisionable(tc.ruta)
			if err != nil {
				t.Fatalf("lector: %v", err)
			}
			c, err := fuente.ObtenerCatalogo(context.Background(), tc.id, tc.version)
			if err != nil {
				t.Fatalf("versión: %v", err)
			}
			if len(c.Entradas) != tc.entradas || c.FuenteRef == reglas.MarcaPaqueteEjemplo {
				t.Fatalf("catálogo/entradas inesperados: %s/%d", c.FuenteRef, len(c.Entradas))
			}
			if err := reglas.ValidarCatalogoBaseReglas(c); err != nil {
				t.Fatalf("reglas tipadas: %v", err)
			}
		})
	}
}
