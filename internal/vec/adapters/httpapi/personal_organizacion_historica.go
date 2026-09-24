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
	RutaOrganizacionHistoricaPersonal = "/api/vec/personal/organizacion-historica"
	maximoQueryOrganizacionHistorica  = 1200
)

var ErrHandlerOrganizacionHistoricaInvalido = errors.New("personal http: consulta de organizacion historica no disponible")

// El contexto efectivo se resuelve en el servidor. Organismo y unidad son el
// ámbito exacto concedido; ningún identificador recibido por HTTP los amplía.
type AutoridadContextoOrganizacionHistorica interface {
	ResolverContextoOrganizacionHistorica(context.Context) (vecdomain.ContextoActor, string, string, error)
}

type ConsultorOrganizacionHistorica interface {
	Consultar(context.Context, personaldomain.SolicitudConsultaOrganizacionHistorica) (personalports.ResultadoConsultaOrganizacionHistorica, error)
}

// La composición conecta este puerto a la autoridad común de auditoría.
// Cada 401/403 debe quedar registrado antes de responder.
type AuditorDenegacionOrganizacionHistorica interface {
	RegistrarDenegacionOrganizacionHistorica(context.Context, DenegacionOrganizacionHistorica) error
}

type DenegacionOrganizacionHistorica struct {
	CorrelacionRef string
	Motivo         string
	Ruta           string
	ActorRef       string
}

type handlerOrganizacionHistorica struct {
	autoridad AutoridadContextoOrganizacionHistorica
	consulta  ConsultorOrganizacionHistorica
	auditoria AuditorDenegacionOrganizacionHistorica
}

var _ ConsultorOrganizacionHistorica = (*personalapp.ServicioConsultaOrganizacionHistorica)(nil)

func NewHandlerOrganizacionHistoricaPersonal(a AutoridadContextoOrganizacionHistorica, c ConsultorOrganizacionHistorica, auditor AuditorDenegacionOrganizacionHistorica) (http.Handler, error) {
	if dependenciaHTTPNula(a) || dependenciaHTTPNula(c) || dependenciaHTTPNula(auditor) {
		return nil, ErrHandlerOrganizacionHistoricaInvalido
	}
	return &handlerOrganizacionHistorica{a, c, auditor}, nil
}

