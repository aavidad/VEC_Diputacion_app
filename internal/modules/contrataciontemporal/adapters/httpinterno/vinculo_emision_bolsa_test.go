package httpinterno

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type canalVinculoPrueba struct{}

func (canalVinculoPrueba) ResolverContextoCanalSeguimiento(context.Context) (application.ContextoCanalSeguimiento, error) {
	return application.ContextoCanalSeguimiento{AutenticacionRef: "aut_" + strings.Repeat("a", 32), SesionRef: "ses_" + strings.Repeat("b", 32),
		PerfilRef: "prf_0123456789abcdefghijkl", OrganizacionRef: "organizacion:prueba"}, nil
}

type ejecutorVinculoPrueba struct {
	recibida ports.SolicitudVinculoEmisionBolsa
}

func (e *ejecutorVinculoPrueba) Vincular(_ context.Context, s ports.SolicitudVinculoEmisionBolsa) (ports.ReciboVinculoEmisionBolsa, error) {
	e.recibida = s
	return ports.ReciboVinculoEmisionBolsa{ExpedienteRef: s.ExpedienteRef, BolsaRef: s.BolsaRef,
		LlamamientoRef: s.LlamamientoRef, ReciboEmisionRef: s.ReciboEmisionRef,
		ReciboVinculoRef: "recibo:ct:bolsa:prueba", AuditoriaRef: "auditoria:ct:bolsa:prueba",
		EventoRef: "evento:ct:bolsa:prueba", VinculadoEn: time.Now().UTC().Truncate(time.Microsecond)}, nil
}

func TestVinculoEmisionBolsaNoAceptaOrganizacionDelCliente(t *testing.T) {
	e := &ejecutorVinculoPrueba{}
	h, err := NuevoManejadorVinculoEmisionBolsa(canalVinculoPrueba{}, e)
	if err != nil {
		t.Fatal(err)
	}
	base := `{"expediente_ref":"expediente:ct:prueba","version_esperada":4,"bolsa_ref":"bolsa:prueba",` +
		`"llamamiento_ref":"llamamiento:` + strings.Repeat("a", 64) + `","recibo_emision_ref":"recibo:llamamiento:` +
		strings.Repeat("a", 64) + `","clave_idempotencia":"11111111-1111-4111-8111-111111111111"}`
	peticion := func(cuerpo string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodPost, RutaVinculosEmisionBolsa, strings.NewReader(cuerpo))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	if w := peticion(strings.Replace(base, `"expediente_ref"`, `"organizacion_ref":"organizacion:ajena","expediente_ref"`, 1)); w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("organización aportada: %d", w.Code)
	}
	if e.recibida.ExpedienteRef != "" {
		t.Fatal("se ejecutó con organización aportada")
	}
	for _, invalida := range []string{
		strings.Replace(base, "llamamiento:"+strings.Repeat("a", 64), "llamamiento:foo", 1),
		strings.Replace(base, "recibo:llamamiento:"+strings.Repeat("a", 64), "recibo:llamamiento:"+strings.Repeat("b", 64), 1),
	} {
		if w := peticion(invalida); w.Code != http.StatusUnprocessableEntity {
			t.Fatalf("vínculo incoherente: %d", w.Code)
		}
		if e.recibida.ExpedienteRef != "" {
			t.Fatal("se ejecutó antes de rechazar el vínculo")
		}
	}
	if w := peticion(base); w.Code != http.StatusCreated {
		t.Fatalf("vínculo nominal: %d %s", w.Code, w.Body.String())
	}
	if e.recibida.OrganizacionRef != "organizacion:prueba" || e.recibida.VersionEsperada != 4 {
		t.Fatalf("canal o versión divergente: %+v", e.recibida)
	}
}
