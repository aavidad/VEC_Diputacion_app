package httpinterno

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
)

func nuevoManejadorResultadoCoberturaPrueba(
	t *testing.T,
) (http.Handler, *consultorResultadoCoberturaPrueba) {
	t.Helper()
	consultor := &consultorResultadoCoberturaPrueba{
		resultado: application.DatosConsultaResultadoCoberturaParaAdaptador{
			Estado: application.ResultadoCoberturaNoObservable,
		},
	}
	manejador, err := NuevoManejadorResultadoCobertura(consultor)
	if err != nil {
		t.Fatal(err)
	}
	return manejador, consultor
}

func TestManejadorResultadoCoberturaRechazaCuerpoNoCerrado(
	t *testing.T,
) {
	t.Parallel()
	casos := map[string][]byte{
		"vacio": {},
		"nulo":  []byte(`null`),
		"lista": []byte(`[]`),
		"campo extra": []byte(
			strings.TrimSuffix(cuerpoResultadoCoberturaPrueba(), "}") +
				`,"actor_ref":"actor:fabricado"}`,
		),
		"organizacion": []byte(
			strings.TrimSuffix(cuerpoResultadoCoberturaPrueba(), "}") +
				`,"organizacion_ref":"organizacion:fabricada"}`,
		),
		"perfil": []byte(
			strings.TrimSuffix(cuerpoResultadoCoberturaPrueba(), "}") +
				`,"perfil_ref":"perfil:fabricado"}`,
		),
		"version": []byte(
			strings.TrimSuffix(cuerpoResultadoCoberturaPrueba(), "}") +
				`,"version_esperada":1}`,
		),
		"via": []byte(
			strings.TrimSuffix(cuerpoResultadoCoberturaPrueba(), "}") +
				`,"via_elegida":"bolsa_vigente"}`,
		),
		"duplicado expediente": []byte(
			`{"expediente_ref":"expediente:ct:0001",` +
				`"expediente_ref":"expediente:ct:0002",` +
				`"clave_idempotencia":"` +
				claveResultadoCoberturaPrueba + `"}`,
		),
		"duplicado clave": []byte(
			`{"expediente_ref":"expediente:ct:0001",` +
				`"clave_idempotencia":"` +
				claveResultadoCoberturaPrueba + `",` +
				`"clave_idempotencia":"` +
				claveResultadoCoberturaPrueba + `"}`,
		),
		"json posterior": []byte(
			cuerpoResultadoCoberturaPrueba() + `{}`,
		),
		"utf8 invalido": {
			'{', '"', 'e', 'x', 'p', 'e', 'd', 'i', 'e', 'n', 't', 'e',
			'_', 'r', 'e', 'f', '"', ':', '"', 0xff, '"', '}',
		},
		"expediente invalido": []byte(
			`{"expediente_ref":"x",` +
				`"clave_idempotencia":"` +
				claveResultadoCoberturaPrueba + `"}`,
		),
		"clave invalida": []byte(
			`{"expediente_ref":"expediente:ct:0001",` +
				`"clave_idempotencia":"00000000-0000-4000-8000-000000000000"}`,
		),
	}
	for nombre, cuerpo := range casos {
		nombre, cuerpo := nombre, cuerpo
		t.Run(nombre, func(t *testing.T) {
			t.Parallel()
			manejador, consultor := nuevoManejadorResultadoCoberturaPrueba(t)
			peticion := httptest.NewRequest(
				http.MethodPost,
				RutaResultadoCobertura,
				bytes.NewReader(cuerpo),
			)
			peticion.Header.Set(
				"Content-Type",
				"application/json; charset=utf-8",
			)
			peticion.Header.Set("Accept", "application/json")
			respuesta := httptest.NewRecorder()
			manejador.ServeHTTP(respuesta, peticion)
			if respuesta.Code != http.StatusBadRequest &&
				respuesta.Code != http.StatusUnprocessableEntity {
				t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body)
			}
			if consultor.total() != 0 {
				t.Fatal("entrada inválida alcanzó el caso de uso")
			}
		})
	}
}

