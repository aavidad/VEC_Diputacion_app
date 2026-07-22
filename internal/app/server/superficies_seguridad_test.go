package server

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
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

type superficieSinCookiesPrueba struct {
	nombre string
	nuevo  func(config.Config, http.Handler) http.Handler
	ruta   string
}

func superficiesSinCookiesPrueba() []superficieSinCookiesPrueba {
	return []superficieSinCookiesPrueba{
		{nombre: "publica", nuevo: NewHandlerPublicoWithConfig, ruta: "/api/publico/consulta"},
		{nombre: "interna", nuevo: NewHandlerInternoWithConfig, ruta: "/api/vec/consulta"},
		{nombre: "integrada_heredada", nuevo: NewHandlerWithConfig, ruta: "/api/consulta"},
	}
}

func TestSuperficiesRechazanCredencialesAmbientales(t *testing.T) {
	for _, superficie := range superficiesSinCookiesPrueba() {
		t.Run(superficie.nombre, func(t *testing.T) {
			llamadas := 0
			handler := superficie.nuevo(config.Config{}, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				llamadas++
				w.WriteHeader(http.StatusNoContent)
			}))

			for _, cabecera := range []string{
				"cOoKiE", "Proxy-Authorization", "X-VEC-Subject", "X-Auth-Token",
				"X-Remote-User", "Remote-User", "X-Forwarded-For", "Forwarded", "Via",
			} {
				t.Run(cabecera, func(t *testing.T) {
					peticion := peticionServidorPrueba(http.MethodGet, superficie.ruta, nil)
					// La presencia se rechaza aunque el valor este vacio y el nombre
					// no use la capitalizacion canonica de HTTP.
					peticion.Header[cabecera] = []string{""}
					respuesta := httptest.NewRecorder()
					handler.ServeHTTP(respuesta, peticion)
					if respuesta.Code != http.StatusBadRequest {
						t.Fatalf("cabecera %q = %d; se esperaba 400", cabecera, respuesta.Code)
					}
					if respuesta.Header().Get("Set-Cookie") != "" {
						t.Fatalf("la respuesta de rechazo emitio una cookie")
					}
				})
			}
			for _, cabecera := range []string{
				"Cookie", "Proxy-Authorization", "X-VEC-Subject", "X-Auth-Token",
				"X-Remote-User", "Remote-User", "X-Forwarded-For", "Forwarded", "Via",
			} {
				t.Run("trailer_"+cabecera, func(t *testing.T) {
					peticion := peticionServidorPrueba(http.MethodGet, superficie.ruta, nil)
					peticion.Trailer = http.Header{cabecera: []string{""}}
					respuesta := httptest.NewRecorder()
					handler.ServeHTTP(respuesta, peticion)
					if respuesta.Code != http.StatusBadRequest {
						t.Fatalf("trailer %q = %d; se esperaba 400", cabecera, respuesta.Code)
					}
				})
			}
			if llamadas != 0 {
				t.Fatalf("la API recibio %d peticiones con credenciales ambientales", llamadas)
			}
		})
	}
}

func TestSuperficiePublicaRechazaAuthorizationInclusoComoTrailer(t *testing.T) {
	llamadas := 0
	handler := NewHandlerPublicoWithConfig(config.Config{}, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		llamadas++
		w.WriteHeader(http.StatusNoContent)
	}))
	pruebas := []struct {
		nombre   string
		preparar func(*http.Request)
	}{
		{nombre: "cabecera", preparar: func(r *http.Request) {
			r.Header["aUtHoRiZaTiOn"] = []string{""}
		}},
		{nombre: "trailer", preparar: func(r *http.Request) {
			r.Trailer = http.Header{"aUtHoRiZaTiOn": []string{""}}
		}},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			peticion := peticionServidorPrueba(http.MethodGet, "/api/publico/consulta", nil)
			prueba.preparar(peticion)
			respuesta := httptest.NewRecorder()
			handler.ServeHTTP(respuesta, peticion)
			if respuesta.Code != http.StatusBadRequest {
				t.Fatalf("Authorization anonima = %d; se esperaba 400", respuesta.Code)
			}
		})
	}
	if llamadas != 0 {
		t.Fatalf("la API anonima recibio %d credenciales de otra audiencia", llamadas)
	}
}

