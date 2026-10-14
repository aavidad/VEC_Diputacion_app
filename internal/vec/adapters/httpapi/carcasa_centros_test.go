package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCarcasaCentrosAbreConsultasInformativasSinActos(t *testing.T) {
	handler := newTestHandler(t)
	for _, rol := range []string{"solicitante_centro", "ratificador_centro"} {
		t.Run(rol, func(t *testing.T) {
			for _, caso := range []struct {
				metodo string
				ruta   string
				estado int
			}{
				{http.MethodGet, "/api/vec/session", http.StatusOK},
				{http.MethodGet, "/api/vec/modules", http.StatusOK},
				{http.MethodGet, "/api/vec/menu", http.StatusOK},
				{http.MethodGet, "/api/vec/workspace", http.StatusForbidden},
				{http.MethodPost, "/api/vec/modules/cronos/action", http.StatusForbidden},
				{http.MethodPost, "/api/vec/modules/administracion/action", http.StatusForbidden},
			} {
				peticion := httptest.NewRequest(caso.metodo, caso.ruta, nil)
				peticion.Header.Set("X-Auth-Subject", "actor-centro-prueba")
				peticion.Header.Set("X-Auth-Mechanism", "certificado")
				peticion.Header.Set("X-Auth-Roles", rol)
				respuesta := httptest.NewRecorder()
				handler.ServeHTTP(respuesta, peticion)
				if respuesta.Code != caso.estado {
					t.Errorf("%s %s: estado %d, esperado %d", caso.metodo, caso.ruta, respuesta.Code, caso.estado)
				}
			}
		})
	}
}

func TestCarcasaCentrosRechazaRolAjenoOMezclado(t *testing.T) {
	handler := newTestHandler(t)
	for _, roles := range []string{"centro", "solicitante_centro,tecnico_rrhh"} {
		peticion := httptest.NewRequest(http.MethodGet, "/api/vec/session", nil)
		peticion.Header.Set("X-Auth-Subject", "actor-centro-prueba")
		peticion.Header.Set("X-Auth-Mechanism", "certificado")
		peticion.Header.Set("X-Auth-Roles", roles)
		respuesta := httptest.NewRecorder()
		handler.ServeHTTP(respuesta, peticion)
		if respuesta.Code != http.StatusForbidden {
			t.Errorf("rol %q: estado %d, esperado 403", roles, respuesta.Code)
		}
	}
}
