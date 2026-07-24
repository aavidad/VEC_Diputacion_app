// Package httpapi expone únicamente proyecciones públicas minimizadas.
// No contiene rutas personales, internas ni de administración.
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	aplicacionbolsa "vec-diputacion-granada/internal/modules/bolsa/publico/aplicacion"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/publico/puertos"
	"vec-diputacion-granada/internal/shared/limiteshttp"
)

const (
	RutaConvocatorias              = "/api/publico/bolsa/convocatorias"
	RutaCategorias                 = "/api/publico/bolsa/categorias"
	duracionMaximaPeticionPublica  = limiteshttp.DuracionMaximaPeticionPublica
	presupuestoEscrituraPublica    = limiteshttp.PresupuestoEscrituraPublica
	reservaLimpiezaPublica         = limiteshttp.ReservaLimpiezaPublica
	duracionMaximaRetencionCupo    = limiteshttp.DuracionMaximaRetencionCupo
	duracionMaximaOperacionPublica = limiteshttp.DuracionMaximaOperacionPublica
	// Peor detalle contractual: 128 categorías, 64 plazos, 256 requisitos,
	// 256 documentos y 128 ayudas con escape JSON conservador de seis bytes por
	// carácter. El fixture codifica menos de 14 MiB; 32 MiB incluyen objeto Go,
	// buffers y margen. Seis respuestas fijan 192 MiB simultáneos.
	maximoBytesRespuestaPublica   = 32 << 20
	presupuestoRespuestasPublicas = 192 << 20
	maximoOperacionesConcurrentes = 6
	maximoRespuestasConcurrentes  = presupuestoRespuestasPublicas / maximoBytesRespuestaPublica
)

type Handler struct {
	servicio          servicioConsultaPublica
	cuposServicio     chan struct{}
	cuposRespuesta    chan struct{}
	duracionOperacion time.Duration
}

type servicioConsultaPublica interface {
	Listar(context.Context, aplicacionbolsa.SolicitudListadoPublico) (aplicacionbolsa.ListadoConvocatoriasPublicas, error)
	Obtener(context.Context, string) (aplicacionbolsa.DetalleConvocatoriaPublica, error)
	ListarCategorias(context.Context) (aplicacionbolsa.DirectorioCategoriasPublicas, error)
}

func NuevoHandler(servicio *aplicacionbolsa.ServicioConsultaPublica) (http.Handler, error) {
	if servicio == nil {
		return nil, aplicacionbolsa.ErrServicioConsultaPublicaInvalido
	}
	return nuevoHandler(
		servicio, maximoOperacionesConcurrentes, maximoRespuestasConcurrentes,
		duracionMaximaOperacionPublica,
	), nil
}

func nuevoHandler(
	servicio servicioConsultaPublica,
	concurrenciaServicio int,
	concurrenciaRespuesta int,
	duracionOperacion time.Duration,
) *Handler {
	if !servicioConsultaPublicaDisponible(servicio) || concurrenciaServicio < 1 ||
		concurrenciaRespuesta < concurrenciaServicio || duracionOperacion <= 0 {
		return &Handler{}
	}
	return &Handler{
		servicio: servicio, cuposServicio: make(chan struct{}, concurrenciaServicio),
		cuposRespuesta: make(chan struct{}, concurrenciaRespuesta), duracionOperacion: duracionOperacion,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r == nil || h == nil || !servicioConsultaPublicaDisponible(h.servicio) ||
		h.cuposServicio == nil || h.cuposRespuesta == nil || h.duracionOperacion <= 0 {
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
	var servir func(http.ResponseWriter, *http.Request)
	if r.URL.Path == RutaConvocatorias {
		servir = h.listar
	} else if r.URL.Path == RutaCategorias {
		servir = h.listarCategorias
	} else {
		prefijo := RutaConvocatorias + "/"
		if strings.HasPrefix(r.URL.Path, prefijo) {
			identificador := strings.TrimPrefix(r.URL.Path, prefijo)
			if identificador == "" || strings.Contains(identificador, "/") {
				responderError(w, http.StatusNotFound, "recurso_no_encontrado", "Recurso no encontrado.")
				return
			}
			servir = func(w http.ResponseWriter, r *http.Request) { h.detalle(w, r, identificador) }
		}
	}
	if servir == nil {
		responderError(w, http.StatusNotFound, "recurso_no_encontrado", "Recurso no encontrado.")
		return
	}
	select {
	case h.cuposRespuesta <- struct{}{}:
		defer func() { <-h.cuposRespuesta }()
	default:
		w.Header().Set("Retry-After", "1")
		responderError(w, http.StatusTooManyRequests, "capacidad_temporal_agotada", "Inténtelo de nuevo en unos instantes.")
		return
	}
	ctx, cancelar := context.WithTimeout(r.Context(), h.duracionOperacion)
	defer cancelar()
	servir(w, r.WithContext(ctx))
}

func servicioConsultaPublicaDisponible(servicio servicioConsultaPublica) bool {
	if servicio == nil {
		return false
	}
	valor := reflect.ValueOf(servicio)
	switch valor.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return !valor.IsNil()
	default:
		return true
	}
}

func (h *Handler) listarCategorias(w http.ResponseWriter, r *http.Request) {
	if !metodoLectura(w, r) {
		return
	}
	if r.URL.RawQuery != "" {
		responderError(w, http.StatusBadRequest, "consulta_invalida", "El directorio de categorías no admite parámetros.")
		return
	}
	resultado, err, ejecutada := ejecutarServicioConCupo(h, w, func() (aplicacionbolsa.DirectorioCategoriasPublicas, error) {
		return h.servicio.ListarCategorias(r.Context())
	})
	if !ejecutada {
		return
	}
	if err != nil {
		responderErrorAplicacion(w, err)
		return
	}
	responderJSON(w, r, http.StatusOK, resultado)
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
	resultado, err, ejecutada := ejecutarServicioConCupo(h, w, func() (aplicacionbolsa.ListadoConvocatoriasPublicas, error) {
		return h.servicio.Listar(r.Context(), consulta)
	})
	if !ejecutada {
		return
	}
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
	resultado, err, ejecutada := ejecutarServicioConCupo(h, w, func() (aplicacionbolsa.DetalleConvocatoriaPublica, error) {
		return h.servicio.Obtener(r.Context(), identificador)
	})
	if !ejecutada {
		return
	}
	if err != nil {
		responderErrorAplicacion(w, err)
		return
	}
	responderJSON(w, r, http.StatusOK, resultado)
}

func ejecutarServicioConCupo[T any](
	h *Handler,
	w http.ResponseWriter,
	ejecutar func() (T, error),
) (T, error, bool) {
	var cero T
	select {
	case h.cuposServicio <- struct{}{}:
		defer func() { <-h.cuposServicio }()
	default:
		w.Header().Set("Retry-After", "1")
		responderError(w, http.StatusTooManyRequests, "capacidad_temporal_agotada", "Inténtelo de nuevo en unos instantes.")
		return cero, nil, false
	}
	resultado, err := ejecutar()
	return resultado, err, true
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
	case errors.Is(err, context.DeadlineExceeded):
		responderError(w, http.StatusGatewayTimeout, "tiempo_operacion_agotado", "La consulta ha superado el tiempo disponible.")
	case errors.Is(err, context.Canceled):
		responderError(w, http.StatusRequestTimeout, "peticion_cancelada", "La petición fue cancelada.")
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
