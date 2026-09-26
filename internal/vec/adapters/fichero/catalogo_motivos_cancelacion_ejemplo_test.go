package fichero

import (
	"context"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

// El catálogo de motivos de cancelación de ejemplo se carga con el adaptador
// real; cada motivo lleva su clave i18n, la marca de ejemplo y los canales
// (centro, RRHH) que pueden usarlo. Cambiar motivos o canales no toca código.
func TestCatalogoMotivosCancelacionEjemploCargaConAdaptadorReal(t *testing.T) {
	consulta, err := NuevaConsultaCatalogos("../../../../data/demo/reglas/ct_motivos_cancelacion.demo.json")
	if err != nil {
		t.Fatalf("el adaptador rechaza el paquete: %v", err)
	}
	metadatos, err := consulta.ObtenerMetadatosFuenteCatalogos(context.Background())
	if err != nil || !metadatos.Demostracion {
		t.Fatalf("el paquete debe declararse de demostración: %+v %v", metadatos, err)
	}
	catalogo, err := consulta.ObtenerCatalogo(context.Background(), "motivos_cancelacion_contratacion_temporal", 1)
	if err != nil {
		t.Fatal(err)
	}
	if catalogo.ModuloID != "contratacion_temporal" || catalogo.FuenteRef != "paquete:ejemplo:vec:v1" ||
		catalogo.Estado != domain.EstadoCatalogoPublicado || len(catalogo.Entradas) < 4 {
		t.Fatalf("catálogo inesperado: %s %s %s %d", catalogo.ModuloID, catalogo.FuenteRef, catalogo.Estado, len(catalogo.Entradas))
	}
	instante := time.Date(2026, 9, 28, 9, 0, 0, 0, time.UTC)
	centro, rrhh := 0, 0
	for _, entrada := range catalogo.Entradas {
		a := entrada.Atributos
		if !entrada.VigenteEn(instante) ||
			a["clave_i18n"] != "contratacion_temporal.cancelacion.motivo."+entrada.Clave ||
			a["origen"] != "ejemplo" || a["norma"] == "" || a["canales"] == "" {
			t.Errorf("motivo %s incompleto: %v", entrada.Clave, a)
		}
		for _, canal := range strings.Split(a["canales"], ",") {
			switch canal {
			case "centro":
				centro++
			case "rrhh":
				rrhh++
			default:
				t.Errorf("motivo %s con canal desconocido %q", entrada.Clave, canal)
			}
		}
	}
	if centro == 0 || rrhh == 0 {
		t.Fatalf("cada canal necesita al menos un motivo: centro=%d rrhh=%d", centro, rrhh)
	}
}
