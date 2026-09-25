package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
)

type operadorCatalogosEmpleadoB2Prueba struct {
	consulta  personaldomain.SolicitudConsultaCatalogoEmpleadoB2
	cambio    personaldomain.SolicitudCambioCatalogoEmpleadoB2
	resultado personalports.ResultadoConsultaCatalogoEmpleadoB2
	replay    bool
	llamadas  int
}

func (o *operadorCatalogosEmpleadoB2Prueba) Consultar(_ context.Context, s personaldomain.SolicitudConsultaCatalogoEmpleadoB2) (personalports.ResultadoConsultaCatalogoEmpleadoB2, error) {
	o.llamadas++
	o.consulta = s
	return o.resultado, nil
}

func (o *operadorCatalogosEmpleadoB2Prueba) Cambiar(_ context.Context, s personaldomain.SolicitudCambioCatalogoEmpleadoB2) (personalports.ResultadoCambioCatalogoEmpleadoB2, error) {
	o.llamadas++
	o.cambio = s
	instante := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	estado := "registrado"
	if o.replay {
		estado = "replay"
	}
	return personalports.ResultadoCambioCatalogoEmpleadoB2{
		Entrada: personaldomain.EntradaCatalogoRegistroEmpleadoB2{
			OrganismoRef: s.OrganismoRef, Tipo: s.Tipo, Ref: s.Ref, Version: s.Version,
			Revision: s.Revision, Denominacion: s.Denominacion, HuellaSHA256: s.HuellaSHA256,
			VigenteDesde: s.VigenteDesde, VigenteHasta: s.VigenteHasta, Estado: "publicada",
		},
		Recibo:       personalports.ReciboCatalogoEmpleadoB2{DecisionRef: "decision_1", AuditoriaRef: "audit_1", ConsumoHuellaSHA256: strings.Repeat("a", 64), RegistradoEn: instante},
		AccesoActual: personalports.AccesoActualCatalogoEmpleadoB2{DecisionRef: "decision_1", AuditoriaRef: "audit_1", ConsumoHuellaSHA256: strings.Repeat("a", 64), RegistradoEn: instante, EstadoReplay: estado},
	}, nil
}

func TestCatalogosEmpleadoB2HTTPConsultaFiltraPorOrganismoServidor(t *testing.T) {
	a := &autoridadRegistroEmpleadoB2Prueba{actor: actorOrganizacionHistoricaPrueba(t), organismo: "org_prueba"}
	o := &operadorCatalogosEmpleadoB2Prueba{resultado: personalports.ResultadoConsultaCatalogoEmpleadoB2{
		OrganismoRef: "org_prueba",
		Entradas:     []personaldomain.EntradaCatalogoRegistroEmpleadoB2{},
		Evidencia:    personalports.EvidenciaCatalogoEmpleadoB2{DecisionRef: "decision_1", AuditoriaRef: "audit_1", ConsumoHuellaSHA256: strings.Repeat("a", 64), EfectoRef: "org_prueba:regimen", ConsultadaEn: time.Now().UTC()},
	}}
	h, err := NewHandlerCatalogosRegistroEmpleadoB2(a, o, &auditorRegistroEmpleadoB2Prueba{})
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaCatalogosRegistroEmpleadoB2+"?tipo=regimen&estado=publicada&limite=25", nil))
	if w.Code != 200 || o.llamadas != 1 || o.consulta.OrganismoRef != "org_prueba" || o.consulta.Estado != "publicada" || o.consulta.Limite != 25 || o.consulta.Actor.Principal.ID != a.actor.Principal.ID {
		t.Fatalf("consulta=%d orden=%+v", w.Code, o.consulta)
	}
	o.resultado.Entradas = []personaldomain.EntradaCatalogoRegistroEmpleadoB2{{OrganismoRef: "org_ajeno"}}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaCatalogosRegistroEmpleadoB2+"?tipo=regimen", nil))
	if w.Code != 503 || strings.Contains(w.Body.String(), "org_ajeno") {
		t.Fatalf("respuesta ajena=%d %s", w.Code, w.Body.String())
	}
}

