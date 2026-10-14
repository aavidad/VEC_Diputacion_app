package httpinterno

import (
	"archive/zip"
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/dietas/adapters/documentos"
	"vec-diputacion-granada/internal/modules/dietas/ports"
)

func manejadorDocumentoLiquidacionPrueba(t *testing.T) (*ManejadorDocumentoLiquidacion, *capturaLiquidacionHTTPPrueba, *lectorLiquidacionHTTPPrueba) {
	t.Helper()
	_, _, _, registro, captura, lector := manejadorLiquidacionHTTPPrueba(t)
	lector.resultado.Liquidacion = registro.resultado.Instantanea()
	lector.resultado.Auditoria.Referencia = "auditoria:lectura:uno"
	textos := map[string]documentos.TextosLiquidacionDocumento{}
	for _, idioma := range []string{"es", "en"} {
		f, err := os.Open("../../../../../web/static/textos/" + idioma + "/dietas-liquidacion-documento.json")
		if err != nil {
			t.Fatal(err)
		}
		c, err := documentos.CargarTextosLiquidacionDocumento(f)
		_ = f.Close()
		if err != nil {
			t.Fatal(err)
		}
		textos[idioma] = c
	}
	h, err := NuevoManejadorDocumentoLiquidacion(captura, lector, textos)
	if err != nil {
		t.Fatal(err)
	}
	return h, captura, lector
}

func pedirDocumentoLiquidacion(t *testing.T, h http.Handler, formato, idioma string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(http.MethodGet, RutaLiquidaciones+referenciaLiquidacionPrueba+"/liquidaciones/3/documento?formato="+formato, nil)
	r.Header.Set("Accept-Language", idioma)
	if formato == "pdf" {
		r.Header.Set("Accept", "application/pdf")
	} else {
		r.Header.Set("Accept", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestDocumentoLiquidacionPDFYWordDesdeInstantaneaAuditada(t *testing.T) {
	h, c, l := manejadorDocumentoLiquidacionPrueba(t)
	pdf := pedirDocumentoLiquidacion(t, h, "pdf", "es-ES")
	if pdf.Code != 200 || !bytes.HasPrefix(pdf.Body.Bytes(), []byte("%PDF-")) || pdf.Header().Get("Content-Type") != "application/pdf" || pdf.Header().Get("Cache-Control") != "no-store" || pdf.Header().Get("X-Content-Type-Options") != "nosniff" || pdf.Header().Get("X-Content-SHA256") == "" || c.llamadas != 1 || l.llamadas != 1 || l.consulta.Version != 3 {
		t.Fatalf("pdf: estado=%d bytes=%d captura=%d lectura=%d", pdf.Code, pdf.Body.Len(), c.llamadas, l.llamadas)
	}
	if pdf.Header().Get("Content-Disposition") != `attachment; filename="VEC-D-2026-000001.pdf"` {
		t.Fatal("nombre PDF incompatible con el cliente")
	}
	word := pedirDocumentoLiquidacion(t, h, "docx", "en-GB")
	if word.Code != 200 || !bytes.HasPrefix(word.Body.Bytes(), []byte("PK\x03\x04")) || !strings.Contains(word.Header().Get("Content-Type"), "wordprocessingml") || c.llamadas != 2 || l.llamadas != 2 {
		t.Fatalf("word: estado=%d bytes=%d", word.Code, word.Body.Len())
	}
	if word.Header().Get("Content-Disposition") != `attachment; filename="VEC-D-2026-000001.docx"` {
		t.Fatal("nombre Word incompatible con el cliente")
	}
	z, err := zip.NewReader(bytes.NewReader(word.Body.Bytes()), int64(word.Body.Len()))
	if err != nil {
		t.Fatal(err)
	}
	var xml []byte
	for _, f := range z.File {
		if f.Name == "word/document.xml" {
			r, e := f.Open()
			if e != nil {
				t.Fatal(e)
			}
			xml, e = io.ReadAll(r)
			_ = r.Close()
			if e != nil {
				t.Fatal(e)
			}
		}
	}
	for _, esperado := range []string{"Business trip expense settlement", "VEC-D-2026-000001", "4.00 €", "5.00 €", "1.00 €", "recibo:uno", "revision_importe", "Amount under published rates: 5.00 €", "Reduction from the published calculation: 1.00 €", l.resultado.Liquidacion.SnapshotSHA256, "https://www.dipgra.es/bases"} {
		if !bytes.Contains(xml, []byte(esperado)) {
			t.Fatalf("falta %q en Word", esperado)
		}
	}
	if bytes.Contains(xml, []byte(l.resultado.Liquidacion.Constancia.ClaveIdempotencia)) {
		t.Fatal("clave técnica visible en Word")
	}
}

func TestDocumentoLiquidacionDeniegaSinBytesNiCabeceraDeArchivo(t *testing.T) {
	h, c, l := manejadorDocumentoLiquidacionPrueba(t)
	c.err = ports.ErrAccesoCircuitoDenegado
	w := pedirDocumentoLiquidacion(t, h, "pdf", "es")
	if w.Code != http.StatusForbidden || l.llamadas != 0 || bytes.HasPrefix(w.Body.Bytes(), []byte("%PDF-")) || w.Header().Get("Content-Disposition") != "" {
		t.Fatalf("denegacion: %d %q", w.Code, w.Header().Get("Content-Disposition"))
	}
	c.err = nil
	s := l.resultado.Liquidacion
	s.Economia.Lineas[0].ReconocidoPropuestoCentimos++
	l.resultado.Liquidacion = s
	w = pedirDocumentoLiquidacion(t, h, "pdf", "es")
	if w.Code != http.StatusServiceUnavailable || bytes.HasPrefix(w.Body.Bytes(), []byte("%PDF-")) || w.Header().Get("Content-Disposition") != "" {
		t.Fatalf("instantanea alterada: %d", w.Code)
	}
}
