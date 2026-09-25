package httpinterno

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const (
	// RutaOfertasPublicadas: GET consulta las ofertas de una bolsa y POST
	// publica una nueva (Idempotency-Key obligatoria).
	RutaOfertasPublicadas = "/api/vec/bolsa/ofertas"
	// RutaResolucionesOferta: POST confirma la propuesta de adjudicación o el
	// paso a llamamiento directo de una oferta vencida.
	RutaResolucionesOferta = "/api/vec/bolsa/ofertas/resoluciones"

	maximoCuerpoOferta     = 16384
	limiteOfertasOmision   = 20
	maximoLimiteOfertasWeb = 100
)

type EntradaPublicarOferta struct {
	BolsaRef          string
	Datos             dominiobolsa.DatosOferta
	ClaveIdempotencia string
}

type EntradaResolverOferta struct {
	BolsaRef, OfertaRef, ParticipacionRef, ClaveIdempotencia string
}

// PreparadorOfertasPublicadas liga la entrada mínima a la sesión revalidada;
// nunca toma identidad, unidad ni ámbito del cliente.
type PreparadorOfertasPublicadas interface {
	PrepararSolicitudPublicarOferta(context.Context, EntradaPublicarOferta) (ports.SolicitudPublicarOferta, error)
	PrepararSolicitudResolverOferta(context.Context, EntradaResolverOferta) (ports.SolicitudResolverOferta, error)
	PrepararSolicitudConsultarOfertas(context.Context, string, int) (ports.SolicitudConsultarOfertas, error)
}

type OperadorOfertasPublicadas interface {
	PublicarOferta(context.Context, ports.SolicitudPublicarOferta) (ports.OfertaPublicada, error)
	ResolverOferta(context.Context, ports.SolicitudResolverOferta) (ports.OfertaPublicada, error)
	ConsultarOfertas(context.Context, ports.SolicitudConsultarOfertas) ([]ports.OfertaPublicada, error)
}

type HandlerOfertasPublicadas struct {
	preparador PreparadorOfertasPublicadas
	operador   OperadorOfertasPublicadas
}

func NuevoHandlerOfertasPublicadas(p PreparadorOfertasPublicadas, o OperadorOfertasPublicadas) (http.Handler, error) {
	if p == nil || o == nil {
		return nil, ports.ErrOfertaNoDisponible
	}
	return &HandlerOfertasPublicadas{p, o}, nil
}

func (h *HandlerOfertasPublicadas) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || r == nil || r.URL == nil || r.URL.RawPath != "" {
		responderOferta(w, http.StatusNotFound, "no_encontrado", nil)
		return
	}
	switch {
	case r.URL.Path == RutaOfertasPublicadas && r.Method == http.MethodGet:
		h.consultar(w, r)
	case r.URL.Path == RutaOfertasPublicadas && r.Method == http.MethodPost:
		h.publicar(w, r)
	case r.URL.Path == RutaResolucionesOferta && r.Method == http.MethodPost:
		h.resolver(w, r)
	case r.URL.Path == RutaOfertasPublicadas:
		w.Header().Set("Allow", "GET, POST")
		responderOferta(w, http.StatusMethodNotAllowed, "metodo_no_permitido", nil)
	case r.URL.Path == RutaResolucionesOferta:
		w.Header().Set("Allow", "POST")
		responderOferta(w, http.StatusMethodNotAllowed, "metodo_no_permitido", nil)
	default:
		responderOferta(w, http.StatusNotFound, "no_encontrado", nil)
	}
}

func (h *HandlerOfertasPublicadas) consultar(w http.ResponseWriter, r *http.Request) {
	valores := r.URL.Query()
	bolsa := valores.Get("bolsa_ref")
	limite := limiteOfertasOmision
	if texto := valores.Get("limite"); texto != "" {
		n, err := strconv.Atoi(texto)
		if err != nil || n < 1 || n > maximoLimiteOfertasWeb {
			responderOferta(w, http.StatusBadRequest, "solicitud_invalida", nil)
			return
		}
		limite = n
	}
	for clave, lista := range valores {
		if (clave != "bolsa_ref" && clave != "limite") || len(lista) != 1 {
			responderOferta(w, http.StatusBadRequest, "solicitud_invalida", nil)
			return
		}
	}
	if r.ContentLength != 0 || bolsa == "" || strings.TrimSpace(bolsa) != bolsa || len(bolsa) > 256 {
		responderOferta(w, http.StatusBadRequest, "solicitud_invalida", nil)
		return
	}
	q, err := h.preparador.PrepararSolicitudConsultarOfertas(r.Context(), bolsa, limite)
	if err != nil {
		responderErrorOferta(w, err)
		return
	}
	ofertas, err := h.operador.ConsultarOfertas(r.Context(), q)
	if err != nil {
		responderErrorOferta(w, err)
		return
	}
	responderOferta(w, http.StatusOK, "", map[string]any{"esquema": "vec.bolsa.rrhh.ofertas.v1", "bolsa_ref": bolsa, "ofertas": ofertas})
}

