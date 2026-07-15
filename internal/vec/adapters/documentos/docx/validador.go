package docx

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"io"
	"strings"
)

var ErrSalidaDOCXInvalida = errors.New("docx: salida generada invalida")

const (
	maximoPartesDOCX       = 16
	maximoBytesParteDOCX   = 20 * 1024 * 1024
	maximoBytesTotalesDOCX = 32 * 1024 * 1024
)

var partesDOCXPermitidas = map[string]bool{
	"[Content_Types].xml":          true,
	"_rels/.rels":                  true,
	"docProps/core.xml":            true,
	"docProps/app.xml":             true,
	"word/document.xml":            true,
	"word/_rels/document.xml.rels": true,
}

// ValidarSalida rechaza partes inesperadas, macros, relaciones externas,
// cifrado y expansiones ZIP desproporcionadas.
func (Renderizador) ValidarSalida(ctx context.Context, contenido []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	lector, err := zip.NewReader(bytes.NewReader(contenido), int64(len(contenido)))
	if err != nil || len(lector.File) == 0 || len(lector.File) > maximoPartesDOCX {
		return ErrSalidaDOCXInvalida
	}
	vistas := make(map[string]bool, len(lector.File))
	total := uint64(0)
	for _, archivo := range lector.File {
		if err := ctx.Err(); err != nil {
			return err
		}
		if archivo == nil || !partesDOCXPermitidas[archivo.Name] || vistas[archivo.Name] ||
			archivo.Flags&0x1 != 0 || archivo.UncompressedSize64 > maximoBytesParteDOCX ||
			total > maximoBytesTotalesDOCX-archivo.UncompressedSize64 {
			return ErrSalidaDOCXInvalida
		}
		vistas[archivo.Name] = true
		total += archivo.UncompressedSize64
		lectorParte, err := archivo.Open()
		if err != nil {
			return ErrSalidaDOCXInvalida
		}
		datos, err := io.ReadAll(io.LimitReader(lectorParte, maximoBytesParteDOCX+1))
		errorCierre := lectorParte.Close()
		if err != nil || errorCierre != nil || len(datos) > maximoBytesParteDOCX || !xmlValido(datos) {
			return ErrSalidaDOCXInvalida
		}
		if strings.HasSuffix(archivo.Name, ".rels") && relacionExterna(datos) {
			return ErrSalidaDOCXInvalida
		}
		if archivo.Name == "[Content_Types].xml" &&
			(bytes.Contains(bytes.ToLower(datos), []byte("macroenabled")) || bytes.Contains(bytes.ToLower(datos), []byte("vba"))) {
			return ErrSalidaDOCXInvalida
		}
	}
	for parte := range partesDOCXPermitidas {
		if !vistas[parte] {
			return ErrSalidaDOCXInvalida
		}
	}
	return nil
}

func xmlValido(datos []byte) bool {
	decodificador := xml.NewDecoder(bytes.NewReader(datos))
	for {
		_, err := decodificador.Token()
		switch {
		case err == nil:
			continue
		case errors.Is(err, io.EOF):
			return true
		default:
			return false
		}
	}
}

func relacionExterna(datos []byte) bool {
	decodificador := xml.NewDecoder(bytes.NewReader(datos))
	for {
		token, err := decodificador.Token()
		if errors.Is(err, io.EOF) {
			return false
		}
		if err != nil {
			return true
		}
		inicio, ok := token.(xml.StartElement)
		if !ok || inicio.Name.Local != "Relationship" {
			continue
		}
		for _, atributo := range inicio.Attr {
			if atributo.Name.Local == "TargetMode" && strings.EqualFold(strings.TrimSpace(atributo.Value), "External") {
				return true
			}
		}
	}
}
