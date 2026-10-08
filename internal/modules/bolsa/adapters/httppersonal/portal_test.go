package httppersonal

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/application/mibolsa"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

func TestAccionesPortalPausaAusenteEsNullExplicito(t *testing.T) {
	reglas := &puertosbolsa.ReglasPortalVisibles{CausasRenuncia: []string{"enfermedad"},
		ModoRespuesta: puertosbolsa.ModoRespuestaPortalFirme}
	_, acciones := respuestaPortal(puertosbolsa.InstantaneaMiBolsa{ReglasPortal: reglas})
	contenido, err := json.Marshal(acciones)
	if err != nil || !strings.Contains(string(contenido), `"pausa_maxima":null`) {
		t.Fatalf("la ausencia de pausa no quedó explícita: %v", err)
	}
	fecha := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	reglas.PausaMaxima = &fecha
	_, acciones = respuestaPortal(puertosbolsa.InstantaneaMiBolsa{ReglasPortal: reglas})
	contenido, err = json.Marshal(acciones)
	if err != nil || !strings.Contains(string(contenido), `"pausa_maxima":"2027-01-01T00:00:00.000000Z"`) {
		t.Fatalf("la pausa vigente perdió su fecha: %v", err)
	}
}

type ejecutorPortalPrueba struct {
	llamadas   int
	hasta      time.Time
	comando    mibolsa.ComandoRespuestaPortal
	documental mibolsa.ComandoSolicitudDocumentalPortal
	err        error
	repetida   bool
}

func (e *ejecutorPortalPrueba) PresentarSolicitudDocumental(_ context.Context, _ mibolsa.Orden, c mibolsa.ComandoSolicitudDocumentalPortal) (puertosbolsa.ReciboSolicitudDocumentalPortal, error) {
	e.llamadas++
	e.documental = c
	return puertosbolsa.ReciboSolicitudDocumentalPortal{
		SolicitudRef:    "solicitud-documental:" + strings.Repeat("a", 64),
		ReciboRef:       "recibo:solicitud-documental:" + strings.Repeat("b", 64),
		ContenidoSHA256: strings.Repeat("c", 64), RegistradaEn: time.Date(2026, 10, 2, 10, 0, 0, 0, time.UTC),
		Version: 1, Estado: "pendiente_rrhh", Reutilizada: e.repetida,
	}, e.err
}

func (e *ejecutorPortalPrueba) SolicitarPausa(_ context.Context, _ mibolsa.Orden, bolsa string, hasta time.Time, clave string) (puertosbolsa.ReciboSolicitudPortal, error) {
	e.llamadas++
	e.hasta = hasta
	return puertosbolsa.ReciboSolicitudPortal{Reutilizada: e.repetida, SolicitudRef: "solicitud-portal:x", ReciboRef: "recibo:solicitud-portal:x", RegistradaEn: time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)}, e.err
}

func (e *ejecutorPortalPrueba) SolicitarReactivacion(_ context.Context, _ mibolsa.Orden, _, _ string) (puertosbolsa.ReciboSolicitudPortal, error) {
	e.llamadas++
	return puertosbolsa.ReciboSolicitudPortal{SolicitudRef: "solicitud-portal:y", ReciboRef: "recibo:solicitud-portal:y", RegistradaEn: time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)}, e.err
}

func (e *ejecutorPortalPrueba) Responder(_ context.Context, _ mibolsa.Orden, c mibolsa.ComandoRespuestaPortal) (puertosbolsa.ReciboRespuestaPortal, error) {
	e.llamadas++
	e.comando = c
	return puertosbolsa.ReciboRespuestaPortal{RespuestaRef: "respuesta-portal:x", ReciboRef: "recibo:respuesta-portal:x", Modo: "firme", RespondidaEn: time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC), VenceAntesDe: time.Date(2026, 9, 26, 22, 0, 0, 0, time.UTC)}, e.err
}

