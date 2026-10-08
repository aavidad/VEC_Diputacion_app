// Package httpinscripcion expone el circuito propio y la bandeja RRHH de
// solicitudes. La identidad nunca se toma del cuerpo, URL ni cabeceras HTTP.
package httpinscripcion

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"vec-diputacion-granada/internal/modules/bolsa/application/inscripcion"
)

const (
	RutaAbiertas = "/api/vec/bolsa/inscripciones/convocatorias-abiertas"
	RutaPropias  = "/api/vec/bolsa/mi-bolsa/inscripciones"
	RutaRRHH     = "/api/vec/bolsa/rrhh/inscripciones"
	maximoCuerpo = 8192
)

type Preparador interface {
	PrepararLecturaAspirante(*http.Request, string, string, inscripcion.Filtro, string) (inscripcion.Actor, error)
	PrepararLecturaRRHH(*http.Request, string, string, inscripcion.Filtro, string) (inscripcion.Actor, error)
	PrepararPresentacion(*http.Request, inscripcion.Presentacion) (inscripcion.Actor, error)
	PrepararDecision(*http.Request, inscripcion.Decision) (inscripcion.Actor, error)
	PrepararIncorporacion(*http.Request, inscripcion.Incorporacion) (inscripcion.Actor, error)
}

type Aplicacion interface {
	Abiertas(context.Context, inscripcion.Actor, int, string) (inscripcion.PaginaAbiertas, error)
	DetalleAbierta(context.Context, inscripcion.Actor, string) (inscripcion.BolsaAbierta, error)
	Presentar(context.Context, inscripcion.Actor, inscripcion.Presentacion) (inscripcion.Recibo, error)
	Propias(context.Context, inscripcion.Actor, inscripcion.Filtro) (inscripcion.Pagina, error)
	Propia(context.Context, inscripcion.Actor, string) (inscripcion.Solicitud, error)
	PendientesRRHH(context.Context, inscripcion.Actor, inscripcion.Filtro) (inscripcion.Pagina, error)
	DetalleRRHH(context.Context, inscripcion.Actor, string) (inscripcion.Solicitud, error)
	MotivosRRHH(context.Context, inscripcion.Actor, string) (inscripcion.CatalogoMotivos, error)
	Decidir(context.Context, inscripcion.Actor, inscripcion.Decision) (inscripcion.Recibo, error)
	Incorporar(context.Context, inscripcion.Actor, inscripcion.Incorporacion) (inscripcion.Recibo, error)
}

type Handler struct {
	preparador Preparador
	servicio   Aplicacion
}

