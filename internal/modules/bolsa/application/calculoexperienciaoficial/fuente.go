package calculoexperienciaoficial

import (
	"context"
	"time"

	oficial "vec-diputacion-granada/internal/modules/bolsa/domain/calculoexperienciaoficial"
	reglas "vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

func (s *Servicio) obtenerFuente(
	ctx context.Context,
	datos DatosOrdenConfiable,
	autorizacion puertosvec.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2,
	solicitadaEn time.Time,
) (puertosbolsa.FuenteExactaCalculoReglasBaremo, time.Time, error) {
	if err := ctx.Err(); err != nil {
		return puertosbolsa.FuenteExactaCalculoReglasBaremo{}, time.Time{}, err
	}
	solicitud := puertosbolsa.SolicitudFuenteExactaCalculoReglasBaremo{
		Selector: datos.Selector, Autorizacion: autorizacion, SolicitadaEn: solicitadaEn,
	}
	fuente, err := s.fuente.ObtenerFuenteExacta(ctx, solicitud)
	if err != nil {
		return puertosbolsa.FuenteExactaCalculoReglasBaremo{}, time.Time{}, err
	}
	if err := ctx.Err(); err != nil {
		return puertosbolsa.FuenteExactaCalculoReglasBaremo{}, time.Time{}, err
	}
	ahora := instanteCanonico(s.reloj.Ahora())
	if ahora.IsZero() || ahora.Before(solicitadaEn) || autorizacion.ValidarEn(ahora) != nil ||
		validarFuenteExacta(
			fuente, solicitud.Selector, autorizacion, solicitadaEn, ahora,
		) != nil {
		return puertosbolsa.FuenteExactaCalculoReglasBaremo{}, time.Time{},
			ErrFuenteNoConfiable
	}
	return fuente, ahora, nil
}

func validarFuenteExacta(
	fuente puertosbolsa.FuenteExactaCalculoReglasBaremo,
	selector puertosbolsa.SelectorFuenteExactaCalculoReglasBaremo,
	autorizacion puertosvec.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2,
	solicitadaEn, comprobadaEn time.Time,
) error {
	if !selectorValido(selector) || fuente.Version.Validar() != nil ||
		fuente.Version.Estado() != reglas.EstadoReglasBaremoActiva ||
		fuente.Entrada.Validar() != nil || !instanteFuenteValido(fuente.ObtenidaEn) ||
		fuente.ObtenidaEn.Before(solicitadaEn) || fuente.ObtenidaEn.After(comprobadaEn) ||
		validarPruebaFuente(fuente.Prueba, selector, fuente.ObtenidaEn, comprobadaEn) != nil ||
		!referenciaValida(fuente.Auditoria) ||
		fuente.ConsumoAutorizacion.Validar() != nil ||
		!referenciaValida(fuente.ConsumoPrueba) ||
		!referenciasFuenteDistintas(fuente) {
		return ErrFuenteNoConfiable
	}
	vinculo, errVinculo := fuente.Version.VinculoEstado()
	convocatoria, activa, errConvocatoria := fuente.Version.ConvocatoriaActivacion()
	contenido, errContenido := fuente.Version.ReferenciaContenido()
	conjunto, errConjunto := fuente.Version.Conjunto()
	huellaEntrada, errHuella := fuente.Entrada.HuellaSHA256()
	if errVinculo != nil || errConvocatoria != nil || !activa || errContenido != nil ||
		errConjunto != nil || conjunto.Validar() != nil || errHuella != nil ||
		!vinculosEstadoIguales(vinculo, selector.EstadoReglas) ||
		!referenciasIguales(contenido, selector.EstadoReglas.Contenido()) ||
		!referenciasIguales(convocatoria, selector.Convocatoria) ||
		!referenciasIguales(fuente.Entrada.Instantanea(), selector.InstantaneaEntrada) ||
		huellaEntrada != fuente.Prueba.HuellaEntradaSHA256 {
		return ErrFuenteNoConfiable
	}
	referenciaConjunto, err := conjunto.ReferenciaVersionada()
	if err != nil || !referenciasIguales(referenciaConjunto, contenido) {
		return ErrFuenteNoConfiable
	}
	if validarReciboConsumoFuente(
		fuente, selector, autorizacion, solicitadaEn,
	) != nil {
		return ErrFuenteNoConfiable
	}
	return nil
}

func validarReciboConsumoFuente(
	fuente puertosbolsa.FuenteExactaCalculoReglasBaremo,
	selector puertosbolsa.SelectorFuenteExactaCalculoReglasBaremo,
	autorizacion puertosvec.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2,
	solicitadaEn time.Time,
) error {
	datos, errDatos := autorizacion.Datos()
	huellaSelector, errSelector := selector.HuellaSHA256V1()
	recurso, errRecurso := recursoLectura(selector)
	huellaContexto, errContexto := recurso.HuellaContextoAutorizacionSHA256()
	if errDatos != nil || errSelector != nil || errRecurso != nil || errContexto != nil ||
		datos.EsquemaHuella != puertosvec.EsquemaHuellaDecisionAutorizacionReforzadaV2 ||
		datos.VerificadaEn.After(solicitadaEn) {
		return ErrFuenteNoConfiable
	}
	decision := datos.Decision
	if decision.RecursoRef != recurso.Referencia ||
		decision.ContextoRecursoHuellaSHA256 != huellaContexto {
		return ErrFuenteNoConfiable
	}
	return fuente.ConsumoAutorizacion.ValidarPara(
		decision.DecisionRef, datos.EsquemaHuella, datos.HuellaDecisionSHA256,
		decision.RecursoRef, decision.ContextoRecursoHuellaSHA256,
		decision.CorrelacionRef, huellaSelector,
		referenciaExactaOficial(fuente.Prueba.Evidencia),
		referenciaExactaOficial(fuente.ConsumoPrueba),
		referenciaExactaOficial(fuente.Auditoria),
		solicitadaEn, fuente.ObtenidaEn,
	)
}

func validarPruebaFuente(
	prueba puertosbolsa.PruebaFuenteExactaCalculoReglasBaremo,
	selector puertosbolsa.SelectorFuenteExactaCalculoReglasBaremo,
	obtenidaEn, comprobadaEn time.Time,
) error {
	if !referenciaValida(prueba.Evidencia) || !referenciaValida(prueba.Verificador) ||
		!vinculosEstadoIguales(prueba.EstadoReglas, selector.EstadoReglas) ||
		!referenciasIguales(prueba.InstantaneaEntrada, selector.InstantaneaEntrada) ||
		!referenciasIguales(prueba.SujetoPseudonimo, selector.SujetoPseudonimo) ||
		!referenciasIguales(prueba.Convocatoria, selector.Convocatoria) ||
		!huellaSHA256Valida(prueba.HuellaEntradaSHA256) ||
		!instanteFuenteValido(prueba.EmitidaEn) || !instanteFuenteValido(prueba.ValidaHasta) ||
		!prueba.ValidaHasta.After(prueba.EmitidaEn) || prueba.EmitidaEn.After(obtenidaEn) ||
		comprobadaEn.Before(prueba.EmitidaEn) || !comprobadaEn.Before(prueba.ValidaHasta) {
		return ErrFuenteNoConfiable
	}
	return nil
}

func referenciasFuenteDistintas(fuente puertosbolsa.FuenteExactaCalculoReglasBaremo) bool {
	consumo, err := fuente.ConsumoAutorizacion.Consumo()
	if err != nil {
		return false
	}
	referencias := []reglas.ReferenciaVersionada{
		fuente.Prueba.Evidencia, fuente.Prueba.Verificador, fuente.Auditoria,
		fuente.ConsumoPrueba,
		fuente.Prueba.EstadoReglas.Contenido(), fuente.Prueba.InstantaneaEntrada,
		fuente.Prueba.SujetoPseudonimo, fuente.Prueba.Convocatoria,
	}
	vistas := make(map[string]struct{}, len(referencias)+1)
	vistas[consumo.Referencia] = struct{}{}
	for _, referencia := range referencias {
		clave := referencia.Referencia()
		if _, existe := vistas[clave]; existe {
			return false
		}
		vistas[clave] = struct{}{}
	}
	return true
}

func referenciaExactaOficial(referencia reglas.ReferenciaVersionada) oficial.ReferenciaExactaV1 {
	return oficial.ReferenciaExactaV1{
		Referencia: referencia.Referencia(), Version: referencia.Version(),
		HuellaSHA256: referencia.HuellaSHA256(),
	}
}

func referenciasIguales(a, b reglas.ReferenciaVersionada) bool {
	return a.Referencia() == b.Referencia() && a.Version() == b.Version() &&
		a.HuellaSHA256() == b.HuellaSHA256()
}

func vinculosEstadoIguales(a, b reglas.VinculoEstadoReglasBaremo) bool {
	return referenciasIguales(a.Contenido(), b.Contenido()) && a.Revision() == b.Revision() &&
		a.HuellaEstadoSHA256() == b.HuellaEstadoSHA256()
}

func instanteFuenteValido(instante time.Time) bool {
	return !instante.IsZero() && instante.Location() == time.UTC &&
		instante.Equal(instante.Truncate(time.Microsecond))
}
