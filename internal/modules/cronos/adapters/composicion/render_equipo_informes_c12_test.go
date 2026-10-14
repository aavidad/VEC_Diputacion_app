package composicion

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

func TestRenderInformeC12CSVConservaDesconocidoYNeutralizaFormula(t *testing.T) {
	r, err := NuevoRenderizadorInformeC12(CatalogoInformeC12{Idioma: "es-ES", Titulo: "Informe", Desconocido: "Sin dato", Etiquetas: map[string]string{"estado": "=Estado"}})
	if err != nil {
		t.Fatal(err)
	}
	b, err := r.RenderizarInforme(context.Background(), domain.InformeCSV, ports.PreparacionInformeC12{Campos: []string{"estado"}, Datos: [][]json.RawMessage{{json.RawMessage(`"=1+1"`)}, {json.RawMessage(`null`)}, {json.RawMessage(`42`)}}})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b, []byte("'=Estado\n'=1+1\nSin dato\n42\n")) {
		t.Fatalf("CSV no neutralizado: %q", b)
	}
}

func TestRenderInformeC12PDFUsaValidadorComun(t *testing.T) {
	r, err := NuevoRenderizadorInformeC12(CatalogoInformeC12{Idioma: "es-ES", Titulo: "Informe", Desconocido: "Sin dato", Etiquetas: map[string]string{"estado": "Estado"}})
	if err != nil {
		t.Fatal(err)
	}
	b, err := r.RenderizarInforme(context.Background(), domain.InformePDF, ports.PreparacionInformeC12{Campos: []string{"estado"}, Datos: [][]json.RawMessage{{json.RawMessage(`null`)}}})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(b, []byte("%PDF-")) || !bytes.HasSuffix(bytes.TrimSpace(b), []byte("%%EOF")) {
		t.Fatal("PDF inválido")
	}
}
