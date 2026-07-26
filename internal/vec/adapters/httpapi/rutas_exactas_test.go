package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"vec-diputacion-granada/internal/vec/adapters/memory"
	"vec-diputacion-granada/internal/vec/application"
)

const (
	rutaAltaContratacionPrueba       = "/api/vec/contratacion-temporal/solicitudes"
	rutaPropuestaCoberturaPrueba     = "/api/vec/contratacion-temporal/cobertura/propuesta"
	rutaDecisionCoberturaPrueba      = "/api/vec/contratacion-temporal/cobertura/decisiones"
	rutaRectificacionCoberturaPrueba = "/api/vec/contratacion-temporal/cobertura/rectificaciones"
)

type manejadorExactoPrueba struct {
	mu           sync.Mutex
	invocaciones int
	ruta         string
	consulta     string
}

type autoridadRutasExactasPrueba struct {
	err error
}

func (a autoridadRutasExactasPrueba) AutorizarRutaExacta(
	context.Context,
	string,
) error {
	return a.err
}

func (m *manejadorExactoPrueba) ServeHTTP(
	respuesta http.ResponseWriter,
	peticion *http.Request,
) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.invocaciones++
	m.ruta = peticion.URL.Path
	m.consulta = peticion.URL.RawQuery
	respuesta.WriteHeader(http.StatusNoContent)
}

func (m *manejadorExactoPrueba) estado() (int, string, string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.invocaciones, m.ruta, m.consulta
}

func TestRutasExactasDeleganRutaCompletaSinIdentidadDeCarcasa(
	t *testing.T,
) {
	t.Parallel()
	rutas := []string{
		rutaAltaContratacionPrueba,
		rutaPropuestaCoberturaPrueba,
		rutaDecisionCoberturaPrueba,
		rutaRectificacionCoberturaPrueba,
	}
	manejadores := make([]*manejadorExactoPrueba, len(rutas))
	declaradas := make([]RutaExacta, len(rutas))
	for indice, ruta := range rutas {
		manejadores[indice] = &manejadorExactoPrueba{}
		declaradas[indice] = RutaExacta{
			Ruta:      ruta,
			Manejador: manejadores[indice],
		}
	}
	handler := newTestHandlerWithOptions(t, HandlerOptions{
		RutasExactas:          declaradas,
		AutoridadRutasExactas: autoridadRutasExactasPrueba{},
	})

	for indice, ruta := range rutas {
		peticion := httptest.NewRequest(
			http.MethodPost,
			ruta+"?traza=opaca",
			nil,
		)
		respuesta := httptest.NewRecorder()
		handler.ServeHTTP(respuesta, peticion)
		if respuesta.Code != http.StatusNoContent {
			t.Fatalf("%s: estado = %d", ruta, respuesta.Code)
		}
		invocaciones, recibida, consulta := manejadores[indice].estado()
		if invocaciones != 1 || recibida != ruta || consulta != "traza=opaca" {
			t.Fatalf(
				"%s: delegacion alterada: llamadas=%d ruta=%q consulta=%q",
				ruta,
				invocaciones,
				recibida,
				consulta,
			)
		}
	}
}