func TestManejadorResultadoCoberturaRechazaAutoridadYCookies(
	t *testing.T,
) {
	t.Parallel()
	cabeceras := []string{
		"Authorization",
		"Cookie",
		"Set-Cookie",
		"Proxy-Authorization",
		"Forwarded",
		"Remote-User",
		"X-Remote-User",
		"X-Forwarded-User",
		"X-Forwarded-For",
		"X-Envoy-External-Address",
		"Actor",
		"X-Actor",
		"Perfil",
		"Organizacion",
		"Roles",
		"Idempotency-Key",
		"X-Idempotency-Key",
		"Via",
		"Connection",
		"Keep-Alive",
		"Upgrade",
	}
	for _, nombre := range cabeceras {
		nombre := nombre
		t.Run(nombre, func(t *testing.T) {
			t.Parallel()
			manejador, consultor := nuevoManejadorResultadoCoberturaPrueba(t)
			peticion := nuevaPeticionResultadoCoberturaPrueba()
			peticion.Header.Set(nombre, "fabricada")
			respuesta := httptest.NewRecorder()
			manejador.ServeHTTP(respuesta, peticion)
			if respuesta.Code != http.StatusBadRequest {
				t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body)
			}
			if consultor.total() != 0 {
				t.Fatal("cabecera prohibida alcanzó el caso de uso")
			}
		})
	}
	t.Run("token hop by hop", func(t *testing.T) {
		t.Parallel()
		manejador, consultor := nuevoManejadorResultadoCoberturaPrueba(t)
		peticion := nuevaPeticionResultadoCoberturaPrueba()
		peticion.Header.Set("Connection", "X-Autoridad-Fabricada")
		peticion.Header.Set("X-Autoridad-Fabricada", "actor:externo")
		respuesta := httptest.NewRecorder()
		manejador.ServeHTTP(respuesta, peticion)
		if respuesta.Code != http.StatusBadRequest ||
			consultor.total() != 0 {
			t.Fatalf("estado=%d llamadas=%d", respuesta.Code, consultor.total())
		}
	})
}

func TestManejadorResultadoCoberturaExigeRutaCanonicaYPOST(
	t *testing.T,
) {
	t.Parallel()
	type mutacionRuta struct {
		nombre   string
		mutar    func(*http.Request)
		objetivo string
	}
	casos := []mutacionRuta{
		{"consulta", nil, RutaResultadoCobertura + "?clave=secreta"},
		{"barra", nil, RutaResultadoCobertura + "/"},
		{"escape", nil, "/api/vec/contratacion-temporal/cobertura/%72esultados"},
		{"force query", func(r *http.Request) { r.URL.ForceQuery = true }, ""},
		{"raw path", func(r *http.Request) { r.URL.RawPath = RutaResultadoCobertura }, ""},
		{"scheme", func(r *http.Request) { r.URL.Scheme = "https" }, ""},
		{"host", func(r *http.Request) { r.URL.Host = "host.fabricado" }, ""},
		{"user", func(r *http.Request) { r.URL.User = url.User("actor") }, ""},
		{"opaque", func(r *http.Request) { r.URL.Opaque = RutaResultadoCobertura }, ""},
		{"fragmento", func(r *http.Request) { r.URL.Fragment = "secreto" }, ""},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			manejador, consultor := nuevoManejadorResultadoCoberturaPrueba(t)
			peticion := nuevaPeticionResultadoCoberturaPrueba()
			if caso.objetivo != "" {
				peticion = httptest.NewRequest(
					http.MethodPost,
					caso.objetivo,
					bytes.NewBufferString(cuerpoResultadoCoberturaPrueba()),
				)
				peticion.Header.Set(
					"Content-Type",
					"application/json; charset=utf-8",
				)
				peticion.Header.Set("Accept", "application/json")
			}
			if caso.mutar != nil {
				caso.mutar(peticion)
			}
			respuesta := httptest.NewRecorder()
			manejador.ServeHTTP(respuesta, peticion)
			if respuesta.Code != http.StatusNotFound ||
				consultor.total() != 0 {
				t.Fatalf(
					"estado=%d llamadas=%d",
					respuesta.Code,
					consultor.total(),
				)
			}
		})
	}
	t.Run("método", func(t *testing.T) {
		t.Parallel()
		manejador, consultor := nuevoManejadorResultadoCoberturaPrueba(t)
		peticion := nuevaPeticionResultadoCoberturaPrueba()
		peticion.Method = http.MethodGet
		respuesta := httptest.NewRecorder()
		manejador.ServeHTTP(respuesta, peticion)
		if respuesta.Code != http.StatusMethodNotAllowed ||
			respuesta.Header().Get("Allow") != http.MethodPost ||
			consultor.total() != 0 {
			t.Fatalf(
				"estado=%d allow=%q llamadas=%d",
				respuesta.Code,
				respuesta.Header().Get("Allow"),
				consultor.total(),
			)
		}
	})
}

