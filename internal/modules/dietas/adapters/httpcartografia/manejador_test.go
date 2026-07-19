package httpcartografia

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
)

type calculadorPrueba struct {
	consultas atomic.Int32
	resultado dietasports.ResultadoCalculoRuta
	error     error
}

func (c *calculadorPrueba) Calcular(_ context.Context, _ dietasports.SolicitudCalculoRuta) (dietasports.ResultadoCalculoRuta, error) {
	c.consultas.Add(1)
	return c.resultado, c.error
}

func resultadoRutaPrueba() dietasports.ResultadoCalculoRuta {
	return dietasports.ResultadoCalculoRuta{
		VersionGrafo: "grafo-osm-granada-v1",
		Motor:        "osrm_on_premise",
		Ambito:       "Granada provincia + 15 km",
		Alternativas: []dietasports.AlternativaRuta{{
			DistanciaMetros: 13400, DuracionSegundos: 1080,
			Geometria: dietasports.GeometriaRuta{Tipo: "LineString", Coordenadas: []dietasports.PuntoGeometriaRuta{
				{Longitud: -3.5986, Latitud: 37.1773}, {Longitud: -3.6554, Latitud: 37.2306},
			}},
			Tramos: []dietasports.TramoRuta{{DistanciaMetros: 13400, DuracionSegundos: 1080}},
		}},
	}
}

func peticionRutaPrueba(cuerpo string) *http.Request {
	peticion := httptest.NewRequest(http.MethodPost, RutaPresentacion, strings.NewReader(cuerpo))
	peticion.Header.Set("Content-Type", "application/json; charset=utf-8")
	return peticion
}

func TestManejadorEntregaDTOCartograficoDirectoSinEstadoHTTP(t *testing.T) {
	calculador := &calculadorPrueba{resultado: resultadoRutaPrueba()}
	manejador, err := NuevoManejador(calculador, OpcionesManejador{})
	if err != nil {
		t.Fatal(err)
	}
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, peticionRutaPrueba(`{"coordinates":[{"lat":37.1773,"lon":-3.5986},{"lat":37.2306,"lon":-3.6554}]}`))
	if respuesta.Code != http.StatusOK {
		t.Fatalf("estado = %d: %s", respuesta.Code, respuesta.Body.String())
	}
	contenido := respuesta.Body.String()
	for _, esperado := range []string{`"code":"Ok"`, `"routes":[`, `"data_version":"grafo-osm-granada-v1"`, `"engine":"osrm_on_premise"`, `"route_scope":"Granada provincia + 15 km"`} {
		if !strings.Contains(contenido, esperado) {
			t.Fatalf("falta %s en %s", esperado, contenido)
		}
	}
	if strings.Contains(contenido, `"data":`) {
		t.Fatalf("la superficie aislada no debe añadir la envolvente productiva: %s", contenido)
	}
	if respuesta.Header().Get("Cache-Control") != "no-store" ||
		respuesta.Header().Get("Content-Type") != "application/json; charset=utf-8" ||
		len(respuesta.Result().Cookies()) != 0 || respuesta.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("cabeceras no seguras: %#v", respuesta.Header())
	}
}

func TestManejadorProductivoReutilizaContratoYConservaEnvolvente(t *testing.T) {
	calculador := &calculadorPrueba{resultado: resultadoRutaPrueba()}
	manejador, err := NuevoManejador(calculador, OpcionesManejador{EnvolverEnDatos: true})
	if err != nil {
		t.Fatal(err)
	}
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, peticionRutaPrueba(`{"coordinates":[{"lat":37.1773,"lon":-3.5986},{"lat":37.2306,"lon":-3.6554}]}`))
	if respuesta.Code != http.StatusOK || !strings.Contains(respuesta.Body.String(), `{"data":{"code":"Ok"`) {
		t.Fatalf("envolvente productiva alterada: %d %s", respuesta.Code, respuesta.Body.String())
	}
}

