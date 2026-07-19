package osrm

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
)

const versionGrafoPrueba = "grafo-osm-granada-2026-07-19T04:00:00Z"

func configuracionPrueba(urlBase string) Configuracion {
	return Configuracion{
		URLBase:        urlBase,
		NombreAmbito:   "Granada provincia + 15 km",
		LimitesAmbito:  "36.45,-4.6,38.25,-2.15",
		CIDRPermitidas: []string{"127.0.0.1/32"},
		VersionGrafo:   versionGrafoPrueba,
	}
}

func solicitudPrueba() dietasports.SolicitudCalculoRuta {
	return dietasports.SolicitudCalculoRuta{Coordenadas: []dietasports.CoordenadaRuta{
		{Latitud: 37.1773, Longitud: -3.5986, Nombre: "Granada"},
		{Latitud: 37.2306, Longitud: -3.6554, Nombre: "Albolote"},
	}}
}

func respuestaValida(versionJSON string) string {
	return fmt.Sprintf(`{"code":"Ok","data_version":%s,"routes":[{"distance":13400,"duration":1080,"geometry":{"type":"LineString","coordinates":[[-3.5986,37.1773],[-3.62,37.205],[-3.6554,37.2306]]},"legs":[{"distance":13400,"duration":1080}]}]}`, versionJSON)
}

func TestCalculadorImponeVersionGobernadaCuandoOSRMDevuelveNull(t *testing.T) {
	var rutaConsultada string
	osrm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rutaConsultada = r.URL.RequestURI()
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write([]byte(respuestaValida("null")))
	}))
	defer osrm.Close()

	calculador, err := Nuevo(configuracionPrueba(osrm.URL))
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := calculador.Calcular(context.Background(), solicitudPrueba())
	if err != nil {
		t.Fatal(err)
	}
	if resultado.VersionGrafo != versionGrafoPrueba || resultado.Motor != "osrm_on_premise" ||
		resultado.Ambito != "Granada provincia + 15 km" || len(resultado.Alternativas) != 1 {
		t.Fatalf("resultado inesperado: %+v", resultado)
	}
	if !strings.Contains(rutaConsultada, "/route/v1/driving/-3.598600,37.177300;-3.655400,37.230600") ||
		!strings.Contains(rutaConsultada, "overview=full") || !strings.Contains(rutaConsultada, "geometries=geojson") ||
		!strings.Contains(rutaConsultada, "steps=false") || !strings.Contains(rutaConsultada, "alternatives=1") {
		t.Fatalf("consulta OSRM fuera del contrato fijo: %s", rutaConsultada)
	}
}

func TestCalculadorAceptaSoloVersionDeclaradaCoincidente(t *testing.T) {
	for _, prueba := range []struct {
		nombre      string
		versionJSON string
		quiereError bool
	}{
		{nombre: "coincidente", versionJSON: `"` + versionGrafoPrueba + `"`},
		{nombre: "ausente", versionJSON: "null"},
		{nombre: "conflicto", versionJSON: `"otro-grafo"`, quiereError: true},
		{nombre: "tipo no admitido", versionJSON: "7", quiereError: true},
	} {
		t.Run(prueba.nombre, func(t *testing.T) {
			osrm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(respuestaValida(prueba.versionJSON)))
			}))
			defer osrm.Close()
			calculador, err := Nuevo(configuracionPrueba(osrm.URL))
			if err != nil {
				t.Fatal(err)
			}
			_, err = calculador.Calcular(context.Background(), solicitudPrueba())
			if prueba.quiereError && !errors.Is(err, dietasports.ErrRespuestaMotorRutasInvalida) {
				t.Fatalf("error = %v; se esperaba respuesta no valida", err)
			}
			if !prueba.quiereError && err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCalculadorRechazaConfiguracionParcialOInsegura(t *testing.T) {
	base := configuracionPrueba("http://127.0.0.1:5000")
	pruebas := []struct {
		nombre string
		muta   func(*Configuracion)
	}{
		{nombre: "sin URL", muta: func(c *Configuracion) { c.URLBase = "" }},
		{nombre: "sin ambito", muta: func(c *Configuracion) { c.NombreAmbito = "" }},
		{nombre: "sin limites", muta: func(c *Configuracion) { c.LimitesAmbito = "" }},
		{nombre: "sin redes", muta: func(c *Configuracion) { c.CIDRPermitidas = nil }},
		{nombre: "sin version", muta: func(c *Configuracion) { c.VersionGrafo = "" }},
		{nombre: "red universal", muta: func(c *Configuracion) { c.CIDRPermitidas = []string{"0.0.0.0/0"} }},
		{nombre: "URL con ruta", muta: func(c *Configuracion) { c.URLBase += "/api" }},
		{nombre: "OSRM publico", muta: func(c *Configuracion) { c.URLBase = "https://router.project-osrm.org" }},
		{nombre: "version no canonica", muta: func(c *Configuracion) { c.VersionGrafo = " version " }},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			configuracion := base
			prueba.muta(&configuracion)
			if calculador, err := Nuevo(configuracion); err == nil || calculador != nil {
				t.Fatalf("configuracion insegura aceptada: calculador=%v error=%v", calculador, err)
			}
		})
	}
	if calculador, err := Nuevo(Configuracion{}); err != nil || calculador != nil {
		t.Fatalf("la ausencia total debe dejar el puerto desconectado: calculador=%v error=%v", calculador, err)
	}
}

