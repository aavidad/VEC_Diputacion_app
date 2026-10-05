package informeliquidacion

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/dietas/application/preparacionliquidacion"
)

func comparacionPrueba(t *testing.T) *preparacionliquidacion.ComparacionLiquidacion {
	t.Helper()
	datos, err := os.ReadFile(filepath.Join(raiz, "cmd/vec-dietas/testdata/preparacion_liquidacion_gastos.json"))
	if err != nil {
		t.Fatal(err)
	}
	var entrada preparacionliquidacion.Entrada
	if err := json.Unmarshal(datos, &entrada); err != nil {
		t.Fatal(err)
	}
	anterior, err := preparacionliquidacion.Preparar(entrada)
	if err != nil {
		t.Fatal(err)
	}
	entrada.Revisiones[2].ReconocidoPropuestoCentimos = 1800
	entrada.Revisiones[2].MotivoCodigo = ""
	propuesta, err := preparacionliquidacion.Preparar(entrada)
	if err != nil {
		t.Fatal(err)
	}
	comparacion, err := preparacionliquidacion.Comparar(preparacionliquidacion.EntradaComparacion{
		Esquema:  preparacionliquidacion.EsquemaComparacionLiquidacion,
		Anterior: anterior.Instantanea(), Propuesta: propuesta.Instantanea(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return comparacion
}

func textosComparacionPrueba(t *testing.T, idioma string) TextosComparacion {
	t.Helper()
	datos, err := os.ReadFile(filepath.Join(raiz, "web/static/textos", idioma, "dietas-comparacion-liquidacion-informe.json"))
	if err != nil {
		t.Fatal(err)
	}
	textos, err := CargarTextosComparacion(datos)
	if err != nil {
		t.Fatal(err)
	}
	return textos
}

func TestComparacionHTMLLocalizadaConImportesYHuellasConservados(t *testing.T) {
	r, err := Nuevo(Configuracion{TemaCSS: temaPrueba(t)})
	if err != nil {
		t.Fatal(err)
	}
	c := comparacionPrueba(t)
	for _, idioma := range []string{"es", "en"} {
		t.Run(idioma, func(t *testing.T) {
			textos := textosComparacionPrueba(t, idioma)
			b, err := r.RenderizarComparacion(c, textos)
			if err != nil {
				t.Fatal(err)
			}
			s := string(b)
			for _, esperado := range []string{
				`<html lang="` + idioma + `">`, textos.Rotulos["estado"], textos.Rotulos["limite"],
				textos.Rotulos["rechazos"], textos.Rotulos["criterios"], textos.Motivos["revision_justificante"],
				c.Anterior.CatalogoSHA256, c.Propuesta.CatalogoSHA256,
				c.Anterior.SnapshotSHA256, c.Propuesta.SnapshotSHA256, c.DocumentoSHA256,
				`scope="row"`, `tabindex="0"`, `@media print`,
			} {
				if !strings.Contains(s, esperado) {
					t.Errorf("falta contenido %q", esperado)
				}
			}
			if strings.Count(s, "<table>") != 3 {
				t.Fatal("faltan las tablas separadas de importes, rechazo y criterios")
			}
			if aviso, primerCatalogo := strings.Index(s, `class="limite-huella">`),
				strings.Index(s, "<h3>"+textos.Rotulos["anterior"]+"</h3>"); aviso < 0 || primerCatalogo < 0 || aviso > primerCatalogo {
				t.Fatal("el aviso de huellas quedó separado de los catálogos")
			}
			for _, centimos := range []int64{6670, 5970, 6270, 700, 400, 300, -300} {
				if !strings.Contains(s, moneda(centimos, textos.Formato)) {
					t.Errorf("falta importe %d", centimos)
				}
			}
			if strings.Contains(s, "revision_justificante") || strings.Contains(s, "comparacion_local_sin_registrar") || strings.Contains(s, "<script") {
				t.Fatal("código o script visible en el informe")
			}
			otra, err := r.RenderizarComparacion(c, textos)
			if err != nil || !bytes.Equal(b, otra) {
				t.Fatal("renderizado inestable", err)
			}
		})
	}
}

func TestComparacionHTMLEscapaDatosYRechazaCatalogoAmbiguo(t *testing.T) {
	r, err := Nuevo(Configuracion{TemaCSS: temaPrueba(t)})
	if err != nil {
		t.Fatal(err)
	}
	c := comparacionPrueba(t)
	c.Lineas[0].ReglaAnteriorRef = `<img src=x onerror=alert(1)>`
	b, err := r.RenderizarComparacion(c, textosComparacionPrueba(t, "es"))
	if err != nil || bytes.Contains(b, []byte(`<img`)) || !bytes.Contains(b, []byte(`&lt;img`)) {
		t.Fatal("regla sin escapar", err)
	}
	datos, err := os.ReadFile(filepath.Join(raiz, "web/static/textos/es/dietas-comparacion-liquidacion-informe.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, alterado := range [][]byte{
		bytes.Replace(datos, []byte(`"titulo":`), []byte(`"titulo":"duplicado","titulo":`), 1),
		bytes.Replace(datos, []byte(`"catalogos_iguales":`), []byte(`"otra":"ajena","catalogos_iguales":`), 1),
		bytes.Replace(datos, []byte(`"catalogos_iguales":`), []byte(`"catalogos_iguales":null,"catalogos_iguales":`), 1),
	} {
		if _, err := CargarTextosComparacion(alterado); err != ErrTextos {
			t.Fatal("catálogo ambiguo aceptado", err)
		}
	}
	textos := textosComparacionPrueba(t, "es")
	delete(textos.Motivos, "revision_justificante")
	if b, err := r.RenderizarComparacion(comparacionPrueba(t), textos); err != ErrTextos || b != nil {
		t.Fatal("motivo sin traducción produjo informe", err)
	}
}
