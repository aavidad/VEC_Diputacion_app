package httpinterno

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func peticionPropuestaFormalizacionConCuerpo(
	cuerpo string,
) *http.Request {
	peticion := httptest.NewRequest(
		http.MethodPost,
		RutaPropuestaFormalizacion,
		strings.NewReader(cuerpo),
	)
	peticion.Header.Set("Content-Type", "application/json; charset=utf-8")
	peticion.Header.Set("Accept", "application/json")
	return peticion
}

func codificarEntradaPropuestaFormalizacionHTTPPrueba(
	t *testing.T,
	entrada propuestaFormalizacionEntradaJSON,
) string {
	t.Helper()
	contenido, err := json.Marshal(entrada)
	if err != nil {
		t.Fatalf("codificar entrada: %v", err)
	}
	return string(contenido)
}

func TestManejadorPropuestaFormalizacionRechazaSuperficieHostilAntesDeAutoridad(
	t *testing.T,
) {
	t.Parallel()
	casos := []struct {
		nombre string
		mutar  func(*http.Request)
		estado int
	}{
		{"URL nula", func(r *http.Request) { r.URL = nil }, http.StatusNotFound},
		{"ruta distinta", func(r *http.Request) { r.URL.Path += "/" }, http.StatusNotFound},
		{"query", func(r *http.Request) { r.URL.RawQuery = "actor=privado" }, http.StatusNotFound},
		{"force query", func(r *http.Request) { r.URL.ForceQuery = true }, http.StatusNotFound},
		{"raw path", func(r *http.Request) { r.URL.RawPath = RutaPropuestaFormalizacion }, http.StatusNotFound},
		{"scheme", func(r *http.Request) { r.URL.Scheme = "https" }, http.StatusNotFound},
		{"host", func(r *http.Request) { r.URL.Host = "interno.invalid" }, http.StatusNotFound},
		{"usuario URL", func(r *http.Request) { r.URL.User = url.User("persona") }, http.StatusNotFound},
		{"opaque", func(r *http.Request) { r.URL.Opaque = RutaPropuestaFormalizacion }, http.StatusNotFound},
		{"fragmento", func(r *http.Request) { r.URL.Fragment = "perfil" }, http.StatusNotFound},
		{"fragmento bruto", func(r *http.Request) { r.URL.RawFragment = "perfil" }, http.StatusNotFound},
		{"get", func(r *http.Request) { r.Method = http.MethodGet }, http.StatusMethodNotAllowed},
		{"put", func(r *http.Request) { r.Method = http.MethodPut }, http.StatusMethodNotAllowed},
		{"options", func(r *http.Request) { r.Method = http.MethodOptions }, http.StatusMethodNotAllowed},
		{"tipo texto", func(r *http.Request) { r.Header.Set("Content-Type", "text/plain") }, http.StatusUnsupportedMediaType},
		{"charset no UTF-8", func(r *http.Request) { r.Header.Set("Content-Type", "application/json; charset=iso-8859-1") }, http.StatusUnsupportedMediaType},
		{"tipo duplicado", func(r *http.Request) { r.Header.Add("Content-Type", "application/json") }, http.StatusUnsupportedMediaType},
		{"accept HTML", func(r *http.Request) { r.Header.Set("Accept", "text/html") }, http.StatusNotAcceptable},
		{"accept JSON excluido", func(r *http.Request) { r.Header.Set("Accept", "application/json;q=0, */*;q=1") }, http.StatusNotAcceptable},
		{"cookie", func(r *http.Request) { r.Header.Set("Cookie", "sesion=privada") }, http.StatusBadRequest},
		{"authorization", func(r *http.Request) { r.Header.Set("Authorization", "Bearer privado") }, http.StatusBadRequest},
		{"organizacion", func(r *http.Request) { r.Header.Set("X-Organizacion", "organizacion:forjada") }, http.StatusBadRequest},
		{"identidad", func(r *http.Request) { r.Header.Set("X-Vec-Actor", "persona:privada") }, http.StatusBadRequest},
		{"rol", func(r *http.Request) { r.Header.Set("X-Role", "admin") }, http.StatusBadRequest},
		{"idempotencia cabecera", func(r *http.Request) { r.Header.Set("Idempotency-Key", clavePropuestaFormalizacionHTTPPrueba) }, http.StatusBadRequest},
		{"metodo alternativo", func(r *http.Request) { r.Header.Set("X-HTTP-Method-Override", "GET") }, http.StatusBadRequest},
		{"compresion", func(r *http.Request) { r.Header.Set("Content-Encoding", "gzip") }, http.StatusBadRequest},
		{"transferencia", func(r *http.Request) { r.TransferEncoding = []string{"gzip"} }, http.StatusBadRequest},
		{"trailer", func(r *http.Request) {
			r.Trailer = make(http.Header)
			r.Trailer.Set("X-Actor", "persona")
		}, http.StatusBadRequest},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			autoridad := autoridadPropuestaFormalizacionHTTPValidaPrueba()
			ejecutor := ejecutorPropuestaFormalizacionHTTPValidoPrueba()
			manejador := nuevoManejadorPropuestaFormalizacionHTTPPrueba(
				t,
				autoridad,
				ejecutor,
			)
			peticion := peticionPropuestaFormalizacionHTTPPrueba(t)
			caso.mutar(peticion)
			respuesta := httptest.NewRecorder()
			manejador.ServeHTTP(respuesta, peticion)
			if respuesta.Code != caso.estado || autoridad.total() != 0 ||
				ejecutor.total() != 0 {
				t.Fatalf(
					"estado=%d cuerpo=%s llamadas=%d/%d",
					respuesta.Code,
					respuesta.Body,
					autoridad.total(),
					ejecutor.total(),
				)
			}
			if caso.estado == http.StatusMethodNotAllowed &&
				respuesta.Header().Get("Allow") != http.MethodPost {
				t.Fatalf("Allow=%q", respuesta.Header().Get("Allow"))
			}
		})
	}
}

