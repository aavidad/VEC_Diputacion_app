package httpinterno

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestManejadorAnalisisRRHHRechazaJSONAbiertoOLimitadoAntesDeAutoridad(
	t *testing.T,
) {
	base := cuerpoRegistroAnalisisRRHHPrueba()
	casos := []struct {
		nombre string
		cuerpo string
		estado int
	}{
		{
			"identidad en cuerpo",
			strings.Replace(
				base,
				"{",
				`{"actor_ref":"actor:aportado",`,
				1,
			),
			http.StatusBadRequest,
		},
		{
			"campo desconocido",
			strings.Replace(base, "{", `{"extra":true,`, 1),
			http.StatusBadRequest,
		},
		{
			"clave duplicada",
			strings.Replace(
				base,
				`"version_esperada":1,`,
				`"version_esperada":1,"version_esperada":1,`,
				1,
			),
			http.StatusBadRequest,
		},
		{
			"rectificacion en registro",
			strings.Replace(
				base,
				`"analisis":`,
				`"motivo_rectificacion_clave":"ajuste_jornada","analisis":`,
				1,
			),
			http.StatusBadRequest,
		},
		{
			"rectificacion sin motivo",
			strings.Replace(
				base,
				`"version_esperada":1`,
				`"version_esperada":2`,
				1,
			),
			http.StatusUnprocessableEntity,
		},
		{
			"fecha no civil",
			strings.Replace(
				base,
				"2026-09-01T00:00:00Z",
				"2026-09-01T00:00:01Z",
				1,
			),
			http.StatusUnprocessableEntity,
		},
		{
			"periodo superior a cien anos",
			strings.Replace(
				base,
				"2027-02-28T00:00:00Z",
				"2126-09-02T00:00:00Z",
				1,
			),
			http.StatusUnprocessableEntity,
		},
		{
			"cuerpo demasiado grande",
			strings.Repeat(" ", MaximoCuerpoAnalisisRRHHBytes+1),
			http.StatusRequestEntityTooLarge,
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			autoridad := &autoridadAnalisisRRHHPrueba{
				contexto: contextoCanalAnalisisRRHHPrueba(),
			}
			ejecutor := &ejecutorAnalisisRRHHPrueba{}
			manejador, err := NuevoManejadorAnalisisRRHH(
				autoridad,
				ejecutor,
			)
			if err != nil {
				t.Fatal(err)
			}
			respuesta := httptest.NewRecorder()
			ruta := RutaRegistroAnalisisRRHH
			if caso.nombre == "rectificacion sin motivo" {
				ruta = RutaRectificacionAnalisisRRHH
			}
			manejador.ServeHTTP(
				respuesta,
				nuevaPeticionAnalisisRRHHPrueba(
					ruta,
					caso.cuerpo,
				),
			)
			if respuesta.Code != caso.estado || autoridad.llamadas != 0 ||
				ejecutor.registros != 0 || ejecutor.rectificaciones != 0 {
				t.Fatalf(
					"estado=%d autoridad=%d ejecutor=%d/%d cuerpo=%s",
					respuesta.Code,
					autoridad.llamadas,
					ejecutor.registros,
					ejecutor.rectificaciones,
					respuesta.Body.String(),
				)
			}
			comprobarRespuestaSeguraAnalisisRRHH(t, respuesta)
		})
	}
}

func TestManejadorAnalisisRRHHRechazaCabecerasDeAutoridadYCookies(
	t *testing.T,
) {
	for _, cabecera := range []string{
		"Cookie",
		"Authorization",
		"X-Actor",
		"X-Perfil",
		"X-Organizacion",
		"X-Forwarded-User",
		"Remote-User",
	} {
		t.Run(cabecera, func(t *testing.T) {
			autoridad := &autoridadAnalisisRRHHPrueba{
				contexto: contextoCanalAnalisisRRHHPrueba(),
			}
			ejecutor := &ejecutorAnalisisRRHHPrueba{}
			manejador, err := NuevoManejadorAnalisisRRHH(
				autoridad,
				ejecutor,
			)
			if err != nil {
				t.Fatal(err)
			}
			peticion := nuevaPeticionAnalisisRRHHPrueba(
				RutaRegistroAnalisisRRHH,
				cuerpoRegistroAnalisisRRHHPrueba(),
			)
			peticion.Header.Set(cabecera, "valor-aportado")
			respuesta := httptest.NewRecorder()
			respuesta.Header().Add("Set-Cookie", "sesion=prohibida")
			manejador.ServeHTTP(respuesta, peticion)
			if respuesta.Code != http.StatusBadRequest ||
				autoridad.llamadas != 0 || ejecutor.registros != 0 {
				t.Fatalf("respuesta inesperada: %d %s", respuesta.Code, respuesta.Body)
			}
			comprobarRespuestaSeguraAnalisisRRHH(t, respuesta)
		})
	}
}

