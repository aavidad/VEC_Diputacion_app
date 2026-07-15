package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/config"
)

func TestServerHealthzIsJSON(t *testing.T) {
	handler := NewHandler(http.NotFoundHandler())
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, peticionServidorPrueba(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("/healthz status = %d, want 200", rec.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("/healthz json: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("/healthz body = %#v, want status ok", body)
	}
}

func TestServerServesStaticUI(t *testing.T) {
	handler := NewHandler(http.NotFoundHandler())
	for _, tc := range []struct {
		path        string
		contentType string
		want        string
	}{
		{path: "/", contentType: "text/html", want: "VEC Diputacion"},
		{path: "/app.js", contentType: "text/javascript", want: `fetch("/api/demo"`},
		{path: "/styles.css", contentType: "text/css", want: ".listings"},
		{path: "/locales/es.json", contentType: "application/json", want: "api.candidate.created"},
	} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, peticionServidorPrueba(http.MethodGet, tc.path, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status = %d, want 200", tc.path, rec.Code)
		}
		if got := rec.Header().Get("Content-Type"); !strings.Contains(got, tc.contentType) {
			t.Fatalf("%s content-type = %q, want %q", tc.path, got, tc.contentType)
		}
		if !strings.Contains(rec.Body.String(), tc.want) {
			t.Fatalf("%s body missing %q", tc.path, tc.want)
		}
	}
}

func TestServerSirvePortalBolsaPermanenteSinEstilosInline(t *testing.T) {
	handler := NewHandler(http.NotFoundHandler())
	for _, prueba := range []struct{ ruta, tipo, contenido string }{
		{ruta: "/bolsa/", tipo: "text/html", contenido: "Convocatorias abiertas y próximas"},
		{ruta: "/bolsa/bolsa.css?v=1", tipo: "text/css", contenido: ".espacio-trabajo-publico"},
		{ruta: "/bolsa/bolsa.js?v=1", tipo: "text/javascript", contenido: "/api/publico/bolsa/convocatorias"},
	} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, peticionServidorPrueba(http.MethodGet, prueba.ruta, nil))
		if rec.Code != http.StatusOK || !strings.Contains(rec.Header().Get("Content-Type"), prueba.tipo) || !strings.Contains(rec.Body.String(), prueba.contenido) {
			t.Fatalf("%s = %d %q", prueba.ruta, rec.Code, rec.Body.String())
		}
		if prueba.tipo == "text/html" && (strings.Contains(strings.ToLower(rec.Body.String()), "<style") || strings.Contains(strings.ToLower(rec.Body.String()), " style=")) {
			t.Fatalf("%s contiene CSS inline", prueba.ruta)
		}
		if prueba.ruta == "/bolsa/bolsa.js?v=1" && !strings.Contains(rec.Body.String(), "datos.fuente.demostracion === true") {
			t.Fatal("la UI no gobierna el aviso DEMOSTRACIÓN desde la fuente")
		}
	}
}

func TestInterfazNoDeduceTitularidadNiPermisosPorContenido(t *testing.T) {
	handler := NewHandler(http.NotFoundHandler())
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, peticionServidorPrueba(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("/app.js status = %d, want 200", rec.Code)
	}

	contenido := rec.Body.String()
	for _, patronProhibido := range []string{
		"__persona_privada__|__dni_privado__",
		`["administrador", "tecnico_rrhh", "rrhh", "personal_rrhh"]`,
	} {
		if strings.Contains(strings.ToLower(contenido), strings.ToLower(patronProhibido)) {
			t.Fatalf("la interfaz conserva una inferencia permisiva prohibida: %q", patronProhibido)
		}
	}
	for _, cierreEsperado := range []string{
		`["empleado", "ciudadano"].includes(sessionAccessProfile().id)`,
		`return sessionAccessProfile().id === "tecnico_rrhh"`,
	} {
		if !strings.Contains(contenido, cierreEsperado) {
			t.Fatalf("la interfaz no conserva el cierre explicito %q", cierreEsperado)
		}
	}
}

func TestServerCachesVersionedStaticAssets(t *testing.T) {
	handler := NewHandler(http.NotFoundHandler())
	for _, tc := range []struct {
		path      string
		wantCache string
	}{
		{path: "/", wantCache: "no-store"},
		{path: "/app.js", wantCache: "no-cache"},
		{path: "/app.js?v=20260621-fixes", wantCache: "public, max-age=31536000, immutable"},
		{path: "/styles.css?v=20260621-fixes", wantCache: "public, max-age=31536000, immutable"},
	} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, peticionServidorPrueba(http.MethodGet, tc.path, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status = %d, want 200", tc.path, rec.Code)
		}
		if got := rec.Header().Get("Cache-Control"); got != tc.wantCache {
			t.Fatalf("%s cache-control = %q, want %q", tc.path, got, tc.wantCache)
		}
	}
}

