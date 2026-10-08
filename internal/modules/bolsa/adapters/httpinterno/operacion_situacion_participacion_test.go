package httpinterno

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type proyectorFichaOperacionesHTTPPrueba struct {
	llamadas  int
	err       error
	resultado DisponibilidadFichaOperaciones
}

func (p *proyectorFichaOperacionesHTTPPrueba) ProyectarDisponibilidadFichaOperaciones(_ context.Context, _ ports.SolicitudCambiarSituacionParticipacion) (DisponibilidadFichaOperaciones, error) {
	p.llamadas++
	return p.resultado, p.err
}

func TestOperacionesSituaFichaSoloTrasLecturaYConservaHistorial(t *testing.T) {
	var registro bytes.Buffer
	anterior := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&registro, nil)))
	t.Cleanup(func() { slog.SetDefault(anterior) })
	ruta := RutaBolsasGestion + "/bolsa:01/candidatos/participacion:01/operaciones"
	get := func(h http.Handler) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodGet, ruta, nil)
		r.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	base := EstadoDisponibilidadFichaOperaciones{Estado: "disponible", BolsaRef: "bolsa:01", ParticipacionRef: "participacion:01"}
	p := &proyectorFichaOperacionesHTTPPrueba{resultado: DisponibilidadFichaOperaciones{
		SolicitudesDocumentales:  base,
		ReincorporacionesTitular: EstadoDisponibilidadFichaOperaciones{Estado: "sin_montaje", BolsaRef: base.BolsaRef, ParticipacionRef: base.ParticipacionRef},
	}}
	h, err := NuevoHandlerOperacionesSituacion(preparadorSituacionHTTPPrueba{}, operadorOperacionesHTTPPrueba{items: []ports.RegistroOperacionSituacion{}}, p)
	if err != nil {
		t.Fatal(err)
	}
	w := get(h)
	if w.Code != 200 || p.llamadas != 1 || !strings.Contains(w.Body.String(), `"items":[]`) ||
		!strings.Contains(w.Body.String(), `"solicitudes_documentales":{"estado":"disponible","bolsa_ref":"bolsa:01","participacion_ref":"participacion:01"}`) ||
		!strings.Contains(w.Body.String(), `"reincorporaciones_titular":{"estado":"sin_montaje"`) {
		t.Fatalf("ficha tras GET: %d %s llamadas=%d", w.Code, w.Body.String(), p.llamadas)
	}
	p.err = errors.New("fuente caída con dato-privado-y-dsn")
	w = get(h)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"items":[]`) || !strings.Contains(w.Body.String(), `"estado":"indisponible"`) ||
		!strings.Contains(registro.String(), `"codigo":"dependencia_indisponible"`) || strings.Contains(registro.String(), "dato-privado-y-dsn") {
		t.Fatalf("fallo opcional destruyó historial: %d %s", w.Code, w.Body.String())
	}
	p.err = nil
	p.resultado.SolicitudesDocumentales.ParticipacionRef = "participacion:otra"
	w = get(h)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"items":[]`) || !strings.Contains(w.Body.String(), `"estado":"indisponible"`) ||
		!strings.Contains(registro.String(), `"codigo":"forma_incompatible"`) {
		t.Fatalf("forma incompatible sin registro técnico: %d %s log=%s", w.Code, w.Body.String(), registro.String())
	}
	p.resultado = DisponibilidadFichaOperaciones{
		SolicitudesDocumentales:  EstadoDisponibilidadFichaOperaciones{Estado: "indisponible", BolsaRef: "bolsa:01", ParticipacionRef: "participacion:01"},
		ReincorporacionesTitular: EstadoDisponibilidadFichaOperaciones{Estado: "sin_montaje", BolsaRef: "bolsa:01", ParticipacionRef: "participacion:01"},
	}
	w = get(h)
	if w.Code != 200 || !strings.Contains(registro.String(), `"codigo":"fuente_o_montaje_indisponible"`) {
		t.Fatalf("indisponibilidad sin registro técnico: %d %s log=%s", w.Code, w.Body.String(), registro.String())
	}
	h, _ = NuevoHandlerOperacionesSituacion(preparadorSituacionHTTPPrueba{}, operadorOperacionesHTTPPrueba{err: dominiovec.ErrAutorizacionDenegada}, p)
	w = get(h)
	if w.Code != 403 || p.llamadas != 4 || strings.Contains(w.Body.String(), "capacidades_ficha") {
		t.Fatalf("denegación emitió metadata: %d %s llamadas=%d", w.Code, w.Body.String(), p.llamadas)
	}
	h, _ = NuevoHandlerOperacionesSituacion(preparadorSituacionHTTPPrueba{}, operadorOperacionesHTTPPrueba{items: []ports.RegistroOperacionSituacion{}})
	w = get(h)
	if w.Code != 200 || strings.Contains(w.Body.String(), "capacidades_ficha") {
		t.Fatalf("constructor legado añadió metadata: %d %s", w.Code, w.Body.String())
	}
}

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
