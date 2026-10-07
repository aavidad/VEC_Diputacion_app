package internactproveedores

import (
	"context"

	"vec-diputacion-granada/internal/app/composicion/internagobierno"
	"vec-diputacion-granada/internal/vec/adapters/httpapi"
	core "vec-diputacion-granada/internal/vec/domain"
	ports "vec-diputacion-granada/internal/vec/ports"
)

type AutoridadContextoOrganizacionHistorica struct{ Fuente *internagobierno.FuenteF1 }

func (a AutoridadContextoOrganizacionHistorica) ResolverContextoOrganizacionHistorica(ctx context.Context) (core.ContextoActor, string, string, error) {
	if a.Fuente == nil {
		return core.ContextoActor{}, "", "", ErrAutoridadCTNoDisponible
	}
	r, org, unidad, err := a.Fuente.ContextoVinculadoOrganizacionHistorica(ctx)
	if err != nil {
		return core.ContextoActor{}, "", "", httpapi.ErrAccesoRutaExactaDenegado
	}
	actor, err := r.Resultado.Contexto.Clonar()
	if err != nil {
		return core.ContextoActor{}, "", "", ErrAutoridadCTNoDisponible
	}
	return actor, org, unidad, nil
}

// La denegación conserva solo ruta nominal, correlación y actor verificado;
// ningún filtro temporal o identificador de fuente llega a esta bitácora.
type AuditorDenegacionOrganizacionHistorica struct {
	Registrador ports.RegistradorAuditoriaFronteraRutaExacta
}

func (a AuditorDenegacionOrganizacionHistorica) RegistrarDenegacionOrganizacionHistorica(ctx context.Context, d httpapi.DenegacionOrganizacionHistorica) error {
	if ctx == nil || ctx.Err() != nil || interfazNula(a.Registrador) {
		return ErrAutoridadCTNoDisponible
	}
	o := ports.OrdenAuditoriaFronteraRutaExacta{CorrelacionRef: d.CorrelacionRef, Motivo: ports.MotivoAuditoriaFronteraRutaExacta(d.Motivo), Superficie: ports.SuperficieAuditoriaFronteraRutaExactaOrganizacionHistoricaPersonal, Ruta: d.Ruta, ActorRef: d.ActorRef}
	if o.Validar() != nil {
		return ErrAutoridadCTNoDisponible
	}
	return a.Registrador.RegistrarAuditoriaFronteraRutaExacta(ctx, o)
}

var _ httpapi.AutoridadContextoOrganizacionHistorica = AutoridadContextoOrganizacionHistorica{}
var _ httpapi.AuditorDenegacionOrganizacionHistorica = AuditorDenegacionOrganizacionHistorica{}
