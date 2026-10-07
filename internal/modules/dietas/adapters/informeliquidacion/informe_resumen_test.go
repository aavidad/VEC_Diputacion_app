package informeliquidacion

import (
	"html"
	"math"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/dietas/domain"
)

// Leer la tabla emitida comprueba el resultado visible, no sólo el acumulador.
func comprobarResumen(t *testing.T, b []byte, textos Textos, esperados [4][3]int64, totales domain.TotalesLiquidacionPropuesta) {
	t.Helper()
	_, resto, ok := strings.Cut(string(b), `<section class="panel" aria-labelledby="resumen">`)
	if !ok {
		t.Fatal("falta resumen")
	}
	resumen, _, _ := strings.Cut(resto, "</section>")
	filas := regexp.MustCompile(`<tr><th scope="row">([^<]*)</th><td class="importe">([^<]*)</td><td class="importe">([^<]*)</td><td class="importe">([^<]*)</td></tr>`).FindAllStringSubmatch(resumen, -1)
	if len(filas) != 5 {
		t.Fatalf("filas de resumen y total: %d", len(filas))
	}
	var suma [3]int64
	for i, fila := range filas {
		rotulo := textos.Rotulos.Total
		if i < 4 {
			rotulo = textos.Familias[familiasInforme[i]]
		}
		if html.UnescapeString(fila[1]) != rotulo {
			t.Errorf("rótulo de fila %d: %s", i, fila[1])
		}
		for j := 0; j < 3; j++ {
			valor := html.UnescapeString(fila[j+2])
			valor = strings.TrimSuffix(valor, "\u00a0"+textos.Formato.Moneda)
			valor = strings.ReplaceAll(valor, textos.Formato.Agrupacion, "")
			valor = strings.ReplaceAll(valor, textos.Formato.Decimal, "")
			n, err := strconv.ParseInt(valor, 10, 64)
			if err != nil {
				t.Fatal(err)
			}
			if i < 4 {
				if n != esperados[i][j] {
					t.Errorf("familia %d columna %d: %d, esperado %d", i, j, n, esperados[i][j])
				}
				suma[j] += n
			} else if n != suma[j] || n != [3]int64{totales.OriginalCentimos, totales.ReconocidoPropuestoCentimos, totales.RechazadoCentimos}[j] {
				t.Errorf("total de columna %d no coincide: %d / %d", j, n, suma[j])
			}
		}
	}
}

func TestResumenFamiliasExactoConDetalleYHuellasConservados(t *testing.T) {
	r, err := Nuevo(Configuracion{TemaCSS: temaPrueba(t)})
	if err != nil {
		t.Fatal(err)
	}
	for _, caso := range []struct {
		nombre    string
		p         *domain.PreparacionLiquidacion
		esperados [4][3]int64
	}{
		{"cuatro_familias_y_rechazo_total", preparacionGastosInforme(t, ""), [4][3]int64{{1870, 1870, 0}, {2600, 2600, 0}, {1800, 1500, 300}, {400, 0, 400}}},
		{"varios_trayectos_y_familias_ausentes", preparacionKilometrajeInforme(t, nil), [4][3]int64{{1870, 1870, 0}, {5100, 4900, 200}, {}, {}}},
		{"solo_manutencion", preparacionPrueba(t), [4][3]int64{{3200, 3000, 200}, {}, {}, {}}},
	} {
		for _, idioma := range []string{"es", "en"} {
			t.Run(caso.nombre+"_"+idioma, func(t *testing.T) {
				antes := caso.p.Instantanea()
				textos := textosPrueba(t, idioma)
				b, err := r.Renderizar(caso.p, textos)
				if err != nil {
					t.Fatal(err)
				}
				comprobarResumen(t, b, textos, caso.esperados, antes.Totales)
				_, detalle, ok := strings.Cut(string(b), `<section class="panel" aria-labelledby="conceptos">`)
				if !ok {
					t.Fatal("falta detalle original")
				}
				for _, linea := range antes.Lineas {
					for _, importe := range []int64{linea.OriginalCentimos, linea.ReconocidoPropuestoCentimos, linea.RechazadoCentimos} {
						if !strings.Contains(detalle, moneda(importe, textos.Formato)) {
							t.Fatal("importe omitido del detalle")
						}
					}
				}
				for _, huella := range []string{antes.DocumentoSHA256, antes.CatalogoSHA256, antes.SnapshotSHA256} {
					if !strings.Contains(detalle, huella) {
						t.Fatal("huella omitida")
					}
				}
				if !reflect.DeepEqual(antes, caso.p.Instantanea()) {
					t.Fatal("preparación alterada")
				}
			})
		}
	}
}

