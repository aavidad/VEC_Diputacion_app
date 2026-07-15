// Package httppublico expone únicamente proyecciones públicas minimizadas.
// No contiene rutas personales, internas ni de administración.
package httppublico

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	aplicacionbolsa "vec-diputacion-granada/internal/modules/bolsa/application"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

const RutaConvocatorias = "/api/publico/bolsa/convocatorias"

type Handler struct {
	servicio *aplicacionbolsa.ServicioConsultaPublica
}

func NuevoHandler(servicio *aplicacionbolsa.ServicioConsultaPublica) (http.Handler, error) {
	if servicio == nil {
		return nil, aplicacionbolsa.ErrServicioConsultaPublicaInvalido
	}
	return &Handler{servicio: servicio}, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r == nil || h == nil || h.servicio == nil {
		responderError(w, http.StatusServiceUnavailable, "servicio_no_disponible", "Servicio no disponible.")
		return
	}
	if r.Method == http.MethodHead {
		w = escritorSinCuerpo{ResponseWriter: w}
	}
	if r.URL.RawPath != "" || strings.Contains(r.URL.EscapedPath(), "%") {
		responderError(w, http.StatusBadRequest, "ruta_invalida", "La ruta no es válida.")
		return
	}
	if r.URL.Path == RutaConvocatorias {
		h.listar(w, r)
		return
	}
	prefijo := RutaConvocatorias + "/"
	if strings.HasPrefix(r.URL.Path, prefijo) {
		identificador := strings.TrimPrefix(r.URL.Path, prefijo)
		if identificador == "" || strings.Contains(identificador, "/") {
			responderError(w, http.StatusNotFound, "recurso_no_encontrado", "Recurso no encontrado.")
			return
		}
		h.detalle(w, r, identificador)
		return
	}
	responderError(w, http.StatusNotFound, "recurso_no_encontrado", "Recurso no encontrado.")
}

type escritorSinCuerpo struct{ http.ResponseWriter }

func (escritorSinCuerpo) Write(contenido []byte) (int, error) { return len(contenido), nil }

func (h *Handler) listar(w http.ResponseWriter, r *http.Request) {
	if !metodoLectura(w, r) {
		return
	}
	consulta, err := decodificarConsulta(r)
	if err != nil {
		responderError(w, http.StatusBadRequest, "consulta_invalida", "Los filtros de consulta no son válidos.")
		return
	}
	resultado, err := h.servicio.Listar(r.Context(), consulta)
	if err != nil {
		responderErrorAplicacion(w, err)
		return
	}
	responderJSON(w, r, http.StatusOK, resultado)
}

func (h *Handler) detalle(w http.ResponseWriter, r *http.Request, identificador string) {
	if !metodoLectura(w, r) {
		return
	}
	if r.URL.RawQuery != "" {
		responderError(w, http.StatusBadRequest, "consulta_invalida", "El detalle no admite parámetros.")
		return
	}
	resultado, err := h.servicio.Obtener(r.Context(), identificador)
	if err != nil {
		responderErrorAplicacion(w, err)
		return
	}
	responderJSON(w, r, http.StatusOK, resultado)
}

func decodificarConsulta(r *http.Request) (aplicacionbolsa.SolicitudListadoPublico, error) {
	permitidos := map[string]bool{"texto": true, "tipo": true, "categoria": true, "estado": true, "plazo": true, "pagina": true, "tamano": true}
	if len(r.URL.RawQuery) > 2048 {
		return aplicacionbolsa.SolicitudListadoPublico{}, aplicacionbolsa.ErrFiltroPublicoInvalido
	}
	valores, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return aplicacionbolsa.SolicitudListadoPublico{}, aplicacionbolsa.ErrFiltroPublicoInvalido
	}
	for clave, lista := range valores {
		if !permitidos[clave] || len(lista) != 1 || lista[0] == "" {
			return aplicacionbolsa.SolicitudListadoPublico{}, aplicacionbolsa.ErrFiltroPublicoInvalido
		}
	}
	consulta := aplicacionbolsa.SolicitudListadoPublico{
		Texto: valores.Get("texto"), Tipo: valores.Get("tipo"), Categoria: valores.Get("categoria"), Estado: valores.Get("estado"),
	}
	if plazo := valores.Get("plazo"); plazo != "" {
		if plazo != "abierto" {
			return aplicacionbolsa.SolicitudListadoPublico{}, aplicacionbolsa.ErrFiltroPublicoInvalido
		}
		consulta.SoloPlazoAbierto = true
	}
	err = nil
	if pagina := valores.Get("pagina"); pagina != "" {
		consulta.Pagina, err = strconv.Atoi(pagina)
		if err != nil {
			return aplicacionbolsa.SolicitudListadoPublico{}, err
		}
	}
	if tamano := valores.Get("tamano"); tamano != "" {
		consulta.Tamano, err = strconv.Atoi(tamano)
		if err != nil {
			return aplicacionbolsa.SolicitudListadoPublico{}, err
		}
	}
	return consulta, nil
}

func metodoLectura(w http.ResponseWriter, r *http.Request) bool {
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		return true
	}
	w.Header().Set("Allow", "GET, HEAD")
	responderError(w, http.StatusMethodNotAllowed, "metodo_no_permitido", "Método no permitido.")
	return false
}

func responderErrorAplicacion(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, puertosbolsa.ErrConvocatoriaNoEncontrada):
		responderError(w, http.StatusNotFound, "convocatoria_no_encontrada", "Convocatoria no encontrada.")
	case errors.Is(err, aplicacionbolsa.ErrFiltroPublicoInvalido), errors.Is(err, puertosbolsa.ErrConsultaConvocatoriasInvalida):
		responderError(w, http.StatusBadRequest, "consulta_invalida", "La consulta no es válida.")
	default:
		responderError(w, http.StatusInternalServerError, "error_interno", "No se ha podido completar la consulta.")
	}
}

type respuestaError struct {
	Esquema string `json:"esquema"`
	Error   struct {
		Codigo  string `json:"codigo"`
		Mensaje string `json:"mensaje"`
	} `json:"error"`
}

func responderError(w http.ResponseWriter, estado int, codigo, mensaje string) {
	respuesta := respuestaError{Esquema: "vec.error.publico.v1"}
	respuesta.Error.Codigo = codigo
	respuesta.Error.Mensaje = mensaje
	responderJSONSinPeticion(w, estado, respuesta)
}

func responderJSON(w http.ResponseWriter, r *http.Request, estado int, valor any) {
	aplicarHeadersPublicos(w)
	w.WriteHeader(estado)
	if r.Method == http.MethodHead {
		return
	}
	_ = json.NewEncoder(w).Encode(valor)
}

func responderJSONSinPeticion(w http.ResponseWriter, estado int, valor any) {
	aplicarHeadersPublicos(w)
	w.WriteHeader(estado)
	_ = json.NewEncoder(w).Encode(valor)
}

func aplicarHeadersPublicos(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; base-uri 'none'; frame-ancestors 'none'")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=()")
	w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
}