func TestNewHTTPServerUsesConfigAndRoutesAPI(t *testing.T) {
	api := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusAccepted) })
	srv, err := NewHTTPServer(config.Config{Address: "127.0.0.1:0", ReadHeaderTimeout: time.Second}, api)
	if err != nil {
		t.Fatalf("NewHTTPServer() error = %v", err)
	}
	if srv.Addr != "127.0.0.1:0" || srv.ReadHeaderTimeout != time.Second ||
		srv.ReadTimeout <= 0 || srv.WriteTimeout <= 0 || srv.IdleTimeout <= 0 || srv.MaxHeaderBytes <= 0 {
		t.Fatalf("server config = %s/%s", srv.Addr, srv.ReadHeaderTimeout)
	}
	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, peticionServidorPrueba(http.MethodPost, "/api/candidates", nil))
	if rec.Code != http.StatusAccepted {
		t.Fatalf("api status = %d, want 202", rec.Code)
	}
}

func TestServerAplicaCabecerasDeSeguridad(t *testing.T) {
	handler := NewHandler(http.NotFoundHandler())
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, peticionServidorPrueba(http.MethodGet, "/healthz", nil))
	for nombre, esperado := range map[string]string{
		"Cache-Control":             "no-store",
		"Pragma":                    "no-cache",
		"Content-Security-Policy":   "default-src 'self'",
		"Referrer-Policy":           "no-referrer",
		"X-Content-Type-Options":    "nosniff",
		"X-Frame-Options":           "DENY",
		"Strict-Transport-Security": "max-age=31536000",
	} {
		if valor := rec.Header().Get(nombre); !strings.Contains(valor, esperado) {
			t.Errorf("%s = %q; debe contener %q", nombre, valor, esperado)
		}
	}
}

func TestServerLimitaCuerpoDePeticion(t *testing.T) {
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var destino map[string]any
		if err := json.NewDecoder(r.Body).Decode(&destino); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	handler := NewHandlerWithConfig(config.Config{MaxRequestBodyBytes: 16}, api)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, peticionServidorPrueba(http.MethodPost, "/api/dato", strings.NewReader(`{"contenido":"demasiado largo"}`)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("cuerpo excesivo status = %d: %s", rec.Code, rec.Body.String())
	}
}

func TestNewHTTPServerRoutesAPIPrefixWithoutPanic(t *testing.T) {
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/demo" {
			t.Fatalf("api path = %q, want /api/demo", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	srv, err := NewHTTPServer(config.Config{Address: "127.0.0.1:0", ReadHeaderTimeout: time.Second}, api)
	if err != nil {
		t.Fatalf("NewHTTPServer() error = %v", err)
	}
	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, peticionServidorPrueba(http.MethodPost, "/api/demo", nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("/api/demo status = %d, want 204", rec.Code)
	}
}

func TestNewHTTPServerRoutesProfessionalPortalEndpoint(t *testing.T) {
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/portal" {
			t.Fatalf("api path = %q, want /api/portal", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	})
	srv, err := NewHTTPServer(config.Config{Address: "127.0.0.1:0", ReadHeaderTimeout: time.Second}, api)
	if err != nil {
		t.Fatalf("NewHTTPServer() error = %v", err)
	}
	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, peticionServidorPrueba(http.MethodGet, "/api/portal", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("/api/portal status = %d, want 200", rec.Code)
	}
}

func TestHTTPAllowedCIDRsRestrictsRemoteAddress(t *testing.T) {
	api := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	srv, err := NewHTTPServer(config.Config{
		Address:          "127.0.0.1:0",
		HTTPAllowedCIDRs: []string{"10.1.1.91/32"},
	}, api)
	if err != nil {
		t.Fatalf("NewHTTPServer() error = %v", err)
	}
	for _, tc := range []struct {
		name       string
		remoteAddr string
		want       int
	}{
		{name: "allowed proxy", remoteAddr: "10.1.1.91:50100", want: http.StatusOK},
		{name: "loopback no enumerado", remoteAddr: "127.0.0.1:50100", want: http.StatusForbidden},
		{name: "blocked client", remoteAddr: "10.1.1.92:50100", want: http.StatusForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/demo", nil)
			req.RemoteAddr = tc.remoteAddr
			srv.Handler.ServeHTTP(rec, req)
			if rec.Code != tc.want {
				t.Fatalf("status = %d, want %d", rec.Code, tc.want)
			}
		})
	}
}

