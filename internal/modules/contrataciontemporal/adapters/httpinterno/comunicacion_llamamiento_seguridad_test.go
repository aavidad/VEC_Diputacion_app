package httpinterno

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func ejecutorComunicacionHTTPValidoPrueba() *ejecutorComunicacionLlamamientoHTTPPrueba {
	solicitudRegistro := solicitudRegistroComunicacionHTTPPrueba()
	solicitudResolucion := solicitudResolucionComunicacionHTTPPrueba(
		ports.RespuestaLlamamientoAceptada,
	)
	return &ejecutorComunicacionLlamamientoHTTPPrueba{
		comunicacion: comunicacionHTTPPrueba(solicitudRegistro),
		resolucion:   resolucionComunicacionHTTPPrueba(solicitudResolucion),
	}
}

func TestManejadorComunicacionLlamamientoRechazaSuperficieHostilAntesDelServicio(
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
		{"raw path", func(r *http.Request) { r.URL.RawPath = RutaRegistroComunicacionLlamamiento }, http.StatusNotFound},
		{"scheme", func(r *http.Request) { r.URL.Scheme = "https" }, http.StatusNotFound},
		{"host", func(r *http.Request) { r.URL.Host = "interno.invalid" }, http.StatusNotFound},
		{"usuario URL", func(r *http.Request) { r.URL.User = url.User("persona") }, http.StatusNotFound},
		{"opaque", func(r *http.Request) { r.URL.Opaque = RutaRegistroComunicacionLlamamiento }, http.StatusNotFound},
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
		{"identidad", func(r *http.Request) { r.Header.Set("X-Vec-Actor", "persona:privada") }, http.StatusBadRequest},
		{"rol", func(r *http.Request) { r.Header.Set("X-Role", "admin") }, http.StatusBadRequest},
		{"idempotencia cabecera", func(r *http.Request) { r.Header.Set("Idempotency-Key", claveRegistroComunicacionHTTPPrueba) }, http.StatusBadRequest},
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
			ejecutor := ejecutorComunicacionHTTPValidoPrueba()
			manejador := nuevoManejadorComunicacionHTTPPrueba(t, ejecutor)
			peticion := peticionComunicacionHTTPPrueba(
				RutaRegistroComunicacionLlamamiento,
				cuerpoRegistroComunicacionHTTPPrueba(),
			)
			caso.mutar(peticion)
			respuesta := httptest.NewRecorder()
			manejador.ServeHTTP(respuesta, peticion)
			registros, resoluciones := ejecutor.totales()
			if respuesta.Code != caso.estado || registros != 0 || resoluciones != 0 {
				t.Fatalf(
					"estado=%d cuerpo=%s llamadas=%d/%d",
					respuesta.Code,
					respuesta.Body,
					registros,
					resoluciones,
				)
			}
			if caso.estado == http.StatusMethodNotAllowed &&
				respuesta.Header().Get("Allow") != http.MethodPost {
				t.Fatalf("Allow=%q", respuesta.Header().Get("Allow"))
			}
		})
	}
}

