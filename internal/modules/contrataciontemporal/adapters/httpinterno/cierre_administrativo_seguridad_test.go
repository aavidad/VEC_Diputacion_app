package httpinterno

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestManejadorCierreAdministrativoRechazaSuperficieAntesDeAutoridad(
	t *testing.T,
) {
	casos := []struct {
		nombre string
		mutar  func(*http.Request)
		estado int
	}{
		{"URL nula", func(r *http.Request) { r.URL = nil }, http.StatusNotFound},
		{"ruta distinta", func(r *http.Request) { r.URL.Path += "/" }, http.StatusNotFound},
		{"query", func(r *http.Request) { r.URL.RawQuery = "organizacion=forjada" }, http.StatusNotFound},
		{"force query", func(r *http.Request) { r.URL.ForceQuery = true }, http.StatusNotFound},
		{"raw path", func(r *http.Request) { r.URL.RawPath = RutaCerrarAdministrativamente }, http.StatusNotFound},
		{"scheme", func(r *http.Request) { r.URL.Scheme = "https" }, http.StatusNotFound},
		{"host URL", func(r *http.Request) { r.URL.Host = "interno.invalid" }, http.StatusNotFound},
		{"userinfo", func(r *http.Request) { r.URL.User = url.User("actor") }, http.StatusNotFound},
		{"opaque", func(r *http.Request) { r.URL.Opaque = RutaCerrarAdministrativamente }, http.StatusNotFound},
		{"fragmento", func(r *http.Request) { r.URL.Fragment = "privado" }, http.StatusNotFound},
		{"fragmento bruto", func(r *http.Request) { r.URL.RawFragment = "privado" }, http.StatusNotFound},
		{"GET", func(r *http.Request) { r.Method = http.MethodGet }, http.StatusMethodNotAllowed},
		{"PUT", func(r *http.Request) { r.Method = http.MethodPut }, http.StatusMethodNotAllowed},
		{"OPTIONS", func(r *http.Request) { r.Method = http.MethodOptions }, http.StatusMethodNotAllowed},
		{"cabecera X", func(r *http.Request) { r.Header.Set("X-Vec-Actor", "forjado") }, http.StatusBadRequest},
		{"authorization", func(r *http.Request) { r.Header.Set("Authorization", "Bearer privado") }, http.StatusBadRequest},
		{"cookie", func(r *http.Request) { r.Header.Set("Cookie", "sesion=privada") }, http.StatusBadRequest},
		{"set cookie", func(r *http.Request) { r.Header.Set("Set-Cookie", "sesion=privada") }, http.StatusBadRequest},
		{"host cabecera", func(r *http.Request) { r.Header.Set("Host", "interno.invalid") }, http.StatusBadRequest},
		{"user agent", func(r *http.Request) { r.Header.Set("User-Agent", "cliente") }, http.StatusBadRequest},
		{"transfer encoding cabecera", func(r *http.Request) { r.Header.Set("Transfer-Encoding", "chunked") }, http.StatusBadRequest},
		{"content length cabecera", func(r *http.Request) { r.Header.Set("Content-Length", "1") }, http.StatusBadRequest},
		{"content encoding", func(r *http.Request) { r.Header.Set("Content-Encoding", "gzip") }, http.StatusBadRequest},
		{"content type ausente", func(r *http.Request) { r.Header.Del("Content-Type") }, http.StatusUnsupportedMediaType},
		{"content type texto", func(r *http.Request) { r.Header.Set("Content-Type", "text/plain") }, http.StatusUnsupportedMediaType},
		{"charset no UTF-8", func(r *http.Request) { r.Header.Set("Content-Type", "application/json; charset=iso-8859-1") }, http.StatusUnsupportedMediaType},
		{"content type duplicado", func(r *http.Request) { r.Header.Add("Content-Type", "application/json") }, http.StatusUnsupportedMediaType},
		{"accept HTML", func(r *http.Request) { r.Header.Set("Accept", "text/html") }, http.StatusNotAcceptable},
		{"accept JSON excluido", func(r *http.Request) { r.Header.Set("Accept", "application/json;q=0, */*;q=1") }, http.StatusNotAcceptable},
		{"transferencia desconocida", func(r *http.Request) { r.TransferEncoding = []string{"gzip"} }, http.StatusBadRequest},
		{"transferencia duplicada", func(r *http.Request) { r.TransferEncoding = []string{"chunked", "chunked"} }, http.StatusBadRequest},
		{"CL y chunked", func(r *http.Request) { r.TransferEncoding = []string{"chunked"} }, http.StatusBadRequest},
		{"trailer", func(r *http.Request) { r.Trailer = http.Header{"X-Actor": []string{"forjado"}} }, http.StatusBadRequest},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			autoridad := autoridadCierreAdministrativoHTTPValidaPrueba()
			ejecutor := &ejecutorCierreAdministrativoHTTPPrueba{}
			peticion := peticionCierreAdministrativoHTTPPrueba(
				t,
				RutaCerrarAdministrativamente,
			)
			caso.mutar(peticion)
			respuesta := httptest.NewRecorder()
			nuevoManejadorCierreAdministrativoHTTPPrueba(
				t,
				autoridad,
				ejecutor,
			).ServeHTTP(respuesta, peticion)
			if respuesta.Code != caso.estado || autoridad.llamadas != 0 ||
				ejecutor.llamadasCerrar != 0 || ejecutor.llamadasReabrir != 0 {
				t.Fatalf(
					"estado=%d llamadas=%d/%d/%d cuerpo=%s",
					respuesta.Code,
					autoridad.llamadas,
					ejecutor.llamadasCerrar,
					ejecutor.llamadasReabrir,
					respuesta.Body,
				)
			}
			if caso.estado == http.StatusMethodNotAllowed &&
				respuesta.Header().Get("Allow") != http.MethodPost {
				t.Fatalf("Allow=%q", respuesta.Header().Get("Allow"))
			}
		})
	}
}

