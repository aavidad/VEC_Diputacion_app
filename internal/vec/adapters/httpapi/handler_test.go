package httpapi

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	adminmodule "vec-diputacion-granada/internal/modules/administracion"
	bolsamodule "vec-diputacion-granada/internal/modules/bolsa"
	cronosmodule "vec-diputacion-granada/internal/modules/cronos"
	dietasmodule "vec-diputacion-granada/internal/modules/dietas"
	personalmodule "vec-diputacion-granada/internal/modules/personal"
	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/vec/adapters/memory"
	"vec-diputacion-granada/internal/vec/application"
)

func newTestHandler(t *testing.T) *Handler {
	t.Helper()
	return newTestHandlerWithOptions(t, HandlerOptions{})
}

func testOSRMOptions(baseURL string) HandlerOptions {
	return HandlerOptions{
		OSRMBaseURL:      baseURL,
		OSRMScopeName:    "Granada provincia + 15 km",
		OSRMScopeBounds:  "36.45,-4.6,38.25,-2.15",
		OSRMAllowedCIDRs: []string{"127.0.0.1/32"},
	}
}

func newTestHandlerWithOptions(t *testing.T, options HandlerOptions) *Handler {
	t.Helper()
	options.AllowDemoIdentity = true
	options.DemoIdentityResolver = resolvedorIdentidadPruebas{}
	store := memory.NewStore()
	service, internal, err := application.NewServiceWithInternalOperations(store, store, store)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	options.InternalOperations = internal
	handler, err := NewHandlerWithOptions(service, options)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	cargarFixturesCatalogoPersonalPrueba(t, handler.personalCatalog)
	return handler
}

func TestAdministradorTecnicoSoloAccedeACarcasaYAdministracion(t *testing.T) {
	store := memory.NewStore()
	service, internal, err := application.NewServiceWithInternalOperations(store, store, store)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	for _, manifest := range []string{
		personalmodule.Manifest().ID,
		cronosmodule.Manifest().ID,
		dietasmodule.Manifest().ID,
		bolsamodule.Manifest().ID,
		adminmodule.Manifest().ID,
	} {
		if manifest == "" {
			t.Fatalf("manifest id is empty")
		}
	}
	for _, module := range []struct {
		name string
		fn   func() error
	}{
		{name: personalmodule.ModuleID, fn: func() error { return internal.RegisterModule(context.Background(), personalmodule.Manifest()) }},
		{name: cronosmodule.ModuleID, fn: func() error { return internal.RegisterModule(context.Background(), cronosmodule.Manifest()) }},
		{name: dietasmodule.ModuleID, fn: func() error { return internal.RegisterModule(context.Background(), dietasmodule.Manifest()) }},
		{name: bolsamodule.ModuleID, fn: func() error { return internal.RegisterModule(context.Background(), bolsamodule.Manifest()) }},
		{name: adminmodule.ModuleID, fn: func() error { return internal.RegisterModule(context.Background(), adminmodule.Manifest()) }},
	} {
		if err := module.fn(); err != nil {
			t.Fatalf("RegisterModule(%s) error = %v", module.name, err)
		}
	}
	handler, err := NewHandlerWithOptions(service, HandlerOptions{
		InternalOperations:   internal,
		AllowDemoIdentity:    true,
		DemoIdentityResolver: resolvedorIdentidadPruebas{},
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	for _, path := range []string{"/api/vec/modules", "/api/vec/menu"} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status = %d: %s", path, rec.Code, rec.Body.String())
		}
		if path == "/api/vec/menu" {
			body := rec.Body.String()
			if !strings.Contains(body, "admin.catalogos") {
				t.Fatalf("el menu tecnico no contiene su capacidad administrativa: %s", body)
			}
			for _, prohibido := range []string{"personal.", "cronos.", "dietas.", "bolsa."} {
				if strings.Contains(body, prohibido) {
					t.Fatalf("el menu tecnico contiene capacidad funcional %q: %s", prohibido, body)
				}
			}
		}
	}
	workspace := httptest.NewRecorder()
	handler.ServeHTTP(workspace, httptest.NewRequest(http.MethodGet, "/api/vec/workspace", nil))
	if workspace.Code != http.StatusForbidden {
		t.Fatalf("el administrador tecnico no debe alcanzar el workspace: status = %d: %s", workspace.Code, workspace.Body.String())
	}
	for _, caso := range []struct {
		path   string
		status int
	}{
		{path: "/api/vec/modules/cronos/action", status: http.StatusForbidden},
		{path: "/api/vec/modules/horarios/action", status: http.StatusForbidden},
		{path: "/api/vec/modules/permisos/action", status: http.StatusForbidden},
		{path: "/api/vec/modules/dietas/action", status: http.StatusForbidden},
		{path: "/api/vec/modules/rutas/action", status: http.StatusForbidden},
		{path: "/api/vec/modules/administracion/action", status: http.StatusAccepted},
		{path: "/api/vec/modules/personal/action", status: http.StatusForbidden},
		{path: "/api/vec/modules/nominas/action", status: http.StatusForbidden},
		{path: "/api/vec/modules/bolsa/action", status: http.StatusForbidden},
	} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, caso.path, nil))
		if rec.Code != caso.status {
			t.Fatalf("%s status = %d, want %d: %s", caso.path, rec.Code, caso.status, rec.Body.String())
		}
		if caso.status != http.StatusAccepted {
			continue
		}
		var body map[string]any
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatalf("decode action response: %v", err)
		}
		if body["data"] == nil {
			t.Fatalf("action response missing data: %#v", body)
		}
	}
}

