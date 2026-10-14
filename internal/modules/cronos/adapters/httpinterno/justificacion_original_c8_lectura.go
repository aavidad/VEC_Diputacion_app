package httpinterno

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

const (
	RutaOriginalC8Propio      = "/api/interna/cronos/permisos/justificantes/propio"
	RutaOriginalC8DetalleRRHH = "/api/interna/cronos/permisos/justificantes/detalle"
	RutaOriginalC8BandejaRRHH = "/api/interna/cronos/permisos/justificantes/bandeja"
)

type ManejadorLecturaOriginalC8 struct {
	resolver   ResolverOriginalC8
	capturador ports.CapturadorLecturaOriginalC8
	lector     ports.LectorOriginalC8
}

func NuevoManejadorLecturaOriginalC8(r ResolverOriginalC8, c ports.CapturadorLecturaOriginalC8,
	l ports.LectorOriginalC8) (*ManejadorLecturaOriginalC8, error) {
	if dependenciaCronosHTTPNula(r) || dependenciaCronosHTTPNula(c) || dependenciaCronosHTTPNula(l) {
		return nil, ports.ErrJustificacionNoDisponible
	}
	return &ManejadorLecturaOriginalC8{r, c, l}, nil
}

func (m *ManejadorLecturaOriginalC8) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cabecerasJSON(w)
	if r == nil || r.URL == nil || r.URL.RawPath != "" ||
		(r.URL.Path != RutaOriginalC8Propio && r.URL.Path != RutaOriginalC8DetalleRRHH && r.URL.Path != RutaOriginalC8BandejaRRHH) {
		errorJSON(w, http.StatusNotFound, "no_disponible")
		return
	}
	if !soloMetodo(w, r, http.MethodGet) {
		return
	}
	if m == nil || dependenciaCronosHTTPNula(m.resolver) || dependenciaCronosHTTPNula(m.capturador) || dependenciaCronosHTTPNula(m.lector) {
		responderErrorAccesoCronos(w, ports.ErrJustificacionNoDisponible)
		return
	}
	bandeja := r.URL.Path == RutaOriginalC8BandejaRRHH
	var ref string
	if bandeja {
		if r.URL.RawQuery != "" {
			errorJSON(w, http.StatusBadRequest, "peticion_invalida")
			return
		}
	} else {
		q, e := url.ParseQuery(r.URL.RawQuery)
		if e != nil || len(q) != 1 || len(q["solicitud_ref"]) != 1 || !domain.SolicitudPermisoRefValida(q.Get("solicitud_ref")) {
			errorJSON(w, http.StatusBadRequest, "peticion_invalida")
			return
		}
		ref = q.Get("solicitud_ref")
	}
	orden, err := m.resolver.ResolverOriginal(r)
	if err != nil {
		responderErrorAccesoCronos(w, err)
		return
	}
	if bandeja {
		captura, e := m.capturador.CapturarBandejaOriginal(r.Context(), orden)
		if e != nil {
			responderErrorAccesoCronos(w, e)
			return
		}
		resultado, e := m.lector.ConsultarBandejaOriginal(r.Context(), captura)
		if e != nil {
			responderErrorLecturaOriginalC8(w, e)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"pendientes": resultado.Pendientes})
		return
	}
	var captura ports.CapturaLecturaOriginalC8
	if r.URL.Path == RutaOriginalC8Propio {
		captura, err = m.capturador.CapturarLecturaOriginalPropia(r.Context(), orden, ref)
	} else {
		captura, err = m.capturador.CapturarLecturaOriginalRRHH(r.Context(), orden, ref)
	}
	if err != nil {
		responderErrorAccesoCronos(w, err)
		return
	}
	resultado, err := m.lector.ConsultarOriginal(r.Context(), captura, ref)
	if err != nil {
		responderErrorLecturaOriginalC8(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"solicitud_ref": resultado.SolicitudRef, "estado_permiso": resultado.EstadoPermiso,
		"version_permiso": resultado.VersionPermiso, "pendiente_justificar": resultado.PendienteJustificar,
		"historia": resultado.Historia,
	})
}

func responderErrorLecturaOriginalC8(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrJustificacionInvalida):
		errorJSON(w, http.StatusBadRequest, "peticion_invalida")
	case errors.Is(err, domain.ErrJustificacionConflicto):
		errorJSON(w, http.StatusConflict, "estado_cambiado")
	case errors.Is(err, ports.ErrResolucionNoCompetente):
		errorJSON(w, http.StatusForbidden, "no_competente")
	case errors.Is(err, ports.ErrBandejaOriginalC8DemasiadoGrande):
		errorJSON(w, http.StatusConflict, "bandeja_demasiado_grande")
	default:
		responderErrorAccesoCronos(w, err)
	}
}
