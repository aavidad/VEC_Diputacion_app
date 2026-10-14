package httpinterno

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/application"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
)

const RutaCalendarioLaboralPropioC4 = "/api/interna/cronos/calendario/propio"

// AdaptadorCalendarioLaboralC4 evita que HTTP defina identidades o fechas por omisión.
type AdaptadorCalendarioLaboralC4 struct {
	resolver  ResolverConsultaSaldoPropio
	consultar func(*http.Request, application.ConsultaCalendarioLaboralC4) (application.ResultadoCalendarioLaboralC4, error)
}

func NuevoManejadorCalendarioLaboralC4(servicio *application.ServicioCalendarioLaboralC4, resolver ResolverConsultaSaldoPropio) (*AdaptadorCalendarioLaboralC4, error) {
	if nuloCalendarioC4(servicio) || nuloCalendarioC4(resolver) {
		return nil, application.ErrCalendarioLaboralC4NoDisponible
	}
	return &AdaptadorCalendarioLaboralC4{resolver: resolver, consultar: func(r *http.Request, q application.ConsultaCalendarioLaboralC4) (application.ResultadoCalendarioLaboralC4, error) {
		return servicio.ConsultarFechaPropia(r.Context(), q)
	}}, nil
}

func nuloCalendarioC4(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	return r.Kind() == reflect.Pointer && r.IsNil()
}

func (m *AdaptadorCalendarioLaboralC4) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if m == nil || r == nil || r.URL == nil || m.consultar == nil || nuloCalendarioC4(m.resolver) {
		errorJSON(w, http.StatusServiceUnavailable, "no_disponible")
		return
	}
	if r.URL.Path != RutaCalendarioLaboralPropioC4 || r.URL.RawPath != "" {
		errorJSON(w, http.StatusNotFound, "no_disponible")
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		errorJSON(w, http.StatusMethodNotAllowed, "metodo_no_permitido")
		return
	}
	fecha, ok := fechaCalendarioC4(r.URL.RawQuery)
	if !ok {
		errorJSON(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	orden, err := m.resolver.ResolverConsultaSaldoPropio(r, ports.PeriodoSaldoRango, fecha, fecha)
	if err != nil {
		responderErrorAccesoCronos(w, err)
		return
	}
	actor, err := orden.ContextoActor()
	if err != nil {
		responderErrorAccesoCronos(w, ports.ErrDependenciaNoDisponible)
		return
	}
	resultado, err := m.consultar(r, application.ConsultaCalendarioLaboralC4{Actor: actor, Fecha: fecha, OrdenSaldo: orden})
	if err != nil {
		if errors.Is(err, application.ErrCalendarioLaboralC4Invalido) {
			errorJSON(w, http.StatusBadRequest, "peticion_invalida")
		} else if errors.Is(err, personaldomain.ErrUnidadTrabajoCronosDenegada) {
			errorJSON(w, http.StatusForbidden, "acceso_denegado")
		} else {
			responderErrorAccesoCronos(w, err)
		}
		return
	}
	_ = json.NewEncoder(w).Encode(resultado)
}

func fechaCalendarioC4(raw string) (string, bool) {
	if len(raw) == 0 || len(raw) > 32 {
		return "", false
	}
	v, err := url.ParseQuery(raw)
	if err != nil || len(v) != 1 || len(v["fecha"]) != 1 {
		return "", false
	}
	fecha := v.Get("fecha")
	if !fechaCivilCanonica(fecha) {
		return "", false
	}
	// Las fechas civilmente válidas se limitan al rango de lectura de Calendarios.
	anio, err := time.Parse("2006-01-02", fecha)
	return fecha, err == nil && anio.Year() >= 1900 && anio.Year() <= 2100
}
