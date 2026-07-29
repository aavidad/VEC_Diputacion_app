package httpinterno

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestManejadorConsultaRRHHCierraRutaMetodoYTiposExactos(t *testing.T) {
	casos := []struct {
		nombre    string
		preparar  func(*http.Request)
		estado    int
		comprobar func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			"ruta ajena",
			func(r *http.Request) { r.URL.Path += "/" },
			http.StatusNotFound,
			nil,
		},
		{
			"query",
			func(r *http.Request) { r.URL.RawQuery = "perfil=admin" },
			http.StatusNotFound,
			nil,
		},
		{
			"método",
			func(r *http.Request) { r.Method = http.MethodGet },
			http.StatusMethodNotAllowed,
			func(t *testing.T, w *httptest.ResponseRecorder) {
				if w.Header().Get("Allow") != http.MethodPost {
					t.Fatalf("Allow=%q", w.Header().Get("Allow"))
				}
			},
		},
		{
			"Content-Type ausente",
			func(r *http.Request) { r.Header.Del("Content-Type") },
			http.StatusUnsupportedMediaType,
			nil,
		},
		{
			"Content-Type con charset",
			func(r *http.Request) {
				r.Header.Set("Content-Type", "application/json; charset=utf-8")
			},
			http.StatusUnsupportedMediaType,
			nil,
		},
		{
			"Content-Type múltiple",
			func(r *http.Request) {
				r.Header.Add("Content-Type", "application/json")
			},
			http.StatusUnsupportedMediaType,
			nil,
		},
		{
			"Accept ausente",
			func(r *http.Request) { r.Header.Del("Accept") },
			http.StatusNotAcceptable,
			nil,
		},
		{
			"Accept comodín",
			func(r *http.Request) { r.Header.Set("Accept", "*/*") },
			http.StatusNotAcceptable,
			nil,
		},
		{
			"Accept múltiple",
			func(r *http.Request) { r.Header.Add("Accept", "application/json") },
			http.StatusNotAcceptable,
			nil,
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			consultor := &consultorCuadroRRHHPrueba{}
			manejador, err := NuevoManejadorConsultaCuadroRRHH(consultor)
			if err != nil {
				t.Fatal(err)
			}
			peticion := nuevaPeticionConsultaRRHHPrueba(
				RutaConsultaCuadroRRHH,
				cuerpoCuadroRRHHPrueba(),
			)
			caso.preparar(peticion)
			respuesta := httptest.NewRecorder()
			manejador.ServeHTTP(respuesta, peticion)
			if respuesta.Code != caso.estado {
				t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body)
			}
			if consultor.llamadas != 0 {
				t.Fatal("la denegación invocó negocio")
			}
			if caso.comprobar != nil {
				caso.comprobar(t, respuesta)
			}
			comprobarCabecerasConsultaRRHH(t, respuesta)
		})
	}
}

func TestManejadorConsultaRRHHRechazaAutoridadEnCabeceras(t *testing.T) {
	for _, cabecera := range []string{
		"Cookie", "Authorization", "Remote-User", "X-Remote-User",
		"X-Actor", "X-Perfil", "X-Organizacion", "X-Forwarded-User",
		"X-Auth-Context", "X-Vec-Perfil", "Role", "Via", "Connection",
	} {
		t.Run(cabecera, func(t *testing.T) {
			consultor := &consultorCuadroRRHHPrueba{}
			manejador, err := NuevoManejadorConsultaCuadroRRHH(consultor)
			if err != nil {
				t.Fatal(err)
			}
			peticion := nuevaPeticionConsultaRRHHPrueba(
				RutaConsultaCuadroRRHH,
				cuerpoCuadroRRHHPrueba(),
			)
			peticion.Header.Set(cabecera, "autoridad_fabricada")
			respuesta := httptest.NewRecorder()
			manejador.ServeHTTP(respuesta, peticion)
			if respuesta.Code != http.StatusBadRequest {
				t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body)
			}
			if consultor.llamadas != 0 {
				t.Fatal("la cabecera prohibida invocó negocio")
			}
		})
	}
}

