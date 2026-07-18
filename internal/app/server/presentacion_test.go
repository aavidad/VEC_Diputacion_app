package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/config"
)

func configuracionPresentacionValida() config.Config {
	return config.Config{
		Address:                  "127.0.0.1:0",
		AuthMode:                 config.AuthModeDisabled,
		ExecutionProfile:         config.ExecutionProfileRRHHPresentation,
		RRHHPresentationEnabled:  true,
		RRHHPresentationGuardOne: config.RRHHPresentationGuardOneAcknowledgement,
		RRHHPresentationGuardTwo: config.RRHHPresentationGuardTwoAcknowledgement,
		HTTPAllowedCIDRs:         []string{"127.0.0.1/32", "::1/128"},
	}
}

func TestServidorPresentacionExigeDosGuardasListenerYRedLocal(t *testing.T) {
	valida := configuracionPresentacionValida()
	if _, err := NewHTTPServerPresentacion(valida, http.NotFoundHandler()); err != nil {
		t.Fatalf("configuracion valida: %v", err)
	}
	pruebas := []struct {
		nombre string
		muta   func(*config.Config)
	}{
		{"sin primera guarda", func(c *config.Config) { c.RRHHPresentationGuardOne = "" }},
		{"sin segunda guarda", func(c *config.Config) { c.RRHHPresentationGuardTwo = "" }},
		{"listener comodin", func(c *config.Config) { c.Address = ":8080" }},
		{"listener publico", func(c *config.Config) { c.Address = "8.8.8.8:8080" }},
		{"red global", func(c *config.Config) { c.HTTPAllowedCIDRs = []string{"0.0.0.0/0"} }},
		{"red publica", func(c *config.Config) { c.HTTPAllowedCIDRs = []string{"8.8.8.8/32"} }},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			cfg := valida
			prueba.muta(&cfg)
			if _, err := NewHTTPServerPresentacion(cfg, http.NotFoundHandler()); err == nil {
				t.Fatal("la configuracion insegura fue aceptada")
			}
		})
	}
}

