package bootstrap

import (
	"context"

	inc "vec-diputacion-granada/internal/app/incorporacionejercicio"
	bp "vec-diputacion-granada/internal/modules/bolsa/ports"
	dom "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	seg "vec-diputacion-granada/internal/vec/adapters/seguridad"
	core "vec-diputacion-granada/internal/vec/domain"
)

// Las fuentes de los dos efectos pertenecen a Bolsa. La composición les da
// su actor nominal y su motivo gobernado; el cliente sólo aporta la clave de
// su acto CT/Personal original.
type efectosBolsaNativosB115 struct {
	anclaje        bp.ConsultaAceptacionNativaCT
	alta           bp.AplicadorAltaNativaCT
	cese           bp.AplicadorCeseNativoCT
	actorAnclaje   func(context.Context) (bp.ActorConfiablePersonaAceptacionCT, error)
	actorSituacion func(context.Context, *ct.ContextoAutorizacionAltaV3) (context.Context, ct.ContextoAutorizacionAltaV3, error)
	motivo         core.ReferenciaEntradaCatalogo
}

func (e *efectosBolsaNativosB115) anclajeConfirmado(
	ctx context.Context, x dom.SelectorBolsaPlanB2Nativo,
) (*bp.AnclajeAceptacionNativaCT, error) {
	if e == nil || ctx == nil || ctx.Err() != nil || !x.Valido() ||
		dependenciaBootstrapNula(e.anclaje) || e.actorAnclaje == nil {
		return nil, ct.ErrPlanNominalB2NoDisponible
	}
	actor, err := e.actorAnclaje(ctx)
	if err != nil {
		return nil, err
	}
	selector := selectorAceptacionNativaB2(x)
	r, err := e.anclaje.ConsultarAnclajeAceptacionNativaCT(ctx,
		bp.SolicitudConsultaAceptacionNativaCT{Selector: selector, ActorConfiable: actor})
	if err != nil || ctx.Err() != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, err
	}
	if r.Estado != "acreditado" || !anclajeNativoB113Valido(r.Anclaje, selector) ||
		!dom.ReferenciaOpacaValida(r.Evidencia.DecisionRef) ||
		!dom.ReferenciaOpacaValida(r.Evidencia.AuditoriaRef) ||
		!dom.HuellaPlanPersonalB2Valida(r.Evidencia.ConsumoHuellaSHA256) ||
		!dom.InstanteUTCCanonico(r.Evidencia.ConsultadaEn) {
		return nil, ct.ErrPlanNominalB2NoDisponible
	}
	return r.Anclaje, nil
}

func (e *efectosBolsaNativosB115) actorEfecto(
	ctx context.Context, actorCT string, actualCT *ct.ContextoAutorizacionAltaV3,
) (context.Context, ct.ContextoAutorizacionAltaV3, core.ReferenciaCorrelacionAutorizacionV2, error) {
	var vacio ct.ContextoAutorizacionAltaV3
	var correlacion core.ReferenciaCorrelacionAutorizacionV2
	if e == nil || ctx == nil || ctx.Err() != nil || e.actorSituacion == nil ||
		!core.ReferenciaMotivoAutorizacionV2Valida(e.motivo) {
		return nil, vacio, correlacion, ct.ErrPlanNominalB2NoDisponible
	}
	hijo, actor, err := e.actorSituacion(ctx, actualCT)
	if err != nil {
		return nil, vacio, correlacion, err
	}
	datos, err := actor.Vinculo.Datos()
	if hijo == nil || hijo.Err() != nil || err != nil || actor.Resultado.Validar() != nil || actor.Vinculo.ValidarPara(actor.Resultado) != nil ||
		datos.PrincipalID != actorCT {
		return nil, vacio, correlacion, ct.ErrAutorizacionDenegada
	}
	correlacion, err = core.GenerarReferenciaCorrelacionAutorizacionV2(hijo, seg.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return nil, vacio, correlacion, ct.ErrPlanNominalB2NoDisponible
	}
	return hijo, actor, correlacion, nil
}

