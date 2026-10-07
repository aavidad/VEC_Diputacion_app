package informeperiodo

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	app "vec-diputacion-granada/internal/modules/dietas/application/informeperiodo"
	"vec-diputacion-granada/internal/vec/adapters/documentos/pdf"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

// rendererEspia conserva el contenido para comprobar los párrafos exactos.
type rendererEspia struct {
	contenido vecdomain.ContenidoDocumento
	idioma    string
}

func (*rendererEspia) Formato() vecdomain.FormatoDocumento { return vecdomain.FormatoDocumentoPDF }
func (r *rendererEspia) Renderizar(_ context.Context, c vecdomain.ContenidoDocumento) ([]byte, error) {
	r.contenido = c
	idioma := r.idioma
	if idioma == "" {
		idioma = "es-ES"
	}
	return []byte("%PDF-espia /Lang (" + idioma + ")"), nil
}
func (*rendererEspia) ValidarSalida(context.Context, []byte) error { return nil }

func catalogoPDF(t *testing.T, idioma string) CatalogoPDF {
	t.Helper()
	raw, err := os.ReadFile("../../../../../web/static/textos/" + idioma + "/dietas-informes-pdf.json")
	if err != nil {
		t.Fatal(err)
	}
	c, err := CargarCatalogoPDF(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func informePDF(persona string) app.Informe {
	inf := informe(persona)
	desde := time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC)
	hasta := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)
	inf.Desde, inf.Hasta = &desde, &hasta
	inf.Filas[0].ImporteIncluidoCentimos = 123456
	inf.TotalCentimos = 123456
	return inf
}

func TestPDFComponeResumenConceptosYFilasDelCatalogo(t *testing.T) {
	espia := &rendererEspia{}
	if _, err := PrepararPDF(context.Background(), espia, catalogoPDF(t, "es"), informePDF("Luis Pérez")); err != nil {
		t.Fatal(err)
	}
	p := espia.contenido.Parrafos
	if espia.contenido.Titulo != "Informe de dietas por período" || !strings.HasPrefix(p[0], "EJEMPLO SINTÉTICO") ||
		!strings.Contains(p[1], "fecha de liquidación") || p[2] != "Período consultado: del 05/09/2026 al 20/09/2026." ||
		p[3] != "Informes incluidos: 1. Importe incluido: 1.234,56 €." || p[4] != "Manutención: 42,50 €." ||
		p[5] != "Kilometraje: 0,00 €." ||
		p[6] != "DI-002 (versión 2). Luis Pérez, Cultura. Liquidado, 06/09/2026. Importe incluido: 1.234,56 €." ||
		!strings.HasPrefix(p[7], "Los importes son los que conservó") || len(p) != 8 {
		t.Fatalf("%q", p)
	}
}

func TestPDFNoInterpretaDatosComoPlantilla(t *testing.T) {
	espia := &rendererEspia{}
	if _, err := PrepararPDF(context.Background(), espia, catalogoPDF(t, "es"), informePDF("{{importe}} {{persona}}")); err != nil {
		t.Fatal(err)
	}
	if fila := espia.contenido.Parrafos[6]; !strings.Contains(fila, "{{importe}} {{persona}}, Cultura") {
		t.Fatalf("%q", fila)
	}
}

func TestPDFRealEsYEnLlevaIdioma(t *testing.T) {
	for _, idioma := range []string{"es", "en"} {
		c := catalogoPDF(t, idioma)
		inf := informePDF("Luis Pérez")
		inf.Desde = nil
		contenido, err := PrepararPDF(context.Background(), pdf.Renderizador{Idioma: c.Idioma}, c, inf)
		if err != nil {
			t.Fatal(idioma, err)
		}
		if !bytes.HasPrefix(contenido, []byte("%PDF-")) || !bytes.Contains(contenido, []byte("/Lang ("+c.Idioma+")")) {
			t.Fatal(idioma)
		}
	}
}

