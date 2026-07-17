// Package httpinterno expone exclusivamente la frontera HTTP del panel
// operativo interno de Bolsa. La identidad, el perfil activo, el alcance y el
// motivo de acceso se resuelven fuera de este adaptador por una frontera de
// confianza; nunca se reconstruyen desde parámetros enviados por el cliente.
package httpinterno

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	aplicacionbolsa "vec-diputacion-granada/internal/modules/bolsa/application"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const RutaPanel = "/api/vec/bolsa/panel"

var (
	ErrHandlerPanelInternoInvalido = errors.New("bolsa http interno: handler de panel invalido")
	// ErrAutenticacionInternaAusente permite a la frontera de confianza indicar
	// que no existe una autenticación apta. No convierte una cabecera HTTP en
	// identidad ni afirma qué mecanismo debe utilizar la composición final.
	ErrAutenticacionInternaAusente = errors.New("bolsa http interno: autenticacion interna ausente")
	// ErrDependenciaPanelInternoNoDisponible es el error estable con el que una
	// composición puede ocultar fallos de identidad, PDP o persistencia sin
	// filtrar detalles de infraestructura a la respuesta HTTP.
	ErrDependenciaPanelInternoNoDisponible = errors.New("bolsa http interno: dependencia no disponible")
)

// PreparadorOrdenConsultaPanelInterno constituye la frontera de confianza.
// Su implementación productiva debe resolver del lado servidor el actor, la
// sesión revalidada, el perfil, el selector, el motivo catalogado y la
// correlación. El handler no intenta obtenerlos de cabeceras, query ni cuerpo.
type PreparadorOrdenConsultaPanelInterno interface {
	PrepararOrdenConsultaPanelInterno(*http.Request) (aplicacionbolsa.OrdenConsultaPanelInterno, error)
}

// ConsultorPanelInterno es la superficie mínima que satisface el caso de uso
// de aplicación. Mantiene este adaptador independiente de su composición.
type ConsultorPanelInterno interface {
	Consultar(
		context.Context,
		aplicacionbolsa.OrdenConsultaPanelInterno,
	) (puertosbolsa.InstantaneaPanelInterno, error)
}

type Handler struct {
	preparador PreparadorOrdenConsultaPanelInterno
	consultor  ConsultorPanelInterno
}

var (
	_ http.Handler          = (*Handler)(nil)
	_ ConsultorPanelInterno = (*aplicacionbolsa.ServicioConsultaPanelInterno)(nil)
)

