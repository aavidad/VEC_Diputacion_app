package ports

import (
	"context"
	"time"
)

func (v *VerificadorEvidenciaIntegracionBolsa) VerificarDisponibilidad(
	ctx context.Context,
	solicitud SolicitudDisponibilidadBolsa,
	resultado ResultadoDisponibilidadBolsa,
	instante time.Time,
) (ComprobanteEvidenciaIntegracionBolsa, error) {
	if resultado.ValidarParaEn(solicitud, instante) != nil {
		return ComprobanteEvidenciaIntegracionBolsa{}, ErrRespuestaBolsaNoConfiable
	}
	material := materialDisponibilidadBolsa(solicitud, resultado)
	peticion := nuevaSolicitudVerificacionBolsa(material, resultado.Procedencia, solicitud.Contexto)
	comprobante, err := v.verificar(ctx, peticion, instante)
	if err != nil {
		return ComprobanteEvidenciaIntegracionBolsa{}, err
	}
	if !comprobante.coincide(peticion) {
		return ComprobanteEvidenciaIntegracionBolsa{}, ErrEvidenciaBolsaNoAutenticada
	}
	return comprobante, nil
}

func (v *VerificadorEvidenciaIntegracionBolsa) VerificarReciboOrden(
	ctx context.Context,
	comando ComandoPrepararOrdenBolsa,
	recibo ReciboOrdenBolsa,
	instante time.Time,
) (ComprobanteEvidenciaIntegracionBolsa, error) {
	if recibo.ValidarParaEn(comando, instante) != nil {
		return ComprobanteEvidenciaIntegracionBolsa{}, ErrRespuestaBolsaNoConfiable
	}
	material := materialReciboOrdenBolsa(comando, recibo)
	peticion := nuevaSolicitudVerificacionBolsa(material, recibo.Procedencia, comando.Contexto)
	comprobante, err := v.verificar(ctx, peticion, instante)
	if err != nil {
		return ComprobanteEvidenciaIntegracionBolsa{}, err
	}
	if !comprobante.coincide(peticion) {
		return ComprobanteEvidenciaIntegracionBolsa{}, ErrEvidenciaBolsaNoAutenticada
	}
	return comprobante, nil
}

func (v *VerificadorEvidenciaIntegracionBolsa) VerificarReciboLlamamiento(
	ctx context.Context,
	comando ComandoSolicitarLlamamientoBolsa,
	recibo ReciboSolicitudLlamamientoBolsa,
	instante time.Time,
) (ComprobanteEvidenciaIntegracionBolsa, error) {
	if recibo.ValidarParaEn(comando, instante) != nil {
		return ComprobanteEvidenciaIntegracionBolsa{}, ErrRespuestaBolsaNoConfiable
	}
	datosComando, err := comando.DatosEn(instante)
	if err != nil {
		return ComprobanteEvidenciaIntegracionBolsa{}, ErrRespuestaBolsaNoConfiable
	}
	material := materialReciboLlamamientoBolsa(comando, recibo)
	peticion := nuevaSolicitudVerificacionBolsa(material, recibo.Procedencia, datosComando.Contexto)
	comprobante, err := v.verificar(ctx, peticion, instante)
	if err != nil {
		return ComprobanteEvidenciaIntegracionBolsa{}, err
	}
	if !comprobante.coincide(peticion) {
		return ComprobanteEvidenciaIntegracionBolsa{}, ErrEvidenciaBolsaNoAutenticada
	}
	return comprobante, nil
}

func materialDisponibilidadBolsa(
	solicitud SolicitudDisponibilidadBolsa,
	resultado ResultadoDisponibilidadBolsa,
) []byte {
	c := canonicoRespuestaBolsa(
		"respuesta-disponibilidad-volatil", solicitud.Contexto,
		resultado.OperacionRef, resultado.OrganizacionRef, resultado.ExpedienteRef,
		resultado.VersionExpediente, resultado.CorrelacionRef,
		resultado.Necesidad, resultado.Resultado, resultado.Procedencia,
	)
	c.referencia("peticion_necesidad", solicitud.Necesidad)
	c.campo("peticion_categoria_ref", solicitud.CategoriaRef)
	c.entero("peticion_maximo_resultados", uint64(solicitud.MaximoResultados))
	c.campo("respuesta_categoria_ref", resultado.CategoriaRef)
	c.booleano("respuesta_bolsa_encontrada", resultado.BolsaEncontrada)
	c.referencia("respuesta_bolsa", resultado.Bolsa)
	c.booleano("respuesta_disponible", resultado.Disponible)
	c.entero("respuesta_cantidad_disponible", uint64(resultado.CantidadDisponible))
	c.booleano("respuesta_cantidad_exacta", resultado.CantidadExacta)
	return c.bytes()
}

