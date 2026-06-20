package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"vec-diputacion-granada/internal/candidate/domain"
	"vec-diputacion-granada/internal/candidate/ports"
	"vec-diputacion-granada/internal/shared/i18n"
)

func TestHTTPHandlerRoutesCandidateFlowToInjectedService(t *testing.T) {
	service := &recordingService{
		createView: CandidateView{ID: "cand-1", CallID: "call-1"},
		addView:    MeritView{ID: "merit-1"},
		baremoView: BaremoView{TotalPoints: 2.4},
		expediente: ExpedienteView{Candidate: CandidateView{ID: "cand-1"}, Merits: []MeritView{{ID: "merit-1"}}, Baremo: BaremoView{TotalPoints: 2.4}},
	}
	handler := mustTestHandler(t, service, &recordingAuthenticator{principal: ports.AuthPrincipal{Subject: "cand-1", Role: ports.AuthRoleCiudadano, Mechanism: ports.AuthMechanismClave}})
	create := performJSON(t, handler, http.MethodPost, "/candidates", `{"id":"cand-1","dni":"12345678A","nombre":"Ana Perez","email":"ana@example.test"}`)
	assertStatus(t, create, http.StatusCreated)
	add := performJSON(t, handler, http.MethodPost, "/candidates/cand-1/merits", `{"id":"merit-1","tipo":"experiencia_misma_categoria","datos":{"meses":12},"estado":"Presentado"}`)
	assertStatus(t, add, http.StatusCreated)
	baremo := performJSON(t, handler, http.MethodPost, "/candidates/cand-1/baremo", "")
	assertStatus(t, baremo, http.StatusOK)
	var score BaremoView
	decodeData(t, baremo, &score)
	if score.TotalPoints != 2.4 {
		t.Fatalf("baremo total = %v, want 2.4", score.TotalPoints)
	}
	exported := performJSON(t, handler, http.MethodGet, "/candidates/cand-1/expediente", "")
	assertStatus(t, exported, http.StatusOK)
	var expediente ExpedienteView
	decodeData(t, exported, &expediente)
	if expediente.Candidate.ID != "cand-1" || len(expediente.Merits) != 1 || expediente.Baremo.TotalPoints != 2.4 {
		t.Fatalf("expediente = %+v", expediente)
	}
	if service.addCandidateID != "cand-1" || service.baremoCandidateID != "cand-1" || service.exportCandidateID != "cand-1" {
		t.Fatalf("candidate ids routed incorrectly: service=%+v", service)
	}
}

func TestHTTPHandlerRunsAdministrativeAPIDemoThroughInjectedRunner(t *testing.T) {
	service := &recordingService{}
	runner := &recordingDemoRunner{view: ProcedureDemoView{
		Convocatoria: ConvocatoriaView{ID: "demo-convocatoria"},
		Definitivo: ListadoView{Items: []ListadoItemView{
			{Rank: 1, CandidateID: "demo-cand-a"},
			{Rank: 2, CandidateID: "demo-cand-b"},
		}},
	}}
	handler := mustTestHandlerWithDemo(t, service, runner, &recordingAuthenticator{principal: ports.AuthPrincipal{Subject: "staff", Role: ports.AuthRolePersonalInterno}})
	demo := performStaffJSON(t, handler, http.MethodPost, "/api/demo", "")
	assertStatus(t, demo, http.StatusOK)
	var result ProcedureDemoView
	decodeData(t, demo, &result)
	if result.Convocatoria.ID != "demo-convocatoria" || len(result.Definitivo.Items) != 2 {
		t.Fatalf("demo result = %+v", result)
	}
	if result.Definitivo.Items[0].Rank != 1 || result.Definitivo.Items[0].CandidateID != "demo-cand-a" || runner.calls != 1 {
		t.Fatalf("demo runner = calls %d result %+v", runner.calls, result.Definitivo.Items)
	}
}

