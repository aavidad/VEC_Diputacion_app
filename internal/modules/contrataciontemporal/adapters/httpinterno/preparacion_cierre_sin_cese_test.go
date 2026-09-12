package httpinterno

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type autoridadPreparacionCierrePrueba struct {
	org string
	err error
}

func (a autoridadPreparacionCierrePrueba) ResolverOrganizacionCierreAdministrativo(context.Context) (string, error) {
	return a.org, a.err
}

type lectorPreparacionCierrePrueba struct {
	s        ports.SolicitudPreparacionCierreAdministrativo
	p        ports.PreparacionCierreAdministrativo
	err      error
	llamadas int
}

func (l *lectorPreparacionCierrePrueba) ConsultarPreparacionCierreAdministrativo(_ context.Context, s ports.SolicitudPreparacionCierreAdministrativo) (ports.PreparacionCierreAdministrativo, error) {
	l.llamadas++
	l.s = s
	return l.p, l.err
}
func TestManejadorPreparacionCierreSinCeseConsultaEstrica(t *testing.T) {
	e, s := refA(), refB()
	l := &lectorPreparacionCierrePrueba{p: ports.PreparacionCierreAdministrativo{ExpedienteRef: e, SeguimientoRef: s, VersionActual: 1, EstadoActual: domain.ClaveCatalogo("vigente"), Acciones: []ports.AccionPreparacionCierreAdministrativo{{TransicionClave: domain.TransicionCerrarAdministrativamenteSinCese, Motivos: []ports.MotivoPreparacionCierreAdministrativo{{MotivoClave: "sin_cese"}}}}, PreparadaEn: time.Date(2026, 9, 10, 12, 0, 0, 123456000, time.UTC)}}
	h, err := NuevoManejadorPreparacionCierreSinCese(autoridadPreparacionCierrePrueba{org: refD()}, l)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodGet, RutaPreparacionCierreSinCese+"?expediente_ref="+e+"&seguimiento_ref="+s, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 || l.llamadas != 1 || l.s.OrganizacionRef != refD() {
		t.Fatalf("estado=%d llamadas=%d solicitud=%+v", w.Code, l.llamadas, l.s)
	}
	var out struct {
		Data map[string]any `json:"data"`
	}
	if json.Unmarshal(w.Body.Bytes(), &out) != nil || out.Data["organizacion_ref"] != nil {
		t.Fatalf("salida=%s", w.Body.String())
	}
	r = httptest.NewRequest(http.MethodGet, RutaPreparacionCierreSinCese+"?expediente_ref="+e+"&expediente_ref="+e+"&seguimiento_ref="+s, nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 400 || l.llamadas != 1 {
		t.Fatalf("duplicada=%d llamadas=%d", w.Code, l.llamadas)
	}
}