func TestManejadorCierreAdministrativoAdmiteSoloCabecerasPositivas(
	t *testing.T,
) {
	autoridad := autoridadCierreAdministrativoHTTPValidaPrueba()
	ejecutor := &ejecutorCierreAdministrativoHTTPPrueba{}
	peticion := peticionCierreAdministrativoHTTPPrueba(
		t,
		RutaCerrarAdministrativamente,
	)
	peticion.Header = http.Header{
		"content-type": {"application/json; charset=utf-8"},
		"ACCEPT":       {"application/json", "application/*;q=0.5"},
	}
	respuesta := httptest.NewRecorder()
	nuevoManejadorCierreAdministrativoHTTPPrueba(
		t,
		autoridad,
		ejecutor,
	).ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusCreated || autoridad.llamadas != 1 ||
		ejecutor.llamadasCerrar != 1 {
		t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body)
	}

	peticion = peticionCierreAdministrativoHTTPPrueba(
		t,
		RutaCerrarAdministrativamente,
	)
	peticion.TransferEncoding = []string{"chunked"}
	peticion.ContentLength = -1
	respuesta = httptest.NewRecorder()
	nuevoManejadorCierreAdministrativoHTTPPrueba(
		t,
		autoridadCierreAdministrativoHTTPValidaPrueba(),
		&ejecutorCierreAdministrativoHTTPPrueba{},
	).ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusCreated {
		t.Fatalf("chunked no ambiguo: estado=%d cuerpo=%s", respuesta.Code, respuesta.Body)
	}
}