func TestRutasExactasNoAlteranCarcasaNiSePublicanEnDescubrimiento(
	t *testing.T,
) {
	t.Parallel()
	handler := newTestHandlerWithOptions(t, HandlerOptions{
		AutoridadRutasExactas: autoridadRutasExactasPrueba{},
		RutasExactas: []RutaExacta{
			{
				Ruta:      rutaPropuestaCoberturaPrueba,
				Manejador: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
			},
			{
				Ruta:      rutaAltaContratacionPrueba,
				Manejador: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
			},
		},
	})

	respuestaSesion := httptest.NewRecorder()
	handler.ServeHTTP(
		respuestaSesion,
		httptest.NewRequest(http.MethodGet, "/api/vec/session", nil),
	)
	if respuestaSesion.Code != http.StatusOK {
		t.Fatalf(
			"la carcasa dejo de responder: estado=%d cuerpo=%s",
			respuestaSesion.Code,
			respuestaSesion.Body.String(),
		)
	}

	respuestaRaiz := httptest.NewRecorder()
	handler.ServeHTTP(
		respuestaRaiz,
		httptest.NewRequest(http.MethodGet, "/api/vec", nil),
	)
	if respuestaRaiz.Code != http.StatusOK {
		t.Fatalf(
			"descubrimiento: estado=%d cuerpo=%s",
			respuestaRaiz.Code,
			respuestaRaiz.Body.String(),
		)
	}
	var cuerpo struct {
		Datos struct {
			Rutas []string `json:"routes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(respuestaRaiz.Body).Decode(&cuerpo); err != nil {
		t.Fatalf("decodificar descubrimiento: %v", err)
	}
	for _, esperada := range []string{
		rutaAltaContratacionPrueba,
		rutaPropuestaCoberturaPrueba,
	} {
		if ocurrencias(cuerpo.Datos.Rutas, esperada) != 0 {
			t.Fatalf(
				"ruta interna %q fue publicada: %#v",
				esperada,
				cuerpo.Datos.Rutas,
			)
		}
	}
	if len(cuerpo.Datos.Rutas) != len(rutasBaseVEC()) {
		t.Fatalf("descubrimiento inesperado: %#v", cuerpo.Datos.Rutas)
	}
}

func TestRutasExactasSeCopianDefensivamente(t *testing.T) {
	t.Parallel()
	original := &manejadorExactoPrueba{}
	reemplazo := &manejadorExactoPrueba{}
	declaradas := []RutaExacta{{
		Ruta:      rutaAltaContratacionPrueba,
		Manejador: original,
	}}
	handler := newTestHandlerWithOptions(t, HandlerOptions{
		RutasExactas:          declaradas,
		AutoridadRutasExactas: autoridadRutasExactasPrueba{},
	})
	declaradas[0] = RutaExacta{
		Ruta:      rutaDecisionCoberturaPrueba,
		Manejador: reemplazo,
	}

	respuesta := httptest.NewRecorder()
	handler.ServeHTTP(
		respuesta,
		httptest.NewRequest(http.MethodPost, rutaAltaContratacionPrueba, nil),
	)
	if respuesta.Code != http.StatusNoContent {
		t.Fatalf("estado = %d", respuesta.Code)
	}
	if llamadas, _, _ := original.estado(); llamadas != 1 {
		t.Fatalf("el manejador original recibio %d llamadas", llamadas)
	}
	if llamadas, _, _ := reemplazo.estado(); llamadas != 0 {
		t.Fatalf("el manejador sustituto recibio %d llamadas", llamadas)
	}
}

func TestRutasExactasInvalidasFallanAlConstruir(t *testing.T) {
	t.Parallel()
	var manejadorNulo *manejadorExactoPrueba
	manejadorValido := http.HandlerFunc(
		func(http.ResponseWriter, *http.Request) {},
	)
	casos := []struct {
		nombre string
		rutas  []RutaExacta
	}{
		{
			nombre: "repetida",
			rutas: []RutaExacta{
				{Ruta: rutaAltaContratacionPrueba, Manejador: manejadorValido},
				{Ruta: rutaAltaContratacionPrueba, Manejador: manejadorValido},
			},
		},
		{
			nombre: "ruta de la carcasa",
			rutas:  []RutaExacta{{Ruta: "/api/vec/session", Manejador: manejadorValido}},
		},
		{
			nombre: "familia dinamica de puesto",
			rutas: []RutaExacta{{
				Ruta:      "/api/vec/personal/rpt/positions/42",
				Manejador: manejadorValido,
			}},
		},
		{
			nombre: "familia dinamica de categoria",
			rutas: []RutaExacta{{
				Ruta:      "/api/vec/personal/categories/tecnico",
				Manejador: manejadorValido,
			}},
		},
		{
			nombre: "familia dinamica de accion",
			rutas: []RutaExacta{{
				Ruta:      "/api/vec/modules/nuevo/action",
				Manejador: manejadorValido,
			}},
		},
		{
			nombre: "fuera de VEC",
			rutas:  []RutaExacta{{Ruta: "/api/otro/ruta", Manejador: manejadorValido}},
		},
		{
			nombre: "barra final",
			rutas:  []RutaExacta{{Ruta: rutaAltaContratacionPrueba + "/", Manejador: manejadorValido}},
		},
		{
			nombre: "segmento vacio",
			rutas:  []RutaExacta{{Ruta: "/api/vec/contratacion//alta", Manejador: manejadorValido}},
		},
		{
			nombre: "segmento reservado",
			rutas:  []RutaExacta{{Ruta: "/api/vec/contratacion/{id}", Manejador: manejadorValido}},
		},
		{
			nombre: "manejador nulo",
			rutas:  []RutaExacta{{Ruta: rutaAltaContratacionPrueba}},
		},
		{
			nombre: "manejador nulo tipado",
			rutas:  []RutaExacta{{Ruta: rutaAltaContratacionPrueba, Manejador: manejadorNulo}},
		},
		{
			nombre: "mux privado",
			rutas:  []RutaExacta{{Ruta: rutaAltaContratacionPrueba, Manejador: http.NewServeMux()}},
		},
		{
			nombre: "mux global",
			rutas:  []RutaExacta{{Ruta: rutaAltaContratacionPrueba, Manejador: http.DefaultServeMux}},
		},
	}

	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			servicio, err := nuevoServicioVECVacioPrueba()
			if err != nil {
				t.Fatal(err)
			}
			handler, err := NewHandlerWithOptions(
				servicio,
				HandlerOptions{
					RutasExactas:          caso.rutas,
					AutoridadRutasExactas: autoridadRutasExactasPrueba{},
				},
			)
			if handler != nil || !errors.Is(err, ErrRutaExactaInvalida) {
				t.Fatalf(
					"resultado = (%T, %v), se esperaba ruta invalida",
					handler,
					err,
				)
			}
		})
	}
}

func TestRutasExactasNoAceptanPatronesNiCaracteresAmbiguos(t *testing.T) {
	t.Parallel()
	manejador := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	for _, sufijo := range []string{
		"mayuscula",
		"con.punto",
		"con%2fescape",
		"con?consulta",
		"con#fragmento",
		"con\\barra",
		"{variable}",
		"comodin*",
	} {
		ruta := "/api/vec/" + sufijo
		if ruta == "/api/vec/mayuscula" {
			ruta = "/api/vec/Mayuscula"
		}
		if rutaExactaAdicionalValida(ruta) {
			t.Fatalf("ruta ambigua aceptada: %q", ruta)
		}
	}
	if manejadorRutaExactaInvalido(manejador) {
		t.Fatal("un manejador de funcion valido fue rechazado")
	}
}

func TestHandlerNuloFallaCerradoSinDetalle(t *testing.T) {
	t.Parallel()
	var handler *Handler
	respuesta := httptest.NewRecorder()
	handler.ServeHTTP(
		respuesta,
		httptest.NewRequest(http.MethodGet, rutaAltaContratacionPrueba, nil),
	)
	if respuesta.Code != http.StatusServiceUnavailable {
		t.Fatalf("estado = %d", respuesta.Code)
	}
	if respuesta.Body.Len() != 0 {
		t.Fatalf("la respuesta filtro detalle: %q", respuesta.Body.String())
	}
	if respuesta.Header().Get("Cache-Control") != "no-store" ||
		respuesta.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("cabeceras de seguridad incompletas: %v", respuesta.Header())
	}
}

func ocurrencias(valores []string, buscado string) int {
	total := 0
	for _, valor := range valores {
		if valor == buscado {
			total++
		}
	}
	return total
}

func nuevoServicioVECVacioPrueba() (*application.Service, error) {
	almacen := memory.NewStore()
	return application.NewService(almacen, almacen, almacen)
}
