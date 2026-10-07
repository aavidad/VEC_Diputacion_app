package httpapi

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/ports"
)

func TestRutasExactasBolsaAuditanDenegacionYCierranSiFallaElRegistro(t *testing.T) {
	rutas := []string{
		"/api/vec/bolsa/mi-bolsa", "/api/vec/bolsa/mi-bolsa/historial",
		"/api/vec/bolsa/mi-bolsa/solicitudes", "/api/vec/bolsa/mi-bolsa/respuestas",
		"/api/vec/bolsa/mi-bolsa/solicitudes-documentales",
		"/api/vec/bolsa/mi-bolsa/disposiciones", "/api/vec/bolsa/mi-bolsa/contacto",
	}
	for _, ruta := range rutas {
		for _, caso := range []struct {
			nombre string
			fallo  error
			actor  bool
			motivo ports.MotivoAuditoriaFronteraRutaExacta
		}{
			{"401", ErrAutenticacionRutaExactaRequerida, false, ports.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida},
			{"403", ErrAccesoRutaExactaDenegado, true, ports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado},
		} {
			t.Run(ruta+caso.nombre, func(t *testing.T) {
				manejador := &manejadorExactoPrueba{}
				registrador := &registradorAuditoriaFronteraRutaExactaEspia{}
				h, err := NewHandlerSoloRutasExactas([]RutaExacta{{Ruta: ruta, Manejador: manejador}},
					autoridadRutasExactasPrueba{err: caso.fallo}, registrador)
				if err != nil {
					t.Fatal(err)
				}
				metodo := http.MethodGet
				if ruta == "/api/vec/bolsa/mi-bolsa/solicitudes-documentales" {
					metodo = http.MethodPost
				}
				peticion := httptest.NewRequest(metodo, ruta+"?dato=privado", nil)
				if caso.actor {
					ctx, err := ConActorVerificadoAuditoriaBolsa(peticion.Context(), actorOrganizacionHistoricaPrueba(t))
					if err != nil {
						t.Fatal(err)
					}
					peticion = peticion.WithContext(ctx)
				}
				respuesta := httptest.NewRecorder()
				h.ServeHTTP(respuesta, peticion)
				estado := http.StatusUnauthorized
				if caso.actor {
					estado = http.StatusForbidden
				}
				ordenes := registrador.ordenesRegistradas()
				if respuesta.Code != estado || len(ordenes) != 1 || ordenes[0].Validar() != nil ||
					ordenes[0].Superficie != ports.SuperficieAuditoriaFronteraRutaExactaBolsaCandidato ||
					ordenes[0].Ruta != ruta || ordenes[0].Motivo != caso.motivo {
					t.Fatalf("estado=%d ordenes=%#v", respuesta.Code, ordenes)
				}
				actor := ""
				if caso.actor {
					actor = actorOrganizacionHistoricaPrueba(t).PersonaRef
				}
				if ordenes[0].ActorRef != actor {
					t.Fatalf("actor no minimizado: %q", ordenes[0].ActorRef)
				}
				if llamadas, _, _ := manejador.estado(); llamadas != 0 {
					t.Fatalf("negocio invocado %d veces", llamadas)
				}
				registrador.err = errors.New("detalle privado de PostgreSQL")
				respuesta = httptest.NewRecorder()
				h.ServeHTTP(respuesta, peticion)
				if respuesta.Code != http.StatusServiceUnavailable ||
					strings.Contains(respuesta.Body.String(), "detalle privado") ||
					strings.Contains(respuesta.Body.String(), "per_") {
					t.Fatalf("fallo de auditoria: estado=%d cuerpo=%s", respuesta.Code, respuesta.Body.String())
				}
			})
		}
	}
}

func TestRutaExactaBolsaSinRegistradorOActorVerificadoCierra(t *testing.T) {
	const ruta = "/api/vec/bolsa/mi-bolsa"
	manejador := &manejadorExactoPrueba{}
	if h, err := NewHandlerSoloRutasExactas([]RutaExacta{{Ruta: ruta, Manejador: manejador}},
		autoridadRutasExactasPrueba{err: ErrAccesoRutaExactaDenegado}, nil); h != nil ||
		!errors.Is(err, ErrRutaExactaInvalida) {
		t.Fatalf("se compuso sin registrador: (%T, %v)", h, err)
	}
	registrador := &registradorAuditoriaFronteraRutaExactaEspia{}
	h, err := NewHandlerSoloRutasExactas([]RutaExacta{{Ruta: ruta, Manejador: manejador}},
		autoridadRutasExactasPrueba{err: ErrAccesoRutaExactaDenegado}, registrador)
	if err != nil {
		t.Fatal(err)
	}
	respuesta := httptest.NewRecorder()
	h.ServeHTTP(respuesta, httptest.NewRequest(http.MethodGet, ruta, nil))
	if respuesta.Code != http.StatusServiceUnavailable || len(registrador.ordenesRegistradas()) != 0 {
		t.Fatalf("sin actor verificado: estado=%d ordenes=%#v", respuesta.Code, registrador.ordenesRegistradas())
	}
	if llamadas, _, _ := manejador.estado(); llamadas != 0 {
		t.Fatalf("negocio invocado %d veces", llamadas)
	}
}