func Nuevo(preparador Preparador, servicio Aplicacion) (*Handler, error) {
	if preparador == nil || servicio == nil {
		return nil, inscripcion.ErrNoDisponible
	}
	return &Handler{preparador: preparador, servicio: servicio}, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.preparador == nil || h.servicio == nil || r == nil || r.URL == nil {
		responderError(w, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	if r.URL.RawPath != "" || r.URL.EscapedPath() != r.URL.Path ||
		strings.Contains(r.URL.Path, "//") || cabeceraProhibida(r.Header) ||
		(r.Header.Get("Accept") != "" && r.Header.Get("Accept") != "application/json") ||
		(r.Header.Get("Sec-Fetch-Site") != "" && r.Header.Get("Sec-Fetch-Site") != "same-origin") {
		responderError(w, http.StatusBadRequest, "peticion_no_permitida")
		return
	}
	switch {
	case r.URL.Path == RutaAbiertas:
		h.abiertas(w, r)
	case strings.HasPrefix(r.URL.Path, RutaAbiertas+"/"):
		h.detalleAbierta(w, r)
	case r.URL.Path == RutaPropias:
		h.propias(w, r)
	case strings.HasPrefix(r.URL.Path, RutaPropias+"/"):
		h.propia(w, r)
	case r.URL.Path == RutaRRHH:
		h.rrhh(w, r)
	case r.URL.Path == RutaRRHH+"/motivos":
		h.motivos(w, r)
	case strings.HasPrefix(r.URL.Path, RutaRRHH+"/"):
		h.detalleODecisionRRHH(w, r)
	default:
		responderError(w, http.StatusNotFound, "recurso_no_encontrado")
	}
}

func (h *Handler) abiertas(w http.ResponseWriter, r *http.Request) {
	if !soloGET(w, r) {
		return
	}
	limite, cursor, err := paginarCon(r.URL.Query(), "idioma")
	if err != nil {
		responderError(w, 400, "datos_no_validos")
		return
	}
	idiomaActivo, err := idioma(r.URL.Query())
	if err != nil {
		responderError(w, 400, "datos_no_validos")
		return
	}
	actor, err := h.preparador.PrepararLecturaAspirante(r, inscripcion.AccionListarAbiertas, "", inscripcion.Filtro{Limite: limite, Cursor: cursor}, idiomaActivo)
	if err != nil {
		responderFallo(w, err)
		return
	}
	p, err := h.servicio.Abiertas(r.Context(), actor, limite, cursor)
	if err != nil {
		responderFallo(w, err)
		return
	}
	responder(w, 200, "vec.bolsa.inscripciones.convocatorias_abiertas.v1", p)
}

func (h *Handler) detalleAbierta(w http.ResponseWriter, r *http.Request) {
	if !soloGET(w, r) {
		return
	}
	if !soloIdioma(r.URL.Query()) || r.URL.ForceQuery {
		responderError(w, 400, "datos_no_validos")
		return
	}
	ref, ok := segmento(RutaAbiertas, r.URL.Path)
	if !ok {
		responderError(w, 404, "recurso_no_encontrado")
		return
	}
	idiomaActivo, err := idioma(r.URL.Query())
	if err != nil {
		responderError(w, 400, "datos_no_validos")
		return
	}
	actor, err := h.preparador.PrepararLecturaAspirante(r, inscripcion.AccionDetalleAbierta, ref, inscripcion.Filtro{}, idiomaActivo)
	if err != nil {
		responderFallo(w, err)
		return
	}
	b, err := h.servicio.DetalleAbierta(r.Context(), actor, ref)
	if err != nil {
		responderFallo(w, err)
		return
	}
	responder(w, 200, "vec.bolsa.inscripcion.convocatoria.v1", map[string]any{"convocatoria": b})
}

func (h *Handler) propias(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if !sinCuerpo(r) {
			responderError(w, 400, "peticion_no_permitida")
			return
		}
		limite, cursor, err := paginarCon(r.URL.Query(), "idioma")
		if err != nil {
			responderError(w, 400, "datos_no_validos")
			return
		}
		idiomaActivo, err := idioma(r.URL.Query())
		if err != nil {
			responderError(w, 400, "datos_no_validos")
			return
		}
		filtro := inscripcion.Filtro{Limite: limite, Cursor: cursor}
		actor, err := h.preparador.PrepararLecturaAspirante(r, inscripcion.AccionListarPropias, "", filtro, idiomaActivo)
		if err != nil {
			responderFallo(w, err)
			return
		}
		p, err := h.servicio.Propias(r.Context(), actor, filtro)
		if err != nil {
			responderFallo(w, err)
			return
		}
		responder(w, 200, "vec.bolsa.inscripciones.propias.v1", p)
	case http.MethodPost:
		if r.URL.RawQuery != "" || r.URL.ForceQuery {
			responderError(w, 400, "datos_no_validos")
			return
		}
		var cuerpo struct {
			ConvocatoriaRef   string                    `json:"convocatoria_ref"`
			CategoriaRef      string                    `json:"categoria_ref"`
			CatalogoVersion   uint64                    `json:"catalogo_version"`
			ClaveIdempotencia string                    `json:"clave_idempotencia"`
			Declaraciones     []inscripcion.Declaracion `json:"declaraciones"`
		}
		if !decodificar(w, r, &cuerpo) {
			return
		}
		presentacion := inscripcion.Presentacion{
			ConvocatoriaRef: cuerpo.ConvocatoriaRef, CategoriaRef: cuerpo.CategoriaRef,
			CatalogoVersion:   cuerpo.CatalogoVersion,
			ClaveIdempotencia: cuerpo.ClaveIdempotencia, Declaraciones: cuerpo.Declaraciones,
		}
		actor, err := h.preparador.PrepararPresentacion(r, presentacion)
		if err != nil {
			responderFallo(w, err)
			return
		}
		recibo, err := h.servicio.Presentar(r.Context(), actor, presentacion)
		if err != nil {
			responderFallo(w, err)
			return
		}
		estado := http.StatusCreated
		if recibo.Repetida {
			estado = http.StatusOK
		}
		responder(w, estado, "vec.bolsa.inscripcion.recibo.v1", recibo)
	default:
		w.Header().Set("Allow", "GET, POST")
		responderError(w, 405, "metodo_no_permitido")
	}
}

func (h *Handler) propia(w http.ResponseWriter, r *http.Request) {
	if !soloGET(w, r) {
		return
	}
	if !soloIdioma(r.URL.Query()) || r.URL.ForceQuery {
		responderError(w, 400, "datos_no_validos")
		return
	}
	ref, ok := segmento(RutaPropias, r.URL.Path)
	if !ok {
		responderError(w, 404, "recurso_no_encontrado")
		return
	}
	idiomaActivo, err := idioma(r.URL.Query())
	if err != nil {
		responderError(w, 400, "datos_no_validos")
		return
	}
	actor, err := h.preparador.PrepararLecturaAspirante(r, inscripcion.AccionDetallePropia, ref, inscripcion.Filtro{}, idiomaActivo)
	if err != nil {
		responderFallo(w, err)
		return
	}
	s, err := h.servicio.Propia(r.Context(), actor, ref)
	if err != nil {
		responderFallo(w, err)
		return
	}
	responder(w, 200, "vec.bolsa.inscripcion.propias.detalle.v1", map[string]any{"solicitud": s})
}

func (h *Handler) rrhh(w http.ResponseWriter, r *http.Request) {
	if !soloGET(w, r) {
		return
	}
	limite, cursor, err := paginarCon(r.URL.Query(), "estado", "convocatoria_ref", "idioma")
	if err != nil {
		responderError(w, 400, "datos_no_validos")
		return
	}
	estado := r.URL.Query().Get("estado")
	if estado == "" {
		estado = inscripcion.EstadoPendiente
	}
	idiomaActivo, err := idioma(r.URL.Query())
	if err != nil {
		responderError(w, 400, "datos_no_validos")
		return
	}
	filtro := inscripcion.Filtro{Estado: estado, ConvocatoriaRef: r.URL.Query().Get("convocatoria_ref"), Limite: limite, Cursor: cursor}
	actor, err := h.preparador.PrepararLecturaRRHH(r, inscripcion.AccionListarRRHH, "", filtro, idiomaActivo)
	if err != nil {
		responderFallo(w, err)
		return
	}
	p, err := h.servicio.PendientesRRHH(r.Context(), actor, filtro)
	if err != nil {
		responderFallo(w, err)
		return
	}
	responder(w, 200, "vec.bolsa.inscripciones.rrhh.v1", p)
}

func (h *Handler) motivos(w http.ResponseWriter, r *http.Request) {
	if !soloGET(w, r) {
		return
	}
	query := r.URL.Query()
	if len(query["decision"]) != 1 || len(query) > 2 || !soloIdiomaYDecision(query) {
		responderError(w, 400, "datos_no_validos")
		return
	}
	idiomaActivo, err := idioma(query)
	if err != nil {
		responderError(w, 400, "datos_no_validos")
		return
	}
	actor, err := h.preparador.PrepararLecturaRRHH(r, inscripcion.AccionMotivosRRHH, query.Get("decision"), inscripcion.Filtro{}, idiomaActivo)
	if err != nil {
		responderFallo(w, err)
		return
	}
	m, err := h.servicio.MotivosRRHH(r.Context(), actor, query.Get("decision"))
	if err != nil {
		responderFallo(w, err)
		return
	}
	responder(w, 200, "vec.bolsa.inscripcion.motivos.v1", m)
}

func (h *Handler) detalleODecisionRRHH(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, RutaRRHH+"/")
	partes := strings.Split(path, "/")
	if len(partes) < 1 || len(partes) > 2 || !segmentoValido(partes[0]) ||
		(len(partes) == 2 && partes[1] != "decisiones" && partes[1] != "incorporaciones") {
		responderError(w, 404, "recurso_no_encontrado")
		return
	}
	if (len(partes) == 1 && !soloIdioma(r.URL.Query())) ||
		(len(partes) == 2 && r.URL.RawQuery != "") || r.URL.ForceQuery {
		responderError(w, 400, "datos_no_validos")
		return
	}
	if len(partes) == 1 {
		if !soloGET(w, r) {
			return
		}
		idiomaActivo, err := idioma(r.URL.Query())
		if err != nil {
			responderError(w, 400, "datos_no_validos")
			return
		}
		actor, err := h.preparador.PrepararLecturaRRHH(r, inscripcion.AccionDetalleRRHH, partes[0], inscripcion.Filtro{}, idiomaActivo)
		if err != nil {
			responderFallo(w, err)
			return
		}
		s, err := h.servicio.DetalleRRHH(r.Context(), actor, partes[0])
		if err != nil {
			responderFallo(w, err)
			return
		}
		responder(w, 200, "vec.bolsa.inscripcion.rrhh.v1", map[string]any{"solicitud": s})
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		responderError(w, 405, "metodo_no_permitido")
		return
	}
	if partes[1] == "incorporaciones" {
		var cuerpo struct {
			EvidenciaRef      string `json:"evidencia_ref"`
			VersionEsperada   uint64 `json:"version_esperada"`
			ClaveIdempotencia string `json:"clave_idempotencia"`
		}
		if !decodificar(w, r, &cuerpo) {
			return
		}
		incorporacion := inscripcion.Incorporacion{
			SolicitudRef: partes[0], EvidenciaRef: cuerpo.EvidenciaRef,
			VersionEsperada: cuerpo.VersionEsperada, ClaveIdempotencia: cuerpo.ClaveIdempotencia,
		}
		actor, err := h.preparador.PrepararIncorporacion(r, incorporacion)
		if err != nil {
			responderFallo(w, err)
			return
		}
		recibo, err := h.servicio.Incorporar(r.Context(), actor, incorporacion)
		if err != nil {
			responderFallo(w, err)
			return
		}
		estado := http.StatusCreated
		if recibo.Repetida {
			estado = http.StatusOK
		}
		responder(w, estado, "vec.bolsa.inscripcion.incorporacion.recibo.v1", recibo)
		return
	}
	var cuerpo struct {
		Decision          string `json:"decision"`
		MotivoCodigo      string `json:"motivo_codigo"`
		VersionEsperada   uint64 `json:"version_esperada"`
		ClaveIdempotencia string `json:"clave_idempotencia"`
	}
	if !decodificar(w, r, &cuerpo) {
		return
	}
	decision := inscripcion.Decision{
		SolicitudRef: partes[0], Tipo: cuerpo.Decision, MotivoCodigo: cuerpo.MotivoCodigo,
		VersionEsperada: cuerpo.VersionEsperada, ClaveIdempotencia: cuerpo.ClaveIdempotencia,
	}
	actor, err := h.preparador.PrepararDecision(r, decision)
	if err != nil {
		responderFallo(w, err)
		return
	}
	recibo, err := h.servicio.Decidir(r.Context(), actor, decision)
	if err != nil {
		responderFallo(w, err)
		return
	}
	estado := http.StatusCreated
	if recibo.Repetida {
		estado = http.StatusOK
	}
	responder(w, estado, "vec.bolsa.inscripcion.decision.recibo.v1", recibo)
}

