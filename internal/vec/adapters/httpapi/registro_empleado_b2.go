package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	personalapp "vec-diputacion-granada/internal/modules/personal/application"
	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

const (
	PrefijoFichaEmpleadoB2        = "/api/vec/personal/empleados/"
	RutaVacantesEmpleadoB2        = "/api/vec/personal/vacantes"
	maximoQueryRegistroEmpleadoB2 = 1200
)

var ErrHandlerRegistroEmpleadoB2Invalido = errors.New("personal http: registro de empleado no disponible")

// La autoridad resuelve el contexto y el organismo por el canal del servidor.
// Ninguna cabecera ni ruta de la petición concede acceso a los datos.
type AutoridadContextoRegistroEmpleadoB2 interface {
	ResolverContextoRegistroEmpleadoB2(context.Context) (vecdomain.ContextoActor, string, error)
}

type ConsultorRegistroEmpleadoB2 interface {
	ConsultarFicha(context.Context, personaldomain.SolicitudFichaEmpleadoB2) (personalports.ResultadoFichaEmpleadoB2, error)
	ConsultarVacantes(context.Context, personaldomain.SolicitudVacantesB2) (personalports.ResultadoVacantesB2, error)
}

type DenegacionRegistroEmpleadoB2 struct {
	CorrelacionRef string
	Motivo         string
	Ruta           string
	ActorRef       string
}

type AuditorDenegacionRegistroEmpleadoB2 interface {
	RegistrarDenegacionRegistroEmpleadoB2(context.Context, DenegacionRegistroEmpleadoB2) error
}

type handlerRegistroEmpleadoB2 struct {
	autoridad AutoridadContextoRegistroEmpleadoB2
	consulta  ConsultorRegistroEmpleadoB2
	auditoria AuditorDenegacionRegistroEmpleadoB2
	vacantes  bool
}

var _ ConsultorRegistroEmpleadoB2 = (*personalapp.ServicioRegistroEmpleadoB2)(nil)

func NewHandlerFichaEmpleadoB2(a AutoridadContextoRegistroEmpleadoB2, c ConsultorRegistroEmpleadoB2, auditor AuditorDenegacionRegistroEmpleadoB2) (http.Handler, error) {
	return nuevoHandlerRegistroEmpleadoB2(a, c, auditor, false)
}

func NewHandlerVacantesEmpleadoB2(a AutoridadContextoRegistroEmpleadoB2, c ConsultorRegistroEmpleadoB2, auditor AuditorDenegacionRegistroEmpleadoB2) (http.Handler, error) {
	return nuevoHandlerRegistroEmpleadoB2(a, c, auditor, true)
}

func nuevoHandlerRegistroEmpleadoB2(a AutoridadContextoRegistroEmpleadoB2, c ConsultorRegistroEmpleadoB2, auditor AuditorDenegacionRegistroEmpleadoB2, vacantes bool) (http.Handler, error) {
	if dependenciaHTTPNula(a) || dependenciaHTTPNula(c) || dependenciaHTTPNula(auditor) {
		return nil, ErrHandlerRegistroEmpleadoB2Invalido
	}
	return &handlerRegistroEmpleadoB2{autoridad: a, consulta: c, auditoria: auditor, vacantes: vacantes}, nil
}

