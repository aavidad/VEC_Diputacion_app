package mibolsa

import (
	"context"
	"errors"
	"log/slog"

	bolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// LectorEstadoPreferenciasModalidades consulta la elección propia con la misma
// orden autenticada que la consulta de Mi Bolsa. La lectura hace su propia
// autorización y su propia auditoría.
type LectorEstadoPreferenciasModalidades interface {
	ConsultarPreferenciasModalidades(context.Context, Orden) (bolsa.EstadoPreferenciasModalidades, error)
}

// ConsultaOfertasModalidades aplica la elección de modalidades solo a la vista
// de ofertas publicadas. No toca participaciones, posiciones, llamamientos ni
// el resto de la instantánea, que ya vienen autorizados por la consulta propia.
type ConsultaOfertasModalidades struct {
	consulta ConsultorLecturaPropia
	lector   LectorEstadoPreferenciasModalidades
}

func NuevaConsultaOfertasModalidades(consulta ConsultorLecturaPropia, lector LectorEstadoPreferenciasModalidades) (*ConsultaOfertasModalidades, error) {
	if nula(consulta) || nula(lector) {
		return nil, ErrServicioMiBolsaInvalido
	}
	return &ConsultaOfertasModalidades{consulta: consulta, lector: lector}, nil
}

// Consultar devuelve la instantánea propia y oculta las ofertas abiertas, sin
// respuesta de la persona, cuya modalidad no eligió. Las ofertas sin modalidad
// identificada y las ya respondidas se muestran siempre.
//
// Si la elección no se puede leer (rol sin la concesión, función ausente o
// fallo pasajero), se muestran todas las ofertas y OfertasSegunModalidades
// queda en falso: la elección solo reduce lo que se ve, y ocultar ofertas por
// un fallo haría perder a la persona ofertas a las que puede responder.
func (s *ConsultaOfertasModalidades) Consultar(ctx context.Context, orden Orden) (bolsa.InstantaneaMiBolsa, error) {
	if s == nil || nula(s.consulta) || nula(s.lector) || ctx == nil {
		return bolsa.InstantaneaMiBolsa{}, ErrServicioMiBolsaInvalido
	}
	instantanea, err := s.consulta.Consultar(ctx, orden)
	if err != nil || !hayOfertasFiltrables(instantanea.Ofertas) {
		return instantanea, err
	}
	estado, err := s.lector.ConsultarPreferenciasModalidades(ctx, orden)
	if err != nil {
		if ctx.Err() != nil {
			return bolsa.InstantaneaMiBolsa{}, ctx.Err()
		}
		slog.WarnContext(ctx, "elección de modalidades no aplicada a las ofertas",
			"operacion", "consultar_mi_bolsa", "etapa", "modalidades", "causa", causaModalidadesNoAplicadas(err))
		return instantanea, nil
	}
	instantanea.OfertasSegunModalidades = true
	if estado.Version == 0 || estado.ModalidadesGuardadas == nil {
		return instantanea, nil
	}
	elegidas := make(map[string]struct{}, len(estado.ModalidadesGuardadas))
	for _, clave := range estado.ModalidadesGuardadas {
		elegidas[clave] = struct{}{}
	}
	visibles := make([]bolsa.OfertaPortalCandidato, 0, len(instantanea.Ofertas))
	for _, oferta := range instantanea.Ofertas {
		if filtrable(oferta) {
			if _, elegida := elegidas[oferta.Datos.ModalidadClave]; !elegida {
				continue
			}
		}
		visibles = append(visibles, oferta)
	}
	instantanea.Ofertas = visibles
	return instantanea, nil
}

// ConsultarTodasOfertas devuelve la instantánea sin aplicar la elección:
// «Ver todas» consulta una vez todas las ofertas sin cambiar lo guardado.
// No lee la elección ni marca OfertasSegunModalidades.
func (s *ConsultaOfertasModalidades) ConsultarTodasOfertas(ctx context.Context, orden Orden) (bolsa.InstantaneaMiBolsa, error) {
	if s == nil || nula(s.consulta) || ctx == nil {
		return bolsa.InstantaneaMiBolsa{}, ErrServicioMiBolsaInvalido
	}
	return s.consulta.Consultar(ctx, orden)
}

func filtrable(o bolsa.OfertaPortalCandidato) bool {
	return o.Estado == bolsa.EstadoOfertaPortalAbierta && o.Disposicion == nil && o.Datos.ModalidadClave != ""
}

// Sin ninguna oferta que la elección pueda ocultar no se lee la elección: se
// evita una lectura y su auditoría que no cambiarían la vista.
func hayOfertasFiltrables(ofertas []bolsa.OfertaPortalCandidato) bool {
	for _, o := range ofertas {
		if filtrable(o) {
			return true
		}
	}
	return false
}

// El registro técnico solo lleva un código cerrado; nunca el texto del error.
func causaModalidadesNoAplicadas(err error) string {
	switch {
	case errors.Is(err, dominiovec.ErrAutorizacionDenegada), errors.Is(err, dominiovec.ErrPermissionDenied):
		return "denegada"
	case errors.Is(err, bolsa.ErrPreferenciasNoDisponibles):
		return "no_disponible"
	default:
		return "error"
	}
}