func TestResumenConservaCentimosHastaMaxInt64(t *testing.T) {
	s := preparacionPrueba(t).Instantanea()
	primero := int64(math.MaxInt64 / 2)
	segundo := int64(math.MaxInt64) - primero
	s.Documento.Lineas[0].ImporteCentimos = primero
	l := s.Documento.Lineas[0]
	indice := 1
	l.IndiceTramo = &indice
	l.ImporteCentimos = segundo
	s.Documento.Lineas = append(s.Documento.Lineas, l)
	s.Documento.TramosAceptados = []int{0, 1}
	s.Documento.ManutencionCentimos, s.Documento.TotalOrientativoCentimos = math.MaxInt64, math.MaxInt64
	s.Catalogo.Reglas[0].TopeCentimos = math.MaxInt64
	sha, err := domain.HuellaDatosLiquidacion(s.Documento)
	if err != nil {
		t.Fatal(err)
	}
	p, err := domain.PrepararLiquidacion(s.ComisionRef, s.ComisionVersion, sha, s.Documento, s.Catalogo, []domain.RevisionLineaLiquidacion{
		{Indice: 0, ReglaRef: s.Catalogo.Reglas[0].Referencia, ReconocidoPropuestoCentimos: primero},
		{Indice: 1, ReglaRef: s.Catalogo.Reglas[0].Referencia, ReconocidoPropuestoCentimos: segundo},
	})
	if err != nil {
		t.Fatal(err)
	}
	r, err := Nuevo(Configuracion{TemaCSS: temaPrueba(t)})
	if err != nil {
		t.Fatal(err)
	}
	for _, idioma := range []string{"es", "en"} {
		textos := textosPrueba(t, idioma)
		b, err := r.Renderizar(p, textos)
		if err != nil {
			t.Fatal(err)
		}
		comprobarResumen(t, b, textos, [4][3]int64{{math.MaxInt64, math.MaxInt64, 0}, {}, {}, {}}, p.Instantanea().Totales)
	}
}

func TestResumenTraduccionesObligatoriasYEscape(t *testing.T) {
	r, err := Nuevo(Configuracion{TemaCSS: temaPrueba(t)})
	if err != nil {
		t.Fatal(err)
	}
	for _, cambiar := range []func(*Textos){
		func(t *Textos) { t.Rotulos.Resumen = "" },
		func(t *Textos) { t.Rotulos.Familia = "" },
		func(t *Textos) { t.Rotulos.ResumenAyuda = "" },
		func(t *Textos) { delete(t.Familias, domain.ClaseOtroMedio) },
		func(t *Textos) { t.Familias["manutencion"] = "\n" },
	} {
		textos := textosPrueba(t, "es")
		cambiar(&textos)
		if b, err := r.Renderizar(preparacionPrueba(t), textos); err != ErrTextos || b != nil {
			t.Fatal("resumen sin traducción válida")
		}
	}
	textos := textosPrueba(t, "es")
	textos.Familias["manutencion"] = `<img src=x onerror=alert(1)> & familia`
	textos.Rotulos.Resumen = `<script>alert(1)</script>`
	b, err := r.Renderizar(preparacionPrueba(t), textos)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "<img") || strings.Contains(string(b), "<script") || !strings.Contains(string(b), "&lt;img src=x onerror=alert(1)&gt; &amp; familia") {
		t.Fatal("resumen sin escapar")
	}
}