func soloGET(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		responderError(w, 405, "metodo_no_permitido")
		return false
	}
	if !sinCuerpo(r) {
		responderError(w, 400, "peticion_no_permitida")
		return false
	}
	return true
}

func sinCuerpo(r *http.Request) bool {
	return r.ContentLength == 0 && len(r.TransferEncoding) == 0 &&
		(r.Body == nil || r.Body == http.NoBody || r.ProtoMajor >= 2)
}

func decodificar(w http.ResponseWriter, r *http.Request, destino any) bool {
	if r.Header.Get("Content-Type") != "application/json" || r.ContentLength < 2 ||
		r.ContentLength > maximoCuerpo || len(r.TransferEncoding) != 0 || r.Body == nil {
		responderError(w, 400, "peticion_no_permitida")
		return false
	}
	contenido, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maximoCuerpo))
	if err != nil || int64(len(contenido)) != r.ContentLength {
		responderError(w, 400, "peticion_no_permitida")
		return false
	}
	dec := json.NewDecoder(bytes.NewReader(contenido))
	dec.DisallowUnknownFields()
	if dec.Decode(destino) != nil || dec.Decode(new(any)) != io.EOF {
		responderError(w, 400, "datos_no_validos")
		return false
	}
	return true
}

func paginar(query url.Values) (int, string, error) { return paginarCon(query) }