func TestManejadorPropuestaFormalizacionExigeJSONCanonicoCerrado(
	t *testing.T,
) {
	t.Parallel()
	valido := cuerpoPropuestaFormalizacionHTTPPrueba(t)
	nuloAnexos := entradaPropuestaFormalizacionHTTPPrueba()
	nuloAnexos.Anexos = nil
	ordenDistinto := strings.Replace(
		valido,
		`{"clave_idempotencia":"`+clavePropuestaFormalizacionHTTPPrueba+
			`","expediente_ref":"expediente:http-formalizacion"`,
		`{"expediente_ref":"expediente:http-formalizacion",`+
			`"clave_idempotencia":"`+clavePropuestaFormalizacionHTTPPrueba+`"`,
		1,
	)
	casos := []struct {
		nombre string
		cuerpo string
		estado int
	}{
		{"vacio", "", http.StatusBadRequest},
		{"objeto vacio", `{}`, http.StatusUnprocessableEntity},
		{"null", `null`, http.StatusBadRequest},
		{"cadena", `"persona"`, http.StatusBadRequest},
		{"coleccion", `[]`, http.StatusBadRequest},
		{"organizacion en cuerpo", strings.TrimSuffix(valido, "}") + `,"organizacion_ref":"organizacion:forjada"}`, http.StatusBadRequest},
		{"actor en cuerpo", strings.TrimSuffix(valido, "}") + `,"actor_ref":"persona:forjada"}`, http.StatusBadRequest},
		{"campo desconocido", strings.TrimSuffix(valido, "}") + `,"estado":"firmada"}`, http.StatusBadRequest},
		{"campo duplicado", strings.Replace(valido, `"expediente_ref":`, `"expediente_ref":"expediente:otra","expediente_ref":`, 1), http.StatusBadRequest},
		{"duplicado anidado", strings.Replace(valido, `"version":7`, `"version":7,"version":7`, 1), http.StatusBadRequest},
		{"segundo JSON", valido + `{}`, http.StatusBadRequest},
		{"orden distinto", ordenDistinto, http.StatusUnprocessableEntity},
		{"espacio exterior", " " + valido, http.StatusUnprocessableEntity},
		{"espacio interior", strings.Replace(valido, `":13`, `": 13`, 1), http.StatusUnprocessableEntity},
		{"clave escapada", strings.Replace(valido, "clave_idempotencia", `clave\u005fidempotencia`, 1), http.StatusUnprocessableEntity},
		{"UUID mayuscula", strings.Replace(valido, clavePropuestaFormalizacionHTTPPrueba, strings.ToUpper(clavePropuestaFormalizacionHTTPPrueba), 1), http.StatusUnprocessableEntity},
		{"version cero", strings.Replace(valido, `"version_esperada":13`, `"version_esperada":0`, 1), http.StatusUnprocessableEntity},
		{"version maxima", strings.Replace(valido, `"version_esperada":13`, `"version_esperada":9007199254740991`, 1), http.StatusUnprocessableEntity},
		{"version decimal", strings.Replace(valido, `"version_esperada":13`, `"version_esperada":13.0`, 1), http.StatusBadRequest},
		{"referencia directa", strings.Replace(valido, "expediente:http-formalizacion", "persona@example.invalid", 1), http.StatusUnprocessableEntity},
		{"anexos nulos", codificarEntradaPropuestaFormalizacionHTTPPrueba(t, nuloAnexos), http.StatusBadRequest},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			autoridad := autoridadPropuestaFormalizacionHTTPValidaPrueba()
			ejecutor := ejecutorPropuestaFormalizacionHTTPValidoPrueba()
			manejador := nuevoManejadorPropuestaFormalizacionHTTPPrueba(
				t,
				autoridad,
				ejecutor,
			)
			respuesta := httptest.NewRecorder()
			manejador.ServeHTTP(
				respuesta,
				peticionPropuestaFormalizacionConCuerpo(caso.cuerpo),
			)
			if respuesta.Code != caso.estado || ejecutor.total() != 0 {
				t.Fatalf(
					"estado=%d cuerpo=%s autoridad=%d ejecutor=%d",
					respuesta.Code,
					respuesta.Body,
					autoridad.total(),
					ejecutor.total(),
				)
			}
		})
	}
}

