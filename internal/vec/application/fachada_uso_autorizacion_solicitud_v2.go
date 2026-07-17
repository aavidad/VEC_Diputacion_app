package application

import (
	"context"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// FachadaUsoDecisionAutorizacionSolicitudLigadaV2 es la composicion explicita
// para modulos nuevos. No implementa la interfaz V1 ni acepta su autorizador.
type FachadaUsoDecisionAutorizacionSolicitudLigadaV2 struct {
	autorizador ports.AutorizadorSolicitudLigadaV2
	reloj       ports.Reloj
}

func NuevaFachadaUsoDecisionAutorizacionSolicitudLigadaV2(
	autorizador ports.AutorizadorSolicitudLigadaV2,
	reloj ports.Reloj,
) (*FachadaUsoDecisionAutorizacionSolicitudLigadaV2, error) {
	if dependenciaAutorizacionNula(autorizador) || dependenciaAutorizacionNula(reloj) {
		return nil, domain.ErrConfiguracionAccesoInvalida
	}
	return &FachadaUsoDecisionAutorizacionSolicitudLigadaV2{
		autorizador: autorizador,
		reloj:       reloj,
	}, nil
}

func (f *FachadaUsoDecisionAutorizacionSolicitudLigadaV2) ExigirEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2(
	ctx context.Context,
	actor domain.ContextoActor,
	vinculo domain.VinculoAutenticacionActorV1,
	recurso domain.RecursoAutorizable,
	correlacion domain.ReferenciaCorrelacionAutorizacionV2,
	motivo domain.ReferenciaEntradaCatalogo,
	politica PoliticaUsoDecisionAutorizacion,
) (ports.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2, error) {
	vacia := ports.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2{}
	if f == nil || dependenciaAutorizacionNula(f.autorizador) || dependenciaAutorizacionNula(f.reloj) ||
		ctx == nil || politica.validar() != nil {
		return vacia, errorUsoDecisionAutorizacion(domain.ErrConfiguracionAccesoInvalida)
	}
	if err := ctx.Err(); err != nil {
		return vacia, errorUsoDecisionAutorizacion(err)
	}
	if err := validarPerfilProteccionUsoAutorizacion(actor, vinculo, politica); err != nil {
		return vacia, errorUsoDecisionAutorizacion(err)
	}
	if recurso.ModuloID != politica.datos.moduloID || recurso.Tipo != politica.datos.tipoRecurso {
		return vacia, errorUsoDecisionAutorizacion(domain.ErrSolicitudAutorizacionInvalida)
	}
	decision, err := exigirDecisionAutorizacionVinculadaSolicitudLigadaV2(
		ctx, f.autorizador, f.reloj, actor, vinculo,
		politica.datos.accion, recurso, politica.datos.finalidad, correlacion, motivo,
		usoCamposDecisionConsumidos,
	)
	if err != nil {
		return vacia, err
	}
	if !camposDecisionCoincidenConPolitica(decision.CamposPermitidos, politica) ||
		(politica.datos.perfil == PerfilProteccionUsoAutorizacionInternoAlto &&
			decision.GarantiaMinima != domain.AuthAssuranceHigh) {
		return vacia, errorUsoDecisionAutorizacion(domain.ErrDecisionAutorizacionInvalida)
	}
	verificadaEn := f.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if verificadaEn.IsZero() || !decision.VigenteParaEfectoEn(verificadaEn) {
		return vacia, errorUsoDecisionAutorizacion(domain.ErrDecisionAutorizacionInvalida)
	}
	evidencia, err := ports.NuevaEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2(decision, verificadaEn)
	if err != nil {
		return vacia, errorUsoDecisionAutorizacion(err)
	}
	if err := evidencia.ValidarMotivo(motivo); err != nil {
		return vacia, errorUsoDecisionAutorizacion(err)
	}
	if err := ctx.Err(); err != nil {
		return vacia, errorUsoDecisionAutorizacion(err)
	}
	return evidencia, nil
}

var _ ExigidorEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2 = (*FachadaUsoDecisionAutorizacionSolicitudLigadaV2)(nil)
