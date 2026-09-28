package httpinterno

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
	puertos "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const RutaCatalogoCausasParticipacion = "/api/vec/bolsa/causas-participacion"
const RutaPropuestasCausasParticipacion = RutaCatalogoCausasParticipacion + "/propuestas"

var codigoCausaHTTP = regexp.MustCompile(`^[a-z][a-z0-9_]{2,63}$`)
var referenciaPropuestaCausaHTTP = regexp.MustCompile(`^propuesta:causa:[a-f0-9]{64}$`)

type EntradaPublicarCatalogoCausaParticipacion struct {
	Codigo          string `json:"codigo"`
	Version         int64  `json:"version"`
	Etiqueta        string `json:"etiqueta"`
	AplicaSituacion *bool  `json:"aplica_situacion"`
	AplicaContacto  *bool  `json:"aplica_contacto"`
	Publicable      *bool  `json:"publicable"`
	Activa          *bool  `json:"activa"`
}
type PreparadorCatalogoCausasParticipacion interface {
	PrepararPropuesta(context.Context, EntradaPublicarCatalogoCausaParticipacion) (puertos.SolicitudProponerCausaParticipacion, error)
	PrepararPublicacion(context.Context, string, EntradaPublicarCatalogoCausaParticipacion) (puertos.SolicitudPublicarCausaParticipacion, error)
	PrepararConsulta(context.Context) (puertos.SolicitudConsultarCausasParticipacion, error)
	PrepararConsultaPropuesta(context.Context, string) (puertos.SolicitudConsultarPropuestaCausaParticipacion, error)
}
type OperadorCatalogoCausasParticipacion interface {
	Proponer(context.Context, puertos.SolicitudProponerCausaParticipacion) (puertos.PropuestaCausaParticipacion, string, error)
	Publicar(context.Context, puertos.SolicitudPublicarCausaParticipacion) (puertos.CausaParticipacionCatalogada, string, error)
	Consultar(context.Context, puertos.SolicitudConsultarCausasParticipacion) ([]puertos.CausaParticipacionCatalogada, error)
	ConsultarPropuesta(context.Context, puertos.SolicitudConsultarPropuestaCausaParticipacion) (puertos.PropuestaCausaParticipacionLeida, error)
}
type HandlerCatalogoCausasParticipacion struct {
	preparador PreparadorCatalogoCausasParticipacion
	operador   OperadorCatalogoCausasParticipacion
}

func NuevoHandlerCatalogoCausasParticipacion(p PreparadorCatalogoCausasParticipacion, o OperadorCatalogoCausasParticipacion) (http.Handler, error) {
	if p == nil || o == nil {
		return nil, errors.New("bolsa http interno: catalogo no disponible")
	}
	return &HandlerCatalogoCausasParticipacion{p, o}, nil
}
func (h *HandlerCatalogoCausasParticipacion) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL == nil || r.URL.RawPath != "" || r.URL.RawQuery != "" || r.RequestURI != r.URL.Path || strings.Contains(r.URL.EscapedPath(), "%") {
		respuestaCatalogo(w, 404, map[string]any{"error": map[string]string{"codigo": "recurso_no_encontrado"}})
		return
	}
	if r.URL.Path == RutaCatalogoCausasParticipacion {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			respuestaCatalogo(w, 405, map[string]any{"error": map[string]string{"codigo": "metodo_no_permitido"}})
			return
		}
		h.get(w, r)
		return
	}
	if r.URL.Path == RutaPropuestasCausasParticipacion && r.Method == http.MethodPost {
		h.propuesta(w, r)
		return
	}
	pref := RutaPropuestasCausasParticipacion + "/"
	if strings.HasPrefix(r.URL.Path, pref) && strings.HasSuffix(r.URL.Path, "/publicar") && r.Method == http.MethodPost {
		ref := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, pref), "/publicar")
		if referenciaPropuestaCausaHTTP.MatchString(ref) {
			h.publicar(w, r, ref)
			return
		}
	}
	if strings.HasPrefix(r.URL.Path, pref) && r.Method == http.MethodGet {
		ref := strings.TrimPrefix(r.URL.Path, pref)
		if referenciaPropuestaCausaHTTP.MatchString(ref) {
			h.consultarPropuesta(w, r, ref)
			return
		}
	}
	respuestaCatalogo(w, 404, map[string]any{"error": map[string]string{"codigo": "recurso_no_encontrado"}})
}
func (h *HandlerCatalogoCausasParticipacion) consultarPropuesta(w http.ResponseWriter, r *http.Request, ref string) {
	if r.Body != nil && r.Body != http.NoBody || len(r.TransferEncoding) != 0 || r.Header.Get("Accept") != "application/json" {
		invalidoCatalogo(w)
		return
	}
	q, e := h.preparador.PrepararConsultaPropuesta(r.Context(), ref)
	if e == nil {
		x, e := h.operador.ConsultarPropuesta(r.Context(), q)
		if e == nil {
			respuestaCatalogo(w, 200, map[string]any{"data": x})
			return
		}
	}
	errorCatalogo(w, e)
}
func (h *HandlerCatalogoCausasParticipacion) get(w http.ResponseWriter, r *http.Request) {
	if r.Body != nil && r.Body != http.NoBody || len(r.TransferEncoding) != 0 || r.Header.Get("Accept") != "application/json" {
		invalidoCatalogo(w)
		return
	}
	q, e := h.preparador.PrepararConsulta(r.Context())
	if e == nil {
		x, e := h.operador.Consultar(r.Context(), q)
		if e == nil {
			type salida struct {
				Codigo          string `json:"codigo"`
				Version         int64  `json:"version"`
				HuellaSHA256    string `json:"huella_sha256"`
				Etiqueta        string `json:"etiqueta"`
				AplicaSituacion bool   `json:"aplica_situacion"`
				AplicaContacto  bool   `json:"aplica_contacto"`
			}
			y := make([]salida, 0, len(x))
			for _, c := range x {
				y = append(y, salida{c.Codigo, c.Version, c.HuellaSHA256, c.Etiqueta, c.AplicaSituacion, c.AplicaContacto})
			}
			respuestaCatalogo(w, 200, map[string]any{"data": y})
			return
		}
	}
	errorCatalogo(w, e)
}
func entradaCatalogo(r *http.Request) (EntradaPublicarCatalogoCausaParticipacion, bool) {
	if r.Body == nil || r.Body == http.NoBody || r.ContentLength <= 0 || r.ContentLength > 4096 || len(r.TransferEncoding) != 0 || r.Header.Get("Content-Type") != "application/json" || r.Header.Get("Accept") != "application/json" {
		return EntradaPublicarCatalogoCausaParticipacion{}, false
	}
	var x EntradaPublicarCatalogoCausaParticipacion
	d := json.NewDecoder(io.LimitReader(r.Body, 4097))
	d.DisallowUnknownFields()
	if d.Decode(&x) != nil || d.Decode(&struct{}{}) != io.EOF || !codigoCausaHTTP.MatchString(x.Codigo) || x.Version < 1 || !etiquetaCausaHTTPValida(x.Etiqueta) || x.AplicaSituacion == nil || x.AplicaContacto == nil || x.Publicable == nil || x.Activa == nil || (!*x.AplicaSituacion && !*x.AplicaContacto) {
		return x, false
	}
	return x, true
}

