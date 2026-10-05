package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strings"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type manejadorEntregaPeticionDesarrollo struct {
	proveedor   *proveedorEntregaPeticionDesarrollo
	repositorio ports.RepositorioEntregasPeticionCentro
	servicio    *application.ServicioEntregaPeticionCentro
}

func (m *manejadorEntregaPeticionDesarrollo) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	prepararCabecerasCatalogosAltaContratacionTemporalDesarrollo(w)
	fallo := func(estado int, codigo string) {
		w.WriteHeader(estado)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"codigo": codigo, "clave_i18n": "api.contratacion_temporal.entrega_peticion.error." + codigo}})
	}
	if m == nil || m.proveedor == nil || m.repositorio == nil || m.servicio == nil || r == nil || r.URL == nil {
		fallo(503, "servicio_no_disponible")
		return
	}
	if r.URL.Path != rutaEntregaPeticionCentro || r.URL.RawPath != "" || r.URL.RawQuery != "" || r.URL.ForceQuery || len(r.TransferEncoding) != 0 || len(r.Trailer) != 0 || cabeceraCatalogosAltaContratacionTemporalDesarrolloProhibida(r.Header) {
		fallo(400, "solicitud_invalida")
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		fallo(405, "metodo_no_permitido")
		return
	}
	if _, _, err := m.proveedor.ActorEntregaPeticionCentro(r.Context()); err != nil {
		fallo(falloEntregaPeticionDesarrollo(r.Method, errors.Join(err, r.Context().Err())))
		return
	}
	responder := func(data any) { _ = json.NewEncoder(w).Encode(map[string]any{"data": data}) }
	if r.Method == http.MethodGet {
		if r.ContentLength != 0 {
			fallo(400, "solicitud_invalida")
			return
		}
		filas, err := m.repositorio.ListarPeticionesRRHH(r.Context())
		if err != nil {
			fallo(falloEntregaPeticionDesarrollo(r.Method, errors.Join(err, r.Context().Err())))
			return
		}
		responder(map[string]any{"peticiones": filas, "limite": 50})
		return
	}
	tipo, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || tipo != "application/json" || len(params) > 0 && (len(params) != 1 || !strings.EqualFold(params["charset"], "utf-8")) || r.ContentLength > 1024 || r.Body == nil {
		fallo(400, "solicitud_invalida")
		return
	}
	b, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1024))
	if err != nil || validarClavesJSONUnicas(b) != nil {
		fallo(400, "solicitud_invalida")
		return
	}
	defer clear(b)
	var c ports.ComandoEntregarPeticionCentro
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(&c) != nil || d.Decode(&struct{}{}) != io.EOF || c.Validar() != nil {
		fallo(400, "solicitud_invalida")
		return
	}
	e, err := m.servicio.Entregar(r.Context(), c)
	if err != nil {
		switch {
		case errors.Is(err, application.ErrSolicitudRegistroInvalida), errors.Is(err, ports.ErrNumeroMOADAusente):
			fallo(http.StatusUnprocessableEntity, "solicitud_invalida")
		case errors.Is(err, domain.ErrPeticionCentroInvalida):
			fallo(400, "solicitud_invalida")
		case (errors.Is(err, ports.ErrEntregaPeticionEnConflicto) || errors.Is(err, ports.ErrClaveIdempotenciaUsada)) && causaFalloEntregaPeticionDesarrollo(err) != "autorizacion_denegada":
			fallo(409, "peticion_en_conflicto")
		default:
			fallo(falloEntregaPeticionDesarrollo(r.Method, errors.Join(err, r.Context().Err())))
		}
		return
	}
	// La clave de alta y el dueño de la reserva no salen al navegador. El
	// recibo sí conserva la trazabilidad original y permite retomar el análisis.
	e.ClaveAlta, e.AmbitoAltaHMAC, e.ActorRef, e.PerfilRef = "", "", "", ""
	if e.ReservaCreadaAhora && e.ConfirmadaAhora {
		w.WriteHeader(http.StatusCreated)
	}
	responder(e)
}

// falloEntregaPeticionDesarrollo separa la denegación (403) de la
// indisponibilidad (503) y deja la causa en el registro. Solo escribe un
// código fijo de causa, el método y la ruta: nunca el texto del error, que
// puede arrastrar referencias o datos de la petición.
func falloEntregaPeticionDesarrollo(metodo string, err error) (int, string) {
	causa := causaFalloEntregaPeticionDesarrollo(err)
	if causa == "autorizacion_denegada" {
		slog.Warn("recepción de peticiones de centro denegada", "ruta", rutaEntregaPeticionCentro, "metodo", metodo, "causa", causa)
		return http.StatusForbidden, "operacion_denegada"
	}
	slog.Error("recepción de peticiones de centro no disponible", "ruta", rutaEntregaPeticionCentro, "metodo", metodo, "causa", causa)
	return http.StatusServiceUnavailable, "servicio_no_disponible"
}

// causaFalloEntregaPeticionDesarrollo clasifica primero la indisponibilidad:
// el servicio V3 y el PDP común envuelven sus fallos de fuente, registro,
// instantánea o configuración en ErrAutorizacionDenegada, y eso no es una
// denegación de la persona (patrón del panel interno de Bolsa).
func causaFalloEntregaPeticionDesarrollo(err error) string {
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return "peticion_cancelada_o_vencida"
	case errors.Is(err, errAutorizacionComunDesarrolloNoDisponible), errors.Is(err, vecdomain.ErrConfiguracionAccesoInvalida):
		return "configuracion_autorizacion_no_disponible"
	case errors.Is(err, vecports.ErrFuenteAutorizacionNoDisponible),
		errors.Is(err, vecports.ErrFuenteContextoActorNoDisponible):
		return "fuente_autorizacion_no_disponible"
	case errors.Is(err, vecports.ErrInstantaneaAutorizacionObsoleta):
		return "permiso_vigente_no_operativo"
	case errors.Is(err, vecports.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible),
		errors.Is(err, vecports.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible),
		errors.Is(err, vecports.ErrRegistroDecisionNoDisponible),
		errors.Is(err, vecports.ErrRegistroDenegacionNoDisponible):
		return "registro_decision_no_disponible"
	case errors.Is(err, ports.ErrReciboPeticionCentroNoConfiable):
		return "respuesta_base_no_confiable"
	case errors.Is(err, ports.ErrConsultaRRHHNoDisponible):
		return "sesion_no_disponible"
	case errors.Is(err, ports.ErrPeticionCentroNoDisponible):
		return "persistencia_no_disponible"
	case errors.Is(err, ports.ErrAutorizacionDenegada), errors.Is(err, vecdomain.ErrAutorizacionDenegada):
		return "autorizacion_denegada"
	default:
		return "no_clasificada"
	}
}