type lectorVigiladoPropuestaFormalizacionHTTP struct {
	lecturas int
}

func (l *lectorVigiladoPropuestaFormalizacionHTTP) Read([]byte) (int, error) {
	l.lecturas++
	return 0, errors.New("el cuerpo no debia leerse")
}

func TestManejadorPropuestaFormalizacionLimitaAntesDeLeerOInvocar(
	t *testing.T,
) {
	t.Parallel()
	autoridad := autoridadPropuestaFormalizacionHTTPValidaPrueba()
	ejecutor := ejecutorPropuestaFormalizacionHTTPValidoPrueba()
	manejador := nuevoManejadorPropuestaFormalizacionHTTPPrueba(
		t,
		autoridad,
		ejecutor,
	)
	lector := &lectorVigiladoPropuestaFormalizacionHTTP{}
	peticion := httptest.NewRequest(
		http.MethodPost,
		RutaPropuestaFormalizacion,
		http.NoBody,
	)
	peticion.Body = io.NopCloser(lector)
	peticion.ContentLength = MaximoCuerpoPropuestaFormalizacionBytes + 1
	peticion.Header.Set("Content-Type", "application/json")
	peticion.Header.Set("Accept", "application/json")
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusRequestEntityTooLarge || lector.lecturas != 0 ||
		autoridad.total() != 0 || ejecutor.total() != 0 {
		t.Fatalf(
			"estado=%d lecturas=%d llamadas=%d/%d cuerpo=%s",
			respuesta.Code,
			lector.lecturas,
			autoridad.total(),
			ejecutor.total(),
			respuesta.Body,
		)
	}
}

func TestManejadorPropuestaFormalizacionLimitaTransferenciaSinLongitud(
	t *testing.T,
) {
	t.Parallel()
	autoridad := autoridadPropuestaFormalizacionHTTPValidaPrueba()
	ejecutor := ejecutorPropuestaFormalizacionHTTPValidoPrueba()
	manejador := nuevoManejadorPropuestaFormalizacionHTTPPrueba(
		t,
		autoridad,
		ejecutor,
	)
	peticion := peticionPropuestaFormalizacionConCuerpo(
		strings.Repeat("x", MaximoCuerpoPropuestaFormalizacionBytes+2),
	)
	peticion.ContentLength = -1
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusRequestEntityTooLarge ||
		autoridad.total() != 0 || ejecutor.total() != 0 {
		t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body)
	}
}

