package httpinterno

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const RutaGobiernoTransicionesSituacion = "/api/vec/bolsa/politica-transiciones"
const maximoCuerpoGobiernoTransiciones = 8192

type EntradaGobiernoTransicionesSituacion struct {
	VersionEsperada   int64
	Transiciones      []string
	Motivo            string
	ClaveIdempotencia string
}

// El preparador toma identidad, motivo catalogado y ámbito de la sesión
// certificada; ninguna cabecera o propiedad del cuerpo los sustituye.
type PreparadorGobiernoTransicionesSituacion interface {
	PrepararConsultaGobiernoTransiciones(context.Context) (bool, error)
	AuditarConsultaGobiernoTransiciones(context.Context, bool) error
	PrepararPublicacionGobiernoTransiciones(context.Context, EntradaGobiernoTransicionesSituacion) (ports.ComandoGobiernoTransicionesSituacion, error)
}

type OperadorGobiernoTransicionesSituacion interface {
	Consultar(context.Context) (ports.GobiernoTransicionesSituacion, error)
	Publicar(context.Context, ports.ComandoGobiernoTransicionesSituacion) (ports.GobiernoTransicionesSituacion, error)
}

type HandlerGobiernoTransicionesSituacion struct {
	preparador PreparadorGobiernoTransicionesSituacion
	operador   OperadorGobiernoTransicionesSituacion
}

func NuevoHandlerGobiernoTransicionesSituacion(p PreparadorGobiernoTransicionesSituacion,
	o OperadorGobiernoTransicionesSituacion) (http.Handler, error) {
	if p == nil || o == nil {
		return nil, ports.ErrGobiernoTransicionesNoDisponible
	}
	return &HandlerGobiernoTransicionesSituacion{preparador: p, operador: o}, nil
}

func (h *HandlerGobiernoTransicionesSituacion) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.preparador == nil || h.operador == nil || r == nil || r.URL == nil {
		responderOferta(w, http.StatusServiceUnavailable, "servicio_no_disponible", nil)
		return
	}
	if r.URL.Path != RutaGobiernoTransicionesSituacion || r.URL.RawPath != "" ||
		r.URL.EscapedPath() != RutaGobiernoTransicionesSituacion {
		responderOferta(w, http.StatusNotFound, "no_encontrado", nil)
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.consultar(w, r)
	case http.MethodPost:
		h.publicar(w, r)
	default:
		w.Header().Set("Allow", "GET, POST")
		responderOferta(w, http.StatusMethodNotAllowed, "metodo_no_permitido", nil)
	}
}

func (h *HandlerGobiernoTransicionesSituacion) consultar(w http.ResponseWriter, r *http.Request) {
	if r.URL.RawQuery != "" || r.URL.ForceQuery || r.ContentLength != 0 || len(r.TransferEncoding) != 0 ||
		cabeceraPresente(r.Header, "Cookie") || cabeceraPresente(r.Header, "Authorization") ||
		cabeceraPresente(r.Header, "Proxy-Authorization") || cabeceraIdentidadHeredadaPresente(r.Header) {
		responderOferta(w, http.StatusBadRequest, "solicitud_invalida", nil)
		return
	}
	puede, err := h.preparador.PrepararConsultaGobiernoTransiciones(r.Context())
	if err != nil {
		responderErrorGobiernoTransiciones(w, err)
		return
	}
	v, err := h.operador.Consultar(r.Context())
	if h.preparador.AuditarConsultaGobiernoTransiciones(r.Context(), err == nil) != nil {
		responderOferta(w, http.StatusServiceUnavailable, "servicio_no_disponible", nil)
		return
	}
	if err != nil {
		responderErrorGobiernoTransiciones(w, err)
		return
	}
	v.PuedePublicar = puede
	responderOferta(w, http.StatusOK, "", v)
}

