package pdf

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/vec/domain"
)

func TestRenderizadorGeneraPDFConUnicodeSinAccionesActivas(t *testing.T) {
	renderizador := Renderizador{}
	contenido := domain.ContenidoDocumento{
		Titulo: "Resolucion de seleccion publica",
		Parrafos: []string{
			"Maria Nunez acredita titulacion y experiencia.",
			"Texto con simbolos: euro €, eñe ñ y comillas «administrativas».",
		},
	}
	datos, err := renderizador.Renderizar(context.Background(), contenido)
	if err != nil {
		t.Fatalf("Renderizar() error = %v", err)
	}
	if !bytes.HasPrefix(datos, []byte("%PDF-")) || !bytes.HasSuffix(bytes.TrimSpace(datos), []byte("%%EOF")) {
		t.Fatalf("salida no parece un PDF completo: inicio=%q tamano=%d", datos[:min(8, len(datos))], len(datos))
	}
	for _, prohibido := range [][]byte{[]byte("/JavaScript"), []byte("/Launch"), []byte("/Type /EmbeddedFile"), []byte("/URI")} {
		if bytes.Contains(datos, prohibido) {
			t.Fatalf("el PDF contiene una accion o adjunto no permitido: %s", prohibido)
		}
	}
	repetido, err := renderizador.Renderizar(context.Background(), contenido)
	if err != nil {
		t.Fatalf("segunda Renderizar() error = %v", err)
	}
	if !bytes.Equal(datos, repetido) {
		t.Fatal("el mismo contenido no produjo exactamente el mismo PDF")
	}
}

func TestRenderizadorRechazaControles(t *testing.T) {
	_, err := (Renderizador{}).Renderizar(context.Background(), domain.ContenidoDocumento{
		Titulo:   "Documento",
		Parrafos: []string{"dato\x00"},
	})
	if !errors.Is(err, ErrTextoInvalido) {
		t.Fatalf("Renderizar() error = %v; esperado ErrTextoInvalido", err)
	}
}

func TestValidadorPDFRechazaContenidoActivo(t *testing.T) {
	contenido := []byte("%PDF-1.7\n1 0 obj << /S /JavaScript /JS (alert) >> endobj\n%%EOF")
	if err := (Renderizador{}).ValidarSalida(context.Background(), contenido); !errors.Is(err, ErrSalidaPDFInvalida) {
		t.Fatalf("ValidarSalida() error = %v", err)
	}
}