func TestHTTPAllowedCIDRsVacioNoAbreElServicio(t *testing.T) {
	api := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	handler := restrictRemoteAddrs(api, nil)
	rec := httptest.NewRecorder()
	peticion := peticionServidorPrueba(http.MethodGet, "/api/demo", nil)
	handler.ServeHTTP(rec, peticion)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("lista vacia status = %d; debe denegar", rec.Code)
	}
}

func peticionServidorPrueba(metodo, ruta string, cuerpo io.Reader) *http.Request {
	peticion := httptest.NewRequest(metodo, ruta, cuerpo)
	peticion.RemoteAddr = "127.0.0.1:50000"
	return peticion
}

func TestHTTPAllowedCIDRsRechazaConfiguracionInvalidaSinAbrirAcceso(t *testing.T) {
	api := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	if servidor, err := NewHTTPServer(config.Config{
		Address:          "127.0.0.1:0",
		HTTPAllowedCIDRs: []string{"10.1.1.91/32", "red-invalida"},
	}, api); err == nil || servidor != nil {
		t.Fatalf("NewHTTPServer() = (%v, %v), debe fallar cerrado", servidor, err)
	}

	handler := NewHandlerWithConfig(config.Config{
		HTTPAllowedCIDRs: []string{"red-invalida"},
	}, api)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/demo", nil)
	req.RemoteAddr = "10.1.1.91:50100"
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("configuracion invalida status = %d, debe denegar con 503", rec.Code)
	}
}

func TestAutenticacionFakeSoloPuedeEscucharRedesLoopback(t *testing.T) {
	api := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	for _, redes := range [][]string{
		{"0.0.0.0/0"},
		{"10.20.30.0/24"},
		{"127.0.0.1/32", "::/0"},
	} {
		servidor, err := NewHTTPServer(config.Config{
			Address:          "127.0.0.1:8080",
			AuthMode:         config.AuthModeFake,
			HTTPAllowedCIDRs: redes,
		}, api)
		if err == nil || servidor != nil {
			t.Fatalf("fake abierto en %v: servidor=%v err=%v", redes, servidor, err)
		}
	}

	servidor, err := NewHTTPServer(config.Config{
		Address:          "127.0.0.1:8080",
		AuthMode:         config.AuthModeFake,
		HTTPAllowedCIDRs: []string{"127.0.0.0/8", "::1/128"},
	}, api)
	if err != nil || servidor == nil {
		t.Fatalf("fake local rechazado: servidor=%v err=%v", servidor, err)
	}
}

func TestAutenticacionFakeExigeListenerLoopbackLiteral(t *testing.T) {
	api := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	for _, direccion := range []string{":8080", "0.0.0.0:8080", "[::]:8080", "localhost:8080", "10.20.30.4:8080"} {
		servidor, err := NewHTTPServer(config.Config{
			Address:          direccion,
			AuthMode:         config.AuthModeFake,
			HTTPAllowedCIDRs: []string{"127.0.0.1/32", "::1/128"},
		}, api)
		if err == nil || servidor != nil {
			t.Fatalf("fake acepto listener %q: servidor=%v err=%v", direccion, servidor, err)
		}
	}
	servidorPredeterminado, err := NewHTTPServer(config.Config{
		AuthMode:         config.AuthModeFake,
		HTTPAllowedCIDRs: []string{"127.0.0.1/32", "::1/128"},
	}, api)
	if err != nil || servidorPredeterminado == nil || servidorPredeterminado.Addr != config.DefaultAddress {
		t.Fatalf("fake no uso listener local predeterminado: servidor=%v err=%v", servidorPredeterminado, err)
	}
	for _, direccion := range []string{"127.0.0.1:8080", "[::1]:8080"} {
		servidor, err := NewHTTPServer(config.Config{
			Address:          direccion,
			AuthMode:         config.AuthModeFake,
			HTTPAllowedCIDRs: []string{"127.0.0.1/32", "::1/128"},
		}, api)
		if err != nil || servidor == nil {
			t.Fatalf("fake rechazo listener loopback %q: servidor=%v err=%v", direccion, servidor, err)
		}
	}
}

func TestAutenticacionFakeRechazaCabecerasDeProxy(t *testing.T) {
	api := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	handler := NewHandlerWithConfig(config.Config{
		AuthMode:         config.AuthModeFake,
		HTTPAllowedCIDRs: []string{"127.0.0.1/32"},
	}, api)
	for _, nombre := range []string{"Forwarded", "Via", "X-Forwarded-For", "X-Forwarded-Proto", "X-Forwarded-Tls-Client-Cert"} {
		rec := httptest.NewRecorder()
		req := peticionServidorPrueba(http.MethodGet, "/api/demo", nil)
		req.Header.Set(nombre, "valor-no-confiable")
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("cabecera %s obtuvo %d; se esperaba 400", nombre, rec.Code)
		}
	}
}
