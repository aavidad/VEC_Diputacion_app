package ajustesreglas

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	app "vec-diputacion-granada/internal/modules/bolsa/application/ajustesreglas"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/reglas"
)

const Ruta = "/api/vec/bolsa/reglas/ajustes"
const Esquema = "vec.bolsa.reglas.ajustes.v1"

// La raíz aporta la sesión corporativa certificada; ningún campo del request
// puede sustituir el contexto de actor verificado.
type AutoridadSesion interface {
	Actor(context.Context) (vecdomain.ContextoActor, error)
}
type Operador interface {
	Consultar(context.Context, vecdomain.ContextoActor, int, *int64) (app.Lectura, error)
	Publicar(context.Context, vecdomain.ContextoActor, app.Solicitud) (app.Resultado, error)
	Motivos() []app.Motivo
}
type AuditorRechazo interface {
	RegistrarRechazo(context.Context, string, error) error
}

type Handler struct {
	sesion   AutoridadSesion
	operador Operador
	auditor  AuditorRechazo
}

func NuevoHandler(s AutoridadSesion, o Operador, a AuditorRechazo) (*Handler, error) {
	if dependenciaNula(s) || dependenciaNula(o) || dependenciaNula(a) {
		return nil, app.ErrNoDisponible
	}
	return &Handler{sesion: s, operador: o, auditor: a}, nil
}

func dependenciaNula(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return r.IsNil()
	}
	return false
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r == nil || r.URL == nil || r.URL.Path != Ruta || r.URL.RawPath != "" || r.URL.EscapedPath() != Ruta ||
		r.URL.Scheme != "" || r.URL.Host != "" || r.URL.User != nil || r.URL.Opaque != "" || r.URL.Fragment != "" || r.URL.RawFragment != "" ||
		(r.RequestURI != "" && r.RequestURI != Ruta && !strings.HasPrefix(r.RequestURI, Ruta+"?")) {
		responder(w, http.StatusNotFound, "recurso_no_encontrado")
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		responder(w, http.StatusMethodNotAllowed, "metodo_no_permitido")
		return
	}
	if cabecerasProhibidas(r.Header) || !origenPermitido(r) || r.Header.Get("Accept") != "application/json" {
		responder(w, http.StatusBadRequest, "ajustes_bolsa_entrada_invalida")
		return
	}
	if h == nil || h.sesion == nil || h.operador == nil || h.auditor == nil {
		responder(w, http.StatusServiceUnavailable, "ajustes_bolsa_no_disponibles")
		return
	}
	actor, err := h.sesion.Actor(r.Context())
	if err != nil || actor.Validar() != nil {
		if err == nil {
			err = app.ErrNoAutenticado
		}
		if h.auditor.RegistrarRechazo(r.Context(), Ruta, err) != nil {
			responder(w, http.StatusServiceUnavailable, "ajustes_bolsa_no_disponibles")
			return
		}
		responder(w, http.StatusUnauthorized, "ajustes_bolsa_no_autenticado")
		return
	}
	if r.Method == http.MethodGet {
		h.get(w, r, actor)
		return
	}
	h.post(w, r, actor)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request, actor vecdomain.ContextoActor) {
	if r.ContentLength != 0 || len(r.TransferEncoding) != 0 || r.Body != nil && r.Body != http.NoBody {
		responder(w, http.StatusBadRequest, "ajustes_bolsa_entrada_invalida")
		return
	}
	q, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		responder(w, http.StatusBadRequest, "ajustes_bolsa_entrada_invalida")
		return
	}
	if len(q) > 2 || r.URL.ForceQuery || r.URL.RawQuery != "" && strings.ContainsAny(r.URL.RawQuery, ";#") {
		responder(w, http.StatusBadRequest, "ajustes_bolsa_entrada_invalida")
		return
	}
	limite := 50
	if xs, ok := q["limite"]; ok {
		if len(xs) != 1 {
			responder(w, http.StatusBadRequest, "ajustes_bolsa_entrada_invalida")
			return
		}
		v, e := strconv.Atoi(xs[0])
		if e != nil || v < 1 || v > 50 {
			responder(w, http.StatusBadRequest, "ajustes_bolsa_entrada_invalida")
			return
		}
		limite = v
	}
	var antes *int64
	if xs, ok := q["antes_de_version"]; ok {
		if len(xs) != 1 {
			responder(w, http.StatusBadRequest, "ajustes_bolsa_entrada_invalida")
			return
		}
		v, e := strconv.ParseInt(xs[0], 10, 64)
		if e != nil || v < 2 || v > 10_000_000 {
			responder(w, http.StatusBadRequest, "ajustes_bolsa_entrada_invalida")
			return
		}
		antes = &v
	}
	for k := range q {
		if k != "limite" && k != "antes_de_version" {
			responder(w, http.StatusBadRequest, "ajustes_bolsa_entrada_invalida")
			return
		}
	}
	lectura, err := h.operador.Consultar(r.Context(), actor, limite, antes)
	if err != nil {
		responderError(w, err)
		return
	}
	reglasVista := make([]reglaVista, 0, len(lectura.Reglas))
	for _, regla := range lectura.Reglas {
		if regla.Edicion != nil || regla.AjusteNoAplicable {
			reglasVista = append(reglasVista, vistaRegla(regla))
		}
	}
	programados := lectura.Programados
	if programados == nil {
		programados = []app.VersionProgramada{}
	}
	historial := lectura.Historial
	if historial == nil {
		historial = []app.CambioHistorico{}
	}
	version := 0
	if lectura.Cabeza != nil {
		version = lectura.Cabeza.Version
	}
	escribir(w, http.StatusOK, map[string]any{"data": map[string]any{
		"esquema": Esquema, "catalogo_id": app.CatalogoAjustes, "activacion": map[string]string{"estado": "activa"},
		"version_esperada": version, "puede_ajustar": lectura.PuedeAjustar,
		"cabeza":      vistaVersion(lectura.Cabeza, lectura.CabezaPublicadaEn),
		"vigente_hoy": vistaVersion(lectura.VigenteHoy, lectura.VigentePublicadaEn),
		"programados": programados, "reglas": reglasVista, "motivos": h.operador.Motivos(),
		"historial": historial, "hay_mas": lectura.HayMas,
	}})
}

