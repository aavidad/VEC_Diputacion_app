package contactopropio

import (
	"context"
	"net/http"

	"vec-diputacion-granada/internal/shared/i18n"
	"vec-diputacion-granada/internal/vec/ports"
)

type EjecutorVersionContactoPropio interface {
	VersionPropia(context.Context) (ports.ResultadoVersionContactoUsuario, error)
}
type manejadorContactoConVersion struct {
	post     http.Handler
	version  EjecutorVersionContactoPropio
	catalogo *i18n.Catalog
}

func (h *manejadorContactoConVersion) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.post == nil || dependenciaContactoPropioNula(h.version) || h.catalogo == nil {
		responderContactoPropio(w, nil, http.StatusForbidden, claveErrorAccesoDenegado)
		return
	}
	if r == nil || r.URL == nil || !rutaContactoPropioExacta(r) {
		responderContactoPropio(w, h.catalogo, http.StatusNotFound, claveErrorNoEncontrado)
		return
	}
	if r.Method != http.MethodGet {
		h.post.ServeHTTP(w, r)
		return
	}
	if r.ContentLength > 0 || r.Body == nil && r.ContentLength != 0 {
		responderContactoPropio(w, h.catalogo, http.StatusBadRequest, claveErrorPeticion)
		return
	}
	resultado, err := h.version.VersionPropia(r.Context())
	if err != nil {
		responderContactoPropio(w, h.catalogo, http.StatusForbidden, claveErrorAccesoDenegado)
		return
	}
	responderJSONContactoPropio(w, http.StatusOK, struct {
		Encontrado bool   `json:"encontrado"`
		Version    uint64 `json:"version"`
	}{resultado.Encontrado, resultado.Version})
}