func TestHTTPHandlerRunsProcedureDemoUseCaseAdapter(t *testing.T) {
	procedure, err := newTestProcedureUseCase()
	if err != nil {
		t.Fatalf("procedure usecase: %v", err)
	}
	handler := mustTestHandlerWithDemo(t, &recordingService{}, NewProcedureDemoRunner(procedure), &recordingAuthenticator{principal: ports.AuthPrincipal{Subject: "staff", Role: ports.AuthRolePersonalInterno}})
	demo := performStaffJSON(t, handler, http.MethodPost, "/api/demo", "")
	assertStatus(t, demo, http.StatusOK)
	var result ProcedureDemoView
	decodeData(t, demo, &result)
	if result.Convocatoria.ID != "demo-convocatoria" || len(result.Definitivo.Items) != 2 {
		t.Fatalf("demo result = %+v", result)
	}
	if result.Definitivo.Items[0].Rank != 1 || result.Definitivo.Items[0].CandidateID != "demo-cand-a" {
		t.Fatalf("demo ranking = %+v", result.Definitivo.Items)
	}
}

func TestHTTPHandlerUsesInjectedPortsAuthAndI18n(t *testing.T) {
	service := &recordingService{createView: CandidateView{ID: "cand-1", CallID: "call-1"}, expediente: ExpedienteView{Candidate: CandidateView{ID: "cand-1"}}}
	auth := &recordingAuthenticator{principal: ports.AuthPrincipal{Subject: "cand-1", Role: ports.AuthRoleCiudadano, Mechanism: ports.AuthMechanismClave}}
	handler := mustTestHandler(t, service, auth)
	create := performJSON(t, handler, http.MethodPost, "/candidates", `{"id":"cand-1","dni":"12345678A","nombre":"Ana","email":"ana@example.test","call_id":"call-1"}`)
	assertStatus(t, create, http.StatusCreated)
	assertMessage(t, create, "creado")
	if service.createCommand.CallID != "call-1" || auth.lastCredentials.Token != "citizen-token" {
		t.Fatalf("ports not called: service=%+v auth=%+v", service.createCommand, auth.lastCredentials)
	}
	exported := performJSON(t, handler, http.MethodGet, "/candidates/cand-1/expediente", "")
	assertStatus(t, exported, http.StatusOK)
	assertMessage(t, exported, "expediente")
	if service.exportCandidateID != "cand-1" {
		t.Fatalf("export candidate = %q", service.exportCandidateID)
	}
}

func TestHTTPHandlerRejectsUnauthenticatedForbiddenAndBadJSON(t *testing.T) {
	service := &recordingService{}
	unauth := mustTestHandler(t, service, &recordingAuthenticator{err: ports.ErrAuthenticationFailed})
	assertStatus(t, performJSON(t, unauth, http.MethodPost, "/candidates", `{}`), http.StatusUnauthorized)
	forbidden := mustTestHandler(t, service, &recordingAuthenticator{principal: ports.AuthPrincipal{Subject: "auditor", Role: ports.AuthRole("auditor")}})
	assertStatus(t, performJSON(t, forbidden, http.MethodPost, "/candidates", `{}`), http.StatusForbidden)
	authed := mustTestHandler(t, service, &recordingAuthenticator{principal: ports.AuthPrincipal{Subject: "cand-1", Role: ports.AuthRoleCiudadano}})
	bad := performJSON(t, authed, http.MethodPost, "/candidates", `{"id":"cand-1","unknown":true}`)
	assertStatus(t, bad, http.StatusBadRequest)
	assertMessage(t, bad, "mal json")
	if service.createCalls != 0 || bad.Error == "" {
		t.Fatalf("bad json leaked: calls=%d error=%q", service.createCalls, bad.Error)
	}
}

