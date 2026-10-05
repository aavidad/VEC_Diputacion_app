package informeperiodo

import (
	"encoding/csv"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	app "vec-diputacion-granada/internal/modules/dietas/application/informeperiodo"
)

func catalogo(t *testing.T, idioma string) CatalogoCSV {
	t.Helper()
	f, err := os.Open("../../../../../web/static/textos/" + idioma + "/dietas-informes-csv.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	c, err := CargarCatalogo(f)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func informe(persona string) app.Informe {
	return app.Informe{CampoFecha: "fecha_liquidacion", Moneda: "EUR", Conceptos: []string{"manutencion", "kilometraje"},
		ConceptosCentimos: []int64{4250, 0}, TotalCentimos: 4250,
		Filas: []app.Fila{{Referencia: "DI-002", VersionComision: 2, Situacion: "liquidado", Persona: persona,
			Unidad: "Cultura", Fecha: time.Date(2026, 9, 6, 0, 0, 0, 0, time.UTC),
			ConceptosCentimos: []int64{4250, 0}, ImporteIncluidoCentimos: 4250}}}
}

func leer(t *testing.T, b []byte) [][]string {
	t.Helper()
	filas, err := csv.NewReader(strings.NewReader(string(b))).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	return filas
}

func TestCatalogosEsEnTienenLasMismasClaves(t *testing.T) {
	es, en := catalogo(t, "es"), catalogo(t, "en")
	for _, par := range [][2]map[string]string{{es.Cabeceras, en.Cabeceras}, {es.CamposFecha, en.CamposFecha},
		{es.Conceptos, en.Conceptos}, {es.Situaciones, en.Situaciones}} {
		if len(par[0]) != len(par[1]) {
			t.Fatal("claves distintas")
		}
		for k := range par[0] {
			if _, ok := par[1][k]; !ok {
				t.Fatalf("falta %s en inglés", k)
			}
		}
	}
	if es.Idioma != "es-ES" || en.Idioma != "en-GB" {
		t.Fatal(es.Idioma, en.Idioma)
	}
}

func TestEscribeColumnasDeLaSeleccionConFechaDelCatalogo(t *testing.T) {
	b, err := Escribir(catalogo(t, "es"), informe("Luis Pérez"))
	if err != nil {
		t.Fatal(err)
	}
	filas := leer(t, b)
	if len(filas) != 2 || strings.Join(filas[0][:7], "|") != "Referencia|Versión|Persona|Unidad|Situación del ejemplo|Fecha de liquidación|Manutención (céntimos)" ||
		len(filas[0]) != 10 {
		t.Fatalf("cabecera %q", filas[0])
	}
	if strings.Join(filas[1][:9], "|") != "DI-002|2|Luis Pérez|Cultura|Liquidado|06/09/2026|4250|0|4250" {
		t.Fatalf("fila %q", filas[1])
	}
	if strings.Contains(string(b), "persona-demo") || strings.Contains(string(b), "unidad-demo") {
		t.Fatal("el CSV no debe llevar referencias opacas")
	}
}

func TestNeutralizaFormulasEnTextos(t *testing.T) {
	b, err := Escribir(catalogo(t, "en"), informe(`=HYPERLINK("http://x")`))
	if err != nil {
		t.Fatal(err)
	}
	if got := leer(t, b)[1][2]; got != `'=HYPERLINK("http://x")` {
		t.Fatalf("persona %q", got)
	}
}

func TestRechazaRotulosAusentes(t *testing.T) {
	c := catalogo(t, "es")
	inf := informe("Luis Pérez")
	inf.Conceptos[1] = "peajes"
	if _, err := Escribir(c, inf); !errors.Is(err, ErrCatalogoInvalido) {
		t.Fatal(err)
	}
	inf = informe("Luis Pérez")
	inf.Filas[0].Situacion = "pagado"
	if _, err := Escribir(c, inf); !errors.Is(err, ErrCatalogoInvalido) {
		t.Fatal(err)
	}
	for _, roto := range []string{`{}`, `{"esquema":"cronos-saldo-csv-v1"}`, strings.Repeat(" ", 70000)} {
		if _, err := CargarCatalogo(strings.NewReader(roto)); !errors.Is(err, ErrCatalogoInvalido) {
			t.Fatal(err)
		}
	}
}
