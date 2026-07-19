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
		{path: "/modulos/cronos/resumen.js", contentType: "text/javascript", want: "renderizarResumenCronos"},
		{path: "/modulos/cronos/resumen.css", contentType: "text/css", want: ".cronos-table-wrap"},
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

func TestInterfazParteCerradaHastaResolverUnaSesionAutorizada(t *testing.T) {
	handler := NewHandler(http.NotFoundHandler())
	for _, recurso := range []struct {
		ruta    string
		cierres []string
	}{
		{ruta: "/", cierres: []string{
			`class="module-group" aria-label="Navegacion principal" hidden`,
			`class="search-form" role="search" hidden`,
			`class="filter-bar" aria-label="Filtros de expedientes" hidden`,
			`class="queue-panel" aria-labelledby="queue-title" hidden`,
			`class="right-column" aria-label="Detalle del expediente seleccionado" hidden`,
		}},
		{ruta: "/app.js", cierres: []string{
			`activeModule: "sin_acceso"`,
			`renderizarAccesoCerrado();`,
		}},
	} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, peticionServidorPrueba(http.MethodGet, recurso.ruta, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status = %d, want 200", recurso.ruta, rec.Code)
		}
		for _, cierre := range recurso.cierres {
			if !strings.Contains(rec.Body.String(), cierre) {
				t.Errorf("%s no conserva el cierre inicial %q", recurso.ruta, cierre)
			}
		}
	}
}

func TestServerSirvePortalBolsaPermanenteSinEstilosInline(t *testing.T) {
	handler := NewHandler(http.NotFoundHandler())
	for _, prueba := range []struct{ ruta, tipo, contenido string }{
		{ruta: "/bolsa/", tipo: "text/html", contenido: "menu-lateral-publico"},
		{ruta: "/bolsa/bolsa.css?v=1", tipo: "text/css", contenido: ".grupos-directorio"},
		{ruta: "/bolsa/bolsa.js?v=1", tipo: "text/javascript", contenido: "/api/publico/bolsa/categorias"},
		{ruta: "/bolsa/favicon.svg", tipo: "image/svg+xml", contenido: "<svg"},
	} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, peticionServidorPrueba(http.MethodGet, prueba.ruta, nil))
		if rec.Code != http.StatusOK || !strings.Contains(rec.Header().Get("Content-Type"), prueba.tipo) || !strings.Contains(rec.Body.String(), prueba.contenido) {
			t.Fatalf("%s = %d %q", prueba.ruta, rec.Code, rec.Body.String())
		}
		if prueba.tipo == "text/html" && (strings.Contains(strings.ToLower(rec.Body.String()), "<style") || strings.Contains(strings.ToLower(rec.Body.String()), " style=")) {
			t.Fatalf("%s contiene CSS inline", prueba.ruta)
		}
		if prueba.ruta == "/bolsa/bolsa.js?v=1" && !strings.Contains(rec.Body.String(), "fuente?.demostracion === true") {
			t.Fatal("la UI no gobierna el aviso DEMOSTRACIÓN desde la fuente")
		}
		if prueba.ruta == "/bolsa/" {
			contenido := rec.Body.String()
			inicioMenu := strings.Index(contenido, `<aside class="menu-lateral-publico"`)
			if inicioMenu < 0 {
				t.Fatal("la bolsa pública servida no contiene el menú lateral")
			}
			finMenu := strings.Index(contenido[inicioMenu:], "</aside>")
			if finMenu < 0 {
				t.Fatal("el menú lateral público servido no está cerrado")
			}
			menu := contenido[inicioMenu : inicioMenu+finMenu]
			for _, destinoPublico := range []string{"#contenido-principal", "#filtros-convocatorias", "#directorio-categorias", "#ayuda-publica"} {
				if !strings.Contains(menu, `href="`+destinoPublico+`"`) {
					t.Errorf("el menú público servido no contiene el destino %q", destinoPublico)
				}
			}
			for _, accesoInterno := range []string{"Cronos", "Nóminas", "Dietas", "Administración", "Auditoría"} {
				if strings.Contains(menu, accesoInterno) {
					t.Errorf("el menú público servido expone el acceso interno %q", accesoInterno)
				}
			}
		}
	}
}