func (e *efectosBolsaNativosB115) confirmarAlta(
	ctxAnclaje, ctxBolsa context.Context, contrato ct.ContratoPlanNominalB2,
	origen ct.OrigenIncorporacionPersonalB2, actualCT ct.ContextoAutorizacionAltaV3,
) error {
	x := contrato.Material.BolsaNativa
	if x.IsZero() {
		return nil
	}
	if e == nil || ctxBolsa == nil || dependenciaBootstrapNula(e.alta) ||
		origen.Protocolo != ct.ProtocoloIncorporacionPersonalB2 ||
		origen.Confirmacion.PlanRef != contrato.PlanRef ||
		origen.Confirmacion.PlanSHA256 != contrato.PlanSHA256 ||
		origen.Confirmacion.ExpedienteRef != x.ExpedienteRef {
		return ct.ErrPlanNominalB2NoDisponible
	}
	anclaje, err := e.anclajeConfirmado(ctxAnclaje, x)
	if err != nil {
		return err
	}
	actorCT, err := actualCT.Vinculo.Datos()
	if err != nil {
		return ct.ErrAutorizacionDenegada
	}
	hijo, actor, correlacion, err := e.actorEfecto(ctxBolsa, actorCT.PrincipalID, &actualCT)
	if err != nil {
		return err
	}
	h := origen.Confirmacion.Hechos
	r, err := e.alta.AplicarAltaNativaCT(hijo, bp.SolicitudAltaNativaCT{
		Selector: selectorAceptacionNativaB2(x), ParticipacionRef: anclaje.ParticipacionRef,
		Personal: bp.PruebaAltaPersonalCT{ReciboCTRef: origen.ReciboRef,
			VersionExpediente: contrato.Material.VersionExpediente, ActorRef: actorCT.PrincipalID,
			EmpleadoRef: h.EmpleadoRef, RelacionRef: h.RelacionRef,
			RelacionVersion: h.RelacionVersion, RelacionReciboRef: h.RelacionReciboRef,
			OcupacionRef: h.OcupacionRef, OcupacionVersion: h.OcupacionVersion,
			OcupacionReciboRef: h.OcupacionReciboRef, PlanRef: contrato.PlanRef,
			PlanSHA256: contrato.PlanSHA256},
		Vinculo: actor.Vinculo, ResultadoContexto: actor.Resultado,
		Correlacion: correlacion, MotivoAutorizacion: e.motivo,
	})
	if err != nil || ctxBolsa.Err() != nil {
		if ctxBolsa.Err() != nil {
			return ctxBolsa.Err()
		}
		return err
	}
	if r.BolsaRef != x.BolsaRef || r.ParticipacionRef != anclaje.ParticipacionRef ||
		!dom.ReferenciaOpacaValida(r.ReciboSituacion) || !dom.ReferenciaOpacaValida(r.EventoRef) {
		return ct.ErrPlanNominalB2NoDisponible
	}
	return nil
}

func (e *efectosBolsaNativosB115) confirmarCese(
	ctxAnclaje, ctxBolsa context.Context, recibo ct.ReciboOperacionSeguimiento, resultado inc.ResultadoCesePersonalB2,
) error {
	x := resultado.SelectorBolsaNativa
	if x.IsZero() {
		return nil
	}
	if e == nil || ctxBolsa == nil || dependenciaBootstrapNula(e.cese) || !resultado.Aplica ||
		x.OrganizacionRef != recibo.OrganizacionRef || x.ExpedienteRef != recibo.ExpedienteRef {
		return ct.ErrOperacionSeguimientoNoDisponible
	}
	anclaje, err := e.anclajeConfirmado(ctxAnclaje, x)
	if err != nil {
		return err
	}
	hijo, actor, correlacion, err := e.actorEfecto(ctxBolsa, recibo.ActorRef, nil)
	if err != nil {
		return err
	}
	relacionVersion, okRelacion := versionB2DesdeInt64(resultado.Recibo.Version)
	ocupacionVersion, okOcupacion := versionB2DesdeInt64(resultado.OcupacionRecibo.Version)
	if !okRelacion || !okOcupacion {
		return ct.ErrOperacionSeguimientoNoDisponible
	}
	r, err := e.cese.AplicarCeseNativoCT(hijo, bp.SolicitudCeseNativoCT{
		Selector: selectorAceptacionNativaB2(x), ParticipacionRef: anclaje.ParticipacionRef,
		Personal: bp.PruebaCierrePersonalCeseCT{Esquema: bp.EsquemaPruebaCierrePersonalCeseCT,
			ReciboCTRef: recibo.ReciboRef, VersionCT: recibo.VersionResultante,
			ActorRef: recibo.ActorRef, EmpleadoRef: resultado.EmpleadoRef,
			RelacionRef: resultado.RelacionRef, OcupacionRef: resultado.OcupacionRef,
			OrganismoRef: resultado.OrganismoRef, UnidadRef: resultado.UnidadRef,
			RelacionReciboRef: resultado.Recibo.ReciboRef, RelacionVersion: relacionVersion,
			OcupacionReciboRef: resultado.OcupacionRecibo.ReciboRef,
			OcupacionVersion:   ocupacionVersion,
			VigenteHasta:       string(resultado.VigenteHasta), FuenteHuellaSHA256: resultado.FuenteHuellaSHA256},
		Vinculo: actor.Vinculo, ResultadoContexto: actor.Resultado,
		Correlacion: correlacion, MotivoAutorizacion: e.motivo,
	})
	if err != nil || ctxBolsa.Err() != nil {
		if ctxBolsa.Err() != nil {
			return ctxBolsa.Err()
		}
		return err
	}
	if r.BolsaRef != x.BolsaRef || r.ParticipacionRef != anclaje.ParticipacionRef ||
		r.FechaEfecto != recibo.FechaEfecto || !dom.ReferenciaOpacaValida(r.ReciboRef) {
		return ct.ErrOperacionSeguimientoNoDisponible
	}
	return nil
}