func TestSuperficiesAislanTrailersTardiosDelTransporteHTTP(t *testing.T) {
	trailerVisibleSinBarrera := false
	servidorSinBarrera := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		trailerVisibleSinBarrera = contieneCabecera(r.Trailer, "X-VEC-Subject")
		w.WriteHeader(http.StatusNoContent)
	}))
	estadoSinBarrera := ejecutarPeticionConTrailerTardio(
		t,
		servidorSinBarrera.Listener.Addr().String(),
		"/control",
		"X-VEC-Subject",
		"sujeto-inyectado",
	)
	servidorSinBarrera.Close()
	if estadoSinBarrera != http.StatusNoContent || !trailerVisibleSinBarrera {
		t.Fatalf(
			"el control no reprodujo el trailer tardio: estado=%d visible=%t",
			estadoSinBarrera,
			trailerVisibleSinBarrera,
		)
	}

	for _, superficie := range superficiesSinCookiesPrueba() {
		t.Run(superficie.nombre, func(t *testing.T) {
			llamadas := 0
			handler := superficie.nuevo(config.Config{}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				llamadas++
				if _, err := io.ReadAll(r.Body); err != nil {
					t.Errorf("leer cuerpo: %v", err)
				}
				w.WriteHeader(http.StatusNoContent)
			}))
			servidor := httptest.NewServer(handler)
			defer servidor.Close()

			trailers := []string{"Cookie", "X-VEC-Subject"}
			if superficie.nombre == "publica" || superficie.nombre == "interna" {
				trailers = append(trailers, "Authorization")
			}
			for _, trailer := range trailers {
				t.Run(trailer, func(t *testing.T) {
					estado := ejecutarPeticionConTrailerTardio(
						t,
						servidor.Listener.Addr().String(),
						superficie.ruta,
						trailer,
						"valor-inyectado",
					)
					if estado != http.StatusBadRequest {
						t.Fatalf("estado = %d; se esperaba %d", estado, http.StatusBadRequest)
					}
				})
			}
			if llamadas != 0 {
				t.Fatalf("la aplicacion recibio %d peticiones con trailers tardios", llamadas)
			}
		})
	}
}

func ejecutarPeticionConTrailerTardio(
	t *testing.T,
	direccion string,
	ruta string,
	nombre string,
	valor string,
) int {
	t.Helper()
	conexion, err := net.Dial("tcp", direccion)
	if err != nil {
		t.Fatalf("conectar con servidor de prueba: %v", err)
	}
	defer conexion.Close()
	if _, err := fmt.Fprintf(
		conexion,
		"POST %s HTTP/1.1\r\nHost: prueba\r\nTransfer-Encoding: chunked\r\nConnection: close\r\n\r\n4\r\ndato\r\n0\r\n%s: %s\r\n\r\n",
		ruta,
		nombre,
		valor,
	); err != nil {
		t.Fatalf("escribir peticion: %v", err)
	}
	respuesta, err := http.ReadResponse(bufio.NewReader(conexion), nil)
	if err != nil {
		t.Fatalf("leer respuesta: %v", err)
	}
	defer respuesta.Body.Close()
	_, _ = io.Copy(io.Discard, respuesta.Body)
	return respuesta.StatusCode
}

func TestBarreraDeTrailersAcotaElCuerpoAntesDeLaAplicacion(t *testing.T) {
	llamadas := 0
	handler := NewHandlerInternoWithConfig(
		config.Config{MaxRequestBodyBytes: 4},
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			llamadas++
			w.WriteHeader(http.StatusNoContent)
		}),
	)
	peticion := peticionServidorPrueba(
		http.MethodPost,
		"/api/vec/consulta",
		strings.NewReader("12345"),
	)
	peticion.TransferEncoding = []string{"chunked"}
	respuesta := httptest.NewRecorder()
	handler.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("cuerpo chunked = %d; se esperaba 413", respuesta.Code)
	}
	if llamadas != 0 {
		t.Fatalf("la aplicacion recibio %d cuerpos por encima del limite", llamadas)
	}
}

func TestBarreraDeTrailersReponeCuerpoChunkedDentroDelLimite(t *testing.T) {
	contenidoRecibido := ""
	longitudRecibida := int64(-1)
	transferenciaRecibida := []string(nil)
	expusoGetBody := false
	handler := NewHandlerInternoWithConfig(
		config.Config{MaxRequestBodyBytes: 4},
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			contenido, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("leer cuerpo repuesto: %v", err)
			}
			contenidoRecibido = string(contenido)
			longitudRecibida = r.ContentLength
			transferenciaRecibida = append([]string(nil), r.TransferEncoding...)
			expusoGetBody = r.GetBody != nil
			w.WriteHeader(http.StatusNoContent)
		}),
	)
	peticion := peticionServidorPrueba(
		http.MethodPost,
		"/api/vec/consulta",
		strings.NewReader("1234"),
	)
	peticion.TransferEncoding = []string{"chunked"}
	respuesta := httptest.NewRecorder()
	handler.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusNoContent {
		t.Fatalf("cuerpo chunked valido = %d; se esperaba 204", respuesta.Code)
	}
	if contenidoRecibido != "1234" || longitudRecibida != 4 || len(transferenciaRecibida) != 0 {
		t.Fatalf(
			"cuerpo repuesto = (%q, longitud=%d, transferencia=%v)",
			contenidoRecibido,
			longitudRecibida,
			transferenciaRecibida,
		)
	}
	if expusoGetBody {
		t.Fatal("la aplicacion recibio GetBody de la peticion de transporte")
	}
}