func TestManejadorPropuestaFormalizacionAdmiteMaximoDeAnexosDelPuerto(
	t *testing.T,
) {
	t.Parallel()
	entrada := entradaPropuestaFormalizacionHTTPPrueba()
	anexos := make(
		[]anexoPropuestaFormalizacionJSON,
		ports.MaximoAnexosPropuestaFormalizacion,
	)
	for indice := range anexos {
		anexos[indice] = anexoPropuestaFormalizacionHTTPPrueba(
			fmt.Sprintf("anexo:http-%03d", indice),
			fmt.Sprintf("%x", indice%15+1),
			1,
		)
	}
	entrada.Anexos = &anexos
	cuerpo := codificarEntradaPropuestaFormalizacionHTTPPrueba(t, entrada)
	if len(cuerpo) > MaximoCuerpoPropuestaFormalizacionBytes {
		t.Fatalf("el maximo contractual no cabe: %d", len(cuerpo))
	}
	autoridad := autoridadPropuestaFormalizacionHTTPValidaPrueba()
	ejecutor := ejecutorPropuestaFormalizacionHTTPValidoPrueba()
	manejador := nuevoManejadorPropuestaFormalizacionHTTPPrueba(
		t,
		autoridad,
		ejecutor,
	)
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(
		respuesta,
		peticionPropuestaFormalizacionConCuerpo(cuerpo),
	)
	solicitud, existe := ejecutor.ultima()
	if respuesta.Code != http.StatusCreated || !existe ||
		len(solicitud.Anexos) != ports.MaximoAnexosPropuestaFormalizacion ||
		autoridad.total() != 1 || ejecutor.total() != 1 {
		t.Fatalf(
			"estado=%d anexos=%d llamadas=%d/%d cuerpo=%s",
			respuesta.Code,
			len(solicitud.Anexos),
			autoridad.total(),
			ejecutor.total(),
			respuesta.Body,
		)
	}
}

func TestManejadorPropuestaFormalizacionRechazaLimitesSemanticosAntesDelServicio(
	t *testing.T,
) {
	t.Parallel()
	casos := []struct {
		nombre   string
		preparar func(*propuestaFormalizacionEntradaJSON)
	}{
		{
			"cardinalidad",
			func(e *propuestaFormalizacionEntradaJSON) {
				anexos := make(
					[]anexoPropuestaFormalizacionJSON,
					ports.MaximoAnexosPropuestaFormalizacion+1,
				)
				for indice := range anexos {
					anexos[indice] = anexoPropuestaFormalizacionHTTPPrueba(
						fmt.Sprintf("anexo:exceso-%03d", indice),
						"a",
						1,
					)
				}
				e.Anexos = &anexos
			},
		},
		{
			"suma declarada",
			func(e *propuestaFormalizacionEntradaJSON) {
				anexos := []anexoPropuestaFormalizacionJSON{
					anexoPropuestaFormalizacionHTTPPrueba(
						"anexo:limite",
						"a",
						ports.MaximoBytesAnexosPropuestaFormalizacion,
					),
					anexoPropuestaFormalizacionHTTPPrueba(
						"anexo:extra",
						"b",
						1,
					),
				}
				e.Anexos = &anexos
			},
		},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			entrada := entradaPropuestaFormalizacionHTTPPrueba()
			caso.preparar(&entrada)
			autoridad := autoridadPropuestaFormalizacionHTTPValidaPrueba()
			ejecutor := ejecutorPropuestaFormalizacionHTTPValidoPrueba()
			manejador := nuevoManejadorPropuestaFormalizacionHTTPPrueba(
				t,
				autoridad,
				ejecutor,
			)
			respuesta := httptest.NewRecorder()
			manejador.ServeHTTP(
				respuesta,
				peticionPropuestaFormalizacionConCuerpo(
					codificarEntradaPropuestaFormalizacionHTTPPrueba(t, entrada),
				),
			)
			if respuesta.Code != http.StatusUnprocessableEntity ||
				ejecutor.total() != 0 {
				t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body)
			}
		})
	}
}

type lectorCanceladorPropuestaFormalizacionHTTP struct {
	lector    *strings.Reader
	cancelar  context.CancelFunc
	cancelado bool
}

func (l *lectorCanceladorPropuestaFormalizacionHTTP) Read(p []byte) (int, error) {
	n, err := l.lector.Read(p)
	if !l.cancelado && l.lector.Len() == 0 {
		l.cancelado = true
		l.cancelar()
	}
	return n, err
}

