package bootstrap

import (
	"context"
	"os"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

func TestLectoresVisualesRRHHEligenTripleSinReescribirLegado(t *testing.T) {
	contenido, err := os.ReadFile("../../../config/contratacion_temporal_flujo_visual_rrhh_v1.json")
	if err != nil {
		t.Fatal(err)
	}
	legado, err := LeerLectorFlujoVisualRRHH(strings.NewReader(string(contenido)))
	if err != nil {
		t.Fatal(err)
	}
	nuevo := *legado
	nuevo.origen = domain.ReferenciaFlujo{
		DefinicionRef: "flujo:ct:rrhh:prueba", Version: 2,
		HuellaSHA256: strings.Repeat("a", 64),
	}
	lectores, err := NuevoLectorFlujosVisualesRRHH(legado, &nuevo)
	if err != nil {
		t.Fatal(err)
	}
	for _, flujo := range []domain.ReferenciaFlujo{legado.origen, nuevo.origen} {
		if _, err := lectores.Resolver(context.Background(), flujo, "solicitud"); err != nil {
			t.Fatalf("triple %v: %v", flujo, err)
		}
	}
	if _, err := lectores.Resolver(context.Background(), domain.ReferenciaFlujo{
		DefinicionRef: nuevo.origen.DefinicionRef, Version: 3, HuellaSHA256: nuevo.origen.HuellaSHA256,
	}, "solicitud"); err == nil {
		t.Fatal("presentó un flujo con otra versión")
	}
	if _, err := NuevoLectorFlujosVisualesRRHH(legado, legado); err == nil {
		t.Fatal("aceptó dos manifiestos para un mismo triple")
	}
}
