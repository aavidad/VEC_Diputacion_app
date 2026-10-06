package administracion

import (
	"context"
	"errors"

	personalpg "vec-diputacion-granada/internal/modules/personal/adapters/postgres/cargoadmin"
	api "vec-diputacion-granada/internal/vec/adapters/httpapi/administracionperfiles"
	efecto "vec-diputacion-granada/internal/vec/adapters/postgres/efectonominaladmin"
)

// NuevaConfianzaEfectoNominalV3 compone la cadena V3 con una sola capacidad:
// la de la audiencia del efecto (por ejemplo, la de cargos competenciales del
// conjunto 3 de AD204).
func NuevaConfianzaEfectoNominalV3(cfg ConfiguracionConfianzaPerfilesV3, deps DependenciasConfianzaPerfilesV3, audiencia string) (ConfianzaPerfilesV3, error) {
	if audiencia != AudienciaCargoCompetencialV3 {
		return ConfianzaPerfilesV3{}, ErrConfiguracion
	}
	return nuevaConfianzaConAudienciasV3(cfg, deps, []string{audiencia})
}

type ejecutorEfectoNominal interface {
	Aplicar(context.Context, efecto.Solicitud) (efecto.Recibo, error)
}

// ServicioEfectoNominal traduce entre la ruta de vec-admin y el ejecutor del
// efecto. No decide nada: tipos y errores.
type ServicioEfectoNominal struct{ ejecutor ejecutorEfectoNominal }

var _ api.ServicioEfectoNominalADMIN = (*ServicioEfectoNominal)(nil)

func NuevoServicioEfectoNominal(e ejecutorEfectoNominal) (*ServicioEfectoNominal, error) {
	if dependenciaComposicionNula(e) {
		return nil, ErrConfiguracion
	}
	return &ServicioEfectoNominal{ejecutor: e}, nil
}

func (s *ServicioEfectoNominal) AplicarEfectoNominal(ctx context.Context, x api.SolicitudEfectoNominalADMIN) (api.ReciboEfectoNominalADMIN, error) {
	if s == nil || dependenciaComposicionNula(s.ejecutor) {
		return api.ReciboEfectoNominalADMIN{}, efecto.ErrNoDisponible
	}
	r, err := s.ejecutor.Aplicar(ctx, efecto.Solicitud{Actor: x.Actor, Evidencia: x.Evidencia, Instantanea: x.Instantanea,
		Material: x.Material, CorrelacionRef: x.CorrelacionRef})
	if err != nil {
		if errors.Is(err, efecto.ErrConflicto) {
			return api.ReciboEfectoNominalADMIN{}, errors.Join(api.ErrConflictoEstado, err)
		}
		return api.ReciboEfectoNominalADMIN{}, err
	}
	return api.ReciboEfectoNominalADMIN{Cuerpo: r.Cuerpo, ConsumoAuditoriaRef: r.ConsumoAuditoriaRef}, nil
}

// MaximoMaterialCargoCompetencial es el límite de Personal28 para el material.
const MaximoMaterialCargoCompetencial = personalpg.MaximoMaterialPublicacionCargo