func TestCalculadorRechazaEntradaAntesDeConectar(t *testing.T) {
	var consultas atomic.Int32
	osrm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		consultas.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(respuestaValida("null")))
	}))
	defer osrm.Close()
	calculador, err := Nuevo(configuracionPrueba(osrm.URL))
	if err != nil {
		t.Fatal(err)
	}

	pruebas := []dietasports.SolicitudCalculoRuta{
		{},
		{Alternativas: 4, Coordenadas: solicitudPrueba().Coordenadas},
		{Coordenadas: []dietasports.CoordenadaRuta{{Latitud: 37.1773, Longitud: -3.5986}, {Latitud: 40.4168, Longitud: -3.7038}}},
		{Coordenadas: []dietasports.CoordenadaRuta{{Latitud: 37.1773, Longitud: -3.5986, Nombre: " Granada"}, {Latitud: 37.2306, Longitud: -3.6554}}},
	}
	for indice, solicitud := range pruebas {
		_, err := calculador.Calcular(context.Background(), solicitud)
		if !errors.Is(err, dietasports.ErrSolicitudRutaInvalida) {
			t.Errorf("caso %d: error = %v", indice, err)
		}
	}
	if consultas.Load() != 0 {
		t.Fatalf("entradas invalidas alcanzaron OSRM %d veces", consultas.Load())
	}
}

func TestCalculadorNoUsaProxyNiSigueRedirecciones(t *testing.T) {
	var consultasDestino atomic.Int32
	destino := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		consultasDestino.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(respuestaValida("null")))
	}))
	defer destino.Close()
	origen := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, destino.URL+r.URL.RequestURI(), http.StatusTemporaryRedirect)
	}))
	defer origen.Close()

	t.Setenv("HTTP_PROXY", "http://127.0.0.1:1")
	calculador, err := Nuevo(configuracionPrueba(origen.URL))
	if err != nil {
		t.Fatal(err)
	}
	_, err = calculador.Calcular(context.Background(), solicitudPrueba())
	if !errors.Is(err, dietasports.ErrMotorRutasNoDisponible) {
		t.Fatalf("redireccion devuelta como error = %v", err)
	}
	if consultasDestino.Load() != 0 {
		t.Fatal("el conector siguio una redireccion")
	}
}

func TestCalculadorRechazaRespuestaFueraDeContrato(t *testing.T) {
	pruebas := []struct {
		nombre        string
		tipo          string
		contenido     string
		longitudFalsa int64
	}{
		{nombre: "tipo no JSON", tipo: "text/plain", contenido: respuestaValida("null")},
		{nombre: "JSON no UTF-8", tipo: "application/json", contenido: string([]byte{0xff})},
		{nombre: "sin rutas", tipo: "application/json", contenido: `{"code":"Ok","routes":[]}`},
		{nombre: "ruta incompleta", tipo: "application/json", contenido: `{"code":"Ok","routes":[{"distance":1,"duration":1}]}`},
		{nombre: "longitud declarada excesiva", tipo: "application/json", contenido: `{}`, longitudFalsa: maximoBytesRespuesta + 1},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			osrm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", prueba.tipo)
				if prueba.longitudFalsa > 0 {
					w.Header().Set("Content-Length", fmt.Sprint(prueba.longitudFalsa))
				}
				_, _ = w.Write([]byte(prueba.contenido))
			}))
			defer osrm.Close()
			calculador, err := Nuevo(configuracionPrueba(osrm.URL))
			if err != nil {
				t.Fatal(err)
			}
			_, err = calculador.Calcular(context.Background(), solicitudPrueba())
			if !errors.Is(err, dietasports.ErrRespuestaMotorRutasInvalida) {
				t.Fatalf("error = %v; se esperaba respuesta no valida", err)
			}
		})
	}
}
