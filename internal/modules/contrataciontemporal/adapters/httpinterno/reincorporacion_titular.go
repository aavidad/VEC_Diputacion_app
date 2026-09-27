package httpinterno

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// RutaReincorporacionesTitular registra el retorno acreditado del titular de
// una sustitución ya cesada. El evento de retorno no cambia la disponibilidad
// de Bolsa: ese efecto corresponde al cese CT publicado con anterioridad.
const RutaReincorporacionesTitular = "/api/vec/contratacion-temporal/reincorporaciones-titular"

type EjecutorReincorporacionTitular interface {
	RegistrarReincorporacionTitular(context.Context, application.SolicitudRegistrarReincorporacionTitular) (ports.ReciboReincorporacionTitular, error)
}

func leerContenidoReincorporacion(w http.ResponseWriter, r *http.Request) ([]byte, error) {
	contenido, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maximoCuerpoSeguimiento+1))
	if err != nil || len(contenido) == 0 || len(contenido) > maximoCuerpoSeguimiento {
		return nil, errContenidoSeguimiento
	}
	return contenido, nil
}

func estadoErrorReincorporacionTitular(err error) (int, string) {
	switch {
	case errors.Is(err, ports.ErrAutorizacionDenegada):
		return http.StatusForbidden, "acceso_denegado"
	case errors.Is(err, ports.ErrClaveIdempotenciaUsada):
		return http.StatusConflict, "clave_reutilizada"
	case errors.Is(err, ports.ErrReincorporacionSinCese):
		return http.StatusConflict, "sin_cese"
	case errors.Is(err, ports.ErrReincorporacionCeseNoCoincide):
		return http.StatusConflict, "cese_no_coincide"
	case errors.Is(err, ports.ErrReincorporacionYaRegistrada):
		return http.StatusConflict, "reincorporacion_existente"
	case errors.Is(err, domain.ErrVersionEnConflicto):
		return http.StatusConflict, "version_en_conflicto"
	case errors.Is(err, application.ErrSolicitudReincorporacionTitularInvalida):
		return http.StatusUnprocessableEntity, "contenido_no_valido"
	default:
		return http.StatusServiceUnavailable, "servicio_no_disponible"
	}
}

type manejadorReincorporacionTitular struct {
	autoridad AutoridadCanalSeguimiento
	ejecutor  EjecutorReincorporacionTitular
}

func NuevoManejadorReincorporacionTitular(a AutoridadCanalSeguimiento, e EjecutorReincorporacionTitular) (http.Handler, error) {
	if dependenciaNula(a) || dependenciaNula(e) {
		return nil, ports.ErrOperacionSeguimientoNoDisponible
	}
	return &manejadorReincorporacionTitular{autoridad: a, ejecutor: e}, nil
}

func (h *manejadorReincorporacionTitular) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r == nil || r.URL == nil || r.URL.Path != RutaReincorporacionesTitular || r.URL.RawQuery != "" || r.URL.ForceQuery {
		responderErrorSeguimiento(w, r, http.StatusNotFound, "recurso_no_encontrado")
		return
	}
	if r.Method != http.MethodPost {
		responderErrorSeguimiento(w, r, http.StatusMethodNotAllowed, "metodo_no_permitido")
		return
	}
	if !tipoContenidoJSON(r.Header) || cabeceraCoberturaProhibida(r.Header) {
		responderErrorSeguimiento(w, r, http.StatusBadRequest, "peticion_no_permitida")
		return
	}
	canal, err := h.autoridad.ResolverContextoCanalSeguimiento(r.Context())
	if err != nil || !canal.Valido() {
		responderErrorSeguimiento(w, r, http.StatusForbidden, "acceso_denegado")
		return
	}
	var in struct {
		ExpedienteRef     string `json:"expediente_ref"`
		RelacionRef       string `json:"relacion_ref"`
		FechaEfectiva     string `json:"fecha_efectiva"`
		DocumentoRef      string `json:"documento_ref"`
		DocumentoSHA256   string `json:"documento_sha256"`
		VersionEsperada   uint64 `json:"version_esperada"`
		ClaveIdempotencia string `json:"clave_idempotencia"`
	}
	contenido, err := leerContenidoReincorporacion(w, r)
	if err != nil || decodificarCuerpoSeguimiento(contenido, &in) != nil ||
		!domain.ReferenciaOpacaValida(in.ExpedienteRef) || !domain.ReferenciaOpacaValida(in.RelacionRef) {
		responderErrorSeguimiento(w, r, http.StatusUnprocessableEntity, "contenido_no_valido")
		return
	}
	fecha, err := fechaCivilSeguimiento(in.FechaEfectiva)
	if err != nil {
		responderErrorSeguimiento(w, r, http.StatusUnprocessableEntity, "contenido_no_valido")
		return
	}
	recibo, err := h.ejecutor.RegistrarReincorporacionTitular(r.Context(), application.SolicitudRegistrarReincorporacionTitular{
		Canal: canal, ExpedienteRef: in.ExpedienteRef, RelacionRef: in.RelacionRef,
		FechaEfectiva: fecha, DocumentoRef: in.DocumentoRef, DocumentoSHA256: in.DocumentoSHA256,
		VersionEsperada: in.VersionEsperada, ClaveIdempotencia: in.ClaveIdempotencia,
	})
	if err != nil {
		estado, codigo := estadoErrorReincorporacionTitular(err)
		responderErrorSeguimiento(w, r, estado, codigo, err)
		return
	}
	if recibo.ExpedienteRef != in.ExpedienteRef || recibo.RelacionRef != in.RelacionRef ||
		recibo.FechaEfectiva != fecha.Format(time.DateOnly) || recibo.OrganizacionRef != canal.OrganizacionRef ||
		!domain.InstanteUTCCanonico(recibo.RegistradaEn) {
		responderErrorSeguimiento(w, r, http.StatusBadGateway, "resultado_no_confiable")
		return
	}
	responderJSONCobertura(w, r, http.StatusCreated, map[string]any{"data": map[string]any{
		"esquema":        "vec.contratacion-temporal.recibo-reincorporacion-titular.v1",
		"expediente_ref": recibo.ExpedienteRef, "relacion_ref": recibo.RelacionRef,
		"fecha_efectiva": recibo.FechaEfectiva, "version_anterior": recibo.VersionAnterior,
		"version_resultante": recibo.VersionResultante, "recibo_ref": recibo.ReciboRef,
		"evento_ref": recibo.EventoRef, "cese_evento_ref": recibo.CeseEventoRef,
		"cese_recibo_ref": recibo.CeseReciboRef, "registrada_en": recibo.RegistradaEn.UTC().Format(time.RFC3339Nano),
		"estado_bolsa": "pendiente_confirmacion",
	}})
}
