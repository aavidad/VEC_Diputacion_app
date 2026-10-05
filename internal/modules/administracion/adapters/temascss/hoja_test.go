package temascss

import (
	"bytes"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/administracion/domain/temas"
)

func material(t *testing.T) (temas.Paquete, temas.Politica) {
	t.Helper()
	politica := temas.Politica{
		Esquema: 1, PoliticaRef: "politica-prueba-v1", SistemaDisenoRef: "sistema-prueba-v1",
		Tokens:            []string{"--portal-superficie", "--portal-tinta", "--portal-fondo-logo"},
		IdiomasRequeridos: []string{"es"}, ValoresFijos: map[string]string{"--portal-fondo-logo": "#ffffff"},
		Contrastes: []temas.ParContraste{
			{PrimerPlano: "--portal-tinta", Fondo: "--portal-superficie", Tipo: "texto", Minimo: 4.5},
			{PrimerPlano: "--portal-tinta", Fondo: "--portal-fondo-logo", Tipo: "componente", Minimo: 3},
		},
		Limites: temas.Limites{Bytes: 65536, Tokens: 3, Idiomas: 1, Nombre: 100},
	}
	paquete := temas.Paquete{
		Esquema: 1, TemaID: "tema-tercero", Version: 2,
		PoliticaRef: politica.PoliticaRef, SistemaDisenoRef: politica.SistemaDisenoRef,
		NombreKey: "ui.temas.tema-tercero.nombre",
		Textos:    map[string]map[string]string{"es": {"ui.temas.tema-tercero.nombre": "Tema de prueba"}},
		Variantes: temas.Variantes{
			Clara:  map[string]string{"--portal-tinta": "#000000", "--portal-superficie": "#FFFFFF", "--portal-fondo-logo": "#ffffff"},
			Oscura: map[string]string{"--portal-tinta": "#000000", "--portal-superficie": "#FFFFFF", "--portal-fondo-logo": "#ffffff"},
		},
	}
	return paquete, politica
}

func TestRecursoDeterministaVinculadoAlContenido(t *testing.T) {
	p, politica := material(t)
	hoja, err := Generar(p, politica)
	if err != nil {
		t.Fatal(err)
	}
	canon, err := p.Normalizar(politica)
	if err != nil {
		t.Fatal(err)
	}
	b, err := canon.Canonico()
	if err != nil {
		t.Fatal(err)
	}
	if hoja.PaqueteSHA256 != huella(b) || hoja.ContenidoSHA256 != huella(hoja.Contenido) {
		t.Fatal("el recurso perdió el vínculo con sus bytes")
	}
	politica.Tokens = []string{"--portal-tinta", "--portal-fondo-logo", "--portal-superficie"}
	p.Variantes.Clara["--portal-superficie"] = "#ffffff"
	reordenada, err := Generar(p, politica)
	if err != nil || !bytes.Equal(hoja.Contenido, reordenada.Contenido) || hoja.PaqueteSHA256 != reordenada.PaqueteSHA256 {
		t.Fatal("la representación equivalente cambió la hoja")
	}
	p.TemaID = "otro-tema"
	p.NombreKey = "ui.temas.otro-tema.nombre" // gitleaks:allow -- clave i18n pública de ejemplo, no secreto
	p.Textos = map[string]map[string]string{"es": {p.NombreKey: "Otro tema de prueba"}}
	otra, err := Generar(p, politica)
	if err != nil || otra.PaqueteSHA256 == hoja.PaqueteSHA256 || otra.ContenidoSHA256 == hoja.ContenidoSHA256 {
		t.Fatal("dos identidades quedaron ligadas al mismo selector")
	}
}

func TestRevalidarImpideInyeccionYContrasteInsuficiente(t *testing.T) {
	for _, valor := range []string{"#fff; color: red", "url(https://example.invalid)", "var(--portal-radio)", "#ffffff"} {
		p, politica := material(t)
		p.Variantes.Clara["--portal-tinta"] = valor
		if hoja, err := Generar(p, politica); err == nil || len(hoja.Contenido) != 0 {
			t.Fatalf("se emitió una hoja para %q", valor)
		}
	}
	p, politica := material(t)
	politica.Tokens[0] = "--portal-superficie;}body{color"
	if hoja, err := Generar(p, politica); err == nil || len(hoja.Contenido) != 0 {
		t.Fatal("se interpretó un selector como token")
	}
}

func TestCapaDeAccesibilidadYMetadatos(t *testing.T) {
	p, politica := material(t)
	antes, err := p.Canonico()
	if err != nil {
		t.Fatal(err)
	}
	hoja, err := Generar(p, politica)
	if err != nil {
		t.Fatal(err)
	}
	css := string(hoja.Contenido)
	if !strings.HasPrefix(css, "@media (forced-colors: none) {\n") || strings.Count(css, ":not([data-contraste=\"true\"]):not(.alto-contraste)") != 2 {
		t.Fatal("la paleta puede prevalecer sobre accesibilidad")
	}
	for _, prohibido := range []string{"!important", "@import", "Tema de prueba", "ui.temas", "url(", "--portal-radio", "#FFFFFF"} {
		if strings.Contains(css, prohibido) {
			t.Fatalf("salida ajena a los colores normalizados: %s", prohibido)
		}
	}
	despues, err := p.Canonico()
	if err != nil || !bytes.Equal(antes, despues) || politica.Tokens[0] != "--portal-superficie" {
		t.Fatal("generar alteró el material de entrada")
	}
}
