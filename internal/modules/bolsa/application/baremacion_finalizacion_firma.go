package application

import (
	"context"
	"errors"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// OrdenFinalizarFirmaBaremacion aporta solo claves de reintento y motivacion;
// no acepta artefactos, validaciones ni atestaciones declaradas por el cliente.
type OrdenFinalizarFirmaBaremacion struct {
	Actor                     ActorBaremacion
	Firma                     FirmaBaremacionPreparada
	OperacionRef              string
	ClaveIdempotenciaSello    string
	ClaveIdempotenciaAumento  string
	MotivoClaveConfirmacion   string
	MotivoConfirmacion        string
	OperacionCustodiaRef      string
	ClaveIdempotenciaCustodia string
	CargaDocumentoFirmadoRef  string
	ClaveIdempotenciaReserva  string
}

// ResultadoFinalizarFirmaBaremacion expone la decision inmutable y todas las
// capas verificadas que condujeron a su confirmacion transaccional.
type ResultadoFinalizarFirmaBaremacion struct {
	Decision            dominiobolsa.DecisionTecnica
	ValidacionInicial   puertosbolsa.ValidacionFirmaServidor
	SelloTiempo         *puertosbolsa.SelloTiempoFirma
	ValidacionTrasSello *puertosbolsa.ValidacionFirmaServidor
	Aumento             *puertosbolsa.ResultadoAumentoFirma
	ValidacionFinal     puertosbolsa.ValidacionFirmaServidor
	DocumentoFirmado    puertosbolsa.DocumentoFirmadoCustodiado
	Confirmacion        puertosbolsa.ResultadoConfirmarCambioBaremacion
}

// FinalizarFirma consulta el firmador, valida en servidor, aplica las capas
// exigidas por politica y confirma agregado, auditoria y outbox mediante el
// repositorio transaccional.
//
// BLOQUEANTE PRODUCTIVO: esta primera vertical exige que adopcion y firma
// pertenezcan al mismo actor. El doble control maker-checker necesita un flujo
// persistente separado, con asignacion, caducidad, sustitucion y firma propias;
// no debe simularse relajando esta vinculacion ni reutilizando autorizaciones.
func (s *ServicioBaremacion) FinalizarFirma(
	ctx context.Context,
	orden OrdenFinalizarFirmaBaremacion,
) (resultadoRetorno ResultadoFinalizarFirmaBaremacion, errRetorno error) {
	if validarFirmaPreparada(orden.Firma) != nil {
		return ResultadoFinalizarFirmaBaremacion{}, ErrOrdenBaremacionInvalida
	}
	if err := s.validarRevisionVigente(orden.Firma.decision.decision.revision.revision); err != nil {
		return ResultadoFinalizarFirmaBaremacion{}, err
	}
	if validarActorRevision(orden.Actor, orden.Firma.decision.decision.revision.revision) != nil ||
		!referenciaAplicacionBaremacionValida(orden.OperacionRef) ||
		!referenciaAplicacionBaremacionValida(orden.OperacionCustodiaRef) ||
		!referenciaAplicacionBaremacionValida(orden.ClaveIdempotenciaCustodia) ||
		!referenciaAplicacionBaremacionValida(orden.CargaDocumentoFirmadoRef) ||
		!referenciaAplicacionBaremacionValida(orden.ClaveIdempotenciaReserva) ||
		!claveAplicacionBaremacionValida(orden.MotivoClaveConfirmacion) ||
		!textoAplicacionBaremacionValido(orden.MotivoConfirmacion, 8000, true) {
		return ResultadoFinalizarFirmaBaremacion{}, ErrOrdenBaremacionInvalida
	}
	revision := orden.Firma.decision.decision.revision.revision
	contenido := orden.Firma.decision.decision.revision.contenido
	politica := orden.Firma.decision.decision.politica
	autorizaciones := referenciasAutorizacionPrevias(orden.Firma)
	proyeccionesFinales := make([]puertosbolsa.ProyeccionAutorizacionBaremacion, 0, 12)
	registrarAutorizacionFirma := func(
		capacidad puertosbolsa.ContextoOperacionBaremacion,
		sufijo string,
	) (puertosbolsa.ContextoOperacionFirma, error) {
		ref := capacidad.Proyeccion().AutorizacionRef
		if _, repetida := autorizaciones[ref]; repetida {
			return puertosbolsa.ContextoOperacionFirma{}, ErrResultadoBaremacionNoConfiable
		}
		autorizaciones[ref] = struct{}{}
		proyeccionesFinales = append(proyeccionesFinales, capacidad.Proyeccion())
		return puertosbolsa.ContextoOperacionFirma{
			ContextoOperacionBaremacion: capacidad, OperacionRef: orden.OperacionRef + ":" + sufijo,
		}, nil
	}
	autorizarFirma := func(accion puertosbolsa.AccionOperacionBaremacion, clase puertosbolsa.ClaseRecursoOperacionBaremacion, recurso, sufijo string) (puertosbolsa.ContextoOperacionFirma, error) {
		capacidad, err := s.autorizarRevision(ctx, orden.Actor, revision, accion, clase, recurso, contenido.SujetoRef,
			contenido.FinalidadClave, contenido.CorrelacionRef)
		if err != nil {
			return puertosbolsa.ContextoOperacionFirma{}, err
		}
		return registrarAutorizacionFirma(capacidad, sufijo)
	}
	autorizarFirmaAlmacen := func(
		accion puertosbolsa.AccionOperacionBaremacion,
		clase puertosbolsa.ClaseRecursoOperacionBaremacion,
		recurso, sufijo string,
		recursoAutorizable dominiovec.RecursoAutorizable,
	) (puertosbolsa.ContextoOperacionFirma, error) {
		capacidad, err := s.autorizarAlmacenRevision(
			ctx, orden.Actor, revision, accion, clase, recurso, contenido.SujetoRef,
			contenido.FinalidadClave, contenido.CorrelacionRef, recursoAutorizable,
		)
		if err != nil {
			return puertosbolsa.ContextoOperacionFirma{}, err
		}
		return registrarAutorizacionFirma(capacidad, sufijo)
	}
	contextoConsulta, err := autorizarFirma(puertosbolsa.AccionConsultarFirmaDecisionBaremacion,
		puertosbolsa.ClaseRecursoSesionFirma, orden.Firma.sesion.SesionRef, "consulta")
	if err != nil {
		return ResultadoFinalizarFirmaBaremacion{}, err
	}
	solicitudConsulta := puertosbolsa.SolicitudConsultarFirmaInteractiva{
		Contexto: contextoConsulta, SesionRef: orden.Firma.sesion.SesionRef,
		Documento: orden.Firma.decision.documento, HuellaContenidoSHA256: orden.Firma.decision.documento.HuellaContenidoSHA256,
		PoliticaFirmaRef: politica.Referencia, PoliticaFirmaVersion: politica.Version,
		HuellaPoliticaSHA256: politica.HuellaSHA256, FirmanteRef: contenido.DecisorRef,
		PerfilFirmanteClave: contenido.PerfilDecisorClave,
	}
	consulta, err := s.firmador.ConsultarFirmaInteractiva(ctx, solicitudConsulta)
	if err != nil || consulta.ValidarPara(solicitudConsulta) != nil {
		return ResultadoFinalizarFirmaBaremacion{}, errors.Join(ErrResultadoBaremacionNoConfiable, err)
	}
	if consulta.Estado != puertosbolsa.EstadoSesionFirmaCompletada || consulta.Artefacto == nil {
		return ResultadoFinalizarFirmaBaremacion{}, ErrFirmaBaremacionNoCompletada
	}
	artefacto := *consulta.Artefacto
	if artefacto.ValidarPara(orden.Firma.solicitud, orden.Firma.sesion) != nil {
		return ResultadoFinalizarFirmaBaremacion{}, ErrResultadoBaremacionNoConfiable
	}
	contextoValidacion, err := autorizarFirma(puertosbolsa.AccionValidarFirmaDecisionBaremacion,
		puertosbolsa.ClaseRecursoArtefactoFirma, artefacto.FirmaRef, "validacion_inicial")
	if err != nil {
		return ResultadoFinalizarFirmaBaremacion{}, err
	}
	ahora, err := s.ahora()
	if err != nil {
		return ResultadoFinalizarFirmaBaremacion{}, err
	}
	solicitudValidacion := puertosbolsa.SolicitudValidarFirmaServidor{
		Contexto: contextoValidacion, Artefacto: artefacto, Politica: politica,
		FirmanteEsperadoRef: contenido.DecisorRef, PerfilEsperadoClave: contenido.PerfilDecisorClave,
		PerfilFirmaEsperadoClave: puertosbolsa.PerfilFirmaPAdESBaselineB,
		SolicitadaEn:             ahora,
	}
	validacionInicial, err := s.validadorFirma.ValidarFirmaServidor(ctx, solicitudValidacion)
	if err != nil || validacionInicial.ValidarPara(solicitudValidacion) != nil ||
		!validacionInicial.AptaParaPerfil(politica, puertosbolsa.PerfilFirmaPAdESBaselineB) {
		return ResultadoFinalizarFirmaBaremacion{}, errors.Join(ErrResultadoBaremacionNoConfiable, err)
	}
	artefactoActual := artefacto
	validacionActual := validacionInicial
	var sello *puertosbolsa.SelloTiempoFirma
	var validacionTrasSello *puertosbolsa.ValidacionFirmaServidor
	if politica.RequiereSelloTiempo {
		if !referenciaAplicacionBaremacionValida(orden.ClaveIdempotenciaSello) {
			return ResultadoFinalizarFirmaBaremacion{}, ErrOrdenBaremacionInvalida
		}
		contextoSello, err := autorizarFirma(puertosbolsa.AccionSellarTiempoDecisionBaremacion,
			puertosbolsa.ClaseRecursoArtefactoFirma, artefacto.FirmaRef, "sello_tiempo")
		if err != nil {
			return ResultadoFinalizarFirmaBaremacion{}, err
		}
		ahoraSello, err := s.ahora()
		if err != nil {
			return ResultadoFinalizarFirmaBaremacion{}, err
		}
		solicitudSello := puertosbolsa.SolicitudSellarTiempoFirma{
			Contexto: contextoSello, ClaveIdempotencia: orden.ClaveIdempotenciaSello,
			ArtefactoOrigen: artefacto, ValidacionOrigen: validacionInicial,
			Politica: politica, SolicitadaEn: ahoraSello,
		}
		resultadoSello, err := s.selladorTiempo.SellarTiempoFirma(ctx, solicitudSello)
		if err != nil || resultadoSello.ValidarPara(solicitudSello) != nil {
			return ResultadoFinalizarFirmaBaremacion{}, errors.Join(ErrResultadoBaremacionNoConfiable, err)
		}
		sello = &resultadoSello
		artefactoActual = resultadoSello.ArtefactoSellado
		contextoValidacionT, err := autorizarFirma(puertosbolsa.AccionValidarFirmaDecisionBaremacion,
			puertosbolsa.ClaseRecursoArtefactoFirma, artefactoActual.FirmaRef, "validacion_tras_sello")
		if err != nil {
			return ResultadoFinalizarFirmaBaremacion{}, err
		}
		ahoraValidacionT, err := s.ahora()
		if err != nil {
			return ResultadoFinalizarFirmaBaremacion{}, err
		}
		solicitudValidacionT := puertosbolsa.SolicitudValidarFirmaServidor{
			Contexto: contextoValidacionT, Artefacto: artefactoActual, Politica: politica,
			FirmanteEsperadoRef: contenido.DecisorRef, PerfilEsperadoClave: contenido.PerfilDecisorClave,
			PerfilFirmaEsperadoClave:        puertosbolsa.PerfilFirmaPAdESBaselineT,
			SelloTiempoEsperadoRef:          resultadoSello.SelloTiempoRef,
			HuellaSelloTiempoEsperadaSHA256: resultadoSello.HuellaSelloTiempoSHA256,
			SolicitadaEn:                    ahoraValidacionT,
		}
		resultadoValidacionT, err := s.validadorFirma.ValidarFirmaServidor(ctx, solicitudValidacionT)
		if err != nil || resultadoValidacionT.ValidarPara(solicitudValidacionT) != nil ||
			!resultadoValidacionT.AptaParaPerfil(politica, puertosbolsa.PerfilFirmaPAdESBaselineT) {
			return ResultadoFinalizarFirmaBaremacion{}, errors.Join(ErrResultadoBaremacionNoConfiable, err)
		}
		validacionTrasSello = &resultadoValidacionT
		validacionActual = resultadoValidacionT
	} else if orden.ClaveIdempotenciaSello != "" {
		return ResultadoFinalizarFirmaBaremacion{}, ErrOrdenBaremacionInvalida
	}
	var aumento *puertosbolsa.ResultadoAumentoFirma
	if politica.RequiereAumentoLongevidad {
		if !referenciaAplicacionBaremacionValida(orden.ClaveIdempotenciaAumento) {
			return ResultadoFinalizarFirmaBaremacion{}, ErrOrdenBaremacionInvalida
		}
		contextoAumento, err := autorizarFirma(puertosbolsa.AccionAumentarFirmaDecisionBaremacion,
			puertosbolsa.ClaseRecursoArtefactoFirma, artefactoActual.FirmaRef, "aumento")
		if err != nil {
			return ResultadoFinalizarFirmaBaremacion{}, err
		}
		ahoraAumento, err := s.ahora()
		if err != nil {
			return ResultadoFinalizarFirmaBaremacion{}, err
		}
		solicitudAumento := puertosbolsa.SolicitudAumentarFirma{
			Contexto: contextoAumento, ClaveIdempotencia: orden.ClaveIdempotenciaAumento,
			Artefacto: artefactoActual, Validacion: validacionActual, SelloTiempo: sello,
			Politica: politica, SolicitadaEn: ahoraAumento,
		}
		resultadoAumento, err := s.aumentadorFirma.AumentarFirma(ctx, solicitudAumento)
		if err != nil || resultadoAumento.ValidarPara(solicitudAumento) != nil {
			return ResultadoFinalizarFirmaBaremacion{}, errors.Join(ErrResultadoBaremacionNoConfiable, err)
		}
		aumento = &resultadoAumento
		artefactoActual = resultadoAumento.Artefacto
		contextoValidacionFinal, err := autorizarFirma(puertosbolsa.AccionValidarFirmaDecisionBaremacion,
			puertosbolsa.ClaseRecursoArtefactoFirma, artefactoActual.FirmaRef, "validacion_final")
		if err != nil {
			return ResultadoFinalizarFirmaBaremacion{}, err
		}
		ahoraValidacionFinal, err := s.ahora()
		if err != nil {
			return ResultadoFinalizarFirmaBaremacion{}, err
		}
		solicitudFinal := puertosbolsa.SolicitudValidarFirmaServidor{
			Contexto: contextoValidacionFinal, Artefacto: artefactoActual, Politica: politica,
			FirmanteEsperadoRef: contenido.DecisorRef, PerfilEsperadoClave: contenido.PerfilDecisorClave,
			PerfilFirmaEsperadoClave:              puertosbolsa.PerfilFirmaPAdESBaselineLTA,
			SelloTiempoEsperadoRef:                sello.SelloTiempoRef,
			HuellaSelloTiempoEsperadaSHA256:       sello.HuellaSelloTiempoSHA256,
			AumentoLongevidadEsperadoRef:          resultadoAumento.EvidenciaAumentoRef,
			HuellaAumentoLongevidadEsperadaSHA256: resultadoAumento.HuellaEvidenciaSHA256,
			SolicitadaEn:                          ahoraValidacionFinal,
		}
		validacionActual, err = s.validadorFirma.ValidarFirmaServidor(ctx, solicitudFinal)
		if err != nil || validacionActual.ValidarPara(solicitudFinal) != nil || !validacionActual.AptaParaPolitica(politica) {
			return ResultadoFinalizarFirmaBaremacion{}, errors.Join(ErrResultadoBaremacionNoConfiable, err)
		}
	} else if orden.ClaveIdempotenciaAumento != "" {
		return ResultadoFinalizarFirmaBaremacion{}, ErrOrdenBaremacionInvalida
	}
	validacionFinal := validacionActual
	artefactoFinal := artefactoActual
	documentoFirmado, err := s.custodiarBinarioFirmado(
		ctx, orden, contenido, artefactoFinal, autorizarFirma, autorizarFirmaAlmacen, autorizaciones,
	)
	if err != nil {
		return ResultadoFinalizarFirmaBaremacion{}, err
	}
	autorizacionReserva, err := s.autorizarRevision(ctx, orden.Actor, revision, puertosbolsa.AccionReservarDecisionBaremacion,
		puertosbolsa.ClaseRecursoBaremacion, contenido.BaremacionMeritoRef, contenido.SujetoRef,
		contenido.FinalidadClave, contenido.CorrelacionRef)
	if err != nil {
		return ResultadoFinalizarFirmaBaremacion{}, &ErrorDocumentoFirmadoHuerfano{DecisionRef: contenido.ID, Documento: documentoFirmado, Causa: err}
	}
	if _, repetida := autorizaciones[autorizacionReserva.Proyeccion().AutorizacionRef]; repetida {
		return ResultadoFinalizarFirmaBaremacion{}, &ErrorDocumentoFirmadoHuerfano{
			DecisionRef: contenido.ID, Documento: documentoFirmado, Causa: ErrResultadoBaremacionNoConfiable,
		}
	}
	autorizaciones[autorizacionReserva.Proyeccion().AutorizacionRef] = struct{}{}
	proyeccionesFinales = append(proyeccionesFinales, autorizacionReserva.Proyeccion())
	reservadaEn, err := s.ahora()
	if err != nil || reservadaEn.Before(validacionFinal.ValidadaEn) {
		return ResultadoFinalizarFirmaBaremacion{}, &ErrorDocumentoFirmadoHuerfano{DecisionRef: contenido.ID, Documento: documentoFirmado, Causa: errors.Join(ErrResultadoBaremacionNoConfiable, err)}
	}
	referenciaVersion := orden.Firma.decision.decision.revision.revision.version.Referencia
	solicitudReserva := puertosbolsa.SolicitudReservarCambioBaremacion{
		Contexto: autorizacionReserva, Clase: puertosbolsa.ClaseCambioIncorporarDecision,
		ClaveIdempotencia: orden.ClaveIdempotenciaReserva, BaremacionMeritoRef: contenido.BaremacionMeritoRef,
		VersionEsperada: &referenciaVersion, HuellaSolicitudHMAC: hmacBaremacionPendiente,
		SolicitadaEn: reservadaEn, ExpiraEn: reservadaEn.Add(s.duracionReserva),
	}
	solicitudReserva, err = s.sellarReserva(ctx, solicitudReserva)
	if err != nil {
		return ResultadoFinalizarFirmaBaremacion{}, &ErrorDocumentoFirmadoHuerfano{DecisionRef: contenido.ID, Documento: documentoFirmado, Causa: err}
	}
	reserva, err := s.repositorio.ReservarCambio(ctx, solicitudReserva)
	if err != nil || reserva.ValidarPara(solicitudReserva) != nil || reserva.Repetida {
		return ResultadoFinalizarFirmaBaremacion{}, &ErrorDocumentoFirmadoHuerfano{
			DecisionRef: contenido.ID, Documento: documentoFirmado,
			Causa: errors.Join(puertosbolsa.ErrVersionBaremacionConflicto, err),
		}
	}
	confirmacionInvocada := false
	defer func() {
		if confirmacionInvocada {
			return
		}
		if errRetorno == nil {
			resultadoRetorno = ResultadoFinalizarFirmaBaremacion{}
			errRetorno = &ErrorDocumentoFirmadoHuerfano{
				DecisionRef: contenido.ID, Documento: documentoFirmado,
				Causa: ErrResultadoBaremacionNoConfiable,
			}
		}
		errAbandono := s.abandonarReservaAntesDeConfirmar(
			ctx, orden.Actor, revision, contenido, solicitudReserva, reserva, autorizaciones,
		)
		if errAbandono == nil {
			return
		}
		if huerfano, ok := errRetorno.(*ErrorDocumentoFirmadoHuerfano); ok {
			huerfano.Causa = errors.Join(huerfano.Causa, errAbandono)
			return
		}
		errRetorno = errors.Join(errRetorno, errAbandono)
	}()
	autorizacionConfirmacion, err := s.autorizarRevision(ctx, orden.Actor, revision, puertosbolsa.AccionConfirmarDecisionBaremacion,
		puertosbolsa.ClaseRecursoBaremacion, contenido.BaremacionMeritoRef, contenido.SujetoRef,
		contenido.FinalidadClave, contenido.CorrelacionRef)
	if err != nil {
		return ResultadoFinalizarFirmaBaremacion{}, &ErrorDocumentoFirmadoHuerfano{DecisionRef: contenido.ID, Documento: documentoFirmado, Causa: err}
	}
	if _, repetida := autorizaciones[autorizacionConfirmacion.Proyeccion().AutorizacionRef]; repetida {
		return ResultadoFinalizarFirmaBaremacion{}, &ErrorDocumentoFirmadoHuerfano{
			DecisionRef: contenido.ID, Documento: documentoFirmado, Causa: ErrResultadoBaremacionNoConfiable,
		}
	}
	autorizaciones[autorizacionConfirmacion.Proyeccion().AutorizacionRef] = struct{}{}
	proyeccionesFinales = append(proyeccionesFinales, autorizacionConfirmacion.Proyeccion())
	confirmadaEn, err := s.ahora()
	if err != nil || confirmadaEn.Before(validacionFinal.ValidadaEn) || !confirmadaEn.Before(solicitudReserva.ExpiraEn) {
		return ResultadoFinalizarFirmaBaremacion{}, &ErrorDocumentoFirmadoHuerfano{DecisionRef: contenido.ID, Documento: documentoFirmado, Causa: errors.Join(ErrResultadoBaremacionNoConfiable, err)}
	}
	manifiesto, err := s.construirManifiestoProbatorio(
		ctx, orden.Firma, contenido, consulta, validacionInicial, sello, validacionTrasSello, aumento,
		validacionFinal, documentoFirmado, proyeccionesFinales, confirmadaEn,
	)
	if err != nil || manifiesto.ValidarPara(referenciaVersion, contenido) != nil {
		return ResultadoFinalizarFirmaBaremacion{}, &ErrorDocumentoFirmadoHuerfano{
			DecisionRef: contenido.ID, Documento: documentoFirmado,
			Causa: errors.Join(ErrResultadoBaremacionNoConfiable, err),
		}
	}
	firma, err := puertosbolsa.ConstituirFirmaDecisionConfiable(
		contenido, politica, artefacto, validacionInicial, sello, validacionTrasSello,
		aumento, validacionFinal, documentoFirmado, manifiesto,
	)
	if err != nil {
		return ResultadoFinalizarFirmaBaremacion{}, &ErrorDocumentoFirmadoHuerfano{DecisionRef: contenido.ID, Documento: documentoFirmado, Causa: err}
	}
	decision, err := dominiobolsa.ConstituirDecisionFirmada(contenido, firma)
	if err != nil {
		return ResultadoFinalizarFirmaBaremacion{}, &ErrorDocumentoFirmadoHuerfano{DecisionRef: contenido.ID, Documento: documentoFirmado, Causa: err}
	}
	agregado, err := orden.Firma.decision.decision.revision.revision.version.Agregado.IncorporarDecision(decision)
	if err != nil {
		return ResultadoFinalizarFirmaBaremacion{}, &ErrorDocumentoFirmadoHuerfano{DecisionRef: contenido.ID, Documento: documentoFirmado, Causa: err}
	}
	solicitudConfirmacion := puertosbolsa.SolicitudConfirmarCambioBaremacion{
		Contexto: autorizacionConfirmacion, Token: reserva.Token,
		Clase: puertosbolsa.ClaseCambioIncorporarDecision, VersionEsperada: &referenciaVersion,
		HuellaSolicitudHMAC: hmacBaremacionPendiente, Agregado: agregado, Manifiesto: &manifiesto,
		Trazabilidad: puertosbolsa.TrazabilidadCambioBaremacion{
			MotivoClave: orden.MotivoClaveConfirmacion, Motivo: orden.MotivoConfirmacion,
		},
		ConfirmadaEn: confirmadaEn,
	}
	solicitudConfirmacion, err = s.sellarConfirmacion(ctx, solicitudConfirmacion)
	if err != nil {
		return ResultadoFinalizarFirmaBaremacion{}, &ErrorDocumentoFirmadoHuerfano{DecisionRef: contenido.ID, Documento: documentoFirmado, Causa: err}
	}
	if err := ctx.Err(); err != nil {
		return ResultadoFinalizarFirmaBaremacion{}, &ErrorDocumentoFirmadoHuerfano{
			DecisionRef: contenido.ID, Documento: documentoFirmado, Causa: err,
		}
	}
	// Desde esta asignacion el COMMIT puede haberse aplicado aunque el adaptador
	// devuelva error o no llegue a devolver. Ya no es seguro abandonar la reserva.
	confirmacionInvocada = true
	confirmacion, err := s.repositorio.ConfirmarCambio(ctx, solicitudConfirmacion)
	if errDesenlace := clasificarDesenlaceConfirmacionBaremacion(
		confirmacion, solicitudConfirmacion, err,
	); errDesenlace != nil {
		return ResultadoFinalizarFirmaBaremacion{}, &ErrorDocumentoFirmadoHuerfano{
			DecisionRef: contenido.ID, Documento: documentoFirmado, Causa: errDesenlace,
		}
	}
	return ResultadoFinalizarFirmaBaremacion{
		Decision: decision, ValidacionInicial: validacionInicial, SelloTiempo: sello,
		ValidacionTrasSello: validacionTrasSello, Aumento: aumento, ValidacionFinal: validacionFinal,
		DocumentoFirmado: documentoFirmado, Confirmacion: confirmacion,
	}, nil
}
