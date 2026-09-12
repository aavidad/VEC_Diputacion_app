package httpinterno

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type autoridadSubsanacionReparosPrueba struct {
	contexto ContextoCanalSubsanacionReparos
}

func (a autoridadSubsanacionReparosPrueba) ResolverContextoCanalSubsanacionReparos(context.Context) (ContextoCanalSubsanacionReparos, error) {
	return a.contexto, nil
}

type ejecutorSubsanacionReparosPrueba struct {
	solicitud application.SolicitudRegistrarSubsanacionReparo
}

func (e *ejecutorSubsanacionReparosPrueba) RegistrarSubsanacionReparo(_ context.Context, s application.SolicitudRegistrarSubsanacionReparo) (ports.ReciboSubsanacionReparo, error) {
	e.solicitud = s
	return ports.ReciboSubsanacionReparo{Operacion: ports.OperacionRegistrarSubsanacionReparo, OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef, VersionAnterior: s.VersionEsperada, VersionResultante: s.VersionEsperada + 1, FaseResultante: domain.FaseSubsanacionUnidad, EstadoResultante: domain.EstadoIncidencia, ReciboRef: "recibo:subsanacion:http:001", AuditoriaRef: "auditoria:subsanacion:http:001", EventoRef: "evento:subsanacion:http:001", ActorRef: "actor:unidad:http:001", RegistradaEn: time.Date(2026, 9, 12, 10, 0, 0, 0, time.UTC)}, nil
}

func TestManejadorSubsanacionReparosRegistraYPublicaRecibo(t *testing.T) {
	e := &ejecutorSubsanacionReparosPrueba{}
	h, err := NuevoManejadorSubsanacionReparos(autoridadSubsanacionReparosPrueba{ContextoCanalSubsanacionReparos{"aut_aaaaaaaaaaaaaaaaaaaaaaaa", "ses_bbbbbbbbbbbbbbbbbbbbbbbb", "prf_cccccccccccccccccccccccc", "organizacion:subsanacion:http:001"}}, e)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodPost, RutaSubsanacionReparos, strings.NewReader(`{"expediente_ref":"expediente:subsanacion:http:001","version_esperada":6,"clave_idempotencia":"11111111-2222-4333-8444-555555555555","observaciones":"Corrección sintética."}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusCreated || e.solicitud.Observaciones != "Corrección sintética." || e.solicitud.VersionEsperada != 6 {
		t.Fatalf("respuesta=%d solicitud=%#v", w.Code, e.solicitud)
	}
}
