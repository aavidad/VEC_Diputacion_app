package firmasct

import (
	"context"
	"path/filepath"
	"testing"
)

func TestEtiquetasOriginalR5UsaPublicacionesYPierdeUnidadAusente(t *testing.T) {
	raiz := filepath.Join("..", "..", "..", "..", "data", "catalogos")
	organizacion := filepath.Join(raiz, "estructura-organizativa", "v1.rpt-publica.json")
	rpt := filepath.Join(raiz, "rpt", "v1.rpt-2026.json")
	etiquetar, err := CargarEtiquetasOriginalR5(context.Background(), organizacion, rpt)
	if err != nil {
		t.Fatalf("publicaciones: %v", err)
	}
	if etiquetar("centro:rpt:101") != "GABINETE DE PRESIDENCIA" ||
		etiquetar("categoria:rpt:ctpd-auxilar-de-enfermaria") != "CTPD/AUXILAR DE ENFERMARÍA" {
		t.Fatal("nombres de centro o categoría no proceden de los catálogos publicados")
	}
	if etiquetar("unidad:desarrollo:rrhh") != "" {
		t.Fatal("unidad ausente recibió un nombre inventado")
	}
}
