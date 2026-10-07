package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/ports"
)

func TestAuditoriaFronteraPreparacionNoOcultaFallo(t *testing.T) {
	for _, ruta := range []string{
		"/api/vec/seleccion/preparacion-bases/guardar",
		"/api/vec/seleccion/preparacion-bases/consultar",
		"/api/vec/bolsa/reglas-baremo/borradores/alta",
		"/api/vec/bolsa/reglas-baremo/versiones/consultar",
		"/api/vec/bolsa/reglas-baremo/recibos/recuperar",
	} {
		for _, caso := range []struct {
			nombre  string
			ausente bool
			fallo   error
			estado  int
		}{
			{"confirmada", false, nil, http.StatusForbidden},
			{"fallida", false, errors.New("fallo infraestructura"), http.StatusServiceUnavailable},
			{"ausente", true, nil, http.StatusServiceUnavailable},
		} {
			t.Run(ruta+"/"+caso.nombre, func(t *testing.T) {
				registrador := &registradorAuditoriaFronteraRutaExactaEspia{err: caso.fallo}
				negocio := 0
				h := &Handler{
					rutasExactas:          map[string]http.Handler{ruta: http.HandlerFunc(func(http.ResponseWriter, *http.Request) { negocio++ })},
					autoridadRutasExactas: &autoridadRutasExactasEspia{err: ErrAccesoRutaExactaDenegado},
				}
				if !caso.ausente {
					h.registradorAuditoriaFronteraRutasExactas = registrador
				}
				r := httptest.NewRequest(http.MethodPost, ruta, strings.NewReader(`{"actor":"persona libre"}`))
				r.Header.Set("X-Actor", "persona libre")
				r = r.WithContext(context.WithValue(r.Context(), claveActorAuditoriaBolsa{}, "persona:otra"))
				w := httptest.NewRecorder()
				h.ServeHTTP(w, r)
				if w.Code != caso.estado || negocio != 0 {
					t.Fatalf("estado=%d negocio=%d", w.Code, negocio)
				}
				ordenes := registrador.ordenesRegistradas()
				if caso.ausente {
					if len(ordenes) != 0 {
						t.Fatal("registrador ausente invocado")
					}
					return
				}
				if len(ordenes) != 1 {
					t.Fatal("denegación sin una única auditoría")
				}
				esperada := ports.SuperficieAuditoriaFronteraRutaExactaBolsaReglasBaremo
				if strings.Contains(ruta, "/seleccion/") {
					esperada = ports.SuperficieAuditoriaFronteraRutaExactaSeleccionPreparacionBases
				}
				o := ordenes[0]
				if o.Superficie != esperada || o.Ruta != ruta || o.Motivo != ports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado || o.ActorRef != "" || o.Validar() != nil {
					t.Fatalf("orden no nominal/minimizada: %#v", o)
				}
			})
		}
	}
}

func TestAuditoriaFronteraPreparacionSoloPOSTCanonico(t *testing.T) {
	for _, ruta := range []string{
		"/api/vec/seleccion/preparacion-bases/guardar",
		"/api/vec/seleccion/preparacion-bases/consultar",
		"/api/vec/bolsa/reglas-baremo/borradores/alta",
		"/api/vec/bolsa/reglas-baremo/versiones/consultar",
		"/api/vec/bolsa/reglas-baremo/recibos/recuperar",
	} {
		for _, metodo := range []string{http.MethodGet, http.MethodPut, http.MethodDelete, http.MethodOptions} {
			autoridad := &autoridadRutasExactasEspia{err: ErrAccesoRutaExactaDenegado}
			registrador := &registradorAuditoriaFronteraRutaExactaEspia{}
			negocio := 0
			h := &Handler{rutasExactas: map[string]http.Handler{ruta: http.HandlerFunc(func(http.ResponseWriter, *http.Request) { negocio++ })},
				autoridadRutasExactas: autoridad, registradorAuditoriaFronteraRutasExactas: registrador}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(metodo, ruta, nil))
			n, _ := autoridad.estado()
			if w.Code != http.StatusNotFound || n != 0 || negocio != 0 || len(registrador.ordenesRegistradas()) != 0 {
				t.Fatal("método ajeno alcanzó autoridad/auditoría/negocio")
			}
		}
		if peticionRutaExactaCanonica(httptest.NewRequest(http.MethodPost, ruta+"?actor=libre", nil)) {
			t.Fatal("query de preparación admitida")
		}
	}
}

func TestAuditoriaFronteraBaremoConservaAutenticacionRequerida(t *testing.T) {
	for _, ruta := range []string{
		"/api/vec/bolsa/reglas-baremo/borradores/alta",
		"/api/vec/bolsa/reglas-baremo/versiones/consultar",
		"/api/vec/bolsa/reglas-baremo/recibos/recuperar",
	} {
		registrador := &registradorAuditoriaFronteraRutaExactaEspia{}
		negocio := 0
		h := &Handler{
			rutasExactas:                             map[string]http.Handler{ruta: http.HandlerFunc(func(http.ResponseWriter, *http.Request) { negocio++ })},
			autoridadRutasExactas:                    &autoridadRutasExactasEspia{err: ErrAutenticacionRutaExactaRequerida},
			registradorAuditoriaFronteraRutasExactas: registrador,
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, ruta, nil))
		ordenes := registrador.ordenesRegistradas()
		if w.Code != http.StatusUnauthorized || negocio != 0 || len(ordenes) != 1 ||
			ordenes[0].Motivo != ports.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida || ordenes[0].ActorRef != "" {
			t.Fatal("rechazo de autenticación sin auditoría nominal")
		}
	}
}
