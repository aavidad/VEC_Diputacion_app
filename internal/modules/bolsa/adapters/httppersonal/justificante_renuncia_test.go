package httppersonal

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	vec "vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/pruebas"

	docports "vec-diputacion-granada/internal/vec/documentos/ports"
)

func TestJustificanteMultipartConservaBytesYRechazaCamposAjenos(t *testing.T) {
	for _, campoExtra := range []string{"", "persona_ref", "bolsa"} {
		t.Run(campoExtra, func(t *testing.T) {
			var cuerpo bytes.Buffer
			m := multipart.NewWriter(&cuerpo)
			for k, v := range map[string]string{"bolsa": "bolsa:administrativo:2026", "clave_idempotencia": "portal-123:justificante"} {
				if err := m.WriteField(k, v); err != nil {
					t.Fatal(err)
				}
			}
			if campoExtra != "" {
				if err := m.WriteField(campoExtra, "ajeno"); err != nil {
					t.Fatal(err)
				}
			}
			f, err := m.CreateFormFile("documento", "justificante.pdf")
			if err != nil {
				t.Fatal(err)
			}
			original := []byte("%PDF-1.7\ncontenido\n%%EOF\n")
			if _, err := f.Write(original); err != nil {
				t.Fatal(err)
			}
			if err := m.Close(); err != nil {
				t.Fatal(err)
			}
			r := httptest.NewRequest(http.MethodPost, RutaJustificantesMiBolsa, &cuerpo)
			r.Header.Set("Content-Type", m.FormDataContentType())
			bolsa, clave, pdf, err := leerJustificanteRenuncia(httptest.NewRecorder(), r)
			if campoExtra != "" {
				if err == nil {
					t.Fatal("admitió un campo ajeno o repetido")
				}
				return
			}
			if err != nil || bolsa != "bolsa:administrativo:2026" || clave != "portal-123:justificante" || !bytes.Equal(pdf, original) {
				t.Fatalf("multipart no conserva el original: %v", err)
			}
		})
	}
}

type identidadJustificanteAusente struct{}

func (identidadJustificanteAusente) PrepararJustificanteRenuncia(*http.Request) (IdentidadJustificanteRenuncia, error) {
	return IdentidadJustificanteRenuncia{}, ErrAutenticacionAusente
}

type fabricaJustificanteInaccesible struct{ t *testing.T }

func (f fabricaJustificanteInaccesible) ServicioJustificanteRenuncia(context.Context, IdentidadJustificanteRenuncia, string) (docports.SubidorLocal, error) {
	f.t.Fatal("llegó al servicio sin identidad")
	return nil, errors.New("inaccesible")
}

type cuerpoJustificanteVigilado struct{ leido bool }

func (c *cuerpoJustificanteVigilado) Read([]byte) (int, error) { c.leido = true; return 0, io.EOF }

func TestJustificanteIdentificaAntesDeLeerPDF(t *testing.T) {
	h, err := NuevoHandlerJustificanteRenuncia(identidadJustificanteAusente{}, fabricaJustificanteInaccesible{t})
	if err != nil {
		t.Fatal(err)
	}
	cuerpo := &cuerpoJustificanteVigilado{}
	r := httptest.NewRequest(http.MethodPost, "https://localhost:18445"+RutaJustificantesMiBolsa, cuerpo)
	r.Header.Set("Origin", "https://localhost:18445")
	r.Header.Set("Sec-Fetch-Site", "same-origin")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if cuerpo.leido || w.Code < 400 {
		t.Fatalf("leyó el cuerpo o admitió sin identidad: %d", w.Code)
	}
}

func TestJustificanteRechazaEnvioCruzadoSinLeerPDF(t *testing.T) {
	h, err := NuevoHandlerJustificanteRenuncia(identidadJustificanteAusente{}, fabricaJustificanteInaccesible{t})
	if err != nil {
		t.Fatal(err)
	}
	cuerpo := &cuerpoJustificanteVigilado{}
	r := httptest.NewRequest(http.MethodPost, "https://localhost:18445"+RutaJustificantesMiBolsa, cuerpo)
	r.Header.Set("Origin", "https://origen-ajeno.invalid")
	r.Header.Set("Sec-Fetch-Site", "cross-site")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden || cuerpo.leido {
		t.Fatalf("POST cruzado: estado=%d leyó=%v", w.Code, cuerpo.leido)
	}
}

type identidadJustificanteFijadaPrueba struct{ identidad IdentidadJustificanteRenuncia }

func (p identidadJustificanteFijadaPrueba) PrepararJustificanteRenuncia(*http.Request) (IdentidadJustificanteRenuncia, error) {
	return p.identidad, nil
}

func TestJustificantePDFMayorDelLimiteDevuelve413(t *testing.T) {
	var cuerpo bytes.Buffer
	m := multipart.NewWriter(&cuerpo)
	if err := m.WriteField("bolsa", "bolsa:administrativo:2026"); err != nil {
		t.Fatal(err)
	}
	if err := m.WriteField("clave_idempotencia", "portal-justificante"); err != nil {
		t.Fatal(err)
	}
	f, err := m.CreateFormFile("documento", "justificante.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.Write(append([]byte("%PDF-1.4\n"), bytes.Repeat([]byte("x"), limiteJustificanteRenuncia)...)); err != nil {
		t.Fatal(err)
	}
	if err = m.Close(); err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodPost, RutaJustificantesMiBolsa, &cuerpo)
	r.Header.Set("Content-Type", m.FormDataContentType())
	actor, vinculo, err := pruebas.NuevoContextoRegistradoYVinculoV2(time.Date(2026, 10, 15, 8, 0, 0, 0, time.UTC),
		"per_0123456789abcdefghijkl", "prf_0123456789abcdefghijkl", vec.AuthMethodCertificate, vec.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	h, err := NuevoHandlerJustificanteRenuncia(identidadJustificanteFijadaPrueba{IdentidadJustificanteRenuncia{Actor: actor, Vinculo: vinculo}}, fabricaJustificanteInaccesible{t})
	if err != nil {
		t.Fatal(err)
	}
	r.URL.Scheme = "https"
	r.URL.Host = "example.com"
	r.Host = "example.com"
	r.Header.Set("Origin", "https://example.com")
	r.Header.Set("Sec-Fetch-Site", "same-origin")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusRequestEntityTooLarge || !strings.Contains(w.Body.String(), "peticion_demasiado_grande") {
		t.Fatalf("limite PDF: %d %s", w.Code, w.Body.String())
	}
}