func TestManejadorPropuestaFormalizacionPriorizaCancelacionTrasDecodificar(
	t *testing.T,
) {
	t.Parallel()
	ctx, cancelar := context.WithCancel(context.Background())
	defer cancelar()
	lector := &lectorCanceladorPropuestaFormalizacionHTTP{
		lector:   strings.NewReader(cuerpoPropuestaFormalizacionHTTPPrueba(t)),
		cancelar: cancelar,
	}
	peticion := httptest.NewRequest(
		http.MethodPost,
		RutaPropuestaFormalizacion,
		lector,
	).WithContext(ctx)
	peticion.Header.Set("Content-Type", "application/json")
	peticion.Header.Set("Accept", "application/json")
	autoridad := autoridadPropuestaFormalizacionHTTPValidaPrueba()
	ejecutor := ejecutorPropuestaFormalizacionHTTPValidoPrueba()
	manejador := nuevoManejadorPropuestaFormalizacionHTTPPrueba(
		t,
		autoridad,
		ejecutor,
	)
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusRequestTimeout || autoridad.total() != 0 ||
		ejecutor.total() != 0 {
		t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body)
	}
}

func TestManejadorPropuestaFormalizacionPriorizaCancelacionDespuesDeFronteras(
	t *testing.T,
) {
	t.Parallel()
	t.Run("autoridad", func(t *testing.T) {
		ctx, cancelar := context.WithCancel(context.Background())
		defer cancelar()
		autoridad := autoridadPropuestaFormalizacionHTTPValidaPrueba()
		autoridad.err = errors.New("detalle autoridad")
		autoridad.durante = func(context.Context) { cancelar() }
		ejecutor := ejecutorPropuestaFormalizacionHTTPValidoPrueba()
		manejador := nuevoManejadorPropuestaFormalizacionHTTPPrueba(
			t,
			autoridad,
			ejecutor,
		)
		respuesta := httptest.NewRecorder()
		manejador.ServeHTTP(
			respuesta,
			peticionPropuestaFormalizacionHTTPPrueba(t).WithContext(ctx),
		)
		if respuesta.Code != http.StatusRequestTimeout || autoridad.total() != 1 ||
			ejecutor.total() != 0 {
			t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body)
		}
	})

	t.Run("error de autoridad", func(t *testing.T) {
		autoridad := autoridadPropuestaFormalizacionHTTPValidaPrueba()
		autoridad.err = context.Canceled
		ejecutor := ejecutorPropuestaFormalizacionHTTPValidoPrueba()
		manejador := nuevoManejadorPropuestaFormalizacionHTTPPrueba(
			t,
			autoridad,
			ejecutor,
		)
		respuesta := httptest.NewRecorder()
		manejador.ServeHTTP(
			respuesta,
			peticionPropuestaFormalizacionHTTPPrueba(t),
		)
		if respuesta.Code != http.StatusRequestTimeout || autoridad.total() != 1 ||
			ejecutor.total() != 0 {
			t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body)
		}
	})

	t.Run("servicio", func(t *testing.T) {
		ctx, cancelar := context.WithCancel(context.Background())
		defer cancelar()
		ejecutor := ejecutorPropuestaFormalizacionHTTPValidoPrueba()
		ejecutor.durante = func(context.Context) { cancelar() }
		autoridad := autoridadPropuestaFormalizacionHTTPValidaPrueba()
		manejador := nuevoManejadorPropuestaFormalizacionHTTPPrueba(
			t,
			autoridad,
			ejecutor,
		)
		respuesta := httptest.NewRecorder()
		manejador.ServeHTTP(
			respuesta,
			peticionPropuestaFormalizacionHTTPPrueba(t).WithContext(ctx),
		)
		if respuesta.Code != http.StatusRequestTimeout || ejecutor.total() != 1 ||
			bytes.Contains(respuesta.Body.Bytes(), []byte("propuesta:http-local")) {
			t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body)
		}
	})
}

func TestManejadorPropuestaFormalizacionCancelacionPreviaYPlazoNoInvocan(
	t *testing.T,
) {
	t.Parallel()
	casos := []struct {
		nombre string
		ctx    func() (context.Context, context.CancelFunc)
		estado int
	}{
		{
			"cancelacion",
			func() (context.Context, context.CancelFunc) {
				ctx, cancelar := context.WithCancel(context.Background())
				cancelar()
				return ctx, func() {}
			},
			http.StatusRequestTimeout,
		},
		{
			"plazo",
			func() (context.Context, context.CancelFunc) {
				return context.WithDeadline(
					context.Background(),
					time.Now().Add(-time.Second),
				)
			},
			http.StatusGatewayTimeout,
		},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			ctx, cancelar := caso.ctx()
			defer cancelar()
			autoridad := autoridadPropuestaFormalizacionHTTPValidaPrueba()
			ejecutor := ejecutorPropuestaFormalizacionHTTPValidoPrueba()
			manejador := nuevoManejadorPropuestaFormalizacionHTTPPrueba(
				t,
				autoridad,
				ejecutor,
			)
			respuesta := httptest.NewRecorder()
			manejador.ServeHTTP(
				respuesta,
				peticionPropuestaFormalizacionHTTPPrueba(t).WithContext(ctx),
			)
			if respuesta.Code != caso.estado || autoridad.total() != 0 ||
				ejecutor.total() != 0 {
				t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body)
			}
		})
	}
}