func TestHTTPHandlerRejectsStaffOnCitizenCandidateRoutes(t *testing.T) {
	service := &recordingService{}
	handler := mustTestHandler(t, service, &recordingAuthenticator{
		principal: ports.AuthPrincipal{
			Subject:   "staff",
			Role:      ports.AuthRolePersonalInterno,
			Mechanism: ports.AuthMechanismKerberosAD,
		},
	})
	response := performStaffJSON(t, handler, http.MethodPost, "/candidates", `{"id":"cand-1"}`)
	assertStatus(t, response, http.StatusForbidden)
	assertMessage(t, response, "sin permisos")
	if service.createCalls != 0 {
		t.Fatalf("staff reached citizen service: calls=%d", service.createCalls)
	}
}

func TestHTTPHandlerReturnsJSONEnvelopeForMethodMismatch(t *testing.T) {
	service := &recordingService{}
	handler := mustTestHandler(t, service, &recordingAuthenticator{principal: ports.AuthPrincipal{Subject: "cand-1", Role: ports.AuthRoleCiudadano}})
	response := performJSON(t, handler, http.MethodGet, "/candidates", "")
	assertStatus(t, response, http.StatusMethodNotAllowed)
	assertMessage(t, response, "metodo no permitido")
	if response.Error != "" || service.createCalls != 0 {
		t.Fatalf("method mismatch leaked: calls=%d error=%q", service.createCalls, response.Error)
	}
}

func TestHTTPHandlerRejectsCandidateOwnerMismatch(t *testing.T) {
	service := &recordingService{}
	handler := mustTestHandler(t, service, &recordingAuthenticator{
		principal: ports.AuthPrincipal{Subject: "cand-1", Role: ports.AuthRoleCiudadano, Mechanism: ports.AuthMechanismClave},
	})

	assertStatus(t, performJSON(t, handler, http.MethodPost, "/candidates", `{"id":"cand-2","dni":"12345678A","nombre":"Ana","email":"ana@example.test"}`), http.StatusForbidden)
	assertStatus(t, performJSON(t, handler, http.MethodPost, "/candidates/cand-2/merits", `{"id":"merit-1"}`), http.StatusForbidden)
	assertStatus(t, performJSON(t, handler, http.MethodPost, "/candidates/cand-2/baremo", ""), http.StatusForbidden)
	assertStatus(t, performJSON(t, handler, http.MethodGet, "/candidates/cand-2/expediente", ""), http.StatusForbidden)
	if service.createCalls != 0 || service.addCandidateID != "" || service.baremoCandidateID != "" || service.exportCandidateID != "" {
		t.Fatalf("owner mismatch reached service: %+v", service)
	}
}

func TestHTTPHandlerFallsBackMethodMismatchMessage(t *testing.T) {
	handler, err := NewHTTPHandler(
		&recordingService{},
		&recordingAuthenticator{principal: ports.AuthPrincipal{Subject: "candidate", Role: ports.AuthRoleCiudadano}},
		nil,
	)
	if err != nil {
		t.Fatalf("NewHTTPHandler() error = %v", err)
	}
	response := performJSON(t, handler, http.MethodGet, "/candidates", "")
	assertStatus(t, response, http.StatusMethodNotAllowed)
	assertMessage(t, response, "Metodo no permitido")
}

type recordingService struct {
	createCalls       int
	createCommand     CreateCandidateCommand
	createView        CandidateView
	addCandidateID    string
	addCommand        AddMeritCommand
	addView           MeritView
	baremoCandidateID string
	baremoView        BaremoView
	exportCandidateID string
	expediente        ExpedienteView
}

func (s *recordingService) CreateCandidate(ctx context.Context, command CreateCandidateCommand) (CandidateView, error) {
	s.createCalls++
	s.createCommand = command
	return s.createView, ctx.Err()
}
func (s *recordingService) AddMerit(ctx context.Context, candidateID string, command AddMeritCommand) (MeritView, error) {
	s.addCandidateID, s.addCommand = candidateID, command
	return s.addView, ctx.Err()
}
func (s *recordingService) CalculateBaremo(ctx context.Context, candidateID string) (BaremoView, error) {
	s.baremoCandidateID = candidateID
	return s.baremoView, ctx.Err()
}
func (s *recordingService) ExportExpediente(ctx context.Context, candidateID string) (ExpedienteView, error) {
	s.exportCandidateID = candidateID
	return s.expediente, ctx.Err()
}

