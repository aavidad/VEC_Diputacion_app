package calculoexperienciaoficial

import (
	"context"
	"strings"
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
		!referenciasFuenteDistintas(fuente, autorizacion) ||
		!referenciasFuenteConRolNominal(fuente) {
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
	huellaEntrada, errEntrada := fuente.Entrada.HuellaSHA256()
	recurso, errRecurso := recursoLectura(selector)
	huellaContexto, errContexto := recurso.HuellaContextoAutorizacionSHA256()
	if errDatos != nil || errSelector != nil || errEntrada != nil ||
		errRecurso != nil || errContexto != nil ||
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
		huellaEntrada,
		referenciaExactaOficial(fuente.Prueba.Evidencia),
		referenciaExactaOficial(fuente.Prueba.Verificador),
		referenciaExactaOficial(fuente.ConsumoPrueba),
		referenciaExactaOficial(fuente.Auditoria),
		fuente.Prueba.EmitidaEn, fuente.Prueba.ValidaHasta,
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

func referenciasFuenteDistintas(
	fuente puertosbolsa.FuenteExactaCalculoReglasBaremo,
	autorizacion puertosvec.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2,
) bool {
	consumo, err := fuente.ConsumoAutorizacion.Consumo()
	datos, errAutorizacion := autorizacion.Datos()
	if err != nil || errAutorizacion != nil {
		return false
	}
	decision := datos.Decision
	referencias := []string{
		consumo.Referencia, decision.DecisionRef, decision.RecursoRef, decision.CorrelacionRef,
		fuente.Prueba.Evidencia.Referencia(), fuente.Prueba.Verificador.Referencia(),
		fuente.ConsumoPrueba.Referencia(), fuente.Auditoria.Referencia(),
		fuente.Prueba.EstadoReglas.Contenido().Referencia(),
		fuente.Prueba.InstantaneaEntrada.Referencia(),
		fuente.Prueba.SujetoPseudonimo.Referencia(), fuente.Prueba.Convocatoria.Referencia(),
	}
	vistas := make(map[string]struct{}, len(referencias))
	for _, referencia := range referencias {
		if _, existe := vistas[referencia]; existe {
			return false
		}
		vistas[referencia] = struct{}{}
	}
	return true
}

func referenciasFuenteConRolNominal(
	fuente puertosbolsa.FuenteExactaCalculoReglasBaremo,
) bool {
	roles := []struct {
		referencia reglas.ReferenciaVersionada
		prefijo    string
	}{
		{fuente.Prueba.Evidencia, "evidencia:fuente:"},
		{fuente.Prueba.Verificador, "verificador:fuente:"},
		{fuente.ConsumoPrueba, "consumo:prueba:"},
		{fuente.Auditoria, "auditoria:fuente:"},
		{fuente.Prueba.EstadoReglas.Contenido(), "reglas:"},
		{fuente.Prueba.InstantaneaEntrada, "iex_"},
		{fuente.Prueba.Convocatoria, "convocatoria:"},
	}
	for _, rol := range roles {
		valor := rol.referencia.Referencia()
		if !referenciaValida(rol.referencia) || !strings.HasPrefix(valor, rol.prefijo) ||
			len(valor) <= len(rol.prefijo) || strings.Contains(valor, "..") ||
			strings.ContainsAny(valor, "/@\\") {
			return false
		}
	}
	return oficial.ReferenciaReglasFuenteExactaV1Valida(
		fuente.Prueba.EstadoReglas.Contenido().Referencia(),
	) && oficial.ReferenciaInstantaneaFuenteExactaV1Valida(
		fuente.Prueba.InstantaneaEntrada.Referencia(),
	) && oficial.ReferenciaConvocatoriaFuenteExactaV1Valida(
		fuente.Prueba.Convocatoria.Referencia(),
	)
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
