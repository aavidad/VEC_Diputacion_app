package httpinterno

import (
	"encoding/json"
	"errors"
	"fmt"
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

// errPeticionResolucionInvalida es el motivo cerrado con el que se rechaza
// un cuerpo o una consulta mal formados. Cuando el rechazo procede de un
// fallo de lectura o decodificación, ese fallo queda envuelto como causa
// (errors.Is/errors.As) para el registro interno; al cliente solo llega
// «peticion_invalida».
var errPeticionResolucionInvalida = errors.New("cronos: petición de resolución inválida")

// peticionInvalidaPor envuelve la causa técnica bajo el motivo cerrado.
func peticionInvalidaPor(causa error) error {
	return fmt.Errorf("%w: %w", errPeticionResolucionInvalida, causa)
}

// decodificarCadenasAcotadas lee un único objeto JSON de hasta
// maximoCuerpoResolver bytes cuyos campos son cadenas conocidas, cada una
// con su límite en caracteres, sin duplicados ni campos extra. Todo rechazo
// satisface errors.Is(err, errPeticionResolucionInvalida).
func decodificarCadenasAcotadas(w http.ResponseWriter, r *http.Request, obligatorios []string, maximos map[string]int) (map[string]string, error) {
	if r.Header.Get("Content-Type") != "application/json" || r.Body == nil {
		return nil, errPeticionResolucionInvalida
	}
	defer r.Body.Close()
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maximoCuerpoResolver))
	apertura, err := dec.Token()
	if err != nil {
		return nil, peticionInvalidaPor(err)
	}
	if apertura != json.Delim('{') {
		return nil, errPeticionResolucionInvalida
	}
	valores := make(map[string]string, len(maximos))
	for dec.More() {
		token, err := dec.Token()
		if err != nil {
			return nil, peticionInvalidaPor(err)
		}
		nombre, ok := token.(string)
		maximo, admitido := maximos[nombre]
		if !ok || !admitido {
			return nil, errPeticionResolucionInvalida
		}
		if _, repetido := valores[nombre]; repetido {
			return nil, errPeticionResolucionInvalida
		}
		var v string
		if err := dec.Decode(&v); err != nil {
			return nil, peticionInvalidaPor(err)
		}
		if !utf8.ValidString(v) || utf8.RuneCountInString(v) > maximo {
			return nil, errPeticionResolucionInvalida
		}
		valores[nombre] = v
	}
	cierre, err := dec.Token()
	if err != nil {
		return nil, peticionInvalidaPor(err)
	}
	if cierre != json.Delim('}') {
		return nil, errPeticionResolucionInvalida
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, errPeticionResolucionInvalida
		}
		return nil, peticionInvalidaPor(err)
	}
	for _, c := range obligatorios {
		if _, ok := valores[c]; !ok {
			return nil, errPeticionResolucionInvalida
		}
	}
	return valores, nil
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
	paso, err := parametroPaso(r.URL.RawQuery)
	if err != nil {
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
	v, err := decodificarCadenasAcotadas(w, r, []string{"clave_operacion", "solicitud_ref", "paso", "decision", "version_esperada"},
		map[string]int{"clave_operacion": 128, "solicitud_ref": 160, "paso": 32, "decision": 16, "version_esperada": 3, "motivo": domain.MaximoMotivoResolucion})
	if err != nil {
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

// parametroPaso exige paso=responsable o paso=administracion. Un rechazo
// satisface errors.Is(err, errPeticionResolucionInvalida) y conserva como
// causa el fallo de análisis de la consulta, si lo hubo.
func parametroPaso(raw string) (domain.PasoPermiso, error) {
	if raw == "" || len(raw) > 32 {
		return "", errPeticionResolucionInvalida
	}
	valores, err := url.ParseQuery(raw)
	if err != nil {
		return "", peticionInvalidaPor(err)
	}
	if len(valores) != 1 || len(valores["paso"]) != 1 {
		return "", errPeticionResolucionInvalida
	}
	paso := domain.PasoPermiso(valores.Get("paso"))
	if !domain.PasoResolucionValido(paso) {
		return "", errPeticionResolucionInvalida
	}
	return paso, nil
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