func peticionPortal(ruta, cuerpo string, cabeceras map[string]string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, ruta, strings.NewReader(cuerpo))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json")
	for k, v := range cabeceras {
		r.Header.Set(k, v)
	}
	return r
}

func TestPortalRecuperaSolicitudesHistoricasYConservaRespuestas(t *testing.T) {
	e := new(ejecutorPortalPrueba)
	h, err := NuevoPortal(RutaMiBolsaSolicitudes, new(preparadorPrueba), e)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionPortal(RutaMiBolsaSolicitudes, `{"tipo":"pausa","bolsa":"bolsa:01","pausa_hasta":"2026-12-31T22:59:59Z","clave":"clave-pausa-0001"}`, nil))
	if w.Code != http.StatusCreated || e.llamadas != 1 {
		t.Fatalf("ruta heredada no alcanza el contraste durable de replay: %d %s", w.Code, w.Body.String())
	}
	r, _ := NuevoPortal(RutaMiBolsaRespuestas, new(preparadorPrueba), e)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, peticionPortal(RutaMiBolsaRespuestas, `{"bolsa":"bolsa:01","respuesta":"renuncia_justificada","causa":"enfermedad","justificante_ref":"justificante:1","justificante_sha256":"`+strings.Repeat("a", 64)+`","clave":"clave-respuesta-1"}`, nil))
	if w.Code != 201 || e.comando.Causa != "enfermedad" || !strings.Contains(w.Body.String(), `"estado":"firme"`) || !strings.Contains(w.Body.String(), `"vence_antes_de"`) {
		t.Fatalf("respuesta: %d %s", w.Code, w.Body.String())
	}
}

func TestPortalSolicitudDocumentalSoloPropiaYConRecibo(t *testing.T) {
	e := new(ejecutorPortalPrueba)
	h, err := NuevoPortal(RutaMiBolsaSolicitudesDocumentales, new(preparadorPrueba), e)
	if err != nil {
		t.Fatal(err)
	}
	cuerpo := `{"tipo":"documental_rrhh","bolsa":"bolsa:01","documento_ref":"documento:parte-1","documento_sha256":"` + strings.Repeat("a", 64) + `","fecha_fin_causa":"2026-10-02","clave":"clave-documental-1"}`
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionPortal(RutaMiBolsaSolicitudesDocumentales, cuerpo, nil))
	if w.Code != http.StatusCreated || e.llamadas != 1 || e.documental.DocumentoRef != "documento:parte-1" ||
		!strings.Contains(w.Body.String(), `"estado":"pendiente_rrhh"`) || !strings.Contains(w.Body.String(), `"contenido_sha256"`) {
		t.Fatalf("solicitud documental: %d %s", w.Code, w.Body.String())
	}
	for _, invalido := range []string{
		strings.Replace(cuerpo, `"tipo":"documental_rrhh"`, `"tipo":"pausa"`, 1),
		strings.Replace(cuerpo, `"clave":"clave-documental-1"`, `"actor":"persona:ajena","clave":"clave-documental-1"`, 1),
		strings.Replace(cuerpo, `"documento_ref":"documento:parte-1"`, `"documento_ref":"dni:prueba"`, 1),
	} {
		w = httptest.NewRecorder()
		h.ServeHTTP(w, peticionPortal(RutaMiBolsaSolicitudesDocumentales, invalido, nil))
		if w.Code != http.StatusBadRequest || e.llamadas != 1 {
			t.Fatalf("entrada ajena admitida: %d", w.Code)
		}
	}
	if acciones := AccionPortalEn(http.MethodPost, RutaMiBolsaSolicitudesDocumentales); len(acciones) != 1 || acciones[0] != puertosbolsa.AccionPresentarSolicitudDocumentalPropia {
		t.Fatalf("acción documental no nominal: %v", acciones)
	}
}

