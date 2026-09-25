package fichero

import (
	"context"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

// El catálogo de causas de cese de ejemplo se carga con el adaptador real y
// cada causa lleva su clave i18n, la marca de ejemplo y el justificante que
// se pide al registrarla. Cambiar causas o justificantes no toca código.
func TestCatalogoCausasCeseEjemploCargaConAdaptadorReal(t *testing.T) {
	consulta, err := NuevaConsultaCatalogos("../../../../data/demo/reglas/ct_causas_cese.demo.json")
	if err != nil {
		t.Fatalf("el adaptador rechaza el paquete: %v", err)
	}
	metadatos, err := consulta.ObtenerMetadatosFuenteCatalogos(context.Background())
	if err != nil || !metadatos.Demostracion {
		t.Fatalf("el paquete debe declararse de demostración: %+v %v", metadatos, err)
	}
	catalogo, err := consulta.ObtenerCatalogo(context.Background(), "causas_cese_contratacion_temporal", 1)
	if err != nil {
		t.Fatal(err)
	}
	if catalogo.ModuloID != "contratacion_temporal" || catalogo.FuenteRef != "paquete:ejemplo:vec:v1" ||
		catalogo.Estado != domain.EstadoCatalogoPublicado || len(catalogo.Entradas) < 5 {
		t.Fatalf("catálogo inesperado: %s %s %s %d", catalogo.ModuloID, catalogo.FuenteRef, catalogo.Estado, len(catalogo.Entradas))
	}
	instante := time.Date(2026, 9, 28, 9, 0, 0, 0, time.UTC)
	for _, entrada := range catalogo.Entradas {
		a := entrada.Atributos
		if !entrada.VigenteEn(instante) ||
			a["clave_i18n"] != "contratacion_temporal.seguimiento.cese.causa."+entrada.Clave ||
			a["origen"] != "ejemplo" || a["norma"] == "" || a["justificante"] == "" {
			t.Errorf("causa %s incompleta: %v", entrada.Clave, a)
		}
	}
}