func TestServerNormalizaEntradaBolsaSinBarraFinal(t *testing.T) {
	handler := NewHandler(http.NotFoundHandler())
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, peticionServidorPrueba(http.MethodGet, "/bolsa", nil))

	if rec.Code != http.StatusMovedPermanently {
		t.Fatalf("/bolsa status = %d, want 301", rec.Code)
	}
	if destino := rec.Header().Get("Location"); destino != "bolsa/" {
		t.Fatalf("/bolsa location = %q, want %q", destino, "bolsa/")
	}
}

func TestServerSirvePortalEmpleadoRRHHConPresentacionAislada(t *testing.T) {
	handler := NewHandlerPresentacionWithConfig(configuracionPresentacionValida(), http.NotFoundHandler())
	for _, prueba := range []struct{ ruta, tipo, contenido string }{
		{ruta: "/portal-empleado/", tipo: "text/html", contenido: "Portal del Empleado"},
		{ruta: "/portal-empleado/portal.css?v=1", tipo: "text/css", contenido: ".portal-empleado-shell"},
		{ruta: "/portal-empleado/portal-componentes.css?v=1", tipo: "text/css", contenido: ".tarjeta-modulo"},
		{ruta: "/portal-empleado/portal-flujos.css?v=1", tipo: "text/css", contenido: ".barra-filtros"},
		{ruta: "/portal-empleado/portal.js?v=1", tipo: "text/javascript", contenido: `const API_PANEL_BOLSA = "/api/vec/bolsa/panel"`},
		{ruta: "/portal-empleado/portal-eventos.js?v=1", tipo: "text/javascript", contenido: "crearControladorPortal"},
		{ruta: "/portal-empleado/datos-presentacion.js?v=1", tipo: "text/javascript", contenido: "ADAPTADOR EXCLUSIVO DE PRESENTACIÓN RRHH"},
		{ruta: "/verificar/", tipo: "text/html", contenido: "Comprobación de documentos"},
		{ruta: "/verificar/adaptador-presentacion.js?v=1", tipo: "text/javascript", contenido: "Adaptador local y no autoritativo"},
	} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, peticionServidorPrueba(http.MethodGet, prueba.ruta, nil))
		if rec.Code != http.StatusOK || !strings.Contains(rec.Header().Get("Content-Type"), prueba.tipo) || !strings.Contains(rec.Body.String(), prueba.contenido) {
			t.Fatalf("%s = %d %q", prueba.ruta, rec.Code, rec.Body.String())
		}
		if prueba.tipo == "text/html" && (strings.Contains(strings.ToLower(rec.Body.String()), "<style") || strings.Contains(strings.ToLower(rec.Body.String()), " style=")) {
			t.Fatalf("%s contiene CSS inline", prueba.ruta)
		}
	}
}

