package httpinterno

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type EntradaRegistrarContactoParticipacion struct {
	BolsaRef, ParticipacionRef, LlamamientoRef, Canal, Resultado, Anotacion, ClaveIdempotencia string
	Instante                                                                                   time.Time
}
type PreparadorContactoParticipacion interface {
	PrepararSolicitudRegistrarContacto(context.Context, EntradaRegistrarContactoParticipacion) (puertosbolsa.SolicitudRegistrarContactoParticipacion, error)
}
type OperadorContactoParticipacion interface {
	RegistrarContactoParticipacion(context.Context, puertosbolsa.SolicitudRegistrarContactoParticipacion) (puertosbolsa.RegistroContactoParticipacion, error)
	ListarContactosParticipacion(context.Context, puertosbolsa.ConsultaContactosParticipacion) (puertosbolsa.PaginaContactosParticipacion, error)
	ListarContactosBolsa(context.Context, string, string, int) (puertosbolsa.PaginaContactosParticipacion, error)
}
type HandlerContactoParticipacion struct {
	preparador PreparadorContactoParticipacion
	operador   OperadorContactoParticipacion
}

func NuevoHandlerContactoParticipacion(p PreparadorContactoParticipacion, o OperadorContactoParticipacion) (http.Handler, error) {
	if p == nil || o == nil {
		return nil, errors.New("bolsa http interno: contactos no disponibles")
	}
	return &HandlerContactoParticipacion{p, o}, nil
}
func (h *HandlerContactoParticipacion) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	bolsa, participacion, ok := ReferenciasRutaContactosParticipacion(r)
	if !ok {
		responderContacto(w, 404, map[string]any{"error": map[string]string{"codigo": "recurso_no_encontrado"}})
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.listar(w, r, bolsa, participacion)
	case http.MethodPost:
		h.registrar(w, r, bolsa, participacion)
	default:
		w.Header().Set("Allow", "GET, POST")
		responderContacto(w, 405, map[string]any{"error": map[string]string{"codigo": "metodo_no_permitido"}})
	}
}
func (h *HandlerContactoParticipacion) registrar(w http.ResponseWriter, r *http.Request, bolsa, participacion string) {
	if r.URL.RawQuery != "" || r.Body == nil || r.Body == http.NoBody || r.ContentLength <= 0 || r.ContentLength > 4096 || len(r.TransferEncoding) != 0 || r.Header.Get("Content-Type") != "application/json" || r.Header.Get("Accept") != "application/json" {
		responderContacto(w, 400, map[string]any{"error": map[string]string{"codigo": "solicitud_invalida"}})
		return
	}
	clave := r.Header.Get("Idempotency-Key")
	if len(r.Header.Values("Idempotency-Key")) != 1 || clave == "" || strings.TrimSpace(clave) != clave || len(clave) > 256 {
		responderContacto(w, 400, map[string]any{"error": map[string]string{"codigo": "solicitud_invalida"}})
		return
	}
	var c struct {
		Canal          string    `json:"canal"`
		Instante       time.Time `json:"instante"`
		Resultado      string    `json:"resultado"`
		Anotacion      string    `json:"anotacion"`
		LlamamientoRef string    `json:"llamamiento_ref"`
	}
	d := json.NewDecoder(io.LimitReader(r.Body, 4097))
	d.DisallowUnknownFields()
	if d.Decode(&c) != nil || d.Decode(&struct{}{}) != io.EOF {
		responderContacto(w, 400, map[string]any{"error": map[string]string{"codigo": "solicitud_invalida"}})
		return
	}
	s, err := h.preparador.PrepararSolicitudRegistrarContacto(r.Context(), EntradaRegistrarContactoParticipacion{bolsa, participacion, c.LlamamientoRef, c.Canal, c.Resultado, c.Anotacion, clave, c.Instante})
	if err != nil {
		responderErrorContacto(w, err)
		return
	}
	out, err := h.operador.RegistrarContactoParticipacion(r.Context(), s)
	if err != nil {
		responderErrorContacto(w, err)
		return
	}
	estado := http.StatusCreated
	if out.Reutilizado {
		estado = http.StatusOK
	}
	responderContacto(w, estado, map[string]any{"data": salidaContacto(out.Contacto, out.ReciboRef, out.Reutilizado)})
}
func (h *HandlerContactoParticipacion) listar(w http.ResponseWriter, r *http.Request, bolsa, participacion string) {
	if r.ContentLength != 0 || r.Header.Get("Accept") != "application/json" {
		responderContacto(w, 400, map[string]any{"error": map[string]string{"codigo": "solicitud_invalida"}})
		return
	}
	v, e := url.ParseQuery(r.URL.RawQuery)
	if e != nil || len(v) > 2 {
		responderContacto(w, 400, map[string]any{"error": map[string]string{"codigo": "solicitud_invalida"}})
		return
	}
	cursor := v.Get("cursor")
	limite := 20
	if x := v.Get("limite"); x != "" {
		limite, e = strconv.Atoi(x)
	}
	if e != nil || limite < 1 || limite > 100 {
		responderContacto(w, 400, map[string]any{"error": map[string]string{"codigo": "solicitud_invalida"}})
		return
	}
	p, e := h.operador.ListarContactosParticipacion(r.Context(), puertosbolsa.ConsultaContactosParticipacion{BolsaRef: bolsa, ParticipacionRef: participacion, Cursor: cursor, Limite: limite})
	if e != nil {
		responderErrorContacto(w, e)
		return
	}
	items := make([]map[string]any, 0, len(p.Contactos))
	for _, c := range p.Contactos {
		items = append(items, map[string]any{"contacto_ref": c.ContactoRef, "participacion_ref": c.ParticipacionRef, "llamamiento_ref": valorNulo(c.LlamamientoRef), "canal": c.Canal, "instante": c.Instante.UTC().Format(time.RFC3339Nano), "actor_ref": c.Actor, "resultado": c.Resultado, "anotacion": c.Anotacion})
	}
	responderContacto(w, 200, map[string]any{"data": map[string]any{"esquema": "vec.bolsa.rrhh.contactos.v1", "contactos": items, "cursor_siguiente": valorNulo(p.CursorSiguiente), "hay_mas": len(p.Contactos) == limite}})
}
func ReferenciasRutaContactosParticipacion(r *http.Request) (string, string, bool) {
	if r == nil || r.URL == nil || r.URL.RawPath != "" || strings.Contains(r.URL.EscapedPath(), "%") {
		return "", "", false
	}
	s := strings.Split(strings.TrimPrefix(r.URL.Path, RutaBolsasGestion+"/"), "/")
	if len(s) != 4 || s[0] == "" || s[1] != "candidatos" || s[2] == "" || s[3] != "contactos" {
		return "", "", false
	}
	return s[0], s[2], true
}
func salidaContacto(c dominiobolsa.ContactoParticipacion, recibo string, reutilizado bool) map[string]any {
	return map[string]any{"contacto_ref": c.ContactoRef, "participacion_ref": c.ParticipacionRef, "llamamiento_ref": valorNulo(c.LlamamientoRef), "canal": c.Canal, "instante": c.Instante.UTC().Format(time.RFC3339Nano), "actor_ref": c.Actor, "resultado": c.Resultado, "anotacion": c.Anotacion, "recibo_ref": valorNulo(recibo), "reutilizado": reutilizado}
}
func valorNulo(v string) any {
	if v == "" {
		return nil
	}
	return v
}
func responderErrorContacto(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, dominiovec.ErrAutorizacionDenegada), errors.Is(err, dominiovec.ErrPermissionDenied):
		responderContacto(w, 403, map[string]any{"error": map[string]string{"codigo": "acceso_denegado"}})
	case errors.Is(err, dominiobolsa.ErrContactoParticipacionInvalido):
		responderContacto(w, 409, map[string]any{"error": map[string]string{"codigo": "contacto_en_conflicto"}})
	case errors.Is(err, puertosbolsa.ErrContactoParticipacionNoEncontrado):
		responderContacto(w, 404, map[string]any{"error": map[string]string{"codigo": "recurso_no_encontrado"}})
	default:
		responderContacto(w, 503, map[string]any{"error": map[string]string{"codigo": "servicio_no_disponible"}})
	}
}
func responderContacto(w http.ResponseWriter, estado int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(estado)
	_ = json.NewEncoder(w).Encode(v)
}
