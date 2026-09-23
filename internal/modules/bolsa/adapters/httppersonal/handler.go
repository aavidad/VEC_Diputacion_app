// Package httppersonal transporta la consulta propia sin aceptar identidad,
// perfil, candidato ni selector desde HTTP.
package httppersonal

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"vec-diputacion-granada/internal/modules/bolsa/application/mibolsa"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const RutaMiBolsa = "/api/vec/bolsa/mi-bolsa"

var ErrAutenticacionAusente = errors.New("bolsa mi bolsa: autenticacion ausente")
var ErrDependenciaNoDisponible = errors.New("bolsa mi bolsa: dependencia no disponible")

type Preparador interface {
	PrepararMiBolsa(*http.Request) (mibolsa.Orden, error)
}
type Consultor interface {
	Consultar(context.Context, mibolsa.Orden) (puertosbolsa.InstantaneaMiBolsa, error)
}
type Handler struct {
	preparador Preparador
	consultor  Consultor
}

func Nuevo(preparador Preparador, consultor Consultor) (http.Handler, error) {
	if nula(preparador) || nula(consultor) {
		return nil, ErrDependenciaNoDisponible
	}
	return &Handler{preparador, consultor}, nil
}
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || r == nil || nula(h.preparador) || nula(h.consultor) {
		responder(w, 503, errorRespuesta{"servicio_no_disponible"})
		return
	}
	if r.URL == nil || r.URL.Path != RutaMiBolsa || r.URL.RawPath != "" || r.URL.EscapedPath() != RutaMiBolsa {
		responder(w, 404, errorRespuesta{"recurso_no_encontrado"})
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		responder(w, 405, errorRespuesta{"metodo_no_permitido"})
		return
	}
	if r.URL.RawQuery != "" || r.URL.ForceQuery || r.ContentLength != 0 || len(r.TransferEncoding) != 0 || (r.Body != nil && r.Body != http.NoBody) || cabeceraProhibida(r.Header) {
		responder(w, 400, errorRespuesta{"peticion_no_permitida"})
		return
	}
	orden, err := h.preparador.PrepararMiBolsa(r)
	if err != nil {
		responderError(w, err)
		return
	}
	resultado, err := h.consultor.Consultar(r.Context(), orden)
	if err != nil {
		responderError(w, err)
		return
	}
	responder(w, 200, nuevaRespuesta(resultado))
}
func cabeceraProhibida(h http.Header) bool {
	for k := range h {
		l := strings.ToLower(k)
		if l == "cookie" || l == "proxy-authorization" || strings.HasPrefix(l, "x-vec-") || strings.HasPrefix(l, "x-auth-") || strings.HasPrefix(l, "x-forwarded-") || l == "x-remote-user" || l == "remote-user" {
			return true
		}
	}
	return false
}

type errorRespuesta struct {
	Codigo string `json:"codigo"`
}
type respuestaError struct {
	Error errorRespuesta `json:"error"`
}
type participacion struct {
	Bolsa            string           `json:"bolsa"`
	Categoria        string           `json:"categoria"`
	Version          uint64           `json:"version"`
	OrdenInicial     uint64           `json:"orden_inicial"`
	TotalInstantanea uint64           `json:"total_instantanea"`
	EstadoBolsa      string           `json:"estado_bolsa"`
	VigenteDesde     string           `json:"vigente_desde"`
	VigenteHasta     *string          `json:"vigente_hasta"`
	SituacionActual  *situacionActual `json:"situacion_actual"`
}
type situacionActual struct {
	Estado          string  `json:"estado"`
	Desde           string  `json:"desde"`
	Hasta           *string `json:"hasta"`
	FechaDisponible *string `json:"fecha_disponible"`
}
type respuesta struct {
	Data struct {
		Esquema         string          `json:"esquema"`
		AvisoDesarrollo string          `json:"aviso_desarrollo"`
		ConsultadaEn    string          `json:"consultada_en"`
		Participaciones []participacion `json:"participaciones"`
	} `json:"data"`
}