func TestPresentacionSoloExponeSuperficiesEnumeradas(t *testing.T) {
	api := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"esquema":"presentacion.publica.v1"}`))
	})
	servidor, err := NewHTTPServerPresentacion(configuracionPresentacionValida(), api)
	if err != nil {
		t.Fatal(err)
	}
	permitidas := []struct {
		ruta      string
		contenido string
	}{
		{"/presentacion/", "Recorrido de presentación"},
		{"/area-personal/", "Mi área personal"},
		{"/area-personal/adaptador-presentacion.js", "Adaptador efímero y exclusivo de presentación"},
		{"/portal-empleado/", "Portal del Empleado"},
		{"/portal-empleado/datos-presentacion.js", "ADAPTADOR EXCLUSIVO DE PRESENTACIÓN RRHH"},
		{"/portal-empleado/portal-presentacion-adaptador.js", "Adaptador volátil y sustituible"},
		{"/bolsa/", "Bolsa"},
		{"/api/publico/bolsa/convocatorias", "presentacion.publica.v1"},
		{"/healthz", `"status":"ok"`},
	}
	for _, prueba := range permitidas {
		rec := httptest.NewRecorder()
		servidor.Handler.ServeHTTP(rec, peticionServidorPrueba(http.MethodGet, prueba.ruta, nil))
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), prueba.contenido) {
			t.Errorf("GET %s = %d %q", prueba.ruta, rec.Code, rec.Body.String())
		}
		for _, cabecera := range []string{"Content-Security-Policy", "X-Content-Type-Options", "Referrer-Policy"} {
			if rec.Header().Get(cabecera) == "" {
				t.Errorf("GET %s sin %s", prueba.ruta, cabecera)
			}
		}
		if rec.Header().Get("Set-Cookie") != "" {
			t.Errorf("GET %s emitio cookie", prueba.ruta)
		}
	}

	for _, ruta := range []string{
		"/app.js", "/config/config.go", "/data/demo/convocatorias_publicas.demo.json",
		"/api/vec/session", "/api/demo", "/candidates", "/desconocido",
	} {
		rec := httptest.NewRecorder()
		servidor.Handler.ServeHTTP(rec, peticionServidorPrueba(http.MethodGet, ruta, nil))
		if rec.Code != http.StatusNotFound {
			t.Errorf("GET %s = %d; se esperaba 404", ruta, rec.Code)
		}
	}
}

func TestLauncherPresentacionEnlazaLosTresPuntosDeVistaSinEscaparAllowlist(t *testing.T) {
	servidor, err := NewHTTPServerPresentacion(configuracionPresentacionValida(), http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	if err != nil {
		t.Fatal(err)
	}
	recRaiz := httptest.NewRecorder()
	servidor.Handler.ServeHTTP(recRaiz, peticionServidorPrueba(http.MethodGet, "/", nil))
	if recRaiz.Code != http.StatusMovedPermanently || recRaiz.Header().Get("Location") != "/presentacion/" {
		t.Fatalf("raiz = %d Location=%q", recRaiz.Code, recRaiz.Header().Get("Location"))
	}
	rec := httptest.NewRecorder()
	servidor.Handler.ServeHTTP(rec, peticionServidorPrueba(http.MethodGet, "/presentacion/", nil))
	for _, enlace := range []string{
		`href="/bolsa/"`,
		`href="/area-personal/?presentacion=rrhh"`,
		`href="/portal-empleado/?presentacion=rrhh#portal"`,
	} {
		if !strings.Contains(rec.Body.String(), enlace) {
			t.Errorf("launcher sin %s", enlace)
		}
	}

	recAPI := httptest.NewRecorder()
	servidor.Handler.ServeHTTP(recAPI, peticionServidorPrueba(http.MethodGet, "/api/publico", nil))
	if recAPI.Code != http.StatusNoContent || recAPI.Header().Get("Location") != "" {
		t.Errorf("/api/publico = %d Location=%q", recAPI.Code, recAPI.Header().Get("Location"))
	}
	for _, ruta := range []string{
		"/presentacion/../app.js", "/area-personal/../portal-empleado/", "/api/publico/../vec/session",
	} {
		respuesta := httptest.NewRecorder()
		servidor.Handler.ServeHTTP(respuesta, peticionServidorPrueba(http.MethodGet, ruta, nil))
		if respuesta.Code != http.StatusNotFound || respuesta.Header().Get("Location") != "" {
			t.Errorf("ruta no canonica %s = %d Location=%q", ruta, respuesta.Code, respuesta.Header().Get("Location"))
		}
	}
}

func TestPresentacionHEADNoEntregaCuerpo(t *testing.T) {
	servidor, err := NewHTTPServerPresentacion(configuracionPresentacionValida(), http.NotFoundHandler())
	if err != nil {
		t.Fatal(err)
	}
	for _, ruta := range []string{"/presentacion/", "/area-personal/", "/portal-empleado/", "/bolsa/", "/healthz"} {
		rec := httptest.NewRecorder()
		servidor.Handler.ServeHTTP(rec, peticionServidorPrueba(http.MethodHead, ruta, nil))
		if rec.Code != http.StatusOK || rec.Body.Len() != 0 {
			t.Errorf("HEAD %s = %d cuerpo=%q", ruta, rec.Code, rec.Body.String())
		}
	}
}

func TestPresentacionEsSoloLecturaYRechazaCredencialesAmbientales(t *testing.T) {
	servidor, err := NewHTTPServerPresentacion(configuracionPresentacionValida(), http.NotFoundHandler())
	if err != nil {
		t.Fatal(err)
	}
	for _, ruta := range []string{"/presentacion/", "/area-personal/", "/portal-empleado/", "/bolsa/", "/api/publico/consulta"} {
		rec := httptest.NewRecorder()
		servidor.Handler.ServeHTTP(rec, peticionServidorPrueba(http.MethodPost, ruta, strings.NewReader("dato")))
		if rec.Code != http.StatusMethodNotAllowed || rec.Header().Get("Allow") != "GET, HEAD" {
			t.Errorf("POST %s = %d Allow=%q", ruta, rec.Code, rec.Header().Get("Allow"))
		}
	}
	for _, cabecera := range []string{"Cookie", "Authorization", "Proxy-Authorization", "X-VEC-Subject", "X-Forwarded-For"} {
		peticion := peticionServidorPrueba(http.MethodGet, "/presentacion/", nil)
		peticion.Header[cabecera] = []string{"valor"}
		rec := httptest.NewRecorder()
		servidor.Handler.ServeHTTP(rec, peticion)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("cabecera %s = %d; se esperaba 400", cabecera, rec.Code)
		}
	}
}

func TestHandlerPresentacionDirectoFallaCerradoSinGuardas(t *testing.T) {
	configuraciones := []config.Config{{}}
	redGlobal := configuracionPresentacionValida()
	redGlobal.HTTPAllowedCIDRs = []string{"0.0.0.0/0"}
	configuraciones = append(configuraciones, redGlobal)
	listenerGlobal := configuracionPresentacionValida()
	listenerGlobal.Address = ":8080"
	configuraciones = append(configuraciones, listenerGlobal)
	for _, cfg := range configuraciones {
		handler := NewHandlerPresentacionWithConfig(cfg, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			t.Fatal("la API no debe recibir peticiones")
		}))
		for _, ruta := range []string{"/", "/presentacion/", "/portal-empleado/datos-presentacion.js", "/api/publico/consulta"} {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, peticionServidorPrueba(http.MethodGet, ruta, nil))
			if rec.Code != http.StatusServiceUnavailable {
				t.Errorf("GET %s = %d; se esperaba 503", ruta, rec.Code)
			}
		}
	}
}