func TestManejadorComunicacionLlamamientoExigeJSONCanonicoDeUnaEntidad(
	t *testing.T,
) {
	t.Parallel()
	valido := cuerpoRegistroComunicacionHTTPPrueba()
	ordenDistinto := `{"organizacion_ref":"organizacion:http-comunicacion",` +
		`"clave_idempotencia":"` + claveRegistroComunicacionHTTPPrueba + `",` +
		`"expediente_ref":"expediente:http-comunicacion",` +
		`"llamamiento_ref":"llamamiento:http-comunicacion",` +
		`"version_esperada":7,"prueba_entrega_ref":"entrega:http-probatoria"}`
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
		{"campo desconocido", strings.TrimSuffix(valido, "}") + `,"actor":"privado"}`, http.StatusBadRequest},
		{"campo duplicado", strings.Replace(valido, `"organizacion_ref":`, `"organizacion_ref":"organizacion:otra","organizacion_ref":`, 1), http.StatusBadRequest},
		{"segundo JSON", valido + `{}`, http.StatusBadRequest},
		{"orden distinto", ordenDistinto, http.StatusUnprocessableEntity},
		{"espacio exterior", " " + valido, http.StatusUnprocessableEntity},
		{"espacio interior", strings.Replace(valido, `":7`, `": 7`, 1), http.StatusUnprocessableEntity},
		{"clave escapada", strings.Replace(valido, "clave_idempotencia", `clave\u005fidempotencia`, 1), http.StatusUnprocessableEntity},
		{"UUID mayuscula", strings.Replace(valido, claveRegistroComunicacionHTTPPrueba, strings.ToUpper(claveRegistroComunicacionHTTPPrueba), 1), http.StatusUnprocessableEntity},
		{"UUID nula", strings.Replace(valido, claveRegistroComunicacionHTTPPrueba, "00000000-0000-4000-8000-000000000000", 1), http.StatusUnprocessableEntity},
		{"version cero", strings.Replace(valido, `"version_esperada":7`, `"version_esperada":0`, 1), http.StatusUnprocessableEntity},
		{"version maxima", strings.Replace(valido, `"version_esperada":7`, `"version_esperada":9007199254740991`, 1), http.StatusUnprocessableEntity},
		{"version decimal", strings.Replace(valido, `"version_esperada":7`, `"version_esperada":7.0`, 1), http.StatusBadRequest},
		{"referencia directa", strings.Replace(valido, "expediente:http-comunicacion", "persona@example.invalid", 1), http.StatusUnprocessableEntity},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			ejecutor := ejecutorComunicacionHTTPValidoPrueba()
			manejador := nuevoManejadorComunicacionHTTPPrueba(t, ejecutor)
			respuesta := httptest.NewRecorder()
			manejador.ServeHTTP(
				respuesta,
				peticionComunicacionHTTPPrueba(
					RutaRegistroComunicacionLlamamiento,
					caso.cuerpo,
				),
			)
			registros, resoluciones := ejecutor.totales()
			if respuesta.Code != caso.estado || registros != 0 || resoluciones != 0 {
				t.Fatalf(
					"estado=%d cuerpo=%s llamadas=%d/%d",
					respuesta.Code,
					respuesta.Body,
					registros,
					resoluciones,
				)
			}
		})
	}
}

func TestManejadorComunicacionLlamamientoCierraContratoDeResolucion(
	t *testing.T,
) {
	t.Parallel()
	aceptacion := cuerpoResolucionComunicacionHTTPPrueba(
		ports.RespuestaLlamamientoAceptada,
	)
	expiracion := cuerpoResolucionComunicacionHTTPPrueba(
		ports.RespuestaLlamamientoExpirada,
	)
	casos := []struct {
		nombre string
		cuerpo string
		estado int
	}{
		{"campo omitido", strings.Replace(aceptacion, `,"prueba_respuesta_ref":"respuesta:http-probatoria"`, "", 1), http.StatusUnprocessableEntity},
		{"campo desconocido", strings.TrimSuffix(aceptacion, "}") + `,"perfil":"admin"}`, http.StatusBadRequest},
		{"campo duplicado", strings.Replace(aceptacion, `"respuesta":`, `"respuesta":"renuncia","respuesta":`, 1), http.StatusBadRequest},
		{"contenido posterior", aceptacion + `[]`, http.StatusBadRequest},
		{"respuesta inventada", strings.Replace(aceptacion, `"aceptacion"`, `"seleccion_automatica"`, 1), http.StatusUnprocessableEntity},
		{"aceptacion sin prueba", strings.Replace(aceptacion, "respuesta:http-probatoria", "", 1), http.StatusUnprocessableEntity},
		{"expiracion con respuesta", strings.Replace(expiracion, `"prueba_respuesta_ref":""`, `"prueba_respuesta_ref":"respuesta:forjada"`, 1), http.StatusUnprocessableEntity},
		{"clave en otro orden", strings.Replace(aceptacion, `"respuesta":"aceptacion","prueba_respuesta_ref":"respuesta:http-probatoria"`, `"prueba_respuesta_ref":"respuesta:http-probatoria","respuesta":"aceptacion"`, 1), http.StatusUnprocessableEntity},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			ejecutor := ejecutorComunicacionHTTPValidoPrueba()
			manejador := nuevoManejadorComunicacionHTTPPrueba(t, ejecutor)
			respuesta := httptest.NewRecorder()
			manejador.ServeHTTP(
				respuesta,
				peticionComunicacionHTTPPrueba(
					RutaResolucionComunicacionLlamamiento,
					caso.cuerpo,
				),
			)
			registros, resoluciones := ejecutor.totales()
			if respuesta.Code != caso.estado || registros != 0 || resoluciones != 0 {
				t.Fatalf(
					"estado=%d cuerpo=%s llamadas=%d/%d",
					respuesta.Code,
					respuesta.Body,
					registros,
					resoluciones,
				)
			}
		})
	}
}

type lectorVigiladoComunicacionHTTP struct {
	lecturas int
}

func (l *lectorVigiladoComunicacionHTTP) Read([]byte) (int, error) {
	l.lecturas++
	return 0, errors.New("el cuerpo no debia leerse")
}

