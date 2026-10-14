package httpinterno

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

const RutaEquipoCronosC10 = "/api/interna/cronos/equipos/"

type ResolverConsultaEquipo interface {
	ResolverConsultaEquipo(*http.Request) (ports.OrdenConsultaEquipo, error)
}
type ManejadorEquipoC10 struct {
	resolver ResolverConsultaEquipo
	caso     ports.CasoUsoConsultaEquipo
}

func NuevoManejadorEquipoC10(caso ports.CasoUsoConsultaEquipo, resolver ResolverConsultaEquipo) (*ManejadorEquipoC10, error) {
	if dependenciaCronosHTTPNula(caso) || dependenciaCronosHTTPNula(resolver) {
		return nil, ports.ErrDependenciaNoDisponible
	}
	return &ManejadorEquipoC10{resolver, caso}, nil
}
func (m *ManejadorEquipoC10) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if m == nil || m.caso == nil || m.resolver == nil || r == nil || r.URL == nil {
		errorJSON(w, http.StatusServiceUnavailable, "no_disponible")
		return
	}
	if r.URL.RawPath != "" || !strings.HasPrefix(r.URL.Path, RutaEquipoCronosC10) {
		errorJSON(w, http.StatusNotFound, "no_disponible")
		return
	}
	resto := strings.TrimPrefix(r.URL.Path, RutaEquipoCronosC10)
	trozos := strings.Split(resto, "/")
	if len(trozos) != 2 || trozos[0] == "" || len(trozos[0]) > 128 || strings.ContainsAny(trozos[0], ".%\\") {
		errorJSON(w, http.StatusNotFound, "no_disponible")
		return
	}
	var vista domain.TipoConsultaEquipo
	switch trozos[1] {
	case "presencia":
		vista = domain.ConsultaPresenciaEquipo
	case "incidencias":
		vista = domain.ConsultaIncidenciasEquipo
	default:
		errorJSON(w, http.StatusNotFound, "no_disponible")
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		errorJSON(w, http.StatusMethodNotAllowed, "metodo_no_permitido")
		return
	}
	desde, hasta, cursor, err := parametrosEquipoC10(r.URL.RawQuery)
	if err != nil {
		errorJSON(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	orden, err := m.resolver.ResolverConsultaEquipo(r)
	if err != nil {
		responderErrorAccesoCronos(w, err)
		return
	}
	resultado, err := m.caso.ConsultarEquipo(r.Context(), orden, vista, trozos[0], desde, hasta, cursor)
	if err != nil {
		if errors.Is(err, ports.ErrConsultaEquipoInvalida) {
			errorJSON(w, http.StatusBadRequest, "peticion_invalida")
			return
		}
		responderErrorAccesoCronos(w, err)
		return
	}
	if resultado.Filas == nil {
		resultado.Filas = []ports.FilaConsultaEquipo{}
	}
	_ = json.NewEncoder(w).Encode(resultado)
}
func parametrosEquipoC10(raw string) (string, string, string, error) {
	if raw == "" || len(raw) > 512 {
		return "", "", "", ports.ErrConsultaEquipoInvalida
	}
	q, err := url.ParseQuery(raw)
	if err != nil || len(q) < 2 || len(q) > 3 {
		return "", "", "", ports.ErrConsultaEquipoInvalida
	}
	for k, v := range q {
		if (k != "desde" && k != "hasta" && k != "cursor") || len(v) != 1 {
			return "", "", "", ports.ErrConsultaEquipoInvalida
		}
	}
	desde, hasta, cursor := q.Get("desde"), q.Get("hasta"), q.Get("cursor")
	if !fechaCivilCanonica(desde) || !fechaCivilCanonica(hasta) || desde > hasta || len(cursor) > 256 {
		return "", "", "", ports.ErrConsultaEquipoInvalida
	}
	return desde, hasta, cursor, nil
}
