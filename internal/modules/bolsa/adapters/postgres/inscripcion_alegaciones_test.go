package postgres

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/application/inscripcion"
)

func TestMetadatoB7SoloSobrePrivadoDelObjetoAutorizado(t *testing.T) {
	a := inscripcion.Alegacion{AlegacionRef: "alegacion_" + strings.Repeat("a", 64),
		Objeto: inscripcion.ObjetoAlegacion{Tipo: "acto_participacion", ActoRef: "acta_b7_1", ParticipacionRef: "participacion_b7_1", VersionActo: 1}}
	m := inscripcion.MetadatoNombreAlegacionB7{ActaRef: a.Objeto.ActoRef, ParticipacionRef: a.Objeto.ParticipacionRef,
		HuellaListadoSHA256: strings.Repeat("b", 64), CategoriaRef: "categoria_b7_1", FilaNumero: 2}
	pagina := inscripcion.PaginaAlegaciones{Alegaciones: []inscripcion.Alegacion{a}, Total: 1}
	if err := adjuntarMetadatosB7Pagina(&pagina, map[string]inscripcion.MetadatoNombreAlegacionB7{a.AlegacionRef: m}); err != nil || pagina.Alegaciones[0].MetadatoNombreB7 == nil {
		t.Fatalf("sobre autorizado: %v", err)
	}
	publico, err := json.Marshal(pagina)
	if err != nil || bytes.Contains(publico, []byte(m.HuellaListadoSHA256)) || bytes.Contains(publico, []byte(m.CategoriaRef)) || bytes.Contains(publico, []byte("fila_numero")) {
		t.Fatal("metadato privado salió por JSON")
	}
	m.ActaRef = "acta_ajena"
	if adjuntarMetadatosB7Pagina(&pagina, map[string]inscripcion.MetadatoNombreAlegacionB7{a.AlegacionRef: m}) == nil {
		t.Fatal("acta ajena aceptada")
	}
	if adjuntarMetadatosB7Pagina(&pagina, map[string]inscripcion.MetadatoNombreAlegacionB7{"alegacion_ajena": m}) == nil {
		t.Fatal("fila ajena aceptada")
	}
}
