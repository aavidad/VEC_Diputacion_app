package bootstrap

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

// Ofrece una vía para preparar el paso con la selección nominal vigente.
// El registro vuelve a comprobar V3, firma PAdES y custodia reales.
type preparadorViaFirmaVecCT struct{ montaje *montajeFirmaVecR5CT }

func (p preparadorViaFirmaVecCT) PrepararViasFirmaR5(ctx context.Context, q ports.SolicitudDisponibilidadFirmaR5) ([]ports.PreparacionViaFirmaR5, error) {
	m := p.montaje
	if m == nil || m.nominal == nil || m.seleccion == nil {
		return nil, ports.ErrPreflightFirmaR5NoDisponible
	}
	operativo, err := m.nominal.contexto(ctx)
	if err != nil {
		return nil, ports.ErrFirmaDocumentoDenegada
	}
	actor, err := operativo.Vinculo.Datos()
	if err != nil || actor.PrincipalID != q.ActorRef || actor.PerfilActivoRef != q.PerfilRef ||
		operativo.Resultado.HuellaSHA256 != q.ContextoHuella || actor.GarantiaObservada != core.AuthAssuranceSubstantial {
		return nil, ports.ErrFirmaDocumentoDenegada
	}
	peticion, ok := ctx.Value(clavePeticionNominalFirmaCT{}).(*peticionNominalFirmaCT)
	if !ok || peticion == nil {
		return nil, ports.ErrFirmaDocumentoDenegada
	}
	plan, err := m.plan.Plan(ctx)
	if err != nil {
		return nil, ports.ErrPreflightFirmaR5NoDisponible
	}
	for _, paso := range plan.Pasos {
		if paso.Documento != q.Preflight.Documento || paso.PasoOrden != uint64(q.PasoOrden) || paso.PasoRef != q.PasoRef ||
			paso.Circuito.Referencia != q.CatalogoRef || paso.Circuito.HuellaSHA256 != q.CatalogoHuella {
			continue
		}
		seleccion, err := m.seleccion.SeleccionarFirmanteV2(ctx, ports.SolicitudSeleccionFirmanteV2{
			CertificadoHuella: peticion.solicitud.CertificadoCanalSHA256, CargoRef: paso.CargoRef, RolID: paso.RolID,
			Accion: paso.Accion, TipoRecurso: paso.TipoRecurso, Finalidad: paso.Finalidad,
			OrganizacionRef: paso.OrganizacionRef, UnidadRef: paso.UnidadRef})
		if err != nil {
			return nil, errors.Join(ports.ErrPreflightFirmaR5NoDisponible, err)
		}
		if seleccion.PersonaRef != actor.PrincipalID || seleccion.CuentaRef != actor.CuentaRef ||
			seleccion.PerfilActivoRef != actor.PerfilActivoRef || seleccion.RolID != paso.RolID || seleccion.CargoRef != paso.CargoRef {
			return []ports.PreparacionViaFirmaR5{}, nil
		}
		snapshot, err := m.fuenteAutorizacion.ObtenerInstantaneaAutorizacion(ctx, actor.PrincipalID, actor.PerfilActivoRef)
		ahora := m.reloj.Ahora().UTC().Truncate(time.Microsecond)
		if err != nil || snapshot.Validar() != nil || !snapshot.AsignacionPerfil.VigenteEn(ahora) ||
			snapshot.VersionRol.Referencia() != seleccion.RolRef || snapshot.ControlVigenciaVersionRol.Estado != core.EstadoControlVigenciaVersionRolHabilitada {
			return nil, ports.ErrFirmaDocumentoDenegada
		}
		rolSHA, err := snapshot.VersionRol.HuellaSHA256()
		if err != nil || rolSHA != seleccion.RolHuellaSHA256 {
			return nil, ports.ErrFirmaDocumentoDenegada
		}
		hasta := ahora.Add(20 * time.Second)
		if actor.SesionValidaHasta.Before(hasta) {
			hasta = actor.SesionValidaHasta
		}
		if !hasta.After(ahora) {
			return nil, ports.ErrFirmaDocumentoDenegada
		}
		return []ports.PreparacionViaFirmaR5{{Solicitud: q, Via: ports.ViaFirmaCertificadoVEC,
			Original:    ports.ComprobanteComponenteFirmaR5{Referencia: q.Preflight.OriginalRef, HuellaSHA256: q.OriginalHuella},
			PreparadaEn: ahora, ValidaHasta: hasta}}, nil
	}
	return []ports.PreparacionViaFirmaR5{}, nil
}
