package meritoshttp

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"

	"vec-diputacion-granada/internal/modules/meritos/domain"
	"vec-diputacion-granada/internal/modules/meritos/ports"
)

const RutaRevisionRRHH = "/api/vec/meritos/rrhh/revision"

// El preparador consulta el detalle RRHH actual de Bolsa y entrega la persona
// objetivo desde ese módulo; ningún campo HTTP puede sustituirla.
type PreparadorRevisionRRHH interface {
	PrepararRevisionRRHH(*http.Request, string) (ports.SolicitudRevisionRRHH, error)
}

type HandlerRevisionRRHH struct {
	preparador PreparadorRevisionRRHH
	lector     ports.LectorRevisionRRHH
}

func NuevoHandlerRevisionRRHH(p PreparadorRevisionRRHH,
	l ports.LectorRevisionRRHH) (http.Handler, error) {
	if p == nil || l == nil {
		return nil, ports.ErrRevisionRRHHNoDisponible
	}
	return &HandlerRevisionRRHH{preparador: p, lector: l}, nil
}

func (h *HandlerRevisionRRHH) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	if h == nil || h.preparador == nil || h.lector == nil {
		responderRevisionRRHH(w, http.StatusServiceUnavailable, ports.ErrRevisionRRHHNoDisponible)
		return
	}
	if r == nil || r.URL == nil || r.URL.Path != RutaRevisionRRHH ||
		r.URL.EscapedPath() != RutaRevisionRRHH || r.URL.RawPath != "" || r.URL.ForceQuery {
		responderRevisionRRHH(w, http.StatusNotFound, ports.ErrRevisionRRHHSolicitud)
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		responderRevisionRRHH(w, http.StatusMethodNotAllowed, ports.ErrRevisionRRHHSolicitud)
		return
	}
	if r.ContentLength != 0 || len(r.TransferEncoding) != 0 {
		responderRevisionRRHH(w, http.StatusBadRequest, ports.ErrRevisionRRHHSolicitud)
		return
	}
	valores, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil || len(valores) < 1 || len(valores) > 3 ||
		len(valores["solicitud_ref"]) != 1 || len(valores["limite"]) > 1 ||
		len(valores["cursor"]) > 1 {
		responderRevisionRRHH(w, http.StatusBadRequest, ports.ErrRevisionRRHHSolicitud)
		return
	}
	for clave := range valores {
		if clave != "solicitud_ref" && clave != "limite" && clave != "cursor" {
			responderRevisionRRHH(w, http.StatusBadRequest, ports.ErrRevisionRRHHSolicitud)
			return
		}
	}
	ref := valores.Get("solicitud_ref")
	limite := 25
	if _, presente := valores["limite"]; presente {
		limite, err = strconv.Atoi(valores.Get("limite"))
		if err != nil || limite < 1 || limite > 25 {
			responderRevisionRRHH(w, http.StatusBadRequest, ports.ErrRevisionRRHHSolicitud)
			return
		}
	}
	cursor := valores.Get("cursor")
	if !domain.ReferenciaValida(ref) || len(ref) != len("solicitud_inscripcion_")+64 ||
		cursor != "" && !domain.ReferenciaValida(cursor) {
		responderRevisionRRHH(w, http.StatusBadRequest, ports.ErrRevisionRRHHSolicitud)
		return
	}
	solicitud, err := h.preparador.PrepararRevisionRRHH(r, ref)
	if err != nil {
		if errors.Is(err, ports.ErrRevisionRRHHNoDisponible) {
			responderRevisionRRHH(w, http.StatusServiceUnavailable, ports.ErrRevisionRRHHNoDisponible)
			return
		}
		responderRevisionRRHH(w, http.StatusForbidden, ports.ErrRevisionRRHHDenegada)
		return
	}
	solicitud.Limite, solicitud.Cursor = limite, cursor
	if solicitud.SolicitudRef != ref || solicitud.Validar() != nil {
		responderRevisionRRHH(w, http.StatusServiceUnavailable, ports.ErrRevisionRRHHNoDisponible)
		return
	}
	pagina, err := h.lector.ConsultarRevisionRRHH(r.Context(), solicitud)
	if err != nil {
		if errors.Is(err, ports.ErrRevisionRRHHDenegada) {
			responderRevisionRRHH(w, http.StatusForbidden, err)
			return
		}
		responderRevisionRRHH(w, http.StatusServiceUnavailable, ports.ErrRevisionRRHHNoDisponible)
		return
	}
	if pagina.Esquema != ports.EsquemaRevisionRRHH || pagina.Hechos == nil ||
		pagina.ReciboAcceso.AuditoriaRef == "" || pagina.ReciboAcceso.CorrelacionRef == "" {
		responderRevisionRRHH(w, http.StatusServiceUnavailable, ports.ErrRevisionRRHHNoDisponible)
		return
	}
	b, err := json.Marshal(struct {
		Data ports.PaginaRevisionRRHH `json:"data"`
	}{Data: pagina})
	if err != nil {
		responderRevisionRRHH(w, http.StatusServiceUnavailable, ports.ErrRevisionRRHHNoDisponible)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(b)
}

func responderRevisionRRHH(w http.ResponseWriter, codigo int, err error) {
	w.WriteHeader(codigo)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