func TestManejadorPropuestaFormalizacionFallaCerradoConAutoridadONulosEnEjecucion(
	t *testing.T,
) {
	t.Parallel()
	t.Run("autoridad invalida", func(t *testing.T) {
		autoridad := autoridadPropuestaFormalizacionHTTPValidaPrueba()
		autoridad.contexto.OrganizacionRef = "persona@example.invalid"
		ejecutor := ejecutorPropuestaFormalizacionHTTPValidoPrueba()
		manejador := nuevoManejadorPropuestaFormalizacionHTTPPrueba(
			t,
			autoridad,
			ejecutor,
		)
		respuesta := httptest.NewRecorder()
		manejador.ServeHTTP(respuesta, peticionPropuestaFormalizacionHTTPPrueba(t))
		if respuesta.Code != http.StatusServiceUnavailable || ejecutor.total() != 0 {
			t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body)
		}
	})

	t.Run("nulo tipado", func(t *testing.T) {
		autoridad := autoridadPropuestaFormalizacionHTTPValidaPrueba()
		ejecutor := ejecutorPropuestaFormalizacionHTTPValidoPrueba()
		manejador := nuevoManejadorPropuestaFormalizacionHTTPPrueba(
			t,
			autoridad,
			ejecutor,
		)
		concreto := manejador.(*manejadorPropuestaFormalizacion)
		var nulo *ejecutorPropuestaFormalizacionHTTPPrueba
		concreto.ejecutor = nulo
		respuesta := httptest.NewRecorder()
		manejador.ServeHTTP(respuesta, peticionPropuestaFormalizacionHTTPPrueba(t))
		if respuesta.Code != http.StatusServiceUnavailable || autoridad.total() != 0 {
			t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body)
		}
	})

	t.Run("peticion nula", func(t *testing.T) {
		manejador := nuevoManejadorPropuestaFormalizacionHTTPPrueba(
			t,
			autoridadPropuestaFormalizacionHTTPValidaPrueba(),
			ejecutorPropuestaFormalizacionHTTPValidoPrueba(),
		)
		respuesta := httptest.NewRecorder()
		manejador.ServeHTTP(respuesta, nil)
		if respuesta.Code != http.StatusServiceUnavailable {
			t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body)
		}
	})
}

func TestManejadorPropuestaFormalizacionSaneaCabecerasDeTodaRespuesta(
	t *testing.T,
) {
	t.Parallel()
	manejador := nuevoManejadorPropuestaFormalizacionHTTPPrueba(
		t,
		autoridadPropuestaFormalizacionHTTPValidaPrueba(),
		ejecutorPropuestaFormalizacionHTTPValidoPrueba(),
	)
	respuesta := httptest.NewRecorder()
	respuesta.Header().Set("Set-Cookie", "sesion=privada")
	respuesta.Header().Set("Retry-After", "1")
	respuesta.Header().Set("Location", "https://privado.invalid")
	manejador.ServeHTTP(respuesta, peticionPropuestaFormalizacionHTTPPrueba(t))
	if respuesta.Code != http.StatusCreated ||
		respuesta.Header().Get("Content-Type") != "application/json; charset=utf-8" ||
		respuesta.Header().Get("Cache-Control") != "no-store, no-transform" ||
		respuesta.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("cabeceras obligatorias ausentes: %#v", respuesta.Header())
	}
	for _, prohibida := range []string{"Set-Cookie", "Retry-After", "Location"} {
		if respuesta.Header().Get(prohibida) != "" {
			t.Fatalf("cabecera %q emitida: %#v", prohibida, respuesta.Header())
		}
	}
}

var _ io.Reader = (*lectorCanceladorPropuestaFormalizacionHTTP)(nil)