func TestManejadorCierreAdministrativoRechazaJSONNoCerrado(
	t *testing.T,
) {
	valido := cuerpoCierreAdministrativoHTTPPrueba(t)
	casos := []struct {
		nombre string
		cuerpo string
		estado int
	}{
		{"vacio", "", http.StatusBadRequest},
		{"null", "null", http.StatusBadRequest},
		{"coleccion", "[]", http.StatusBadRequest},
		{"segundo JSON", valido + `{}`, http.StatusBadRequest},
		{"trailing", valido + `x`, http.StatusBadRequest},
		{"duplicado", strings.Replace(valido, `"expediente_ref":`, `"expediente_ref":"ref:duplicada","expediente_ref":`, 1), http.StatusBadRequest},
		{"desconocido", strings.TrimSuffix(valido, "}") + `,"estado":"cerrado"}`, http.StatusBadRequest},
		{"organizacion", strings.TrimSuffix(valido, "}") + `,"organizacion_ref":"forjada"}`, http.StatusBadRequest},
		{"actor", strings.TrimSuffix(valido, "}") + `,"actor_ref":"persona:privada"}`, http.StatusBadRequest},
		{"perfil", strings.TrimSuffix(valido, "}") + `,"perfil_ref":"admin"}`, http.StatusBadRequest},
		{"unidad", strings.TrimSuffix(valido, "}") + `,"unidad_ref":"rrhh"}`, http.StatusBadRequest},
		{"autorizacion", strings.TrimSuffix(valido, "}") + `,"autorizacion_ref":"forjada"}`, http.StatusBadRequest},
		{"referencia invalida", strings.Replace(valido, entradaCierreAdministrativoHTTPPrueba().ExpedienteRef, "persona@example.invalid", 1), http.StatusUnprocessableEntity},
		{"UUID invalida", strings.Replace(valido, claveCierreAdministrativoHTTPPrueba, "clave-humana", 1), http.StatusUnprocessableEntity},
		{"transicion invalida", strings.Replace(valido, "cierre_administrativo", "Cerrar Ahora", 1), http.StatusUnprocessableEntity},
		{"version maxima", strings.Replace(valido, `"version_esperada":7`, `"version_esperada":18446744073709551615`, 1), http.StatusUnprocessableEntity},
		{"version decimal", strings.Replace(valido, `"version_esperada":7`, `"version_esperada":7.0`, 1), http.StatusBadRequest},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			autoridad := autoridadCierreAdministrativoHTTPValidaPrueba()
			ejecutor := &ejecutorCierreAdministrativoHTTPPrueba{}
			peticion := httptest.NewRequest(
				http.MethodPost,
				RutaCerrarAdministrativamente,
				strings.NewReader(caso.cuerpo),
			)
			peticion.Header.Set("Content-Type", "application/json")
			peticion.Header.Set("Accept", "application/json")
			respuesta := httptest.NewRecorder()
			nuevoManejadorCierreAdministrativoHTTPPrueba(
				t,
				autoridad,
				ejecutor,
			).ServeHTTP(respuesta, peticion)
			if respuesta.Code != caso.estado || autoridad.llamadas != 0 ||
				ejecutor.llamadasCerrar != 0 {
				t.Fatalf(
					"estado=%d llamadas=%d/%d cuerpo=%s",
					respuesta.Code,
					autoridad.llamadas,
					ejecutor.llamadasCerrar,
					respuesta.Body,
				)
			}
		})
	}
}