func TestManejadorComunicacionLlamamientoLimitaAntesDeLeerOInvocar(
	t *testing.T,
) {
	t.Parallel()
	ejecutor := ejecutorComunicacionHTTPValidoPrueba()
	manejador := nuevoManejadorComunicacionHTTPPrueba(t, ejecutor)
	lector := &lectorVigiladoComunicacionHTTP{}
	peticion := httptest.NewRequest(
		http.MethodPost,
		RutaRegistroComunicacionLlamamiento,
		http.NoBody,
	)
	peticion.Body = io.NopCloser(lector)
	peticion.ContentLength = MaximoCuerpoComunicacionLlamamientoBytes + 1
	peticion.Header.Set("Content-Type", "application/json")
	peticion.Header.Set("Accept", "application/json")
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, peticion)
	registros, resoluciones := ejecutor.totales()
	if respuesta.Code != http.StatusRequestEntityTooLarge || lector.lecturas != 0 ||
		registros != 0 || resoluciones != 0 {
		t.Fatalf(
			"estado=%d lecturas=%d llamadas=%d/%d cuerpo=%s",
			respuesta.Code,
			lector.lecturas,
			registros,
			resoluciones,
			respuesta.Body,
		)
	}
}

func TestManejadorComunicacionLlamamientoLimitaTransferenciaSinLongitud(
	t *testing.T,
) {
	t.Parallel()
	ejecutor := ejecutorComunicacionHTTPValidoPrueba()
	manejador := nuevoManejadorComunicacionHTTPPrueba(t, ejecutor)
	peticion := peticionComunicacionHTTPPrueba(
		RutaRegistroComunicacionLlamamiento,
		strings.Repeat("x", MaximoCuerpoComunicacionLlamamientoBytes+2),
	)
	peticion.ContentLength = -1
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, peticion)
	registros, resoluciones := ejecutor.totales()
	if respuesta.Code != http.StatusRequestEntityTooLarge ||
		registros != 0 || resoluciones != 0 {
		t.Fatalf(
			"estado=%d cuerpo=%s llamadas=%d/%d",
			respuesta.Code,
			respuesta.Body,
			registros,
			resoluciones,
		)
	}
}

type lectorCanceladorComunicacionHTTP struct {
	lector    *strings.Reader
	cancelar  context.CancelFunc
	cancelado bool
}

func (l *lectorCanceladorComunicacionHTTP) Read(p []byte) (int, error) {
	n, err := l.lector.Read(p)
	if !l.cancelado && l.lector.Len() == 0 {
		l.cancelado = true
		l.cancelar()
	}
	return n, err
}

func TestManejadorComunicacionLlamamientoCanceladoTrasDecodificarNoInvoca(
	t *testing.T,
) {
	t.Parallel()
	casos := []struct {
		ruta   string
		cuerpo string
	}{
		{RutaRegistroComunicacionLlamamiento, cuerpoRegistroComunicacionHTTPPrueba()},
		{RutaResolucionComunicacionLlamamiento, cuerpoResolucionComunicacionHTTPPrueba(ports.RespuestaLlamamientoAceptada)},
	}
	for indice, caso := range casos {
		ctx, cancelar := context.WithCancel(context.Background())
		lector := &lectorCanceladorComunicacionHTTP{
			lector:   strings.NewReader(caso.cuerpo),
			cancelar: cancelar,
		}
		peticion := httptest.NewRequest(http.MethodPost, caso.ruta, lector).WithContext(ctx)
		peticion.Header.Set("Content-Type", "application/json")
		peticion.Header.Set("Accept", "application/json")
		ejecutor := ejecutorComunicacionHTTPValidoPrueba()
		manejador := nuevoManejadorComunicacionHTTPPrueba(t, ejecutor)
		respuesta := httptest.NewRecorder()
		manejador.ServeHTTP(respuesta, peticion)
		cancelar()
		registros, resoluciones := ejecutor.totales()
		if respuesta.Code != http.StatusRequestTimeout || registros != 0 || resoluciones != 0 {
			t.Fatalf(
				"caso %d: estado=%d cuerpo=%s llamadas=%d/%d",
				indice,
				respuesta.Code,
				respuesta.Body,
				registros,
				resoluciones,
			)
		}
	}
}

