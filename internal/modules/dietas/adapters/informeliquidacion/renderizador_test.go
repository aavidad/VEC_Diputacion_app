package informeliquidacion

import (
	"bytes"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/dietas/domain"
)

const raiz = "../../../../../"

func textosPrueba(t *testing.T, idioma string) Textos {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(raiz, "web/static/textos", idioma, "dietas-liquidacion-informe.json"))
	if err != nil {
		t.Fatal(err)
	}
	c, err := CargarTextos(b)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func temaPrueba(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(raiz, "web/static/comun/tema-vec.css"))
	if err != nil {
		t.Fatal(err)
	}
	// Usar la base canónica, sin duplicar tokens ni los temas no seleccionados.
	inicio := strings.Index(string(b), ":root {")
	fin := strings.Index(string(b)[inicio:], "}") + inicio + 1
	return string(b)[inicio:fin]
}

func preparacionPrueba(t *testing.T) *domain.PreparacionLiquidacion {
	t.Helper()
	indice := 0
	d := domain.DocumentoComision{GrupoDieta: 1, VersionTarifaAceptada: "tarifa:ejemplo", TramosAceptados: []int{0}, Lineas: []domain.LineaDocumentoComision{
		{Tipo: "dieta", Grupo: 1, IndiceTramo: &indice, Fecha: "2026-10-01", Concepto: "manutencion", ImporteCentimos: 3200, VersionTarifaRef: "tarifa:ejemplo"},
	}, ManutencionCentimos: 3200, TotalOrientativoCentimos: 3200}
	c := domain.CatalogoLiquidacionPropuesta{Referencia: "catalogo:sintetico", Version: "version:v1", VersionTarifaRef: "tarifa:ejemplo", PaisISO2: "ES", Ambito: "nacional_ordinario_sin_alojamiento", Procedencia: "ejemplo_retirable_sin_aprobacion", Fuentes: []string{"https://www.boe.es/buscar/act.php?id=BOE-A-2002-10337"}, Reglas: []domain.ReglaLiquidacionPropuesta{
		{Referencia: "regla:manutencion", Tipo: "dieta", Concepto: "manutencion", Grupo: 1, TopeCentimos: 3000},
	}}
	dSHA, err := domain.HuellaDatosLiquidacion(d)
	if err != nil {
		t.Fatal(err)
	}
	p, err := domain.PrepararLiquidacion("dco_ejemplo_sintetico_20261001", 3, dSHA, d, c, []domain.RevisionLineaLiquidacion{{Indice: 0, ReglaRef: "regla:manutencion", ReconocidoPropuestoCentimos: 3000, MotivoCodigo: "revision_documental"}})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestInformeLocalizadoSeguroConPreparacionReal(t *testing.T) {
	r, err := Nuevo(Configuracion{TemaCSS: temaPrueba(t)})
	if err != nil {
		t.Fatal(err)
	}
	p := preparacionPrueba(t)
	for _, idioma := range []string{"es", "en"} {
		t.Run(idioma, func(t *testing.T) {
			textos := textosPrueba(t, idioma)
			b, err := r.Renderizar(p, textos)
			if err != nil {
				t.Fatal(err)
			}
			s := string(b)
			if strings.Index(s, p.Instantanea().ComisionRef) < strings.Index(s, "<details") {
				t.Fatal("referencia técnica en la vista principal")
			}
			for _, esperado := range []string{`<html lang="` + idioma + `">`, textos.Rotulos.Estado, textos.Rotulos.Limite, `scope="col"`, `scope="row"`, `tabindex="0"`, p.Instantanea().SnapshotSHA256} {
				if !strings.Contains(s, esperado) {
					t.Errorf("falta %q", esperado)
				}
			}
			for _, n := range []string{"32" + textos.Formato.Decimal + "00", "30" + textos.Formato.Decimal + "00", "2" + textos.Formato.Decimal + "00"} {
				if !strings.Contains(s, n) {
					t.Errorf("falta total %q", n)
				}
			}
			for _, prohibido := range []string{"<script", "<link", "@import", "url(", "revision_documental", "revision_tarifa_importada", "nacional_ordinario_sin_alojamiento"} {
				if strings.Contains(s, prohibido) {
					t.Errorf("salida prohibida %q", prohibido)
				}
			}
			otra, err := r.Renderizar(p, textos)
			if err != nil || !bytes.Equal(b, otra) {
				t.Fatal("renderizado no determinista")
			}
			if carpeta := os.Getenv("VEC_INFORME_EVIDENCIA"); carpeta != "" {
				if err := os.WriteFile(filepath.Join(carpeta, idioma+".html"), b, 0600); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestRechazaPreparacionNulaOVaciaYTextosIncompletos(t *testing.T) {
	r, err := Nuevo(Configuracion{TemaCSS: temaPrueba(t)})
	if err != nil {
		t.Fatal(err)
	}
	textos := textosPrueba(t, "es")
	for _, p := range []*domain.PreparacionLiquidacion{nil, {}} {
		if b, err := r.Renderizar(p, textos); err != ErrPreparacion || b != nil {
			t.Fatal("aceptó preparación inválida")
		}
	}
	delete(textos.Motivos, "revision_documental")
	if b, err := r.Renderizar(preparacionPrueba(t), textos); err != ErrTextos || b != nil {
		t.Fatal("aceptó motivo sin traducir")
	}
}

func TestTemaSinCargaExternaNiCierreDeStyle(t *testing.T) {
	for _, css := range []string{"", `</style><script>alert(1)</script>`, `@import 'https://ejemplo.invalid/x';`, `p { background: url (https://ejemplo.invalid); }`, `@\69mport 'x';`} {
		if _, err := Nuevo(Configuracion{TemaCSS: css}); err != ErrTema {
			t.Fatal("aceptó CSS inseguro")
		}
	}
}

func TestMonedaEnteraYCatalogos(t *testing.T) {
	for _, c := range []struct{ idioma, grande, minimo string }{{"es", "92.233.720.368.547.758,07\u00a0€", "-92.233.720.368.547.758,08\u00a0€"}, {"en", "92,233,720,368,547,758.07\u00a0€", "-92,233,720,368,547,758.08\u00a0€"}} {
		textos := textosPrueba(t, c.idioma)
		if moneda(math.MaxInt64, textos.Formato) != c.grande || moneda(math.MinInt64, textos.Formato) != c.minimo {
			t.Fatal("redondeó moneda mediante flotantes")
		}
	}
	for _, b := range [][]byte{nil, []byte(`{}`), []byte(`null`), []byte(`{} {}`)} {
		if _, err := CargarTextos(b); err != ErrTextos {
			t.Fatal("aceptó catálogo incompleto")
		}
	}
}

func TestEscapeDeTextosInyectados(t *testing.T) {
	r, err := Nuevo(Configuracion{TemaCSS: temaPrueba(t)})
	if err != nil {
		t.Fatal(err)
	}
	textos := textosPrueba(t, "es")
	textos.Rotulos.Estado = "<script>alert(1)</script> & revisión"
	b, err := r.Renderizar(preparacionPrueba(t), textos)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "<script>") || !strings.Contains(string(b), "&lt;script&gt;alert(1)&lt;/script&gt; &amp; revisión") {
		t.Fatal("texto inyectado sin escapar")
	}
}