func (h *handlerRegistroEmpleadoB2) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || r == nil || dependenciaHTTPNula(h.autoridad) || dependenciaHTTPNula(h.consulta) || dependenciaHTTPNula(h.auditoria) {
		responderRegistroEmpleadoB2(w, http.StatusServiceUnavailable, "servicio_no_disponible", nil)
		return
	}
	if !peticionRutaExactaCanonica(r) || !h.rutaValida(r.URL.Path) {
		responderRegistroEmpleadoB2(w, http.StatusNotFound, "recurso_no_encontrado", nil)
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		responderRegistroEmpleadoB2(w, http.StatusMethodNotAllowed, "metodo_no_permitido", nil)
		return
	}
	if r.ContentLength != 0 || len(r.TransferEncoding) != 0 || len(r.Trailer) != 0 ||
		(r.Body != nil && r.Body != http.NoBody) || cabeceraOrganizacionHistoricaPresente(r.Header, "Cookie") ||
		cabeceraOrganizacionHistoricaPresente(r.Header, "Proxy-Authorization") || cabeceraOrganizacionHistoricaPresente(r.Header, "Content-Encoding") {
		responderRegistroEmpleadoB2(w, http.StatusBadRequest, "peticion_no_valida", nil)
		return
	}
	filtro, err := leerFiltroRegistroEmpleadoB2(r.URL.RawQuery, h.vacantes)
	if err != nil {
		responderRegistroEmpleadoB2(w, http.StatusBadRequest, "peticion_no_valida", nil)
		return
	}
	empleadoRef := ""
	if !h.vacantes {
		empleadoRef = strings.TrimPrefix(r.URL.Path, PrefijoFichaEmpleadoB2)
		if !personaldomain.ReferenciaEmpleadoValida(empleadoRef) {
			responderRegistroEmpleadoB2(w, http.StatusNotFound, "recurso_no_encontrado", nil)
			return
		}
	}
	actor, organismo, err := h.autoridad.ResolverContextoRegistroEmpleadoB2(r.Context())
	if err != nil {
		switch {
		case errors.Is(err, ErrAutenticacionRutaExactaRequerida), errors.Is(err, vecdomain.ErrContextoActorNoResuelto):
			h.denegar(w, r.Context(), http.StatusUnauthorized, "autenticacion_requerida", "")
		case errors.Is(err, ErrAccesoRutaExactaDenegado), errors.Is(err, vecdomain.ErrAutorizacionDenegada), errors.Is(err, vecdomain.ErrPermissionDenied):
			h.denegar(w, r.Context(), http.StatusForbidden, "acceso_denegado", "")
		default:
			responderRegistroEmpleadoB2(w, http.StatusServiceUnavailable, "servicio_no_disponible", nil)
		}
		return
	}
	if actor.Validar() != nil || (h.vacantes && organismo == "") {
		responderRegistroEmpleadoB2(w, http.StatusServiceUnavailable, "servicio_no_disponible", nil)
		return
	}
	if h.vacantes {
		solicitud := personaldomain.SolicitudVacantesB2{Actor: actor, OrganismoRef: organismo, Corte: personaldomain.CorteEmpleadoB2{VigenteEn: filtro.vigenteEn, ConocidoEn: filtro.conocidoEn}, Limite: filtro.limite, Cursor: filtro.cursor}
		resultado, err := h.consulta.ConsultarVacantes(r.Context(), solicitud)
		if err != nil {
			h.errorConsulta(w, r.Context(), actor.Principal.ID, err)
			return
		}
		material, err := personaldomain.NuevoMaterialVacantesB2(solicitud)
		if err != nil || resultado.Pagina.ValidarPara(material) != nil || !evidenciaRegistroEmpleadoB2HTTPValida(resultado.Evidencia) {
			responderRegistroEmpleadoB2(w, http.StatusServiceUnavailable, "servicio_no_disponible", nil)
			return
		}
		responderRegistroEmpleadoB2(w, http.StatusOK, "", map[string]any{"data": map[string]any{"pagina": resultado.Pagina, "evidencia": resultado.Evidencia}})
		return
	}
	solicitud := personaldomain.SolicitudFichaEmpleadoB2{Actor: actor, EmpleadoRef: empleadoRef, Corte: personaldomain.CorteEmpleadoB2{VigenteEn: filtro.vigenteEn, ConocidoEn: filtro.conocidoEn}}
	resultado, err := h.consulta.ConsultarFicha(r.Context(), solicitud)
	if err != nil {
		h.errorConsulta(w, r.Context(), actor.Principal.ID, err)
		return
	}
	material, err := personaldomain.NuevoMaterialFichaEmpleadoB2(solicitud)
	if err != nil || resultado.Ficha.ValidarPara(material) != nil || !evidenciaRegistroEmpleadoB2HTTPValida(resultado.Evidencia) {
		responderRegistroEmpleadoB2(w, http.StatusServiceUnavailable, "servicio_no_disponible", nil)
		return
	}
	responderRegistroEmpleadoB2(w, http.StatusOK, "", map[string]any{"data": map[string]any{"ficha": resultado.Ficha, "evidencia": resultado.Evidencia}})
}

func evidenciaRegistroEmpleadoB2HTTPValida(e personalports.EvidenciaRegistroEmpleadoB2) bool {
	return e.ReciboRef != "" && e.DecisionRef != "" && e.EfectoRef != "" &&
		e.ConsumoHuellaSHA256 != "" && e.AuditoriaRef != "" && !e.ConsultadaEn.IsZero()
}

func (h *handlerRegistroEmpleadoB2) rutaValida(ruta string) bool {
	if h.vacantes {
		return ruta == RutaVacantesEmpleadoB2
	}
	return strings.HasPrefix(ruta, PrefijoFichaEmpleadoB2) &&
		!strings.Contains(strings.TrimPrefix(ruta, PrefijoFichaEmpleadoB2), "/")
}

type filtroRegistroEmpleadoB2 struct {
	vigenteEn  personaldomain.FechaCivil
	conocidoEn time.Time
	limite     int
	cursor     string
}

