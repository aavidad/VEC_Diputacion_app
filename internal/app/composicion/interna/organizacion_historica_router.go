package interna

import (
	"net/http"

	"vec-diputacion-granada/internal/vec/adapters/httpapi"
)

type enrutadorOrganizacionHistorica struct{ siguiente, consulta http.Handler }

func nuevoEnrutadorOrganizacionHistorica(siguiente, consulta http.Handler) (http.Handler, error) {
	if manejadorNulo(siguiente) {
		return nil, ErrAPIInternaNoDisponible
	}
	return &enrutadorOrganizacionHistorica{siguiente, consulta}, nil
}

func (e *enrutadorOrganizacionHistorica) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if e == nil || r == nil || r.URL == nil {
		responderPuenteSeguimiento(w, http.StatusServiceUnavailable)
		return
	}
	if r.URL.Path == httpapi.RutaOrganizacionHistoricaPersonal {
		if manejadorNulo(e.consulta) {
			responderPuenteSeguimiento(w, http.StatusNotFound)
			return
		}
		e.consulta.ServeHTTP(w, r)
		return
	}
	e.siguiente.ServeHTTP(w, r)
}
