package httpinterno

import (
	"encoding/json"
	"errors"
	"net/http"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

// Rutas de las notificaciones de la persona a RRHH (C9). Identidad, empleado
// propio y autorización V3 los aporta el resolver desde la frontera; el
// cliente sólo envía tipo, fecha, texto, adjunto por referencia y huella, la
// notificación que se atiende y la clave de su operación.
const (
	RutaNotificacionesPropias = "/api/interna/cronos/notificaciones/propio"
	RutaRegistrarNotificacion = "/api/interna/cronos/notificaciones/envios"
	RutaBandejaNotificaciones = "/api/interna/cronos/notificaciones/bandeja"
	RutaAtenderNotificacion   = "/api/interna/cronos/notificaciones/atenciones"
)

type ResolverNotificacionesPropias interface {
	ResolverNotificacionesPropias(*http.Request) (ports.OrdenNotificacionesPropias, error)
}

type ResolverBandejaNotificaciones interface {
	ResolverBandejaNotificaciones(*http.Request) (ports.OrdenBandejaNotificaciones, error)
}

func responderErrorNotificacion(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ports.ErrTipoNotificacionNoVigente):
		errorJSON(w, http.StatusConflict, "tipo_no_vigente")
	case errors.Is(err, ports.ErrBandejaNotificacionesDemasiadoGrande):
		// Conflicto con el estado actual: hay más de las que se muestran.
		errorJSON(w, http.StatusConflict, "bandeja_demasiado_grande")
	case errors.Is(err, domain.ErrNotificacionInvalida):
		errorJSON(w, http.StatusBadRequest, "peticion_invalida")
	default:
		responderErrorResolucion(w, err)
	}
}

func escribirRecibo(w http.ResponseWriter, replay bool, recibo any) {
	if !replay {
		w.WriteHeader(http.StatusCreated)
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"recibo": recibo})
}

// ---- La persona ----

type ManejadorNotificacionesPropias struct {
	resolver ResolverNotificacionesPropias
	casoUso  ports.CasoUsoNotificacionesPropias
}

func NuevoManejadorNotificacionesPropias(casoUso ports.CasoUsoNotificacionesPropias, resolver ResolverNotificacionesPropias) (*ManejadorNotificacionesPropias, error) {
	if dependenciaCronosHTTPNula(casoUso) || dependenciaCronosHTTPNula(resolver) {
		return nil, ports.ErrDependenciaNoDisponible
	}
	return &ManejadorNotificacionesPropias{resolver: resolver, casoUso: casoUso}, nil
}

func (m *ManejadorNotificacionesPropias) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cabecerasJSON(w)
	if r.URL == nil || r.URL.RawPath != "" || r.URL.RawQuery != "" || (r.URL.Path != RutaNotificacionesPropias && r.URL.Path != RutaRegistrarNotificacion) {
		errorJSON(w, http.StatusNotFound, "no_disponible")
		return
	}
	if r.URL.Path == RutaRegistrarNotificacion {
		m.registrar(w, r)
		return
	}
	if !soloMetodo(w, r, http.MethodGet) {
		return
	}
	orden, err := m.resolver.ResolverNotificacionesPropias(r)
	if err != nil {
		responderErrorAccesoCronos(w, err)
		return
	}
	c, err := m.casoUso.ConsultarPropias(r.Context(), orden)
	if err != nil {
		responderErrorNotificacion(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(c)
}

func (m *ManejadorNotificacionesPropias) registrar(w http.ResponseWriter, r *http.Request) {
	if !soloMetodo(w, r, http.MethodPost) {
		return
	}
	v, ok := decodificarCadenasAcotadas(w, r, []string{"clave_operacion", "tipo_version_ref", "fecha_referida", "texto"},
		map[string]int{"clave_operacion": 128, "tipo_version_ref": 160, "fecha_referida": 10, "texto": domain.MaximoTextoNotificacion,
			"adjunto_ref": 128, "adjunto_sha256": 64})
	if !ok {
		errorJSON(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	orden, err := m.resolver.ResolverNotificacionesPropias(r)
	if err != nil {
		responderErrorAccesoCronos(w, err)
		return
	}
	recibo, err := m.casoUso.RegistrarNotificacion(r.Context(), orden, ports.PeticionRegistroNotificacion{
		ClaveOperacion: v["clave_operacion"], TipoVersionRef: v["tipo_version_ref"], FechaReferida: v["fecha_referida"], Texto: v["texto"],
		AdjuntoRef: v["adjunto_ref"], AdjuntoSHA256: v["adjunto_sha256"],
	})
	if err != nil {
		responderErrorNotificacion(w, err)
		return
	}
	escribirRecibo(w, recibo.Replay, recibo)
}

// ---- RRHH ----

type ManejadorBandejaNotificaciones struct {
	resolver ResolverBandejaNotificaciones
	casoUso  ports.CasoUsoBandejaNotificaciones
}

func NuevoManejadorBandejaNotificaciones(casoUso ports.CasoUsoBandejaNotificaciones, resolver ResolverBandejaNotificaciones) (*ManejadorBandejaNotificaciones, error) {
	if dependenciaCronosHTTPNula(casoUso) || dependenciaCronosHTTPNula(resolver) {
		return nil, ports.ErrDependenciaNoDisponible
	}
	return &ManejadorBandejaNotificaciones{resolver: resolver, casoUso: casoUso}, nil
}

func (m *ManejadorBandejaNotificaciones) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cabecerasJSON(w)
	if r.URL == nil || r.URL.RawPath != "" || r.URL.RawQuery != "" || (r.URL.Path != RutaBandejaNotificaciones && r.URL.Path != RutaAtenderNotificacion) {
		errorJSON(w, http.StatusNotFound, "no_disponible")
		return
	}
	if r.URL.Path == RutaAtenderNotificacion {
		m.atender(w, r)
		return
	}
	if !soloMetodo(w, r, http.MethodGet) {
		return
	}
	orden, err := m.resolver.ResolverBandejaNotificaciones(r)
	if err != nil {
		responderErrorAccesoCronos(w, err)
		return
	}
	b, err := m.casoUso.ConsultarBandeja(r.Context(), orden)
	if err != nil {
		responderErrorNotificacion(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(b)
}

func (m *ManejadorBandejaNotificaciones) atender(w http.ResponseWriter, r *http.Request) {
	if !soloMetodo(w, r, http.MethodPost) {
		return
	}
	v, ok := decodificarCadenas(w, r, []string{"clave_operacion", "notificacion_ref"}, nil)
	if !ok {
		errorJSON(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	orden, err := m.resolver.ResolverBandejaNotificaciones(r)
	if err != nil {
		responderErrorAccesoCronos(w, err)
		return
	}
	recibo, err := m.casoUso.AtenderNotificacion(r.Context(), orden, ports.PeticionAtencionNotificacion{ClaveOperacion: v["clave_operacion"], NotificacionRef: v["notificacion_ref"]})
	if err != nil {
		responderErrorNotificacion(w, err)
		return
	}
	escribirRecibo(w, recibo.Replay, recibo)
}