func (h *Handler) post(w http.ResponseWriter, r *http.Request, actor vecdomain.ContextoActor) {
	if r.URL.RawQuery != "" || r.URL.ForceQuery || r.Header.Get("Content-Encoding") != "" {
		responder(w, http.StatusBadRequest, "ajustes_bolsa_entrada_invalida")
		return
	}
	tipo, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || tipo != "application/json" {
		responder(w, http.StatusUnsupportedMediaType, "tipo_no_admitido")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	defer r.Body.Close()
	b, err := io.ReadAll(r.Body)
	var grande *http.MaxBytesError
	if errors.As(err, &grande) {
		responder(w, http.StatusRequestEntityTooLarge, "solicitud_demasiado_grande")
		return
	}
	if err != nil || len(b) == 0 || validarJSONSinDuplicados(b) != nil {
		responder(w, http.StatusBadRequest, "ajustes_bolsa_entrada_invalida")
		return
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	var solicitud app.Solicitud
	if d.Decode(&solicitud) != nil || d.Decode(new(any)) != io.EOF {
		responder(w, http.StatusBadRequest, "ajustes_bolsa_entrada_invalida")
		return
	}
	resultado, err := h.operador.Publicar(r.Context(), actor, solicitud)
	if err != nil {
		responderError(w, err)
		return
	}
	estado := http.StatusCreated
	if resultado.Replay {
		estado = http.StatusOK
	}
	escribir(w, estado, map[string]any{"data": map[string]any{"esquema": Esquema, "replay": resultado.Replay, "recibo": resultado.Recibo}})
}

func cabecerasProhibidas(h http.Header) bool {
	for nombre := range h {
		n := strings.ToLower(nombre)
		if n == "cookie" || n == "authorization" || n == "proxy-authorization" || n == "forwarded" || n == "remote-user" || n == "idempotency-key" || n == "x-http-method-override" || strings.HasPrefix(n, "x-forwarded-") || strings.HasPrefix(n, "x-auth-") || strings.HasPrefix(n, "x-vec-") || strings.Contains(n, "role") {
			return true
		}
	}
	return false
}
func origenPermitido(r *http.Request) bool {
	origen := r.Header.Get("Origin")
	if origen == "" || origen == "https://"+r.Host {
		return true
	}
	return origen == "http://"+r.Host && (r.Host == "localhost" || strings.HasPrefix(r.Host, "localhost:") || r.Host == "127.0.0.1" || strings.HasPrefix(r.Host, "127.0.0.1:"))
}

func vistaVersion(v *reglas.VersionAjustes, publicada time.Time) any {
	if v == nil {
		return nil
	}
	return map[string]any{"version": v.Version, "huella_sha256": v.HuellaSHA256, "ajustes": v.Ajustes, "vigente_desde": v.VigenteDesde, "publicada_en": publicada}
}

type reglaVista struct {
	Clave             string            `json:"clave"`
	Etiqueta          string            `json:"etiqueta"`
	Unidad            string            `json:"unidad,omitempty"`
	Cantidad          *int              `json:"cantidad,omitempty"`
	Computo           string            `json:"computo,omitempty"`
	Valores           map[string]string `json:"valores,omitempty"`
	Edicion           *edicionVista     `json:"edicion,omitempty"`
	AjusteNoAplicable bool              `json:"ajuste_no_aplicable"`
}

func vistaRegla(r reglas.Regla) reglaVista {
	v := reglaVista{Clave: r.Clave, Etiqueta: r.Etiqueta, AjusteNoAplicable: r.AjusteNoAplicable}
	if r.Edicion != nil {
		v.Edicion = &edicionVista{Campos: append([]string{}, r.Edicion.Campos...), OpcionesUnidad: append([]reglas.Unidad{}, r.Edicion.OpcionesUnidad...), OpcionesComputo: append([]reglas.Computo{}, r.Edicion.OpcionesComputo...), CantidadMinima: r.Edicion.CantidadMinima, CantidadMaxima: r.Edicion.CantidadMaxima}
	}
	if r.AjusteNoAplicable || r.Edicion == nil {
		return v
	}
	v.Unidad = string(r.Unidad)
	v.Computo = string(r.Computo)
	v.Cantidad = &r.Cantidad
	v.Valores = make(map[string]string, len(r.Edicion.Campos))
	for _, campo := range r.Edicion.Campos {
		switch campo {
		case reglas.CampoCantidad:
			v.Valores[campo] = strconv.Itoa(r.Cantidad)
		case reglas.CampoUnidad:
			v.Valores[campo] = string(r.Unidad)
		case reglas.CampoComputo:
			v.Valores[campo] = string(r.Computo)
		default:
			v.Valores[campo] = r.Atributos[campo]
		}
	}
	return v
}

type edicionVista struct {
	Campos          []string         `json:"campos"`
	OpcionesUnidad  []reglas.Unidad  `json:"opciones_unidad"`
	OpcionesComputo []reglas.Computo `json:"opciones_computo"`
	CantidadMinima  int              `json:"cantidad_minima"`
	CantidadMaxima  int              `json:"cantidad_maxima"`
}

func responderError(w http.ResponseWriter, err error) {
	estado, codigo := http.StatusServiceUnavailable, "ajustes_bolsa_no_disponibles"
	switch {
	case errors.Is(err, app.ErrNoAutenticado):
		estado, codigo = http.StatusUnauthorized, "ajustes_bolsa_no_autenticado"
	case errors.Is(err, app.ErrProhibido), errors.Is(err, vecdomain.ErrAutorizacionDenegada):
		estado, codigo = http.StatusForbidden, "ajustes_bolsa_prohibido"
	case errors.Is(err, app.ErrConflicto):
		estado, codigo = http.StatusConflict, "ajustes_bolsa_conflicto"
	case errors.Is(err, app.ErrEntradaInvalida):
		estado, codigo = http.StatusUnprocessableEntity, "ajustes_bolsa_entrada_invalida"
	}
	responder(w, estado, codigo)
}
func responder(w http.ResponseWriter, estado int, codigo string) {
	escribir(w, estado, map[string]any{"error": map[string]string{"codigo": codigo}})
}
func escribir(w http.ResponseWriter, estado int, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(estado)
	_, _ = w.Write(b)
}
