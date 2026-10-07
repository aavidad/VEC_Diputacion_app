package httpinterno

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"mime"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
)

type exportadorServiciosHTTPPrueba struct {
	err      error
	llamadas int
	in       domain.SolicitudExportacionServiciosPropios
	alterar  bool
}

func (e *exportadorServiciosHTTPPrueba) Exportar(_ context.Context, in domain.SolicitudExportacionServiciosPropios) (ports.ResultadoExportacionServiciosPropios, error) {
	e.llamadas++
	e.in = in
	if e.err != nil {
		return ports.ResultadoExportacionServiciosPropios{}, e.err
	}
	b := []byte("c1,c2,c3,c4,c5\n2019-01-01,2019-12-31,fuente,365,reconocido\n")
	h := sha256.Sum256(b)
	sha := hex.EncodeToString(h[:])
	if e.alterar {
		sha = strings.Repeat("a", 64)
	}
	return ports.ResultadoExportacionServiciosPropios{Corte: in.Corte, ContenidoCSV: b, ContenidoSHA256: sha, NombreArchivo: "servicios.csv", Evidencia: ports.EvidenciaRegistroEmpleadoB2{ReciboRef: in.ReciboRef}}, nil
}

type registroExportHTTPPrueba struct {
	err, errorPreflight error
	intentos            []ports.IntentoFichaPropia
}

func (r *registroExportHTTPPrueba) VerificarRegistroExportacionServiciosPropios(context.Context) error {
	return r.errorPreflight
}
func (r *registroExportHTTPPrueba) RegistrarIntentoExportacionServiciosPropios(_ context.Context, in ports.IntentoFichaPropia) error {
	r.intentos = append(r.intentos, in)
	return r.err
}

const cuerpoExportHTTPPrueba = `{"recibo_ref":"fichapropia:0f0e0d0c-0b0a-4908-8706-050403020100","corte":{"vigente_en":"2026-09-25","conocido_en":"2026-09-25T09:59:59.000000Z"},"idioma":"xx"}`

