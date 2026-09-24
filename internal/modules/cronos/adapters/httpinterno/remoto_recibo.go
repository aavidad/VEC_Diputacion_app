package httpinterno

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

const RutaRecuperarReciboMarcajeRemoto = "/api/interna/cronos/marcajes/remoto/recibo"
const cabeceraClaveOperacionRemota = "X-Cronos-Clave-Operacion"
const cabeceraMovimientoRemoto = "X-Cronos-Movimiento"

var claveOperacionRecuperacionRemota = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{7,127}$`)

// La recuperación tiene una autoridad V3 de lectura propia distinta de la
// escritura. No consulta el permiso de teletrabajo actual ni crea marcajes.
type ResolverRecuperacionMarcajeRemoto interface {
	ResolverRecuperacionMarcajeRemoto(*http.Request) (ports.ContextoRecuperacionMarcajeRemoto, error)
}

type CasoUsoRecuperacionMarcajeRemoto interface {
	RecuperarReciboMarcajeRemoto(context.Context, ports.ContextoRecuperacionMarcajeRemoto, ports.SolicitudMarcajePropio) (ports.ReciboMarcajePropio, error)
}

type ManejadorRecuperacionMarcajeRemoto struct {
	resolver ResolverRecuperacionMarcajeRemoto
	casoUso  CasoUsoRecuperacionMarcajeRemoto
}

func NuevoManejadorRecuperacionMarcajeRemoto(casoUso CasoUsoRecuperacionMarcajeRemoto, resolver ResolverRecuperacionMarcajeRemoto) (*ManejadorRecuperacionMarcajeRemoto, error) {
	if dependenciaCronosHTTPNula(casoUso) || dependenciaCronosHTTPNula(resolver) {
		return nil, ports.ErrDependenciaNoDisponible
	}
	return &ManejadorRecuperacionMarcajeRemoto{resolver: resolver, casoUso: casoUso}, nil
}

func (m *ManejadorRecuperacionMarcajeRemoto) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r.URL == nil || r.URL.Path != RutaRecuperarReciboMarcajeRemoto || r.URL.RawPath != "" || r.URL.RawQuery != "" {
		errorJSON(w, http.StatusNotFound, "no_disponible")
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		errorJSON(w, http.StatusMethodNotAllowed, "metodo_no_permitido")
		return
	}
	if r.ContentLength != 0 || len(r.TransferEncoding) != 0 {
		errorJSON(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	solicitud, ok := solicitudRecuperacionRemota(r.Header)
	if !ok {
		errorJSON(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	contexto, err := m.resolver.ResolverRecuperacionMarcajeRemoto(r)
	if err != nil {
		responderErrorAccesoCronos(w, err)
		return
	}
	if contexto.CanalAcreditado.Validar() != nil || contexto.CanalAcreditado.OrigenRef() != domain.OrigenMarcajeRemoto {
		responderErrorAccesoCronos(w, ports.ErrDependenciaNoDisponible)
		return
	}
	if _, err := contexto.OrdenLectura.ContextoActor(); err != nil || contexto.OrdenLectura.ProveedorMaterial() == nil {
		responderErrorAccesoCronos(w, ports.ErrDependenciaNoDisponible)
		return
	}
	recibo, err := m.casoUso.RecuperarReciboMarcajeRemoto(r.Context(), contexto, solicitud)
	if err != nil {
		switch {
		case errors.Is(err, ports.ErrDependenciaNoDisponible):
			responderErrorAccesoCronos(w, err)
		case errors.Is(err, ports.ErrMarcajeRemotoNoEncontrado):
			errorJSON(w, http.StatusNotFound, "ausencia_confirmada")
		case errors.Is(err, ports.ErrClaveOperacionEnConflicto):
			errorJSON(w, http.StatusConflict, "conflicto")
		default:
			responderErrorAccesoCronos(w, err)
		}
		return
	}
	if recibo.Referencia == "" || recibo.MarcajeOriginalRef != "marcaje:cronos:"+solicitud.ClaveOperacion || recibo.InstanteUTC.IsZero() || !recibo.Replay {
		responderErrorAccesoCronos(w, ports.ErrDependenciaNoDisponible)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"recibo": recibo})
}

func solicitudRecuperacionRemota(h http.Header) (ports.SolicitudMarcajePropio, bool) {
	claves := h.Values(cabeceraClaveOperacionRemota)
	movimientos := h.Values(cabeceraMovimientoRemoto)
	if len(claves) != 1 || len(movimientos) != 1 || !claveOperacionRecuperacionRemota.MatchString(claves[0]) {
		return ports.SolicitudMarcajePropio{}, false
	}
	movimiento := domain.PunchKind(movimientos[0])
	switch movimiento {
	case domain.PunchEntry, domain.PunchExit, domain.PunchPauseStart, domain.PunchPauseEnd:
		return ports.SolicitudMarcajePropio{ClaveOperacion: claves[0], Movimiento: movimiento}, true
	default:
		return ports.SolicitudMarcajePropio{}, false
	}
}
