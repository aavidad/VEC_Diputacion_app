package httpinterno

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	almacencanonico "vec-diputacion-granada/internal/vec/canonico/almacen"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type servicioOriginalHTTPPrueba struct {
	llamadas  int
	solicitud vecports.SolicitudOriginalFirmableCT
	contenido []byte
	err       error
	alterar   func(*vecports.OriginalFirmableCT)
}

func (s *servicioOriginalHTTPPrueba) Preparar(_ context.Context, q vecports.SolicitudOriginalFirmableCT) (vecports.OriginalFirmableCT, error) {
	s.llamadas++
	s.solicitud = q
	contenido := s.contenido
	if contenido == nil {
		contenido = []byte("%PDF-1.7\n%%EOF")
	}
	h := sha256.Sum256(contenido)
	i := almacencanonico.IdentidadOriginalCT{OrganizacionRef: q.OrganizacionRef, ExpedienteRef: q.ExpedienteRef, Documento: q.Documento, Version: q.OriginalVersion}
	r := vecports.OriginalFirmableCT{Referencia: i.Referencia(), Version: q.OriginalVersion, TipoRef: "ref:" + strings.Repeat("a", 64), HuellaSHA256: hex.EncodeToString(h[:]), Contenido: bytes.Clone(contenido)}
	if s.alterar != nil {
		s.alterar(&r)
	}
	return r, s.err
}

const cuerpoOriginalHTTPPrueba = `{"expediente_ref":"expediente:ct:001","documento":"resolucion","version_observada":7}`

func peticionOriginalHTTPPrueba(cuerpo string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, RutaOriginalFirmableCT, strings.NewReader(cuerpo))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json")
	return r
}

func TestOriginalFirmableHTTPVersionFuenteBytesYReplay(t *testing.T) {
	s := &servicioOriginalHTTPPrueba{}
	h, err := NuevoManejadorOriginalFirmableCT(autoridadRegistroFirmaVecPrueba{organizacion: "organizacion:desarrollo:dipgra"}, s)
	if err != nil {
		t.Fatal(err)
	}
	var previo string
	for range 2 {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionOriginalHTTPPrueba(cuerpoOriginalHTTPPrueba))
		var salida struct {
			Data map[string]any `json:"data"`
		}
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &salida) != nil {
			t.Fatalf("original: %d %s", w.Code, w.Body)
		}
		d := salida.Data
		pdf, err := base64.StdEncoding.Strict().DecodeString(d["pdf_base64"].(string))
		if err != nil || string(pdf) != "%PDF-1.7\n%%EOF" || d["esquema"] != EsquemaOriginalFirmableCT || d["original_version"] != float64(7) || s.solicitud.OriginalVersion != 7 || s.solicitud.OriginalRef != "" {
			t.Fatalf("vínculo perdido: %s", w.Body)
		}
		if previo != "" && previo != w.Body.String() {
			t.Fatal("replay alteró original")
		}
		previo = w.Body.String()
		if !strings.Contains(w.Header().Get("Cache-Control"), "no-store") || w.Header().Get("Set-Cookie") != "" {
			t.Fatal("cabeceras inseguras")
		}
		if strings.Contains(w.Body.String(), "recibo_ref") || strings.Contains(w.Body.String(), "actor_ref") {
			t.Fatal("metadato inventado")
		}
	}
	if s.llamadas != 2 {
		t.Fatal("replay no revalidó servicio")
	}
}