func (h *HandlerOfertasPublicadas) publicar(w http.ResponseWriter, r *http.Request) {
	clave, ok := cabecerasEscrituraOferta(r)
	if !ok {
		responderOferta(w, http.StatusBadRequest, "solicitud_invalida", nil)
		return
	}
	var cuerpo struct {
		BolsaRef string                   `json:"bolsa_ref"`
		Datos    dominiobolsa.DatosOferta `json:"datos"`
	}
	if !decodificarCuerpoOferta(r, &cuerpo) {
		responderOferta(w, http.StatusBadRequest, "solicitud_invalida", nil)
		return
	}
	q, err := h.preparador.PrepararSolicitudPublicarOferta(r.Context(), EntradaPublicarOferta{BolsaRef: cuerpo.BolsaRef, Datos: cuerpo.Datos, ClaveIdempotencia: clave})
	if err != nil {
		responderErrorOferta(w, err)
		return
	}
	oferta, err := h.operador.PublicarOferta(r.Context(), q)
	if err != nil {
		responderErrorOferta(w, err)
		return
	}
	responderOferta(w, estadoEscrituraOferta(oferta), "", oferta)
}

func (h *HandlerOfertasPublicadas) resolver(w http.ResponseWriter, r *http.Request) {
	clave, ok := cabecerasEscrituraOferta(r)
	if !ok {
		responderOferta(w, http.StatusBadRequest, "solicitud_invalida", nil)
		return
	}
	var cuerpo struct {
		BolsaRef         string  `json:"bolsa_ref"`
		OfertaRef        string  `json:"oferta_ref"`
		ParticipacionRef *string `json:"participacion_ref"`
	}
	if !decodificarCuerpoOferta(r, &cuerpo) {
		responderOferta(w, http.StatusBadRequest, "solicitud_invalida", nil)
		return
	}
	participacion := ""
	if cuerpo.ParticipacionRef != nil {
		if *cuerpo.ParticipacionRef == "" {
			responderOferta(w, http.StatusBadRequest, "solicitud_invalida", nil)
			return
		}
		participacion = *cuerpo.ParticipacionRef
	}
	q, err := h.preparador.PrepararSolicitudResolverOferta(r.Context(), EntradaResolverOferta{BolsaRef: cuerpo.BolsaRef, OfertaRef: cuerpo.OfertaRef, ParticipacionRef: participacion, ClaveIdempotencia: clave})
	if err != nil {
		responderErrorOferta(w, err)
		return
	}
	oferta, err := h.operador.ResolverOferta(r.Context(), q)
	if err != nil {
		responderErrorOferta(w, err)
		return
	}
	responderOferta(w, estadoEscrituraOferta(oferta), "", oferta)
}

func cabecerasEscrituraOferta(r *http.Request) (string, bool) {
	if r.Body == nil || r.ContentLength < 1 || r.ContentLength > maximoCuerpoOferta || len(r.TransferEncoding) != 0 ||
		r.Header.Get("Content-Type") != "application/json" || r.Header.Get("Accept") != "application/json" {
		return "", false
	}
	clave := r.Header.Get("Idempotency-Key")
	if len(r.Header.Values("Idempotency-Key")) != 1 || len(clave) < 8 || len(clave) > 256 || strings.TrimSpace(clave) != clave {
		return "", false
	}
	return clave, true
}

func decodificarCuerpoOferta(r *http.Request, destino any) bool {
	d := json.NewDecoder(io.LimitReader(r.Body, maximoCuerpoOferta+1))
	d.DisallowUnknownFields()
	return d.Decode(destino) == nil && d.Decode(&struct{}{}) == io.EOF
}

func estadoEscrituraOferta(oferta ports.OfertaPublicada) int {
	if oferta.Reutilizada {
		return http.StatusOK
	}
	return http.StatusCreated
}

func responderErrorOferta(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, dominiovec.ErrAutorizacionDenegada), errors.Is(err, dominiovec.ErrPermissionDenied):
		responderOferta(w, http.StatusForbidden, "acceso_denegado", nil)
	case errors.Is(err, ports.ErrOfertaConflicto):
		responderOferta(w, http.StatusConflict, "clave_divergente", nil)
	case errors.Is(err, ports.ErrOfertaYaResuelta):
		responderOferta(w, http.StatusConflict, "oferta_ya_resuelta", nil)
	case errors.Is(err, ports.ErrOfertaPlazoAbierto):
		responderOferta(w, http.StatusConflict, "plazo_abierto", nil)
	case errors.Is(err, ports.ErrOfertaPropuestaCambiada):
		responderOferta(w, http.StatusConflict, "propuesta_cambiada", nil)
	case errors.Is(err, ports.ErrOfertaInvalida), errors.Is(err, dominiobolsa.ErrDatosOfertaInvalidos):
		responderOferta(w, http.StatusUnprocessableEntity, "oferta_invalida", nil)
	case errors.Is(err, ports.ErrPlazoOfertaNoConfigurado):
		responderOferta(w, http.StatusServiceUnavailable, "plazo_no_configurado", nil)
	default:
		responderOferta(w, http.StatusServiceUnavailable, "servicio_no_disponible", nil)
	}
}

func responderOferta(w http.ResponseWriter, estado int, codigo string, datos any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(estado)
	if codigo != "" {
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"codigo": codigo}})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"data": datos})
}
