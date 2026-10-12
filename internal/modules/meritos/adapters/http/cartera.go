package meritoshttp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"

	"vec-diputacion-granada/internal/modules/meritos/application"
	"vec-diputacion-granada/internal/modules/meritos/ports"
)

const RutaCartera = "/api/vec/meritos/cartera"

// La identidad la prepara la composición desde certificado y sesión vigentes.
// El navegador solo elige filtros y página, nunca una persona ni un perfil.
type ProveedorSolicitudCartera interface {
	PrepararSolicitudCartera(*http.Request) (application.SolicitudCartera, error)
}

type LectorCartera interface {
	Consultar(context.Context, application.SolicitudCartera) (ports.PaginaCartera, error)
}

type HandlerCartera struct {
	proveedor ProveedorSolicitudCartera
	lector    LectorCartera
}

var _ http.Handler = (*HandlerCartera)(nil)
var _ LectorCartera = (*application.ServicioCartera)(nil)

func NuevoHandlerCartera(proveedor ProveedorSolicitudCartera, lector LectorCartera) (*HandlerCartera, error) {
	if dependenciaConsultaNula(proveedor) || dependenciaConsultaNula(lector) {
		return nil, ports.ErrCarteraNoDisponible
	}
	return &HandlerCartera{proveedor, lector}, nil
}

func (h *HandlerCartera) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	if r == nil || r.URL == nil || r.URL.Path != RutaCartera ||
		r.URL.EscapedPath() != RutaCartera || r.URL.ForceQuery {
		responderErrorCartera(w, http.StatusNotFound, "meritos.error.ruta_no_encontrada")
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		responderErrorCartera(w, http.StatusMethodNotAllowed, "meritos.error.metodo_no_admitido")
		return
	}
	if h == nil || dependenciaConsultaNula(h.proveedor) || dependenciaConsultaNula(h.lector) ||
		r.Context().Err() != nil {
		responderErrorCartera(w, http.StatusServiceUnavailable, "meritos.error.cartera_no_disponible")
		return
	}
	// Se identifica y registra sesión antes de interpretar el selector para que
	// una petición autenticada inválida deje constancia nominal en el servicio.
	solicitud, err := h.proveedor.PrepararSolicitudCartera(r)
	if err != nil {
		responderFalloCartera(w, err)
		return
	}
	if r.Body != nil && r.Body != http.NoBody && r.ContentLength != 0 {
		h.responderEntradaInvalida(w, r, solicitud)
		return
	}
	filtro, limite, cursor, idioma, err := selectorHTTP(r.URL.RawQuery)
	if err != nil {
		h.responderEntradaInvalida(w, r, solicitud)
		return
	}
	solicitud.Filtro, solicitud.Limite, solicitud.Cursor, solicitud.Idioma = filtro, limite, cursor, idioma
	pagina, err := h.lector.Consultar(r.Context(), solicitud)
	if err != nil {
		responderFalloCartera(w, err)
		return
	}
	if r.Context().Err() != nil {
		responderErrorCartera(w, http.StatusServiceUnavailable, "meritos.error.cartera_no_disponible")
		return
	}
	contenido, err := json.Marshal(struct {
		Data ports.PaginaCartera `json:"data"`
	}{pagina})
	if err != nil || len(contenido) > 262144 {
		responderErrorCartera(w, http.StatusServiceUnavailable, "meritos.error.cartera_no_disponible")
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(contenido)
}

func (h *HandlerCartera) responderEntradaInvalida(w http.ResponseWriter, r *http.Request,
	solicitud application.SolicitudCartera) {
	// Consultar audita la entrada inválida bajo la sesión ya acreditada.
	// Un fallo del asiento impide confirmar incluso el 400.
	solicitud.Filtro, solicitud.Limite = "invalido", 0
	_, falloAuditado := h.lector.Consultar(r.Context(), solicitud)
	if !errors.Is(falloAuditado, ports.ErrCarteraDenegada) {
		responderErrorCartera(w, http.StatusServiceUnavailable, "meritos.error.cartera_no_disponible")
		return
	}
	responderErrorCartera(w, http.StatusBadRequest, "meritos.error.solicitud_invalida")
}

func selectorHTTP(raw string) (string, int, string, string, error) {
	if len(raw) > 4096 {
		return "", 0, "", "", ports.ErrCarteraDenegada
	}
	parametros, err := url.ParseQuery(raw)
	if err != nil || len(parametros) > 4 {
		return "", 0, "", "", ports.ErrCarteraDenegada
	}
	for clave, valores := range parametros {
		if len(valores) != 1 || clave != "filtro" && clave != "limite" && clave != "cursor" && clave != "lang" {
			return "", 0, "", "", ports.ErrCarteraDenegada
		}
	}
	filtro := parametros.Get("filtro")
	if filtro == "" {
		filtro = "todos"
	}
	if filtro != "todos" && filtro != "disponibles" && filtro != "retirados" {
		return "", 0, "", "", ports.ErrCarteraDenegada
	}
	limite := 25
	if parametro := parametros.Get("limite"); parametro != "" {
		valor, err := strconv.Atoi(parametro)
		if err != nil || valor < 1 || valor > 50 || strconv.Itoa(valor) != parametro {
			return "", 0, "", "", ports.ErrCarteraDenegada
		}
		limite = valor
	}
	cursor := parametros.Get("cursor")
	if len(cursor) > 2048 {
		return "", 0, "", "", ports.ErrCarteraDenegada
	}
	idioma := parametros.Get("lang")
	if idioma == "" {
		idioma = "es"
	}
	if idioma != "es" && idioma != "en" {
		return "", 0, "", "", ports.ErrCarteraDenegada
	}
	return filtro, limite, cursor, idioma, nil
}

func responderFalloCartera(w http.ResponseWriter, err error) {
	if errors.Is(err, ports.ErrCarteraSinSesion) {
		responderErrorCartera(w, http.StatusUnauthorized, "meritos.error.sesion_necesaria")
		return
	}
	if errors.Is(err, ports.ErrCarteraDenegada) {
		responderErrorCartera(w, http.StatusForbidden, "meritos.error.cartera_denegada")
		return
	}
	responderErrorCartera(w, http.StatusServiceUnavailable, "meritos.error.cartera_no_disponible")
}

func responderErrorCartera(w http.ResponseWriter, estado int, codigo string) {
	w.WriteHeader(estado)
	_ = json.NewEncoder(w).Encode(struct {
		Error struct {
			Codigo string `json:"codigo"`
		} `json:"error"`
	}{Error: struct {
		Codigo string `json:"codigo"`
	}{Codigo: codigo}})
}