func TestManejadorRechazaHTTPFueraDeContratoAntesDelPuerto(t *testing.T) {
	calculador := &calculadorPrueba{resultado: resultadoRutaPrueba()}
	manejador, err := NuevoManejador(calculador, OpcionesManejador{})
	if err != nil {
		t.Fatal(err)
	}
	pruebas := []struct {
		nombre   string
		peticion func() *http.Request
		esperado int
	}{
		{nombre: "metodo", peticion: func() *http.Request { return httptest.NewRequest(http.MethodGet, RutaPresentacion, nil) }, esperado: http.StatusMethodNotAllowed},
		{nombre: "sin tipo", peticion: func() *http.Request {
			return httptest.NewRequest(http.MethodPost, RutaPresentacion, strings.NewReader(`{}`))
		}, esperado: http.StatusUnsupportedMediaType},
		{nombre: "tipo distinto", peticion: func() *http.Request {
			p := httptest.NewRequest(http.MethodPost, RutaPresentacion, strings.NewReader(`{}`))
			p.Header.Set("Content-Type", "text/plain")
			return p
		}, esperado: http.StatusUnsupportedMediaType},
		{nombre: "parametro de tipo", peticion: func() *http.Request {
			p := peticionRutaPrueba(`{}`)
			p.Header.Set("Content-Type", "application/json; profile=x")
			return p
		}, esperado: http.StatusUnsupportedMediaType},
		{nombre: "consulta", peticion: func() *http.Request {
			p := peticionRutaPrueba(`{}`)
			p.URL.RawQuery = "x=1"
			return p
		}, esperado: http.StatusBadRequest},
		{nombre: "campo desconocido", peticion: func() *http.Request { return peticionRutaPrueba(`{"desconocido":true}`) }, esperado: http.StatusBadRequest},
		{nombre: "segundo JSON", peticion: func() *http.Request { return peticionRutaPrueba(`{} {}`) }, esperado: http.StatusBadRequest},
		{nombre: "no UTF-8", peticion: func() *http.Request { return peticionRutaPrueba(string([]byte{0xff})) }, esperado: http.StatusBadRequest},
		{nombre: "demasiado grande", peticion: func() *http.Request {
			return peticionRutaPrueba(`{"coordinates":"` + strings.Repeat("x", int(maximoCuerpo)) + `"}`)
		}, esperado: http.StatusRequestEntityTooLarge},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			antes := calculador.consultas.Load()
			respuesta := httptest.NewRecorder()
			manejador.ServeHTTP(respuesta, prueba.peticion())
			if respuesta.Code != prueba.esperado {
				t.Fatalf("estado = %d, esperado %d: %s", respuesta.Code, prueba.esperado, respuesta.Body.String())
			}
			if calculador.consultas.Load() != antes {
				t.Fatal("una peticion HTTP invalida alcanzo el puerto")
			}
		})
	}
}

func TestManejadorDistingueEntradaMotorYRespuestaNoValida(t *testing.T) {
	for _, prueba := range []struct {
		nombre string
		error  error
		estado int
	}{
		{nombre: "entrada", error: dietasports.ErrSolicitudRutaInvalida, estado: http.StatusBadRequest},
		{nombre: "motor", error: dietasports.ErrMotorRutasNoDisponible, estado: http.StatusBadGateway},
		{nombre: "respuesta", error: dietasports.ErrRespuestaMotorRutasInvalida, estado: http.StatusBadGateway},
	} {
		t.Run(prueba.nombre, func(t *testing.T) {
			calculador := &calculadorPrueba{error: errors.Join(prueba.error, errors.New("detalle"))}
			manejador, err := NuevoManejador(calculador, OpcionesManejador{})
			if err != nil {
				t.Fatal(err)
			}
			respuesta := httptest.NewRecorder()
			manejador.ServeHTTP(respuesta, peticionRutaPrueba(`{"coordinates":[{"lat":37.1773,"lon":-3.5986},{"lat":37.2306,"lon":-3.6554}]}`))
			if respuesta.Code != prueba.estado {
				t.Fatalf("estado = %d: %s", respuesta.Code, respuesta.Body.String())
			}
		})
	}
}