func NuevoHandler(
	preparador PreparadorOrdenConsultaPanelInterno,
	consultor ConsultorPanelInterno,
) (http.Handler, error) {
	if dependenciaNula(preparador) || dependenciaNula(consultor) {
		return nil, ErrHandlerPanelInternoInvalido
	}
	return &Handler{preparador: preparador, consultor: consultor}, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r == nil || h == nil || dependenciaNula(h.preparador) || dependenciaNula(h.consultor) {
		responderError(w, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	if r.Method == http.MethodHead {
		w = escritorSinCuerpo{ResponseWriter: w}
	}
	if !rutaExacta(r) {
		responderError(w, http.StatusNotFound, "recurso_no_encontrado")
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		responderError(w, http.StatusMethodNotAllowed, "metodo_no_permitido")
		return
	}
	if !entradaVaciaYPermitida(r) {
		responderError(w, http.StatusBadRequest, "peticion_no_permitida")
		return
	}

	orden, err := h.preparador.PrepararOrdenConsultaPanelInterno(r)
	if err != nil {
		responderErrorClasificado(w, err)
		return
	}
	resultado, err := h.consultor.Consultar(r.Context(), orden)
	if err != nil {
		responderErrorClasificado(w, err)
		return
	}
	responderJSON(w, http.StatusOK, nuevaRespuestaPanel(resultado))
}

func rutaExacta(r *http.Request) bool {
	if r.URL == nil || r.URL.Path != RutaPanel || r.URL.RawPath != "" {
		return false
	}
	return r.URL.EscapedPath() == RutaPanel && !strings.Contains(r.URL.EscapedPath(), "%")
}

func entradaVaciaYPermitida(r *http.Request) bool {
	if r.URL.RawQuery != "" || r.URL.ForceQuery || r.ContentLength != 0 ||
		len(r.TransferEncoding) != 0 || (r.Body != nil && r.Body != http.NoBody) {
		return false
	}
	// Incluso una cabecera vacía se rechaza: esta superficie no usa sesión de
	// navegador ni admite credenciales destinadas a un proxy intermedio. Las
	// antiguas cabeceras de identidad tampoco alcanzan la frontera preparadora.
	return !cabeceraPresente(r.Header, "Cookie") &&
		!cabeceraPresente(r.Header, "Proxy-Authorization") &&
		!cabeceraIdentidadHeredadaPresente(r.Header)
}

func cabeceraPresente(cabeceras http.Header, buscada string) bool {
	for nombre := range cabeceras {
		if strings.EqualFold(nombre, buscada) {
			return true
		}
	}
	return false
}

func cabeceraIdentidadHeredadaPresente(cabeceras http.Header) bool {
	for nombre := range cabeceras {
		minusculas := strings.ToLower(nombre)
		if strings.HasPrefix(minusculas, "x-vec-") ||
			strings.HasPrefix(minusculas, "x-auth-") ||
			minusculas == "x-remote-user" || minusculas == "remote-user" ||
			strings.HasPrefix(minusculas, "x-forwarded-") {
			return true
		}
	}
	return false
}

func responderErrorClasificado(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrAutenticacionInternaAusente):
		responderError(w, http.StatusUnauthorized, "autenticacion_requerida")
	case esDependenciaNoDisponible(err):
		responderError(w, http.StatusServiceUnavailable, "servicio_no_disponible")
	case errors.Is(err, dominiovec.ErrAutorizacionDenegada),
		errors.Is(err, dominiovec.ErrPermissionDenied):
		responderError(w, http.StatusForbidden, "acceso_denegado")
	default:
		responderError(w, http.StatusInternalServerError, "error_interno")
	}
}

func esDependenciaNoDisponible(err error) bool {
	return errors.Is(err, ErrDependenciaPanelInternoNoDisponible) ||
		errors.Is(err, aplicacionbolsa.ErrServicioPanelInternoInvalido) ||
		errors.Is(err, puertosvec.ErrFuenteContextoActorNoDisponible) ||
		errors.Is(err, puertosvec.ErrRevalidacionAutenticacionActorNoDisponible) ||
		errors.Is(err, puertosvec.ErrFuenteAutorizacionNoDisponible) ||
		errors.Is(err, puertosvec.ErrRegistroDecisionNoDisponible) ||
		errors.Is(err, puertosvec.ErrRegistroDenegacionNoDisponible)
}

type respuestaError struct {
	Error detalleError `json:"error"`
}

type detalleError struct {
	Codigo  string `json:"codigo"`
	Mensaje string `json:"mensaje"`
}

func responderError(w http.ResponseWriter, estado int, codigo string) {
	mensaje := "No se ha podido completar la consulta."
	switch estado {
	case http.StatusBadRequest:
		mensaje = "La petición no está permitida."
	case http.StatusUnauthorized:
		mensaje = "Se requiere autenticación interna."
	case http.StatusForbidden:
		mensaje = "Acceso denegado."
	case http.StatusNotFound:
		mensaje = "Recurso no encontrado."
	case http.StatusMethodNotAllowed:
		mensaje = "Método no permitido."
	case http.StatusServiceUnavailable:
		mensaje = "Servicio no disponible."
	}
	responderJSON(w, estado, respuestaError{Error: detalleError{Codigo: codigo, Mensaje: mensaje}})
}

