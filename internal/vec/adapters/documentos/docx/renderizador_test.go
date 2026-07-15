package docx

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"io"
	"net/url"
	"sort"
	"strings"
	"testing"
)

func TestRenderizarCreaDOCXValidoYDeterminista(t *testing.T) {
	titulo := `Resolución & propuesta <externa> — selección pública`
	parrafos := []string{
		`Candidata: María Núñez & Hijas`,
		`Texto con <etiqueta> que debe tratarse como contenido.`,
		"Línea uno\nLínea dos",
		"",
	}

	primero, err := Renderizar(titulo, parrafos)
	if err != nil {
		t.Fatalf("Renderizar() error = %v", err)
	}
	segundo, err := Renderizar(titulo, parrafos)
	if err != nil {
		t.Fatalf("segunda Renderizar() error = %v", err)
	}
	if !bytes.Equal(primero, segundo) {
		t.Fatal("Renderizar() no produjo un DOCX determinista")
	}

	partes := abrirPartes(t, primero)
	esperadas := []string{
		"[Content_Types].xml",
		"_rels/.rels",
		"docProps/app.xml",
		"docProps/core.xml",
		"word/_rels/document.xml.rels",
		"word/document.xml",
	}
	nombres := make([]string, 0, len(partes))
	for nombre := range partes {
		nombres = append(nombres, nombre)
	}
	sort.Strings(nombres)
	if strings.Join(nombres, "\n") != strings.Join(esperadas, "\n") {
		t.Fatalf("partes DOCX = %v; esperadas = %v", nombres, esperadas)
	}

	for nombre, contenido := range partes {
		if !strings.HasSuffix(nombre, ".xml") && !strings.HasSuffix(nombre, ".rels") {
			continue
		}
		decodificador := xml.NewDecoder(bytes.NewReader(contenido))
		for {
			_, err := decodificador.Token()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				t.Fatalf("parte %q contiene XML no válido: %v", nombre, err)
			}
		}
	}

	documento := string(partes["word/document.xml"])
	if strings.Contains(documento, "<etiqueta>") || strings.Contains(documento, "<externa>") {
		t.Fatalf("document.xml contiene texto sin escapar: %s", documento)
	}
	for _, fragmento := range []string{"&amp;", "&lt;etiqueta&gt;", "María Núñez", "selección pública"} {
		if !strings.Contains(documento, fragmento) {
			t.Errorf("document.xml no contiene %q", fragmento)
		}
	}

	if tituloObtenido := leerPrimerElemento(t, partes["docProps/core.xml"], "title"); tituloObtenido != "Documento administrativo" {
		t.Errorf("título de metadatos = %q", tituloObtenido)
	}
}

func TestRenderizarNoIncluyeMacrosNiRelacionesExternas(t *testing.T) {
	datos, err := Renderizar("Documento", []string{"Contenido"})
	if err != nil {
		t.Fatalf("Renderizar() error = %v", err)
	}
	partes := abrirPartes(t, datos)

	for nombre := range partes {
		nombreMinusculas := strings.ToLower(nombre)
		if strings.Contains(nombreMinusculas, "vbaproject") || strings.HasPrefix(nombreMinusculas, "word/media/") {
			t.Fatalf("recurso ejecutable o externo inesperado: %q", nombre)
		}
	}
	for nombre, contenido := range partes {
		if !strings.HasSuffix(nombre, ".rels") {
			continue
		}
		comprobarRelacionesInternas(t, nombre, contenido)
	}
	if bytes.Contains(bytes.ToLower(partes["[Content_Types].xml"]), []byte("macroenabled")) {
		t.Fatal("[Content_Types].xml declara contenido con macros")
	}
}

func TestRenderizarRechazaTextoNoValidoParaXML(t *testing.T) {
	pruebas := []struct {
		nombre   string
		titulo   string
		parrafos []string
	}{
		{nombre: "utf8 incorrecto", titulo: string([]byte{0xff}), parrafos: nil},
		{nombre: "control en titulo", titulo: "título\x00", parrafos: nil},
		{nombre: "control en parrafo", titulo: "título", parrafos: []string{"dato\x01"}},
	}

	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			_, err := Renderizar(prueba.titulo, prueba.parrafos)
			if !errors.Is(err, ErrTextoInvalido) {
				t.Fatalf("Renderizar() error = %v; esperado ErrTextoInvalido", err)
			}
		})
	}
}

