package informeliquidacion

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/dietas/domain"
)

type entradaGastosInforme struct {
	ComisionRef     string                              `json:"comision_ref"`
	ComisionVersion int64                               `json:"comision_version"`
	Documento       domain.DocumentoComision            `json:"documento"`
	Catalogo        domain.CatalogoLiquidacionPropuesta `json:"catalogo"`
	Revisiones      []domain.RevisionLineaLiquidacion   `json:"revisiones"`
}

func preparacionGastosInforme(t *testing.T, descripcion string) *domain.PreparacionLiquidacion {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(raiz, "cmd/vec-dietas/testdata/preparacion_liquidacion_gastos.json"))
	if err != nil {
		t.Fatal(err)
	}
	var e entradaGastosInforme
	if err := json.Unmarshal(b, &e); err != nil {
		t.Fatal(err)
	}
	if descripcion != "" {
		e.Documento.Lineas[2].Concepto = descripcion
	}
	sha, err := domain.HuellaDatosLiquidacion(e.Documento)
	if err != nil {
		t.Fatal(err)
	}
	p, err := domain.PrepararLiquidacion(e.ComisionRef, e.ComisionVersion, sha, e.Documento, e.Catalogo, e.Revisiones)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestInformeGastosD5LocalizadoEscapadoYDetalleTecnico(t *testing.T) {
	r, err := Nuevo(Configuracion{TemaCSS: temaPrueba(t)})
	if err != nil {
		t.Fatal(err)
	}
	const descripcion = `<img src=x onerror=alert(1)> & tren`
	p := preparacionGastosInforme(t, descripcion)
	visual := preparacionGastosInforme(t, "")
	for _, idioma := range []string{"es", "en"} {
		t.Run(idioma, func(t *testing.T) {
			textos := textosPrueba(t, idioma)
			fechaEsperada := "01/10/2026"
			if idioma == "en" {
				fechaEsperada = "01 Oct 2026"
			}
			for _, tipo := range domain.CatalogoOtrosGastosVigente().Tipos {
				if textos.TiposGasto[tipo.Codigo] == "" {
					t.Errorf("falta traducción de tipo %q", tipo.Codigo)
				}
			}
			b, err := r.Renderizar(p, textos)
			if err != nil {
				t.Fatal(err)
			}
			s := string(b)
			principal, detalle, ok := strings.Cut(s, "<details")
			if !ok {
				t.Fatal("falta detalle técnico")
			}
			for _, esperado := range []string{`<html lang="` + idioma + `">`, textos.TiposGasto["tren"], textos.TiposGasto["peaje"], textos.Rotulos.DescripcionDeclarada, textos.Rotulos.FechaGasto, fechaEsperada, "&lt;img src=x onerror=alert(1)&gt; &amp; tren", moneda(0, textos.Formato), textos.Motivos["gasto_no_admitido"], textos.Motivos["revision_justificante"]} {
				if !strings.Contains(principal, esperado) {
					t.Errorf("falta en el cuadro %q", esperado)
				}
			}
			for _, prohibido := range []string{"<img", "otro_medio", "otro_gasto", "revision_justificante", "gasto_no_admitido", "justificante:tren:ejemplo", strings.Repeat("a", 64)} {
				if strings.Contains(principal, prohibido) {
					t.Errorf("dato técnico o HTML sin escapar en el cuadro %q", prohibido)
				}
			}
			for _, esperado := range []string{textos.Rotulos.CatalogoOtrosGastos, textos.Rotulos.TopeLinea, moneda(2000, textos.Formato), moneda(500, textos.Formato), textos.Rotulos.JustificanteRef, "justificante:tren:ejemplo", textos.Rotulos.JustificanteHuella, strings.Repeat("a", 64), textos.Rotulos.JustificanteLimite} {
				if !strings.Contains(detalle, esperado) {
					t.Errorf("falta en el detalle %q", esperado)
				}
			}
			if strings.Contains(s, descripcion) || strings.Contains(s, "<script") || strings.Contains(s, "<link") {
				t.Fatal("HTML inyectado o carga externa")
			}
			if carpeta := os.Getenv("VEC_INFORME_GASTOS_EVIDENCIA"); carpeta != "" {
				visualHTML, err := r.Renderizar(visual, textos)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(carpeta, idioma+".html"), visualHTML, 0600); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestInformeAnteriorAdmiteCatalogoSinCamposD5(t *testing.T) {
	r, err := Nuevo(Configuracion{TemaCSS: temaPrueba(t)})
	if err != nil {
		t.Fatal(err)
	}
	textos := textosPrueba(t, "es")
	textos.TiposGasto = nil
	textos.Rotulos.FechaGasto = ""
	textos.Rotulos.DescripcionDeclarada = ""
	textos.Rotulos.JustificanteRef = ""
	textos.Rotulos.JustificanteHuella = ""
	textos.Rotulos.CatalogoOtrosGastos = ""
	textos.Rotulos.TopeLinea = ""
	textos.Rotulos.JustificanteLimite = ""
	b, err := json.Marshal(textos)
	if err != nil {
		t.Fatal(err)
	}
	antiguos, err := CargarTextos(b)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Renderizar(preparacionPrueba(t), antiguos); err != nil {
		t.Fatalf("el informe anterior dejó de ser compatible: %v", err)
	}
}

func TestInformeGastosD5RechazaTraduccionAusente(t *testing.T) {
	r, err := Nuevo(Configuracion{TemaCSS: temaPrueba(t)})
	if err != nil {
		t.Fatal(err)
	}
	p := preparacionGastosInforme(t, "")
	for _, cambiar := range []func(*Textos){
		func(textos *Textos) { delete(textos.TiposGasto, "tren") },
		func(textos *Textos) { delete(textos.Motivos, "gasto_no_admitido") },
		func(textos *Textos) { textos.Rotulos.JustificanteLimite = "" },
	} {
		textos := textosPrueba(t, "es")
		cambiar(&textos)
		if b, err := r.Renderizar(p, textos); err != ErrTextos || b != nil {
			t.Fatalf("admitió un informe D5 sin traducción: %v", err)
		}
	}
}