func (h *HandlerGobiernoTransicionesSituacion) publicar(w http.ResponseWriter, r *http.Request) {
	if r.URL.RawQuery != "" || r.URL.ForceQuery || r.Body == nil || r.Body == http.NoBody ||
		r.ContentLength < 1 || r.ContentLength > maximoCuerpoGobiernoTransiciones || len(r.TransferEncoding) != 0 ||
		len(r.Header.Values("Content-Type")) != 1 || r.Header.Get("Content-Type") != "application/json" ||
		len(r.Header.Values("Accept")) != 1 || r.Header.Get("Accept") != "application/json" ||
		cabeceraPresente(r.Header, "Cookie") || cabeceraPresente(r.Header, "Authorization") ||
		cabeceraPresente(r.Header, "Proxy-Authorization") || cabeceraIdentidadHeredadaPresente(r.Header) ||
		cabeceraPresente(r.Header, "Idempotency-Key") {
		responderOferta(w, http.StatusBadRequest, "solicitud_invalida", nil)
		return
	}
	var cuerpo struct {
		VersionEsperada   *int64   `json:"version_esperada"`
		Transiciones      []string `json:"transiciones"`
		Motivo            string   `json:"motivo"`
		ClaveIdempotencia string   `json:"clave_idempotencia"`
	}
	if !decodificarCuerpoOferta(r, &cuerpo) || cuerpo.VersionEsperada == nil || *cuerpo.VersionEsperada < 1 ||
		len(cuerpo.Transiciones) == 0 || len(cuerpo.Transiciones) > 56 ||
		cuerpo.Motivo == "" || len(cuerpo.Motivo) > 1000 || strings.TrimSpace(cuerpo.Motivo) != cuerpo.Motivo ||
		len(cuerpo.ClaveIdempotencia) < 8 || len(cuerpo.ClaveIdempotencia) > 256 {
		responderOferta(w, http.StatusUnprocessableEntity, "politica_invalida", nil)
		return
	}
	entrada := EntradaGobiernoTransicionesSituacion{VersionEsperada: *cuerpo.VersionEsperada,
		Transiciones: cuerpo.Transiciones, Motivo: cuerpo.Motivo, ClaveIdempotencia: cuerpo.ClaveIdempotencia}
	comando, err := h.preparador.PrepararPublicacionGobiernoTransiciones(r.Context(), entrada)
	if err != nil {
		responderErrorGobiernoTransiciones(w, err)
		return
	}
	if comando.VersionEsperada != entrada.VersionEsperada || comando.Motivo != entrada.Motivo ||
		comando.ClaveIdempotencia != entrada.ClaveIdempotencia || len(comando.Transiciones) != len(entrada.Transiciones) {
		responderOferta(w, http.StatusForbidden, "acceso_denegado", nil)
		return
	}
	for i := range entrada.Transiciones {
		if comando.Transiciones[i] != entrada.Transiciones[i] {
			responderOferta(w, http.StatusForbidden, "acceso_denegado", nil)
			return
		}
	}
	v, err := h.operador.Publicar(r.Context(), comando)
	if err != nil {
		responderErrorGobiernoTransiciones(w, err)
		return
	}
	v.PuedePublicar = true
	estado := http.StatusCreated
	if v.Reutilizada {
		estado = http.StatusOK
	}
	responderOferta(w, estado, "", v)
}

func responderErrorGobiernoTransiciones(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, dominiovec.ErrAutorizacionDenegada), errors.Is(err, dominiovec.ErrPermissionDenied):
		responderOferta(w, http.StatusForbidden, "acceso_denegado", nil)
	case errors.Is(err, ports.ErrGobiernoTransicionesConflicto):
		responderOferta(w, http.StatusConflict, "version_o_clave_en_conflicto", nil)
	case errors.Is(err, ports.ErrGobiernoTransicionesInvalido):
		responderOferta(w, http.StatusUnprocessableEntity, "politica_invalida", nil)
	default:
		responderOferta(w, http.StatusServiceUnavailable, "servicio_no_disponible", nil)
	}
}
