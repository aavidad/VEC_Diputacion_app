package postgres

import (
	"context"
	"errors"

	api "vec-diputacion-granada/internal/vec/adapters/httpapi/administracionperfiles"
	"vec-diputacion-granada/internal/vec/ports"
)

// El compuesto nunca transforma un fallo de AD169 en un evento sin Persona.
// La familia técnica sólo registra la fase anterior al par V2 usable.
type AuditorFronteraCompuesto struct {
	nominal api.AuditorFrontera
	tecnico *RegistradorFronteraTecnica
}

var _ api.AuditorFrontera = (*AuditorFronteraCompuesto)(nil)

func NuevoAuditorFronteraCompuesto(nominal api.AuditorFrontera, tecnico *RegistradorFronteraTecnica) (*AuditorFronteraCompuesto, error) {
	if ausente(nominal) || tecnico == nil {
		return nil, ports.ErrFronteraAdminTecnicaNoDisponible
	}
	return &AuditorFronteraCompuesto{nominal: nominal, tecnico: tecnico}, nil
}
func (a *AuditorFronteraCompuesto) RegistrarDenegacionADMIN(ctx context.Context, d api.DenegacionADMIN) error {
	if a == nil || ctx == nil || ausente(a.nominal) || a.tecnico == nil {
		return ports.ErrFronteraAdminTecnicaNoDisponible
	}
	if d.Evidencia.ValidarPara(d.Actor) == nil {
		return a.nominal.RegistrarDenegacionADMIN(ctx, d)
	}
	if d.SesionResuelta || d.Actor.PersonaRef != "" || d.Evidencia.ResultadoContexto.RegistroContextoRef != "" || d.Evidencia.Vinculo.Validar() == nil {
		return ports.ErrFronteraAdminTecnicaNoDisponible
	}
	correlacion, err := ports.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
	if err != nil {
		return falloFronteraTecnica(err)
	}
	ref, err := correlacion.ValorCanonico()
	if err != nil {
		return falloFronteraTecnica(err)
	}
	recurso, err := ports.RecursoFronteraAdminTecnica(ref)
	if err != nil {
		return falloFronteraTecnica(err)
	}
	evento, err := ports.NuevaReferenciaEventoFronteraAdminTecnica()
	if err != nil {
		return falloFronteraTecnica(err)
	}
	resultado, ok := a.tecnico.codigos[d.Codigo]
	if !ok {
		return ports.ErrFronteraAdminTecnicaNoDisponible
	}
	e := ports.EventoFronteraAdminTecnica{TipoRegistro: "frontera_admin_tecnica", EventoRef: evento, OperadorLogin: a.tecnico.login, Accion: "controlar_frontera_admin_v1", RecursoRef: recurso, Resultado: resultado, CodigoRef: d.Codigo, Proceso: a.tecnico.config.Proceso, Canal: a.tecnico.config.Canal, FinalidadRef: "control_frontera_admin", CorrelacionRef: ref}
	registroCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), a.tecnico.config.Plazo)
	defer cancel()
	acuse, err := a.tecnico.AppendFronteraAdminTecnica(registroCtx, e)
	if errors.Is(err, ports.ErrFronteraAdminTecnicaCommitIncierto) {
		acuse, err = a.tecnico.AppendFronteraAdminTecnica(registroCtx, e)
	}
	if err != nil {
		return falloFronteraTecnica(err)
	}
	if acuse.ValidarPara(e) != nil {
		return ports.ErrFronteraAdminTecnicaNoDisponible
	}
	return nil
}
