package ports

import (
	"context"
	"time"
)

func materialSolicitudDisponibilidadBolsa(
	solicitud SolicitudDisponibilidadBolsa,
) []byte {
	c := nuevoCanonicoBolsa("peticion-disponibilidad")
	c.contextoCapacidad(solicitud.Contexto)
	c.referencia("necesidad", solicitud.Necesidad)
	c.campo("categoria_ref", solicitud.CategoriaRef)
	c.entero("maximo_resultados", uint64(solicitud.MaximoResultados))
	return c.bytes()
}

func materialComandoOrdenBolsa(comando ComandoPrepararOrdenBolsa) []byte {
	c := nuevoCanonicoBolsa("peticion-orden")
	c.contextoCapacidad(comando.Contexto)
	c.referencia("necesidad", comando.Necesidad)
	c.referencia("bolsa", comando.Bolsa)
	c.referencia("politica", comando.Politica)
	c.entero("maximo_posiciones", uint64(comando.MaximoPosiciones))
	return c.bytes()
}

func materialComandoLlamamientoBolsa(
	comando ComandoSolicitarLlamamientoBolsa,
) []byte {
	datos, err := comando.datosCanonicos()
	if err != nil {
		return nil
	}
	c := nuevoCanonicoBolsa("peticion-llamamiento")
	c.contextoCapacidad(datos.Contexto)
	c.referencia("necesidad", datos.Necesidad)
	c.referencia("bolsa", datos.Bolsa)
	c.referencia("orden", datos.Orden)
	c.referencia("politica", datos.Politica)
	c.entero("total_posiciones_orden", uint64(datos.TotalPosicionesOrden))
	c.entero("maxima_posicion_evaluable", uint64(datos.MaximaPosicionEvaluable))
	c.campo("huella_recibo_orden", datos.HuellaReciboOrden)
	return c.bytes()
}

func (v *VerificadorEvidenciaIntegracionBolsa) VerificarDisponibilidad(
	ctx context.Context,
	solicitud SolicitudDisponibilidadBolsa,
	resultado ResultadoDisponibilidadBolsa,
	instante time.Time,
) (
	ComprobanteEvidenciaIntegracionBolsa,
	EvidenciaDurableIntegracionBolsa,
	error,
) {
	if resultado.ValidarParaEn(solicitud, instante) != nil {
		return ComprobanteEvidenciaIntegracionBolsa{},
			EvidenciaDurableIntegracionBolsa{},
			ErrRespuestaBolsaNoConfiable
	}
	datos, _ := solicitud.Contexto.datosDurables()
	return v.verificarFresco(
		ctx,
		"disponibilidad_volatil",
		datos.OperacionRef,
		materialSolicitudDisponibilidadBolsa(solicitud),
		materialDisponibilidadBolsa(solicitud, resultado),
		resultado.Procedencia,
		instante,
	)
}

func (v *VerificadorEvidenciaIntegracionBolsa) VerificarReciboOrden(
	ctx context.Context,
	comando ComandoPrepararOrdenBolsa,
	recibo ReciboOrdenBolsa,
	instante time.Time,
) (
	ComprobanteEvidenciaIntegracionBolsa,
	EvidenciaDurableIntegracionBolsa,
	error,
) {
	if recibo.ValidarParaEn(comando, instante) != nil {
		return ComprobanteEvidenciaIntegracionBolsa{},
			EvidenciaDurableIntegracionBolsa{},
			ErrRespuestaBolsaNoConfiable
	}
	datos, _ := comando.Contexto.datosDurables()
	return v.verificarFresco(
		ctx,
		"recibo_orden",
		datos.OperacionRef,
		materialComandoOrdenBolsa(comando),
		materialReciboOrdenBolsa(comando, recibo),
		recibo.Procedencia,
		instante,
	)
}

