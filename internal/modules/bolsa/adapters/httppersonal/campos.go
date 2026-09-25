package httppersonal

import (
	"context"
	"errors"
	"net/http"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

// NuevoConCampos compone la consulta con la lista de datos visibles que fija
// el catálogo. Sin lista (campos nulo) la conducta es la de Nuevo.
func NuevoConCampos(preparador Preparador, consultor Consultor, campos puertosbolsa.CamposPortalMiBolsa) (http.Handler, error) {
	if nula(campos) {
		return nil, ErrDependenciaNoDisponible
	}
	manejador, err := Nuevo(preparador, consultor)
	if err != nil {
		return nil, err
	}
	h := manejador.(*Handler)
	h.campos = campos
	return h, nil
}

// camposVisibles resuelve la lista antes de consultar: un catálogo roto o
// no disponible impide la consulta en lugar de mostrar todo.
func (h *Handler) camposVisibles(ctx context.Context) (map[string]bool, error) {
	lista := puertosbolsa.CamposPortalMiBolsaTodos()
	if !nula(h.campos) {
		resuelta, err := h.campos.CamposVisiblesMiBolsa(ctx)
		if err != nil {
			return nil, errors.Join(puertosbolsa.ErrCamposPortalMiBolsaNoDisponibles, err)
		}
		if lista, err = puertosbolsa.ValidarCamposPortalMiBolsa(resuelta); err != nil {
			return nil, err
		}
	}
	visibles := make(map[string]bool, len(lista))
	for _, campo := range lista {
		visibles[campo] = true
	}
	return visibles, nil
}

// filtrarRespuesta retira del servidor los datos ocultos: la web no recibe
// lo que el catálogo no muestra.
func filtrarRespuesta(r respuesta, visibles map[string]bool) respuesta {
	r.Data.CamposVisibles = make([]string, 0, len(visibles))
	for _, campo := range puertosbolsa.CamposPortalMiBolsaTodos() {
		if visibles[campo] {
			r.Data.CamposVisibles = append(r.Data.CamposVisibles, campo)
		}
	}
	for i := range r.Data.Participaciones {
		p := &r.Data.Participaciones[i]
		if !visibles[puertosbolsa.CampoPortalPosicion] {
			p.OrdenInicial, p.TotalInstantanea = nil, nil
		}
		if !visibles[puertosbolsa.CampoPortalUltimoLlamamiento] {
			p.UltimoLlamamiento = nil
		}
		actual, _ := p.SituacionActual.(*situacionActual)
		if actual == nil {
			p.SituacionActual = nil
			if !visibles[puertosbolsa.CampoPortalEstado] {
				p.EstadoBolsa = nil
			}
			continue
		}
		if !visibles[puertosbolsa.CampoPortalFechaDisponible] {
			actual.FechaDisponible = nil
		}
		if !visibles[puertosbolsa.CampoPortalEstado] {
			p.EstadoBolsa = nil
			p.SituacionActual = nil
			if actual.FechaDisponible != nil {
				p.SituacionActual = soloFechaDisponible{FechaDisponible: *actual.FechaDisponible}
			}
		}
	}
	return r
}

// soloFechaDisponible es la situación cuando el estado está oculto y la
// fecha de disponibilidad no.
type soloFechaDisponible struct {
	FechaDisponible string `json:"fecha_disponible"`
}
