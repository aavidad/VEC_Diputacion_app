package analisiscontenido

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// El corpus contiene solo documentos sintéticos. La acción JavaScript válida
// del PDF activo es la expresión inerte "0"; este test nunca abre el documento.
func TestInspeccionarPDFCorpus(t *testing.T) {
	bwrap, err := exec.LookPath("bwrap")
	if err != nil {
		t.Skip("bwrap no está instalado")
	}
	qpdf, err := exec.LookPath("qpdf")
	if err != nil {
		t.Skip("qpdf no está instalado")
	}
	prlimit, err := exec.LookPath("prlimit")
	if err != nil {
		t.Skip("prlimit no está instalado")
	}
	cfg := configuracionQPDF{Bwrap: bwrap, QPDF: qpdf, Prlimit: prlimit}
	for _, caso := range []struct {
		archivo string
		codigo  string
	}{
		{"pasivo.pdf", "PDF_PASIVO"},
		{"accion_openaction_js.pdf", "PDF_ACTIVO"},
		{"cifrado.pdf", "PDF_CIFRADO"},
		{"truncado.pdf", "PDF_DANADO"},
		{"byte_extra.pdf", "PDF_DANADO"},
	} {
		t.Run(caso.archivo, func(t *testing.T) {
			contenido, err := os.ReadFile(filepath.Join("testdata", caso.archivo))
			if err != nil {
				t.Fatal(err)
			}
			codigo, deteccion, err := inspeccionarPDF(context.Background(), contenido, cfg)
			if err != nil {
				t.Fatalf("inspeccionarPDF: %v", err)
			}
			if codigo != caso.codigo {
				t.Fatalf("código = %q, esperado %q", codigo, caso.codigo)
			}
			if caso.codigo == "PDF_PASIVO" && deteccion != nil {
				t.Fatalf("el PDF pasivo produjo detección: %+v", deteccion)
			}
			if caso.codigo != "PDF_PASIVO" && (deteccion == nil || deteccion.Codigo != caso.codigo) {
				t.Fatalf("detección = %+v, esperada %q", deteccion, caso.codigo)
			}
		})
	}
}
