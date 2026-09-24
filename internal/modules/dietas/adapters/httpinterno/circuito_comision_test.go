package httpinterno

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/dietas/domain"
	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
)

type identidadCircuitoPrueba struct {
	llamadas  int
	solicitud dietasports.SolicitudOperacionCircuito
	err       error
}

func (r *identidadCircuitoPrueba) ResolverIdentidadEfectivaCircuito(_ context.Context, s dietasports.SolicitudOperacionCircuito) (dietasports.IdentidadEfectivaCircuito, error) {
	r.llamadas++
	r.solicitud = s
	return dietasports.IdentidadEfectivaCircuito{UnidadCompetenciaRef: "unidad:acreditada"}, r.err
}

type usoCircuitoPrueba struct {
	decisiones, listas int
	decision           dietasports.SolicitudDecisionCircuito
	consulta           dietasports.ConsultaBandejaCircuito
	replay             bool
}

func (u *usoCircuitoPrueba) Decidir(_ context.Context, _ dietasports.IdentidadEfectivaCircuito, s dietasports.SolicitudDecisionCircuito) (dietasports.ResultadoCircuitoComision, error) {
	u.decisiones++
	u.decision = s
	return dietasports.ResultadoCircuitoComision{Comision: dietasports.VistaComisionCircuito{Referencia: s.Referencia, Estado: domain.EstadoPendienteAutorizacion, Version: 2}, Recibo: dietasports.ReciboBorradorComision{Referencia: "recibo:prueba", Version: 2, RegistradoEn: time.Date(2026, 9, 24, 12, 0, 0, 123456000, time.UTC), Repeticion: u.replay}}, nil
}
func (u *usoCircuitoPrueba) ListarPendientes(_ context.Context, _ dietasports.IdentidadEfectivaCircuito, q dietasports.ConsultaBandejaCircuito) (dietasports.PaginaBandejaCircuito, error) {
	u.listas++
	u.consulta = q
	return dietasports.PaginaBandejaCircuito{Items: []dietasports.VistaComisionCircuito{}}, nil
}

func TestCircuitoHTTPDerivaUnidadYRechazaSuplantacion(t *testing.T) {
	r, u := &identidadCircuitoPrueba{}, &usoCircuitoPrueba{}
	m, err := NuevoManejadorCircuito(r, u)
	if err != nil {
		t.Fatal(err)
	}
	ref := "dco_" + strings.Repeat("a", 22)
	cuerpo := `{"etapa":"revision","decision":"aprobar","motivo":"","clave_idempotencia":"clave_0123456789abcdef","version_esperada":1}`
	enviar := func(body string) *httptest.ResponseRecorder {
		p := httptest.NewRequest(http.MethodPost, RutaCircuito+"/"+ref+"/decisiones", strings.NewReader(body))
		p.Header.Set("Accept", "application/json")
		p.Header.Set("Content-Type", "application/json; charset=utf-8")
		w := httptest.NewRecorder()
		m.ServeHTTP(w, p)
		return w
	}
	w := enviar(strings.TrimSuffix(cuerpo, "}") + `,"unidad_ref":"unidad:inventada"}`)
	if w.Code != http.StatusBadRequest || r.llamadas != 0 {
		t.Fatalf("suplantación: %d, %d", w.Code, r.llamadas)
	}
	w = enviar(cuerpo)
	if w.Code != http.StatusCreated || u.decisiones != 1 || r.solicitud.Decision.UnidadRef != "" || u.decision.UnidadRef != "unidad:acreditada" {
		t.Fatalf("unidad frontera: %d, %#v, %#v", w.Code, r.solicitud.Decision, u.decision)
	}
	if !strings.Contains(w.Body.String(), `"registrado_en":"2026-09-24T12:00:00.123456Z"`) {
		t.Fatalf("recibo: %s", w.Body.String())
	}
	u.replay = true
	if w = enviar(cuerpo); w.Code != http.StatusOK {
		t.Fatalf("replay: %d", w.Code)
	}
}

func TestBandejaHTTPFiltraFechasSinUnidadLibre(t *testing.T) {
	r, u := &identidadCircuitoPrueba{}, &usoCircuitoPrueba{}
	m, err := NuevoManejadorCircuito(r, u)
	if err != nil {
		t.Fatal(err)
	}
	get := func(query string) int {
		p := httptest.NewRequest(http.MethodGet, RutaCircuito+query, nil)
		p.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		m.ServeHTTP(w, p)
		return w.Code
	}
	if got := get("?etapa=revision&fecha_desde=2026-09-01&fecha_hasta=2026-09-30&limit=20"); got != http.StatusOK || u.listas != 1 || u.consulta.UnidadRef != "unidad:acreditada" {
		t.Fatalf("bandeja: %d %#v", got, u.consulta)
	}
	if got := get("?etapa=revision&unidad_ref=unidad:inventada"); got != http.StatusBadRequest || u.listas != 1 {
		t.Fatalf("unidad libre: %d", got)
	}
	if got := get("?etapa=revision&fecha_desde=2026-10-01&fecha_hasta=2026-09-30"); got != http.StatusBadRequest || u.listas != 1 {
		t.Fatalf("fechas invertidas: %d", got)
	}
}
