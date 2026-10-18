package auditoria

import (
	"net/http"
	"time"
)

func (h *Manejador) servirBusqueda(w http.ResponseWriter, r *http.Request, identidad IdentidadResuelta, c cuerpoConsulta) {
	o, err := h.opciones.Actuales(r.Context())
	if err != nil {
		h.responderIntento(w, r, identidad, recursoIntentoOpcionesRRHH, http.StatusServiceUnavailable, nil)
		return
	}
	if !o.SeleccionObjetivo || c.FinalidadRef != o.FinalidadRef || c.MotivoRef != o.MotivoRef ||
		c.ExpedienteRef != "" || c.ActorRef != "" || c.Desde != "" || c.Hasta != "" || c.Cursor != "" {
		h.responderIntento(w, r, identidad, recursoIntentoConsultaInvalida, http.StatusBadRequest, &o)
		return
	}
	f := Filtro{Fuente: c.Fuente, ExpedienteRef: "auditoria:objetivos:" + c.Fuente,
		Desde: time.Unix(0, 0).UTC(), Hasta: time.Unix(86400, 0).UTC(), Limite: c.Limite,
		FinalidadRef: o.FinalidadRef, MotivoRef: o.MotivoRef, MotivoConsulta: c.MotivoConsulta}
	pagina, err := h.servicio.BuscarObjetivos(r.Context(), Peticion{Filtro: f, Contexto: contextoConsulta(identidad, o)},
		Busqueda{Texto: c.Busqueda, DespuesRef: c.DespuesRef})
	if err != nil {
		codigo := http.StatusServiceUnavailable
		if err == ErrDenegada {
			codigo = http.StatusForbidden
		}
		h.responderIntento(w, r, identidad, recursoIntentoConsultaInvalida, codigo, &o)
		return
	}
	responderJSON(w, http.StatusOK, pagina)
}