func TestManejadorCierreAdministrativoExigeVersionInteroperableAntesDeAutoridad(
	t *testing.T,
) {
	valido := cuerpoCierreAdministrativoHTTPPrueba(t)
	versionBase := `"version_esperada":7`
	maximaPermitida := ports.MaximoEnteroSeguroOperacionAnalisis - 1
	primeraProhibida := ports.MaximoEnteroSeguroOperacionAnalisis
	reemplazarVersion := func(valor string) string {
		return strings.Replace(
			valido,
			versionBase,
			`"version_esperada":`+valor,
			1,
		)
	}
	casos := []struct {
		nombre string
		cuerpo string
		estado int
	}{
		{
			"ausente",
			strings.Replace(valido, versionBase+",", "", 1),
			http.StatusUnprocessableEntity,
		},
		{"null", reemplazarVersion("null"), http.StatusBadRequest},
		{"cero", reemplazarVersion("0"), http.StatusUnprocessableEntity},
		{
			"maximo permitido",
			reemplazarVersion(strconv.FormatUint(maximaPermitida, 10)),
			http.StatusCreated,
		},
		{
			"primer valor prohibido",
			reemplazarVersion(strconv.FormatUint(primeraProhibida, 10)),
			http.StatusUnprocessableEntity,
		},
	}
	rutas := []struct {
		nombre string
		ruta   string
	}{
		{"cerrar", RutaCerrarAdministrativamente},
		{"reabrir", RutaReabrirExcepcionalmente},
	}
	for _, ruta := range rutas {
		for _, caso := range casos {
			t.Run(ruta.nombre+"/"+caso.nombre, func(t *testing.T) {
				autoridad := autoridadCierreAdministrativoHTTPValidaPrueba()
				ejecutor := &ejecutorCierreAdministrativoHTTPPrueba{}
				peticion := httptest.NewRequest(
					http.MethodPost,
					ruta.ruta,
					strings.NewReader(caso.cuerpo),
				)
				peticion.Header.Set("Content-Type", "application/json")
				peticion.Header.Set("Accept", "application/json")
				respuesta := httptest.NewRecorder()
				nuevoManejadorCierreAdministrativoHTTPPrueba(
					t,
					autoridad,
					ejecutor,
				).ServeHTTP(respuesta, peticion)

				llamadasEjecutor := ejecutor.llamadasCerrar +
					ejecutor.llamadasReabrir
				if caso.estado != http.StatusCreated {
					if respuesta.Code != caso.estado ||
						autoridad.llamadas != 0 || llamadasEjecutor != 0 {
						t.Fatalf(
							"estado=%d llamadas=%d/%d cuerpo=%s",
							respuesta.Code,
							autoridad.llamadas,
							llamadasEjecutor,
							respuesta.Body,
						)
					}
					return
				}
				if respuesta.Code != http.StatusCreated ||
					autoridad.llamadas != 1 || llamadasEjecutor != 1 ||
					!strings.Contains(
						respuesta.Body.String(),
						`"version_seguimiento":`+
							strconv.FormatUint(primeraProhibida, 10),
					) {
					t.Fatalf(
						"maximo permitido: estado=%d llamadas=%d/%d cuerpo=%s",
						respuesta.Code,
						autoridad.llamadas,
						llamadasEjecutor,
						respuesta.Body,
					)
				}
			})
		}
	}
}

type lectorVigiladoCierreAdministrativoHTTP struct {
	lecturas int
}

func (l *lectorVigiladoCierreAdministrativoHTTP) Read([]byte) (int, error) {
	l.lecturas++
	return 0, errors.New("el cuerpo no debia leerse")
}

func TestManejadorCierreAdministrativoLimitaAntesDeDecodificar(
	t *testing.T,
) {
	lector := &lectorVigiladoCierreAdministrativoHTTP{}
	peticion := httptest.NewRequest(
		http.MethodPost,
		RutaCerrarAdministrativamente,
		http.NoBody,
	)
	peticion.Body = io.NopCloser(lector)
	peticion.ContentLength = MaximoCuerpoCierreAdministrativoBytes + 1
	peticion.Header.Set("Content-Type", "application/json")
	peticion.Header.Set("Accept", "application/json")
	autoridad := autoridadCierreAdministrativoHTTPValidaPrueba()
	ejecutor := &ejecutorCierreAdministrativoHTTPPrueba{}
	respuesta := httptest.NewRecorder()
	nuevoManejadorCierreAdministrativoHTTPPrueba(
		t,
		autoridad,
		ejecutor,
	).ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusRequestEntityTooLarge || lector.lecturas != 0 ||
		autoridad.llamadas != 0 || ejecutor.llamadasCerrar != 0 {
		t.Fatalf(
			"estado=%d lecturas=%d llamadas=%d/%d cuerpo=%s",
			respuesta.Code,
			lector.lecturas,
			autoridad.llamadas,
			ejecutor.llamadasCerrar,
			respuesta.Body,
		)
	}

	peticion = httptest.NewRequest(
		http.MethodPost,
		RutaCerrarAdministrativamente,
		strings.NewReader(strings.Repeat("x", MaximoCuerpoCierreAdministrativoBytes+2)),
	)
	peticion.ContentLength = -1
	peticion.Header.Set("Content-Type", "application/json")
	peticion.Header.Set("Accept", "application/json")
	respuesta = httptest.NewRecorder()
	nuevoManejadorCierreAdministrativoHTTPPrueba(
		t,
		autoridadCierreAdministrativoHTTPValidaPrueba(),
		&ejecutorCierreAdministrativoHTTPPrueba{},
	).ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("longitud desconocida: estado=%d cuerpo=%s", respuesta.Code, respuesta.Body)
	}
}