func (v *VerificadorEvidenciaIntegracionBolsa) VerificarReciboLlamamiento(
	ctx context.Context,
	comando ComandoSolicitarLlamamientoBolsa,
	recibo ReciboSolicitudLlamamientoBolsa,
	instante time.Time,
) (
	ComprobanteEvidenciaIntegracionBolsa,
	EvidenciaDurableIntegracionBolsa,
	error,
) {
	if recibo.ValidarParaEn(comando, instante) != nil {
		return ComprobanteEvidenciaIntegracionBolsa{},
			EvidenciaDurableIntegracionBolsa{},
			ErrRespuestaBolsaNoConfiable
	}
	datosComando, err := comando.DatosEn(instante)
	if err != nil {
		return ComprobanteEvidenciaIntegracionBolsa{},
			EvidenciaDurableIntegracionBolsa{},
			ErrRespuestaBolsaNoConfiable
	}
	datosContexto, _ := datosComando.Contexto.datosDurables()
	return v.verificarFresco(
		ctx,
		"recibo_llamamiento",
		datosContexto.OperacionRef,
		materialComandoLlamamientoBolsa(comando),
		materialReciboLlamamientoBolsa(comando, recibo),
		recibo.Procedencia,
		instante,
	)
}

func (v *VerificadorEvidenciaIntegracionBolsa) reautenticarReciboOrden(
	ctx context.Context,
	comando ComandoPrepararOrdenBolsa,
	recibo ReciboOrdenBolsa,
	evidencia EvidenciaDurableIntegracionBolsa,
	instante time.Time,
) (ComprobanteEvidenciaIntegracionBolsa, error) {
	if recibo.ValidarDurablePara(comando) != nil ||
		evidencia.TipoMaterial != "recibo_orden" {
		return ComprobanteEvidenciaIntegracionBolsa{}, ErrRespuestaBolsaNoConfiable
	}
	datosContexto, _ := comando.Contexto.datosDurables()
	esperada := nuevaEvidenciaDurableBolsa(
		"recibo_orden",
		datosContexto.OperacionRef,
		materialComandoOrdenBolsa(comando),
		materialReciboOrdenBolsa(comando, recibo),
		recibo.Procedencia,
	)
	if !evidenciasDurablesBolsaIguales(evidencia, esperada) {
		return ComprobanteEvidenciaIntegracionBolsa{}, ErrEvidenciaBolsaNoAutenticada
	}
	return v.reautenticar(
		ctx,
		evidencia,
		materialComandoOrdenBolsa(comando),
		materialReciboOrdenBolsa(comando, recibo),
		instante,
	)
}

func (v *VerificadorEvidenciaIntegracionBolsa) reautenticarReciboLlamamiento(
	ctx context.Context,
	comando ComandoSolicitarLlamamientoBolsa,
	recibo ReciboSolicitudLlamamientoBolsa,
	evidencia EvidenciaDurableIntegracionBolsa,
	instante time.Time,
) (ComprobanteEvidenciaIntegracionBolsa, error) {
	if recibo.ValidarDurablePara(comando) != nil ||
		evidencia.TipoMaterial != "recibo_llamamiento" {
		return ComprobanteEvidenciaIntegracionBolsa{}, ErrRespuestaBolsaNoConfiable
	}
	datosComando, _ := comando.datosCanonicos()
	datosContexto, _ := datosComando.Contexto.datosDurables()
	esperada := nuevaEvidenciaDurableBolsa(
		"recibo_llamamiento",
		datosContexto.OperacionRef,
		materialComandoLlamamientoBolsa(comando),
		materialReciboLlamamientoBolsa(comando, recibo),
		recibo.Procedencia,
	)
	if !evidenciasDurablesBolsaIguales(evidencia, esperada) {
		return ComprobanteEvidenciaIntegracionBolsa{}, ErrEvidenciaBolsaNoAutenticada
	}
	return v.reautenticar(
		ctx,
		evidencia,
		materialComandoLlamamientoBolsa(comando),
		materialReciboLlamamientoBolsa(comando, recibo),
		instante,
	)
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
	c.referencia("recibo_accion_llamamiento", recibo.AccionLlamamiento)
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
	c.referencia("recibo_accion_evento", recibo.AccionEvento)
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
	c.contextoCapacidad(contexto)
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