func leerFiltroRegistroEmpleadoB2(raw string, vacantes bool) (filtroRegistroEmpleadoB2, error) {
	var f filtroRegistroEmpleadoB2
	if len(raw) > maximoQueryRegistroEmpleadoB2 || strings.Contains(raw, ";") {
		return f, personaldomain.ErrRegistroEmpleadoB2Invalido
	}
	q, err := url.ParseQuery(raw)
	if err != nil {
		return f, personaldomain.ErrRegistroEmpleadoB2Invalido
	}
	for k, valores := range q {
		permitido := k == "vigente_en" || k == "conocido_en" || (vacantes && (k == "limite" || k == "cursor"))
		if !permitido || len(valores) != 1 || valores[0] == "" || valores[0] != strings.TrimSpace(valores[0]) {
			return f, personaldomain.ErrRegistroEmpleadoB2Invalido
		}
	}
	f.vigenteEn, err = personaldomain.NuevaFechaCivil(q.Get("vigente_en"))
	if err != nil {
		return f, personaldomain.ErrRegistroEmpleadoB2Invalido
	}
	valor := q.Get("conocido_en")
	f.conocidoEn, err = time.Parse("2006-01-02T15:04:05.000000Z", valor)
	if err != nil || f.conocidoEn.Format("2006-01-02T15:04:05.000000Z") != valor {
		return f, personaldomain.ErrRegistroEmpleadoB2Invalido
	}
	if vacantes {
		f.limite = 50
		if valor := q.Get("limite"); valor != "" {
			f.limite, err = strconv.Atoi(valor)
			if err != nil || strconv.Itoa(f.limite) != valor {
				return f, personaldomain.ErrRegistroEmpleadoB2Invalido
			}
		}
		if f.limite < 1 || f.limite > 100 || len(q.Get("cursor")) > 256 {
			return f, personaldomain.ErrRegistroEmpleadoB2Invalido
		}
		f.cursor = q.Get("cursor")
	}
	return f, nil
}

func (h *handlerRegistroEmpleadoB2) errorConsulta(w http.ResponseWriter, ctx context.Context, actor string, err error) {
	switch {
	case errors.Is(err, personaldomain.ErrRegistroEmpleadoB2Denegado):
		h.denegar(w, ctx, http.StatusForbidden, "acceso_denegado", actor)
	case errors.Is(err, personaldomain.ErrCoberturaVacantesB2NoAcreditada):
		responderRegistroEmpleadoB2(w, http.StatusServiceUnavailable, "cobertura_no_acreditada", nil)
	case errors.Is(err, personaldomain.ErrRegistroEmpleadoB2NoEncontrado):
		responderRegistroEmpleadoB2(w, http.StatusNotFound, "recurso_no_encontrado", nil)
	case errors.Is(err, personaldomain.ErrRegistroEmpleadoB2Invalido):
		responderRegistroEmpleadoB2(w, http.StatusBadRequest, "peticion_no_valida", nil)
	default:
		responderRegistroEmpleadoB2(w, http.StatusServiceUnavailable, "servicio_no_disponible", nil)
	}
}

func (h *handlerRegistroEmpleadoB2) denegar(w http.ResponseWriter, ctx context.Context, estado int, codigo, actor string) {
	ruta := PrefijoFichaEmpleadoB2 + "{emp_ref}"
	if h.vacantes {
		ruta = RutaVacantesEmpleadoB2
	}
	orden := DenegacionRegistroEmpleadoB2{CorrelacionRef: nuevaCorrelacionRutaExacta(), Motivo: codigo, Ruta: ruta, ActorRef: actor}
	ctxAuditoria, cancelar := context.WithTimeout(context.WithoutCancel(ctx), plazoMaximoAuditoriaFronteraRutaExacta)
	defer cancelar()
	if err := h.auditoria.RegistrarDenegacionRegistroEmpleadoB2(ctxAuditoria, orden); err != nil {
		responderRegistroEmpleadoB2(w, http.StatusServiceUnavailable, "servicio_no_disponible", nil)
		return
	}
	responderRegistroEmpleadoB2(w, estado, codigo, nil)
}

func responderRegistroEmpleadoB2(w http.ResponseWriter, estado int, codigo string, datos any) {
	for _, k := range []string{"Set-Cookie", "Access-Control-Allow-Origin", "Access-Control-Allow-Credentials", "Location", "Content-Encoding"} {
		w.Header().Del(k)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, no-transform")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; base-uri 'none'; frame-ancestors 'none'")
	if datos == nil {
		datos = map[string]any{"error": map[string]string{"codigo": codigo, "clave_i18n": "api.personal.registro_empleado_b2.error." + codigo}}
	}
	contenido, err := json.Marshal(datos)
	if err != nil {
		estado = http.StatusServiceUnavailable
		contenido = []byte(`{"error":{"codigo":"servicio_no_disponible"}}`)
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(contenido)))
	w.WriteHeader(estado)
	_, _ = w.Write(contenido)
}
