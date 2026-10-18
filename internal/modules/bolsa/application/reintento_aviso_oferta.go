package application

import (
	"context"
	"strconv"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const maxRevisionAvisos = 9007199254740991

// ReintentarAvisosSinConfirmacion requiere una confirmación expresa y consume
// una decisión nueva ligada a esta clave y revisión antes del efecto SQL.
func (s *ServicioOfertasPublicadas) ReintentarAvisosSinConfirmacion(
	ctx context.Context, q ports.SolicitudReintentarAvisosOferta,
) (ports.OfertaPublicada, error) {
	if ctx == nil || s == nil || q.ResultadoContexto.Validar() != nil ||
		q.Vinculo.ValidarPara(q.ResultadoContexto) != nil ||
		!referenciaOfertaValida(q.BolsaRef) || !referenciaOfertaValida(q.OfertaRef) ||
		q.RevisionAvisos < 0 || q.RevisionAvisos > maxRevisionAvisos || !q.ConfirmarDuplicado ||
		!claveOfertaValida(q.ClaveIdempotencia) || q.Correlacion.Validar() != nil ||
		!dominiovec.ReferenciaMotivoAutorizacionV2Valida(q.MotivoAutorizacion) {
		return ports.OfertaPublicada{}, ports.ErrOfertaInvalida
	}
	if s.avisos == nil {
		return ports.OfertaPublicada{}, ports.ErrOfertaNoDisponible
	}
	repositorio, ok := s.repositorio.(ports.RepositorioReintentosAvisosOferta)
	if !ok {
		return ports.OfertaPublicada{}, ports.ErrOfertaNoDisponible
	}
	actor := q.ResultadoContexto.Contexto.PersonaRef
	huella := huellaCamposOfertaV5([]string{"vec.bolsa.oferta.avisos.reintentar.v1",
		q.BolsaRef, q.OfertaRef, actor, strconv.FormatInt(q.RevisionAvisos, 10), q.ClaveIdempotencia, "true"})
	emision, err := s.materialEmision(ctx, q.Vinculo, q.ResultadoContexto, q.BolsaRef,
		q.Correlacion, q.MotivoAutorizacion, huella)
	if err != nil {
		return ports.OfertaPublicada{}, err
	}
	o, err := repositorio.ReintentarAvisosSinConfirmacion(ctx, ports.ComandoReintentarAvisosOferta{
		BolsaRef: q.BolsaRef, OfertaRef: q.OfertaRef, ActorRef: emision.ActorRef,
		UnidadRef: emision.UnidadRef, AmbitoRef: emision.AmbitoRef,
		RevisionAvisos: q.RevisionAvisos, ConfirmarDuplicado: true,
		ClaveIdempotencia: q.ClaveIdempotencia, Material: emision.Material,
	})
	if err != nil {
		return ports.OfertaPublicada{}, err
	}
	if o.OfertaRef != q.OfertaRef || o.BolsaRef != q.BolsaRef || o.Aviso == nil ||
		o.ReintentoAviso == nil || o.ReintentoAviso.ReciboRef == "" ||
		o.ReintentoAviso.Rearmados < 1 {
		return ports.OfertaPublicada{}, ports.ErrOfertaNoDisponible
	}
	recibo := *o.ReintentoAviso
	actual := s.despacharAvisos(ctx, o)
	actual.ReintentoAviso = &recibo
	return actual, nil
}
