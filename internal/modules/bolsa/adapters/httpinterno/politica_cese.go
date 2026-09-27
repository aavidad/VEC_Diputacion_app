package httpinterno

import (
	"context"
	"errors"
	"net/http"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
)

const RutaPoliticaCese = "/api/vec/bolsa/politica-cese"
const EsquemaPoliticaCese = "vec.bolsa.rrhh.politica_cese.v1"

// La frontera preparadora obtiene identidad, motivo y correlación de las
// autoridades del servidor. Ningún valor del navegador completa la orden V3.
type PreparadorConsultaPoliticaCese interface {
	PrepararConsultaPoliticaCese(*http.Request) (ports.OrdenConsultaPoliticaCese, error)
}

type ConsultorPoliticaCese interface {
	Consultar(context.Context, ports.OrdenConsultaPoliticaCese) (domain.PoliticaCese, error)
}

type HandlerPoliticaCese struct {
	preparador PreparadorConsultaPoliticaCese
	consultor  ConsultorPoliticaCese
}

func NuevoHandlerPoliticaCese(p PreparadorConsultaPoliticaCese, c ConsultorPoliticaCese) (http.Handler, error) {
	if p == nil || c == nil {
		return nil, ports.ErrConsultaPoliticaCeseNoDisponible
	}
	return &HandlerPoliticaCese{preparador: p, consultor: c}, nil
}

func (h *HandlerPoliticaCese) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || r == nil || r.URL == nil || h.preparador == nil || h.consultor == nil {
		responderError(w, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	if r.Method == http.MethodHead {
		w = escritorSinCuerpo{ResponseWriter: w}
	}
	if r.URL.Path != RutaPoliticaCese || r.URL.RawPath != "" || r.URL.EscapedPath() != RutaPoliticaCese {
		responderError(w, http.StatusNotFound, "recurso_no_encontrado")
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		responderError(w, http.StatusMethodNotAllowed, "metodo_no_permitido")
		return
	}
	if !entradaVaciaYPermitida(r) {
		responderError(w, http.StatusBadRequest, "peticion_no_permitida")
		return
	}
	orden, err := h.preparador.PrepararConsultaPoliticaCese(r)
	if err != nil {
		responderErrorPoliticaCese(w, err)
		return
	}
	politica, err := h.consultor.Consultar(r.Context(), orden)
	if err != nil || politica.Validar() != nil {
		responderErrorPoliticaCese(w, err)
		return
	}
	responderJSON(w, http.StatusOK, map[string]any{"data": map[string]any{
		"esquema": EsquemaPoliticaCese,
		"politica_cese": map[string]any{
			"version": politica.Version, "catalogo_ref": politica.CatalogoRef,
			"catalogo_sha256": politica.CatalogoSHA256, "mapeo": politica.Mapeo,
			"meses_general": politica.MesesGeneral, "meses_acumulacion": politica.MesesAcumulacion,
			"computo": politica.Computo, "estado": politica.Estado,
			"publicada_en": politica.PublicadaEn.UTC().Format(time.RFC3339Nano),
		},
	}})
}

func responderErrorPoliticaCese(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrAutenticacionInternaAusente):
		responderError(w, http.StatusUnauthorized, "autenticacion_requerida")
	case errors.Is(err, vd.ErrAutorizacionDenegada), errors.Is(err, vd.ErrPermissionDenied):
		responderError(w, http.StatusForbidden, "acceso_denegado")
	default:
		responderError(w, http.StatusServiceUnavailable, "servicio_no_disponible")
	}
}