func TestCatalogosEmpleadoB2HTTPPublicaYRecuperaSinIdentidadCliente(t *testing.T) {
	a := &autoridadRegistroEmpleadoB2Prueba{actor: actorOrganizacionHistoricaPrueba(t), organismo: "org_prueba"}
	o := &operadorCatalogosEmpleadoB2Prueba{}
	h, _ := NewHandlerCatalogosRegistroEmpleadoB2(a, o, &auditorRegistroEmpleadoB2Prueba{})
	s := personaldomain.SolicitudCambioCatalogoEmpleadoB2{OrganismoRef: "org_prueba", Tipo: "regimen", Ref: "reg_prueba", Version: 1, Revision: 1, Denominacion: "Régimen de prueba", VigenteDesde: "2026-09-25"}
	hash := personaldomain.HuellaPublicacionCatalogoEmpleadoB2(s)
	cuerpo := fmt.Sprintf(`{"operacion":"publicar","tipo":"regimen","ref":"reg_prueba","version":1,"revision":1,"denominacion":"Régimen de prueba","huella_sha256":"%s","vigente_desde":"2026-09-25","vigente_hasta":""}`, hash)
	r := httptest.NewRequest(http.MethodPost, RutaCatalogosRegistroEmpleadoB2, strings.NewReader(cuerpo))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Idempotency-Key", claveRegistroEmpleadoB2Prueba)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 201 || o.llamadas != 1 || o.cambio.OrganismoRef != "org_prueba" || o.cambio.Actor.Principal.ID != a.actor.Principal.ID || w.Header().Get("Set-Cookie") != "" {
		t.Fatalf("publicación=%d orden=%+v", w.Code, o.cambio)
	}
	o.replay = true
	r = httptest.NewRequest(http.MethodPost, RutaCatalogosRegistroEmpleadoB2, strings.NewReader(cuerpo))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Idempotency-Key", claveRegistroEmpleadoB2Prueba)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 || o.llamadas != 2 || !strings.Contains(w.Body.String(), `"estado_replay":"replay"`) {
		t.Fatalf("replay=%d cuerpo=%s", w.Code, w.Body.String())
	}
	mal := strings.TrimSuffix(cuerpo, "}") + `,"organismo_ref":"org_ajeno"}`
	r = httptest.NewRequest(http.MethodPost, RutaCatalogosRegistroEmpleadoB2, strings.NewReader(mal))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Idempotency-Key", claveRegistroEmpleadoB2Prueba)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 400 || o.llamadas != 2 {
		t.Fatalf("cliente suministra organismo=%d llamadas=%d", w.Code, o.llamadas)
	}
	mal = strings.TrimSuffix(cuerpo, "}") + `,"acto_ref":"acto_espurio"}`
	r = httptest.NewRequest(http.MethodPost, RutaCatalogosRegistroEmpleadoB2, strings.NewReader(mal))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Idempotency-Key", claveRegistroEmpleadoB2Prueba)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 400 || o.llamadas != 2 || a.llamadas != 2 {
		t.Fatalf("cliente suministra acto=%d autoridad=%d llamadas=%d", w.Code, a.llamadas, o.llamadas)
	}
}

func TestCatalogosEmpleadoB2HTTPDenegacionAuditada(t *testing.T) {
	a := &autoridadRegistroEmpleadoB2Prueba{err: ErrAccesoRutaExactaDenegado}
	o := &operadorCatalogosEmpleadoB2Prueba{}
	auditor := &auditorRegistroEmpleadoB2Prueba{}
	h, _ := NewHandlerCatalogosRegistroEmpleadoB2(a, o, auditor)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaCatalogosRegistroEmpleadoB2+"?tipo=regimen", nil))
	if w.Code != 403 || o.llamadas != 0 || len(auditor.ordenes) != 1 {
		t.Fatalf("denegación=%d auditoría=%+v", w.Code, auditor.ordenes)
	}
}
