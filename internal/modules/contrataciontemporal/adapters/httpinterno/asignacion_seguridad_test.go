package httpinterno

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestManejadorAsignacionRechazaCookiesYCabecerasDeAutoridad(t *testing.T) {
	for _, cabecera := range []string{
		"Cookie", "Set-Cookie", "Authorization", "Proxy-Authorization",
		"Remote-User", "X-Remote-User", "X-Actor", "X-Perfil",
		"X-Organizacion", "X-Vec-Identidad", "X-Forwarded-For",
		"Idempotency-Key", "Role", "Connection", "User-Agent",
	} {
		t.Run(cabecera, func(t *testing.T) {
			autoridad, ejecutor, manejador := entornoAsignacionHTTPPrueba(t)
			peticion := nuevaPeticionAsignacionPrueba(
				RutaAsignaciones,
				cuerpoAsignacionPrueba(3),
			)
			peticion.Header.Set(cabecera, "valor-no-confiable")
			respuesta := httptest.NewRecorder()
			manejador.ServeHTTP(respuesta, peticion)
			if respuesta.Code != http.StatusBadRequest ||
				autoridad.llamadas != 0 || ejecutor.asignaciones != 0 {
				t.Fatalf(
					"estado=%d autoridad=%d ejecutor=%d cuerpo=%s",
					respuesta.Code,
					autoridad.llamadas,
					ejecutor.asignaciones,
					respuesta.Body.String(),
				)
			}
		})
	}
}

func TestManejadorAsignacionRechazaAutoridadEnJSON(t *testing.T) {
	for _, campo := range []string{
		"autenticacion_ref", "sesion_ref", "perfil_ref", "organizacion_ref",
		"actor_ref", "rol", "permiso",
	} {
		t.Run(campo, func(t *testing.T) {
			autoridad, ejecutor, manejador := entornoAsignacionHTTPPrueba(t)
			cuerpo := strings.TrimSuffix(cuerpoAsignacionPrueba(3), "}") +
				`,"` + campo + `":"autoridad:no-confiable"}`
			respuesta := httptest.NewRecorder()
			manejador.ServeHTTP(
				respuesta,
				nuevaPeticionAsignacionPrueba(RutaAsignaciones, cuerpo),
			)
			if respuesta.Code != http.StatusBadRequest ||
				autoridad.llamadas != 0 || ejecutor.asignaciones != 0 {
				t.Fatalf("campo %s aceptado: %d", campo, respuesta.Code)
			}
		})
	}
}

func TestManejadorAsignacionCierraJSONYLimitesAntesDeAutoridad(t *testing.T) {
	base := cuerpoAsignacionPrueba(3)
	casos := []struct {
		nombre string
		ruta   string
		cuerpo string
		estado int
	}{
		{"duplicado", RutaAsignaciones, strings.Replace(base, `"expediente_ref":`, `"expediente_ref":"expediente:otro:001","expediente_ref":`, 1), http.StatusBadRequest},
		{"desconocido", RutaAsignaciones, strings.TrimSuffix(base, "}") + `,"extra":true}`, http.StatusBadRequest},
		{"compuesto", RutaAsignaciones, strings.Replace(base, `"unidad_ref":"unidad:destino:http:001"`, `"unidad_ref":{"valor":"unidad:destino:http:001"}`, 1), http.StatusBadRequest},
		{"dos valores", RutaAsignaciones, base + `{}`, http.StatusBadRequest},
		{"campo de reasignación en alta", RutaAsignaciones, strings.TrimSuffix(base, "}") + `,"observaciones":"texto"}`, http.StatusBadRequest},
		{"sin versión", RutaAsignaciones, strings.Replace(base, `"version_esperada":3,`, "", 1), http.StatusUnprocessableEntity},
		{"versión cero", RutaAsignaciones, strings.Replace(base, `"version_esperada":3`, `"version_esperada":0`, 1), http.StatusUnprocessableEntity},
		{"clave inválida", RutaAsignaciones, strings.Replace(base, "11111111-2222-4333-8444-555555555555", "clave-humana", 1), http.StatusUnprocessableEntity},
		{"observación vacía", RutaReasignaciones, strings.Replace(cuerpoReasignacionPrueba(7), "Cambio motivado de unidad responsable.", "", 1), http.StatusUnprocessableEntity},
		{"observación no NFC", RutaReasignaciones, strings.Replace(cuerpoReasignacionPrueba(7), "Cambio motivado de unidad responsable.", "revisio\\u0301n", 1), http.StatusUnprocessableEntity},
		{"observación excesiva", RutaReasignaciones, strings.Replace(cuerpoReasignacionPrueba(7), "Cambio motivado de unidad responsable.", strings.Repeat("a", 1001), 1), http.StatusUnprocessableEntity},
		{"cuerpo excesivo", RutaAsignaciones, strings.Repeat(" ", MaximoCuerpoAsignacionBytes+1), http.StatusRequestEntityTooLarge},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			autoridad, ejecutor, manejador := entornoAsignacionHTTPPrueba(t)
			respuesta := httptest.NewRecorder()
			manejador.ServeHTTP(
				respuesta,
				nuevaPeticionAsignacionPrueba(caso.ruta, caso.cuerpo),
			)
			if respuesta.Code != caso.estado || autoridad.llamadas != 0 ||
				ejecutor.asignaciones+ejecutor.reasignaciones != 0 {
				t.Fatalf(
					"estado=%d esperado=%d autoridad=%d ejecutor=%d cuerpo=%s",
					respuesta.Code,
					caso.estado,
					autoridad.llamadas,
					ejecutor.asignaciones+ejecutor.reasignaciones,
					respuesta.Body.String(),
				)
			}
		})
	}
}

