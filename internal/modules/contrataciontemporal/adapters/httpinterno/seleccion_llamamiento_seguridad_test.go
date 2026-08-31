package httpinterno

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func peticionSeleccionLlamamientoConCuerpo(cuerpo string) *http.Request {
	peticion := httptest.NewRequest(
		http.MethodPost,
		RutaSeleccionLlamamiento,
		strings.NewReader(cuerpo),
	)
	peticion.Header.Set("Content-Type", "application/json; charset=utf-8")
	peticion.Header.Set("Accept", "application/json")
	return peticion
}

func TestManejadorSeleccionLlamamientoRechazaSuperficieHostilAntesDelNegocio(
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
		{"raw path", func(r *http.Request) { r.URL.RawPath = RutaSeleccionLlamamiento }, http.StatusNotFound},
		{"scheme", func(r *http.Request) { r.URL.Scheme = "https" }, http.StatusNotFound},
		{"host", func(r *http.Request) { r.URL.Host = "interno.invalid" }, http.StatusNotFound},
		{"usuario URL", func(r *http.Request) { r.URL.User = url.User("persona") }, http.StatusNotFound},
		{"opaque", func(r *http.Request) { r.URL.Opaque = RutaSeleccionLlamamiento }, http.StatusNotFound},
		{"fragmento", func(r *http.Request) { r.URL.Fragment = "perfil" }, http.StatusNotFound},
		{"fragmento bruto", func(r *http.Request) { r.URL.RawFragment = "perfil" }, http.StatusNotFound},
		{"get", func(r *http.Request) { r.Method = http.MethodGet }, http.StatusMethodNotAllowed},
		{"options", func(r *http.Request) { r.Method = http.MethodOptions }, http.StatusMethodNotAllowed},
		{"tipo texto", func(r *http.Request) { r.Header.Set("Content-Type", "text/plain") }, http.StatusUnsupportedMediaType},
		{"charset no UTF-8", func(r *http.Request) { r.Header.Set("Content-Type", "application/json; charset=iso-8859-1") }, http.StatusUnsupportedMediaType},
		{"accept HTML", func(r *http.Request) { r.Header.Set("Accept", "text/html") }, http.StatusNotAcceptable},
		{"accept JSON excluido", func(r *http.Request) { r.Header.Set("Accept", "application/json;q=0, */*;q=1") }, http.StatusNotAcceptable},
		{"cookie", func(r *http.Request) { r.Header.Set("Cookie", "sesion=privada") }, http.StatusBadRequest},
		{"authorization", func(r *http.Request) { r.Header.Set("Authorization", "Bearer privado") }, http.StatusBadRequest},
		{"identidad", func(r *http.Request) { r.Header.Set("X-Vec-Actor", "persona:privada") }, http.StatusBadRequest},
		{"rol", func(r *http.Request) { r.Header.Set("X-Role", "admin") }, http.StatusBadRequest},
		{"idempotencia cabecera", func(r *http.Request) { r.Header.Set("Idempotency-Key", claveSeleccionLlamamientoHTTPPrueba) }, http.StatusBadRequest},
		{"metodo alternativo", func(r *http.Request) { r.Header.Set("X-HTTP-Method-Override", "GET") }, http.StatusBadRequest},
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
			ejecutor := &ejecutorSeleccionLlamamientoHTTPPrueba{
				recibo: reciboSeleccionLlamamientoHTTPPrueba(),
			}
			manejador := nuevoManejadorSeleccionLlamamientoHTTPPrueba(t, ejecutor)
			peticion := nuevaPeticionSeleccionLlamamientoHTTPPrueba()
			caso.mutar(peticion)
			respuesta := httptest.NewRecorder()
			manejador.ServeHTTP(respuesta, peticion)
			if respuesta.Code != caso.estado || ejecutor.total() != 0 {
				t.Fatalf(
					"estado=%d cuerpo=%s llamadas=%d",
					respuesta.Code,
					respuesta.Body,
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

func TestManejadorSeleccionLlamamientoRechazaJSONNoCerradoONoCanonico(
	t *testing.T,
) {
	t.Parallel()
	valida := `{"clave_idempotencia":"` +
		claveSeleccionLlamamientoHTTPPrueba + `"}`
	casos := []struct {
		nombre string
		cuerpo string
		estado int
	}{
		{"vacio", "", http.StatusBadRequest},
		{"objeto vacio", `{}`, http.StatusUnprocessableEntity},
		{"null", `null`, http.StatusBadRequest},
		{"cadena", `"` + claveSeleccionLlamamientoHTTPPrueba + `"`, http.StatusBadRequest},
		{"campo desconocido", `{"clave_idempotencia":"` + claveSeleccionLlamamientoHTTPPrueba + `","actor":"privado"}`, http.StatusBadRequest},
		{"campo duplicado", `{"clave_idempotencia":"` + claveSeleccionLlamamientoHTTPPrueba + `","clave_idempotencia":"` + claveSeleccionLlamamientoHTTPPrueba + `"}`, http.StatusBadRequest},
		{"segundo JSON", valida + `{}`, http.StatusBadRequest},
		{"UUID mayuscula", `{"clave_idempotencia":"4D36E96E-E325-4F9B-BEBC-291D91D6F732"}`, http.StatusUnprocessableEntity},
		{"UUID nula", `{"clave_idempotencia":"00000000-0000-4000-8000-000000000000"}`, http.StatusUnprocessableEntity},
		{"espacio exterior", " " + valida, http.StatusUnprocessableEntity},
		{"espacio interior", `{"clave_idempotencia": "` + claveSeleccionLlamamientoHTTPPrueba + `"}`, http.StatusUnprocessableEntity},
		{"clave escapada", `{"clave\u005fidempotencia":"` + claveSeleccionLlamamientoHTTPPrueba + `"}`, http.StatusUnprocessableEntity},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			ejecutor := &ejecutorSeleccionLlamamientoHTTPPrueba{
				recibo: reciboSeleccionLlamamientoHTTPPrueba(),
			}
			manejador := nuevoManejadorSeleccionLlamamientoHTTPPrueba(t, ejecutor)
			peticion := peticionSeleccionLlamamientoConCuerpo(caso.cuerpo)
			respuesta := httptest.NewRecorder()
			manejador.ServeHTTP(respuesta, peticion)
			if respuesta.Code != caso.estado || ejecutor.total() != 0 {
				t.Fatalf(
					"estado=%d cuerpo=%s llamadas=%d",
					respuesta.Code,
					respuesta.Body,
					ejecutor.total(),
				)
			}
		})
	}
}

func TestManejadorSeleccionLlamamientoAplicaLimiteAntesDelNegocio(t *testing.T) {
	t.Parallel()
	ejecutor := &ejecutorSeleccionLlamamientoHTTPPrueba{
		recibo: reciboSeleccionLlamamientoHTTPPrueba(),
	}
	manejador := nuevoManejadorSeleccionLlamamientoHTTPPrueba(t, ejecutor)
	peticion := peticionSeleccionLlamamientoConCuerpo(
		strings.Repeat("x", MaximoCuerpoCoberturaBytes+1),
	)
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusRequestEntityTooLarge || ejecutor.total() != 0 {
		t.Fatalf(
			"estado=%d cuerpo=%s llamadas=%d",
			respuesta.Code,
			respuesta.Body,
			ejecutor.total(),
		)
	}
}

type lectorCanceladorSeleccionLlamamiento struct {
	lector    *strings.Reader
	cancelar  context.CancelFunc
	cancelado bool
}

func (l *lectorCanceladorSeleccionLlamamiento) Read(p []byte) (int, error) {
	n, err := l.lector.Read(p)
	if !l.cancelado && l.lector.Len() == 0 {
		l.cancelado = true
		l.cancelar()
	}
	return n, err
}

func TestManejadorSeleccionLlamamientoCanceladoTrasDecodificarNoInvocaNegocio(
	t *testing.T,
) {
	t.Parallel()
	ctx, cancelar := context.WithCancel(context.Background())
	defer cancelar()
	cuerpo := `{"clave_idempotencia":"` +
		claveSeleccionLlamamientoHTTPPrueba + `"}`
	lector := &lectorCanceladorSeleccionLlamamiento{
		lector: strings.NewReader(cuerpo), cancelar: cancelar,
	}
	peticion := httptest.NewRequest(
		http.MethodPost,
		RutaSeleccionLlamamiento,
		lector,
	).WithContext(ctx)
	peticion.Header.Set("Content-Type", "application/json")
	peticion.Header.Set("Accept", "application/json")
	ejecutor := &ejecutorSeleccionLlamamientoHTTPPrueba{
		recibo: reciboSeleccionLlamamientoHTTPPrueba(),
	}
	manejador := nuevoManejadorSeleccionLlamamientoHTTPPrueba(t, ejecutor)
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusRequestTimeout || ejecutor.total() != 0 {
		t.Fatalf(
			"estado=%d cuerpo=%s llamadas=%d",
			respuesta.Code,
			respuesta.Body,
			ejecutor.total(),
		)
	}
}

func TestManejadorSeleccionLlamamientoNoPublicaExitoTrasCancelacion(
	t *testing.T,
) {
	t.Parallel()
	ctx, cancelar := context.WithCancel(context.Background())
	defer cancelar()
	ejecutor := &ejecutorSeleccionLlamamientoHTTPPrueba{
		recibo: reciboSeleccionLlamamientoHTTPPrueba(),
		durante: func(context.Context) {
			cancelar()
		},
	}
	manejador := nuevoManejadorSeleccionLlamamientoHTTPPrueba(t, ejecutor)
	peticion := nuevaPeticionSeleccionLlamamientoHTTPPrueba().WithContext(ctx)
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusRequestTimeout || ejecutor.total() != 1 ||
		bytes.Contains(respuesta.Body.Bytes(), []byte("recibo:llamamiento")) {
		t.Fatalf(
			"estado=%d cuerpo=%s llamadas=%d",
			respuesta.Code,
			respuesta.Body,
			ejecutor.total(),
		)
	}
}

func TestManejadorSeleccionLlamamientoSaneaCabecerasDeTodaRespuesta(
	t *testing.T,
) {
	t.Parallel()
	ejecutor := &ejecutorSeleccionLlamamientoHTTPPrueba{
		recibo: reciboSeleccionLlamamientoHTTPPrueba(),
	}
	manejador := nuevoManejadorSeleccionLlamamientoHTTPPrueba(t, ejecutor)
	respuesta := httptest.NewRecorder()
	respuesta.Header().Set("Set-Cookie", "sesion=privada")
	respuesta.Header().Set("Retry-After", "1")
	respuesta.Header().Set("Location", "https://privado.invalid")
	manejador.ServeHTTP(respuesta, nuevaPeticionSeleccionLlamamientoHTTPPrueba())
	if respuesta.Code != http.StatusOK ||
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

func TestManejadorSeleccionLlamamientoNuloTipadoEnEjecucionFallaCerrado(
	t *testing.T,
) {
	t.Parallel()
	ejecutor := &ejecutorSeleccionLlamamientoHTTPPrueba{
		recibo: reciboSeleccionLlamamientoHTTPPrueba(),
	}
	manejador := nuevoManejadorSeleccionLlamamientoHTTPPrueba(t, ejecutor)
	concreto := manejador.(*manejadorSeleccionLlamamiento)
	var nulo *ejecutorSeleccionLlamamientoHTTPPrueba
	concreto.ejecutor = nulo
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, nuevaPeticionSeleccionLlamamientoHTTPPrueba())
	if respuesta.Code != http.StatusServiceUnavailable || ejecutor.total() != 0 {
		t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body)
	}
}

func TestManejadorSeleccionLlamamientoPlazoPrevioFallaSinNegocio(t *testing.T) {
	t.Parallel()
	ctx, cancelar := context.WithDeadline(
		context.Background(),
		time.Now().Add(-time.Second),
	)
	defer cancelar()
	ejecutor := &ejecutorSeleccionLlamamientoHTTPPrueba{
		recibo: reciboSeleccionLlamamientoHTTPPrueba(),
	}
	manejador := nuevoManejadorSeleccionLlamamientoHTTPPrueba(t, ejecutor)
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(
		respuesta,
		nuevaPeticionSeleccionLlamamientoHTTPPrueba().WithContext(ctx),
	)
	if respuesta.Code != http.StatusGatewayTimeout || ejecutor.total() != 0 {
		t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body)
	}
}

var _ io.Reader = (*lectorCanceladorSeleccionLlamamiento)(nil)
