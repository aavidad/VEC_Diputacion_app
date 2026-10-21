package bootstrap

import (
	"context"
	"time"

	bp "vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type claveResolucionPortalDesarrollo struct{}
type resolucionPortalLigadaDesarrollo struct {
	solicitud ports.SolicitudResolucionPortal
	politica  ports.ReferenciaGobernadaComunicacionLlamamiento
	respuesta bp.ResultadoParticipacionEmisionCT
}

func (e *ejecutorComunicacionLlamamientoDesarrollo) ResolverPortal(
	ctx context.Context, entrada ports.SolicitudResolucionPortal,
) (ports.ResultadoResolucionPortal, error) {
	vacio := ports.ResultadoResolucionPortal{}
	if e == nil || contextoInterfazNulo(ctx) || e.soporte == nil ||
		dependenciaEsNulaContratacionTemporalDesarrollo(e.lector) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(e.servicio) {
		return vacio, application.ErrServicioComunicacionLlamamientoInvalido
	}
	if entrada.OrganizacionRef != "" || entrada.ValidarEntrada() != nil {
		return vacio, application.ErrSolicitudComunicacionLlamamientoInvalida
	}
	capacidad, ok := e.soporte.capacidadValida(ctx)
	if !ok || capacidad.ruta != httpinterno.RutaResolucionComunicacionLlamamiento {
		return vacio, application.ErrComunicacionLlamamientoDenegada
	}
	s := entrada
	s.OrganizacionRef = organizacionAltaContratacionTemporalDesarrollo
	if s.CriterioValidacionRef != criterioRevisionManualDesarrollo || s.Validar() != nil {
		return vacio, application.ErrSolicitudComunicacionLlamamientoInvalida
	}
	expediente, err := e.lector.LeerExpedienteParaAvisoConfirmado(ctx,
		s.OrganizacionRef, s.ExpedienteRef, s.LlamamientoRef)
	if ctx.Err() != nil {
		return vacio, ctx.Err()
	}
	if err != nil || !expedienteComunicacionLlamamientoDesarrolloValido(expediente,
		ports.SolicitudRegistrarComunicacionLlamamiento{OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef}) {
		return vacio, application.ErrComunicacionLlamamientoNoDisponible
	}
	ctx = context.WithValue(ctx, clavePreparacionLlamamientoDesarrollo{}, preparacionLlamamientoDesarrollo{expediente: expediente})
	verificador, ok := e.aceptador.(interface {
		verificarRespuestaPortalParaResolucion(context.Context, ports.SolicitudResolucionPortal) (bp.ResultadoParticipacionEmisionCT, error)
	})
	if !ok {
		return vacio, application.ErrComunicacionLlamamientoNoDisponible
	}
	respuesta, err := verificador.verificarRespuestaPortalParaResolucion(ctx, s)
	if ctx.Err() != nil {
		return vacio, ctx.Err()
	}
	if err != nil {
		return vacio, application.ErrComunicacionLlamamientoNoDisponible
	}
	politica := politicaManualDesarrollo()
	ctx = context.WithValue(ctx, claveResolucionPortalDesarrollo{},
		resolucionPortalLigadaDesarrollo{solicitud: s, politica: politica, respuesta: respuesta})
	servicio, ok := e.servicio.(interface {
		ResolverPortal(context.Context, ports.SolicitudResolucionPortal) (ports.ResultadoResolucionPortal, error)
	})
	if !ok {
		return vacio, application.ErrComunicacionLlamamientoNoDisponible
	}
	r, err := servicio.ResolverPortal(ctx, s)
	if err != nil {
		return vacio, err
	}
	if r.ValidarPara(s) != nil || r.Politica != politica ||
		!r.RespuestaPortal.RespondidaEn.Equal(*respuesta.RespondidaEn) {
		return vacio, application.ErrResultadoComunicacionLlamamientoNoConfiable
	}
	return r, nil
}

func (p *puenteBolsaLlamamientoDesarrollo) verificarRespuestaPortalParaResolucion(
	ctx context.Context, s ports.SolicitudResolucionPortal,
) (bp.ResultadoParticipacionEmisionCT, error) {
	vacia := bp.ResultadoParticipacionEmisionCT{}
	if p == nil || p.alta == nil || p.alta.soporte == nil || dependenciaEsNulaContratacionTemporalDesarrollo(p.reloj) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(p.resultadoEmisionNativo) ||
		ctx == nil || s.Validar() != nil {
		return vacia, ports.ErrRespuestaBolsaNoConfiable
	}
	r, err := p.resultadoEmisionNativo.LeerResultadoEmisionCT(ctx,
		s.BolsaRef, s.LlamamientoRef, s.ReciboEmisionRef)
	if err != nil || r.BolsaRef != s.BolsaRef || r.LlamamientoRef != s.LlamamientoRef ||
		r.ReciboEmisionRef != s.ReciboEmisionRef || len(r.Participaciones) != 1 ||
		!domain.InstanteUTCCanonico(r.EmitidoEn) {
		return vacia, ports.ErrRespuestaBolsaNoConfiable
	}
	part := r.Participaciones[0]
	if part.ParticipacionRef != s.ParticipacionRef ||
		part.ReciboRespuestaRef != s.ReciboRespuestaRef ||
		part.Respuesta != bp.RespuestaPortalAcepta ||
		part.Modo != bp.ModoRespuestaPortalFirme || part.RespondidaEn == nil ||
		!domain.InstanteUTCCanonico(*part.RespondidaEn) ||
		part.RespondidaEn.Before(r.EmitidoEn) || part.RespondidaEn.After(p.reloj.Ahora()) {
		return vacia, ports.ErrRespuestaBolsaNoConfiable
	}
	return part, nil
}

func (p *proveedorComunicacionLlamamientoDesarrollo) PrepararResolucionPortal(
	ctx context.Context, s ports.SolicitudResolucionPortal,
) (postgresct.MaterialResolucionPortal, error) {
	vacio := postgresct.MaterialResolucionPortal{}
	if p == nil || contextoInterfazNulo(ctx) || p.soporte == nil ||
		dependenciaEsNulaContratacionTemporalDesarrollo(p.reloj) || s.Validar() != nil {
		return vacio, ports.ErrOperacionComunicacionLlamamientoDenegada
	}
	capacidad, valida := p.soporte.capacidadValida(ctx)
	preparacion, preparada := ctx.Value(clavePreparacionLlamamientoDesarrollo{}).(preparacionLlamamientoDesarrollo)
	ligada, existe := ctx.Value(claveResolucionPortalDesarrollo{}).(resolucionPortalLigadaDesarrollo)
	politica := politicaManualDesarrollo()
	if !valida || capacidad.ruta != httpinterno.RutaResolucionComunicacionLlamamiento ||
		!preparada || !existe || ligada.solicitud != s || ligada.politica != politica ||
		ligada.respuesta.ParticipacionRef != s.ParticipacionRef ||
		ligada.respuesta.ReciboRespuestaRef != s.ReciboRespuestaRef ||
		!expedienteComunicacionLlamamientoDesarrolloValido(preparacion.expediente,
			ports.SolicitudRegistrarComunicacionLlamamiento{OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef}) {
		return vacio, ports.ErrOperacionComunicacionLlamamientoDenegada
	}
	if _, _, vigente := ventanaAutoridadSinteticaContratacionTemporalDesarrollo(p.reloj.Ahora()); !vigente {
		return vacio, ports.ErrOperacionComunicacionLlamamientoDenegada
	}
	return postgresct.MaterialResolucionPortal{Solicitud: s, Politica: politica}, nil
}

func (p *proveedorComunicacionLlamamientoDesarrollo) AutorizarResolucionPortal(
	ctx context.Context, m postgresct.MaterialResolucionPortal,
) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	vacio := puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	esperado, err := p.PrepararResolucionPortal(ctx, m.Solicitud)
	if err != nil || esperado != m || dependenciaEsNulaContratacionTemporalDesarrollo(p.autorizadorResolucion) {
		return vacio, ports.ErrOperacionComunicacionLlamamientoDenegada
	}
	recurso, err := postgresct.RecursoResolucionPortal(m)
	if err != nil {
		return vacio, ports.ErrOperacionComunicacionLlamamientoDenegada
	}
	a, err := p.autorizadorResolucion.AutorizarOperacion(ctx, postgresct.AccionResolucionManualLlamamiento, recurso)
	if err != nil || a.ValidarEstructura() != nil {
		return vacio, ports.ErrOperacionComunicacionLlamamientoDenegada
	}
	r, ahora := a.ResumenCapacidad(), p.reloj.Ahora()
	if !domain.InstanteUTCCanonico(ahora) || ahora.Before(r.EmitidaEn()) || !ahora.Before(r.ExpiraEn()) ||
		r.ExpiraEn().Sub(r.EmitidaEn()) > 5*time.Minute {
		return vacio, ports.ErrOperacionComunicacionLlamamientoDenegada
	}
	return a, nil
}
