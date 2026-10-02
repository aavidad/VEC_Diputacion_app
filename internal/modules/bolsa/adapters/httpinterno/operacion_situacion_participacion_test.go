package httpinterno

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type operadorOperacionesHTTPPrueba struct {
	resultado ports.RegistroSituacionParticipacion
	items     []ports.RegistroOperacionSituacion
	err       error
	recibida  *ports.SolicitudOperacionSituacion
}

type operadorHistorialOperacionHTTPPrueba struct {
	operadorOperacionesHTTPPrueba
	vigente ports.SituacionParticipacion
}

func (o operadorHistorialOperacionHTTPPrueba) ListarHistorial(context.Context, ports.SolicitudCambiarSituacionParticipacion) (ports.HistorialParticipacion, error) {
	return ports.HistorialParticipacion{Operaciones: []ports.RegistroOperacionSituacion{}, Vigente: o.vigente}, nil
}

func (o operadorOperacionesHTTPPrueba) Operar(_ context.Context, q ports.SolicitudOperacionSituacion) (ports.RegistroSituacionParticipacion, error) {
	if o.recibida != nil {
		*o.recibida = q
	}
	return o.resultado, o.err
}

func TestHistorialOperacionEntregaSituacionVigenteParaCAS(t *testing.T) {
	ruta := RutaBolsasGestion + "/bolsa:01/candidatos/participacion:01/operaciones"
	desde := time.Date(2026, 10, 2, 9, 0, 0, 0, time.UTC)
	h, _ := NuevoHandlerOperacionesSituacion(preparadorSituacionHTTPPrueba{}, operadorHistorialOperacionHTTPPrueba{
		vigente: ports.SituacionParticipacion{ParticipacionRef: "participacion:01", Situacion: "en_revision", Desde: desde},
	})
	r := httptest.NewRequest(http.MethodGet, ruta, nil)
	r.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"situacion_vigente":{"desde":"2026-10-02T09:00:00Z","fecha_disponible":null,"situacion":"en_revision"}`) {
		t.Fatalf("CAS de historial: status=%d body=%s", w.Code, w.Body.String())
	}
}
func (o operadorOperacionesHTTPPrueba) ListarOperaciones(context.Context, ports.SolicitudCambiarSituacionParticipacion) ([]ports.RegistroOperacionSituacion, error) {
	return o.items, o.err
}

func TestOperacionDocumentalHTTPConservaSolicitudYFechaCivil(t *testing.T) {
	ruta := RutaBolsasGestion + "/bolsa:01/candidatos/participacion:01/operaciones"
	var recibida ports.SolicitudOperacionSituacion
	o := operadorOperacionesHTTPPrueba{recibida: &recibida, resultado: ports.RegistroSituacionParticipacion{
		ReciboRef: "recibo:regularizacion", ReciboResolucionRef: "recibo:solicitud-documental-resolucion:uno",
		ResueltaEn: func() *time.Time { t := time.Date(2026, 10, 2, 9, 0, 0, 0, time.UTC); return &t }(),
		SituacionParticipacion: ports.SituacionParticipacion{
			Situacion: "disponible", Desde: time.Date(2026, 10, 2, 9, 0, 0, 0, time.UTC)}}}
	h, _ := NuevoHandlerOperacionesSituacion(preparadorSituacionHTTPPrueba{}, o)
	cuerpo := `{"operacion":"regularizar","motivo":"Documento validado","validador":"persona:rrhh",` +
		`"situacion_esperada_desde":"2026-10-01T08:00:00Z","causa_finalizada_en":"2026-10-01",` +
		`"solicitud_ref":"solicitud-documental:` + strings.Repeat("b", 64) + `",` +
		`"solicitud_version_esperada":1,"solicitud_contenido_sha256":"` + strings.Repeat("c", 64) + `",` +
		`"justificante":{"tipo":"solicitud_candidato","referencia":"documento:fin-causa","sha256":"` + strings.Repeat("a", 64) + `"}}`
	r := httptest.NewRequest(http.MethodPost, ruta, strings.NewReader(cuerpo))
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Idempotency-Key", "regularizacion-01")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	madrid, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != 201 || !strings.Contains(w.Body.String(), `"recibo_resolucion_ref":"recibo:solicitud-documental-resolucion:uno"`) ||
		recibida.SolicitudVersionEsperada != 1 || recibida.SolicitudRef != "solicitud-documental:"+strings.Repeat("b", 64) ||
		recibida.CausaFinalizadaEn == nil || recibida.CausaFinalizadaEn.In(madrid).Format("2006-01-02") != "2026-10-01" {
		t.Fatalf("solicitud documental: status=%d recibida=%+v", w.Code, recibida)
	}
}

func TestOperacionesSituacionContratoHTTP(t *testing.T) {
	ruta := RutaBolsasGestion + "/bolsa:01/candidatos/participacion:01/operaciones"
	ahora := time.Date(2026, 9, 23, 9, 0, 0, 0, time.UTC)
	cuerpo := `{"operacion":"pausar","motivo":"Solicitud registrada","validador":"per_validadora","justificante":{"tipo":"solicitud_candidato","referencia":"justificante:01","sha256":"` + strings.Repeat("a", 64) + `"}}`
	for _, caso := range []struct {
		reutilizada bool
		estado      int
	}{{false, 201}, {true, 200}} {
		o := operadorOperacionesHTTPPrueba{resultado: ports.RegistroSituacionParticipacion{Reutilizada: caso.reutilizada, ReciboRef: "recibo:01", SituacionParticipacion: ports.SituacionParticipacion{Situacion: "no_disponible", Desde: ahora}}}
		h, _ := NuevoHandlerOperacionesSituacion(preparadorSituacionHTTPPrueba{}, o)
		r := httptest.NewRequest(http.MethodPost, ruta, strings.NewReader(cuerpo))
		r.Header.Set("Accept", "application/json")
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Idempotency-Key", "b8-01")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != caso.estado || !strings.Contains(w.Body.String(), `"recibo_ref":"recibo:01"`) {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
	}
	h, _ := NuevoHandlerOperacionesSituacion(preparadorSituacionHTTPPrueba{}, operadorOperacionesHTTPPrueba{items: []ports.RegistroOperacionSituacion{}})
	r := httptest.NewRequest(http.MethodGet, ruta, nil)
	r.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"items":[]`) {
		t.Fatalf("GET status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestOperacionesSituacionDenegacionSinDatos(t *testing.T) {
	ruta := RutaBolsasGestion + "/bolsa:01/candidatos/participacion:01/operaciones"
	h, _ := NuevoHandlerOperacionesSituacion(preparadorSituacionHTTPPrueba{}, operadorOperacionesHTTPPrueba{err: dominiovec.ErrAutorizacionDenegada})
	r := httptest.NewRequest(http.MethodGet, ruta, nil)
	r.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 403 || strings.Contains(w.Body.String(), "participacion:01") {
		t.Fatalf("GET status=%d body=%s", w.Code, w.Body.String())
	}
}
