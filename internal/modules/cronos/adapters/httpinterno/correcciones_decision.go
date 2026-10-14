package httpinterno

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

const (
	RutaBandejaCorrecciones   = "/api/interna/cronos/correcciones/bandeja"
	RutaActuacionCorrecciones = "/api/interna/cronos/correcciones/actuaciones"
	RutaHistoriaCorrecciones  = "/api/interna/cronos/correcciones/historia"
)

type ResolverDecisionCorrecciones interface {
	ResolverDecisionCorrecciones(*http.Request) (ports.OrdenConsumoCorreccion, error)
}

type ManejadorDecisionCorrecciones struct {
	resolver ResolverDecisionCorrecciones
	casoUso  ports.CasoUsoDecisionCorrecciones
}

func NuevoManejadorDecisionCorrecciones(casoUso ports.CasoUsoDecisionCorrecciones, resolver ResolverDecisionCorrecciones) (*ManejadorDecisionCorrecciones, error) {
	if dependenciaCronosHTTPNula(casoUso) || dependenciaCronosHTTPNula(resolver) {
		return nil, ports.ErrDependenciaNoDisponible
	}
	return &ManejadorDecisionCorrecciones{casoUso: casoUso, resolver: resolver}, nil
}

func (m *ManejadorDecisionCorrecciones) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cabecerasJSON(w)
	if r.URL == nil || r.URL.RawPath != "" {
		errorJSON(w, http.StatusNotFound, "no_disponible")
		return
	}
	switch r.URL.Path {
	case RutaBandejaCorrecciones:
		m.bandeja(w, r)
	case RutaActuacionCorrecciones:
		m.actuar(w, r)
	case RutaHistoriaCorrecciones:
		m.historia(w, r)
	default:
		errorJSON(w, http.StatusNotFound, "no_disponible")
	}
}

func (m *ManejadorDecisionCorrecciones) historia(w http.ResponseWriter, r *http.Request) {
	if !soloMetodo(w, r, http.MethodGet) {
		return
	}
	if len(r.URL.RawQuery) > 512 {
		errorJSON(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	q, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil || len(q["solicitud_ref"]) != 1 || len(q["vista"]) != 1 ||
		!((q.Get("vista") == "propia" && len(q) == 2) || (q.Get("vista") == "equipo" && len(q) == 3 && len(q["paso"]) == 1 && domain.PasoBandejaCorreccionValido(q.Get("paso")))) {
		errorJSON(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	orden, err := m.resolver.ResolverDecisionCorrecciones(r)
	if err != nil {
		responderErrorAccesoCronos(w, err)
		return
	}
	h, err := m.casoUso.ConsultarHistoriaCorreccion(r.Context(), orden, ports.PeticionHistoriaCorreccion{SolicitudRef: q.Get("solicitud_ref"), Vista: q.Get("vista"), Paso: q.Get("paso")})
	if err != nil {
		responderErrorDecisionCorreccion(w, err)
		return
	}
	escribirJSONDecisionCorreccion(w, http.StatusOK, h)
}

func (m *ManejadorDecisionCorrecciones) bandeja(w http.ResponseWriter, r *http.Request) {
	if !soloMetodo(w, r, http.MethodGet) {
		return
	}
	if len(r.URL.RawQuery) > 64 {
		errorJSON(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	q, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil || len(q) != 1 || len(q["paso"]) != 1 || !domain.PasoBandejaCorreccionValido(q.Get("paso")) {
		errorJSON(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	orden, err := m.resolver.ResolverDecisionCorrecciones(r)
	if err != nil {
		responderErrorAccesoCronos(w, err)
		return
	}
	b, err := m.casoUso.ConsultarBandejaCorrecciones(r.Context(), orden, q.Get("paso"))
	if err != nil {
		responderErrorDecisionCorreccion(w, err)
		return
	}
	escribirJSONDecisionCorreccion(w, http.StatusOK, b)
}

func (m *ManejadorDecisionCorrecciones) actuar(w http.ResponseWriter, r *http.Request) {
	if !soloMetodo(w, r, http.MethodPost) {
		return
	}
	if r.URL.RawQuery != "" {
		errorJSON(w, http.StatusNotFound, "no_disponible")
		return
	}
	v, err := decodificarCadenasAcotadas(w, r,
		[]string{"solicitud_ref", "clave_operacion", "paso", "resultado", "motivo", "version_esperada"},
		map[string]int{"solicitud_ref": 160, "clave_operacion": 128, "paso": 32, "resultado": 16, "motivo": domain.MaximoMotivoResolucion, "version_esperada": 3})
	if err != nil {
		errorJSON(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	version, err := strconv.ParseUint(v["version_esperada"], 10, 64)
	if err != nil || strconv.FormatUint(version, 10) != v["version_esperada"] {
		errorJSON(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	paso := domain.PasoCorreccion(v["paso"])
	switch v["paso"] {
	case "responsable":
		paso = domain.PasoDecisionResponsable
	case "rrhh":
		paso = domain.PasoResolucionRRHH
	case "aplicar":
		paso = domain.PasoAplicacion
	}
	orden, err := m.resolver.ResolverDecisionCorrecciones(r)
	if err != nil {
		responderErrorAccesoCronos(w, err)
		return
	}
	recibo, err := m.casoUso.RegistrarDecisionCorreccion(r.Context(), orden, ports.PeticionActuacionCorreccion{
		SolicitudRef: v["solicitud_ref"], ClaveOperacion: v["clave_operacion"], Paso: paso,
		Resultado: domain.ResultadoCorreccion(v["resultado"]), Motivo: v["motivo"], VersionEsperada: version,
	})
	if err != nil {
		responderErrorDecisionCorreccion(w, err)
		return
	}
	estado := http.StatusOK
	if !recibo.Replay {
		estado = http.StatusCreated
	}
	escribirJSONDecisionCorreccion(w, estado, map[string]any{"recibo": map[string]any{
		"solicitud_ref": recibo.SolicitudRef, "actuacion_ref": recibo.ActuacionRef,
		"recibo_ref": recibo.ReciboRef, "estado": recibo.Estado, "version": recibo.Version,
		"instante_utc": recibo.InstanteUTC, "replay": recibo.Replay,
	}})
}

func escribirJSONDecisionCorreccion(w http.ResponseWriter, estado int, valor any) {
	cuerpo, err := json.Marshal(valor)
	if err != nil {
		responderErrorAccesoCronos(w, ports.ErrDependenciaNoDisponible)
		return
	}
	if estado != http.StatusOK {
		w.WriteHeader(estado)
	}
	if _, err = w.Write(append(cuerpo, '\n')); err != nil {
		slog.Error("cronos: fallo al escribir respuesta de corrección", "error", err)
	}
}

func responderErrorDecisionCorreccion(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrCorreccionInvalida), errors.Is(err, ports.ErrSolicitudCronosInvalida):
		errorJSON(w, http.StatusBadRequest, "peticion_invalida")
	case errors.Is(err, ports.ErrCorreccionEnConflicto):
		errorJSON(w, http.StatusConflict, "conflicto")
	case errors.Is(err, ports.ErrResolucionEstadoCambiado):
		errorJSON(w, http.StatusConflict, "estado_cambiado")
	case errors.Is(err, ports.ErrResolucionNoCompetente):
		errorJSON(w, http.StatusForbidden, "no_competente")
	default:
		responderErrorAccesoCronos(w, err)
	}
}