func nuevaRespuesta(i puertosbolsa.InstantaneaMiBolsa) respuesta {
	var r respuesta
	r.Data.Esquema = puertosbolsa.EsquemaMiBolsaV1
	r.Data.AvisoDesarrollo = "Acceso de desarrollo con certificado sintético. Cl@ve, certificado FNMT y DNIe dependen de la pasarela de Sistemas."
	r.Data.ConsultadaEn = i.ConsultadaEn.Format("2006-01-02T15:04:05.000000Z07:00")
	r.Data.Participaciones = make([]participacion, 0, len(i.Participaciones))
	for _, p := range i.Participaciones {
		x := participacion{Bolsa: p.Bolsa, Categoria: p.Categoria, Version: p.Version, OrdenInicial: p.OrdenInicial, TotalInstantanea: p.TotalInstantanea, EstadoBolsa: p.EstadoBolsa, VigenteDesde: p.VigenteDesde.Format("2006-01-02T15:04:05.000000Z07:00")}
		if p.VigenteHasta != nil {
			v := p.VigenteHasta.Format("2006-01-02T15:04:05.000000Z07:00")
			x.VigenteHasta = &v
		}
		if p.SituacionActual != nil {
			s := p.SituacionActual
			x.SituacionActual = &situacionActual{Estado: s.Estado, Desde: s.Desde.Format("2006-01-02T15:04:05.000000Z07:00")}
			if s.Hasta != nil {
				hasta := s.Hasta.Format("2006-01-02T15:04:05.000000Z07:00")
				x.SituacionActual.Hasta = &hasta
			}
			if s.FechaDisponible != nil {
				fecha := s.FechaDisponible.Format("2006-01-02T15:04:05.000000Z07:00")
				x.SituacionActual.FechaDisponible = &fecha
			}
		}
		r.Data.Participaciones = append(r.Data.Participaciones, x)
	}
	return r
}
func responderError(w http.ResponseWriter, e error) {
	switch {
	case esIndisponibilidad(e):
		responder(w, 503, errorRespuesta{"servicio_no_disponible"})
	case errors.Is(e, ErrAutenticacionAusente):
		responder(w, 401, errorRespuesta{"autenticacion_requerida"})
	case errors.Is(e, dominiovec.ErrAutorizacionDenegada), errors.Is(e, dominiovec.ErrPermissionDenied):
		responder(w, 403, errorRespuesta{"acceso_denegado"})
	default:
		responder(w, 500, errorRespuesta{"error_interno"})
	}
}
func esIndisponibilidad(e error) bool {
	return errors.Is(e, puertosbolsa.ErrMaterialMiBolsaNoDisponible) || errors.Is(e, puertosvec.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible) || errors.Is(e, puertosvec.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible) || errors.Is(e, ErrDependenciaNoDisponible) || errors.Is(e, puertosvec.ErrFuenteContextoActorNoDisponible) || errors.Is(e, puertosvec.ErrRevalidacionAutenticacionActorNoDisponible) || errors.Is(e, puertosvec.ErrFuenteAutorizacionNoDisponible) || errors.Is(e, puertosvec.ErrRegistroDecisionNoDisponible) || errors.Is(e, puertosvec.ErrRegistroDenegacionNoDisponible)
}
func responder(w http.ResponseWriter, status int, v any) {
	if detalle, esError := v.(errorRespuesta); esError {
		v = respuestaError{Error: detalle}
	}
	b, e := json.Marshal(v)
	if e != nil {
		status = 500
		b = []byte(`{"error":{"codigo":"error_interno"}}`)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Length", strconv.Itoa(len(b)))
	w.WriteHeader(status)
	_, _ = w.Write(b)
}
func nula(v any) bool {
	if v == nil {
		return true
	}
	x := reflect.ValueOf(v)
	switch x.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return x.IsNil()
	}
	return false
}
