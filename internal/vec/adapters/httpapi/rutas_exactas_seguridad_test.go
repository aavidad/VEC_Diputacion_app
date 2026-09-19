package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/ports"
)

type autoridadRutasExactasEspia struct {
	mu           sync.Mutex
	invocaciones int
	ruta         string
	err          error
}

type registradorAuditoriaFronteraRutaExactaEspia struct {
	mu                       sync.Mutex
	ordenes                  []ports.OrdenAuditoriaFronteraRutaExacta
	err                      error
	bloquearHastaCancelacion bool
	plazo                    time.Duration
	contexto                 context.Context
	contextoCancelado        bool
	contextoConPlazo         bool
}

func (r *registradorAuditoriaFronteraRutaExactaEspia) RegistrarAuditoriaFronteraRutaExacta(
	ctx context.Context,
	orden ports.OrdenAuditoriaFronteraRutaExacta,
) error {
	r.mu.Lock()
	r.ordenes = append(r.ordenes, orden)
	if limite, existe := ctx.Deadline(); existe {
		r.plazo = time.Until(limite)
	}
	r.contexto = ctx
	r.contextoCancelado = ctx.Err() != nil
	_, r.contextoConPlazo = ctx.Deadline()
	bloquearHastaCancelacion := r.bloquearHastaCancelacion
	err := r.err
	r.mu.Unlock()
	if bloquearHastaCancelacion {
		<-ctx.Done()
		return ctx.Err()
	}
	return err
}

func (r *registradorAuditoriaFronteraRutaExactaEspia) ordenesRegistradas() []ports.OrdenAuditoriaFronteraRutaExacta {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]ports.OrdenAuditoriaFronteraRutaExacta(nil), r.ordenes...)
}

func (r *registradorAuditoriaFronteraRutaExactaEspia) plazoRegistrado() time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.plazo
}

func (r *registradorAuditoriaFronteraRutaExactaEspia) contextoRegistrado() (context.Context, bool, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.contexto, r.contextoCancelado, r.contextoConPlazo
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

func TestRutasExactasAuditanDenegacionTempranaSinAbrirLaRuta(t *testing.T) {
	t.Parallel()
	casos := []struct {
		nombre string
		err    error
		estado int
		motivo ports.MotivoAuditoriaFronteraRutaExacta
	}{
		{
			nombre: "autenticacion requerida",
			err:    ErrAutenticacionRutaExactaRequerida,
			estado: http.StatusUnauthorized,
			motivo: ports.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida,
		},
		{
			nombre: "acceso denegado",
			err:    ErrAccesoRutaExactaDenegado,
			estado: http.StatusForbidden,
			motivo: ports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado,
		},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			manejador := &manejadorExactoPrueba{}
			registrador := &registradorAuditoriaFronteraRutaExactaEspia{
				err: errors.New("fallo durable privado"),
			}
			handler := newTestHandlerWithOptions(t, HandlerOptions{
				RutasExactas: []RutaExacta{{
					Ruta:      rutaAltaContratacionPrueba,
					Manejador: manejador,
				}},
				AutoridadRutasExactas:                    autoridadRutasExactasPrueba{err: caso.err},
				RegistradorAuditoriaFronteraRutasExactas: registrador,
			})
			respuesta := httptest.NewRecorder()
			handler.ServeHTTP(respuesta, httptest.NewRequest(
				http.MethodPost,
				rutaAltaContratacionPrueba+"?secreto=no-registra",
				strings.NewReader("cuerpo que no se registra"),
			))
			if respuesta.Code != caso.estado {
				t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body.String())
			}
			if strings.Contains(respuesta.Body.String(), "fallo durable privado") {
				t.Fatalf("la respuesta revelo el fallo del registrador: %s", respuesta.Body.String())
			}
			var cuerpo struct {
				Error struct {
					CorrelacionRef string `json:"correlacion_ref"`
				} `json:"error"`
			}
			if err := json.NewDecoder(respuesta.Body).Decode(&cuerpo); err != nil {
				t.Fatalf("decodificar respuesta: %v", err)
			}
			ordenes := registrador.ordenesRegistradas()
			if len(ordenes) != 1 {
				t.Fatalf("llamadas al registrador=%d", len(ordenes))
			}
			orden := ordenes[0]
			if orden.Validar() != nil || orden.CorrelacionRef != cuerpo.Error.CorrelacionRef ||
				orden.Motivo != caso.motivo ||
				orden.Superficie != ports.SuperficieAuditoriaFronteraRutaExactaContratacionTemporal ||
				orden.Ruta != rutaAltaContratacionPrueba || orden.ActorRef != "" {
				t.Fatalf("orden inesperada: %#v", orden)
			}
			if llamadas, _, _ := manejador.estado(); llamadas != 0 {
				t.Fatalf("el negocio fue invocado %d veces", llamadas)
			}
		})
	}
}

func TestRutasExactasAcotanElRegistroDeDenegacion(t *testing.T) {
	registrador := &registradorAuditoriaFronteraRutaExactaEspia{
		bloquearHastaCancelacion: true,
	}
	handler := newTestHandlerWithOptions(t, HandlerOptions{
		RutasExactas: []RutaExacta{{
			Ruta:      rutaAltaContratacionPrueba,
			Manejador: &manejadorExactoPrueba{},
		}},
		AutoridadRutasExactas: autoridadRutasExactasPrueba{
			err: ErrAutenticacionRutaExactaRequerida,
		},
		RegistradorAuditoriaFronteraRutasExactas: registrador,
	})
	respuesta := httptest.NewRecorder()
	handler.ServeHTTP(respuesta, httptest.NewRequest(
		http.MethodPost,
		rutaAltaContratacionPrueba,
		nil,
	))
	if respuesta.Code != http.StatusUnauthorized {
		t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body.String())
	}
	plazo := registrador.plazoRegistrado()
	if plazo <= 0 || plazo > plazoMaximoAuditoriaFronteraRutaExacta {
		t.Fatalf("plazo del registrador=%s", plazo)
	}
}

func TestRutasExactasAuditanUnaDenegacionConPeticionCancelada(t *testing.T) {
	type claveContexto struct{}
	registrador := &registradorAuditoriaFronteraRutaExactaEspia{}
	handler := newTestHandlerWithOptions(t, HandlerOptions{
		RutasExactas: []RutaExacta{{
			Ruta:      rutaAltaContratacionPrueba,
			Manejador: &manejadorExactoPrueba{},
		}},
		AutoridadRutasExactas: autoridadRutasExactasPrueba{
			err: ErrAutenticacionRutaExactaRequerida,
		},
		RegistradorAuditoriaFronteraRutasExactas: registrador,
	})
	ctx, cancelar := context.WithCancel(context.WithValue(
		context.Background(),
		claveContexto{},
		"conservar",
	))
	cancelar()
	peticion := httptest.NewRequest(
		http.MethodPost,
		rutaAltaContratacionPrueba,
		nil,
	).WithContext(ctx)
	respuesta := httptest.NewRecorder()
	handler.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusUnauthorized {
		t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body.String())
	}
	if len(registrador.ordenesRegistradas()) != 1 {
		t.Fatalf("llamadas al registrador=%d", len(registrador.ordenesRegistradas()))
	}
	contexto, cancelado, conPlazo := registrador.contextoRegistrado()
	if cancelado || !conPlazo {
		t.Fatalf("contexto de auditoria cancelado=%t con_plazo=%t", cancelado, conPlazo)
	}
	if valor := contexto.Value(claveContexto{}); valor != "conservar" {
		t.Fatalf("valor del contexto=%#v", valor)
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
