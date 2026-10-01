package informeliquidacion

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/dietas/domain"
)

func preparacionKilometrajeInforme(t *testing.T, cambiar func(*domain.DocumentoComision)) *domain.PreparacionLiquidacion {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(raiz, "cmd/vec-dietas/testdata/preparacion_liquidacion.json"))
	if err != nil {
		t.Fatal(err)
	}
	var e entradaGastosInforme
	if err := json.Unmarshal(b, &e); err != nil {
		t.Fatal(err)
	}
	primera := &e.Documento.Lineas[1]
	primera.Kilometros = "102.1250"
	primera.AjusteKilometros = "2.1250"
	primera.MotivoAjuste = "Desvío declarado en el trayecto de ida"
	segunda := *primera
	segunda.RutaIndice = 2
	segunda.OrigenCodigo, segunda.DestinoCodigo = primera.DestinoCodigo, primera.OrigenCodigo
	segunda.Kilometros, segunda.AjusteKilometros = "98.7500", "-1.2500"
	segunda.MotivoAjuste = "Recorrido de regreso más corto declarado"
	segunda.ImporteCentimos = 2500
	e.Documento.Lineas = append(e.Documento.Lineas, segunda)
	e.Documento.KilometrajeCentimos += segunda.ImporteCentimos
	e.Documento.TotalOrientativoCentimos += segunda.ImporteCentimos
	e.Revisiones[1].ReconocidoPropuestoCentimos = 2400
	e.Revisiones[1].MotivoCodigo = "revision_documental"
	e.Revisiones = append(e.Revisiones, domain.RevisionLineaLiquidacion{Indice: 2, ReglaRef: e.Revisiones[1].ReglaRef, ReconocidoPropuestoCentimos: 2500})
	if cambiar != nil {
		cambiar(&e.Documento)
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

func TestInformeKilometrajeDistingueTrayectosLocalizaSinCambiarPreparacion(t *testing.T) {
	r, err := Nuevo(Configuracion{TemaCSS: temaPrueba(t)})
	if err != nil {
		t.Fatal(err)
	}
	p := preparacionKilometrajeInforme(t, nil)
	original := p.Instantanea()
	for _, idioma := range []string{"es", "en"} {
		t.Run(idioma, func(t *testing.T) {
			textos := textosPrueba(t, idioma)
			b, err := r.Renderizar(p, textos)
			if err != nil {
				t.Fatal(err)
			}
			principal, _, _ := strings.Cut(string(b), "<details")
			for _, valor := range []string{textos.Rotulos.Ruta + " 1", textos.Rotulos.Ruta + " 2", textos.Rotulos.OrigenCodigo + ": 18087", textos.Rotulos.DestinoCodigo + ": 18153", textos.Rotulos.OrigenCodigo + ": 18153", textos.Rotulos.DestinoCodigo + ": 18087", textos.Rotulos.KilometrosBase + ": 100" + textos.Formato.Decimal + "0000", textos.Rotulos.KilometrosFinales + ": 102" + textos.Formato.Decimal + "1250", textos.Rotulos.AjusteKilometros + ": 2" + textos.Formato.Decimal + "1250", textos.Rotulos.AjusteKilometros + ": -1" + textos.Formato.Decimal + "2500", original.Documento.Lineas[1].MotivoAjuste, original.Documento.Lineas[2].MotivoAjuste, textos.Motivos["revision_documental"], moneda(original.Totales.OriginalCentimos, textos.Formato), moneda(original.Totales.ReconocidoPropuestoCentimos, textos.Formato), moneda(original.Totales.RechazadoCentimos, textos.Formato)} {
				if !strings.Contains(principal, valor) {
					t.Errorf("falta dato declarado/localizado %q", valor)
				}
			}
			if !reflect.DeepEqual(original, p.Instantanea()) {
				t.Fatal("el informe alteró datos, huellas o totales")
			}
			repetido, err := r.Renderizar(p, textos)
			if err != nil || !bytes.Equal(b, repetido) {
				t.Fatal("informe no determinista")
			}
			for _, prohibido := range []string{"<script", "<button", "<a ", "<form", "window.print", "fetch(", "<link"} {
				if strings.Contains(string(b), prohibido) {
					t.Errorf("acción o carga habilitada: %q", prohibido)
				}
			}
			if carpeta := os.Getenv("VEC_INFORME_KILOMETRAJE_EVIDENCIA"); carpeta != "" {
				if err := os.WriteFile(filepath.Join(carpeta, idioma+".html"), b, 0600); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestInformeKilometrajeCeroMotivoAusenteYEscape(t *testing.T) {
	r, err := Nuevo(Configuracion{TemaCSS: temaPrueba(t)})
	if err != nil {
		t.Fatal(err)
	}
	p := preparacionKilometrajeInforme(t, func(d *domain.DocumentoComision) {
		d.Lineas[1].KilometrosBase, d.Lineas[1].Kilometros = "100.0000", "100.0000"
		d.Lineas[1].AjusteKilometros, d.Lineas[1].MotivoAjuste = "0.0000", ""
		d.Lineas[2].MotivoAjuste = `<img src=x onerror=alert(1)> & motivo declarado`
	})
	for _, idioma := range []string{"es", "en"} {
		textos := textosPrueba(t, idioma)
		b, err := r.Renderizar(p, textos)
		if err != nil {
			t.Fatal(err)
		}
		s := string(b)
		for _, valor := range []string{textos.Rotulos.KilometrosBase + ": 100" + textos.Formato.Decimal + "0000", textos.Rotulos.KilometrosFinales + ": 100" + textos.Formato.Decimal + "0000", textos.Rotulos.MotivoAjuste + ": " + textos.Rotulos.NoConsta, textos.Rotulos.AjusteKilometros + ": 0" + textos.Formato.Decimal + "0000", "&lt;img src=x onerror=alert(1)&gt; &amp; motivo declarado"} {
			if !strings.Contains(s, valor) {
				t.Errorf("motivo ausente, cero o escape incorrecto: %q", valor)
			}
		}
		if strings.Contains(s, "<img") {
			t.Fatal("motivo declarado no escapado")
		}
	}
}

func TestInformeKilometrajeExigeTraduccionesSoloCuandoSeUsan(t *testing.T) {
	r, err := Nuevo(Configuracion{TemaCSS: temaPrueba(t)})
	if err != nil {
		t.Fatal(err)
	}
	textos := textosPrueba(t, "es")
	campos := []*string{&textos.Rotulos.Ruta, &textos.Rotulos.OrigenCodigo, &textos.Rotulos.DestinoCodigo, &textos.Rotulos.KilometrosBase, &textos.Rotulos.KilometrosFinales, &textos.Rotulos.AjusteKilometros, &textos.Rotulos.MotivoAjuste, &textos.Rotulos.NoConsta}
	for _, campo := range campos {
		original := *campo
		*campo = ""
		if b, err := r.Renderizar(preparacionKilometrajeInforme(t, nil), textos); err != ErrTextos || b != nil {
			t.Fatal("informe de kilometraje con rótulo ausente")
		}
		*campo = original
	}
	for _, campo := range campos {
		*campo = ""
	}
	b, err := json.Marshal(textos)
	if err != nil {
		t.Fatal(err)
	}
	anteriores, err := CargarTextos(b)
	if err != nil {
		t.Fatal("catálogo anterior de dietas rechazado")
	}
	if _, err := r.Renderizar(preparacionPrueba(t), anteriores); err != nil {
		t.Fatal("informe sin kilometraje exige rótulos nuevos")
	}
	textos.Rotulos.NoConsta = "\n"
	if _, err := r.Renderizar(preparacionPrueba(t), textos); err != ErrTextos {
		t.Fatal("rótulo opcional inválido admitido")
	}
}

func TestDistanciaMantieneDecimalExacto(t *testing.T) {
	for _, caso := range []struct{ idioma, entrada, salida string }{{"es", "1234.5678", "1.234,5678"}, {"en", "-1234.0001", "-1,234.0001"}, {"es", "texto.declarado", "texto.declarado"}, {"en", "0.0000", "0.0000"}} {
		if obtenido := distancia(caso.entrada, textosPrueba(t, caso.idioma)); obtenido != caso.salida {
			t.Errorf("%q: %q", caso.entrada, obtenido)
		}
	}
}
