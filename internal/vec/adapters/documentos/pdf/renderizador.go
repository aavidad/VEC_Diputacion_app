// Package pdf genera la representacion PDF de trabajo mediante un adaptador
// reemplazable. Firma, sello de tiempo, CSV y registro son pasos posteriores.
package pdf

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"
	"unicode/utf8"

	"codeberg.org/go-pdf/fpdf"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"

	"vec-diputacion-granada/internal/vec/domain"
)

var ErrTextoInvalido = errors.New("pdf: texto invalido")

// Renderizador implementa el puerto PDF sin introducir la libreria en el
// dominio ni en los casos de uso.
type Renderizador struct{}

func (Renderizador) Formato() domain.FormatoDocumento {
	return domain.FormatoDocumentoPDF
}

func (Renderizador) Renderizar(ctx context.Context, contenido domain.ContenidoDocumento) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !textoValido(contenido.Titulo) {
		return nil, fmt.Errorf("%w: titulo", ErrTextoInvalido)
	}
	for indice, parrafo := range contenido.Parrafos {
		if !textoValido(parrafo) {
			return nil, fmt.Errorf("%w: parrafo %d", ErrTextoInvalido, indice+1)
		}
	}

	documento := fpdf.New("P", "mm", "A4", "")
	documento.SetMargins(20, 20, 20)
	documento.SetAutoPageBreak(true, 20)
	fechaDeterminista := time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)
	documento.SetCatalogSort(true)
	documento.SetCreationDate(fechaDeterminista)
	documento.SetModificationDate(fechaDeterminista)
	// El titulo visible puede contener datos fusionados. Los metadatos usan un
	// texto tecnico para no duplicar datos personales fuera del cuerpo.
	documento.SetTitle("Documento administrativo", true)
	documento.SetAuthor("Portal VEC Diputacion de Granada", true)
	documento.SetCreator("Portal VEC Diputacion de Granada", true)
	documento.SetProducer("Portal VEC Diputacion de Granada", true)
	documento.SetLang("es-ES")
	documento.AddUTF8FontFromBytes("vec", "", goregular.TTF)
	documento.AddUTF8FontFromBytes("vec", "B", gobold.TTF)
	documento.AddPage()
	documento.SetFont("vec", "B", 16)
	documento.MultiCell(0, 8, contenido.Titulo, "", "L", false)
	documento.Ln(4)
	documento.SetFont("vec", "", 11)
	for _, parrafo := range contenido.Parrafos {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		documento.MultiCell(0, 6, parrafo, "", "J", false)
		documento.Ln(3)
	}
	if documento.Err() {
		return nil, fmt.Errorf("pdf: componer: %w", documento.Error())
	}
	var salida bytes.Buffer
	if err := documento.Output(&salida); err != nil {
		return nil, fmt.Errorf("pdf: escribir: %w", err)
	}
	return salida.Bytes(), nil
}

func textoValido(texto string) bool {
	if !utf8.ValidString(texto) {
		return false
	}
	for _, caracter := range texto {
		if caracter == '\t' || caracter == '\n' || caracter == '\r' {
			continue
		}
		if caracter < 0x20 || caracter == 0x7f {
			return false
		}
	}
	return true
}
