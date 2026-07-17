package server

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"vec-diputacion-granada/config"
)

func TestSuperficiesRechazanRutasEscapadasONoCanonicas(t *testing.T) {
	casos := []struct {
		nombre string
		nuevo  func(config.Config, http.Handler) http.Handler
		ruta   string
	}{
		{nombre: "publica", nuevo: NewHandlerPublicoWithConfig, ruta: "/api/publico/consulta"},
		{nombre: "interna", nuevo: NewHandlerInternoWithConfig, ruta: "/api/vec/consulta"},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			llamadas := 0
			handler := caso.nuevo(config.Config{}, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				llamadas++
				w.WriteHeader(http.StatusNoContent)
			}))

			peticionCanonica := peticionServidorPrueba(http.MethodGet, caso.ruta, nil)
			recCanonica := httptest.NewRecorder()
			handler.ServeHTTP(recCanonica, peticionCanonica)
			if recCanonica.Code != http.StatusNoContent || llamadas != 1 {
				t.Fatalf("ruta canonica = %d, llamadas=%d", recCanonica.Code, llamadas)
			}

			mutaciones := []struct {
				nombre string
				muta   func(*http.Request)
			}{
				{"RawPath equivalente", func(r *http.Request) { r.URL.RawPath = caso.ruta }},
				{"segmento escapado", func(r *http.Request) { r.URL.RawPath = sustituirConsultaEscapada(caso.ruta) }},
				{"porcentaje literal", func(r *http.Request) { r.URL.Path += "%2fajena" }},
				{"doble barra inicial", func(r *http.Request) { r.URL.Path = "/" + caso.ruta }},
				{"doble barra interior", func(r *http.Request) { r.URL.Path += "//ajena" }},
				{"segmento punto", func(r *http.Request) { r.URL.Path += "/./ajena" }},
				{"salto de directorio", func(r *http.Request) { r.URL.Path += "/../ajena" }},
				{"barra inversa", func(r *http.Request) { r.URL.Path += `\ajena` }},
				{"fragmento", func(r *http.Request) { r.URL.Fragment = "ajeno" }},
			}
			for _, mutacion := range mutaciones {
				t.Run(mutacion.nombre, func(t *testing.T) {
					peticion := peticionServidorPrueba(http.MethodGet, caso.ruta, nil)
					mutacion.muta(peticion)
					rec := httptest.NewRecorder()
					handler.ServeHTTP(rec, peticion)
					if rec.Code != http.StatusNotFound || rec.Header().Get("Location") != "" {
						t.Fatalf("ruta no canonica = %d, Location=%q", rec.Code, rec.Header().Get("Location"))
					}
				})
			}
			if llamadas != 1 {
				t.Fatalf("la API recibio %d rutas; solo debio recibir la canonica", llamadas)
			}
		})
	}
}

func sustituirConsultaEscapada(ruta string) string {
	return ruta[:len(ruta)-len("consulta")] + "%63onsulta"
}

func TestSuperficieInternaRechazaIdentidadHeredadaPeroAdmiteAuthorization(t *testing.T) {
	llamadas := 0
	autorizacionRecibida := ""
	handler := NewHandlerInternoWithConfig(config.Config{}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		llamadas++
		autorizacionRecibida = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusNoContent)
	}))

	for _, cabecera := range []string{
		"X-VEC-Subject", "X-VEC-Roles", "X-VEC-Auth-Mechanism", "X-VEC-Campo-Futuro",
		"X-Auth-Subject", "X-Auth-Role", "X-Auth-Roles", "X-Auth-Assurance", "X-Auth-Token",
		"X-Remote-User", "Remote-User", "X-Forwarded-User", "X-Forwarded-For",
	} {
		t.Run(cabecera, func(t *testing.T) {
			peticion := peticionServidorPrueba(http.MethodGet, "/api/vec/consulta", nil)
			// La mera presencia, incluso vacia, debe cerrar la peticion.
			peticion.Header[cabecera] = []string{""}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, peticion)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("cabecera heredada %q = %d; se esperaba 400", cabecera, rec.Code)
			}
		})
	}
	if llamadas != 0 {
		t.Fatalf("la API recibio %d peticiones con identidad declarada", llamadas)
	}

	peticion := peticionServidorPrueba(http.MethodGet, "/api/vec/consulta", nil)
	peticion.Header.Set("Authorization", "Bearer asercion-protegida-futura")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, peticion)
	if rec.Code != http.StatusNoContent || llamadas != 1 {
		t.Fatalf("Authorization = %d, llamadas=%d; debe alcanzar la futura frontera", rec.Code, llamadas)
	}
	if autorizacionRecibida != "Bearer asercion-protegida-futura" {
		t.Fatalf("Authorization fue alterada: %q", autorizacionRecibida)
	}
}