func materialReciboOrdenBolsa(
	comando ComandoPrepararOrdenBolsa,
	recibo ReciboOrdenBolsa,
) []byte {
	c := canonicoRespuestaBolsa(
		"recibo-orden-durable", comando.Contexto,
		recibo.OperacionRef, recibo.OrganizacionRef, recibo.ExpedienteRef,
		recibo.VersionExpediente, recibo.CorrelacionRef,
		recibo.Necesidad, recibo.Resultado, recibo.Procedencia,
	)
	c.referencia("comando_necesidad", comando.Necesidad)
	c.referencia("comando_bolsa", comando.Bolsa)
	c.referencia("comando_politica", comando.Politica)
	c.entero("comando_maximo_posiciones", uint64(comando.MaximoPosiciones))
	c.referencia("recibo_bolsa", recibo.Bolsa)
	c.referencia("recibo_politica", recibo.Politica)
	c.booleano("recibo_orden_generada", recibo.OrdenGenerada)
	c.booleano("recibo_orden_completa", recibo.OrdenCompleta)
	c.referencia("recibo_orden", recibo.Orden)
	c.entero("recibo_total_posiciones", uint64(recibo.TotalPosiciones))
	c.campo("recibo_ref", recibo.ReciboRef)
	c.campo("auditoria_ref", recibo.AuditoriaRef)
	c.campo("evento_ref", recibo.EventoRef)
	c.instante("confirmada_en", recibo.ConfirmadaEn)
	return c.bytes()
}

func materialReciboLlamamientoBolsa(
	comando ComandoSolicitarLlamamientoBolsa,
	recibo ReciboSolicitudLlamamientoBolsa,
) []byte {
	datosComando, err := comando.datosCanonicos()
	if err != nil {
		return nil
	}
	c := canonicoRespuestaBolsa(
		"recibo-llamamiento-durable", datosComando.Contexto,
		recibo.OperacionRef, recibo.OrganizacionRef, recibo.ExpedienteRef,
		recibo.VersionExpediente, recibo.CorrelacionRef,
		recibo.Necesidad, recibo.Resultado, recibo.Procedencia,
	)
	c.referencia("comando_necesidad", datosComando.Necesidad)
	c.referencia("comando_bolsa", datosComando.Bolsa)
	c.referencia("comando_orden", datosComando.Orden)
	c.referencia("comando_politica", datosComando.Politica)
	c.entero("comando_total_posiciones", uint64(datosComando.TotalPosicionesOrden))
	c.entero("comando_maxima_posicion", uint64(datosComando.MaximaPosicionEvaluable))
	c.campo("comando_huella_recibo_orden", datosComando.HuellaReciboOrden)
	c.referencia("recibo_bolsa", recibo.Bolsa)
	c.referencia("recibo_orden", recibo.Orden)
	c.referencia("recibo_politica", recibo.Politica)
	c.booleano("recibo_propuesta_generada", recibo.PropuestaGenerada)
	c.referencia("recibo_propuesta", recibo.Propuesta)
	c.campo("llamamiento_ref", recibo.LlamamientoRef)
	c.campo("seleccion_ref_seudonimizada", recibo.SeleccionRef)
	c.referencia("retencion_seleccion", recibo.RetencionSeleccion)
	c.entero("orden_seleccionado", uint64(recibo.OrdenSeleccionado))
	c.campo("recibo_ref", recibo.ReciboRef)
	c.campo("auditoria_ref", recibo.AuditoriaRef)
	c.campo("evento_ref", recibo.EventoRef)
	c.instante("confirmada_en", recibo.ConfirmadaEn)
	return c.bytes()
}

func canonicoRespuestaBolsa(
	tipo string,
	contexto ContextoPeticionIntegracionBolsa,
	operacionRef string,
	organizacionRef string,
	expedienteRef string,
	versionExpediente uint64,
	correlacionRef string,
	necesidad ReferenciaVersionadaIntegracionBolsa,
	resultado ReferenciaVersionadaIntegracionBolsa,
	procedencia ProcedenciaIntegracionBolsa,
) *constructorCanonicoBolsa {
	c := nuevoCanonicoBolsa(tipo)
	c.contexto(contexto)
	c.campo("respuesta_operacion_ref", operacionRef)
	c.campo("respuesta_organizacion_ref", organizacionRef)
	c.campo("respuesta_expediente_ref", expedienteRef)
	c.entero("respuesta_version_expediente", versionExpediente)
	c.campo("respuesta_correlacion_ref", correlacionRef)
	c.referencia("respuesta_necesidad", necesidad)
	c.referencia("respuesta_resultado", resultado)
	c.procedencia(procedencia)
	return c
}
