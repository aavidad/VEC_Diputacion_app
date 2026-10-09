package docx

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/vec/adapters/documentos/membrete"
	"vec-diputacion-granada/internal/vec/domain"
)

var contenidoMembrete = domain.ContenidoDocumento{
	Titulo:   "Resolución de prueba",
	Parrafos: []string{"Primer párrafo inventado.", "Segundo párrafo con eñe y €."},
}

// huellaSinMembrete es la salida anterior a la existencia del membrete.
const huellaSinMembrete = "412351659da5a5a8335a8cbda0c0af22231e5d1a5c85c1d88d87169dd806787a"

func TestDOCXSinMembreteConservaLosBytesAnteriores(t *testing.T) {
	datos, err := Renderizador{}.Renderizar(context.Background(), contenidoMembrete)
	if err != nil {
		t.Fatalf("Renderizar() error = %v", err)
	}
	suma := sha256.Sum256(datos)
	if got := hex.EncodeToString(suma[:]); got != huellaSinMembrete {
		t.Fatalf("la salida sin membrete cambió: %s", got)
	}
}

func TestDOCXConMembreteLlevaLogoInternoYPasaValidador(t *testing.T) {
	r := Renderizador{Membrete: true}
	datos, err := r.Renderizar(context.Background(), contenidoMembrete)
	if err != nil {
		t.Fatalf("Renderizar() error = %v", err)
	}
	if err := r.ValidarSalida(context.Background(), datos); err != nil {
		t.Fatalf("ValidarSalida() error = %v", err)
	}
	partes := abrirPartes(t, datos)
	if !membrete.EsLogo(partes["word/media/logo.png"]) {
		t.Fatal("falta word/media/logo.png o no es el logotipo embebido")
	}
	for _, nombre := range []string{"word/_rels/document.xml.rels", "word/_rels/header1.xml.rels"} {
		comprobarRelacionesInternas(t, nombre, partes[nombre])
	}
	if !bytes.Contains(partes["[Content_Types].xml"], []byte(`Extension="png" ContentType="image/png"`)) ||
		!bytes.Contains(partes["word/document.xml"], []byte(`<w:headerReference w:type="default" r:id="rId1"/>`)) {
		t.Fatal("faltan el tipo png o la referencia a la cabecera")
	}
	repetido, err := r.Renderizar(context.Background(), contenidoMembrete)
	if err != nil || !bytes.Equal(datos, repetido) {
		t.Fatalf("el DOCX con membrete no es determinista: %v", err)
	}
}

func TestValidadorDOCXMembreteSoloAdmiteElLogoYCompleto(t *testing.T) {
	datos, err := RenderizarConMembrete("Título", []string{"Texto"})
	if err != nil {
		t.Fatalf("RenderizarConMembrete() error = %v", err)
	}
	casos := map[string]func(map[string][]byte){
		"otra imagen": func(p map[string][]byte) { p["word/media/logo.png"] = append(membrete.LogoPNG(), 0) },
		"sin logo":    func(p map[string][]byte) { delete(p, "word/media/logo.png") },
		"sin cabecera": func(p map[string][]byte) {
			delete(p, "word/header1.xml")
		},
		"cabecera con relación externa": func(p map[string][]byte) {
			p["word/_rels/header1.xml.rels"] = []byte(`<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="https://ejemplo.invalid/logo.png" TargetMode="External"/></Relationships>`)
		},
		"otra imagen con otro nombre": func(p map[string][]byte) { p["word/media/image2.png"] = membrete.LogoPNG() },
	}
	for nombre, alterar := range casos {
		t.Run(nombre, func(t *testing.T) {
			partes := abrirPartes(t, datos)
			alterar(partes)
			err := Renderizador{}.ValidarSalida(context.Background(), crearDOCXDesdePartes(t, partes))
			if !errors.Is(err, ErrSalidaDOCXInvalida) {
				t.Fatalf("ValidarSalida() error = %v; se esperaba rechazo", err)
			}
		})
	}
}