func TestManejadorAnalisisRRHHRechazaContextoNoAcreditado(t *testing.T) {
	autoridad := &autoridadAnalisisRRHHPrueba{}
	ejecutor := &ejecutorAnalisisRRHHPrueba{}
	manejador, err := NuevoManejadorAnalisisRRHH(autoridad, ejecutor)
	if err != nil {
		t.Fatal(err)
	}
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(
		respuesta,
		nuevaPeticionAnalisisRRHHPrueba(
			RutaRegistroAnalisisRRHH,
			cuerpoRegistroAnalisisRRHHPrueba(),
		),
	)
	if respuesta.Code != http.StatusServiceUnavailable ||
		autoridad.llamadas != 1 || ejecutor.registros != 0 {
		t.Fatalf("contexto no acreditado aceptado: %d %s", respuesta.Code, respuesta.Body)
	}
}

func TestManejadorAnalisisRRHHRespetaCancelacionEnCadaFrontera(t *testing.T) {
	t.Run("antes de resolver autoridad", func(t *testing.T) {
		ctx, cancelar := context.WithCancel(context.Background())
		cancelar()
		autoridad := &autoridadAnalisisRRHHPrueba{
			contexto: contextoCanalAnalisisRRHHPrueba(),
		}
		ejecutor := &ejecutorAnalisisRRHHPrueba{}
		manejador, err := NuevoManejadorAnalisisRRHH(autoridad, ejecutor)
		if err != nil {
			t.Fatal(err)
		}
		peticion := nuevaPeticionAnalisisRRHHPrueba(
			RutaRegistroAnalisisRRHH,
			cuerpoRegistroAnalisisRRHHPrueba(),
		).WithContext(ctx)
		respuesta := httptest.NewRecorder()
		manejador.ServeHTTP(respuesta, peticion)
		if respuesta.Code != http.StatusRequestTimeout ||
			autoridad.llamadas != 0 || ejecutor.registros != 0 {
			t.Fatalf("cancelacion inicial ignorada: %d", respuesta.Code)
		}
	})

	t.Run("despues de autoridad", func(t *testing.T) {
		ctx, cancelar := context.WithCancel(context.Background())
		autoridad := &autoridadAnalisisRRHHPrueba{
			contexto: contextoCanalAnalisisRRHHPrueba(),
			antes:    cancelar,
		}
		ejecutor := &ejecutorAnalisisRRHHPrueba{}
		manejador, err := NuevoManejadorAnalisisRRHH(autoridad, ejecutor)
		if err != nil {
			t.Fatal(err)
		}
		peticion := nuevaPeticionAnalisisRRHHPrueba(
			RutaRegistroAnalisisRRHH,
			cuerpoRegistroAnalisisRRHHPrueba(),
		).WithContext(ctx)
		respuesta := httptest.NewRecorder()
		manejador.ServeHTTP(respuesta, peticion)
		if respuesta.Code != http.StatusRequestTimeout ||
			autoridad.llamadas != 1 || ejecutor.registros != 0 {
			t.Fatalf("cancelacion tras autoridad ignorada: %d", respuesta.Code)
		}
	})

	t.Run("recibo durable prevalece", func(t *testing.T) {
		ctx, cancelar := context.WithCancel(context.Background())
		autoridad := &autoridadAnalisisRRHHPrueba{
			contexto: contextoCanalAnalisisRRHHPrueba(),
		}
		ejecutor := &ejecutorAnalisisRRHHPrueba{
			recibo: reciboAnalisisRRHHPrueba(
				ports.OperacionRegistrarAnalisis,
				1,
			),
			err:   context.Canceled,
			antes: cancelar,
		}
		manejador, err := NuevoManejadorAnalisisRRHH(autoridad, ejecutor)
		if err != nil {
			t.Fatal(err)
		}
		peticion := nuevaPeticionAnalisisRRHHPrueba(
			RutaRegistroAnalisisRRHH,
			cuerpoRegistroAnalisisRRHHPrueba(),
		).WithContext(ctx)
		respuesta := httptest.NewRecorder()
		manejador.ServeHTTP(respuesta, peticion)
		if respuesta.Code != http.StatusCreated {
			t.Fatalf("recibo durable degradado: %d %s", respuesta.Code, respuesta.Body)
		}
	})
}