func responderJSON(w http.ResponseWriter, estado int, valor any) {
	contenido, err := json.Marshal(valor)
	if err != nil {
		// Todos los DTO de este paquete son cerrados y serializables. Esta rama
		// mantiene el fallo cerrado si una modificación futura rompe el contrato.
		estado = http.StatusInternalServerError
		contenido = []byte(`{"error":{"codigo":"error_interno","mensaje":"No se ha podido completar la consulta."}}`)
	}
	aplicarCabeceras(w)
	w.Header().Set("Content-Length", strconv.Itoa(len(contenido)))
	w.WriteHeader(estado)
	_, _ = w.Write(contenido)
}

func aplicarCabeceras(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; base-uri 'none'; frame-ancestors 'none'")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=()")
	w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
	w.Header().Set("X-Frame-Options", "DENY")
}

type escritorSinCuerpo struct{ http.ResponseWriter }

func (escritorSinCuerpo) Write(contenido []byte) (int, error) { return len(contenido), nil }

// Los DTO HTTP son deliberadamente independientes de los DTO de puertos. Así,
// añadir un campo de dominio no lo publica por accidente y los time.Time cero
// realmente se omiten en vez de serializarse como 0001-01-01T00:00:00Z.
type respuestaPanel struct {
	Data datosPanel `json:"data"`
}

type datosPanel struct {
	Esquema               string              `json:"esquema"`
	Selector              selectorPanel       `json:"selector"`
	Origen                origenPanel         `json:"origen"`
	PruebaLectura         pruebaLecturaPanel  `json:"prueba_lectura"`
	Indicadores           indicadoresPanel    `json:"indicadores"`
	Convocatorias         []convocatoriaPanel `json:"convocatorias"`
	ActuacionesPendientes []actuacionPanel    `json:"actuaciones_pendientes"`
}

type selectorPanel struct {
	Clase            puertosbolsa.ClaseAmbitoPanelInterno `json:"clase"`
	OrganizacionRef  string                               `json:"organizacion_ref"`
	UnidadGestionRef string                               `json:"unidad_gestion_ref,omitempty"`
}

type origenPanel struct {
	Revision      string    `json:"revision"`
	ActualizadaEn time.Time `json:"actualizada_en"`
	Demostracion  bool      `json:"demostracion"`
}

type pruebaLecturaPanel struct {
	LecturaRef           string    `json:"lectura_ref"`
	AuditoriaRef         string    `json:"auditoria_ref"`
	AuditoriaSecuencia   uint64    `json:"auditoria_secuencia"`
	DecisionRef          string    `json:"decision_ref"`
	HuellaDecisionSHA256 string    `json:"huella_decision_sha256"`
	CorrelacionRef       string    `json:"correlacion_ref"`
	ConfirmadaEn         time.Time `json:"confirmada_en"`
}

type indicadoresPanel struct {
	ConvocatoriasBorrador        int `json:"convocatorias_borrador"`
	ConvocatoriasRevision        int `json:"convocatorias_revision"`
	ConvocatoriasPendientesFirma int `json:"convocatorias_pendientes_firma"`
	ConvocatoriasPublicadas      int `json:"convocatorias_publicadas"`
	BolsasActivas                int `json:"bolsas_activas"`
	BolsasSuspendidas            int `json:"bolsas_suspendidas"`
	BolsasAgotadas               int `json:"bolsas_agotadas"`
	LlamamientosPendientes       int `json:"llamamientos_pendientes"`
	LlamamientosEnCurso          int `json:"llamamientos_en_curso"`
	LlamamientosVencenHoy        int `json:"llamamientos_vencen_hoy"`
	DocumentosPendientesFirma    int `json:"documentos_pendientes_firma"`
	IncidenciasAbiertas          int `json:"incidencias_abiertas"`
}