type lectorCanceladorCierreAdministrativoHTTP struct {
	lector    *strings.Reader
	cancelar  context.CancelFunc
	cancelado bool
}

func (l *lectorCanceladorCierreAdministrativoHTTP) Read(p []byte) (int, error) {
	n, err := l.lector.Read(p)
	if !l.cancelado && l.lector.Len() == 0 {
		l.cancelado = true
		l.cancelar()
	}
	return n, err
}

func TestManejadorCierreAdministrativoPriorizaCancelacionEnFronteras(
	t *testing.T,
) {
	t.Run("antes de leer", func(t *testing.T) {
		ctx, cancelar := context.WithCancel(context.Background())
		cancelar()
		autoridad := autoridadCierreAdministrativoHTTPValidaPrueba()
		ejecutor := &ejecutorCierreAdministrativoHTTPPrueba{}
		respuesta := httptest.NewRecorder()
		nuevoManejadorCierreAdministrativoHTTPPrueba(
			t,
			autoridad,
			ejecutor,
		).ServeHTTP(
			respuesta,
			peticionCierreAdministrativoHTTPPrueba(
				t,
				RutaCerrarAdministrativamente,
			).WithContext(ctx),
		)
		if respuesta.Code != http.StatusRequestTimeout || autoridad.llamadas != 0 ||
			ejecutor.llamadasCerrar != 0 {
			t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body)
		}
	})

	t.Run("despues de decodificar", func(t *testing.T) {
		ctx, cancelar := context.WithCancel(context.Background())
		lector := &lectorCanceladorCierreAdministrativoHTTP{
			lector:   strings.NewReader(cuerpoCierreAdministrativoHTTPPrueba(t)),
			cancelar: cancelar,
		}
		peticion := httptest.NewRequest(
			http.MethodPost,
			RutaCerrarAdministrativamente,
			lector,
		).WithContext(ctx)
		peticion.Header.Set("Content-Type", "application/json")
		peticion.Header.Set("Accept", "application/json")
		autoridad := autoridadCierreAdministrativoHTTPValidaPrueba()
		ejecutor := &ejecutorCierreAdministrativoHTTPPrueba{}
		respuesta := httptest.NewRecorder()
		nuevoManejadorCierreAdministrativoHTTPPrueba(
			t,
			autoridad,
			ejecutor,
		).ServeHTTP(respuesta, peticion)
		if respuesta.Code != http.StatusRequestTimeout || autoridad.llamadas != 0 ||
			ejecutor.llamadasCerrar != 0 {
			t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body)
		}
	})

	t.Run("autoridad", func(t *testing.T) {
		ctx, cancelar := context.WithCancel(context.Background())
		autoridad := autoridadCierreAdministrativoHTTPValidaPrueba()
		autoridad.err = errors.New("detalle autoridad")
		autoridad.durante = func(context.Context) { cancelar() }
		ejecutor := &ejecutorCierreAdministrativoHTTPPrueba{}
		respuesta := httptest.NewRecorder()
		nuevoManejadorCierreAdministrativoHTTPPrueba(
			t,
			autoridad,
			ejecutor,
		).ServeHTTP(
			respuesta,
			peticionCierreAdministrativoHTTPPrueba(
				t,
				RutaCerrarAdministrativamente,
			).WithContext(ctx),
		)
		if respuesta.Code != http.StatusRequestTimeout || autoridad.llamadas != 1 ||
			ejecutor.llamadasCerrar != 0 {
			t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body)
		}
	})

	t.Run("servicio", func(t *testing.T) {
		ctx, cancelar := context.WithCancel(context.Background())
		ejecutor := &ejecutorCierreAdministrativoHTTPPrueba{
			durante: func(context.Context) { cancelar() },
		}
		autoridad := autoridadCierreAdministrativoHTTPValidaPrueba()
		respuesta := httptest.NewRecorder()
		nuevoManejadorCierreAdministrativoHTTPPrueba(
			t,
			autoridad,
			ejecutor,
		).ServeHTTP(
			respuesta,
			peticionCierreAdministrativoHTTPPrueba(
				t,
				RutaCerrarAdministrativamente,
			).WithContext(ctx),
		)
		if respuesta.Code != http.StatusRequestTimeout || autoridad.llamadas != 1 ||
			ejecutor.llamadasCerrar != 1 {
			t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body)
		}
	})
}

