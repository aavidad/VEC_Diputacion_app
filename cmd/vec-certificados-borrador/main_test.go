package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/certificados/domain"
)

func argumentos(t *testing.T, idioma, destino string) []string {
	t.Helper()
	return []string{"-ensayo-sintetico", "-fuente", "../../internal/modules/certificados/adapters/fichero/testdata/servicios.ensayo.json",
		"-indice-idiomas", "../../web/static/textos/idiomas.json", "-plantilla", "../../data/certificados/plantillas/servicios.v1.json", "-idioma", idioma,
		"-textos", "../../web/static/textos/" + idioma + "/certificados.json", "-salida", destino}
}
func TestCLIProducePDFYJSONDeterministasEnDosIdiomas(t *testing.T) {
	for _, idioma := range []string{"es", "en"} {
		t.Run(idioma, func(t *testing.T) {
			raiz := t.TempDir()
			var anterior []byte
			for _, nombre := range []string{"first", "second"} {
				d := filepath.Join(raiz, nombre)
				var out, err bytes.Buffer
				if codigo := ejecutar(argumentos(t, idioma, d), &out, &err); codigo != 0 {
					t.Fatalf("code=%d error=%s", codigo, &err)
				}
				pdf, e := os.ReadFile(filepath.Join(d, "borrador.pdf"))
				if e != nil {
					t.Fatal(e)
				}
				b, e := os.ReadFile(filepath.Join(d, "borrador.json"))
				if e != nil {
					t.Fatal(e)
				}
				var borrador domain.Borrador
				if e := json.Unmarshal(b, &borrador); e != nil {
					t.Fatal(e)
				}
				if borrador.Estado != "borrador" || borrador.Modo != "ensayo_sintetico" || borrador.Idioma != idioma ||
					!bytes.HasPrefix(pdf, []byte("%PDF-")) || len(pdf) < 1000 || strings.Contains(borrador.Contenido.Titulo, "{{") {
					t.Fatal("incorrect artifact")
				}
				if anterior != nil && !bytes.Equal(anterior, pdf) {
					t.Fatal("PDF changed for same input")
				}
				anterior = pdf
				info, e := os.Stat(filepath.Join(d, "borrador.pdf"))
				if e != nil || info.Mode().Perm() != 0600 {
					t.Fatal("private file permissions missing")
				}
			}
		})
	}
}
func TestCLIExigeEnsayoYNoSobrescribe(t *testing.T) {
	d := filepath.Join(t.TempDir(), "new")
	args := argumentos(t, "es", d)
	var out, err bytes.Buffer
	if codigo := ejecutar(args[1:], &out, &err); codigo != 2 {
		t.Fatal("missing explicit guard")
	}
	if _, e := os.Stat(d); !os.IsNotExist(e) {
		t.Fatal("output created despite guard")
	}
	if ejecutar(args, &out, &err) != 0 {
		t.Fatal(err.String())
	}
	antes, e := os.ReadFile(filepath.Join(d, "borrador.pdf"))
	if e != nil {
		t.Fatal(e)
	}
	if ejecutar(args, &out, &err) != 1 {
		t.Fatal("existing folder accepted")
	}
	despues, e := os.ReadFile(filepath.Join(d, "borrador.pdf"))
	if e != nil || !bytes.Equal(antes, despues) {
		t.Fatal("existing artifact changed")
	}
}

func TestCLIIdiomaYRespaldoProcedenDelIndice(t *testing.T) {
	raiz := t.TempDir()
	indice := filepath.Join(raiz, "idiomas.json")
	catalogo := filepath.Join(raiz, "catalogo.json")
	var textos domain.Textos
	b, e := os.ReadFile("../../web/static/textos/en/certificados.json")
	if e != nil {
		t.Fatal(e)
	}
	if e = json.Unmarshal(b, &textos); e != nil {
		t.Fatal(e)
	}
	// An additional language requires only data. This fixture reuses the
	// existing messages deliberately; no translation is claimed.
	textos.Idioma = "pt"
	textos.FormatoFecha = "2006.01.02"
	b, e = json.Marshal(textos)
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(catalogo, b, 0600); e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(indice, []byte(`{"por_defecto":"pt","seguir_navegador":false,"idiomas":[{"codigo":"pt","nombre":"Português","localizacion":"pt-PT"}]}`), 0600); e != nil {
		t.Fatal(e)
	}
	args := []string{"-ensayo-sintetico", "-fuente", "../../internal/modules/certificados/adapters/fichero/testdata/servicios.ensayo.json", "-plantilla", "../../data/certificados/plantillas/servicios.v1.json", "-indice-idiomas", indice, "-textos", catalogo, "-salida", filepath.Join(raiz, "resultado")}
	var out, diagnostico bytes.Buffer
	if codigo := ejecutar(args, &out, &diagnostico); codigo != 0 {
		t.Fatalf("code=%d error=%s", codigo, &diagnostico)
	}
	b, e = os.ReadFile(filepath.Join(raiz, "resultado", "borrador.json"))
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.Contains(b, []byte(`"idioma": "pt"`)) || !bytes.Contains(b, []byte("2026.10.01")) {
		t.Fatal("language or format was compiled")
	}
}

func TestCLIAceptaLaMuestraConFormaDePersonalV1(t *testing.T) {
	for _, idioma := range []string{"es", "en"} {
		d := filepath.Join(t.TempDir(), "salida")
		args := argumentos(t, idioma, d)
		args[2] = "../../internal/modules/certificados/adapters/personalv1/testdata/servicios-personal-v1.ensayo.json"
		var out, err bytes.Buffer
		if codigo := ejecutar(args, &out, &err); codigo != 0 {
			t.Fatalf("%s: code=%d error=%s", idioma, codigo, &err)
		}
		b, e := os.ReadFile(filepath.Join(d, "borrador.json"))
		var borrador domain.Borrador
		if e != nil || json.Unmarshal(b, &borrador) != nil || borrador.Fuente.Cobertura != "parcial" || borrador.Estado != "borrador" {
			t.Fatalf("%s: %v", idioma, e)
		}
		texto := strings.Join(borrador.Contenido.Parrafos, "\n")
		if strings.Contains(texto, "{{") || !strings.Contains(texto, "ensayo:resolucion-2024-119") {
			t.Fatalf("%s: %s", idioma, texto)
		}
		pdf, e := os.ReadFile(filepath.Join(d, "borrador.pdf"))
		if e != nil || !bytes.HasPrefix(pdf, []byte("%PDF-")) {
			t.Fatalf("%s: sin PDF", idioma)
		}
	}
}