func TestManejadorAsignacionRechazaURLYMetodoNoExactos(t *testing.T) {
	casos := []struct {
		nombre   string
		preparar func(*http.Request)
		estado   int
	}{
		{"query", func(r *http.Request) { r.URL.RawQuery = "actor=x" }, http.StatusNotFound},
		{"force query", func(r *http.Request) { r.URL.ForceQuery = true }, http.StatusNotFound},
		{"raw path", func(r *http.Request) { r.URL.RawPath = "/api/vec/contratacion-temporal/%61signaciones" }, http.StatusNotFound},
		{"fragmento", func(r *http.Request) { r.URL.Fragment = "perfil" }, http.StatusNotFound},
		{"usuario URL", func(r *http.Request) { r.URL.User = url.User("actor") }, http.StatusNotFound},
		{"ruta próxima", func(r *http.Request) { r.URL.Path += "/" }, http.StatusNotFound},
		{"método", func(r *http.Request) { r.Method = http.MethodPut }, http.StatusMethodNotAllowed},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			autoridad, ejecutor, manejador := entornoAsignacionHTTPPrueba(t)
			peticion := nuevaPeticionAsignacionPrueba(
				RutaAsignaciones,
				cuerpoAsignacionPrueba(3),
			)
			caso.preparar(peticion)
			respuesta := httptest.NewRecorder()
			manejador.ServeHTTP(respuesta, peticion)
			if respuesta.Code != caso.estado || autoridad.llamadas != 0 ||
				ejecutor.asignaciones != 0 {
				t.Fatalf("estado=%d autoridad=%d ejecutor=%d", respuesta.Code, autoridad.llamadas, ejecutor.asignaciones)
			}
		})
	}
}

