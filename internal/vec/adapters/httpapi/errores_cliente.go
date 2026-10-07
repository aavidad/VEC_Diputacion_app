package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const limiteCuerpoErrorCliente = 512

// El límite global protege la cola técnica. No guarda identidades, IP ni
// contenido y se reinicia al minuto; el navegador aplica otro límite local.
var cupoErroresCliente = &limiteErroresCliente{}

type limiteErroresCliente struct {
	mu        sync.Mutex
	inicio    time.Time
	recibidos int
}

func (l *limiteErroresCliente) permitir(ahora time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.inicio.IsZero() || ahora.Sub(l.inicio) >= time.Minute {
		l.inicio, l.recibidos = ahora, 0
	}
	if l.recibidos >= 120 {
		return false
	}
	l.recibidos++
	return true
}

type errorClienteCerrado struct {
	Pantalla    string
	Codigo      domain.CodigoIncidenciaTecnica
	Correlacion string
}

type falloValidacionTelemetria string

const (
	falloLecturaTelemetria falloValidacionTelemetria = "lectura"
	falloJSONTelemetria    falloValidacionTelemetria = "json"
	falloOrigenTelemetria  falloValidacionTelemetria = "origen"
)

func claseFalloTelemetria(err error, clase falloValidacionTelemetria) falloValidacionTelemetria {
	if err == nil {
		return ""
	}
	return clase
}

func leerErrorCliente(r *http.Request) (errorClienteCerrado, int, falloValidacionTelemetria) {
	if r.Body == nil || r.ContentLength > limiteCuerpoErrorCliente {
		return errorClienteCerrado{}, http.StatusRequestEntityTooLarge, ""
	}
	bruto, err := io.ReadAll(io.LimitReader(r.Body, limiteCuerpoErrorCliente+1))
	if err != nil {
		return errorClienteCerrado{}, http.StatusRequestEntityTooLarge, claseFalloTelemetria(err, falloLecturaTelemetria)
	}
	if len(bruto) > limiteCuerpoErrorCliente {
		return errorClienteCerrado{}, http.StatusRequestEntityTooLarge, ""
	}
	lector := json.NewDecoder(bytes.NewReader(bruto))
	inicio, err := lector.Token()
	if err != nil {
		return errorClienteCerrado{}, http.StatusBadRequest, claseFalloTelemetria(err, falloJSONTelemetria)
	}
	if inicio != json.Delim('{') {
		return errorClienteCerrado{}, http.StatusBadRequest, ""
	}
	var dato errorClienteCerrado
	vistas := map[string]bool{}
	for lector.More() {
		clave, err := lector.Token()
		nombre, ok := clave.(string)
		if err != nil {
			return errorClienteCerrado{}, http.StatusBadRequest, claseFalloTelemetria(err, falloJSONTelemetria)
		}
		if !ok || vistas[nombre] {
			return errorClienteCerrado{}, http.StatusBadRequest, ""
		}
		vistas[nombre] = true
		var valor string
		if err := lector.Decode(&valor); err != nil {
			return errorClienteCerrado{}, http.StatusBadRequest, claseFalloTelemetria(err, falloJSONTelemetria)
		}
		switch nombre {
		case "pantalla":
			dato.Pantalla = valor
		case "codigo":
			dato.Codigo = domain.CodigoIncidenciaTecnica(valor)
		case "correlacion":
			dato.Correlacion = valor
		default:
			return errorClienteCerrado{}, http.StatusBadRequest, ""
		}
	}
	fin, err := lector.Token()
	if err != nil {
		return errorClienteCerrado{}, http.StatusBadRequest, claseFalloTelemetria(err, falloJSONTelemetria)
	}
	if fin != json.Delim('}') || len(vistas) != 3 {
		return errorClienteCerrado{}, http.StatusBadRequest, ""
	}
	if _, err := lector.Token(); err != io.EOF {
		return errorClienteCerrado{}, http.StatusBadRequest, claseFalloTelemetria(err, falloJSONTelemetria)
	}
	if dato.Pantalla != "portal_empleado" || !domain.EsCorrelacionTecnicaValida(dato.Correlacion) ||
		(dato.Codigo != domain.IncidenciaClienteFalloNoClasificado && dato.Codigo != domain.IncidenciaModuloWebNoCargado) {
		return errorClienteCerrado{}, http.StatusBadRequest, ""
	}
	return dato, 0, ""
}