func TestSuperficieInternaRechazaIdentidadHeredadaYAuthorization(t *testing.T) {
	llamadas := 0
	handler := NewHandlerInternoWithConfig(config.Config{}, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		llamadas++
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

	pruebasAutorizacion := []struct {
		nombre   string
		preparar func(*http.Request)
	}{
		{nombre: "cabecera vacia", preparar: func(r *http.Request) {
			r.Header["Authorization"] = []string{""}
		}},
		{nombre: "cabecera Bearer", preparar: func(r *http.Request) {
			r.Header["Authorization"] = []string{"Bearer asercion-no-admitida"}
		}},
		{nombre: "capitalizacion no canonica", preparar: func(r *http.Request) {
			r.Header["aUtHoRiZaTiOn"] = []string{"Basic tampoco-admitida"}
		}},
		{nombre: "valores multiples", preparar: func(r *http.Request) {
			r.Header["Authorization"] = []string{"", "Bearer segunda-credencial"}
		}},
		{nombre: "trailer declarado", preparar: func(r *http.Request) {
			r.Trailer = http.Header{"aUtHoRiZaTiOn": []string{"Bearer trailer-no-admitido"}}
		}},
	}
	for _, prueba := range pruebasAutorizacion {
		t.Run(prueba.nombre, func(t *testing.T) {
			peticion := peticionServidorPrueba(http.MethodGet, "/api/vec/consulta", nil)
			prueba.preparar(peticion)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, peticion)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("Authorization obtuvo %d; se esperaba 400", rec.Code)
			}
		})
	}
	if llamadas != 0 {
		t.Fatalf("la API recibio %d peticiones con identidad o Authorization declaradas", llamadas)
	}
}

func TestSuperficiesSuprimenTodoCuerpoEnHEAD(t *testing.T) {
	casos := []struct {
		nombre string
		nuevo  func(config.Config, http.Handler) http.Handler
		ruta   string
		salud  int
	}{
		{nombre: "publica", nuevo: NewHandlerPublicoWithConfig, ruta: "/api/publico/consulta", salud: http.StatusServiceUnavailable},
		{nombre: "interna", nuevo: NewHandlerInternoWithConfig, ruta: "/api/vec/consulta", salud: http.StatusOK},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			handler := caso.nuevo(config.Config{}, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"dato":"no debe salir"}`))
			}))
			for _, prueba := range []struct {
				ruta   string
				estado int
			}{{"/healthz", caso.salud}, {caso.ruta, http.StatusOK}} {
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, peticionServidorPrueba(http.MethodHead, prueba.ruta, nil))
				if rec.Code != prueba.estado || rec.Body.Len() != 0 {
					t.Errorf("HEAD %s = %d, cuerpo=%q", prueba.ruta, rec.Code, rec.Body.String())
				}
			}
		})
	}
}

func TestSuperficiesSaneanCookiesAntesDeFlush(t *testing.T) {
	for _, superficie := range superficiesSinCookiesPrueba() {
		t.Run(superficie.nombre, func(t *testing.T) {
			expusoFlusher := false
			expusoUnwrap := false
			handler := superficie.nuevo(config.Config{}, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, expusoUnwrap = w.(interface{ Unwrap() http.ResponseWriter })
				flusher, ok := w.(http.Flusher)
				expusoFlusher = ok
				w.Header()["sEt-CoOkIe"] = []string{"caja=no-canonica"}
				w.Header()["tRaIlEr"] = []string{"sEt-CoOkIe, X-Traza-Permitida"}
				w.Header()["tRaIlEr:sEt-CoOkIe"] = []string{"trailer-caja=no-canonica"}
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
			handler.ServeHTTP(rec, peticionServidorPrueba(http.MethodGet, superficie.ruta, nil))
			resultado := rec.Result()
			defer resultado.Body.Close()
			if !expusoFlusher || expusoUnwrap || !rec.Flushed {
				t.Fatalf("interfaces: Flusher=%t Unwrap=%t, flushed=%t", expusoFlusher, expusoUnwrap, rec.Flushed)
			}
			if nombre, valores, existe := buscarCabeceraInsensible(resultado.Header, "Set-Cookie"); existe {
				t.Fatalf("cookies en cabecera %q: %v", nombre, valores)
			}
			if nombre, valores, existe := buscarCabeceraInsensible(resultado.Trailer, "Set-Cookie"); existe {
				t.Fatalf("cookies en trailer %q: %v", nombre, valores)
			}
			if nombre, valores, existe := buscarCabeceraInsensible(rec.Header(), "Set-Cookie"); existe {
				t.Fatalf("cookies conservadas en %q: %v", nombre, valores)
			}
			for _, declaracion := range resultado.Header.Values("Trailer") {
				if strings.Contains(strings.ToLower(declaracion), "set-cookie") {
					t.Fatalf("declaracion de trailer comprometida: %q", declaracion)
				}
			}
		})
	}
}

func buscarCabeceraInsensible(cabeceras http.Header, buscada string) (string, []string, bool) {
	for nombre, valores := range cabeceras {
		if strings.EqualFold(strings.TrimSpace(nombre), buscada) {
			return nombre, valores, true
		}
	}
	return "", nil, false
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