func TestManejadorAsignacionFallaCerradoAnteCancelacion(t *testing.T) {
	t.Run("antes de leer", func(t *testing.T) {
		autoridad, ejecutor, manejador := entornoAsignacionHTTPPrueba(t)
		ctx, cancelar := context.WithCancel(context.Background())
		cancelar()
		peticion := nuevaPeticionAsignacionPrueba(RutaAsignaciones, cuerpoAsignacionPrueba(3)).WithContext(ctx)
		respuesta := httptest.NewRecorder()
		manejador.ServeHTTP(respuesta, peticion)
		comprobarCancelacionAsignacion(t, respuesta, autoridad, ejecutor, 0, 0)
	})
	t.Run("tras autoridad", func(t *testing.T) {
		autoridad, ejecutor, manejador := entornoAsignacionHTTPPrueba(t)
		ctx, cancelar := context.WithCancel(context.Background())
		autoridad.antes = cancelar
		peticion := nuevaPeticionAsignacionPrueba(RutaAsignaciones, cuerpoAsignacionPrueba(3)).WithContext(ctx)
		respuesta := httptest.NewRecorder()
		manejador.ServeHTTP(respuesta, peticion)
		comprobarCancelacionAsignacion(t, respuesta, autoridad, ejecutor, 1, 0)
	})
	t.Run("tras ejecutor con recibo válido", func(t *testing.T) {
		autoridad, ejecutor, manejador := entornoAsignacionHTTPPrueba(t)
		ctx, cancelar := context.WithCancel(context.Background())
		ejecutor.antes = cancelar
		peticion := nuevaPeticionAsignacionPrueba(RutaAsignaciones, cuerpoAsignacionPrueba(3)).WithContext(ctx)
		respuesta := httptest.NewRecorder()
		manejador.ServeHTTP(respuesta, peticion)
		if respuesta.Code != http.StatusCreated || autoridad.llamadas != 1 ||
			ejecutor.asignaciones != 1 {
			t.Fatalf("estado=%d autoridad=%d ejecutor=%d cuerpo=%s", respuesta.Code, autoridad.llamadas, ejecutor.asignaciones, respuesta.Body.String())
		}
		comprobarRespuestaAsignacionMinimizada(t, respuesta)
	})
	t.Run("tras ejecutor sin recibo", func(t *testing.T) {
		autoridad, ejecutor, manejador := entornoAsignacionHTTPPrueba(t)
		ctx, cancelar := context.WithCancel(context.Background())
		ejecutor.recibo = ports.ReciboAsignacion{}
		ejecutor.err = context.Canceled
		ejecutor.antes = cancelar
		peticion := nuevaPeticionAsignacionPrueba(RutaAsignaciones, cuerpoAsignacionPrueba(3)).WithContext(ctx)
		respuesta := httptest.NewRecorder()
		manejador.ServeHTTP(respuesta, peticion)
		comprobarCancelacionAsignacion(t, respuesta, autoridad, ejecutor, 1, 1)
	})
	t.Run("error con recibo tras cancelación", func(t *testing.T) {
		autoridad, ejecutor, manejador := entornoAsignacionHTTPPrueba(t)
		ctx, cancelar := context.WithCancel(context.Background())
		ejecutor.err = application.ErrAsignacionDenegada
		ejecutor.antes = cancelar
		peticion := nuevaPeticionAsignacionPrueba(RutaAsignaciones, cuerpoAsignacionPrueba(3)).WithContext(ctx)
		respuesta := httptest.NewRecorder()
		manejador.ServeHTTP(respuesta, peticion)
		comprobarResultadoAsignacionNoConfiable(t, respuesta, autoridad, ejecutor)
	})
	t.Run("recibo inválido tras cancelación", func(t *testing.T) {
		autoridad, ejecutor, manejador := entornoAsignacionHTTPPrueba(t)
		ctx, cancelar := context.WithCancel(context.Background())
		ejecutor.recibo.ReciboRef = ""
		ejecutor.antes = cancelar
		peticion := nuevaPeticionAsignacionPrueba(RutaAsignaciones, cuerpoAsignacionPrueba(3)).WithContext(ctx)
		respuesta := httptest.NewRecorder()
		manejador.ServeHTTP(respuesta, peticion)
		comprobarResultadoAsignacionNoConfiable(t, respuesta, autoridad, ejecutor)
	})
}