func TestPortalRechazaPeticionesNoSegurasSinEjecutar(t *testing.T) {
	e := new(ejecutorPortalPrueba)
	h, _ := NuevoPortal(RutaMiBolsaSolicitudes, new(preparadorPrueba), e)
	valido := `{"tipo":"reactivacion","bolsa":"bolsa:01","clave":"clave-reactiva-01"}`
	for nombre, r := range map[string]*http.Request{
		"otro origen":      peticionPortal(RutaMiBolsaSolicitudes, valido, map[string]string{"Sec-Fetch-Site": "cross-site"}),
		"formulario":       peticionPortal(RutaMiBolsaSolicitudes, valido, map[string]string{"Content-Type": "text/plain"}),
		"cookie":           peticionPortal(RutaMiBolsaSolicitudes, valido, map[string]string{"Cookie": "a=b"}),
		"campo ajeno":      peticionPortal(RutaMiBolsaSolicitudes, `{"tipo":"reactivacion","bolsa":"bolsa:01","clave":"clave-reactiva-01","candidato":"can_x"}`, nil),
		"pausa sin fin":    peticionPortal(RutaMiBolsaSolicitudes, `{"tipo":"pausa","bolsa":"bolsa:01","clave":"clave-pausa-0001"}`, nil),
		"reactiva con fin": peticionPortal(RutaMiBolsaSolicitudes, `{"tipo":"reactivacion","bolsa":"bolsa:01","pausa_hasta":"2026-12-31T22:59:59Z","clave":"clave-reactiva-01"}`, nil),
		"consulta":         peticionPortal(RutaMiBolsaSolicitudes+"?x=1", valido, nil),
	} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code < 400 || w.Code >= 500 {
			t.Fatalf("%s: status %d", nombre, w.Code)
		}
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaMiBolsaSolicitudes, nil))
	if w.Code != 405 || e.llamadas != 0 {
		t.Fatalf("método o ejecución indebida: %d %d", w.Code, e.llamadas)
	}
}

func TestPortalTraduceConflictosDeNegocio(t *testing.T) {
	for err, codigo := range map[error]string{
		puertosbolsa.ErrPortalSolicitudPendiente:    "solicitud_pendiente",
		puertosbolsa.ErrPortalSituacionNoAdmite:     "situacion_no_admite",
		puertosbolsa.ErrPortalSinLlamamientoAbierto: "sin_llamamiento_abierto",
		puertosbolsa.ErrPortalRespuestaFueraDePlazo: "fuera_de_plazo",
		puertosbolsa.ErrPortalClaveReutilizada:      "clave_reutilizada",
	} {
		h, _ := NuevoPortal(RutaMiBolsaRespuestas, new(preparadorPrueba), &ejecutorPortalPrueba{err: err})
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionPortal(RutaMiBolsaRespuestas, `{"bolsa":"bolsa:01","respuesta":"acepta","clave":"clave-respuesta-01"}`, nil))
		if w.Code != 409 || !strings.Contains(w.Body.String(), codigo) {
			t.Fatalf("%v: %d %s", err, w.Code, w.Body.String())
		}
	}
	h, _ := NuevoPortal(RutaMiBolsaRespuestas, new(preparadorPrueba), &ejecutorPortalPrueba{err: puertosbolsa.ErrPortalCandidatoNoDisponible})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionPortal(RutaMiBolsaRespuestas, `{"bolsa":"bolsa:01","respuesta":"acepta","clave":"clave-respuesta-01"}`, nil))
	if w.Code != 503 {
		t.Fatalf("indisponibilidad: %d", w.Code)
	}
	if AccionPortalEn(http.MethodPost, RutaMiBolsaRespuestas)[0] != puertosbolsa.AccionResponderLlamamientoPropio || AccionPortalEn(http.MethodGet, RutaMiBolsaRespuestas) != nil || !EsRutaPortal(RutaMiBolsa) {
		t.Fatal("rutas y acciones del portal")
	}
}
