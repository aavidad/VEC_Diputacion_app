package httpinterno

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	almacencanonico "vec-diputacion-granada/internal/vec/canonico/almacen"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const (
	RutaOriginalFirmableCT            = "/api/vec/contratacion-temporal/firmas-documento/original"
	EsquemaOriginalFirmableCT         = "vec.contratacion-temporal.original-firmable.v1"
	maximoRespuestaOriginalFirmableCT = ((vecports.LimiteOriginalFirmableCT+2)/3)*4 + (16 << 10)
)

// Preparar reutiliza la custodia inmutable y consume la autoridad nominal V3
// en cada lectura. El transporte no recibe bytes, identidad ni autorizaciones.
type ServicioOriginalFirmableCTHTTP interface {
	Preparar(context.Context, vecports.SolicitudOriginalFirmableCT) (vecports.OriginalFirmableCT, error)
}

type manejadorOriginalFirmableCT struct {
	autoridad AutoridadCanalRegistroFirmaVec
	servicio  ServicioOriginalFirmableCTHTTP
}

func NuevoManejadorOriginalFirmableCT(a AutoridadCanalRegistroFirmaVec, s ServicioOriginalFirmableCTHTTP) (http.Handler, error) {
	if dependenciaNula(a) || dependenciaNula(s) {
		return nil, vecports.ErrOriginalFirmableCTNoDisponible
	}
	return &manejadorOriginalFirmableCT{a, s}, nil
}

type entradaOriginalFirmableCT struct {
	ExpedienteRef    string `json:"expediente_ref"`
	VersionObservada uint64 `json:"version_observada"`
	Documento        string `json:"documento"`
}

func (e *entradaOriginalFirmableCT) UnmarshalJSON(contenido []byte) error {
	if !camposFirmaNominalExactos(contenido, "expediente_ref", "version_observada", "documento") {
		return errEntradaConsultaRRHHInvalida
	}
	type entradaSinMetodo entradaOriginalFirmableCT
	return json.Unmarshal(contenido, (*entradaSinMetodo)(e))
}

func (h *manejadorOriginalFirmableCT) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !rutaRegistroFirmaNominalExacta(r, RutaOriginalFirmableCT) {
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
	var entrada entradaOriginalFirmableCT
	if err := decodificarConsultaRRHH(w, r, MaximoCuerpoConsultaDetalleRRHHBytes, &entrada); err != nil {
		responderErrorPreflightFirma(w, r, err, errorEntradaConsultaRRHH(err))
		return
	}
	if !domain.ReferenciaOpacaValida(entrada.ExpedienteRef) || !domain.ClaveDocumentoFirmaValida(entrada.Documento) ||
		entrada.VersionObservada == 0 || entrada.VersionObservada > 9007199254740991 {
		responderErrorPreflightFirma(w, r, nil, errorContenidoConsultaRRHHNoValido)
		return
	}
	organizacion, err := h.autoridad.ResolverOrganizacionFirmaVec(r.Context())
	if err != nil || !domain.ReferenciaOpacaValida(organizacion) {
		responderErrorAutoridadRegistroFirma(w, r, err, responderErrorRegistroFirmaVec)
		return
	}
	if r.Context().Err() != nil {
		responderErrorPreflightFirma(w, r, r.Context().Err(), clasificarErrorConsultaRRHH(r.Context().Err()))
		return
	}
	solicitud := vecports.SolicitudOriginalFirmableCT{OrganizacionRef: organizacion, ExpedienteRef: entrada.ExpedienteRef,
		Documento: entrada.Documento, OriginalVersion: entrada.VersionObservada}
	original, err := h.servicio.Preparar(r.Context(), solicitud)
	if r.Context().Err() != nil {
		err = r.Context().Err()
	}
	if err != nil {
		responderErrorPreflightFirma(w, r, err, errorOriginalFirmableCT(err))
		return
	}
	defer clear(original.Contenido)
	if !originalFirmableCTConfiable(original, solicitud) {
		responderErrorPreflightFirma(w, r, nil, errorResultadoConsultaRRHHNoConfiable)
		return
	}
	responderJSONFirmaNominal(w, r, http.StatusOK, map[string]any{"data": map[string]any{
		"esquema": EsquemaOriginalFirmableCT, "expediente_ref": entrada.ExpedienteRef,
		"version_observada": entrada.VersionObservada, "documento": entrada.Documento,
		"original_ref": original.Referencia, "original_version": original.Version, "original_sha256": original.HuellaSHA256,
		"tipo_ref": original.TipoRef, "mime": "application/pdf", "pdf_base64": base64.StdEncoding.EncodeToString(original.Contenido),
		"documento_custodiado": documentoCustodiado(organizacion, entrada.ExpedienteRef, original.Referencia, original.Version, original.HuellaSHA256),
	}}, maximoRespuestaOriginalFirmableCT)
}

func originalFirmableCTConfiable(original vecports.OriginalFirmableCT, s vecports.SolicitudOriginalFirmableCT) bool {
	identidad := almacencanonico.IdentidadOriginalCT{OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef,
		Documento: s.Documento, Version: s.OriginalVersion}
	huella := sha256.Sum256(original.Contenido)
	return identidad.Valida() && original.Referencia == identidad.Referencia() && original.Version == s.OriginalVersion &&
		almacencanonico.ReferenciaDocumentoValida(original.TipoRef) && original.HuellaSHA256 == hex.EncodeToString(huella[:]) &&
		len(original.Contenido) >= 8 && len(original.Contenido) <= vecports.LimiteOriginalFirmableCT && bytes.HasPrefix(original.Contenido, []byte("%PDF-"))
}

func errorOriginalFirmableCT(err error) errorPublicoConsultaRRHH {
	switch {
	case errors.Is(err, vecports.ErrOriginalFirmableCTInvalido), errors.Is(err, docports.ErrSolicitudInvalida), errors.Is(err, docports.ErrValidacion):
		return errorContenidoConsultaRRHHNoValido
	case errors.Is(err, vecports.ErrOriginalFirmableCTConflicto), errors.Is(err, docports.ErrConflicto):
		return nuevoErrorConsultaRRHH(http.StatusConflict, "conflicto")
	case errors.Is(err, vecports.ErrOriginalFirmableCTNoEncontrado), errors.Is(err, docports.ErrNoEncontrado), errors.Is(err, docports.ErrAccesoDenegado),
		errors.Is(err, ports.ErrAutorizacionDenegada), errors.Is(err, ports.ErrOriginalFirmaNoAutorizado):
		return errorRecursoConsultaRRHHNoEncontrado
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return clasificarErrorConsultaRRHH(err)
	default:
		return errorServicioConsultaRRHHNoDisponible
	}
}
