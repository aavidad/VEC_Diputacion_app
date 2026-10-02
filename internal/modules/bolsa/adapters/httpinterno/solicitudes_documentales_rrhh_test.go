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

type consultorDocumentalesPrueba struct {
	items    []ports.SolicitudDocumentalPendienteRRHH
	err      error
	llamadas int
}

func (c *consultorDocumentalesPrueba) ListarSolicitudesDocumentalesRRHH(_ context.Context, q ports.SolicitudCambiarSituacionParticipacion) ([]ports.SolicitudDocumentalPendienteRRHH, error) {
	c.llamadas++
	if q.BolsaRef != "bolsa:01" || q.ParticipacionRef != "participacion:01" {
		return nil, dominiovec.ErrAutorizacionDenegada
	}
	return c.items, c.err
}

func TestSolicitudesDocumentalesRRHHFiltraYSoloMuestraPendientes(t *testing.T) {
	consulta := &consultorDocumentalesPrueba{items: []ports.SolicitudDocumentalPendienteRRHH{{
		SolicitudRef: "solicitud-documental:" + strings.Repeat("a", 64), Version: 1, ContenidoSHA256: strings.Repeat("b", 64),
		DocumentoRef: "documento:parte-1", DocumentoSHA256: strings.Repeat("c", 64), FechaFinCausa: "2026-10-02",
		Estado: "pendiente_rrhh", ReciboRef: "recibo:solicitud-documental:" + strings.Repeat("d", 64), RegistradaEn: time.Date(2026, 10, 2, 10, 0, 0, 0, time.UTC),
	}}}
	h, err := NuevoHandlerSolicitudesDocumentalesRRHH(preparadorSituacionHTTPPrueba{}, consulta)
	if err != nil {
		t.Fatal(err)
	}
	ruta := RutaSolicitudesDocumentalesPendientesRRHH + "?bolsa_ref=bolsa%3A01&participacion_ref=participacion%3A01"
	r := httptest.NewRequest(http.MethodGet, ruta, nil)
	r.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK || consulta.llamadas != 1 || !strings.Contains(w.Body.String(), `"esquema":"vec.bolsa.rrhh.solicitudes_documentales.v1"`) ||
		!strings.Contains(w.Body.String(), `"documento_ref":"documento:parte-1"`) || strings.Contains(w.Body.String(), `"candidato_ref"`) {
		t.Fatalf("lectura focal: %d %s", w.Code, w.Body.String())
	}
	for _, rutaMala := range []string{RutaSolicitudesDocumentalesPendientesRRHH + "?bolsa_ref=bolsa%3A01", ruta + "&candidato_ref=can_ajeno", ruta + "&bolsa_ref=bolsa%3A02"} {
		w = httptest.NewRecorder()
		r = httptest.NewRequest(http.MethodGet, rutaMala, nil)
		r.Header.Set("Accept", "application/json")
		h.ServeHTTP(w, r)
		if w.Code != http.StatusBadRequest || consulta.llamadas != 1 {
			t.Fatalf("filtro alterado admitido: %d", w.Code)
		}
	}
	consulta.items[0].FechaFinCausa = ""
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, ruta, nil)
	r.Header.Set("Accept", "application/json")
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"fecha_fin_causa":null`) {
		t.Fatalf("fecha no acreditada inventada: %d %s", w.Code, w.Body.String())
	}
	consulta.err = dominiovec.ErrAutorizacionDenegada
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, ruta, nil)
	r.Header.Set("Accept", "application/json")
	h.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden || strings.Contains(w.Body.String(), "documento:parte-1") {
		t.Fatalf("denegación con datos: %d %s", w.Code, w.Body.String())
	}
}