func TestAdaptadorPresentacionRRHHNoSeSirvePorDefecto(t *testing.T) {
	for _, handler := range []http.Handler{
		NewHandler(http.NotFoundHandler()),
		NewHandlerWithConfig(config.Config{RRHHPresentationEnabled: true}, http.NotFoundHandler()),
		NewHandlerWithConfig(configuracionPresentacionValida(), http.NotFoundHandler()),
		NewHandlerPublicoWithConfig(configuracionPresentacionValida(), http.NotFoundHandler()),
		NewHandlerInternoWithConfig(configuracionPresentacionValida(), http.NotFoundHandler()),
	} {
		for _, ruta := range []string{
			"/presentacion/",
			"/area-personal/adaptador-presentacion.js",
			"/portal-empleado/datos-presentacion.js?v=1",
			"/portal-empleado/portal-presentacion-adaptador.js",
			"/verificar/adaptador-presentacion.js",
			"/bolsa/documentos/bases-demo.css",
			"/bolsa/documentos/bases-auxiliar-demo.html",
			"/bolsa/documentos/bases-auxiliar-demo.pdf",
			"/bolsa/documentos/bases-gestion-demo.html",
			"/bolsa/documentos/bases-gestion-demo.pdf",
			"/bolsa/documentos/bases-operario-demo.html",
			"/bolsa/documentos/bases-operario-demo.pdf",
		} {
			for _, metodo := range []string{http.MethodGet, http.MethodHead} {
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, peticionServidorPrueba(metodo, ruta, nil))
				if rec.Code != http.StatusNotFound {
					t.Fatalf("%s %s = %d; se esperaba 404", metodo, ruta, rec.Code)
				}
			}
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
		{path: "/modulos/cronos/resumen.js?v=20260716", wantCache: "public, max-age=31536000, immutable"},
		{path: "/modulos/cronos/resumen.css?v=20260716", wantCache: "public, max-age=31536000, immutable"},
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

func TestSuperficiePublicaExponeSoloSuListaPositiva(t *testing.T) {
	llamadasAPI := 0
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		llamadasAPI++
		if !strings.HasPrefix(r.URL.Path, "/api/publico") {
			t.Fatalf("la API publica recibio una ruta ajena: %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusAccepted)
	})
	handler := NewHandlerPublicoWithConfig(config.Config{}, api)

	for _, prueba := range []struct {
		metodo string
		ruta   string
		estado int
	}{
		{metodo: http.MethodGet, ruta: "/healthz", estado: http.StatusOK},
		{metodo: http.MethodHead, ruta: "/healthz", estado: http.StatusOK},
		{metodo: http.MethodGet, ruta: "/bolsa", estado: http.StatusMovedPermanently},
		{metodo: http.MethodGet, ruta: "/bolsa/", estado: http.StatusOK},
		{metodo: http.MethodGet, ruta: "/bolsa/bolsa.js?v=1", estado: http.StatusOK},
		{metodo: http.MethodGet, ruta: "/verificar", estado: http.StatusMovedPermanently},
		{metodo: http.MethodGet, ruta: "/verificar/", estado: http.StatusOK},
		{metodo: http.MethodGet, ruta: "/verificar/verificar.js?v=1", estado: http.StatusOK},
		{metodo: http.MethodHead, ruta: "/styles.css", estado: http.StatusOK},
		{metodo: http.MethodGet, ruta: "/portal-empleado/assets/logo-diputacion-granada.svg", estado: http.StatusOK},
		{metodo: http.MethodPost, ruta: "/api/publico/consulta", estado: http.StatusAccepted},
	} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, peticionServidorPrueba(prueba.metodo, prueba.ruta, nil))
		if rec.Code != prueba.estado {
			t.Errorf("%s %s = %d; se esperaba %d", prueba.metodo, prueba.ruta, rec.Code, prueba.estado)
		}
	}

	for _, ruta := range []string{
		"/", "/app.js", "/favicon.svg", "/locales/es.json",
		"/portal-empleado", "/portal-empleado/", "/portal-empleado/portal.js",
		"/portal-empleado/assets/", "/portal-empleado/assets/ayuda-llamamiento-bolsa.mp3",
		"/api", "/api/vec", "/api/vec/session", "/api/publicox", "/bolsax",
	} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, peticionServidorPrueba(http.MethodGet, ruta, nil))
		if rec.Code != http.StatusNotFound {
			t.Errorf("la superficie publica expuso %s con estado %d", ruta, rec.Code)
		}
	}
	if llamadasAPI != 1 {
		t.Fatalf("llamadas a API publica = %d; se esperaba solo la ruta permitida", llamadasAPI)
	}
}