func soloIdioma(query url.Values) bool {
	if len(query) == 0 {
		return true
	}
	return len(query) == 1 && len(query["idioma"]) == 1
}

func soloIdiomaYDecision(query url.Values) bool {
	for clave, valores := range query {
		if (clave != "idioma" && clave != "decision") || len(valores) != 1 {
			return false
		}
	}
	return true
}

func idioma(query url.Values) (string, error) {
	valor := query.Get("idioma")
	if valor == "" {
		return "es", nil
	}
	if valor != "es" && valor != "en" {
		return "", inscripcion.ErrSolicitudInvalida
	}
	return valor, nil
}

func paginarCon(query url.Values, adicionales ...string) (int, string, error) {
	admitidos := map[string]bool{"limite": true, "cursor": true}
	for _, clave := range adicionales {
		admitidos[clave] = true
	}
	for clave, valores := range query {
		if !admitidos[clave] || len(valores) != 1 {
			return 0, "", inscripcion.ErrSolicitudInvalida
		}
	}
	limite := 20
	if valor := query.Get("limite"); valor != "" {
		entero, err := strconv.Atoi(valor)
		if err != nil || entero < 1 || entero > 100 {
			return 0, "", inscripcion.ErrSolicitudInvalida
		}
		limite = entero
	}
	cursor := query.Get("cursor")
	if len(cursor) > 512 {
		return 0, "", inscripcion.ErrSolicitudInvalida
	}
	return limite, cursor, nil
}