type convocatoriaPanel struct {
	ConvocatoriaRef   string     `json:"convocatoria_ref"`
	CategoriaClave    string     `json:"categoria_clave"`
	EstadoClave       string     `json:"estado_clave"`
	PlazoCierraEn     *time.Time `json:"plazo_cierra_en,omitempty"`
	NumeroSolicitudes int        `json:"numero_solicitudes"`
	NumeroPendientes  int        `json:"numero_pendientes"`
}

type actuacionPanel struct {
	ActuacionRef    string     `json:"actuacion_ref"`
	RecursoRef      string     `json:"recurso_ref"`
	TipoClave       string     `json:"tipo_clave"`
	EstadoClave     string     `json:"estado_clave"`
	PrioridadClave  string     `json:"prioridad_clave"`
	FechaLimite     *time.Time `json:"fecha_limite,omitempty"`
	NumeroElementos int        `json:"numero_elementos"`
}

func nuevaRespuestaPanel(origen puertosbolsa.InstantaneaPanelInterno) respuestaPanel {
	convocatorias := make([]convocatoriaPanel, len(origen.Convocatorias))
	for indice, convocatoria := range origen.Convocatorias {
		convocatorias[indice] = convocatoriaPanel{
			ConvocatoriaRef: convocatoria.ConvocatoriaRef, CategoriaClave: convocatoria.CategoriaClave,
			EstadoClave: convocatoria.EstadoClave, PlazoCierraEn: instanteOpcional(convocatoria.PlazoCierraEn),
			NumeroSolicitudes: convocatoria.NumeroSolicitudes, NumeroPendientes: convocatoria.NumeroPendientes,
		}
	}
	actuaciones := make([]actuacionPanel, len(origen.ActuacionesPendientes))
	for indice, actuacion := range origen.ActuacionesPendientes {
		actuaciones[indice] = actuacionPanel{
			ActuacionRef: actuacion.ActuacionRef, RecursoRef: actuacion.RecursoRef,
			TipoClave: actuacion.TipoClave, EstadoClave: actuacion.EstadoClave,
			PrioridadClave: actuacion.PrioridadClave, FechaLimite: instanteOpcional(actuacion.FechaLimite),
			NumeroElementos: actuacion.NumeroElementos,
		}
	}
	return respuestaPanel{Data: datosPanel{
		Esquema: origen.Esquema,
		Selector: selectorPanel{
			Clase: origen.Selector.Clase, OrganizacionRef: origen.Selector.OrganizacionRef,
			UnidadGestionRef: origen.Selector.UnidadGestionRef,
		},
		Origen: origenPanel{
			Revision: origen.Origen.Revision, ActualizadaEn: origen.Origen.ActualizadaEn,
			Demostracion: origen.Origen.Demostracion,
		},
		PruebaLectura: pruebaLecturaPanel{
			LecturaRef: origen.PruebaLectura.LecturaRef, AuditoriaRef: origen.PruebaLectura.AuditoriaRef,
			AuditoriaSecuencia:   origen.PruebaLectura.AuditoriaSecuencia,
			DecisionRef:          origen.PruebaLectura.DecisionRef,
			HuellaDecisionSHA256: origen.PruebaLectura.HuellaDecisionSHA256,
			CorrelacionRef:       origen.PruebaLectura.CorrelacionRef,
			ConfirmadaEn:         origen.PruebaLectura.ConfirmadaEn,
		},
		Indicadores:   indicadoresPanel(origen.Indicadores),
		Convocatorias: convocatorias, ActuacionesPendientes: actuaciones,
	}}
}

func instanteOpcional(instante time.Time) *time.Time {
	if instante.IsZero() {
		return nil
	}
	copia := instante
	return &copia
}

func dependenciaNula(dependencia any) bool {
	if dependencia == nil {
		return true
	}
	valor := reflect.ValueOf(dependencia)
	switch valor.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return valor.IsNil()
	default:
		return false
	}
}
