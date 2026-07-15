// Package docx genera documentos Word Open XML sin macros ni recursos externos.
package docx

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"time"
	"unicode/utf8"
)

// ErrTextoInvalido indica que el título o un párrafo contiene caracteres que
// no se pueden representar en un documento XML 1.0 válido.
var ErrTextoInvalido = errors.New("docx: texto no válido para XML 1.0")

const (
	tiposContenido = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
  <Override PartName="/docProps/core.xml" ContentType="application/vnd.openxmlformats-package.core-properties+xml"/>
  <Override PartName="/docProps/app.xml" ContentType="application/vnd.openxmlformats-officedocument.extended-properties+xml"/>
</Types>`

	relacionesPaquete = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
  <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties" Target="docProps/core.xml"/>
  <Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/extended-properties" Target="docProps/app.xml"/>
</Relationships>`

	relacionesDocumento = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"/>`

	propiedadesAplicacion = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Properties xmlns="http://schemas.openxmlformats.org/officeDocument/2006/extended-properties" xmlns:vt="http://schemas.openxmlformats.org/officeDocument/2006/docPropsVTypes">
  <Application>Portal VEC Diputación de Granada</Application>
  <AppVersion>1.0</AppVersion>
</Properties>`
)

var fechaZIPDeterminista = time.Date(1980, time.January, 1, 0, 0, 0, 0, time.UTC)

// Renderizar genera un DOCX editable con un título y una secuencia de
// párrafos. El resultado contiene únicamente partes internas Open XML.
func Renderizar(titulo string, parrafos []string) ([]byte, error) {
	if !esTextoXMLValido(titulo) {
		return nil, fmt.Errorf("%w: título", ErrTextoInvalido)
	}
	for indice, parrafo := range parrafos {
		if !esTextoXMLValido(parrafo) {
			return nil, fmt.Errorf("%w: párrafo %d", ErrTextoInvalido, indice+1)
		}
	}

	partes := []parte{
		{nombre: "[Content_Types].xml", contenido: []byte(tiposContenido)},
		{nombre: "_rels/.rels", contenido: []byte(relacionesPaquete)},
		{nombre: "docProps/core.xml", contenido: propiedadesNucleo()},
		{nombre: "docProps/app.xml", contenido: []byte(propiedadesAplicacion)},
		{nombre: "word/document.xml", contenido: documento(titulo, parrafos)},
		{nombre: "word/_rels/document.xml.rels", contenido: []byte(relacionesDocumento)},
	}

	var destino bytes.Buffer
	escritor := zip.NewWriter(&destino)
	for _, actual := range partes {
		cabecera := &zip.FileHeader{
			Name:     actual.nombre,
			Method:   zip.Deflate,
			Modified: fechaZIPDeterminista,
		}
		archivo, err := escritor.CreateHeader(cabecera)
		if err != nil {
			return nil, fmt.Errorf("docx: crear parte %q: %w", actual.nombre, err)
		}
		if _, err := archivo.Write(actual.contenido); err != nil {
			return nil, fmt.Errorf("docx: escribir parte %q: %w", actual.nombre, err)
		}
	}
	if err := escritor.Close(); err != nil {
		return nil, fmt.Errorf("docx: cerrar contenedor: %w", err)
	}

	return destino.Bytes(), nil
}

type parte struct {
	nombre    string
	contenido []byte
}

func documento(titulo string, parrafos []string) []byte {
	var salida bytes.Buffer
	salida.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	salida.WriteString(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>`)
	salida.WriteString(`<w:p><w:pPr><w:keepNext/></w:pPr><w:r><w:rPr><w:b/><w:sz w:val="32"/></w:rPr><w:t xml:space="preserve">`)
	escaparXML(&salida, titulo)
	salida.WriteString(`</w:t></w:r></w:p>`)
	for _, parrafo := range parrafos {
		salida.WriteString(`<w:p><w:r><w:t xml:space="preserve">`)
		escaparXML(&salida, parrafo)
		salida.WriteString(`</w:t></w:r></w:p>`)
	}
	salida.WriteString(`<w:sectPr><w:pgSz w:w="11906" w:h="16838"/><w:pgMar w:top="1440" w:right="1440" w:bottom="1440" w:left="1440" w:header="708" w:footer="708" w:gutter="0"/></w:sectPr>`)
	salida.WriteString(`</w:body></w:document>`)
	return salida.Bytes()
}

func propiedadesNucleo() []byte {
	var salida bytes.Buffer
	salida.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	salida.WriteString(`<cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties" xmlns:dc="http://purl.org/dc/elements/1.1/">`)
	salida.WriteString(`<dc:title>`)
	escaparXML(&salida, "Documento administrativo")
	salida.WriteString(`</dc:title><dc:creator>Portal VEC Diputación de Granada</dc:creator><cp:lastModifiedBy>Portal VEC Diputación de Granada</cp:lastModifiedBy><cp:revision>1</cp:revision></cp:coreProperties>`)
	return salida.Bytes()
}

func escaparXML(destino *bytes.Buffer, texto string) {
	for _, caracter := range texto {
		switch caracter {
		case '&':
			destino.WriteString("&amp;")
		case '<':
			destino.WriteString("&lt;")
		case '>':
			destino.WriteString("&gt;")
		default:
			destino.WriteRune(caracter)
		}
	}
}

func esTextoXMLValido(texto string) bool {
	if !utf8.ValidString(texto) {
		return false
	}
	for _, caracter := range texto {
		if caracter == '\t' || caracter == '\n' || caracter == '\r' {
			continue
		}
		if caracter < 0x20 || caracter > 0x10FFFF ||
			(caracter >= 0xD800 && caracter <= 0xDFFF) ||
			caracter == 0xFFFE || caracter == 0xFFFF {
			return false
		}
	}
	return true
}