func segmento(base, ruta string) (string, bool) {
	if !strings.HasPrefix(ruta, base+"/") {
		return "", false
	}
	valor := strings.TrimPrefix(ruta, base+"/")
	return valor, segmentoValido(valor)
}

func segmentoValido(valor string) bool {
	if len(valor) < 3 || len(valor) > 256 || strings.ContainsAny(valor, "/%\\\x00\r\n\t *") {
		return false
	}
	for _, c := range valor {
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == ':' || c == '.' || c == '_' || c == '-' {
			continue
		}
		return false
	}
	return true
}

func cabeceraProhibida(h http.Header) bool {
	for clave := range h {
		c := strings.ToLower(clave)
		if c == "cookie" || c == "proxy-authorization" || c == "remote-user" || c == "x-remote-user" ||
			strings.HasPrefix(c, "x-vec-") || strings.HasPrefix(c, "x-auth-") || strings.HasPrefix(c, "x-forwarded-") {
			return true
		}
	}
	return false
}

func responder(w http.ResponseWriter, estado int, esquema string, datos any) {
	campos, err := fusionarEsquema(esquema, datos)
	if err != nil {
		responderError(w, 503, "servicio_no_disponible")
		return
	}
	contenido, err := json.Marshal(map[string]any{"data": campos})
	if err != nil {
		responderError(w, 503, "servicio_no_disponible")
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(estado)
	if _, err := w.Write(append(contenido, '\n')); err != nil {
		slog.Warn("respuesta de inscripcion interrumpida", "codigo", "escritura_respuesta")
	}
}

func fusionarEsquema(esquema string, datos any) (map[string]any, error) {
	contenido, err := json.Marshal(datos)
	if err != nil {
		return nil, err
	}
	salida := map[string]any{"esquema": esquema}
	var campos map[string]any
	if err := json.Unmarshal(contenido, &campos); err != nil {
		return nil, err
	}
	for clave, valor := range campos {
		salida[clave] = valor
	}
	return salida, nil
}

func responderError(w http.ResponseWriter, estado int, codigo string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(estado)
	if err := json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"codigo": codigo}}); err != nil {
		slog.Warn("error de inscripcion sin respuesta", "codigo", "escritura_error")
	}
}

func responderFallo(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, inscripcion.ErrSolicitudInvalida):
		responderError(w, 400, "datos_no_validos")
	case errors.Is(err, inscripcion.ErrSesionAusente):
		responderError(w, 401, "sesion_ausente")
	case errors.Is(err, inscripcion.ErrAccesoDenegado):
		responderError(w, 403, "acceso_denegado")
	case errors.Is(err, inscripcion.ErrNoEncontrada):
		responderError(w, 404, "recurso_no_encontrado")
	case errors.Is(err, inscripcion.ErrConflicto):
		responderError(w, 409, "conflicto")
	case errors.Is(err, inscripcion.ErrPlazoCerrado):
		responderError(w, 422, "plazo_cerrado")
	case errors.Is(err, inscripcion.ErrCatalogoCambiado):
		responderError(w, 422, "catalogo_cambiado")
	case errors.Is(err, inscripcion.ErrRequisitoInvalido):
		responderError(w, 422, "requisito_invalido")
	case errors.Is(err, inscripcion.ErrDeclaracionInvalida):
		responderError(w, 422, "declaracion_invalida")
	case errors.Is(err, inscripcion.ErrSolicitudExistente):
		responderError(w, 409, "solicitud_existente")
	case errors.Is(err, inscripcion.ErrClaveConflicto):
		responderError(w, 409, "clave_en_conflicto")
	case errors.Is(err, inscripcion.ErrActaNoDisponible):
		responderError(w, 409, "acta_pendiente")
	default:
		responderError(w, 503, "servicio_no_disponible")
	}
}
