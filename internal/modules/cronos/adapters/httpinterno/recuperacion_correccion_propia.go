package httpinterno

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

const RutaRecuperarReciboCorreccionPropia = "/api/interna/cronos/correcciones/recibo"

var claveRecuperacionCorreccionPropia = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

type CasoUsoRecuperarReciboCorreccionPropia interface {
	RecuperarRecibo(context.Context, ports.OrdenConsumoCorreccion, ports.ClaveRecuperacionCorreccion) (ports.ReciboCorreccion, error)
}

type ManejadorRecuperacionCorreccionPropia struct {
	resolver ResolverSolicitudCorreccionPropia
	casoUso  CasoUsoRecuperarReciboCorreccionPropia
}

func NuevoManejadorRecuperacionCorreccionPropia(casoUso CasoUsoRecuperarReciboCorreccionPropia, resolver ResolverSolicitudCorreccionPropia) (*ManejadorRecuperacionCorreccionPropia, error) {
	if dependenciaCronosHTTPNula(casoUso) || dependenciaCronosHTTPNula(resolver) {
		return nil, ports.ErrDependenciaNoDisponible
	}
	return &ManejadorRecuperacionCorreccionPropia{resolver: resolver, casoUso: casoUso}, nil
}

func (m *ManejadorRecuperacionCorreccionPropia) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cabecerasJSON(w)
	if r.URL == nil || r.URL.Path != RutaRecuperarReciboCorreccionPropia || r.URL.RawPath != "" {
		errorJSON(w, http.StatusNotFound, "no_disponible")
		return
	}
	if !soloMetodo(w, r, http.MethodGet) {
		return
	}
	if r.ContentLength != 0 || len(r.TransferEncoding) != 0 {
		errorJSON(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	if len(r.URL.RawQuery) == 0 || len(r.URL.RawQuery) > 80 {
		errorJSON(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	valores, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil || len(valores) != 1 || len(valores["clave_operacion"]) != 1 ||
		!claveRecuperacionCorreccionPropia.MatchString(valores.Get("clave_operacion")) {
		errorJSON(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	clave := valores.Get("clave_operacion")
	orden, err := m.resolver.ResolverSolicitudCorreccionPropia(r)
	if err != nil {
		responderErrorAccesoCronos(w, err)
		return
	}
	recibo, err := m.casoUso.RecuperarRecibo(r.Context(), orden, ports.ClaveRecuperacionCorreccion{
		SolicitudRef: "correccion:cronos:" + clave, ClaveOperacion: clave, Paso: domain.PasoSolicitudCorreccion,
	})
	if err != nil {
		if errors.Is(err, domain.ErrCorreccionInvalida) {
			errorJSON(w, http.StatusBadRequest, "peticion_invalida")
			return
		}
		responderErrorAccesoCronos(w, err)
		return
	}
	if !recibo.Replay || recibo.SolicitudRef != "correccion:cronos:"+clave ||
		recibo.Estado != domain.CorreccionPendienteResponsable || recibo.Version != 1 ||
		recibo.ReciboRef == "" || recibo.ActuacionRef == "" || recibo.InstanteUTC.IsZero() {
		responderErrorAccesoCronos(w, ports.ErrDependenciaNoDisponible)
		return
	}
	cuerpo, err := json.Marshal(map[string]any{"recibo": map[string]any{
		"solicitud_ref": recibo.SolicitudRef, "actuacion_ref": recibo.ActuacionRef, "recibo_ref": recibo.ReciboRef,
		"estado": recibo.Estado, "version": recibo.Version, "instante_utc": recibo.InstanteUTC, "replay": recibo.Replay,
	}})
	if err != nil {
		responderErrorAccesoCronos(w, ports.ErrDependenciaNoDisponible)
		return
	}
	if _, err := w.Write(append(cuerpo, '\n')); err != nil {
		slog.Error("cronos: fallo al escribir respuesta de recibo", "error", err)
	}
}
