package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"
)

type autoridadRutasExactasEspia struct {
	mu           sync.Mutex
	invocaciones int
	ruta         string
	err          error
}

func (a *autoridadRutasExactasEspia) AutorizarRutaExacta(
	_ context.Context,
	ruta string,
) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.invocaciones++
	a.ruta = ruta
	return a.err
}

func (a *autoridadRutasExactasEspia) estado() (int, string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.invocaciones, a.ruta
}

func TestRutasExactasExigenAutoridadEnLaComposicion(t *testing.T) {
	t.Parallel()
	servicio, err := nuevoServicioVECVacioPrueba()
	if err != nil {
		t.Fatal(err)
	}
	rutas := []RutaExacta{{
		Ruta: rutaAltaContratacionPrueba,
		Manejador: http.HandlerFunc(
			func(http.ResponseWriter, *http.Request) {},
		),
	}}
	handler, err := NewHandlerWithOptions(
		servicio,
		HandlerOptions{RutasExactas: rutas},
	)
	if handler != nil || !errors.Is(err, ErrRutaExactaInvalida) {
		t.Fatalf("ruta sin autoridad = (%T, %v)", handler, err)
	}
	handler, err = NewHandlerWithOptions(
		servicio,
		HandlerOptions{
			AutoridadRutasExactas: autoridadRutasExactasPrueba{},
		},
	)
	if handler != nil || !errors.Is(err, ErrRutaExactaInvalida) {
		t.Fatalf("autoridad sin rutas = (%T, %v)", handler, err)
	}
}

func TestRutasExactasDenieganAntesDelManejador(t *testing.T) {
	t.Parallel()
	casos := []struct {
		nombre string
		err    error
		estado int
		codigo string
	}{
		{
			nombre: "sin autenticacion",
			err:    ErrAutenticacionRutaExactaRequerida,
			estado: http.StatusUnauthorized,
			codigo: "autenticacion_requerida",
		},
		{
			nombre: "sin autorizacion",
			err:    ErrAccesoRutaExactaDenegado,
			estado: http.StatusForbidden,
			codigo: "acceso_denegado",
		},
		{
			nombre: "autoridad no disponible",
			err:    ErrAutoridadRutaExactaNoDisponible,
			estado: http.StatusServiceUnavailable,
			codigo: "servicio_no_disponible",
		},
		{
			nombre: "error privado",
			err:    errors.New("detalle privado"),
			estado: http.StatusServiceUnavailable,
			codigo: "servicio_no_disponible",
		},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			manejador := &manejadorExactoPrueba{}
			handler := newTestHandlerWithOptions(t, HandlerOptions{
				RutasExactas: []RutaExacta{{
					Ruta:      rutaAltaContratacionPrueba,
					Manejador: manejador,
				}},
				AutoridadRutasExactas: autoridadRutasExactasPrueba{
					err: caso.err,
				},
			})
			respuesta := httptest.NewRecorder()
			handler.ServeHTTP(
				respuesta,
				httptest.NewRequest(
					http.MethodPost,
					rutaAltaContratacionPrueba,
					nil,
				),
			)
			if respuesta.Code != caso.estado {
				t.Fatalf(
					"estado=%d cuerpo=%s",
					respuesta.Code,
					respuesta.Body.String(),
				)
			}
			if !strings.Contains(
				respuesta.Body.String(),
				`"codigo":"`+caso.codigo+`"`,
			) || strings.Contains(respuesta.Body.String(), "detalle privado") {
				t.Fatalf("error no redactado: %s", respuesta.Body.String())
			}
			if !regexp.MustCompile(
				`"correlacion_ref":"corr_(?:[0-9a-f]{32}|no_disponible)"`,
			).MatchString(respuesta.Body.String()) {
				t.Fatalf(
					"error sin correlacion opaca: %s",
					respuesta.Body.String(),
				)
			}
			if llamadas, _, _ := manejador.estado(); llamadas != 0 {
				t.Fatalf("se invoco el negocio %d veces", llamadas)
			}
			if respuesta.Header().Get("Set-Cookie") != "" ||
				respuesta.Header().Get("Retry-After") != "" {
				t.Fatalf("cabeceras prohibidas: %v", respuesta.Header())
			}
		})
	}
}

func TestRutasExactasRechazanPeticionNoCanonicaAntesDelManejador(
	t *testing.T,
) {
	t.Parallel()
	manejador := &manejadorExactoPrueba{}
	autoridad := &autoridadRutasExactasEspia{}
	handler := newTestHandlerWithOptions(t, HandlerOptions{
		RutasExactas: []RutaExacta{{
			Ruta:      rutaAltaContratacionPrueba,
			Manejador: manejador,
		}},
		AutoridadRutasExactas: autoridad,
	})
	casos := []func(*http.Request){
		func(p *http.Request) {
			p.URL.RawPath = rutaAltaContratacionPrueba + "%2f"
		},
		func(p *http.Request) { p.URL.Opaque = rutaAltaContratacionPrueba },
		func(p *http.Request) { p.URL.Fragment = "fragmento" },
		func(p *http.Request) { p.URL.RawFragment = "fragmento" },
		func(p *http.Request) { p.URL.ForceQuery = true },
		func(p *http.Request) { p.URL.Scheme = "https" },
		func(p *http.Request) { p.URL.Host = "interno.example" },
	}
	for indice, mutar := range casos {
		peticion := httptest.NewRequest(
			http.MethodPost,
			rutaAltaContratacionPrueba,
			nil,
		)
		mutar(peticion)
		respuesta := httptest.NewRecorder()
		handler.ServeHTTP(respuesta, peticion)
		if respuesta.Code != http.StatusNotFound {
			t.Fatalf("caso %d: estado=%d", indice, respuesta.Code)
		}
	}
	if llamadas, _, _ := manejador.estado(); llamadas != 0 {
		t.Fatalf("se invoco el negocio %d veces", llamadas)
	}
	if llamadas, ruta := autoridad.estado(); llamadas != 0 || ruta != "" {
		t.Fatalf("se invoco la autoridad antes de canonizar: %d %q", llamadas, ruta)
	}
}

func TestRutaExactaAutorizaUnaVezConRutaCompleta(t *testing.T) {
	t.Parallel()
	autoridad := &autoridadRutasExactasEspia{}
	manejador := &manejadorExactoPrueba{}
	handler := newTestHandlerWithOptions(t, HandlerOptions{
		RutasExactas: []RutaExacta{{
			Ruta:      rutaDecisionCoberturaPrueba,
			Manejador: manejador,
		}},
		AutoridadRutasExactas: autoridad,
	})
	respuesta := httptest.NewRecorder()
	handler.ServeHTTP(
		respuesta,
		httptest.NewRequest(
			http.MethodPost,
			rutaDecisionCoberturaPrueba,
			nil,
		),
	)
	if respuesta.Code != http.StatusNoContent {
		t.Fatalf("estado = %d", respuesta.Code)
	}
	if llamadas, ruta := autoridad.estado(); llamadas != 1 ||
		ruta != rutaDecisionCoberturaPrueba {
		t.Fatalf("autoridad = (%d, %q)", llamadas, ruta)
	}
}