func etiquetaCausaHTTPValida(v string) bool {
	if len(v) < 1 || len(v) > 120 || v != strings.TrimSpace(v) || !utf8.ValidString(v) {
		return false
	}
	for _, r := range v {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return false
		}
	}
	return true
}
func (h *HandlerCatalogoCausasParticipacion) propuesta(w http.ResponseWriter, r *http.Request) {
	x, ok := entradaCatalogo(r)
	if !ok {
		invalidoCatalogo(w)
		return
	}
	q, e := h.preparador.PrepararPropuesta(r.Context(), x)
	if e == nil {
		p, rec, e := h.operador.Proponer(r.Context(), q)
		if e == nil {
			respuestaCatalogo(w, 201, map[string]any{"data": map[string]any{"propuesta": p, "recibo": rec}})
			return
		}
		errorCatalogo(w, e)
		return
	}
	errorCatalogo(w, e)
}
func (h *HandlerCatalogoCausasParticipacion) publicar(w http.ResponseWriter, r *http.Request, ref string) {
	x, ok := entradaCatalogo(r)
	if !ok {
		invalidoCatalogo(w)
		return
	}
	q, e := h.preparador.PrepararPublicacion(r.Context(), ref, x)
	if e == nil {
		c, rec, e := h.operador.Publicar(r.Context(), q)
		if e == nil {
			respuestaCatalogo(w, 201, map[string]any{"data": map[string]any{"catalogo": c, "recibo": rec}})
			return
		}
		errorCatalogo(w, e)
		return
	}
	errorCatalogo(w, e)
}
func invalidoCatalogo(w http.ResponseWriter) {
	respuestaCatalogo(w, 400, map[string]any{"error": map[string]string{"codigo": "solicitud_invalida"}})
}
func errorCatalogo(w http.ResponseWriter, e error) {
	if errors.Is(e, dominiovec.ErrAutorizacionDenegada) || errors.Is(e, dominiovec.ErrPermissionDenied) {
		respuestaCatalogo(w, 403, map[string]any{"error": map[string]string{"codigo": "acceso_denegado"}})
		return
	}
	if errors.Is(e, puertos.ErrCatalogoCausasParticipacionEnConflicto) {
		respuestaCatalogo(w, 409, map[string]any{"error": map[string]string{"codigo": "catalogo_en_conflicto"}})
		return
	}
	if errors.Is(e, puertos.ErrPropuestaCausaParticipacionNoEncontrada) {
		respuestaCatalogo(w, 404, map[string]any{"error": map[string]string{"codigo": "recurso_no_encontrado"}})
		return
	}
	respuestaCatalogo(w, 503, map[string]any{"error": map[string]string{"codigo": "servicio_no_disponible"}})
}
func respuestaCatalogo(w http.ResponseWriter, s int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(s)
	_ = json.NewEncoder(w).Encode(v)
}