func origenMismoCanal(r *http.Request) (bool, falloValidacionTelemetria) {
	origen := r.Header.Get("Origin")
	if origen == "" || len(r.Header.Values("Origin")) != 1 ||
		(r.Header.Get("Sec-Fetch-Site") != "" && r.Header.Get("Sec-Fetch-Site") != "same-origin") {
		return false, ""
	}
	u, err := url.Parse(origen)
	if err != nil {
		return false, claseFalloTelemetria(err, falloOrigenTelemetria)
	}
	if u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" || u.String() != origen {
		return false, ""
	}
	esquema := "http"
	if r.TLS != nil {
		esquema = "https"
	}
	return u.Scheme == esquema && strings.EqualFold(u.Host, r.Host), ""
}

func (h *Handler) registrarFalloValidacionTelemetria(r *http.Request, fallo falloValidacionTelemetria) {
	switch fallo {
	case falloLecturaTelemetria, falloJSONTelemetria, falloOrigenTelemetria:
		ports.EmitirIncidenciaTecnicaEnPeticion(r.Context(), h.emisorIncidencias, domain.SolicitudIncidenciaTecnica{
			Codigo: domain.IncidenciaRecoleccionDegradada, Componente: domain.ComponenteIncidenciaSupervision,
			Etapa: domain.EtapaIncidenciaValidacion,
		})
	}
}

func (h *Handler) atenderErroresCliente(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if r.URL.RawQuery != "" || !peticionRutaExactaCanonica(r) || r.URL.Path != "/api/vec/observabilidad/errores-cliente" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if !principal.HasPermission("vec.session.read") {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	origenValido, falloOrigen := origenMismoCanal(r)
	if !origenValido {
		h.registrarFalloValidacionTelemetria(r, falloOrigen)
		w.WriteHeader(http.StatusForbidden)
		return
	}
	aceptador, disponible := h.emisorIncidencias.(ports.AceptadorIncidenciasTecnicasConContexto)
	if !disponible || dependenciaRutaExactaNula(aceptador) {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	tipo, parametros, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || tipo != "application/json" || (len(parametros) != 0 && (len(parametros) != 1 || !strings.EqualFold(parametros["charset"], "utf-8"))) {
		w.WriteHeader(http.StatusUnsupportedMediaType)
		return
	}
	dato, estado, falloLectura := leerErrorCliente(r)
	if estado != 0 {
		h.registrarFalloValidacionTelemetria(r, falloLectura)
		w.WriteHeader(estado)
		return
	}
	if !cupoErroresCliente.permitir(time.Now()) {
		w.WriteHeader(http.StatusTooManyRequests)
		return
	}
	// Correlacion es un identificador aleatorio de la entrega del navegador,
	// validado y descartado. La única correlación del registro es la generada
	// por el middleware confiable para esta petición.
	_ = dato.Correlacion
	if !aceptador.AceptarConContexto(r.Context(), domain.SolicitudIncidenciaTecnica{
		Codigo: dato.Codigo, Componente: domain.ComponenteIncidenciaPortalWeb,
		Etapa: etapaErrorCliente(dato.Codigo),
	}) {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func etapaErrorCliente(codigo domain.CodigoIncidenciaTecnica) domain.EtapaIncidenciaTecnica {
	if codigo == domain.IncidenciaModuloWebNoCargado {
		return domain.EtapaIncidenciaCarga
	}
	return domain.EtapaIncidenciaEjecucion
}