func TestManejadorAsignacionRechazaReciboDiscordante(t *testing.T) {
	mutaciones := []struct {
		nombre string
		mutar  func(*ports.ReciboAsignacion)
	}{
		{"operación", func(r *ports.ReciboAsignacion) { r.Operacion = ports.OperacionRegistrarReasignacion }},
		{"organización", func(r *ports.ReciboAsignacion) { r.OrganizacionRef = "organizacion:otra:http:001" }},
		{"expediente", func(r *ports.ReciboAsignacion) { r.ExpedienteRef = "expediente:otro:http:001" }},
		{"versión anterior", func(r *ports.ReciboAsignacion) { r.VersionAnterior++ }},
		{"versión resultante", func(r *ports.ReciboAsignacion) { r.VersionResultante++ }},
		{"unidad", func(r *ports.ReciboAsignacion) { r.UnidadRef = "unidad:otra:http:001" }},
		{"responsable", func(r *ports.ReciboAsignacion) { r.ResponsableRef = "persona:otra:http:001" }},
		{"recibo", func(r *ports.ReciboAsignacion) { r.ReciboRef = "" }},
		{"HMAC", func(r *ports.ReciboAsignacion) { r.HuellaPeticionHMAC = "" }},
		{"instante", func(r *ports.ReciboAsignacion) { r.ConfirmadaEn = r.ConfirmadaEn.Local() }},
	}
	for _, mutacion := range mutaciones {
		t.Run(mutacion.nombre, func(t *testing.T) {
			autoridad, ejecutor, manejador := entornoAsignacionHTTPPrueba(t)
			mutacion.mutar(&ejecutor.recibo)
			respuesta := httptest.NewRecorder()
			manejador.ServeHTTP(respuesta, nuevaPeticionAsignacionPrueba(RutaAsignaciones, cuerpoAsignacionPrueba(3)))
			if respuesta.Code != http.StatusBadGateway || autoridad.llamadas != 1 || ejecutor.asignaciones != 1 {
				t.Fatalf("estado=%d autoridad=%d ejecutor=%d cuerpo=%s", respuesta.Code, autoridad.llamadas, ejecutor.asignaciones, respuesta.Body.String())
			}
		})
	}
}

func TestManejadorAsignacionFallaCerradoAnteErroresYNilRuntime(t *testing.T) {
	t.Run("autoridad", func(t *testing.T) {
		autoridad, ejecutor, manejador := entornoAsignacionHTTPPrueba(t)
		autoridad.err = ErrContextoCanalAusente
		respuesta := httptest.NewRecorder()
		manejador.ServeHTTP(respuesta, nuevaPeticionAsignacionPrueba(RutaAsignaciones, cuerpoAsignacionPrueba(3)))
		if respuesta.Code != http.StatusUnauthorized || autoridad.llamadas != 1 || ejecutor.asignaciones != 0 {
			t.Fatalf("estado=%d autoridad=%d ejecutor=%d", respuesta.Code, autoridad.llamadas, ejecutor.asignaciones)
		}
	})
	t.Run("identidad ausente", func(t *testing.T) {
		autoridad, ejecutor, manejador := entornoAsignacionHTTPPrueba(t)
		autoridad.contexto = ContextoCanalAsignacion{}
		respuesta := httptest.NewRecorder()
		manejador.ServeHTTP(respuesta, nuevaPeticionAsignacionPrueba(RutaAsignaciones, cuerpoAsignacionPrueba(3)))
		if respuesta.Code != http.StatusServiceUnavailable || autoridad.llamadas != 1 || ejecutor.asignaciones != 0 {
			t.Fatalf("estado=%d autoridad=%d ejecutor=%d", respuesta.Code, autoridad.llamadas, ejecutor.asignaciones)
		}
	})
	t.Run("error sin recibo", func(t *testing.T) {
		autoridad, ejecutor, manejador := entornoAsignacionHTTPPrueba(t)
		ejecutor.recibo = ports.ReciboAsignacion{}
		ejecutor.err = application.ErrAsignacionDenegada
		respuesta := httptest.NewRecorder()
		manejador.ServeHTTP(respuesta, nuevaPeticionAsignacionPrueba(RutaAsignaciones, cuerpoAsignacionPrueba(3)))
		if respuesta.Code != http.StatusForbidden || autoridad.llamadas != 1 || ejecutor.asignaciones != 1 {
			t.Fatalf("estado=%d autoridad=%d ejecutor=%d", respuesta.Code, autoridad.llamadas, ejecutor.asignaciones)
		}
	})
	t.Run("resultado con error", func(t *testing.T) {
		autoridad, ejecutor, manejador := entornoAsignacionHTTPPrueba(t)
		ejecutor.err = application.ErrAsignacionDenegada
		respuesta := httptest.NewRecorder()
		manejador.ServeHTTP(respuesta, nuevaPeticionAsignacionPrueba(RutaAsignaciones, cuerpoAsignacionPrueba(3)))
		if respuesta.Code != http.StatusBadGateway || autoridad.llamadas != 1 || ejecutor.asignaciones != 1 {
			t.Fatalf("estado=%d autoridad=%d ejecutor=%d", respuesta.Code, autoridad.llamadas, ejecutor.asignaciones)
		}
	})
	t.Run("nil tipado sobrevenido", func(t *testing.T) {
		autoridad, _, manejador := entornoAsignacionHTTPPrueba(t)
		concreto := manejador.(*manejadorAsignacion)
		var ejecutorNil *ejecutorAsignacionPrueba
		concreto.ejecutor = ejecutorNil
		respuesta := httptest.NewRecorder()
		manejador.ServeHTTP(respuesta, nuevaPeticionAsignacionPrueba(RutaAsignaciones, cuerpoAsignacionPrueba(3)))
		if respuesta.Code != http.StatusServiceUnavailable || autoridad.llamadas != 0 {
			t.Fatalf("estado=%d autoridad=%d", respuesta.Code, autoridad.llamadas)
		}
	})
}

