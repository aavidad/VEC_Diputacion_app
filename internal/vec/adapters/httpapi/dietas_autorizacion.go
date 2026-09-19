package httpapi

import (
	"context"
	"errors"
	"net/http"
)

// AutoridadPeticionRutasDietas verifica identidad y canal de la petición y
// confirma el consumo nominal V3 antes del acceso. No fabrica Principal.
type AutoridadPeticionRutasDietas interface {
	AutorizarPeticionRutaDietas(context.Context, *http.Request) error
}

var (
	ErrRutaDietasNoAutenticada = errors.New("dietas: autenticacion requerida")
	ErrRutaDietasDenegada      = errors.New("dietas: acceso denegado")
	ErrRutaDietasNoDisponible  = errors.New("dietas: autorizacion no disponible")
)

func (h *Handler) atenderRutaDietas(w http.ResponseWriter, r *http.Request) bool {
	var metodo, ruta string
	var destino http.Handler
	switch vecPath(r.URL.Path) {
	case "/dietas/route-catalog":
		metodo, ruta, destino = http.MethodGet, "/api/vec/dietas/route-catalog", h.catalogoRutaDietas
	case "/dietas/road-route":
		metodo, ruta, destino = http.MethodPost, "/api/vec/dietas/road-route", h.roadRoute
	default:
		return false
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r.URL.Path != ruta || r.URL.RawQuery != "" || !peticionRutaExactaCanonica(r) {
		h.writeError(w, http.StatusNotFound, "dietas: ruta no encontrada")
		return true
	}
	if r.Method != metodo {
		w.Header().Set("Allow", metodo)
		h.writeError(w, http.StatusMethodNotAllowed, "dietas: metodo no permitido")
		return true
	}
	if dependenciaRutaExactaNula(h.autoridadRutasDietas) || dependenciaRutaExactaNula(destino) {
		h.writeError(w, http.StatusServiceUnavailable, "dietas: servicio no disponible")
		return true
	}
	if err := h.autoridadRutasDietas.AutorizarPeticionRutaDietas(r.Context(), r); err != nil {
		estado := http.StatusServiceUnavailable
		if errors.Is(err, ErrRutaDietasNoAutenticada) {
			estado = http.StatusUnauthorized
		} else if errors.Is(err, ErrRutaDietasDenegada) {
			estado = http.StatusForbidden
		}
		h.writeError(w, estado, "dietas: acceso no disponible")
		return true
	}
	destino.ServeHTTP(w, r)
	return true
}