func TestPDFRechazaCatalogoIncompletoOMonedaSinRotulo(t *testing.T) {
	es := catalogoPDF(t, "es")
	inf := informePDF("Luis Pérez")
	inf.Moneda = "USD"
	if _, err := PrepararPDF(context.Background(), &rendererEspia{}, es, inf); !errors.Is(err, ErrCatalogoInvalido) {
		t.Fatal(err)
	}
	inf = informePDF("Luis Pérez")
	inf.Filas[0].Situacion = "pagado"
	if _, err := PrepararPDF(context.Background(), &rendererEspia{}, es, inf); !errors.Is(err, ErrCatalogoInvalido) {
		t.Fatal(err)
	}
	raw, err := os.ReadFile("../../../../../web/static/textos/es/dietas-informes-pdf.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, roto := range []string{
		strings.Replace(string(raw), "{{importe}}.\",\n  \"concepto\"", ".\",\n  \"concepto\"", 1),
		strings.Replace(string(raw), "{{referencia}} (versión", "{{referencia}} {{referencia}} (versión", 1),
		strings.Replace(string(raw), `"separador_decimal": ","`, `"separador_decimal": ";"`, 1),
		strings.Replace(string(raw), `"dietas-informes-pdf-v1"`, `"dietas-informes-csv-v1"`, 1),
	} {
		if roto == string(raw) {
			t.Fatal("alteración no aplicada")
		}
		if _, err := CargarCatalogoPDF(strings.NewReader(roto)); !errors.Is(err, ErrCatalogoInvalido) {
			t.Fatal(err)
		}
	}
}

func TestCatalogosPDFEsEnTienenLasMismasClaves(t *testing.T) {
	es, en := catalogoPDF(t, "es"), catalogoPDF(t, "en")
	for _, par := range [][2]map[string]string{{es.CamposFecha, en.CamposFecha}, {es.Conceptos, en.Conceptos},
		{es.Situaciones, en.Situaciones}, {es.Importes, en.Importes}} {
		if len(par[0]) != len(par[1]) {
			t.Fatal("claves distintas")
		}
		for k := range par[0] {
			if _, ok := par[1][k]; !ok {
				t.Fatalf("falta %s en inglés", k)
			}
		}
	}
}

func TestPDFInglesFormateaImportesYVersion(t *testing.T) {
	espia := &rendererEspia{idioma: "en-GB"}
	inf := informePDF("Luis Pérez")
	inf.Filas[0].VersionComision = 1000
	if _, err := PrepararPDF(context.Background(), espia, catalogoPDF(t, "en"), inf); err != nil {
		t.Fatal(err)
	}
	p := espia.contenido.Parrafos
	if p[3] != "Reports included: 1. Included amount: €1,234.56." ||
		p[6] != "DI-002 (version 1000). Luis Pérez, Cultura. Settled, 06/09/2026. Included amount: €1,234.56." {
		t.Fatalf("%q", p)
	}
}

func TestPDFRechazaImportesNegativosYSeparadorAjenoAlIdioma(t *testing.T) {
	inf := informePDF("Luis Pérez")
	inf.Filas[0].ImporteIncluidoCentimos = -1256
	if _, err := PrepararPDF(context.Background(), &rendererEspia{}, catalogoPDF(t, "es"), inf); !errors.Is(err, app.ErrDatosInvalidos) {
		t.Fatal(err)
	}
	raw, err := os.ReadFile("../../../../../web/static/textos/es/dietas-informes-pdf.json")
	if err != nil {
		t.Fatal(err)
	}
	roto := strings.Replace(string(raw), `"separador_decimal": ","`, `"separador_decimal": "."`, 1)
	if _, err := CargarCatalogoPDF(strings.NewReader(roto)); !errors.Is(err, ErrCatalogoInvalido) {
		t.Fatal(err)
	}
}