func TestOriginalFirmableHTTPNoConfiaEnClienteNiResultado(t *testing.T) {
	for _, body := range []string{
		strings.Replace(cuerpoOriginalHTTPPrueba, "version_observada", "VERSION_OBSERVADA", 1),
		strings.TrimSuffix(cuerpoOriginalHTTPPrueba, "}") + `,"original_version":1}`,
		strings.TrimSuffix(cuerpoOriginalHTTPPrueba, "}") + `,"actor_ref":"actor:ajeno"}`,
		strings.TrimSuffix(cuerpoOriginalHTTPPrueba, "}") + `,"version_observada":8}`,
		strings.Replace(cuerpoOriginalHTTPPrueba, ":7", ":9007199254740992", 1),
	} {
		s := &servicioOriginalHTTPPrueba{}
		h, _ := NuevoManejadorOriginalFirmableCT(autoridadRegistroFirmaVecPrueba{organizacion: "organizacion:desarrollo:dipgra"}, s)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionOriginalHTTPPrueba(body))
		if w.Code < 400 || s.llamadas != 0 {
			t.Fatalf("entrada confiada: %d %s", w.Code, w.Body)
		}
	}
	for _, alterar := range []func(*vecports.OriginalFirmableCT){
		func(r *vecports.OriginalFirmableCT) { r.Version++ },
		func(r *vecports.OriginalFirmableCT) { r.Referencia = "ref:" + strings.Repeat("f", 64) },
		func(r *vecports.OriginalFirmableCT) { r.Contenido[0] = 'x' },
		func(r *vecports.OriginalFirmableCT) { r.TipoRef = "tipo:inventado" },
	} {
		s := &servicioOriginalHTTPPrueba{alterar: alterar}
		h, _ := NuevoManejadorOriginalFirmableCT(autoridadRegistroFirmaVecPrueba{organizacion: "organizacion:desarrollo:dipgra"}, s)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionOriginalHTTPPrueba(cuerpoOriginalHTTPPrueba))
		if w.Code != 502 || strings.Contains(w.Body.String(), "pdf_base64") {
			t.Fatalf("respuesta inyectada: %d %s", w.Code, w.Body)
		}
	}
}

func TestOriginalFirmableHTTPLimiteYDenegacion(t *testing.T) {
	pdf := append([]byte("%PDF-1.7"), bytes.Repeat([]byte("x"), vecports.LimiteOriginalFirmableCT-8)...)
	s := &servicioOriginalHTTPPrueba{contenido: pdf}
	h, _ := NuevoManejadorOriginalFirmableCT(autoridadRegistroFirmaVecPrueba{organizacion: "organizacion:desarrollo:dipgra"}, s)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionOriginalHTTPPrueba(cuerpoOriginalHTTPPrueba))
	if w.Code != 200 {
		t.Fatalf("límite válido rechazado: %d %s", w.Code, w.Body)
	}
	s.contenido = append(pdf, 'x')
	w = httptest.NewRecorder()
	h.ServeHTTP(w, peticionOriginalHTTPPrueba(cuerpoOriginalHTTPPrueba))
	if w.Code != 502 {
		t.Fatal("límite excedido aceptado")
	}
	s.err = errors.New("private certificate secret")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, peticionOriginalHTTPPrueba(cuerpoOriginalHTTPPrueba))
	if w.Code != 503 || strings.Contains(w.Body.String(), "private") || strings.Contains(w.Body.String(), "pdf_base64") {
		t.Fatal("dependencia abierta")
	}
}

func TestFirmaHTTPLogUsaContextoYCorrelacionComun(t *testing.T) {
	anterior := slog.Default()
	defer slog.SetDefault(anterior)
	var log bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&log, nil)))
	ctx, err := vecports.ConCorrelacionIncidenciasPeticion(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	ref, _ := vecports.CorrelacionIncidenciasPeticion(ctx)
	r := peticionRegistroFirmaVec(cuerpoRegistroFirmaVecPrueba()).WithContext(ctx)
	r.Header.Set("X-Correlation-ID", "correlacion_forjada")
	w := httptest.NewRecorder()
	responderErrorRegistroFirmaVec(w, r, 503, "servicio_no_disponible", errors.New("PRIVATE-PDF-SECRET"))
	if !strings.Contains(w.Body.String(), "correlacion_"+ref) || !strings.Contains(log.String(), "correlacion_"+ref) || strings.Contains(log.String(), "PRIVATE") || strings.Contains(log.String(), "forjada") {
		t.Fatalf("correlación: %s %s", w.Body, &log)
	}
	w = httptest.NewRecorder()
	responderErrorRegistroFirmaVec(w, peticionRegistroFirmaVec(cuerpoRegistroFirmaVecPrueba()), 503, "servicio_no_disponible")
	if !strings.Contains(w.Body.String(), "corr_no_disponible") {
		t.Fatal("se inventó otra correlación")
	}
}