func peticionExportHTTPPrueba(body string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, RutaExportacionServiciosPropios, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	return r
}
func manejadorExportHTTPPrueba(t *testing.T, e *exportadorServiciosHTTPPrueba, r *registroExportHTTPPrueba) *ManejadorExportacionServiciosPropios {
	t.Helper()
	m, err := NuevoManejadorExportacionServiciosPropios(actorFichaPropiaHTTP{actor: actorFichaPropiaPruebaHTTP(t)}, e, r)
	if err != nil {
		t.Fatal(err)
	}
	return m
}
func TestExportacionServiciosHTTPSirveBytesOriginalesConHuellaYRecibo(t *testing.T) {
	e := &exportadorServiciosHTTPPrueba{}
	registro := &registroExportHTTPPrueba{}
	m := manejadorExportHTTPPrueba(t, e, registro)
	w := httptest.NewRecorder()
	m.ServeHTTP(w, peticionExportHTTPPrueba(cuerpoExportHTTPPrueba))
	h := sha256.Sum256(w.Body.Bytes())
	tipo, params, err := mime.ParseMediaType(w.Header().Get("Content-Disposition"))
	if w.Code != 200 || w.Header().Get("Content-Type") != "text/csv; charset=utf-8" || w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("Content-Length") != strconv.Itoa(w.Body.Len()) || w.Header().Get("X-Content-SHA256") != hex.EncodeToString(h[:]) || w.Header().Get("X-Recibo-Ref") != e.in.ReciboRef || err != nil || tipo != "attachment" || params["filename"] != "servicios.csv" || e.llamadas != 1 || len(registro.intentos) != 0 {
		t.Fatalf("descarga no original %d %v", w.Code, w.Header())
	}
	if w.Header().Get("Set-Cookie") != "" || w.Header().Get("Access-Control-Allow-Origin") != "" || w.Header().Get("Location") != "" {
		t.Fatal("cabeceras no admitidas")
	}
}
func TestExportacionServiciosHTTPRechazaFilasActorCamposAjenosDuplicadosYCorteNulo(t *testing.T) {
	cuerpos := []string{strings.Replace(cuerpoExportHTTPPrueba, `"idioma":"xx"`, `"idioma":"xx","actor_ref":"per_ajeno"`, 1), strings.Replace(cuerpoExportHTTPPrueba, `"idioma":"xx"`, `"idioma":"xx","servicios":[]`, 1), strings.Replace(cuerpoExportHTTPPrueba, `"idioma":"xx"`, `"idioma":"xx","idioma":"yy"`, 1), strings.Replace(cuerpoExportHTTPPrueba, `"conocido_en":"2026-09-25T09:59:59.000000Z"`, `"conocido_en":null`, 1), strings.Replace(cuerpoExportHTTPPrueba, `"idioma":"xx"`, `"Idioma":"xx"`, 1), cuerpoExportHTTPPrueba + `{}`, "null"}
	for _, body := range cuerpos {
		t.Run(body, func(t *testing.T) {
			e := &exportadorServiciosHTTPPrueba{}
			registro := &registroExportHTTPPrueba{}
			m := manejadorExportHTTPPrueba(t, e, registro)
			w := httptest.NewRecorder()
			m.ServeHTTP(w, peticionExportHTTPPrueba(body))
			if w.Code != 400 || e.llamadas != 0 || len(registro.intentos) != 1 || registro.intentos[0].Motivo != "entrada_invalida" || w.Header().Get("Content-Disposition") != "" {
				t.Fatalf("entrada ajena no cerrada %d %s", w.Code, w.Body.String())
			}
		})
	}
}
func TestExportacionServiciosHTTPServicioYaAuditadoNoDuplicaIntento(t *testing.T) {
	for _, caso := range []struct {
		err    error
		estado int
	}{{domain.ErrExportacionServiciosPropiosInvalida, 400}, {domain.ErrExportacionServiciosPropiosDenegada, 403}, {domain.ErrExportacionServiciosPropiosNoDisponible, 503}} {
		e := &exportadorServiciosHTTPPrueba{err: caso.err}
		registro := &registroExportHTTPPrueba{}
		m := manejadorExportHTTPPrueba(t, e, registro)
		w := httptest.NewRecorder()
		m.ServeHTTP(w, peticionExportHTTPPrueba(cuerpoExportHTTPPrueba))
		if w.Code != caso.estado || len(registro.intentos) != 0 || w.Header().Get("Content-Type") == "text/csv; charset=utf-8" {
			t.Fatal("duplicó fallo o sirvió CSV", w.Code)
		}
	}
}
func TestExportacionServiciosHTTPSinIdentidadORegistroNoExponeCSV(t *testing.T) {
	e := &exportadorServiciosHTTPPrueba{}
	registro := &registroExportHTTPPrueba{err: errors.New("detalle privado")}
	m := manejadorExportHTTPPrueba(t, e, registro)
	r := peticionExportHTTPPrueba("{}")
	w := httptest.NewRecorder()
	m.ServeHTTP(w, r)
	if w.Code != 503 || e.llamadas != 0 || strings.Contains(w.Body.String(), "privado") {
		t.Fatal("registro opcional")
	}
	m.actor = actorFichaPropiaHTTP{err: errors.New("sin captura")}
	w = httptest.NewRecorder()
	m.ServeHTTP(w, peticionExportHTTPPrueba(cuerpoExportHTTPPrueba))
	if w.Code != 503 || e.llamadas != 0 || len(registro.intentos) != 1 {
		t.Fatal("identidad inventada")
	}
}
func TestExportacionServiciosHTTPNoSirveContenidoAlterado(t *testing.T) {
	e := &exportadorServiciosHTTPPrueba{alterar: true}
	registro := &registroExportHTTPPrueba{}
	m := manejadorExportHTTPPrueba(t, e, registro)
	w := httptest.NewRecorder()
	m.ServeHTTP(w, peticionExportHTTPPrueba(cuerpoExportHTTPPrueba))
	if w.Code != 503 || len(registro.intentos) != 1 || w.Header().Get("Content-Disposition") != "" {
		t.Fatal("CSV alterado servido")
	}
}
