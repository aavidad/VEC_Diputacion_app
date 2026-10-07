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

	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

const RutaRPTPublicaPersonalV2 = "/api/vec/personal/rpt-publica/v2"

// 100 runas UTF-8 pueden ocupar 400 bytes, codificados como 1200 caracteres
// %XX. Más clave de categoría (60), centro (64) y nombres de parámetros, 2048
// cubre toda consulta válida y sigue acotando el parser HTTP.
const maximoQueryRPTPublicaV2 = 2048

var ErrHandlerRPTPublicaV2Invalido = errors.New("httpapi: consulta RPT publicada v2 no disponible")

// La autoridad recibe una identidad sellada por el canal interno. Ningún
// selector HTTP puede aportarla ni escoger un perfil distinto.
type AutoridadContextoRPTPublicaV2 interface {
	ResolverContextoRPTPublicaV2(context.Context) (vecdomain.ContextoActor, error)
}

type ConsultorRPTPublicaV2 interface {
	Consultar(context.Context, vecdomain.ContextoActor, personaldomain.FiltroRPTPublicaV2) (personalports.PaginaRPTPublicaV2, error)
}

type DenegacionRPTPublicaV2 struct {
	CorrelacionRef string
	Ruta           string
	Motivo         string
	ActorRef       string
}

type AuditorDenegacionRPTPublicaV2 interface {
	RegistrarDenegacionRPTPublicaV2(context.Context, DenegacionRPTPublicaV2) error
}

type handlerRPTPublicaV2 struct {
	autoridad AutoridadContextoRPTPublicaV2
	consulta  ConsultorRPTPublicaV2
	auditoria AuditorDenegacionRPTPublicaV2
}

func NewHandlerRPTPublicaV2(a AutoridadContextoRPTPublicaV2, c ConsultorRPTPublicaV2, u AuditorDenegacionRPTPublicaV2) (http.Handler, error) {
	if dependenciaHTTPNula(a) || dependenciaHTTPNula(c) || dependenciaHTTPNula(u) {
		return nil, ErrHandlerRPTPublicaV2Invalido
	}
	return &handlerRPTPublicaV2{autoridad: a, consulta: c, auditoria: u}, nil
}

func (h *handlerRPTPublicaV2) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || r == nil || dependenciaHTTPNula(h.autoridad) || dependenciaHTTPNula(h.consulta) || dependenciaHTTPNula(h.auditoria) {
		responderRPTPublicaV2(w, http.StatusServiceUnavailable, "servicio_no_disponible", nil)
		return
	}
	if !peticionRutaExactaCanonica(r) || r.URL.Path != RutaRPTPublicaPersonalV2 {
		responderRPTPublicaV2(w, http.StatusNotFound, "recurso_no_encontrado", nil)
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		responderRPTPublicaV2(w, http.StatusMethodNotAllowed, "metodo_no_permitido", nil)
		return
	}
	if r.ContentLength != 0 || len(r.TransferEncoding) != 0 || len(r.Trailer) != 0 ||
		(r.Body != nil && r.Body != http.NoBody) || cabeceraOrganizacionHistoricaPresente(r.Header, "Cookie") ||
		cabeceraOrganizacionHistoricaPresente(r.Header, "Proxy-Authorization") ||
		cabeceraOrganizacionHistoricaPresente(r.Header, "Authorization") {
		responderRPTPublicaV2(w, http.StatusBadRequest, "peticion_no_valida", nil)
		return
	}
	filtro, err := leerFiltroRPTPublicaV2(r.URL.RawQuery)
	if err != nil {
		responderRPTPublicaV2(w, http.StatusBadRequest, "peticion_no_valida", nil)
		return
	}
	actor, err := h.autoridad.ResolverContextoRPTPublicaV2(r.Context())
	if err != nil || actor.Validar() != nil {
		switch {
		case errors.Is(err, ErrAutenticacionRutaExactaRequerida), errors.Is(err, vecdomain.ErrContextoActorNoResuelto):
			h.denegar(w, r.Context(), http.StatusUnauthorized, "autenticacion_requerida", "")
		case errors.Is(err, ErrAccesoRutaExactaDenegado), errors.Is(err, vecdomain.ErrAutorizacionDenegada), errors.Is(err, vecdomain.ErrPermissionDenied):
			h.denegar(w, r.Context(), http.StatusForbidden, "acceso_denegado", "")
		default:
			responderRPTPublicaV2(w, http.StatusServiceUnavailable, "servicio_no_disponible", nil)
		}
		return
	}
	pagina, err := h.consulta.Consultar(r.Context(), actor, filtro)
	if err != nil {
		switch {
		case errors.Is(err, personaldomain.ErrRPTPublicaV2Denegada):
			h.denegar(w, r.Context(), http.StatusForbidden, "acceso_denegado", actor.Principal.ID)
		case errors.Is(err, personaldomain.ErrRPTPublicaV2Invalida):
			responderRPTPublicaV2(w, http.StatusBadRequest, "peticion_no_valida", nil)
		default:
			responderRPTPublicaV2(w, http.StatusServiceUnavailable, "servicio_no_disponible", nil)
		}
		return
	}
	var items any = pagina.Categorias
	if pagina.Vista == "puestos" {
		items = pagina.Puestos
	}
	responderRPTPublicaV2(w, http.StatusOK, "", map[string]any{"data": map[string]any{"rpt": map[string]any{
		"items": items, "total": pagina.Total, "limit": pagina.Limite, "offset": pagina.Offset, "vista": pagina.Vista,
		"esquema": personaldomain.EsquemaCandidatoRPTPublicaV2, "estado": pagina.Estado,
		"publicacion_ref": pagina.PublicacionRef, "corte": pagina.Corte, "huella_sha256": pagina.HuellaSHA256,
		"fuente": map[string]any{"documento": pagina.Fuente.Documento, "importacion": pagina.Fuente.Importacion,
			"generado_en": pagina.Fuente.GeneradoEn, "aviso": pagina.Fuente.Aviso}, "resumen": pagina.Resumen,
		"categorias_pendientes_grupo": pagina.CategoriasPendientesGrupo, "evidencia": pagina.Evidencia,
	}}})
}

