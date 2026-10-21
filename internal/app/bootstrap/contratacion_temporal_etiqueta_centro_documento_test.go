package bootstrap

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestEtiquetaDocumentalCentroCanonicoConservaReferencias(t *testing.T) {
	catalogo, err := nuevoCatalogoDesarrollo(
		"../../../data/catalogos/estructura-organizativa/v1.rpt-publica.json",
		"../../../data/catalogos/rpt/v1.rpt-2026.json",
	)
	if err != nil {
		t.Fatal(err)
	}
	origen := nuevoOrigenConsultasConCatalogoDesarrollo(catalogo)
	etiquetar := origen.etiquetasReferenciasCatalogosAlta()
	if nombre := etiquetar("centro-520"); nombre != "TRANSFORMACIÓN DIGITAL" {
		t.Fatalf("nombre canónico ausente: %q", nombre)
	}
	if etiquetar("centro-520") != etiquetar("centro:rpt:520") {
		t.Fatal("las dos presentaciones discrepan sobre la misma publicación")
	}
	if etiquetar("centro-99999") != "" {
		t.Fatal("se inventó una etiqueta para un centro desconocido")
	}
	// El nombre no convierte una clave de petición en una referencia de alta.
	if origen.centroDeCatalogo("centro-520") || !origen.centroDeCatalogo("centro:rpt:520") ||
		!origen.centroDeOrganizacionPeticion("centro-520") || origen.centroDeOrganizacionPeticion("centro:rpt:520") {
		t.Fatal("el etiquetado cambió las referencias admitidas por cada recorrido")
	}
	publicado, err := json.Marshal(catalogo)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(publicado), `"referencia":"centro-520"`) || strings.Contains(string(publicado), "centrosEtiquetasOrganizacion") {
		t.Fatal("la presentación publicó un alias de alta o su mapa interno")
	}
}

func TestEtiquetaDocumentalCentroSinOrganizacionNoInventaNombre(t *testing.T) {
	origen := nuevoOrigenConsultasContratacionTemporalDesarrollo()
	if origen.etiquetasReferenciasCatalogosAlta()("centro-520") != "" {
		t.Fatal("se dedujo un nombre sin su publicación de Organización")
	}
}