func servirRutaDietasConPermisoExpreso(handler *Handler, rec *httptest.ResponseRecorder, req *http.Request) {
	handler.handleDietasRoadRoute(rec, req, principalRutaDietasPrueba())
}

func TestDietasRoadRouteRequiresInternalOSRM(t *testing.T) {
	handler := newTestHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/api/vec/dietas/road-route", strings.NewReader(`{"coordinates":[{"lat":37.1773,"lon":-3.5986},{"lat":37.2306,"lon":-3.6554}]}`))
	rec := httptest.NewRecorder()
	servirRutaDietasConPermisoExpreso(handler, rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("road route status = %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "OSRM interno no configurado") {
		t.Fatalf("road route error does not explain internal OSRM requirement: %s", rec.Body.String())
	}
}

func TestDietasRoadRouteRejectsIncompleteOrNonCanonicalConfiguration(t *testing.T) {
	store := memory.NewStore()
	service, err := application.NewService(store, store, store)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	valid := testOSRMOptions("http://127.0.0.1:5000")
	tests := []struct {
		name    string
		options HandlerOptions
	}{
		{name: "URL sin ambito ni red", options: HandlerOptions{OSRMBaseURL: valid.OSRMBaseURL}},
		{name: "sin nombre", options: HandlerOptions{
			OSRMBaseURL: valid.OSRMBaseURL, OSRMScopeBounds: valid.OSRMScopeBounds, OSRMAllowedCIDRs: valid.OSRMAllowedCIDRs,
		}},
		{name: "sin limites", options: HandlerOptions{
			OSRMBaseURL: valid.OSRMBaseURL, OSRMScopeName: valid.OSRMScopeName, OSRMAllowedCIDRs: valid.OSRMAllowedCIDRs,
		}},
		{name: "sin redes", options: HandlerOptions{
			OSRMBaseURL: valid.OSRMBaseURL, OSRMScopeName: valid.OSRMScopeName, OSRMScopeBounds: valid.OSRMScopeBounds,
		}},
		{name: "limites mal formados", options: HandlerOptions{
			OSRMBaseURL: valid.OSRMBaseURL, OSRMScopeName: valid.OSRMScopeName,
			OSRMScopeBounds: "36.45,-4.6,38.25", OSRMAllowedCIDRs: valid.OSRMAllowedCIDRs,
		}},
		{name: "limites no canonicos", options: HandlerOptions{
			OSRMBaseURL: valid.OSRMBaseURL, OSRMScopeName: valid.OSRMScopeName,
			OSRMScopeBounds: "36.45,-4.60,38.25,-2.15", OSRMAllowedCIDRs: valid.OSRMAllowedCIDRs,
		}},
		{name: "esquema no permitido", options: HandlerOptions{
			OSRMBaseURL: "ftp://127.0.0.1:5000", OSRMScopeName: valid.OSRMScopeName,
			OSRMScopeBounds: valid.OSRMScopeBounds, OSRMAllowedCIDRs: valid.OSRMAllowedCIDRs,
		}},
		{name: "URL con consulta", options: HandlerOptions{
			OSRMBaseURL: "http://127.0.0.1:5000?destino=otro", OSRMScopeName: valid.OSRMScopeName,
			OSRMScopeBounds: valid.OSRMScopeBounds, OSRMAllowedCIDRs: valid.OSRMAllowedCIDRs,
		}},
		{name: "red universal", options: HandlerOptions{
			OSRMBaseURL: valid.OSRMBaseURL, OSRMScopeName: valid.OSRMScopeName,
			OSRMScopeBounds: valid.OSRMScopeBounds, OSRMAllowedCIDRs: []string{"0.0.0.0/0"},
		}},
		{name: "red no canonica", options: HandlerOptions{
			OSRMBaseURL: valid.OSRMBaseURL, OSRMScopeName: valid.OSRMScopeName,
			OSRMScopeBounds: valid.OSRMScopeBounds, OSRMAllowedCIDRs: []string{"127.0.0.1/8"},
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := NewHandlerWithOptions(service, test.options); err == nil {
				t.Fatal("la configuracion OSRM incompleta o no canonica fue aceptada")
			}
		})
	}
}

func TestDietasRoadRouteUsesConfiguredInternalOSRM(t *testing.T) {
	var requestedPath string
	osrm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.RequestURI()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":"Ok","routes":[{"distance":13400,"duration":1080,"geometry":{"type":"LineString","coordinates":[[-3.5986,37.1773],[-3.6200,37.2050],[-3.6554,37.2306]]}}],"waypoints":[],"data_version":"2026-06-20T00:00:00Z"}`))
	}))
	defer osrm.Close()
	handler := newTestHandlerWithOptions(t, testOSRMOptions(osrm.URL))
	req := httptest.NewRequest(http.MethodPost, "/api/vec/dietas/road-route", strings.NewReader(`{"coordinates":[{"lat":37.1773,"lon":-3.5986,"name":"Granada"},{"lat":37.2306,"lon":-3.6554,"name":"Albolote"}]}`))
	rec := httptest.NewRecorder()
	servirRutaDietasConPermisoExpreso(handler, rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("road route status = %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(requestedPath, "/route/v1/driving/-3.598600,37.177300;-3.655400,37.230600") {
		t.Fatalf("OSRM path = %s", requestedPath)
	}
	if !strings.Contains(requestedPath, "alternatives=1") {
		t.Fatalf("OSRM path does not request default alternative count: %s", requestedPath)
	}
	if !strings.Contains(requestedPath, "steps=false") {
		t.Fatalf("OSRM path should avoid heavy step payloads: %s", requestedPath)
	}
	body := rec.Body.String()
	for _, want := range []string{`"engine":"osrm_on_premise"`, `"distance":13400`, `"data_version":"2026-06-20T00:00:00Z"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("road route response missing %s: %s", want, body)
		}
	}
}

func TestDietasRoadRouteDoesNotConnectOutsideExplicitNetworks(t *testing.T) {
	var hits atomic.Int32
	osrm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		_, _ = w.Write([]byte(`{"code":"Ok","routes":[]}`))
	}))
	defer osrm.Close()
	options := testOSRMOptions(osrm.URL)
	options.OSRMAllowedCIDRs = []string{"10.0.0.0/8"}
	handler := newTestHandlerWithOptions(t, options)
	req := httptest.NewRequest(http.MethodPost, "/api/vec/dietas/road-route", strings.NewReader(`{"coordinates":[{"lat":37.1773,"lon":-3.5986},{"lat":37.2306,"lon":-3.6554}]}`))
	rec := httptest.NewRecorder()
	servirRutaDietasConPermisoExpreso(handler, rec, req)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("road route status = %d: %s", rec.Code, rec.Body.String())
	}
	if hits.Load() != 0 {
		t.Fatalf("OSRM fuera de la red explicita recibio %d peticiones", hits.Load())
	}
}

func TestDietasRoadRouteDoesNotFollowRedirects(t *testing.T) {
	var redirectedHits atomic.Int32
	redirected := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		redirectedHits.Add(1)
		_, _ = w.Write([]byte(`{"code":"Ok","routes":[]}`))
	}))
	defer redirected.Close()
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, redirected.URL+r.URL.RequestURI(), http.StatusTemporaryRedirect)
	}))
	defer origin.Close()
	handler := newTestHandlerWithOptions(t, testOSRMOptions(origin.URL))
	req := httptest.NewRequest(http.MethodPost, "/api/vec/dietas/road-route", strings.NewReader(`{"coordinates":[{"lat":37.1773,"lon":-3.5986},{"lat":37.2306,"lon":-3.6554}]}`))
	rec := httptest.NewRecorder()
	servirRutaDietasConPermisoExpreso(handler, rec, req)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("road route status = %d: %s", rec.Code, rec.Body.String())
	}
	if redirectedHits.Load() != 0 {
		t.Fatalf("se siguio una redireccion OSRM no autorizada: %d peticiones", redirectedHits.Load())
	}
}

func TestDietasRoadRouteCanRequestAlternatives(t *testing.T) {
	var requestedPath string
	osrm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.RequestURI()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":"Ok","routes":[]}`))
	}))
	defer osrm.Close()
	handler := newTestHandlerWithOptions(t, testOSRMOptions(osrm.URL))
	req := httptest.NewRequest(http.MethodPost, "/api/vec/dietas/road-route", strings.NewReader(`{"alternatives":3,"coordinates":[{"lat":37.1773,"lon":-3.5986},{"lat":37.2306,"lon":-3.6554}]}`))
	rec := httptest.NewRecorder()
	servirRutaDietasConPermisoExpreso(handler, rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("road route status = %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(requestedPath, "alternatives=3") {
		t.Fatalf("OSRM path does not request alternatives: %s", requestedPath)
	}
}