func TestManejadorAnalisisRRHHRechazaURLNoExactaYMetodo(t *testing.T) {
	casos := []struct {
		nombre   string
		preparar func(*http.Request)
		metodo   string
		estado   int
	}{
		{"query", func(r *http.Request) { r.URL.RawQuery = "actor=uno" }, http.MethodPost, http.StatusNotFound},
		{"raw path", func(r *http.Request) { r.URL.RawPath = "/api/vec/contratacion-temporal/analisis/%72egistros" }, http.MethodPost, http.StatusNotFound},
		{"host absoluto", func(r *http.Request) { r.URL.Scheme, r.URL.Host = "https", "interno.invalid" }, http.MethodPost, http.StatusNotFound},
		{"usuario URL", func(r *http.Request) { r.URL.User = url.User("actor") }, http.MethodPost, http.StatusNotFound},
		{"fragmento", func(r *http.Request) { r.URL.Fragment = "perfil" }, http.MethodPost, http.StatusNotFound},
		{"get", func(*http.Request) {}, http.MethodGet, http.StatusMethodNotAllowed},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			autoridad := &autoridadAnalisisRRHHPrueba{
				contexto: contextoCanalAnalisisRRHHPrueba(),
			}
			ejecutor := &ejecutorAnalisisRRHHPrueba{}
			manejador, err := NuevoManejadorAnalisisRRHH(autoridad, ejecutor)
			if err != nil {
				t.Fatal(err)
			}
			peticion := nuevaPeticionAnalisisRRHHPrueba(
				RutaRegistroAnalisisRRHH,
				cuerpoRegistroAnalisisRRHHPrueba(),
			)
			peticion.Method = caso.metodo
			caso.preparar(peticion)
			respuesta := httptest.NewRecorder()
			manejador.ServeHTTP(respuesta, peticion)
			if respuesta.Code != caso.estado || autoridad.llamadas != 0 ||
				ejecutor.registros != 0 {
				t.Fatalf("URL no cerrada: %d %s", respuesta.Code, respuesta.Body)
			}
		})
	}
}

func TestManejadorAnalisisRRHHOcultaErroresYValidaRecibo(t *testing.T) {
	const privado = "dsn=postgres://usuario:secreto@interno"
	casos := []struct {
		nombre   string
		preparar func(*ejecutorAnalisisRRHHPrueba)
		estado   int
		codigo   string
	}{
		{
			"dependencia opaca",
			func(e *ejecutorAnalisisRRHHPrueba) {
				e.err = errors.Join(
					application.ErrDependenciaOperacionAnalisisNoDisponible,
					errors.New(privado),
				)
			},
			http.StatusServiceUnavailable,
			"servicio_no_disponible",
		},
		{
			"denegacion",
			func(e *ejecutorAnalisisRRHHPrueba) {
				e.err = application.ErrOperacionAnalisisDenegada
			},
			http.StatusForbidden,
			"acceso_denegado",
		},
		{
			"conflicto",
			func(e *ejecutorAnalisisRRHHPrueba) {
				e.err = application.ErrOperacionAnalisisEnConflicto
			},
			http.StatusConflict,
			"conflicto",
		},
		{
			"recibo adulterado",
			func(e *ejecutorAnalisisRRHHPrueba) {
				e.recibo = reciboAnalisisRRHHPrueba(
					ports.OperacionRegistrarAnalisis,
					1,
				)
				e.recibo.OrganizacionRef = "organizacion:otra:http:001"
			},
			http.StatusBadGateway,
			"resultado_no_confiable",
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			autoridad := &autoridadAnalisisRRHHPrueba{
				contexto: contextoCanalAnalisisRRHHPrueba(),
			}
			ejecutor := &ejecutorAnalisisRRHHPrueba{}
			caso.preparar(ejecutor)
			manejador, err := NuevoManejadorAnalisisRRHH(
				autoridad,
				ejecutor,
			)
			if err != nil {
				t.Fatal(err)
			}
			respuesta := httptest.NewRecorder()
			manejador.ServeHTTP(
				respuesta,
				nuevaPeticionAnalisisRRHHPrueba(
					RutaRegistroAnalisisRRHH,
					cuerpoRegistroAnalisisRRHHPrueba(),
				),
			)
			if respuesta.Code != caso.estado ||
				!strings.Contains(
					respuesta.Body.String(),
					`"codigo":"`+caso.codigo+`"`,
				) || strings.Contains(respuesta.Body.String(), privado) ||
				!strings.Contains(
					respuesta.Body.String(),
					`"clave_i18n":"api.contratacion_temporal.cobertura.error.`,
				) {
				t.Fatalf("error no opaco: %d %s", respuesta.Code, respuesta.Body)
			}
			comprobarRespuestaSeguraAnalisisRRHH(t, respuesta)
		})
	}
}