func TestManejadorResultadoCoberturaConservaErroresTecnicos(
	t *testing.T,
) {
	t.Parallel()
	casos := []struct {
		nombre string
		mutar  func(*http.Request)
		estado int
	}{
		{
			"tipo de contenido",
			func(r *http.Request) {
				r.Header.Set("Content-Type", "text/plain")
			},
			http.StatusUnsupportedMediaType,
		},
		{
			"accept",
			func(r *http.Request) {
				r.Header.Set("Accept", "application/xml")
			},
			http.StatusNotAcceptable,
		},
		{
			"compresion",
			func(r *http.Request) {
				r.Header.Set("Content-Encoding", "gzip")
			},
			http.StatusBadRequest,
		},
		{
			"trailer",
			func(r *http.Request) {
				r.Trailer = http.Header{"X-Final": []string{"valor"}}
			},
			http.StatusBadRequest,
		},
		{
			"transferencia",
			func(r *http.Request) {
				r.TransferEncoding = []string{"gzip"}
			},
			http.StatusBadRequest,
		},
		{
			"sin cuerpo",
			func(r *http.Request) {
				r.Body = http.NoBody
				r.ContentLength = 0
			},
			http.StatusBadRequest,
		},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			manejador, consultor := nuevoManejadorResultadoCoberturaPrueba(t)
			peticion := nuevaPeticionResultadoCoberturaPrueba()
			caso.mutar(peticion)
			respuesta := httptest.NewRecorder()
			manejador.ServeHTTP(respuesta, peticion)
			if respuesta.Code != caso.estado || consultor.total() != 0 {
				t.Fatalf(
					"estado=%d esperado=%d llamadas=%d cuerpo=%s",
					respuesta.Code,
					caso.estado,
					consultor.total(),
					respuesta.Body,
				)
			}
		})
	}
}

func TestManejadorResultadoCoberturaAcotaCuerpoDeclaradoYStreaming(
	t *testing.T,
) {
	t.Parallel()
	casos := []struct {
		nombre   string
		peticion func() *http.Request
	}{
		{
			"declarado",
			func() *http.Request {
				peticion := nuevaPeticionResultadoCoberturaPrueba()
				peticion.ContentLength = MaximoCuerpoCoberturaBytes + 1
				return peticion
			},
		},
		{
			"streaming",
			func() *http.Request {
				peticion := httptest.NewRequest(
					http.MethodPost,
					RutaResultadoCobertura,
					strings.NewReader(
						strings.Repeat("x", MaximoCuerpoCoberturaBytes+1),
					),
				)
				peticion.ContentLength = -1
				peticion.Header.Set(
					"Content-Type",
					"application/json; charset=utf-8",
				)
				peticion.Header.Set("Accept", "application/json")
				return peticion
			},
		},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			manejador, consultor := nuevoManejadorResultadoCoberturaPrueba(t)
			respuesta := httptest.NewRecorder()
			manejador.ServeHTTP(respuesta, caso.peticion())
			if respuesta.Code != http.StatusRequestEntityTooLarge ||
				consultor.total() != 0 {
				t.Fatalf(
					"estado=%d llamadas=%d",
					respuesta.Code,
					consultor.total(),
				)
			}
		})
	}
}