func TestDietasRoadRouteRejectsCoordinatesOutsideGranadaScope(t *testing.T) {
	handler := newTestHandlerWithOptions(t, testOSRMOptions("http://127.0.0.1:5000"))
	req := httptest.NewRequest(http.MethodPost, "/api/vec/dietas/road-route", strings.NewReader(`{"coordinates":[{"lat":37.1773,"lon":-3.5986},{"lat":40.4168,"lon":-3.7038}]}`))
	rec := httptest.NewRecorder()
	servirRutaDietasConPermisoExpreso(handler, rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("road route status = %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Granada provincia + 15 km") {
		t.Fatalf("road route error does not mention scope: %s", rec.Body.String())
	}
}

func TestDietasRoadRouteScopeCanBeChangedByConfiguration(t *testing.T) {
	osrm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"code":"Ok","routes":[]}`))
	}))
	defer osrm.Close()
	handler := newTestHandlerWithOptions(t, HandlerOptions{
		OSRMBaseURL:      osrm.URL,
		OSRMScopeName:    "Ambito prueba",
		OSRMScopeBounds:  "35,-5,42,-1",
		OSRMAllowedCIDRs: []string{"127.0.0.1/32"},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/vec/dietas/road-route", strings.NewReader(`{"coordinates":[{"lat":37.1773,"lon":-3.5986},{"lat":40.4168,"lon":-3.7038}]}`))
	rec := httptest.NewRecorder()
	servirRutaDietasConPermisoExpreso(handler, rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"route_scope":"Ambito prueba"`) {
		t.Fatalf("road route did not use the explicit scope: status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestSessionBuildsPrincipalFromDNIeCertificate(t *testing.T) {
	handler := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/api/vec/session", nil)
	certificado := &x509.Certificate{
		Raw:          []byte("certificado-dnie-verificado-prueba"),
		SerialNumber: big.NewInt(123456),
		Subject: pkix.Name{
			CommonName:   "Persona de Prueba",
			SerialNumber: "00000000T",
		},
		Issuer:      pkix.Name{CommonName: "DNIe Direccion General de la Policia"},
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	req.TLS = &tls.ConnectionState{
		HandshakeComplete: true,
		Version:           tls.VersionTLS13,
		PeerCertificates:  []*x509.Certificate{certificado},
		VerifiedChains:    [][]*x509.Certificate{{certificado}},
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("un certificado sin perfil explicito obtuvo sesion: %d: %s", rec.Code, rec.Body.String())
	}
	identidad := identityFromRequest(req, identityPolicy{})
	for field, got := range map[string]string{
		"id": identidad.subject, "nombre": identidad.displayName, "metodo": string(identidad.method),
		"garantia": string(identidad.assurance), "dni": identidad.attributes["dni"],
	} {
		if got == "" {
			t.Fatalf("identidad TLS verificada sin %s: %+v", field, identidad)
		}
	}
	if identidad.subject == "00000000T" || len(identidad.roles) != 0 {
		t.Fatalf("el certificado se convirtio en ID directo o rol implicito: %+v", identidad)
	}
}

func TestCertificadoTLSNoVerificadoNoAutentica(t *testing.T) {
	store := memory.NewStore()
	service, err := application.NewService(store, store, store)
	if err != nil {
		t.Fatal(err)
	}
	handler, err := NewHandler(service)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/vec/session", nil)
	req.TLS = &tls.ConnectionState{
		HandshakeComplete: true,
		Version:           tls.VersionTLS13,
		PeerCertificates: []*x509.Certificate{{
			Raw: []byte("certificado-no-verificado"), SerialNumber: big.NewInt(7),
			ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		}},
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("certificado sin cadena verificada aceptado: estado=%d cuerpo=%s", rec.Code, rec.Body.String())
	}
}

func TestIdentidadDeCabeceraYCertificadoNoEnlazadosDeniegan(t *testing.T) {
	handler := newTestHandler(t)
	certificado := &x509.Certificate{
		Raw: []byte("certificado-verificado-de-otra-identidad"), SerialNumber: big.NewInt(8),
		Subject:     pkix.Name{SerialNumber: "87654321X"},
		Issuer:      pkix.Name{CommonName: "DNIe Direccion General de la Policia"},
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/vec/session", nil)
	req.Header.Set("X-Auth-Subject", "cuenta-kerberos-distinta")
	req.Header.Set("X-Auth-Mechanism", "kerberos_ad")
	req.TLS = &tls.ConnectionState{
		HandshakeComplete: true, Version: tls.VersionTLS13,
		PeerCertificates: []*x509.Certificate{certificado}, VerifiedChains: [][]*x509.Certificate{{certificado}},
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("identidades no enlazadas aceptadas: estado=%d cuerpo=%s", rec.Code, rec.Body.String())
	}
}

func TestHandlerSeguroDeniegaPeticionSinIdentidad(t *testing.T) {
	store := memory.NewStore()
	service, err := application.NewService(store, store, store)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	handler, err := NewHandler(service)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/vec/session", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("peticion anonima status = %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCabecerasDeIdentidadSoloSeAceptanDesdeProxyConfiable(t *testing.T) {
	store := memory.NewStore()
	service, err := application.NewService(store, store, store)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	handler, err := NewHandlerWithOptions(service, HandlerOptions{
		TrustIdentityHeaders:    true,
		TrustedProxyCIDRs:       []string{"10.20.30.0/24"},
		IdentitySubjectHeader:   "X-VEC-Subject",
		IdentityRolesHeader:     "X-VEC-Roles",
		IdentityMechanismHeader: "X-VEC-Auth-Mechanism",
	})
	if err != nil {
		t.Fatalf("NewHandlerWithOptions() error = %v", err)
	}
	nuevaPeticion := func(remota string) *http.Request {
		req := httptest.NewRequest(http.MethodGet, "/api/vec/session", nil)
		req.RemoteAddr = remota
		req.Header.Set("X-VEC-Subject", "usuario-interno-1")
		req.Header.Set("X-VEC-Roles", "administrador")
		req.Header.Set("X-VEC-Auth-Mechanism", "kerberos_ad")
		req.Header.Set("X-Auth-Assurance", "sustancial")
		return req
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, nuevaPeticion("203.0.113.15:5000"))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("cabeceras falsificadas status = %d: %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	peticionConfiable := nuevaPeticion("10.20.30.8:5000")
	handler.ServeHTTP(rec, peticionConfiable)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("una identidad sin concesion obtuvo la sesion: %d: %s", rec.Code, rec.Body.String())
	}

	peticionPrivilegiada := nuevaPeticion("10.20.30.8:5000")
	peticionPrivilegiada.URL.Path = "/api/vec/audit"
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, peticionPrivilegiada)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("un rol afirmado por el proxy concedio permisos funcionales: %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDietasRoadRouteExigePermisoExpreso(t *testing.T) {
	handler := newTestHandlerWithOptions(t, testOSRMOptions("http://127.0.0.1:5000"))
	req := httptest.NewRequest(http.MethodPost, "/api/vec/dietas/road-route", strings.NewReader(`{"coordinates":[{"lat":37.1773,"lon":-3.5986},{"lat":37.2306,"lon":-3.6554}]}`))
	req.Header.Set("X-Auth-Subject", "candidate")
	req.Header.Set("X-Auth-Mechanism", "clave")
	req.Header.Set("X-Auth-Roles", "ciudadano")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("ruta sin permiso expreso status = %d: %s", rec.Code, rec.Body.String())
	}
}

func TestWorkspaceYAuditoriaAplicanPermisos(t *testing.T) {
	handler := newTestHandler(t)
	ciudadano := func(path string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("X-Auth-Subject", "candidate")
		req.Header.Set("X-Auth-Mechanism", "clave")
		req.Header.Set("X-Auth-Roles", "ciudadano")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}
	if rec := ciudadano("/api/vec/workspace"); rec.Code != http.StatusForbidden {
		t.Fatalf("workspace ciudadano status = %d: %s", rec.Code, rec.Body.String())
	}
	if rec := ciudadano("/api/vec/audit"); rec.Code != http.StatusForbidden {
		t.Fatalf("auditoria ciudadana status = %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSessionUsesConfiguredRolesForExternalIdentity(t *testing.T) {
	handler := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/api/vec/session", nil)
	req.Header.Set("X-Auth-Subject", "rrhh-001")
	req.Header.Set("X-Auth-Mechanism", "dnie")
	req.Header.Set("X-Auth-Roles", "tecnico_rrhh")
	req.Header.Set("X-Auth-Assurance", "alto")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/vec/session status = %d: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{`"id":"rrhh-001"`, `"auth_method":"dnie"`, `"tecnico_rrhh"`, `"vec.session.read"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("session body missing %s: %s", want, body)
		}
	}
	for _, prohibido := range []string{
		personalmodule.PermissionEmployeeManage,
		personalmodule.PermissionPayrollRead,
		cronosmodule.PermissionTimeRead,
		dietasmodule.PermissionExpenseRead,
		adminmodule.PermissionRolesManage,
	} {
		if strings.Contains(body, prohibido) {
			t.Fatalf("la sesion tecnica heredo el permiso no concedido %q: %s", prohibido, body)
		}
	}
}

func servirCatalogoPersonalConPermisosExpresos(handler *Handler, rec *httptest.ResponseRecorder, req *http.Request, permisos ...string) {
	principal := principalConPermisosExpresosPrueba(permisos...)
	path := vecPath(req.URL.Path)
	switch {
	case path == "/personal/rpt/positions":
		handler.handlePersonalRPTPositions(rec, req, principal)
	case strings.HasPrefix(path, "/personal/rpt/positions/"):
		handler.handlePersonalRPTPosition(rec, req, principal, path)
	case path == "/personal/rpt/stats":
		handler.handlePersonalRPTStats(rec, req, principal)
	case path == "/personal/categories":
		handler.handlePersonalCategories(rec, req, principal)
	case strings.HasPrefix(path, "/personal/categories/"):
		handler.handlePersonalCategory(rec, req, principal, path)
	case path == "/personal/catalogs":
		handler.handlePersonalCatalogs(rec, req, principal)
	default:
		handler.writeError(rec, http.StatusNotFound, "ruta de prueba no soportada")
	}
}

func TestPersonalRPTCatalogEndpoints(t *testing.T) {
	handler := newTestHandler(t)
	for _, path := range []string{
		"/api/vec/personal/rpt/positions?q=administrativo",
		"/api/vec/personal/rpt/positions/8",
		"/api/vec/personal/rpt/stats",
		"/api/vec/personal/categories?q=trabajador",
		"/api/vec/personal/catalogs",
	} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		servirCatalogoPersonalConPermisosExpresos(handler, rec, req, personalmodule.PermissionPositionRead)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status = %d: %s", path, rec.Code, rec.Body.String())
		}
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/vec/personal/rpt/positions/999", strings.NewReader(`{"name":"Puesto prueba","dot":1,"group":"A1","provision":"C","state":"Vigente"}`))
	servirCatalogoPersonalConPermisosExpresos(handler, rec, req, personalmodule.PermissionPositionManage)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT /api/vec/personal/rpt/positions/999 status = %d: %s", rec.Code, rec.Body.String())
	}
	if body := rec.Body.String(); !strings.Contains(body, `"code":"999"`) || !strings.Contains(body, "personal.rpt.position.upsert") {
		t.Fatalf("PUT body missing position/audit receipt: %s", body)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/api/vec/personal/rpt/positions/999", nil)
	servirCatalogoPersonalConPermisosExpresos(handler, rec, req, personalmodule.PermissionPositionManage)
	if rec.Code != http.StatusOK {
		t.Fatalf("DELETE /api/vec/personal/rpt/positions/999 status = %d: %s", rec.Code, rec.Body.String())
	}
	if body := rec.Body.String(); !strings.Contains(body, "personal.rpt.position.delete") {
		t.Fatalf("DELETE body missing audit receipt: %s", body)
	}
}

func TestPersonalRPTCatalogCargaFixtureSinteticoEnMemoria(t *testing.T) {
	handler := newTestHandler(t)
	esperadas := posicionesRPTSinteticasPrueba()
	stats, err := handler.personalCatalog.Stats(context.Background())
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}
	if stats.Positions != len(esperadas) {
		t.Fatalf("stats positions = %d, want synthetic fixture with %d rows", stats.Positions, len(esperadas))
	}
	page, err := handler.personalCatalog.ListPositions(context.Background(), personaldomain.RPTPositionFilter{Limit: 2000})
	if err != nil {
		t.Fatalf("ListPositions() error = %v", err)
	}
	if page.Total != len(esperadas) || len(page.Items) != len(esperadas) {
		t.Fatalf("positions page total/items = %d/%d, want %d/%d", page.Total, len(page.Items), len(esperadas), len(esperadas))
	}
	dotaciones := 0
	for _, position := range page.Items {
		dotaciones += position.Dot
	}
	dotacionesEsperadas := 0
	for _, position := range esperadas {
		dotacionesEsperadas += position.Dot
	}
	if dotaciones != dotacionesEsperadas {
		t.Fatalf("dotations = %d, want %d", dotaciones, dotacionesEsperadas)
	}
}

func TestPersonalCategoryCRUDEndpoints(t *testing.T) {
	handler := newTestHandler(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/vec/personal/categories", strings.NewReader(`{"slug":"nueva-categoria","name":"Nueva Categoria","area":"administracion_especial","source":"test","state":"vigente"}`))
	servirCatalogoPersonalConPermisosExpresos(handler, rec, req, personalmodule.PermissionPositionManage)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /api/vec/personal/categories status = %d: %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/api/vec/personal/categories/nueva-categoria", strings.NewReader(`{"name":"Nueva Categoria Editada","area":"administracion_general","source":"test","state":"vigente"}`))
	servirCatalogoPersonalConPermisosExpresos(handler, rec, req, personalmodule.PermissionPositionManage)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Nueva Categoria Editada") {
		t.Fatalf("PUT category status = %d: %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/vec/personal/categories/nueva-categoria", nil)
	servirCatalogoPersonalConPermisosExpresos(handler, rec, req, personalmodule.PermissionPositionRead)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "nueva-categoria") {
		t.Fatalf("GET category status = %d: %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/api/vec/personal/categories/nueva-categoria", nil)
	servirCatalogoPersonalConPermisosExpresos(handler, rec, req, personalmodule.PermissionPositionManage)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "personal.category.delete") {
		t.Fatalf("DELETE category status = %d: %s", rec.Code, rec.Body.String())
	}
}

func TestPersonalCategorySinEstadoExpresoNoSeActiva(t *testing.T) {
	handler := newTestHandler(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/vec/personal/categories", strings.NewReader(`{"slug":"sin-estado","name":"Categoria sin estado","area":"administracion_especial","source":"test"}`))
	servirCatalogoPersonalConPermisosExpresos(handler, rec, req, personalmodule.PermissionPositionManage)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("POST categoria sin estado status = %d: %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/vec/personal/categories/sin-estado", nil)
	servirCatalogoPersonalConPermisosExpresos(handler, rec, req, personalmodule.PermissionPositionRead)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET categoria rechazada status = %d: %s", rec.Code, rec.Body.String())
	}
}

func TestPersonalRPTCatalogNoSeConcedePorRolGrueso(t *testing.T) {
	handler := newTestHandler(t)
	for _, rol := range []string{"ciudadano", "tecnico_rrhh", "administrativo", "jefatura_rrhh", "personal_interno", "jefe_servicio", "jefe_seccion", "administrador"} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/vec/personal/rpt/positions", nil)
		req.Header.Set("X-Auth-Subject", "actor-"+rol)
		req.Header.Set("X-Auth-Mechanism", "clave")
		req.Header.Set("X-Auth-Roles", rol)
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("rol %s GET RPT status = %d: %s", rol, rec.Code, rec.Body.String())
		}
	}
}

func TestCronosTimecardsLegacyDeniegaIdentificadorDeEmpleadoDelCliente(t *testing.T) {
	store := memory.NewStore()
	service, internal, err := application.NewServiceWithInternalOperations(store, store, store)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	if err := internal.RegisterModule(context.Background(), cronosmodule.Manifest()); err != nil {
		t.Fatalf("RegisterModule(cronos) error = %v", err)
	}
	handler, err := NewHandlerWithOptions(service, HandlerOptions{
		AllowDemoIdentity:    true,
		DemoIdentityResolver: resolvedorIdentidadPruebas{},
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	payload := []byte(`{
		"employee_id":"EMP-0999",
		"date":"2026-06-19",
		"age":64,
		"profile_id":"H-FLEX-ADM",
		"punches":[
			{"at":"08:00","kind":"entrada","channel":"web"},
			{"at":"13:30","kind":"salida","channel":"web"}
		]
	}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/vec/cronos/timecards", bytes.NewReader(payload))
	servirCronosCerradoConPermisosPreliminares(handler, rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("POST /api/vec/cronos/timecards legacy status = %d: %s", rec.Code, rec.Body.String())
	}
	if body := rec.Body.String(); strings.Contains(body, "EMP-0999") || strings.Contains(body, "05:30") {
		t.Fatalf("la superficie cerrada reveló o procesó datos del empleado: %s", body)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
	snapshot, err := handler.cronos.Snapshot(context.Background(), workspaceCronosDate, workspaceCronosEmployeeIDs())
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}
	for _, result := range snapshot.Results {
		if result.EmployeeID == "EMP-0999" {
			t.Fatalf("la petición denegada persistió una jornada: %#v", result)
		}
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/vec/cronos/timecards?employee_id=EMP-0031", nil)
	servirCronosCerradoConPermisosPreliminares(handler, rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("GET /api/vec/cronos/timecards legacy status = %d: %s", rec.Code, rec.Body.String())
	}
	if body := rec.Body.String(); strings.Contains(body, "EMP-0031") || strings.Contains(body, "timecards") {
		t.Fatalf("la consulta cerrada reveló datos Cronos: %s", body)
	}
}

func TestCronosPermisosLegacyDeniegaIdentificadorDeEmpleadoDelCliente(t *testing.T) {
	store := memory.NewStore()
	service, internal, err := application.NewServiceWithInternalOperations(store, store, store)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	if err := internal.RegisterModule(context.Background(), cronosmodule.Manifest()); err != nil {
		t.Fatalf("RegisterModule(cronos) error = %v", err)
	}
	handler, err := NewHandlerWithOptions(service, HandlerOptions{
		AllowDemoIdentity:    true,
		DemoIdentityResolver: resolvedorIdentidadPruebas{},
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	requestsBefore, err := handler.cronos.LeaveRequests(context.Background(), "EMP-0031", 2026)
	if err != nil {
		t.Fatalf("LeaveRequests() before error = %v", err)
	}
	payload := []byte(`{
		"employee_id":"EMP-0031",
		"policy_id":"asuntos_propios",
		"from":"2026-06-26",
		"to":"2026-06-26",
		"amount":1,
		"reason":"Asunto propio"
	}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/vec/cronos/leave-requests", bytes.NewReader(payload))
	servirCronosCerradoConPermisosPreliminares(handler, rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("POST /api/vec/cronos/leave-requests legacy status = %d: %s", rec.Code, rec.Body.String())
	}
	if body := rec.Body.String(); strings.Contains(body, "cronos.permiso.request") || strings.Contains(body, "asuntos_propios") || strings.Contains(body, "EMP-0031") {
		t.Fatalf("la superficie cerrada reveló o procesó la solicitud: %s", body)
	}
	requestsAfter, err := handler.cronos.LeaveRequests(context.Background(), "EMP-0031", 2026)
	if err != nil {
		t.Fatalf("LeaveRequests() after error = %v", err)
	}
	if len(requestsAfter) != len(requestsBefore) {
		t.Fatalf("la petición denegada mutó solicitudes: before=%d after=%d", len(requestsBefore), len(requestsAfter))
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/vec/cronos/leave-requests?employee_id=EMP-0031", nil)
	servirCronosCerradoConPermisosPreliminares(handler, rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("GET /api/vec/cronos/leave-requests legacy status = %d: %s", rec.Code, rec.Body.String())
	}
	if body := rec.Body.String(); strings.Contains(body, "Asunto propio") || strings.Contains(body, `"remaining"`) || strings.Contains(body, "EMP-0031") {
		t.Fatalf("la consulta cerrada reveló datos de permisos: %s", body)
	}
}

func TestWorkspaceSnapshotIncludesProfessionalDemoPayload(t *testing.T) {
	snapshot, err := workspaceSnapshot(nil)
	if err != nil {
		t.Fatalf("workspaceSnapshot() error = %v", err)
	}
	for _, key := range []string{
		"generated_at",
		"modules",
		"kpis",
		"operational_records",
		"screen_catalog",
		"payroll_run",
		"expense_policy",
		"action_catalog",
		"flow_states",
		"access_roles",
		"role_assignments",
		"rpt_catalog",
		"rpt_contract_types",
		"rpt_position_samples",
		"professional_categories",
		"professional_category_aliases",
		"bolsa_category_rules",
	} {
		if snapshot[key] == nil {
			t.Fatalf("workspace snapshot missing %q", key)
		}
	}

	screenCatalog, ok := snapshot["screen_catalog"].([]map[string]any)
	if !ok || len(screenCatalog) == 0 {
		t.Fatalf("screen_catalog = %#v, want non-empty []map[string]any", snapshot["screen_catalog"])
	}
	var hasPayrollScreen bool
	for _, screen := range screenCatalog {
		if screen["id"] == "nominas.cierre" {
			hasPayrollScreen = true
			fields, ok := screen["fields"].([]map[string]any)
			if !ok || len(fields) == 0 {
				t.Fatalf("nominas.cierre fields = %#v, want non-empty field catalog", screen["fields"])
			}
		}
	}
	if !hasPayrollScreen {
		t.Fatalf("screen_catalog missing nominas.cierre: %#v", screenCatalog)
	}

	payrollRun, ok := snapshot["payroll_run"].(map[string]any)
	if !ok || payrollRun["period"] != "2026-06" || payrollRun["state"] == "" {
		t.Fatalf("payroll_run = %#v, want demo payroll period and state", snapshot["payroll_run"])
	}
	expensePolicy, ok := snapshot["expense_policy"].(map[string]any)
	if !ok || expensePolicy["demo_notice"] == "" {
		t.Fatalf("expense_policy = %#v, want demo policy metadata", snapshot["expense_policy"])
	}

	actionCatalog, ok := snapshot["action_catalog"].(map[string][]map[string]any)
	if !ok {
		t.Fatalf("action_catalog = %#v, want map[string][]map[string]any", snapshot["action_catalog"])
	}
	for _, module := range []string{"personal", "nominas", "cronos", "horarios", "permisos", "dietas", "rutas", "bolsa"} {
		if len(actionCatalog[module]) == 0 {
			t.Fatalf("action_catalog missing actions for %s: %#v", module, actionCatalog)
		}
	}

	flowStates, ok := snapshot["flow_states"].([]map[string]any)
	if !ok || len(flowStates) == 0 {
		t.Fatalf("flow_states = %#v, want non-empty []map[string]any", snapshot["flow_states"])
	}
	var hasPayrollHandoff bool
	for _, state := range flowStates {
		if state["id"] == "lista_para_nomina" {
			hasPayrollHandoff = true
		}
	}
	if !hasPayrollHandoff {
		t.Fatalf("flow_states missing lista_para_nomina: %#v", flowStates)
	}

	accessRoles, ok := snapshot["access_roles"].([]map[string]any)
	if !ok || len(accessRoles) < 5 {
		t.Fatalf("access_roles = %#v, want role matrix for VEC", snapshot["access_roles"])
	}
	var hasAdmin, hasRRHH, hasJefeServicio bool
	for _, role := range accessRoles {
		switch role["id"] {
		case "administrador":
			hasAdmin = true
		case "tecnico_rrhh":
			hasRRHH = true
		case "jefe_servicio":
			hasJefeServicio = true
		}
	}
	if !hasAdmin || !hasRRHH || !hasJefeServicio {
		t.Fatalf("access_roles missing expected public-sector roles: %#v", accessRoles)
	}

	rptTypes, ok := snapshot["rpt_contract_types"].([]map[string]any)
	if !ok || len(rptTypes) == 0 {
		t.Fatalf("rpt_contract_types = %#v, want RPT/personnel catalog", snapshot["rpt_contract_types"])
	}
	var hasFuncionario, hasProvision bool
	for _, item := range rptTypes {
		if item["code"] == "funcionario_carrera" {
			hasFuncionario = true
		}
		if item["catalog"] == "forma_provision" && item["code"] == "L" {
			hasProvision = true
		}
	}
	if !hasFuncionario || !hasProvision {
		t.Fatalf("rpt_contract_types missing regime/provision entries: %#v", rptTypes)
	}

	categories, ok := snapshot["professional_categories"].([]map[string]any)
	if !ok || len(categories) < 50 {
		t.Fatalf("professional_categories = %#v, want Diputacion category master from Bolsa/OPES", snapshot["professional_categories"])
	}
	var hasAdministrativo, hasTrabajadorSocial bool
	for _, category := range categories {
		switch category["slug"] {
		case "administrativo":
			hasAdministrativo = true
		case "trabajador-social":
			hasTrabajadorSocial = true
		}
	}
	if !hasAdministrativo || !hasTrabajadorSocial {
		t.Fatalf("professional_categories missing expected categories: %#v", categories)
	}
}
