package httpinterno

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/ports"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

const RutaConsultarSaldoPropio = "/api/interna/cronos/saldos/propio"

var (
	ErrAutenticacionCronosRequerida = errors.New("cronos: autenticacion requerida")
	ErrAccesoCronosDenegado         = errors.New("cronos: acceso denegado")
)

// El resolver entrega una orden creada con el ContextoActor registrado y
// concesión V3 positiva para esta lectura y periodo. No interpreta parámetros
// de identidad ni cabeceras libres del cliente.
type ResolverConsultaSaldoPropio interface {
	ResolverConsultaSaldoPropio(*http.Request, ports.PeriodoSaldo, string, string) (ports.OrdenConsultaSaldo, error)
}

type ManejadorSaldoPropio struct {
	resolver ResolverConsultaSaldoPropio
	casoUso  ports.CasoUsoConsultarSaldo
}

func NuevoManejadorSaldoPropio(casoUso ports.CasoUsoConsultarSaldo, resolver ResolverConsultaSaldoPropio) (*ManejadorSaldoPropio, error) {
	if dependenciaCronosHTTPNula(casoUso) || dependenciaCronosHTTPNula(resolver) {
		return nil, ports.ErrDependenciaNoDisponible
	}
	return &ManejadorSaldoPropio{resolver: resolver, casoUso: casoUso}, nil
}

func dependenciaCronosHTTPNula(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return r.IsNil()
	default:
		return false
	}
}

func (m *ManejadorSaldoPropio) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r.URL == nil || r.URL.Path != RutaConsultarSaldoPropio || r.URL.RawPath != "" {
		errorJSON(w, http.StatusNotFound, "no_disponible")
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		errorJSON(w, http.StatusMethodNotAllowed, "metodo_no_permitido")
		return
	}
	periodo, desde, hasta, err := parametrosSaldo(r.URL.RawQuery)
	if err != nil {
		errorJSON(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	orden, err := m.resolver.ResolverConsultaSaldoPropio(r, periodo, desde, hasta)
	if err != nil {
		responderErrorAccesoCronos(w, err)
		return
	}
	if _, err := orden.ContextoActor(); err != nil {
		responderErrorAccesoCronos(w, ports.ErrDependenciaNoDisponible)
		return
	}
	resultado, err := m.casoUso.ConsultarSaldo(r.Context(), orden, periodo, desde, hasta)
	if err != nil {
		if errors.Is(err, ports.ErrConsultaSaldoInvalida) {
			errorJSON(w, http.StatusBadRequest, "peticion_invalida")
			return
		}
		responderErrorAccesoCronos(w, err)
		return
	}
	if resultado.Detalle == nil {
		resultado.Detalle = []ports.DetalleSaldoDia{}
	}
	_ = json.NewEncoder(w).Encode(resultado)
}

// Sólo errores nominales acreditan ausencia de identidad o denegación. Un
// fallo ajeno, incluso del resolver, es dependencia no disponible.
func responderErrorAccesoCronos(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ports.ErrDependenciaNoDisponible):
		errorJSON(w, http.StatusServiceUnavailable, "no_disponible")
	case errors.Is(err, ErrAutenticacionCronosRequerida),
		errors.Is(err, httpseguridad.ErrAsercionAusente),
		errors.Is(err, httpseguridad.ErrAsercionNoValida),
		errors.Is(err, httpseguridad.ErrSesionNoValida),
		errors.Is(err, httpseguridad.ErrCanalProxyNoAutenticado):
		errorJSON(w, http.StatusUnauthorized, "autenticacion_requerida")
	case errors.Is(err, ErrAccesoCronosDenegado), errors.Is(err, vecdomain.ErrPermissionDenied),
		errors.Is(err, ports.ErrTeletrabajoNoAutorizado):
		errorJSON(w, http.StatusForbidden, "acceso_denegado")
	default:
		errorJSON(w, http.StatusServiceUnavailable, "no_disponible")
	}
}

func parametrosSaldo(raw string) (ports.PeriodoSaldo, string, string, error) {
	if len(raw) == 0 || len(raw) > 128 {
		return "", "", "", ports.ErrConsultaSaldoInvalida
	}
	valores, err := url.ParseQuery(raw)
	if err != nil || len(valores) < 1 || len(valores) > 3 {
		return "", "", "", ports.ErrConsultaSaldoInvalida
	}
	for clave, valor := range valores {
		if (clave != "periodo" && clave != "desde" && clave != "hasta") || len(valor) != 1 {
			return "", "", "", ports.ErrConsultaSaldoInvalida
		}
	}
	if len(valores["periodo"]) != 1 {
		return "", "", "", ports.ErrConsultaSaldoInvalida
	}
	periodo := ports.PeriodoSaldo(valores.Get("periodo"))
	switch periodo {
	case ports.PeriodoSaldoHoy, ports.PeriodoSaldoSemana, ports.PeriodoSaldoMes, ports.PeriodoSaldoAnio:
		if len(valores) != 1 {
			return "", "", "", ports.ErrConsultaSaldoInvalida
		}
		return periodo, "", "", nil
	case ports.PeriodoSaldoRango:
		if len(valores) != 3 || !fechaCivilCanonica(valores.Get("desde")) || !fechaCivilCanonica(valores.Get("hasta")) || valores.Get("desde") > valores.Get("hasta") {
			return "", "", "", ports.ErrConsultaSaldoInvalida
		}
		return periodo, valores.Get("desde"), valores.Get("hasta"), nil
	default:
		return "", "", "", ports.ErrConsultaSaldoInvalida
	}
}

func fechaCivilCanonica(s string) bool {
	if len(s) != len("2006-01-02") {
		return false
	}
	fecha, err := time.Parse("2006-01-02", s)
	return err == nil && fecha.Format("2006-01-02") == s
}
