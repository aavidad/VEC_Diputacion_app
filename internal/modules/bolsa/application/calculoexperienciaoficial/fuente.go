package calculoexperienciaoficial

import (
	"context"
	"time"

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
		validarFuenteExacta(fuente, solicitud.Selector, solicitadaEn, ahora) != nil {
		return puertosbolsa.FuenteExactaCalculoReglasBaremo{}, time.Time{},
			ErrFuenteNoConfiable
	}
	return fuente, ahora, nil
}

func validarFuenteExacta(
	fuente puertosbolsa.FuenteExactaCalculoReglasBaremo,
	selector puertosbolsa.SelectorFuenteExactaCalculoReglasBaremo,
	solicitadaEn, comprobadaEn time.Time,
) error {
	if !selectorValido(selector) || fuente.Version.Validar() != nil ||
		fuente.Version.Estado() != reglas.EstadoReglasBaremoActiva ||
		fuente.Entrada.Validar() != nil || !instanteFuenteValido(fuente.ObtenidaEn) ||
		fuente.ObtenidaEn.Before(solicitadaEn) || fuente.ObtenidaEn.After(comprobadaEn) ||
		validarPruebaFuente(fuente.Prueba, selector, fuente.ObtenidaEn, comprobadaEn) != nil ||
		!referenciaValida(fuente.Auditoria) ||
		!referenciaValida(fuente.ConsumoAutorizacion) ||
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
	return nil
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
	referencias := []reglas.ReferenciaVersionada{
		fuente.Prueba.Evidencia, fuente.Prueba.Verificador, fuente.Auditoria,
		fuente.ConsumoAutorizacion, fuente.ConsumoPrueba,
		fuente.Prueba.EstadoReglas.Contenido(), fuente.Prueba.InstantaneaEntrada,
		fuente.Prueba.SujetoPseudonimo, fuente.Prueba.Convocatoria,
	}
	vistas := make(map[string]struct{}, len(referencias))
	for _, referencia := range referencias {
		clave := referencia.Referencia()
		if _, existe := vistas[clave]; existe {
			return false
		}
		vistas[clave] = struct{}{}
	}
	return true
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
