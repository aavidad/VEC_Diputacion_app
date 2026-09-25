package httpinterno

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"unicode/utf8"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

// Rutas de la resolución de permisos (jefatura y RRHH) y de los avisos de
// resolución de la persona. Identidad, empleado propio y autorización V3 los
// aporta el resolver desde la frontera; el cliente sólo envía el paso, la
// solicitud, la decisión, el motivo, la versión vista y su clave.
const (
	RutaBandejaPermisos  = "/api/interna/cronos/permisos/bandeja"
	RutaResolverPermiso  = "/api/interna/cronos/permisos/resoluciones"
	RutaAvisosPropios    = "/api/interna/cronos/avisos/propio"
	RutaArchivarAviso    = "/api/interna/cronos/avisos/archivos"
	maximoCuerpoResolver = 8192
)

type ResolverResolucionPermisos interface {
	ResolverResolucionPermisos(*http.Request) (ports.OrdenResolucionPermisos, error)
}

type ResolverAvisosPropios interface {
	ResolverAvisosPropios(*http.Request) (ports.OrdenAvisosPropios, error)
}

// decodificarCadenasAcotadas lee un único objeto JSON de hasta
// maximoCuerpoResolver bytes cuyos campos son cadenas conocidas, cada una
// con su límite en caracteres, sin duplicados ni campos extra.
func decodificarCadenasAcotadas(w http.ResponseWriter, r *http.Request, obligatorios []string, maximos map[string]int) (map[string]string, bool) {
	if r.Header.Get("Content-Type") != "application/json" || r.Body == nil {
		return nil, false
	}
	defer r.Body.Close()
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maximoCuerpoResolver))
	if apertura, err := dec.Token(); err != nil || apertura != json.Delim('{') {
		return nil, false
	}
	valores := make(map[string]string, len(maximos))
	for dec.More() {
		token, err := dec.Token()
		nombre, ok := token.(string)
		maximo, admitido := maximos[nombre]
		if err != nil || !ok || !admitido {
			return nil, false
		}
		if _, repetido := valores[nombre]; repetido {
			return nil, false
		}
		var v string
		if dec.Decode(&v) != nil || !utf8.ValidString(v) || utf8.RuneCountInString(v) > maximo {
			return nil, false
		}
		valores[nombre] = v
	}
	if cierre, err := dec.Token(); err != nil || cierre != json.Delim('}') {
		return nil, false
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, false
	}
	for _, c := range obligatorios {
		if _, ok := valores[c]; !ok {
			return nil, false
		}
	}
	return valores, true
}

// responderErrorResolucion añade los rechazos nominales de la resolución a
// la política común de acceso.
func responderErrorResolucion(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ports.ErrSolicitudCronosInvalida), errors.Is(err, domain.ErrSolicitudPermisoInvalida), errors.Is(err, domain.ErrMensajeInvalido):
		errorJSON(w, http.StatusBadRequest, "peticion_invalida")
	case errors.Is(err, ports.ErrClaveOperacionEnConflicto):
		errorJSON(w, http.StatusConflict, "conflicto")
	case errors.Is(err, ports.ErrResolucionEstadoCambiado):
		errorJSON(w, http.StatusConflict, "estado_cambiado")
	case errors.Is(err, ports.ErrResolucionPendienteAsignacion):
		errorJSON(w, http.StatusConflict, "pendiente_asignacion")
	case errors.Is(err, ports.ErrResolucionNoCompetente):
		errorJSON(w, http.StatusForbidden, "no_competente")
	default:
		responderErrorAccesoCronos(w, err)
	}
}

// ---- Quien resuelve ----

type ManejadorResolucionPermisos struct {
	resolver ResolverResolucionPermisos
	casoUso  ports.CasoUsoResolucionPermisos
}

func NuevoManejadorResolucionPermisos(casoUso ports.CasoUsoResolucionPermisos, resolver ResolverResolucionPermisos) (*ManejadorResolucionPermisos, error) {
	if dependenciaCronosHTTPNula(casoUso) || dependenciaCronosHTTPNula(resolver) {
		return nil, ports.ErrDependenciaNoDisponible
	}
	return &ManejadorResolucionPermisos{resolver: resolver, casoUso: casoUso}, nil
}

