package bootstrap

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/config"
	httpcartografia "vec-diputacion-granada/internal/modules/dietas/adapters/httpcartografia"
)

const versionGrafoCartografiaPrueba = "grafo-osm-granada-prueba-v1"

func configuracionCartografiaBootstrap(urlOSRM string) config.Config {
	return config.Config{
		Address:                  "127.0.0.1:8081",
		HTTPAllowedCIDRs:         []string{"127.0.0.1/32"},
		ExecutionProfile:         config.ExecutionProfileRRHHPresentation,
		RRHHPresentationEnabled:  true,
		RRHHPresentationGuardOne: config.RRHHPresentationGuardOneAcknowledgement,
		RRHHPresentationGuardTwo: config.RRHHPresentationGuardTwoAcknowledgement,
		AuthMode:                 config.AuthModeDisabled,
		StorageMode:              config.StorageModeMemory,
		PersonalCatalogInMemory:  true,
		OSRMBaseURL:              urlOSRM,
		OSRMScopeName:            "Granada provincia + 15 km",
		OSRMScopeBounds:          "36.45,-4.6,38.25,-2.15",
		OSRMAllowedCIDRs:         []string{"127.0.0.1/32"},
		OSRMGraphVersion:         versionGrafoCartografiaPrueba,
	}
}

func TestComposicionCartograficaSirveRutaRealSinComponerVEC(t *testing.T) {
	osrm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":"Ok","data_version":null,"routes":[{"distance":13400,"duration":1080,"geometry":{"type":"LineString","coordinates":[[-3.5986,37.1773],[-3.6554,37.2306]]},"legs":[{"distance":13400,"duration":1080}]}]}`))
	}))
	defer osrm.Close()
	servidor, err := NewHTTPServerCartografiaPresentacionWithConfig(configuracionCartografiaBootstrap(osrm.URL))
	if err != nil {
		t.Fatal(err)
	}

	peticion := httptest.NewRequest(http.MethodPost, httpcartografia.RutaPresentacion,
		strings.NewReader(`{"coordinates":[{"lat":37.1773,"lon":-3.5986},{"lat":37.2306,"lon":-3.6554}]}`))
	peticion.Header.Set("Content-Type", "application/json")
	peticion.RemoteAddr = "127.0.0.1:50100"
	respuesta := httptest.NewRecorder()
	servidor.Handler.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusOK || !strings.Contains(respuesta.Body.String(), `"data_version":"`+versionGrafoCartografiaPrueba+`"`) {
		t.Fatalf("ruta cartografica = %d %s", respuesta.Code, respuesta.Body.String())
	}

	peticion = httptest.NewRequest(http.MethodGet, "/api/vec/session", nil)
	peticion.RemoteAddr = "127.0.0.1:50100"
	respuesta = httptest.NewRecorder()
	servidor.Handler.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusNotFound {
		t.Fatalf("la composicion cartografica expuso VEC: %d %s", respuesta.Code, respuesta.Body.String())
	}
}

func TestComposicionCartograficaRechazaConcesionParcial(t *testing.T) {
	base := configuracionCartografiaBootstrap("http://127.0.0.1:5000")
	pruebas := []struct {
		nombre string
		muta   func(*config.Config)
	}{
		{nombre: "sin version", muta: func(c *config.Config) { c.OSRMGraphVersion = "" }},
		{nombre: "sin URL", muta: func(c *config.Config) { c.OSRMBaseURL = "" }},
		{nombre: "sin red OSRM", muta: func(c *config.Config) { c.OSRMAllowedCIDRs = nil }},
		{nombre: "sin guarda", muta: func(c *config.Config) { c.RRHHPresentationGuardTwo = "" }},
		{nombre: "con identidad", muta: func(c *config.Config) { c.AuthMode = config.AuthModeFake }},
		{nombre: "con almacenamiento", muta: func(c *config.Config) { c.StorageMode = config.StorageModeFile }},
		{nombre: "listener no privado", muta: func(c *config.Config) { c.Address = ":8081" }},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			cfg := base
			prueba.muta(&cfg)
			if servidor, err := NewHTTPServerCartografiaPresentacionWithConfig(cfg); !errors.Is(err, ErrComposicionCartografiaPresentacionInvalida) || servidor != nil {
				t.Fatalf("composicion parcial aceptada: servidor=%v error=%v", servidor, err)
			}
		})
	}
}