func (h *handlerOrganizacionHistorica) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || r == nil || dependenciaHTTPNula(h.autoridad) || dependenciaHTTPNula(h.consulta) || dependenciaHTTPNula(h.auditoria) {
		responderOrganizacionHistorica(w, http.StatusServiceUnavailable, "servicio_no_disponible", nil)
		return
	}
	if !peticionRutaExactaCanonica(r) || r.URL.Path != RutaOrganizacionHistoricaPersonal {
		responderOrganizacionHistorica(w, http.StatusNotFound, "recurso_no_encontrado", nil)
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		responderOrganizacionHistorica(w, http.StatusMethodNotAllowed, "metodo_no_permitido", nil)
		return
	}
	if r.ContentLength != 0 || len(r.TransferEncoding) != 0 || len(r.Trailer) != 0 ||
		(r.Body != nil && r.Body != http.NoBody) || cabeceraOrganizacionHistoricaPresente(r.Header, "Cookie") ||
		cabeceraOrganizacionHistoricaPresente(r.Header, "Proxy-Authorization") {
		responderOrganizacionHistorica(w, http.StatusBadRequest, "peticion_no_valida", nil)
		return
	}
	filtros, err := leerFiltrosOrganizacionHistorica(r.URL.RawQuery)
	if err != nil {
		responderOrganizacionHistorica(w, http.StatusBadRequest, "peticion_no_valida", nil)
		return
	}
	validacionCliente := filtros
	validacionCliente.OrganismoRef = "org_validacion"
	if validacionCliente.Validar() != nil {
		responderOrganizacionHistorica(w, http.StatusBadRequest, "peticion_no_valida", nil)
		return
	}
	actor, organismo, unidad, err := h.autoridad.ResolverContextoOrganizacionHistorica(r.Context())
	if err != nil {
		if errors.Is(err, ErrAutenticacionRutaExactaRequerida) || errors.Is(err, vecdomain.ErrContextoActorNoResuelto) {
			h.denegar(w, r.Context(), http.StatusUnauthorized, "autenticacion_requerida", "")
			return
		}
		if errors.Is(err, ErrAccesoRutaExactaDenegado) || errors.Is(err, vecdomain.ErrAutorizacionDenegada) || errors.Is(err, vecdomain.ErrPermissionDenied) {
			h.denegar(w, r.Context(), http.StatusForbidden, "acceso_denegado", "")
			return
		}
		responderOrganizacionHistorica(w, http.StatusServiceUnavailable, "servicio_no_disponible", nil)
		return
	}
	if actor.Validar() != nil || organismo == "" {
		responderOrganizacionHistorica(w, http.StatusServiceUnavailable, "servicio_no_disponible", nil)
		return
	}
	if unidad != "" && filtros.UnidadClave != "" && filtros.UnidadClave != unidad {
		h.denegar(w, r.Context(), http.StatusForbidden, "acceso_denegado", actor.Principal.ID)
		return
	}
	filtros.OrganismoRef = organismo
	if filtros.UnidadClave == "" {
		filtros.UnidadClave = unidad
	}
	if filtros.Validar() != nil {
		responderOrganizacionHistorica(w, http.StatusServiceUnavailable, "servicio_no_disponible", nil)
		return
	}
	resultado, err := h.consulta.Consultar(r.Context(), personaldomain.SolicitudConsultaOrganizacionHistorica{Selector: filtros, Actor: actor})
	if err != nil {
		if errors.Is(err, personaldomain.ErrConsultaOrganizacionHistoricaDenegada) {
			h.denegar(w, r.Context(), http.StatusForbidden, "acceso_denegado", actor.Principal.ID)
			return
		}
		if errors.Is(err, personaldomain.ErrConsultaOrganizacionHistoricaInvalida) {
			responderOrganizacionHistorica(w, http.StatusBadRequest, "peticion_no_valida", nil)
			return
		}
		responderOrganizacionHistorica(w, http.StatusServiceUnavailable, "servicio_no_disponible", nil)
		return
	}
	if !selectoresOrganizacionHistoricaIguales(resultado.Pagina.Selector, filtros) || resultado.Evidencia.ReciboRef == "" ||
		resultado.Evidencia.AuditoriaRef == "" || resultado.Evidencia.DecisionRef == "" ||
		resultado.Evidencia.EfectoRef == "" || resultado.Evidencia.ConsumoHuellaSHA256 == "" || resultado.Evidencia.ConsultadaEn.IsZero() {
		responderOrganizacionHistorica(w, http.StatusServiceUnavailable, "servicio_no_disponible", nil)
		return
	}
	responderOrganizacionHistorica(w, http.StatusOK, "", map[string]any{"data": resultado})
}

func selectoresOrganizacionHistoricaIguales(a, b personaldomain.SelectorOrganizacionHistorica) bool {
	if !a.ConocidoEn.Equal(b.ConocidoEn) {
		return false
	}
	a.ConocidoEn = b.ConocidoEn
	return a == b
}

func cabeceraOrganizacionHistoricaPresente(h http.Header, nombre string) bool {
	for clave := range h {
		if strings.EqualFold(clave, nombre) {
			return true
		}
	}
	return false
}

