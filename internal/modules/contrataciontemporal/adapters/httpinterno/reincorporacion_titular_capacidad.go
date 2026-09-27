package httpinterno

import (
	"context"
	"net/http"
	"strconv"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// RutaCapacidadReincorporacionTitular consulta si la identidad del canal
// tiene permiso preliminar para iniciar el registro en el expediente actual.
// El POST vuelve a autorizar la intención completa antes del efecto.
const RutaCapacidadReincorporacionTitular = RutaReincorporacionesTitular + "/capacidad"

// ComprobadorCapacidadReincorporacionTitular exige lectura V3 del expediente y
// evalúa en el PDP central la acción/recurso/finalidad exactos. Una respuesta
// positiva no sustituye la autorización del POST ni acredita cese coincidente.
type ComprobadorCapacidadReincorporacionTitular interface {
	ComprobarCapacidadReincorporacionTitular(context.Context, application.ContextoCanalSeguimiento, string, uint64) (bool, error)
}

type manejadorCapacidadReincorporacionTitular struct {
	autoridad   AutoridadCanalSeguimiento
	comprobador ComprobadorCapacidadReincorporacionTitular
}

func NuevoManejadorCapacidadReincorporacionTitular(a AutoridadCanalSeguimiento, c ComprobadorCapacidadReincorporacionTitular) (http.Handler, error) {
	if dependenciaNula(a) || dependenciaNula(c) {
		return nil, ports.ErrOperacionSeguimientoNoDisponible
	}
	return &manejadorCapacidadReincorporacionTitular{autoridad: a, comprobador: c}, nil
}

func (h *manejadorCapacidadReincorporacionTitular) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r == nil || r.URL == nil || r.URL.Path != RutaCapacidadReincorporacionTitular || r.URL.ForceQuery {
		responderErrorSeguimiento(w, r, http.StatusNotFound, "recurso_no_encontrado")
		return
	}
	if r.Method != http.MethodGet {
		responderErrorSeguimiento(w, r, http.StatusMethodNotAllowed, "metodo_no_permitido")
		return
	}
	if cabeceraCoberturaProhibida(r.Header) {
		responderErrorSeguimiento(w, r, http.StatusBadRequest, "peticion_no_permitida")
		return
	}
	q := r.URL.Query()
	if len(q) != 2 || len(q["expediente_ref"]) != 1 || len(q["version_esperada"]) != 1 ||
		!domain.ReferenciaOpacaValida(q.Get("expediente_ref")) {
		responderErrorSeguimiento(w, r, http.StatusUnprocessableEntity, "contenido_no_valido")
		return
	}
	version, err := strconv.ParseUint(q.Get("version_esperada"), 10, 64)
	if err != nil || !ports.VersionOperacionAnalisisConIncrementoValida(version) {
		responderErrorSeguimiento(w, r, http.StatusUnprocessableEntity, "contenido_no_valido")
		return
	}
	canal, err := h.autoridad.ResolverContextoCanalSeguimiento(r.Context())
	if err != nil || !canal.Valido() {
		responderErrorSeguimiento(w, r, http.StatusForbidden, "acceso_denegado")
		return
	}
	permitido, err := h.comprobador.ComprobarCapacidadReincorporacionTitular(r.Context(), canal, q.Get("expediente_ref"), version)
	if err != nil {
		responderErrorSeguimiento(w, r, http.StatusServiceUnavailable, "servicio_no_disponible", err)
		return
	}
	responderJSONCobertura(w, r, http.StatusOK, map[string]any{"data": map[string]any{
		"esquema": "vec.contratacion-temporal.capacidad-reincorporacion-titular.v1",
		"puede_registrar_reincorporacion_titular": permitido,
	}})
}