func TestManejadorCierreAdministrativoFallaCerradoSinAutoridadValida(
	t *testing.T,
) {
	casos := []struct {
		nombre       string
		organizacion string
		err          error
		estado       int
	}{
		{"organizacion invalida", "organizacion humana", nil, http.StatusServiceUnavailable},
		{"autoridad no disponible", referenciaCierreAdministrativoHTTPPrueba("organizacion"), errors.New("detalle privado"), http.StatusServiceUnavailable},
		{"autoridad cancelada", referenciaCierreAdministrativoHTTPPrueba("organizacion"), context.Canceled, http.StatusRequestTimeout},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			autoridad := &autoridadCierreAdministrativoHTTPPrueba{
				organizacionRef: caso.organizacion,
				err:             caso.err,
			}
			ejecutor := &ejecutorCierreAdministrativoHTTPPrueba{}
			respuesta := httptest.NewRecorder()
			nuevoManejadorCierreAdministrativoHTTPPrueba(
				t,
				autoridad,
				ejecutor,
			).ServeHTTP(
				respuesta,
				peticionCierreAdministrativoHTTPPrueba(
					t,
					RutaCerrarAdministrativamente,
				),
			)
			if respuesta.Code != caso.estado || autoridad.llamadas != 1 ||
				ejecutor.llamadasCerrar != 0 ||
				strings.Contains(respuesta.Body.String(), "detalle privado") {
				t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body)
			}
		})
	}
}

func TestManejadorCierreAdministrativoNoEmiteCookiesNiCache(
	t *testing.T,
) {
	respuesta := httptest.NewRecorder()
	respuesta.Header().Set("Set-Cookie", "sesion=privada")
	respuesta.Header().Set("Location", "https://privado.invalid")
	respuesta.Header().Set("Retry-After", "1")
	respuesta.Header().Set("Access-Control-Allow-Origin", "*")
	nuevoManejadorCierreAdministrativoHTTPPrueba(
		t,
		autoridadCierreAdministrativoHTTPValidaPrueba(),
		&ejecutorCierreAdministrativoHTTPPrueba{},
	).ServeHTTP(
		respuesta,
		peticionCierreAdministrativoHTTPPrueba(
			t,
			RutaCerrarAdministrativamente,
		),
	)
	if respuesta.Code != http.StatusCreated ||
		respuesta.Header().Get("Content-Type") != "application/json; charset=utf-8" ||
		respuesta.Header().Get("Cache-Control") != "no-store, no-transform" ||
		respuesta.Header().Get("Pragma") != "no-cache" ||
		respuesta.Header().Get("Expires") != "0" {
		t.Fatalf("cabeceras obligatorias ausentes: %#v", respuesta.Header())
	}
	for _, prohibida := range []string{
		"Set-Cookie", "Location", "Retry-After", "Access-Control-Allow-Origin",
	} {
		if respuesta.Header().Get(prohibida) != "" {
			t.Fatalf("cabecera %q emitida: %#v", prohibida, respuesta.Header())
		}
	}
}