func TestManejadorResultadoCoberturaMantieneCabecerasSeguras(
	t *testing.T,
) {
	t.Parallel()
	manejador, _ := nuevoManejadorResultadoCoberturaPrueba(t)
	respuesta := httptest.NewRecorder()
	respuesta.Header().Set("Set-Cookie", "sesion=prohibida")
	respuesta.Header().Set("Retry-After", "1")
	respuesta.Header().Set("Access-Control-Allow-Origin", "*")
	respuesta.Header().Set("Location", "https://otro.invalid")
	manejador.ServeHTTP(respuesta, nuevaPeticionResultadoCoberturaPrueba())
	if respuesta.Code != http.StatusAccepted {
		t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body)
	}
	for _, prohibida := range []string{
		"Set-Cookie",
		"Retry-After",
		"Access-Control-Allow-Origin",
		"Location",
		"Content-Encoding",
	} {
		if respuesta.Header().Get(prohibida) != "" {
			t.Fatalf("cabecera %s publicada", prohibida)
		}
	}
	esperadas := map[string]string{
		"Content-Type":                 "application/json; charset=utf-8",
		"Cache-Control":                "no-store, no-transform",
		"Pragma":                       "no-cache",
		"X-Content-Type-Options":       "nosniff",
		"Referrer-Policy":              "no-referrer",
		"X-Frame-Options":              "DENY",
		"Cross-Origin-Resource-Policy": "same-origin",
	}
	for nombre, valor := range esperadas {
		if recibido := respuesta.Header().Get(nombre); recibido != valor {
			t.Fatalf("%s=%q", nombre, recibido)
		}
	}
}

func TestManejadorResultadoCoberturaCanceladoNoConsultaNiPublica202(
	t *testing.T,
) {
	t.Parallel()
	manejador, consultor := nuevoManejadorResultadoCoberturaPrueba(t)
	contexto, cancelar := context.WithCancel(context.Background())
	cancelar()
	peticion := nuevaPeticionResultadoCoberturaPrueba().WithContext(contexto)
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusServiceUnavailable ||
		consultor.total() != 0 ||
		strings.Contains(respuesta.Body.String(), "no_observable") {
		t.Fatalf(
			"estado=%d llamadas=%d cuerpo=%s",
			respuesta.Code,
			consultor.total(),
			respuesta.Body,
		)
	}
}

func TestManejadorResultadoCoberturaEsSeguroEnConcurrencia(
	t *testing.T,
) {
	t.Parallel()
	manejador, consultor := nuevoManejadorResultadoCoberturaPrueba(t)
	const total = 48
	var grupo sync.WaitGroup
	grupo.Add(total)
	errores := make(chan string, total)
	for indice := 0; indice < total; indice++ {
		go func() {
			defer grupo.Done()
			respuesta := httptest.NewRecorder()
			manejador.ServeHTTP(
				respuesta,
				nuevaPeticionResultadoCoberturaPrueba(),
			)
			if respuesta.Code != http.StatusAccepted {
				errores <- respuesta.Body.String()
			}
		}()
	}
	grupo.Wait()
	close(errores)
	for err := range errores {
		t.Errorf("respuesta concurrente: %s", err)
	}
	if consultor.total() != total {
		t.Fatalf("llamadas=%d", consultor.total())
	}
}