func TestManejadorConsultaRRHHRechazaJSONAbiertoOConAutoridad(t *testing.T) {
	casos := []struct {
		nombre string
		cuerpo string
		estado int
	}{
		{"duplicada", `{"filtros":{},"filtros":{},"paginacion":{"limite":1}}`, 400},
		{"desconocida", `{"filtros":{},"paginacion":{"limite":1},"extra":1}`, 400},
		{"trailing", cuerpoCuadroRRHHPrueba() + `{}`, 400},
		{"nulo", `null`, 400},
		{"array", `[]`, 400},
		{"clave no canónica", `{"Filtros":{},"paginacion":{"limite":1}}`, 400},
		{"decimal", `{"filtros":{},"paginacion":{"limite":1.0}}`, 400},
		{
			"actor",
			`{"filtros":{},"paginacion":{"limite":1},"actor_ref":"actor:admin"}`,
			400,
		},
		{
			"perfil",
			`{"filtros":{"perfil":"admin"},"paginacion":{"limite":1}}`,
			400,
		},
		{
			"organización",
			`{"filtros":{},"paginacion":{"limite":1},"organizacion_ref":"org:1"}`,
			400,
		},
		{"filtros ausentes", `{"paginacion":{"limite":1}}`, 422},
		{"límite ausente", `{"filtros":{},"paginacion":{}}`, 422},
		{"filtro inválido", `{"filtros":{"texto":"*"},"paginacion":{"limite":1}}`, 422},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			consultor := &consultorCuadroRRHHPrueba{}
			manejador, err := NuevoManejadorConsultaCuadroRRHH(consultor)
			if err != nil {
				t.Fatal(err)
			}
			respuesta := httptest.NewRecorder()
			manejador.ServeHTTP(
				respuesta,
				nuevaPeticionConsultaRRHHPrueba(
					RutaConsultaCuadroRRHH,
					caso.cuerpo,
				),
			)
			if respuesta.Code != caso.estado {
				t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body)
			}
			if consultor.llamadas != 0 {
				t.Fatal("el cuerpo denegado invocó negocio")
			}
		})
	}
}

func TestManejadorConsultaDetalleRRHHRechazaAutoridadYFormaAbierta(
	t *testing.T,
) {
	casos := []struct {
		nombre string
		cuerpo string
		estado int
	}{
		{"versión ausente", `{"expediente_ref":"expediente:ct:0001"}`, 422},
		{
			"actor",
			`{"expediente_ref":"expediente:ct:0001",` +
				`"version_observada":0,"actor_ref":"actor:admin"}`,
			400,
		},
		{
			"versión insegura",
			`{"expediente_ref":"expediente:ct:0001",` +
				`"version_observada":9007199254740992}`,
			422,
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			consultor := &consultorDetalleRRHHPrueba{}
			manejador, err := NuevoManejadorConsultaDetalleRRHH(consultor)
			if err != nil {
				t.Fatal(err)
			}
			respuesta := httptest.NewRecorder()
			manejador.ServeHTTP(
				respuesta,
				nuevaPeticionConsultaRRHHPrueba(
					RutaConsultaDetalleRRHH,
					caso.cuerpo,
				),
			)
			if respuesta.Code != caso.estado {
				t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body)
			}
			if consultor.llamadas != 0 {
				t.Fatal("el cuerpo denegado invocó negocio")
			}
		})
	}
}

func TestManejadoresConsultaRRHHAplicanLimiteExactoEnAmbasRutas(t *testing.T) {
	casos := []struct {
		nombre string
		ruta   string
		base   string
		nuevo  func() (http.Handler, *int)
	}{
		{
			"cuadro",
			RutaConsultaCuadroRRHH,
			`{"filtros":{},"paginacion":{"limite":1}}`,
			func() (http.Handler, *int) {
				consultor := &consultorCuadroRRHHPrueba{
					pagina: paginaRRHHPrueba(),
				}
				manejador, _ := NuevoManejadorConsultaCuadroRRHH(consultor)
				return manejador, &consultor.llamadas
			},
		},
		{
			"detalle",
			RutaConsultaDetalleRRHH,
			`{"expediente_ref":"expediente:ct:0001","version_observada":0}`,
			func() (http.Handler, *int) {
				consultor := &consultorDetalleRRHHPrueba{
					detalle: detalleRRHHPrueba(),
				}
				manejador, _ := NuevoManejadorConsultaDetalleRRHH(consultor)
				return manejador, &consultor.llamadas
			},
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			manejador, llamadas := caso.nuevo()
			exacto := caso.base + strings.Repeat(
				" ",
				MaximoCuerpoConsultaCuadroRRHHBytes-len(caso.base),
			)
			respuesta := httptest.NewRecorder()
			manejador.ServeHTTP(
				respuesta,
				nuevaPeticionConsultaRRHHPrueba(caso.ruta, exacto),
			)
			if respuesta.Code != http.StatusOK || *llamadas != 1 {
				t.Fatalf("límite exacto: estado=%d llamadas=%d", respuesta.Code, *llamadas)
			}

			manejador, llamadas = caso.nuevo()
			excedido := exacto + " "
			peticion := nuevaPeticionConsultaRRHHPrueba(caso.ruta, excedido)
			peticion.ContentLength = -1
			respuesta = httptest.NewRecorder()
			manejador.ServeHTTP(respuesta, peticion)
			if respuesta.Code != http.StatusRequestEntityTooLarge || *llamadas != 0 {
				t.Fatalf("streaming: estado=%d llamadas=%d", respuesta.Code, *llamadas)
			}

			manejador, llamadas = caso.nuevo()
			peticion = nuevaPeticionConsultaRRHHPrueba(caso.ruta, caso.base)
			peticion.ContentLength = MaximoCuerpoConsultaCuadroRRHHBytes + 1
			respuesta = httptest.NewRecorder()
			manejador.ServeHTTP(respuesta, peticion)
			if respuesta.Code != http.StatusRequestEntityTooLarge || *llamadas != 0 {
				t.Fatalf("declarado: estado=%d llamadas=%d", respuesta.Code, *llamadas)
			}
		})
	}
}