type recordingAuthenticator struct {
	principal       ports.AuthPrincipal
	err             error
	lastCredentials ports.AuthCredentials
}

func (a *recordingAuthenticator) Authenticate(ctx context.Context, credentials ports.AuthCredentials) (ports.AuthPrincipal, error) {
	a.lastCredentials = credentials
	if a.err != nil {
		return ports.AuthPrincipal{}, a.err
	}
	return a.principal, ctx.Err()
}

type testEnvelope struct {
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
	Error   string          `json:"error"`
	status  int
}

func mustTestHandler(t *testing.T, service Service, authenticator ports.Authenticator) http.Handler {
	t.Helper()
	catalog, err := i18n.New(i18n.DefaultLocale, map[string]map[string]string{i18n.DefaultLocale: {
		"api.candidate.created": "creado", "api.candidate.merit_added": "merito", "api.candidate.baremo_calculated": "baremo",
		"api.candidate.expediente_exported": "expediente", "api.error.bad_request": "mal json", "api.error.unauthorized": "auth requerida",
		"api.error.forbidden": "sin permisos", "api.error.not_found": "no encontrado", "api.error.method_not_allowed": "metodo no permitido", "api.error.internal": "interno",
	}})
	if err != nil {
		t.Fatalf("i18n catalog: %v", err)
	}
	handler, err := NewHTTPHandler(service, authenticator, catalog)
	if err != nil {
		t.Fatalf("NewHTTPHandler: %v", err)
	}
	return handler
}

func mustTestHandlerWithDemo(t *testing.T, service Service, runner ProcedureDemoRunner, authenticator ports.Authenticator) http.Handler {
	t.Helper()
	handler, err := NewHTTPHandlerWithDemoRunner(service, runner, authenticator, fallbackCatalog())
	if err != nil {
		t.Fatalf("NewHTTPHandlerWithDemoRunner: %v", err)
	}
	return handler
}

func performJSON(t *testing.T, handler http.Handler, method, path, body string) testEnvelope {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Mechanism", string(ports.AuthMechanismClave))
	req.Header.Set("X-Auth-Subject", "cand-1")
	req.Header.Set("Authorization", "Bearer citizen-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	var envelope testEnvelope
	envelope.status = rec.Code
	if err := json.NewDecoder(rec.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode response: %v body=%q", err, rec.Body.String())
	}
	return envelope
}

func performStaffJSON(t *testing.T, handler http.Handler, method, path, body string) testEnvelope {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Mechanism", string(ports.AuthMechanismKerberosAD))
	req.Header.Set("X-Auth-Subject", "staff")
	req.Header.Set("Authorization", "Bearer staff-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	var envelope testEnvelope
	envelope.status = rec.Code
	if err := json.NewDecoder(rec.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode response: %v body=%q", err, rec.Body.String())
	}
	return envelope
}
func assertStatus(t *testing.T, envelope testEnvelope, want int) {
	t.Helper()
	if envelope.status != want {
		t.Fatalf("status = %d, want %d: %+v", envelope.status, want, envelope)
	}
}
func assertMessage(t *testing.T, envelope testEnvelope, want string) {
	t.Helper()
	if envelope.Message != want {
		t.Fatalf("message = %q, want %q", envelope.Message, want)
	}
}
func decodeData(t *testing.T, envelope testEnvelope, target any) {
	t.Helper()
	if err := json.Unmarshal(envelope.Data, target); err != nil {
		t.Fatalf("decode data: %v raw=%s", err, envelope.Data)
	}
}

var _ = domain.MeritTypeOtros
