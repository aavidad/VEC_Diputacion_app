package httpinterno

import (
	"context"
	"errors"
	"net/http"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const RutaPreflightFirmaR5 = "/api/vec/contratacion-temporal/firma/preflight"

type ConsultorPreflightFirmaR5 interface {
	Consultar(context.Context, ports.SolicitudPreflightFirmaR5) (ports.ResultadoPreflightFirmaR5, error)
}

type manejadorPreflightFirmaR5 struct {
	autoridad AutoridadContextoCanalCircuitoRRHH
	consultor ConsultorPreflightFirmaR5
}

func NuevoManejadorPreflightFirmaR5(autoridad AutoridadContextoCanalCircuitoRRHH, consultor ConsultorPreflightFirmaR5) (http.Handler, error) {
	if dependenciaNula(autoridad) || dependenciaNula(consultor) {
		return nil, ports.ErrPreflightFirmaR5NoDisponible
	}
	return &manejadorPreflightFirmaR5{autoridad: autoridad, consultor: consultor}, nil
}

type entradaPreflightFirmaR5 struct {
	ExpedienteRef    string  `json:"expediente_ref"`
	VersionObservada *uint64 `json:"version_observada"`
	Documento        string  `json:"documento"`
	OriginalRef      string  `json:"original_ref"`
	OriginalVersion  *uint64 `json:"original_version"`
}

type resultadoPreflightFirmaR5JSON struct {
	VersionExpediente uint64   `json:"version_expediente"`
	Documento         string   `json:"documento"`
	CatalogoRef       string   `json:"catalogo_ref"`
	CatalogoHuella    string   `json:"catalogo_huella"`
	PasoPendiente     int      `json:"paso_pendiente"`
	OriginalRef       string   `json:"original_ref"`
	OriginalVersion   uint64   `json:"original_version"`
	ViasDisponibles   []string `json:"vias_disponibles"`
}

func (h *manejadorPreflightFirmaR5) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || dependenciaNula(h.autoridad) || dependenciaNula(h.consultor) {
		responderErrorConsultaRRHH(w, r, nil, errorServicioConsultaRRHHNoDisponible)
		return
	}
	if !rutaConsultaRRHHExacta(r, RutaPreflightFirmaR5) {
		responderErrorConsultaRRHH(w, r, nil, errorRecursoConsultaRRHHNoEncontrado)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		responderErrorConsultaRRHH(w, r, nil, errorMetodoConsultaRRHHNoPermitido)
		return
	}
	if problema := validarMetadatosConsultaRRHH(r, MaximoCuerpoConsultaDetalleRRHHBytes); problema != nil {
		responderErrorConsultaRRHH(w, r, nil, *problema)
		return
	}
	var entrada entradaPreflightFirmaR5
	if err := decodificarConsultaRRHH(w, r, MaximoCuerpoConsultaDetalleRRHHBytes, &entrada); err != nil {
		responderErrorConsultaRRHH(w, r, nil, errorEntradaConsultaRRHH(err))
		return
	}
	if entrada.VersionObservada == nil || entrada.OriginalVersion == nil ||
		!domain.ReferenciaOpacaValida(entrada.ExpedienteRef) || !domain.ClaveDocumentoFirmaValida(entrada.Documento) ||
		!domain.ReferenciaOpacaValida(entrada.OriginalRef) || *entrada.VersionObservada == 0 ||
		*entrada.VersionObservada > 9007199254740991 || *entrada.OriginalVersion == 0 || *entrada.OriginalVersion > 9007199254740991 {
		responderErrorConsultaRRHH(w, r, nil, errorContenidoConsultaRRHHNoValido)
		return
	}
	canal, err := h.autoridad.ResolverContextoCanalCircuitoRRHH(r.Context())
	if r.Context().Err() != nil {
		responderErrorConsultaRRHH(w, r, r.Context().Err(), clasificarErrorConsultaRRHH(r.Context().Err()))
		return
	}
	if err != nil || !canal.valido() {
		responderErrorConsultaRRHH(w, r, nil, errorRecursoConsultaRRHHNoEncontrado)
		return
	}
	q := ports.SolicitudPreflightFirmaR5{
		Canal: ports.SolicitudConsultaCircuitoRRHH{
			AutenticacionRef: canal.AutenticacionRef, SesionRef: canal.SesionRef,
			PerfilRef: canal.PerfilRef, OrganizacionRef: canal.OrganizacionRef,
			ExpedienteRef: entrada.ExpedienteRef, VersionObservada: *entrada.VersionObservada,
		}, Documento: entrada.Documento, OriginalRef: entrada.OriginalRef, OriginalVersion: *entrada.OriginalVersion,
	}
	resultado, err := h.consultor.Consultar(r.Context(), q)
	if r.Context().Err() != nil {
		err = r.Context().Err()
	}
	if err != nil {
		responderErrorConsultaRRHH(w, r, nil, errorPreflightFirmaR5(err))
		return
	}
	if application.ValidarResultadoPreflightFirmaR5(resultado, q) != nil {
		responderErrorConsultaRRHH(w, r, nil, errorResultadoConsultaRRHHNoConfiable)
		return
	}
	responderJSONConsultaRRHH(w, r, http.StatusOK, struct {
		Data resultadoPreflightFirmaR5JSON `json:"data"`
	}{Data: resultadoPreflightFirmaR5JSON{
		VersionExpediente: resultado.VersionExpediente, Documento: resultado.Documento,
		CatalogoRef: resultado.CatalogoRef, CatalogoHuella: resultado.CatalogoHuella,
		PasoPendiente: resultado.PasoPendiente, OriginalRef: resultado.OriginalRef,
		OriginalVersion: resultado.OriginalVersion, ViasDisponibles: resultado.ViasDisponibles,
	}})
}

func errorPreflightFirmaR5(err error) errorPublicoConsultaRRHH {
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return clasificarErrorConsultaRRHH(err)
	case errors.Is(err, ports.ErrPreflightFirmaR5Invalido), errors.Is(err, ports.ErrSolicitudFirmaDocumentoInvalida):
		return errorContenidoConsultaRRHHNoValido
	case errors.Is(err, ports.ErrFirmaDocumentoDenegada), errors.Is(err, ports.ErrAutorizacionDenegada),
		errors.Is(err, ports.ErrOriginalFirmaNoAutorizado), errors.Is(err, ports.ErrExpedienteConsultaFirmasNoEncontrado):
		return errorRecursoConsultaRRHHNoEncontrado
	case errors.Is(err, ports.ErrPreflightFirmaR5NoConfiable), errors.Is(err, ports.ErrResultadoFirmaDocumentoInvalido):
		return errorResultadoConsultaRRHHNoConfiable
	case errors.Is(err, domain.ErrVersionEnConflicto):
		return errorConsultaCircuitoRRHH(err)
	default:
		return errorServicioConsultaRRHHNoDisponible
	}
}
