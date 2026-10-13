package bootstrap

import (
	"slices"
	"strings"
	"testing"

	meritosports "vec-diputacion-granada/internal/modules/meritos/ports"
)

func TestMaterialCarteraExternaUsaUnaAudienciaPropia(t *testing.T) {
	const consumidor = consumidorMeritosCarteraPortalExternoV3
	if !slices.Contains(consumidoresPortalExternoV3, consumidor) ||
		!slices.Equal(audienciasConsumidorPortalExternoV3(consumidor), []string{meritosports.AudienciaRetirarCartera}) {
		t.Fatal("consumidor M4 no está cerrado a retirada propia")
	}
	d := descriptorMaterialRetirarCarteraExternaV3()
	if d.Audiencia != meritosports.AudienciaRetirarCartera || d.ProveedorNominal == "" ||
		!strings.HasPrefix(d.Dominio, "vec.meritos.cartera.") ||
		!strings.HasPrefix(d.Prefijo, "clave:capacidad:meritos-cartera-") ||
		!strings.HasSuffix(d.Prefijo, ":") {
		t.Fatalf("descriptor M4 inexacto: %+v", d)
	}
	presentes := 0
	for _, otro := range descriptoresMaterialPortalExternoV3() {
		if otro.Audiencia == d.Audiencia {
			presentes++
		}
		if otro.Audiencia != d.Audiencia && (otro.Dominio == d.Dominio ||
			strings.HasPrefix(otro.Prefijo, d.Prefijo) || strings.HasPrefix(d.Prefijo, otro.Prefijo)) {
			t.Fatalf("dominio/clave M4 coincide con %s", otro.Audiencia)
		}
	}
	if presentes != 1 {
		t.Fatalf("descriptor M4 en inventario: actual=%d esperado=1", presentes)
	}
}