func leerFiltroRPTPublicaV2(raw string) (personaldomain.FiltroRPTPublicaV2, error) {
	var f personaldomain.FiltroRPTPublicaV2
	if len(raw) > maximoQueryRPTPublicaV2 || strings.Contains(raw, ";") {
		return f, personaldomain.ErrRPTPublicaV2Invalida
	}
	q, err := url.ParseQuery(raw)
	if err != nil {
		return f, personaldomain.ErrRPTPublicaV2Invalida
	}
	for clave, valores := range q {
		if (clave != "vista" && clave != "q" && clave != "limit" && clave != "offset" &&
			clave != "categoria_clave" && clave != "centro_codigo") || len(valores) != 1 {
			return f, personaldomain.ErrRPTPublicaV2Invalida
		}
	}
	f.Vista = q.Get("vista")
	if f.Vista == "" {
		f.Vista = "categorias"
	}
	f.Q = q.Get("q")
	f.CategoriaClave = q.Get("categoria_clave")
	f.CentroCodigo = q.Get("centro_codigo")
	f.Limite, err = enteroRPTPublica(q.Get("limit"))
	if err != nil {
		return f, personaldomain.ErrRPTPublicaV2Invalida
	}
	f.Offset, err = enteroRPTPublica(q.Get("offset"))
	if err != nil || f.Validar() != nil {
		return f, personaldomain.ErrRPTPublicaV2Invalida
	}
	return f, nil
}

func (h *handlerRPTPublicaV2) denegar(w http.ResponseWriter, ctx context.Context, estado int, motivo, actor string) {
	orden := DenegacionRPTPublicaV2{CorrelacionRef: nuevaCorrelacionRutaExacta(), Ruta: RutaRPTPublicaPersonalV2, Motivo: motivo, ActorRef: actor}
	ctxAudit, cancelar := context.WithTimeout(context.WithoutCancel(ctx), plazoarranque.Ampliar(250*time.Millisecond))
	defer cancelar()
	if h.auditoria.RegistrarDenegacionRPTPublicaV2(ctxAudit, orden) != nil {
		responderRPTPublicaV2(w, http.StatusServiceUnavailable, "servicio_no_disponible", nil)
		return
	}
	responderRPTPublicaV2(w, estado, motivo, nil)
}

func responderRPTPublicaV2(w http.ResponseWriter, estado int, codigo string, datos any) {
	for _, nombre := range []string{"Set-Cookie", "Access-Control-Allow-Origin", "Access-Control-Allow-Credentials", "Location", "Content-Encoding"} {
		w.Header().Del(nombre)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, no-transform")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; base-uri 'none'; frame-ancestors 'none'")
	if datos == nil {
		datos = map[string]any{"error": map[string]string{"codigo": codigo, "clave_i18n": "api.personal.rpt_publica_v2.error." + codigo}}
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
