package httpinterno

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// Rutas del correo personalizado del asistente B7: la plantilla del catálogo
// (sin datos de personas) y la vista previa de un destinatario. Ninguna
// reserva, envía ni consume autorización.
const (
	RutaPlantillaCorreoLlamamiento   = RutaEmisionesLlamamiento + "/plantilla"
	RutaVistaPreviaCorreoLlamamiento = RutaEmisionesLlamamiento + "/vista-previa"
)

type EntradaVistaPreviaLlamamiento struct {
	BolsaRef         string
	ParticipacionRef string
	Configuracion    ports.ConfiguracionLlamamiento
}

type PreparadorCorreoLlamamiento interface {
	PrepararSolicitudVistaPreviaLlamamiento(context.Context, EntradaVistaPreviaLlamamiento) (ports.SolicitudVistaPreviaLlamamiento, error)
	PrepararConsultaPlantillaCorreoLlamamiento(context.Context) (dominiovec.ContextoActor, error)
}

type OperadorCorreoLlamamiento interface {
	VistaPreviaLlamamiento(context.Context, ports.SolicitudVistaPreviaLlamamiento) (ports.VistaPreviaCorreoLlamamiento, error)
	PlantillaCorreoLlamamiento(context.Context, dominiovec.ContextoActor, string) (ports.PlantillaCorreoLlamamientoPublica, error)
}

type HandlerCorreoLlamamiento struct {
	preparador PreparadorCorreoLlamamiento
	operador   OperadorCorreoLlamamiento
	idioma     string
}

// NuevoHandlerCorreoLlamamiento atiende las dos rutas; el idioma de los
// textos por defecto lo fija la composición.
func NuevoHandlerCorreoLlamamiento(p PreparadorCorreoLlamamiento, o OperadorCorreoLlamamiento, idioma string) (http.Handler, error) {
	if p == nil || o == nil || idioma == "" {
		return nil, ports.ErrEmisionLlamamientoNoDisponible
	}
	return &HandlerCorreoLlamamiento{p, o, idioma}, nil
}

func (h *HandlerCorreoLlamamiento) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r == nil || r.URL == nil || r.URL.RawPath != "" {
		responderEmision(w, 404, map[string]any{"error": map[string]string{"codigo": "no_encontrado"}})
		return
	}
	switch r.URL.Path {
	case RutaPlantillaCorreoLlamamiento:
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			responderEmision(w, 405, map[string]any{"error": map[string]string{"codigo": "metodo_no_permitido"}})
			return
		}
		h.plantilla(w, r)
	case RutaVistaPreviaCorreoLlamamiento:
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "POST")
			responderEmision(w, 405, map[string]any{"error": map[string]string{"codigo": "metodo_no_permitido"}})
			return
		}
		h.vistaPrevia(w, r)
	default:
		responderEmision(w, 404, map[string]any{"error": map[string]string{"codigo": "no_encontrado"}})
	}
}

func (h *HandlerCorreoLlamamiento) plantilla(w http.ResponseWriter, r *http.Request) {
	if r.ContentLength != 0 || len(r.URL.Query()) != 0 || r.Header.Get("Accept") != "application/json" {
		responderEmision(w, 400, map[string]any{"error": map[string]string{"codigo": "solicitud_invalida"}})
		return
	}
	actor, err := h.preparador.PrepararConsultaPlantillaCorreoLlamamiento(r.Context())
	if err != nil {
		responderErrorEmision(w, err)
		return
	}
	out, err := h.operador.PlantillaCorreoLlamamiento(r.Context(), actor, h.idioma)
	if err != nil {
		responderErrorEmision(w, err)
		return
	}
	responderEmision(w, 200, map[string]any{"data": out})
}

func (h *HandlerCorreoLlamamiento) vistaPrevia(w http.ResponseWriter, r *http.Request) {
	if r.Body == nil || r.ContentLength < 1 || r.ContentLength > 32768 || len(r.TransferEncoding) != 0 || r.Header.Get("Content-Type") != "application/json" || r.Header.Get("Accept") != "application/json" {
		responderEmision(w, 400, map[string]any{"error": map[string]string{"codigo": "solicitud_invalida"}})
		return
	}
	var cuerpo struct {
		BolsaRef         string                         `json:"bolsa_ref"`
		ParticipacionRef string                         `json:"participacion_ref"`
		Configuracion    ports.ConfiguracionLlamamiento `json:"configuracion"`
	}
	d := json.NewDecoder(io.LimitReader(r.Body, 32769))
	d.DisallowUnknownFields()
	if d.Decode(&cuerpo) != nil || d.Decode(&struct{}{}) != io.EOF {
		responderEmision(w, 400, map[string]any{"error": map[string]string{"codigo": "solicitud_invalida"}})
		return
	}
	q, err := h.preparador.PrepararSolicitudVistaPreviaLlamamiento(r.Context(), EntradaVistaPreviaLlamamiento{cuerpo.BolsaRef, cuerpo.ParticipacionRef, cuerpo.Configuracion})
	if err != nil {
		responderErrorEmision(w, err)
		return
	}
	out, err := h.operador.VistaPreviaLlamamiento(r.Context(), q)
	if err != nil {
		responderErrorCorreoLlamamiento(w, err)
		return
	}
	responderEmision(w, 200, map[string]any{"data": out})
}

// responderErrorCorreoLlamamiento concreta el motivo de un 422 para que RRHH sepa
// qué corregir en la plantilla; el resto sigue el contrato de la emisión.
func responderErrorCorreoLlamamiento(w http.ResponseWriter, err error) {
	codigo := ""
	switch {
	case errors.Is(err, dominiobolsa.ErrCorreoLlamamientoExcedeLimite):
		codigo = "correo_excede_limite"
	case errors.Is(err, dominiobolsa.ErrDatosCorreoLlamamientoIncompletos):
		codigo = "datos_incompletos"
	case errors.Is(err, dominiobolsa.ErrPlantillaCorreoLlamamiento):
		codigo = "plantilla_invalida"
	}
	if codigo != "" && errors.Is(err, ports.ErrEmisionLlamamientoInvalida) {
		responderEmision(w, 422, map[string]any{"error": map[string]string{"codigo": codigo}})
		return
	}
	responderErrorEmision(w, err)
}
