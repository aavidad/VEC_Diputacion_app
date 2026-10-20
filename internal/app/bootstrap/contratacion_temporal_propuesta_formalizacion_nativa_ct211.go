package bootstrap

import (
	"context"
	"time"

	bp "vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	dom "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// CT211 registra sólo la propuesta local. B98 proporciona la respuesta firme
// B30; B113 liga esa respuesta a B110/B7 con autorización nominal propia.
// SQL CT211 repite el cotejo propietario después de consumir la decisión CT.
type claveAnclajeNativoPropuestaCT struct{}

func (e *ejecutorPropuestaFormalizacionDesarrollo) materialPropuestaNativaCT(
	ctx context.Context, s ct.SolicitudPropuestaFormalizacion,
	a ct.AntecedentePropuestaFormalizacion, p preparacionLlamamientoDesarrollo,
) (ct.MaterialPropuestaFormalizacion, error) {
	vacio := ct.MaterialPropuestaFormalizacion{}
	if e == nil || e.soporte == nil || ctx == nil || ctx.Err() != nil || a.ValidarPara(s) != nil {
		return vacio, application.ErrResolucionFormalizacionNoAceptada
	}
	if dependenciaEsNulaContratacionTemporalDesarrollo(e.resultadoEmisionNativo) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(e.aceptacionNativa) || e.actorAceptacionNativa == nil {
		if a.Portal != nil {
			return vacio, application.ErrPropuestaFormalizacionNoDisponible
		}
		return vacio, application.ErrResolucionFormalizacionNoAceptada
	}
	var bolsa, reciboEmision, necesidad, unidad, categoria, clave, participacion string
	var confirmada, resuelta time.Time
	var version uint64
	if a.Portal != nil {
		seleccion := a.Portal.Seleccion
		bolsa, reciboEmision, necesidad = seleccion.BolsaRef, seleccion.ReciboEmisionRef, seleccion.NecesidadRef
		unidad, categoria, clave = seleccion.UnidadRef, seleccion.CategoriaRef, seleccion.ClaveCT
		participacion = a.Portal.ParticipacionRef
		confirmada, resuelta, version = seleccion.ConfirmadaEn, a.Portal.ResueltaEn, seleccion.VersionExpediente
	} else {
		if a.Justificante.Continuacion != nil || !a.Justificante.Seleccion.Procedencia.SinRetencionDeclarada ||
			a.Resolucion.Solicitud.Respuesta != ct.RespuestaLlamamientoAceptada {
			return vacio, application.ErrResolucionFormalizacionNoAceptada
		}
		j := a.Justificante.Seleccion
		bolsa, reciboEmision, necesidad = j.Bolsa.Referencia, j.ReciboRef, j.Necesidad.Referencia
		unidad, categoria, clave = p.unidad, p.categoria, a.SeleccionClave
		confirmada, resuelta, version = j.ConfirmadaEn, a.Resolucion.ResueltaEn, j.VersionExpediente
	}
	if clave != p.clave || necesidad != p.necesidad || unidad != p.unidad || categoria != p.categoria {
		return vacio, application.ErrResolucionFormalizacionNoAceptada
	}
	r, err := e.resultadoEmisionNativo.LeerResultadoEmisionCT(ctx,
		bolsa, s.LlamamientoRef, reciboEmision)
	if err != nil || ctx.Err() != nil {
		if ctx.Err() != nil {
			return vacio, ctx.Err()
		}
		return vacio, application.ErrPropuestaFormalizacionNoDisponible
	}
	if r.BolsaRef != bolsa || r.LlamamientoRef != s.LlamamientoRef ||
		r.ReciboEmisionRef != reciboEmision || !r.EmitidoEn.Equal(confirmada) ||
		len(r.Participaciones) != 1 {
		return vacio, application.ErrResolucionFormalizacionNoAceptada
	}
	respuesta := r.Participaciones[0]
	if respuesta.Respuesta != bp.RespuestaPortalAcepta ||
		respuesta.Modo != bp.ModoRespuestaPortalFirme ||
		!dom.ReferenciaOpacaValida(respuesta.ReciboRespuestaRef) ||
		respuesta.RespondidaEn == nil || !dom.InstanteUTCCanonico(*respuesta.RespondidaEn) ||
		respuesta.RespondidaEn.Before(r.EmitidoEn) ||
		respuesta.RespondidaEn.After(resuelta) {
		return vacio, application.ErrResolucionFormalizacionNoAceptada
	}
	if a.Portal != nil && (respuesta.ParticipacionRef != participacion ||
		respuesta.ReciboRespuestaRef != a.Portal.ReciboRespuestaPortalRef ||
		!respuesta.RespondidaEn.Equal(a.Portal.RespondidaEn)) {
		return vacio, application.ErrResolucionFormalizacionNoAceptada
	}
	selector := bp.SelectorAceptacionNativaCT{
		OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef,
		ClaveCT: clave, UnidadRef: p.unidad, CategoriaRef: p.categoria,
		NecesidadRef: necesidad, BolsaRef: bolsa,
		LlamamientoRef: s.LlamamientoRef, ReciboEmisionRef: reciboEmision,
		ReciboRespuestaRef: respuesta.ReciboRespuestaRef,
	}
	// El contexto conserva la ruta real de propuesta y la misma identidad.
	// La ligadura privada permite exclusivamente el anclaje B113 verificado
	// por la autoridad nominal; no convierte el POST en una ruta B2.
	ctx = context.WithValue(ctx, claveIncorporacionV2Desarrollo{}, e.soporte.sello)
	ctx = context.WithValue(ctx, claveContextosNominalesIncorporacion{}, &capturaContextosNominalesIncorporacion{})
	ctx = context.WithValue(ctx, claveAnclajeNativoPropuestaCT{}, selector)
	actor, err := e.actorAceptacionNativa(ctx)
	if err != nil || ctx.Err() != nil {
		if ctx.Err() != nil {
			return vacio, ctx.Err()
		}
		return vacio, application.ErrPropuestaFormalizacionNoDisponible
	}
	anclaje, err := e.aceptacionNativa.ConsultarAnclajeAceptacionNativaCT(ctx,
		bp.SolicitudConsultaAceptacionNativaCT{Selector: selector, ActorConfiable: actor})
	if err != nil || ctx.Err() != nil {
		if ctx.Err() != nil {
			return vacio, ctx.Err()
		}
		return vacio, application.ErrPropuestaFormalizacionNoDisponible
	}
	if anclaje.Estado != "acreditado" || anclaje.Anclaje == nil ||
		anclaje.Anclaje.SelectorAceptacionNativaCT != selector ||
		!dom.ReferenciaOpacaValida(anclaje.Anclaje.RespuestaRef) ||
		!dom.InstanteUTCCanonico(anclaje.Anclaje.RespondidaEn) ||
		!anclaje.Anclaje.RespondidaEn.Equal(*respuesta.RespondidaEn) ||
		anclaje.Anclaje.VersionExpediente != version ||
		!dom.ReferenciaOpacaValida(anclaje.Evidencia.DecisionRef) ||
		!dom.ReferenciaOpacaValida(anclaje.Evidencia.AuditoriaRef) ||
		!dom.HuellaPlanPersonalB2Valida(anclaje.Evidencia.ConsumoHuellaSHA256) ||
		!dom.InstanteUTCCanonico(anclaje.Evidencia.ConsultadaEn) {
		return vacio, application.ErrResolucionFormalizacionNoAceptada
	}
	if a.Portal != nil && (anclaje.Anclaje.ParticipacionRef != participacion ||
		anclaje.Anclaje.RespuestaRef != a.Portal.RespuestaPortalRef) {
		return vacio, application.ErrResolucionFormalizacionNoAceptada
	}
	material := ct.MaterialPropuestaFormalizacion{
		Etapa: "confirmacion", Solicitud: s.Clonar(),
		AceptacionBolsaNativa: &ct.EvidenciaAceptacionBolsaNativa{
			Selector: dom.SelectorBolsaPlanB2Nativo{
				OrganizacionRef: selector.OrganizacionRef, ExpedienteRef: selector.ExpedienteRef,
				ClaveCT: selector.ClaveCT, UnidadRef: selector.UnidadRef,
				CategoriaRef: selector.CategoriaRef, NecesidadRef: selector.NecesidadRef,
				BolsaRef: selector.BolsaRef, LlamamientoRef: selector.LlamamientoRef,
				ReciboEmisionRef:   selector.ReciboEmisionRef,
				ReciboRespuestaRef: selector.ReciboRespuestaRef,
			},
			RespuestaRef: anclaje.Anclaje.RespuestaRef,
			RespondidaEn: anclaje.Anclaje.RespondidaEn.In(time.UTC),
		},
	}
	if material.Validar() != nil || material.AceptacionBolsaNativa.ValidarPara(a) != nil {
		return vacio, application.ErrResolucionFormalizacionNoAceptada
	}
	return material, nil
}
