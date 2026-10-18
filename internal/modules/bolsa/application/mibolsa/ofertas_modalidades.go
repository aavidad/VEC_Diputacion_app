package mibolsa

import (
	"context"

	bolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

// LectorEstadoPreferenciasModalidades consulta la elección propia con la misma
// orden autenticada que la consulta de Mi Bolsa.
type LectorEstadoPreferenciasModalidades interface {
	ConsultarPreferenciasModalidades(context.Context, Orden) (bolsa.EstadoPreferenciasModalidades, error)
}

// ConsultaOfertasModalidades aplica la elección solo a la vista de ofertas.
type ConsultaOfertasModalidades struct {
	consulta ConsultorLecturaPropia
	lector   LectorEstadoPreferenciasModalidades
}

func NuevoConsultaOfertasModalidades(consulta ConsultorLecturaPropia, lector LectorEstadoPreferenciasModalidades) (*ConsultaOfertasModalidades, error) {
	if nula(consulta) || nula(lector) {
		return nil, ErrServicioMiBolsaInvalido
	}
	return &ConsultaOfertasModalidades{consulta: consulta, lector: lector}, nil
}

func (s *ConsultaOfertasModalidades) Consultar(ctx context.Context, orden Orden) (bolsa.InstantaneaMiBolsa, error) {
	if s == nil || nula(s.consulta) || nula(s.lector) || ctx == nil {
		return bolsa.InstantaneaMiBolsa{}, ErrServicioMiBolsaInvalido
	}
	instantanea, err := s.consulta.Consultar(ctx, orden)
	if err != nil {
		return bolsa.InstantaneaMiBolsa{}, err
	}
	if len(instantanea.Ofertas) == 0 {
		return instantanea, nil
	}
	estado, err := s.lector.ConsultarPreferenciasModalidades(ctx, orden)
	if err != nil {
		return bolsa.InstantaneaMiBolsa{}, err
	}
	if estado.Version == 0 {
		return instantanea, nil
	}
	guardadas := make(map[string]struct{}, len(estado.ModalidadesGuardadas))
	for _, clave := range estado.ModalidadesGuardadas {
		guardadas[clave] = struct{}{}
	}
	filtradas := make([]bolsa.OfertaPortalCandidato, 0, len(instantanea.Ofertas))
	for _, oferta := range instantanea.Ofertas {
		if oferta.Estado == bolsa.EstadoOfertaPortalAbierta && oferta.Disposicion == nil && oferta.Datos.ModalidadClave != "" {
			if _, elegida := guardadas[oferta.Datos.ModalidadClave]; !elegida {
				continue
			}
		}
		filtradas = append(filtradas, oferta)
	}
	instantanea.Ofertas = filtradas
	return instantanea, nil
}

// ConsultarTodasLasOfertas ofrece una vista temporal de todas las ofertas
// mediante la misma consulta autorizada; no lee preferencias.
func (s *ConsultaOfertasModalidades) ConsultarTodasLasOfertas(ctx context.Context, orden Orden) (bolsa.InstantaneaMiBolsa, error) {
	if s == nil || nula(s.consulta) || ctx == nil {
		return bolsa.InstantaneaMiBolsa{}, ErrServicioMiBolsaInvalido
	}
	return s.consulta.Consultar(ctx, orden)
}