func TestSuperficieInternaExponeSoloSuListaPositiva(t *testing.T) {
	llamadasAPI := 0
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		llamadasAPI++
		if !strings.HasPrefix(r.URL.Path, "/api/vec") {
			t.Fatalf("la API interna recibio una ruta ajena: %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusAccepted)
	})
	handler := NewHandlerInternoWithConfig(config.Config{}, api)

	for _, prueba := range []struct {
		metodo string
		ruta   string
		estado int
	}{
		{metodo: http.MethodGet, ruta: "/healthz", estado: http.StatusOK},
		{metodo: http.MethodGet, ruta: "/portal-empleado", estado: http.StatusMovedPermanently},
		{metodo: http.MethodGet, ruta: "/portal-empleado/", estado: http.StatusOK},
		{metodo: http.MethodHead, ruta: "/portal-empleado/portal.css?v=1", estado: http.StatusOK},
		{metodo: http.MethodGet, ruta: "/locales/es.json", estado: http.StatusOK},
		{metodo: http.MethodPost, ruta: "/api/vec/bolsa/panel", estado: http.StatusAccepted},
	} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, peticionServidorPrueba(prueba.metodo, prueba.ruta, nil))
		if rec.Code != prueba.estado {
			t.Errorf("%s %s = %d; se esperaba %d", prueba.metodo, prueba.ruta, rec.Code, prueba.estado)
		}
	}

	for _, ruta := range []string{
		"/", "/app.js", "/styles.css", "/favicon.svg",
		"/bolsa", "/bolsa/", "/bolsa/bolsa.js", "/api", "/api/publico", "/api/publico/bolsa",
		"/api/vecino",
	} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, peticionServidorPrueba(http.MethodGet, ruta, nil))
		if rec.Code != http.StatusNotFound {
			t.Errorf("la superficie interna expuso %s con estado %d", ruta, rec.Code)
		}
	}
	if llamadasAPI != 1 {
		t.Fatalf("llamadas a API interna = %d; se esperaba solo la ruta permitida", llamadasAPI)
	}
}

func TestSuperficiesRechazanMetodosDeEscrituraEnRecursosEstaticos(t *testing.T) {
	pruebas := []struct {
		nombre  string
		handler http.Handler
		rutas   []string
	}{
		{
			nombre:  "publica",
			handler: NewHandlerPublicoWithConfig(config.Config{}, http.NotFoundHandler()),
			rutas:   []string{"/healthz", "/bolsa", "/bolsa/bolsa.js", "/verificar", "/verificar/verificar.js", "/styles.css", "/portal-empleado/assets/logo-diputacion-granada.svg"},
		},
		{
			nombre:  "interna",
			handler: NewHandlerInternoWithConfig(config.Config{}, http.NotFoundHandler()),
			rutas:   []string{"/healthz", "/portal-empleado", "/portal-empleado/portal.js"},
		},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			for _, ruta := range prueba.rutas {
				rec := httptest.NewRecorder()
				prueba.handler.ServeHTTP(rec, peticionServidorPrueba(http.MethodPost, ruta, strings.NewReader("contenido")))
				if rec.Code != http.StatusMethodNotAllowed || rec.Header().Get("Allow") != "GET, HEAD" {
					t.Errorf("POST %s = %d Allow=%q; se esperaba 405 y GET, HEAD", ruta, rec.Code, rec.Header().Get("Allow"))
				}
			}
		})
	}
}