func TestValidadorDOCXRechazaRelacionExterna(t *testing.T) {
	datos, err := Renderizar("Documento", []string{"Contenido"})
	if err != nil {
		t.Fatalf("Renderizar() error = %v", err)
	}
	partes := abrirPartes(t, datos)
	partes["word/_rels/document.xml.rels"] = []byte(`<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="imagen" Target="https://externo.invalid/imagen.png" TargetMode="External"/>
</Relationships>`)
	alterado := crearDOCXDesdePartes(t, partes)
	if err := (Renderizador{}).ValidarSalida(context.Background(), alterado); !errors.Is(err, ErrSalidaDOCXInvalida) {
		t.Fatalf("ValidarSalida() error = %v", err)
	}
}

func abrirPartes(t *testing.T, datos []byte) map[string][]byte {
	t.Helper()
	lector, err := zip.NewReader(bytes.NewReader(datos), int64(len(datos)))
	if err != nil {
		t.Fatalf("abrir DOCX como ZIP: %v", err)
	}
	partes := make(map[string][]byte, len(lector.File))
	for _, archivo := range lector.File {
		lectorParte, err := archivo.Open()
		if err != nil {
			t.Fatalf("abrir parte %q: %v", archivo.Name, err)
		}
		contenido, err := io.ReadAll(lectorParte)
		errCierre := lectorParte.Close()
		if err != nil {
			t.Fatalf("leer parte %q: %v", archivo.Name, err)
		}
		if errCierre != nil {
			t.Fatalf("cerrar parte %q: %v", archivo.Name, errCierre)
		}
		partes[archivo.Name] = contenido
	}
	return partes
}

func crearDOCXDesdePartes(t *testing.T, partes map[string][]byte) []byte {
	t.Helper()
	nombres := make([]string, 0, len(partes))
	for nombre := range partes {
		nombres = append(nombres, nombre)
	}
	sort.Strings(nombres)
	var destino bytes.Buffer
	escritor := zip.NewWriter(&destino)
	for _, nombre := range nombres {
		parte, err := escritor.Create(nombre)
		if err != nil {
			t.Fatalf("crear parte %q: %v", nombre, err)
		}
		if _, err := parte.Write(partes[nombre]); err != nil {
			t.Fatalf("escribir parte %q: %v", nombre, err)
		}
	}
	if err := escritor.Close(); err != nil {
		t.Fatalf("cerrar DOCX: %v", err)
	}
	return destino.Bytes()
}

func leerPrimerElemento(t *testing.T, contenido []byte, nombreLocal string) string {
	t.Helper()
	decodificador := xml.NewDecoder(bytes.NewReader(contenido))
	for {
		token, err := decodificador.Token()
		if errors.Is(err, io.EOF) {
			t.Fatalf("no se encontró el elemento %q", nombreLocal)
		}
		if err != nil {
			t.Fatalf("leer XML: %v", err)
		}
		inicio, correcto := token.(xml.StartElement)
		if !correcto || inicio.Name.Local != nombreLocal {
			continue
		}
		var valor string
		if err := decodificador.DecodeElement(&valor, &inicio); err != nil {
			t.Fatalf("leer elemento %q: %v", nombreLocal, err)
		}
		return valor
	}
}

func comprobarRelacionesInternas(t *testing.T, nombre string, contenido []byte) {
	t.Helper()
	decodificador := xml.NewDecoder(bytes.NewReader(contenido))
	for {
		token, err := decodificador.Token()
		if errors.Is(err, io.EOF) {
			return
		}
		if err != nil {
			t.Fatalf("leer relaciones de %q: %v", nombre, err)
		}
		inicio, correcto := token.(xml.StartElement)
		if !correcto || inicio.Name.Local != "Relationship" {
			continue
		}
		var destino, modo string
		for _, atributo := range inicio.Attr {
			switch atributo.Name.Local {
			case "Target":
				destino = atributo.Value
			case "TargetMode":
				modo = atributo.Value
			}
		}
		if strings.EqualFold(modo, "External") {
			t.Fatalf("relación externa en %q hacia %q", nombre, destino)
		}
		direccion, err := url.Parse(destino)
		if err != nil || direccion.IsAbs() || strings.HasPrefix(destino, "//") {
			t.Fatalf("destino de relación no interno en %q: %q", nombre, destino)
		}
	}
}
