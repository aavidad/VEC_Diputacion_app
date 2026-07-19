package httpcartografia

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/config"
)

func configuracionSuperficiePrueba() config.Config {
	return config.Config{
		Address:                  "127.0.0.1:8080",
		HTTPAllowedCIDRs:         []string{"127.0.0.1/32"},
		ExecutionProfile:         config.ExecutionProfileRRHHPresentation,
		RRHHPresentationEnabled:  true,
		RRHHPresentationGuardOne: config.RRHHPresentationGuardOneAcknowledgement,
		RRHHPresentationGuardTwo: config.RRHHPresentationGuardTwoAcknowledgement,
		AuthMode:                 config.AuthModeDisabled,
		StorageMode:              config.StorageModeMemory,
		PersonalCatalogInMemory:  true,
	}
}

func TestSuperficiePresentacionSoloExponeSaludYRutaExacta(t *testing.T) {
	calculador := &calculadorPrueba{resultado: resultadoRutaPrueba()}
	superficie, err := NuevaSuperficiePresentacion(configuracionSuperficiePrueba(), calculador)
	if err != nil {
		t.Fatal(err)
	}
	pruebas := []struct {
		nombre   string
		metodo   string
		ruta     string
		cuerpo   string
		tipoJSON bool
		estado   int
	}{
		{nombre: "salud", metodo: http.MethodGet, ruta: "/healthz", estado: http.StatusOK},
		{nombre: "ruta", metodo: http.MethodPost, ruta: RutaPresentacion, cuerpo: `{"coordinates":[{"lat":37.1773,"lon":-3.5986},{"lat":37.2306,"lon":-3.6554}]}`, tipoJSON: true, estado: http.StatusOK},
		{nombre: "API VEC no existe", metodo: http.MethodPost, ruta: "/api/vec/dietas/road-route", tipoJSON: true, estado: http.StatusNotFound},
		{nombre: "sufijo no existe", metodo: http.MethodPost, ruta: RutaPresentacion + "/", tipoJSON: true, estado: http.StatusNotFound},
		{nombre: "ruta codificada no existe", metodo: http.MethodPost, ruta: "/api/presentacion/cartografia/%72utas", tipoJSON: true, estado: http.StatusNotFound},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			peticion := httptest.NewRequest(prueba.metodo, prueba.ruta, strings.NewReader(prueba.cuerpo))
			peticion.RemoteAddr = "127.0.0.1:50100"
			if prueba.tipoJSON {
				peticion.Header.Set("Content-Type", "application/json")
			}
			respuesta := httptest.NewRecorder()
			superficie.ServeHTTP(respuesta, peticion)
			if respuesta.Code != prueba.estado {
				t.Fatalf("estado = %d, esperado %d: %s", respuesta.Code, prueba.estado, respuesta.Body.String())
			}
			if respuesta.Header().Get("Cache-Control") != "no-store" ||
				respuesta.Header().Get("Content-Type") != "application/json; charset=utf-8" ||
				len(respuesta.Result().Cookies()) != 0 || respuesta.Header().Get("Access-Control-Allow-Origin") != "" {
				t.Fatalf("cabeceras no seguras: %#v", respuesta.Header())
			}
		})
	}
}

func TestSuperficiePresentacionRechazaComposicionAmbigua(t *testing.T) {
	base := configuracionSuperficiePrueba()
	pruebas := []struct {
		nombre string
		muta   func(*config.Config)
	}{
		{nombre: "sin selector", muta: func(c *config.Config) { c.RRHHPresentationEnabled = false }},
		{nombre: "sin primera guarda", muta: func(c *config.Config) { c.RRHHPresentationGuardOne = "" }},
		{nombre: "con autenticacion", muta: func(c *config.Config) { c.AuthMode = config.AuthModeFake }},
		{nombre: "con almacenamiento", muta: func(c *config.Config) { c.StorageMode = config.StorageModeFile }},
		{nombre: "con credenciales", muta: func(c *config.Config) { c.FakeCredentialsPath = "/run/credenciales.json" }},
		{nombre: "con catalogo durable", muta: func(c *config.Config) {
			c.PersonalCatalogInMemory = false
			c.PersonalCatalogPath = "/data/personal.json"
		}},
		{nombre: "listener universal", muta: func(c *config.Config) { c.Address = ":8080" }},
		{nombre: "listener DNS", muta: func(c *config.Config) { c.Address = "vec-cartografia:8080" }},
		{nombre: "red publica", muta: func(c *config.Config) { c.HTTPAllowedCIDRs = []string{"8.8.8.8/32"} }},
		{nombre: "red universal", muta: func(c *config.Config) { c.HTTPAllowedCIDRs = []string{"0.0.0.0/0"} }},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			cfg := base
			prueba.muta(&cfg)
			if superficie, err := NuevaSuperficiePresentacion(cfg, &calculadorPrueba{}); !errors.Is(err, ErrSuperficiePresentacionInvalida) || superficie != nil {
				t.Fatalf("composicion insegura aceptada: superficie=%v error=%v", superficie, err)
			}
		})
	}
}

func TestSuperficiePresentacionRechazaIdentidadEstadoYProxyAntesDelPuerto(t *testing.T) {
	calculador := &calculadorPrueba{resultado: resultadoRutaPrueba()}
	superficie, err := NuevaSuperficiePresentacion(configuracionSuperficiePrueba(), calculador)
	if err != nil {
		t.Fatal(err)
	}
	pruebas := []struct {
		nombre   string
		cabecera string
		valor    string
		remoto   string
	}{
		{nombre: "cookie", cabecera: "Cookie", valor: "sesion=x", remoto: "127.0.0.1:50100"},
		{nombre: "autorizacion", cabecera: "Authorization", valor: "Bearer x", remoto: "127.0.0.1:50100"},
		{nombre: "proxy", cabecera: "X-Forwarded-For", valor: "127.0.0.1", remoto: "127.0.0.1:50100"},
		{nombre: "principal ambiental", cabecera: "X-Vec-Subject", valor: "persona", remoto: "127.0.0.1:50100"},
		{nombre: "remoto no permitido", remoto: "10.0.0.5:50100"},
		{nombre: "remoto mal formado", remoto: "127.0.0.1"},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			antes := calculador.consultas.Load()
			peticion := peticionRutaPrueba(`{"coordinates":[{"lat":37.1773,"lon":-3.5986},{"lat":37.2306,"lon":-3.6554}]}`)
			peticion.RemoteAddr = prueba.remoto
			if prueba.cabecera != "" {
				peticion.Header.Set(prueba.cabecera, prueba.valor)
			}
			respuesta := httptest.NewRecorder()
			superficie.ServeHTTP(respuesta, peticion)
			if respuesta.Code != http.StatusBadRequest && respuesta.Code != http.StatusForbidden {
				t.Fatalf("estado = %d: %s", respuesta.Code, respuesta.Body.String())
			}
			if calculador.consultas.Load() != antes {
				t.Fatal("una peticion con estado o identidad alcanzo el puerto")
			}
			if len(respuesta.Result().Cookies()) != 0 || respuesta.Header().Get("Access-Control-Allow-Origin") != "" {
				t.Fatalf("respuesta filtro estado ambiental: %#v", respuesta.Header())
			}
		})
	}
}
