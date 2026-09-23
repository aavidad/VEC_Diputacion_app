package registropropio

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// El preparador obtiene credencial y actor desde la frontera autenticada; el
// navegador sólo aporta el correo y su confirmación.
type PreparadorSolicitudRegistroPropio interface {
	PrepararSolicitudRegistroPropio(context.Context) (ports.SolicitudRegistroPropioV1, error)
}

type RegistradorPropio interface {
	Registrar(context.Context, ports.SolicitudRegistroPropioV1) (domain.ReciboRegistroPropioV1, error)
}

type ResultadoContactoAlta struct {
	PersonaRef   string
	Version      uint64
	ReciboRef    string
	EvidenciaRef string
}

type CompletadorContactoRegistroPropio interface {
	CompletarContactoPropio(context.Context, domain.ReciboRegistroPropioV1, string) (ResultadoContactoAlta, error)
}

type Handler struct {
	preparador  PreparadorSolicitudRegistroPropio
	registrador RegistradorPropio
	contacto    CompletadorContactoRegistroPropio
}

func NuevoHandler(preparador PreparadorSolicitudRegistroPropio, registrador RegistradorPropio, contacto CompletadorContactoRegistroPropio) (*Handler, error) {
	if preparador == nil || registrador == nil || contacto == nil {
		return nil, ports.ErrRegistroPropioNoDisponible
	}
	return &Handler{preparador, registrador, contacto}, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if h == nil || h.preparador == nil || h.registrador == nil || h.contacto == nil {
		http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		return
	}
	if r.Method != http.MethodPost || r.Header.Get("Cookie") != "" {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}
	var cuerpo struct {
		Correo       string `json:"correo"`
		Confirmacion string `json:"confirmacion"`
	}
	dec := json.NewDecoder(io.LimitReader(r.Body, 2048))
	dec.DisallowUnknownFields()
	if dec.Decode(&cuerpo) != nil || dec.Decode(new(any)) != io.EOF || cuerpo.Correo == "" || cuerpo.Correo != cuerpo.Confirmacion || strings.TrimSpace(cuerpo.Correo) != cuerpo.Correo {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	solicitud, err := h.preparador.PrepararSolicitudRegistroPropio(r.Context())
	if err != nil {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}
	recibo, err := h.registrador.Registrar(r.Context(), solicitud)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		return
	}
	if recibo.ValidarPendiente() != nil {
		http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		return
	}
	contacto, err := h.contacto.CompletarContactoPropio(r.Context(), recibo, cuerpo.Correo)
	if err != nil || contacto.PersonaRef != recibo.PersonaRef || contacto.Version == 0 || contacto.ReciboRef == "" || contacto.EvidenciaRef == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(struct {
			Estado    string `json:"estado"`
			ReciboRef string `json:"recibo_ref"`
		}{recibo.Estado, recibo.ReciboRef})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(struct {
		Estado            string `json:"estado"`
		ReciboRef         string `json:"recibo_ref"`
		ContactoReciboRef string `json:"contacto_recibo_ref"`
	}{"alta_completa", recibo.ReciboRef, contacto.ReciboRef})
}