func (m *ManejadorResolucionPermisos) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cabecerasJSON(w)
	if r.URL == nil || r.URL.RawPath != "" || (r.URL.Path != RutaBandejaPermisos && r.URL.Path != RutaResolverPermiso) {
		errorJSON(w, http.StatusNotFound, "no_disponible")
		return
	}
	if r.URL.Path == RutaResolverPermiso {
		m.registrarResolucion(w, r)
		return
	}
	if !soloMetodo(w, r, http.MethodGet) {
		return
	}
	paso, ok := parametroPaso(r.URL.RawQuery)
	if !ok {
		errorJSON(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	orden, err := m.resolver.ResolverResolucionPermisos(r)
	if err != nil {
		responderErrorAccesoCronos(w, err)
		return
	}
	bandeja, err := m.casoUso.ConsultarBandeja(r.Context(), orden, paso)
	if err != nil {
		responderErrorResolucion(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(bandeja)
}

func (m *ManejadorResolucionPermisos) registrarResolucion(w http.ResponseWriter, r *http.Request) {
	if !soloMetodo(w, r, http.MethodPost) {
		return
	}
	if r.URL.RawQuery != "" {
		errorJSON(w, http.StatusNotFound, "no_disponible")
		return
	}
	v, ok := decodificarCadenasAcotadas(w, r, []string{"clave_operacion", "solicitud_ref", "paso", "decision", "version_esperada"},
		map[string]int{"clave_operacion": 128, "solicitud_ref": 160, "paso": 32, "decision": 16, "version_esperada": 3, "motivo": domain.MaximoMotivoResolucion})
	if !ok {
		errorJSON(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	version, err := strconv.Atoi(v["version_esperada"])
	if err != nil || strconv.Itoa(version) != v["version_esperada"] || version < 1 {
		errorJSON(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	orden, err := m.resolver.ResolverResolucionPermisos(r)
	if err != nil {
		responderErrorAccesoCronos(w, err)
		return
	}
	recibo, err := m.casoUso.ResolverPermiso(r.Context(), orden, ports.PeticionResolucionPermiso{
		ClaveOperacion: v["clave_operacion"], SolicitudRef: v["solicitud_ref"], Paso: domain.PasoPermiso(v["paso"]),
		Decision: domain.DecisionPermiso(v["decision"]), Motivo: v["motivo"], VersionEsperada: version,
	})
	if err != nil {
		responderErrorResolucion(w, err)
		return
	}
	if !recibo.Replay {
		w.WriteHeader(http.StatusCreated)
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"recibo": recibo})
}

// parametroPaso exige paso=responsable o paso=administracion.
func parametroPaso(raw string) (domain.PasoPermiso, bool) {
	if raw == "" || len(raw) > 32 {
		return "", false
	}
	valores, err := url.ParseQuery(raw)
	if err != nil || len(valores) != 1 || len(valores["paso"]) != 1 {
		return "", false
	}
	paso := domain.PasoPermiso(valores.Get("paso"))
	return paso, domain.PasoResolucionValido(paso)
}

// ---- La persona: avisos ----

type ManejadorAvisosPropios struct {
	resolver ResolverAvisosPropios
	casoUso  ports.CasoUsoAvisosPropios
}

func NuevoManejadorAvisosPropios(casoUso ports.CasoUsoAvisosPropios, resolver ResolverAvisosPropios) (*ManejadorAvisosPropios, error) {
	if dependenciaCronosHTTPNula(casoUso) || dependenciaCronosHTTPNula(resolver) {
		return nil, ports.ErrDependenciaNoDisponible
	}
	return &ManejadorAvisosPropios{resolver: resolver, casoUso: casoUso}, nil
}

func (m *ManejadorAvisosPropios) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cabecerasJSON(w)
	if r.URL == nil || r.URL.RawPath != "" || r.URL.RawQuery != "" || (r.URL.Path != RutaAvisosPropios && r.URL.Path != RutaArchivarAviso) {
		errorJSON(w, http.StatusNotFound, "no_disponible")
		return
	}
	if r.URL.Path == RutaArchivarAviso {
		m.archivar(w, r)
		return
	}
	if !soloMetodo(w, r, http.MethodGet) {
		return
	}
	orden, err := m.resolver.ResolverAvisosPropios(r)
	if err != nil {
		responderErrorAccesoCronos(w, err)
		return
	}
	avisos, err := m.casoUso.ConsultarAvisos(r.Context(), orden)
	if err != nil {
		responderErrorResolucion(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(avisos)
}

func (m *ManejadorAvisosPropios) archivar(w http.ResponseWriter, r *http.Request) {
	if !soloMetodo(w, r, http.MethodPost) {
		return
	}
	v, ok := decodificarCadenas(w, r, []string{"clave_operacion", "aviso_ref"}, nil)
	if !ok {
		errorJSON(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	orden, err := m.resolver.ResolverAvisosPropios(r)
	if err != nil {
		responderErrorAccesoCronos(w, err)
		return
	}
	recibo, err := m.casoUso.ArchivarAviso(r.Context(), orden, ports.PeticionArchivoAviso{ClaveOperacion: v["clave_operacion"], AvisoRef: v["aviso_ref"]})
	if err != nil {
		responderErrorResolucion(w, err)
		return
	}
	if !recibo.Replay {
		w.WriteHeader(http.StatusCreated)
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"recibo": recibo})
}