func TestManejadorAsignacionSaneaCabecerasDeRespuesta(t *testing.T) {
	_, _, manejador := entornoAsignacionHTTPPrueba(t)
	respuesta := httptest.NewRecorder()
	respuesta.Header().Set("Set-Cookie", "sesion=prohibida")
	respuesta.Header().Set("Location", "https://privado.invalid")
	manejador.ServeHTTP(respuesta, nuevaPeticionAsignacionPrueba(RutaAsignaciones, cuerpoAsignacionPrueba(3)))
	if respuesta.Code != http.StatusCreated || respuesta.Header().Get("Set-Cookie") != "" || respuesta.Header().Get("Location") != "" || len(respuesta.Result().Cookies()) != 0 || respuesta.Header().Get("Content-Length") != strconv.Itoa(respuesta.Body.Len()) {
		t.Fatalf("respuesta insegura: estado=%d cabeceras=%#v", respuesta.Code, respuesta.Header())
	}
}

func entornoAsignacionHTTPPrueba(
	t *testing.T,
) (*autoridadAsignacionPrueba, *ejecutorAsignacionPrueba, http.Handler) {
	t.Helper()
	autoridad := &autoridadAsignacionPrueba{contexto: contextoCanalAsignacionPrueba()}
	ejecutor := &ejecutorAsignacionPrueba{recibo: reciboAsignacionHTTPPrueba(ports.OperacionRegistrarAsignacion, 3)}
	manejador, err := NuevoManejadorAsignacion(autoridad, ejecutor)
	if err != nil {
		t.Fatal(err)
	}
	return autoridad, ejecutor, manejador
}

func comprobarCancelacionAsignacion(
	t *testing.T,
	respuesta *httptest.ResponseRecorder,
	autoridad *autoridadAsignacionPrueba,
	ejecutor *ejecutorAsignacionPrueba,
	llamadasAutoridad int,
	llamadasEjecutor int,
) {
	t.Helper()
	if respuesta.Code != http.StatusRequestTimeout || autoridad.llamadas != llamadasAutoridad || ejecutor.asignaciones != llamadasEjecutor || !strings.Contains(respuesta.Body.String(), "peticion_cancelada") {
		t.Fatalf("estado=%d autoridad=%d ejecutor=%d cuerpo=%s", respuesta.Code, autoridad.llamadas, ejecutor.asignaciones, respuesta.Body.String())
	}
}

func comprobarResultadoAsignacionNoConfiable(
	t *testing.T,
	respuesta *httptest.ResponseRecorder,
	autoridad *autoridadAsignacionPrueba,
	ejecutor *ejecutorAsignacionPrueba,
) {
	t.Helper()
	if respuesta.Code != http.StatusBadGateway || autoridad.llamadas != 1 ||
		ejecutor.asignaciones != 1 ||
		!strings.Contains(respuesta.Body.String(), "resultado_no_confiable") {
		t.Fatalf("estado=%d autoridad=%d ejecutor=%d cuerpo=%s", respuesta.Code, autoridad.llamadas, ejecutor.asignaciones, respuesta.Body.String())
	}
}
