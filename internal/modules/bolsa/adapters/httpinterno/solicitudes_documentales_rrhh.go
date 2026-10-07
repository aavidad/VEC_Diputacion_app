package httpinterno

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/application"
	"vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

const RutaSolicitudesDocumentalesPendientesRRHH = "/api/vec/bolsa/solicitudes-documentales/pendientes"

var referenciaConsultaDocumental = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]{0,511}$`)

type ConsultorSolicitudesDocumentalesRRHH interface {
	ListarSolicitudesDocumentalesRRHH(context.Context, ports.SolicitudCambiarSituacionParticipacion) ([]ports.SolicitudDocumentalPendienteRRHH, error)
}

type HandlerSolicitudesDocumentalesRRHH struct {
	preparador PreparadorSituacionParticipacion
	consultor  ConsultorSolicitudesDocumentalesRRHH
}

func NuevoHandlerSolicitudesDocumentalesRRHH(p PreparadorSituacionParticipacion, c ConsultorSolicitudesDocumentalesRRHH) (http.Handler, error) {
	if p == nil || c == nil {
		return nil, ports.ErrSituacionParticipacionNoDisponible
	}
	return &HandlerSolicitudesDocumentalesRRHH{preparador: p, consultor: c}, nil
}

func (h *HandlerSolicitudesDocumentalesRRHH) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || r == nil || r.URL == nil || r.URL.Path != RutaSolicitudesDocumentalesPendientesRRHH || r.URL.RawPath != "" || r.URL.ForceQuery ||
		r.URL.EscapedPath() != RutaSolicitudesDocumentalesPendientesRRHH {
		responderOperacion(w, http.StatusNotFound, "recurso_no_encontrado")
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		responderOperacion(w, http.StatusMethodNotAllowed, "metodo_no_permitido")
		return
	}
	if r.ContentLength != 0 || len(r.TransferEncoding) != 0 || len(r.Header.Values("Accept")) != 1 || r.Header.Get("Accept") != "application/json" {
		responderOperacion(w, http.StatusBadRequest, "solicitud_invalida")
		return
	}
	valores, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil || len(valores) != 2 || len(valores["bolsa_ref"]) != 1 || len(valores["participacion_ref"]) != 1 ||
		!referenciaConsultaDocumental.MatchString(valores.Get("bolsa_ref")) || !referenciaConsultaDocumental.MatchString(valores.Get("participacion_ref")) {
		responderOperacion(w, http.StatusBadRequest, "solicitud_invalida")
		return
	}
	q, err := h.preparador.PrepararSolicitudCambiarSituacion(r.Context(), EntradaCambiarSituacionParticipacion{
		BolsaRef: valores.Get("bolsa_ref"), ParticipacionRef: valores.Get("participacion_ref"),
		Destino: domain.SituacionDisponible, Motivo: "consulta", ClaveIdempotencia: "consulta",
	})
	if err != nil {
		responderErrorOperacion(w, err)
		return
	}
	items, err := h.consultor.ListarSolicitudesDocumentalesRRHH(r.Context(), q)
	if err != nil {
		if acuse, confirmado := application.AcuseConsultaDocumentalesFallida(err); confirmado {
			w.Header().Set("X-Audit-Ref", acuse.AuditoriaRef)
			w.Header().Set("X-Correlation-Ref", acuse.CorrelacionRef)
		}
		if errors.Is(err, ports.ErrSituacionParticipacionNoDisponible) || errors.Is(err, application.ErrCambioSituacionParticipacionNoDisponible) {
			responderOperacion(w, http.StatusServiceUnavailable, "servicio_no_disponible")
			return
		}
		responderErrorOperacion(w, err)
		return
	}
	type itemHTTP struct {
		SolicitudRef    string  `json:"solicitud_ref"`
		Version         int64   `json:"version"`
		ContenidoSHA256 string  `json:"contenido_sha256"`
		DocumentoRef    string  `json:"documento_ref"`
		DocumentoSHA256 string  `json:"documento_sha256"`
		FechaFinCausa   *string `json:"fecha_fin_causa"`
		Estado          string  `json:"estado"`
		ReciboRef       string  `json:"recibo_ref"`
		RegistradaEn    string  `json:"registrada_en"`
	}
	salida := make([]itemHTTP, 0, len(items))
	for _, item := range items {
		if item.SolicitudRef == "" || item.Version != 1 || item.ContenidoSHA256 == "" || item.DocumentoRef == "" || item.DocumentoSHA256 == "" ||
			item.Estado != "pendiente_rrhh" || item.ReciboRef == "" || item.RegistradaEn.IsZero() {
			responderOperacion(w, http.StatusServiceUnavailable, "servicio_no_disponible")
			return
		}
		var fin *string
		if item.FechaFinCausa != "" {
			fecha, err := time.Parse(time.DateOnly, item.FechaFinCausa)
			if err != nil || fecha.Format(time.DateOnly) != item.FechaFinCausa {
				responderOperacion(w, http.StatusServiceUnavailable, "servicio_no_disponible")
				return
			}
			fin = &item.FechaFinCausa
		}
		salida = append(salida, itemHTTP{item.SolicitudRef, item.Version, item.ContenidoSHA256, item.DocumentoRef, item.DocumentoSHA256,
			fin, item.Estado, item.ReciboRef, item.RegistradaEn.UTC().Format("2006-01-02T15:04:05.000000Z07:00")})
	}
	responderSituacion(w, http.StatusOK, map[string]any{"data": map[string]any{"esquema": "vec.bolsa.rrhh.solicitudes_documentales.v1", "items": salida}})
}