func TestManejadorConsultaRRHHPreservaCancelacionPreviaSinInvocarNegocio(
	t *testing.T,
) {
	consultor := &consultorCuadroRRHHPrueba{}
	manejador, err := NuevoManejadorConsultaCuadroRRHH(consultor)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	peticion := nuevaPeticionConsultaRRHHPrueba(
		RutaConsultaCuadroRRHH,
		cuerpoCuadroRRHHPrueba(),
	).WithContext(ctx)
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusRequestTimeout || consultor.llamadas != 0 {
		t.Fatalf("estado=%d llamadas=%d", respuesta.Code, consultor.llamadas)
	}
}

func TestManejadorConsultaRRHHPreservaCancelacionDuranteNegocio(t *testing.T) {
	ctx, cancelar := context.WithCancel(context.Background())
	consultor := &consultorCuadroRRHHPrueba{
		pagina: paginaRRHHPrueba(),
		alConsultar: func(context.Context) {
			cancelar()
		},
	}
	manejador, err := NuevoManejadorConsultaCuadroRRHH(consultor)
	if err != nil {
		t.Fatal(err)
	}
	peticion := nuevaPeticionConsultaRRHHPrueba(
		RutaConsultaCuadroRRHH,
		cuerpoCuadroRRHHPrueba(),
	).WithContext(ctx)
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusRequestTimeout || consultor.llamadas != 1 {
		t.Fatalf("estado=%d llamadas=%d", respuesta.Code, consultor.llamadas)
	}
}

func TestManejadorConsultaRRHHRechazaProyeccionCorrupta(t *testing.T) {
	casos := []struct {
		nombre    string
		manejador func() http.Handler
		ruta      string
		cuerpo    string
	}{
		{
			"cuadro sin instante",
			func() http.Handler {
				consultor := &consultorCuadroRRHHPrueba{}
				manejador, _ := NuevoManejadorConsultaCuadroRRHH(consultor)
				return manejador
			},
			RutaConsultaCuadroRRHH,
			cuerpoCuadroRRHHPrueba(),
		},
		{
			"detalle sin hitos",
			func() http.Handler {
				detalle := detalleRRHHPrueba()
				detalle.Hitos = nil
				consultor := &consultorDetalleRRHHPrueba{detalle: detalle}
				manejador, _ := NuevoManejadorConsultaDetalleRRHH(consultor)
				return manejador
			},
			RutaConsultaDetalleRRHH,
			cuerpoDetalleRRHHPrueba(),
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			respuesta := httptest.NewRecorder()
			caso.manejador().ServeHTTP(
				respuesta,
				nuevaPeticionConsultaRRHHPrueba(caso.ruta, caso.cuerpo),
			)
			if respuesta.Code != http.StatusBadGateway ||
				!strings.Contains(
					respuesta.Body.String(),
					`"codigo":"resultado_no_confiable"`,
				) {
				t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body)
			}
		})
	}
}

func TestRespuestaConsultaRRHHEstaAcotada(t *testing.T) {
	respuesta := httptest.NewRecorder()
	responderJSONConsultaRRHH(
		respuesta,
		http.StatusOK,
		struct {
			Dato string `json:"dato"`
		}{Dato: strings.Repeat("x", MaximoRespuestaConsultaRRHHBytes)},
	)
	if respuesta.Code != http.StatusInternalServerError ||
		respuesta.Body.Len() >= MaximoRespuestaConsultaRRHHBytes ||
		strings.Contains(respuesta.Body.String(), strings.Repeat("x", 32)) {
		t.Fatalf("respuesta no acotada: estado=%d tamaño=%d", respuesta.Code, respuesta.Body.Len())
	}
}

func TestCabecerasConsultaRRHHNoPuedenEmitirCookie(t *testing.T) {
	consultor := &consultorCuadroRRHHPrueba{}
	manejador, err := NuevoManejadorConsultaCuadroRRHH(consultor)
	if err != nil {
		t.Fatal(err)
	}
	respuesta := httptest.NewRecorder()
	respuesta.Header().Add("Set-Cookie", "sesion=secreta")
	peticion := httptest.NewRequest(
		http.MethodPost,
		RutaConsultaCuadroRRHH,
		bytes.NewBufferString(cuerpoCuadroRRHHPrueba()),
	)
	peticion.Header["Content-Type"] = []string{"application/json"}
	peticion.Header["content-type"] = []string{"application/json"}
	peticion.Header.Set("Accept", "application/json")
	manejador.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusUnsupportedMediaType ||
		respuesta.Header().Get("Set-Cookie") != "" ||
		consultor.llamadas != 0 {
		t.Fatalf(
			"estado=%d cookie=%q llamadas=%d",
			respuesta.Code,
			respuesta.Header().Get("Set-Cookie"),
			consultor.llamadas,
		)
	}
}
