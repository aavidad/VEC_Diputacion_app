package httpinterno

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	aplicacionbolsa "vec-diputacion-granada/internal/modules/bolsa/application"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

type preparadorParticipacionesHTTPPrueba struct{ llamadas int }

func (p *preparadorParticipacionesHTTPPrueba) PrepararOrdenConsultaParticipacionesPropias(context.Context) (aplicacionbolsa.OrdenConsultaParticipacionesPropias, error) {
	p.llamadas++
	return aplicacionbolsa.OrdenConsultaParticipacionesPropias{}, nil
}

type consultorParticipacionesHTTPPrueba struct{ llamadas int }

func (c *consultorParticipacionesHTTPPrueba) Consultar(context.Context, aplicacionbolsa.OrdenConsultaParticipacionesPropias) (puertosbolsa.ResultadoParticipacionesPropias, error) {
	c.llamadas++
	return puertosbolsa.ResultadoParticipacionesPropias{Esquema: puertosbolsa.EsquemaParticipacionesPropiasV1, ConsultadaEn: time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC), Participaciones: []puertosbolsa.ParticipacionPropia{{BolsaRef: "bol_" + strings.Repeat("b", 22), CategoriaRef: "cat_" + strings.Repeat("c", 22), VersionBolsa: 1, Orden: 2, TotalParticipaciones: 8, EstadoBolsa: "activa", VigenteDesde: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}}}, nil
}
func TestHandlerParticipacionesPropiasRechazaEntradaClienteYExponeContrato(t *testing.T) {
	p := &preparadorParticipacionesHTTPPrueba{}
	c := &consultorParticipacionesHTTPPrueba{}
	h, err := NuevoHandlerParticipacionesPropias(p, c)
	if err != nil {
		t.Fatal(err)
	}
	mala := httptest.NewRequest(http.MethodGet, RutaParticipacionesPropias+"?candidato_ref=can_falso", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, mala)
	if w.Code != http.StatusBadRequest || p.llamadas != 0 {
		t.Fatalf("entrada cliente admitida: %d", w.Code)
	}
	identidad := httptest.NewRequest(http.MethodGet, RutaParticipacionesPropias, nil)
	identidad.Header.Set("Authorization", "Bearer cliente-no-confiable")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, identidad)
	if w.Code != http.StatusBadRequest || p.llamadas != 0 {
		t.Fatalf("identidad cliente admitida: %d", w.Code)
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaParticipacionesPropias, nil))
	if w.Code != http.StatusOK || p.llamadas != 1 || c.llamadas != 1 || !strings.Contains(w.Body.String(), `"esquema":"vec.bolsa.mi-bolsa.v1"`) {
		t.Fatalf("respuesta incorrecta: %d %s", w.Code, w.Body.String())
	}
}