func TestSuperficieInternaNoAceptaNiEmiteCookies(t *testing.T) {
	llamadasAPI := 0
	api := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		llamadasAPI++
		w.Header().Add("Set-Cookie", "sesion=no-debe-salir; HttpOnly")
		w.Header().Add("Set-Cookie", "otra=tampoco")
		w.WriteHeader(http.StatusNoContent)
	})
	handler := NewHandlerInternoWithConfig(config.Config{}, api)

	for _, nombre := range []string{"Cookie", "Proxy-Authorization"} {
		rec := httptest.NewRecorder()
		req := peticionServidorPrueba(http.MethodGet, "/api/vec/session", nil)
		// Una cabecera presente incluso sin valor debe cerrar la peticion.
		req.Header[nombre] = []string{""}
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("cabecera %s obtuvo %d; se esperaba 400", nombre, rec.Code)
		}
	}
	if llamadasAPI != 0 {
		t.Fatalf("la API se ejecuto %d veces con credenciales prohibidas", llamadasAPI)
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, peticionServidorPrueba(http.MethodGet, "/api/vec/session", nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("peticion interna sin cookies = %d; se esperaba 204", rec.Code)
	}
	if cookies := rec.Header().Values("Set-Cookie"); len(cookies) != 0 {
		t.Fatalf("la superficie interna emitio cookies: %v", cookies)
	}
}

func TestSuperficiesNoRedirigenRutasNoCanonicasEntreZonas(t *testing.T) {
	pruebas := []struct {
		handler http.Handler
		rutas   []string
	}{
		{
			handler: NewHandlerPublicoWithConfig(config.Config{}, http.NotFoundHandler()),
			rutas: []string{
				"/bolsa//documento", "/bolsa/../portal-empleado/", "/api/publico/../vec/session",
			},
		},
		{
			handler: NewHandlerInternoWithConfig(config.Config{}, http.NotFoundHandler()),
			rutas: []string{
				"/portal-empleado//portal.js", "/portal-empleado/../bolsa/", "/api/vec/../publico/bolsa",
			},
		},
	}
	for _, prueba := range pruebas {
		for _, ruta := range prueba.rutas {
			rec := httptest.NewRecorder()
			prueba.handler.ServeHTTP(rec, peticionServidorPrueba(http.MethodGet, ruta, nil))
			if rec.Code != http.StatusNotFound || rec.Header().Get("Location") != "" {
				t.Errorf("ruta no canonica %q = %d Location=%q; se esperaba 404 sin redireccion", ruta, rec.Code, rec.Header().Get("Location"))
			}
		}
	}
}

func TestNuevosServidoresConservanLimitesYListaDeRed(t *testing.T) {
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	for _, constructor := range []struct {
		nombre string
		nuevo  func(config.Config, http.Handler) (*http.Server, error)
		ruta   string
	}{
		{nombre: "publico", nuevo: NewHTTPServerPublico, ruta: "/api/publico/dato"},
		{nombre: "interno", nuevo: NewHTTPServerInterno, ruta: "/api/vec/dato"},
	} {
		t.Run(constructor.nombre, func(t *testing.T) {
			servidor, err := constructor.nuevo(config.Config{
				Address:             "127.0.0.1:0",
				HTTPAllowedCIDRs:    []string{"10.8.0.4/32"},
				MaxRequestBodyBytes: 4,
			}, api)
			if err != nil {
				t.Fatalf("constructor: %v", err)
			}

			bloqueada := httptest.NewRequest(http.MethodPost, constructor.ruta, strings.NewReader("12345"))
			bloqueada.RemoteAddr = "127.0.0.1:50000"
			recBloqueada := httptest.NewRecorder()
			servidor.Handler.ServeHTTP(recBloqueada, bloqueada)
			if recBloqueada.Code != http.StatusForbidden {
				t.Fatalf("red no enumerada = %d; se esperaba 403", recBloqueada.Code)
			}

			permitida := httptest.NewRequest(http.MethodPost, constructor.ruta, strings.NewReader("12345"))
			permitida.RemoteAddr = "10.8.0.4:50000"
			recPermitida := httptest.NewRecorder()
			servidor.Handler.ServeHTTP(recPermitida, permitida)
			if recPermitida.Code != http.StatusRequestEntityTooLarge {
				t.Fatalf("cuerpo por encima del limite = %d; se esperaba 413", recPermitida.Code)
			}
			if recPermitida.Header().Get("X-Frame-Options") != "DENY" {
				t.Fatalf("la superficie no aplico cabeceras de seguridad")
			}
		})
	}
}
