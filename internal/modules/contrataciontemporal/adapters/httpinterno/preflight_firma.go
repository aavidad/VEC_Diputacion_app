package httpinterno

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const RutaPreflightFirmaR5 = "/api/vec/contratacion-temporal/firma/preflight"

const EsquemaPreflightFirmaR5V2 = "vec.contratacion-temporal.preflight-firma.v2"

type ConsultorPreflightFirmaR5 interface {
	Consultar(context.Context, ports.SolicitudPreflightFirmaR5) (ports.ResultadoPreflightFirmaR5, error)
}

type ConsultorPreflightFirmaR5V2 interface {
	ConsultarV2(context.Context, ports.SolicitudPreflightFirmaR5) (ports.ResultadoPreflightFirmaR5V2, error)
}

type manejadorPreflightFirmaR5 struct {
	autoridad   AutoridadContextoCanalCircuitoRRHH
	consultor   ConsultorPreflightFirmaR5
	consultorV2 ConsultorPreflightFirmaR5V2
}

func NuevoManejadorPreflightFirmaR5V2(autoridad AutoridadContextoCanalCircuitoRRHH, consultor ConsultorPreflightFirmaR5V2) (http.Handler, error) {
	if dependenciaNula(autoridad) || dependenciaNula(consultor) {
		return nil, ports.ErrPreflightFirmaR5NoDisponible
	}
	return &manejadorPreflightFirmaR5{autoridad: autoridad, consultorV2: consultor}, nil
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

type resultadoPreflightFirmaR5V2JSON struct {
	resultadoPreflightFirmaR5JSON
	Esquema                 string `json:"esquema"`
	EntradaDocumentoRef     string `json:"entrada_documento_ref"`
	EntradaDocumentoVersion uint64 `json:"entrada_documento_version"`
	EntradaDocumentoSHA256  string `json:"entrada_documento_sha256"`
}

func (h *manejadorPreflightFirmaR5) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || dependenciaNula(h.autoridad) || (dependenciaNula(h.consultor) && dependenciaNula(h.consultorV2)) {
		responderErrorPreflightFirma(w, r, nil, errorServicioConsultaRRHHNoDisponible)
		return
	}
	if !rutaConsultaRRHHExacta(r, RutaPreflightFirmaR5) {
		responderErrorPreflightFirma(w, r, nil, errorRecursoConsultaRRHHNoEncontrado)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		responderErrorPreflightFirma(w, r, nil, errorMetodoConsultaRRHHNoPermitido)
		return
	}
	if problema := validarMetadatosConsultaRRHH(r, MaximoCuerpoConsultaDetalleRRHHBytes); problema != nil {
		responderErrorPreflightFirma(w, r, nil, *problema)
		return
	}
	var entrada entradaPreflightFirmaR5
	if err := decodificarConsultaRRHH(w, r, MaximoCuerpoConsultaDetalleRRHHBytes, &entrada); err != nil {
		responderErrorPreflightFirma(w, r, nil, errorEntradaConsultaRRHH(err))
		return
	}
	if entrada.VersionObservada == nil || entrada.OriginalVersion == nil ||
		!domain.ReferenciaOpacaValida(entrada.ExpedienteRef) || !domain.ClaveDocumentoFirmaValida(entrada.Documento) ||
		!domain.ReferenciaOpacaValida(entrada.OriginalRef) || *entrada.VersionObservada == 0 ||
		*entrada.VersionObservada > 9007199254740991 || *entrada.OriginalVersion == 0 || *entrada.OriginalVersion > 9007199254740991 {
		responderErrorPreflightFirma(w, r, nil, errorContenidoConsultaRRHHNoValido)
		return
	}
	canal, err := h.autoridad.ResolverContextoCanalCircuitoRRHH(r.Context())
	if r.Context().Err() != nil {
		responderErrorPreflightFirma(w, r, r.Context().Err(), clasificarErrorConsultaRRHH(r.Context().Err()))
		return
	}
	if err != nil || !canal.valido() {
		responderErrorPreflightFirma(w, r, nil, errorRecursoConsultaRRHHNoEncontrado)
		return
	}
	q := ports.SolicitudPreflightFirmaR5{
		Canal: ports.SolicitudConsultaCircuitoRRHH{
			AutenticacionRef: canal.AutenticacionRef, SesionRef: canal.SesionRef,
			PerfilRef: canal.PerfilRef, OrganizacionRef: canal.OrganizacionRef,
			ExpedienteRef: entrada.ExpedienteRef, VersionObservada: *entrada.VersionObservada,
		}, Documento: entrada.Documento, OriginalRef: entrada.OriginalRef, OriginalVersion: *entrada.OriginalVersion,
	}
	var resultado ports.ResultadoPreflightFirmaR5
	var resultadoV2 *ports.ResultadoPreflightFirmaR5V2
	if !dependenciaNula(h.consultorV2) {
		v2, fallo := h.consultorV2.ConsultarV2(r.Context(), q)
		err = fallo
		resultado, resultadoV2 = v2.ResultadoPreflightFirmaR5, &v2
	} else {
		resultado, err = h.consultor.Consultar(r.Context(), q)
	}
	if r.Context().Err() != nil {
		err = r.Context().Err()
	}
	if err != nil {
		responderErrorPreflightFirma(w, r, err, errorPreflightFirmaR5(err))
		return
	}
	if resultadoV2 != nil && application.ValidarResultadoPreflightFirmaR5V2(*resultadoV2, q) != nil {
		responderErrorPreflightFirma(w, r, nil, errorResultadoConsultaRRHHNoConfiable)
		return
	}
	if application.ValidarResultadoPreflightFirmaR5(resultado, q) != nil {
		responderErrorPreflightFirma(w, r, nil, errorResultadoConsultaRRHHNoConfiable)
		return
	}
	vista := resultadoPreflightFirmaR5JSON{
		VersionExpediente: resultado.VersionExpediente, Documento: resultado.Documento,
		CatalogoRef: resultado.CatalogoRef, CatalogoHuella: resultado.CatalogoHuella,
		PasoPendiente: resultado.PasoPendiente, OriginalRef: resultado.OriginalRef,
		OriginalVersion: resultado.OriginalVersion, ViasDisponibles: resultado.ViasDisponibles,
	}
	var data any = vista
	if resultadoV2 != nil {
		data = resultadoPreflightFirmaR5V2JSON{resultadoPreflightFirmaR5JSON: vista,
			Esquema: EsquemaPreflightFirmaR5V2, EntradaDocumentoRef: resultadoV2.EntradaDocumentoRef,
			EntradaDocumentoVersion: resultadoV2.EntradaDocumentoVersion, EntradaDocumentoSHA256: resultadoV2.EntradaDocumentoHuella}
	}
	responderJSONFirmaNominal(w, r, http.StatusOK, map[string]any{"data": data}, MaximoRespuestaConsultaRRHHBytes)
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

func (e *entradaPreflightFirmaR5) UnmarshalJSON(contenido []byte) error {
	if !camposFirmaNominalExactos(contenido, "expediente_ref", "version_observada", "documento", "original_ref", "original_version") {
		return errEntradaConsultaRRHHInvalida
	}
	type entradaSinMetodo entradaPreflightFirmaR5
	return json.Unmarshal(contenido, (*entradaSinMetodo)(e))
}
