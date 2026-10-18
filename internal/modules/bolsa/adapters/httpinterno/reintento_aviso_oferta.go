package httpinterno

import (
	"context"
	"net/http"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

type EntradaReintentarAvisosOferta struct {
	BolsaRef, OfertaRef, ClaveIdempotencia string
	RevisionAvisos                         int64
	ConfirmarPosibleDuplicado              bool
}

type preparadorReintentoAvisosOferta interface {
	PrepararSolicitudReintentarAvisosOferta(context.Context, EntradaReintentarAvisosOferta) (ports.SolicitudReintentarAvisosOferta, error)
}

type operadorReintentoAvisosOferta interface {
	ReintentarAvisosSinConfirmacion(context.Context, ports.SolicitudReintentarAvisosOferta) (ports.OfertaPublicada, error)
}

func (h *HandlerOfertasPublicadas) reintentar(w http.ResponseWriter, r *http.Request,
	entrada EntradaReintentarAvisosOferta,
) {
	if entrada.BolsaRef == "" || entrada.OfertaRef == "" ||
		entrada.RevisionAvisos < 0 || entrada.RevisionAvisos > 9007199254740991 ||
		!entrada.ConfirmarPosibleDuplicado {
		responderOferta(w, http.StatusBadRequest, "solicitud_invalida", nil)
		return
	}
	p, okP := h.preparador.(preparadorReintentoAvisosOferta)
	o, okO := h.operador.(operadorReintentoAvisosOferta)
	if !okP || !okO {
		responderErrorOferta(w, ports.ErrOfertaNoDisponible)
		return
	}
	q, err := p.PrepararSolicitudReintentarAvisosOferta(r.Context(), entrada)
	if err != nil {
		responderErrorOferta(w, err)
		return
	}
	oferta, err := o.ReintentarAvisosSinConfirmacion(r.Context(), q)
	if err != nil {
		responderErrorOferta(w, err)
		return
	}
	responderOferta(w, estadoEscrituraOferta(oferta), "", oferta)
}