func TestManejadorComunicacionLlamamientoNoPublicaExitoTrasCancelacionDelServicio(
	t *testing.T,
) {
	t.Parallel()
	casos := []struct {
		nombre   string
		ruta     string
		cuerpo   string
		preparar func(*ejecutorComunicacionLlamamientoHTTPPrueba, context.CancelFunc)
	}{
		{
			"registro",
			RutaRegistroComunicacionLlamamiento,
			cuerpoRegistroComunicacionHTTPPrueba(),
			func(e *ejecutorComunicacionLlamamientoHTTPPrueba, cancelar context.CancelFunc) {
				e.duranteRegistro = func(context.Context) { cancelar() }
			},
		},
		{
			"resolucion",
			RutaResolucionComunicacionLlamamiento,
			cuerpoResolucionComunicacionHTTPPrueba(ports.RespuestaLlamamientoAceptada),
			func(e *ejecutorComunicacionLlamamientoHTTPPrueba, cancelar context.CancelFunc) {
				e.duranteResolucion = func(context.Context) { cancelar() }
			},
		},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			ctx, cancelar := context.WithCancel(context.Background())
			defer cancelar()
			ejecutor := ejecutorComunicacionHTTPValidoPrueba()
			caso.preparar(ejecutor, cancelar)
			manejador := nuevoManejadorComunicacionHTTPPrueba(t, ejecutor)
			peticion := peticionComunicacionHTTPPrueba(caso.ruta, caso.cuerpo).
				WithContext(ctx)
			respuesta := httptest.NewRecorder()
			manejador.ServeHTTP(respuesta, peticion)
			registros, resoluciones := ejecutor.totales()
			if respuesta.Code != http.StatusRequestTimeout || registros+resoluciones != 1 {
				t.Fatalf(
					"estado=%d cuerpo=%s llamadas=%d/%d",
					respuesta.Code,
					respuesta.Body,
					registros,
					resoluciones,
				)
			}
			for _, privado := range []string{
				"comunicacion:http", "resolucion:http", "recibo:http", "auditoria:http",
			} {
				if bytes.Contains(respuesta.Body.Bytes(), []byte(privado)) {
					t.Fatalf("resultado publicado tras cancelar: %s", respuesta.Body)
				}
			}
		})
	}
}

func TestManejadorComunicacionLlamamientoSaneaTodaRespuesta(t *testing.T) {
	t.Parallel()
	ejecutor := ejecutorComunicacionHTTPValidoPrueba()
	manejador := nuevoManejadorComunicacionHTTPPrueba(t, ejecutor)
	respuesta := httptest.NewRecorder()
	respuesta.Header().Set("Set-Cookie", "sesion=privada")
	respuesta.Header().Set("Retry-After", "1")
	respuesta.Header().Set("Location", "https://privado.invalid")
	manejador.ServeHTTP(
		respuesta,
		peticionComunicacionHTTPPrueba(
			RutaRegistroComunicacionLlamamiento,
			cuerpoRegistroComunicacionHTTPPrueba(),
		),
	)
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

func TestManejadorComunicacionLlamamientoNuloTipadoYPlazoFallanCerrado(
	t *testing.T,
) {
	t.Parallel()
	ejecutor := ejecutorComunicacionHTTPValidoPrueba()
	manejador := nuevoManejadorComunicacionHTTPPrueba(t, ejecutor)
	concreto := manejador.(*manejadorComunicacionLlamamiento)
	var nulo *ejecutorComunicacionLlamamientoHTTPPrueba
	concreto.ejecutor = nulo
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(
		respuesta,
		peticionComunicacionHTTPPrueba(
			RutaRegistroComunicacionLlamamiento,
			cuerpoRegistroComunicacionHTTPPrueba(),
		),
	)
	if respuesta.Code != http.StatusServiceUnavailable {
		t.Fatalf("nulo tipado: estado=%d cuerpo=%s", respuesta.Code, respuesta.Body)
	}

	ctx, cancelar := context.WithDeadline(
		context.Background(),
		time.Now().Add(-time.Second),
	)
	defer cancelar()
	ejecutor = ejecutorComunicacionHTTPValidoPrueba()
	manejador = nuevoManejadorComunicacionHTTPPrueba(t, ejecutor)
	respuesta = httptest.NewRecorder()
	manejador.ServeHTTP(
		respuesta,
		peticionComunicacionHTTPPrueba(
			RutaRegistroComunicacionLlamamiento,
			cuerpoRegistroComunicacionHTTPPrueba(),
		).WithContext(ctx),
	)
	registros, resoluciones := ejecutor.totales()
	if respuesta.Code != http.StatusGatewayTimeout || registros != 0 || resoluciones != 0 {
		t.Fatalf(
			"plazo: estado=%d cuerpo=%s llamadas=%d/%d",
			respuesta.Code,
			respuesta.Body,
			registros,
			resoluciones,
		)
	}
}

var _ io.Reader = (*lectorCanceladorComunicacionHTTP)(nil)