func TestSuperficiesSuprimenTodoCuerpoEnHEAD(t *testing.T) {
	casos := []struct {
		nombre string
		nuevo  func(config.Config, http.Handler) http.Handler
		ruta   string
	}{
		{nombre: "publica", nuevo: NewHandlerPublicoWithConfig, ruta: "/api/publico/consulta"},
		{nombre: "interna", nuevo: NewHandlerInternoWithConfig, ruta: "/api/vec/consulta"},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			handler := caso.nuevo(config.Config{}, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"dato":"no debe salir"}`))
			}))
			for _, ruta := range []string{"/healthz", caso.ruta} {
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, peticionServidorPrueba(http.MethodHead, ruta, nil))
				if rec.Code != http.StatusOK || rec.Body.Len() != 0 {
					t.Errorf("HEAD %s = %d, cuerpo=%q", ruta, rec.Code, rec.Body.String())
				}
			}
		})
	}
}

func TestSuperficieInternaSaneaCookiesAntesDeFlush(t *testing.T) {
	expusoFlusher := false
	expusoUnwrap := false
	handler := NewHandlerInternoWithConfig(config.Config{}, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, expusoUnwrap = w.(interface{ Unwrap() http.ResponseWriter })
		flusher, ok := w.(http.Flusher)
		expusoFlusher = ok
		w.Header().Add("Set-Cookie", "antes=prohibida")
		w.Header().Add("Trailer", "Set-Cookie, X-Traza-Permitida")
		w.Header().Set(http.TrailerPrefix+"Set-Cookie", "trailer=prohibida")
		if ok {
			flusher.Flush()
		}
		w.Header().Add("Set-Cookie", "despues=tambien-prohibida")
		w.Header().Set(http.TrailerPrefix+"Set-Cookie", "trailer-final=prohibida")
		_, _ = w.Write([]byte("respuesta"))
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, peticionServidorPrueba(http.MethodGet, "/api/vec/consulta", nil))
	resultado := rec.Result()
	defer resultado.Body.Close()
	if !expusoFlusher || !expusoUnwrap || !rec.Flushed {
		t.Fatalf("interfaces: Flusher=%t Unwrap=%t, flushed=%t", expusoFlusher, expusoUnwrap, rec.Flushed)
	}
	if cookies := resultado.Header.Values("Set-Cookie"); len(cookies) != 0 {
		t.Fatalf("cookies en cabeceras comprometidas: %v", cookies)
	}
	if cookies := resultado.Trailer.Values("Set-Cookie"); len(cookies) != 0 {
		t.Fatalf("cookies en trailers comprometidos: %v", cookies)
	}
	if cookies := rec.Header().Values("Set-Cookie"); len(cookies) != 0 {
		t.Fatalf("cookies conservadas tras la respuesta: %v", cookies)
	}
}

func TestEscritorSinCookiesNoConfundeInformativaConRespuestaFinal(t *testing.T) {
	escritor := &escritorEstadosPrueba{cabeceras: make(http.Header)}
	handler := prohibirCookiesYAutorizacionProxy(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Set-Cookie", "informativa=prohibida")
		w.WriteHeader(http.StatusEarlyHints)
		w.Header().Set("Set-Cookie", "final=prohibida")
		w.WriteHeader(http.StatusNoContent)
	}))
	handler.ServeHTTP(escritor, peticionServidorPrueba(http.MethodGet, "/api/vec/consulta", nil))
	if !reflect.DeepEqual(escritor.estados, []int{http.StatusEarlyHints, http.StatusNoContent}) {
		t.Fatalf("estados escritos = %v", escritor.estados)
	}
	for indice, cookies := range escritor.cookiesPorEstado {
		if len(cookies) != 0 {
			t.Fatalf("respuesta %d con cookies: %v", indice, cookies)
		}
	}
}

type escritorEstadosPrueba struct {
	cabeceras        http.Header
	estados          []int
	cookiesPorEstado [][]string
}

func (w *escritorEstadosPrueba) Header() http.Header { return w.cabeceras }

func (w *escritorEstadosPrueba) WriteHeader(estado int) {
	w.estados = append(w.estados, estado)
	w.cookiesPorEstado = append(w.cookiesPorEstado, append([]string(nil), w.cabeceras.Values("Set-Cookie")...))
}

func (w *escritorEstadosPrueba) Write(contenido []byte) (int, error) {
	return len(contenido), nil
}
