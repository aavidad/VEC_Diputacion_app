package informejuridico

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	docxvec "vec-diputacion-granada/internal/vec/adapters/documentos/docx"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

type renderizadorDocumentoDOCXPrueba struct{ llamadas int }

func (r *renderizadorDocumentoDOCXPrueba) Formato() vecdomain.FormatoDocumento {
	return vecdomain.FormatoDocumentoDOCX
}
func (r *renderizadorDocumentoDOCXPrueba) Renderizar(context.Context, vecdomain.ContenidoDocumento) ([]byte, error) {
	r.llamadas++
	return nil, nil
}
func (*renderizadorDocumentoDOCXPrueba) ValidarSalida(context.Context, []byte) error { return nil }

func TestBorradoresRRHHDOCXSeisRepresentacionesValidasYMarcadas(t *testing.T) {
	renderizador := RenderizadorBorradorDOCXDesarrollo{DOCX: docxvec.Renderizador{}}
	for _, caso := range []struct {
		tipo     ports.TipoBorradorRRHH
		esperado string
	}{
		{ports.BorradorInformeDefinitivo, "Informe definitivo — borrador de desarrollo"},
		{ports.BorradorResolucion, "Resolución — borrador de desarrollo"},
		{ports.BorradorDiligencia, "Diligencia — borrador de desarrollo"},
		{ports.BorradorTomaPosesion, "Toma de posesión — borrador de desarrollo"},
		{ports.BorradorNotificacion, "Notificación — borrador de desarrollo"},
		{ports.BorradorComunicacionCentro, "Comunicación al centro — borrador de desarrollo"},
	} {
		t.Run(string(caso.tipo), func(t *testing.T) {
			contenido, err := renderizador.RenderizarBorradorDOCX(
				context.Background(), caso.tipo, detalleInformeDefinitivoPrueba(),
			)
			if err != nil {
				t.Fatal(err)
			}
			if err := (docxvec.Renderizador{}).ValidarSalida(context.Background(), contenido); err != nil {
				t.Fatalf("DOCX no válido: %v", err)
			}
			lector, err := zip.NewReader(bytes.NewReader(contenido), int64(len(contenido)))
			if err != nil {
				t.Fatal(err)
			}
			var documento []byte
			for _, archivo := range lector.File {
				if archivo.Name != "word/document.xml" {
					continue
				}
				entrada, err := archivo.Open()
				if err != nil {
					t.Fatal(err)
				}
				documento, err = io.ReadAll(entrada)
				_ = entrada.Close()
				if err != nil {
					t.Fatal(err)
				}
			}
			texto := string(documento)
			for _, esperado := range []string{caso.esperado, "NO FIRMADO NI VALIDADO", "2026/CT-0001", "SIN EFECTOS ADMINISTRATIVOS"} {
				if !strings.Contains(texto, esperado) {
					t.Fatalf("falta %q", esperado)
				}
			}
		})
	}
}

func TestBorradorRRHHDOCXDeniegaAntesDeRenderizar(t *testing.T) {
	doble := &renderizadorDocumentoDOCXPrueba{}
	detalle := detalleInformeDefinitivoPrueba()
	detalle.Hitos[6].AccionClave = "ajeno"
	_, err := (RenderizadorBorradorDOCXDesarrollo{DOCX: doble}).RenderizarBorradorDOCX(
		context.Background(), ports.BorradorResolucion, detalle,
	)
	if !errors.Is(err, ports.ErrBorradorRRHHNoDisponible) || doble.llamadas != 0 {
		t.Fatalf("documento denegado se renderizó: llamadas=%d err=%v", doble.llamadas, err)
	}
}