func (h *handlerOrganizacionHistorica) denegar(w http.ResponseWriter, ctx context.Context, estado int, codigo, actor string) {
	orden := DenegacionOrganizacionHistorica{
		CorrelacionRef: nuevaCorrelacionRutaExacta(), Motivo: codigo,
		Ruta: RutaOrganizacionHistoricaPersonal, ActorRef: actor,
	}
	ctxAuditoria, cancelar := context.WithTimeout(context.WithoutCancel(ctx), plazoMaximoAuditoriaFronteraRutaExacta)
	defer cancelar()
	if err := h.auditoria.RegistrarDenegacionOrganizacionHistorica(ctxAuditoria, orden); err != nil {
		responderOrganizacionHistorica(w, http.StatusServiceUnavailable, "servicio_no_disponible", nil)
		return
	}
	responderOrganizacionHistorica(w, estado, codigo, nil)
}

func leerFiltrosOrganizacionHistorica(raw string) (personaldomain.SelectorOrganizacionHistorica, error) {
	var s personaldomain.SelectorOrganizacionHistorica
	if len(raw) > maximoQueryOrganizacionHistorica || strings.Contains(raw, ";") {
		return s, personaldomain.ErrConsultaOrganizacionHistoricaInvalida
	}
	q, err := url.ParseQuery(raw)
	if err != nil {
		return s, personaldomain.ErrConsultaOrganizacionHistoricaInvalida
	}
	for k, valores := range q {
		switch k {
		case "unidad_clave", "vigente_en", "conocido_en", "version_rpt_ref", "version_plantilla_ref", "limite", "cursor":
		default:
			return s, personaldomain.ErrConsultaOrganizacionHistoricaInvalida
		}
		if len(valores) != 1 || valores[0] == "" || valores[0] != strings.TrimSpace(valores[0]) {
			return s, personaldomain.ErrConsultaOrganizacionHistoricaInvalida
		}
	}
	s.UnidadClave, s.VersionRPTRef, s.VersionPlantillaRef, s.Cursor = q.Get("unidad_clave"), q.Get("version_rpt_ref"), q.Get("version_plantilla_ref"), q.Get("cursor")
	if q.Get("vigente_en") == "" || q.Get("conocido_en") == "" {
		return s, personaldomain.ErrConsultaOrganizacionHistoricaInvalida
	}
	s.VigenteEn, err = personaldomain.NuevaFechaCivil(q.Get("vigente_en"))
	if err != nil {
		return s, personaldomain.ErrConsultaOrganizacionHistoricaInvalida
	}
	v := q.Get("conocido_en")
	s.ConocidoEn, err = time.Parse("2006-01-02T15:04:05.000000Z", v)
	if err != nil || s.ConocidoEn.Format("2006-01-02T15:04:05.000000Z") != v {
		return s, personaldomain.ErrConsultaOrganizacionHistoricaInvalida
	}
	s.Limite = 50
	if v := q.Get("limite"); v != "" {
		s.Limite, err = strconv.Atoi(v)
		if err != nil || strconv.Itoa(s.Limite) != v {
			return s, personaldomain.ErrConsultaOrganizacionHistoricaInvalida
		}
	}
	if s.VigenteEn.Validar() != nil || s.Limite < 1 || s.Limite > personaldomain.LimiteMaximoOrganizacionHistorica ||
		len(s.UnidadClave) > 160 || len(s.VersionRPTRef) > 160 || len(s.VersionPlantillaRef) > 160 || len(s.Cursor) > 256 {
		return s, personaldomain.ErrConsultaOrganizacionHistoricaInvalida
	}
	return s, nil
}

func responderOrganizacionHistorica(w http.ResponseWriter, estado int, codigo string, datos any) {
	for _, k := range []string{"Set-Cookie", "Access-Control-Allow-Origin", "Access-Control-Allow-Credentials", "Location", "Content-Encoding"} {
		w.Header().Del(k)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, no-transform")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; base-uri 'none'; frame-ancestors 'none'")
	if datos == nil {
		datos = map[string]any{"error": map[string]string{"codigo": codigo, "clave_i18n": "api.personal.organizacion_historica.error." + codigo}}
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
