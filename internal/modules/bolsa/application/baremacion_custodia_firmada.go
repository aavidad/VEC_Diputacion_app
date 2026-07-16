package application

import (
	"context"
	"errors"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

func (s *ServicioBaremacion) custodiarBinarioFirmado(
	ctx context.Context,
	orden OrdenFinalizarFirmaBaremacion,
	contenido dominiobolsa.ContenidoDecisionTecnica,
	artefacto puertosbolsa.ArtefactoFirma,
	autorizarFirma func(puertosbolsa.AccionOperacionBaremacion, puertosbolsa.ClaseRecursoOperacionBaremacion, string, string) (puertosbolsa.ContextoOperacionFirma, error),
	autorizarFirmaAlmacen func(puertosbolsa.AccionOperacionBaremacion, puertosbolsa.ClaseRecursoOperacionBaremacion, string, string, dominiovec.RecursoAutorizable) (puertosbolsa.ContextoOperacionFirma, error),
	_ map[string]struct{},
) (puertosbolsa.DocumentoFirmadoCustodiado, error) {
	seudonimo := orden.Firma.seudonimoFirmado
	efectoCustodiaRef := orden.Firma.efectoCustodiaRef
	huellaPlanCustodia, err := s.sellarPartesBaremacion(ctx, []string{
		"plan_custodia_documento_firmado_baremacion_v2", orden.OperacionCustodiaRef,
		orden.ClaveIdempotenciaCustodia, orden.CargaDocumentoFirmadoRef, seudonimo,
		efectoCustodiaRef, artefacto.DocumentoFirmadoRef, artefacto.FirmaRef,
		artefacto.HuellaDocumentoSHA256, s.clasificacion, contenido.CorrelacionRef,
	})
	if err != nil {
		return puertosbolsa.DocumentoFirmadoCustodiado{}, err
	}
	vinculosCustodia := puertosvec.VinculosOperacionAlmacen{
		OperacionRef: orden.OperacionCustodiaRef, CargaRef: orden.CargaDocumentoFirmadoRef,
		Clasificacion: s.clasificacion, SujetoSeudonimoHMAC: seudonimo,
		HuellaSolicitudHMAC: huellaPlanCustodia, EfectoRef: efectoCustodiaRef,
	}
	recursoCustodia := recursoAlmacenBaremacion(
		artefacto.DocumentoFirmadoRef, puertosbolsa.ClaseRecursoDocumentoFirmado,
		contenido.SujetoRef, vinculosCustodia,
	)
	contextoRecuperacion, err := autorizarFirma(
		puertosbolsa.AccionRecuperarBinarioFirmadoBaremacion,
		puertosbolsa.ClaseRecursoDocumentoFirmado, artefacto.DocumentoFirmadoRef, "recuperar_binario",
	)
	if err != nil {
		return puertosbolsa.DocumentoFirmadoCustodiado{}, err
	}
	contextoCustodia, err := autorizarFirmaAlmacen(
		puertosbolsa.AccionCustodiarDocumentoFirmadoBaremacion,
		puertosbolsa.ClaseRecursoDocumentoFirmado, artefacto.DocumentoFirmadoRef,
		"custodiar_binario", recursoCustodia,
	)
	if err != nil {
		return puertosbolsa.DocumentoFirmadoCustodiado{}, err
	}
	contextoRecuperacionBase := contextoRecuperacion.ContextoOperacionBaremacion
	contextoCustodiaBase := contextoCustodia.ContextoOperacionBaremacion
	referenciaRecuperacion := contextoRecuperacion.Proyeccion().AutorizacionRef
	referenciaCustodia := contextoCustodia.Proyeccion().AutorizacionRef
	if !contextoRecuperacionBase.MismoVinculoAutenticacionQue(contextoCustodiaBase) ||
		referenciaRecuperacion == referenciaCustodia {
		return puertosbolsa.DocumentoFirmadoCustodiado{}, errors.Join(
			dominiovec.ErrAutorizacionDenegada,
			ErrResultadoBaremacionNoConfiable,
		)
	}
	solicitudRecuperacion := puertosbolsa.SolicitudRecuperarBinarioFirmado{
		Contexto: contextoRecuperacion, DocumentoFirmadoRef: artefacto.DocumentoFirmadoRef,
		HuellaDocumentoSHA256: artefacto.HuellaDocumentoSHA256, LimiteBytes: limiteDocumentoFirmadoBaremacion,
	}
	binario, err := s.recuperadorBinario.RecuperarBinarioFirmado(ctx, solicitudRecuperacion)
	if err != nil || binario.ValidarPara(solicitudRecuperacion) != nil {
		return puertosbolsa.DocumentoFirmadoCustodiado{}, errors.Join(ErrResultadoBaremacionNoConfiable, err)
	}
	defer binario.Contenido.Close()
	ahora, err := s.ahora()
	if err != nil {
		return puertosbolsa.DocumentoFirmadoCustodiado{}, err
	}
	retenidoHasta := ahora.Add(s.duracionRetencion).UTC()
	capacidades, err := s.almacen.Capacidades(ctx)
	if err != nil || capacidades.ConectorID != s.conectorAlmacen ||
		puertosvec.VerificarCapacidadesAlmacen(capacidades, puertosvec.RequisitosAlmacenObjetos{
			EscrituraEnFlujo: true, ReferenciasOpacas: true, IntegridadSHA256: true, Versionado: true,
			Retencion: true, BloqueoLegal: true, CifradoEnTransito: true, CifradoEnReposo: true,
			CifradoPorObjeto: true, PreservaObjetoOriginal: true,
		}) != nil || capacidades.TamanoMaximoObjeto < binario.Tamano {
		return puertosbolsa.DocumentoFirmadoCustodiado{}, errors.Join(ErrResultadoBaremacionNoConfiable, err)
	}
	contextoAlmacenCustodia, err := contextoCustodiaBase.CrearContextoAlmacenCustodiarDocumentoFirmado(
		vinculosCustodia,
	)
	if err != nil {
		return puertosbolsa.DocumentoFirmadoCustodiado{}, errors.Join(dominiovec.ErrAutorizacionDenegada, err)
	}
	lector := nuevoLectorFirmadoVerificado(binario.Contenido, binario.Tamano, binario.HuellaDocumentoSHA256)
	solicitudEscritura := puertosvec.SolicitudEscribirObjeto{
		Contexto:          contextoAlmacenCustodia,
		ClaveIdempotencia: orden.ClaveIdempotenciaCustodia, Zona: puertosvec.ZonaAlmacenAdmitida,
		MIME: binario.MIME, Tamano: binario.Tamano, HuellaSHA256: binario.HuellaDocumentoSHA256,
		Contenido: lector,
	}
	if err := solicitudEscritura.Validar(); err != nil {
		return puertosbolsa.DocumentoFirmadoCustodiado{}, err
	}
	escritura, err := s.almacen.Escribir(ctx, solicitudEscritura)
	errLector := lector.Verificar()
	errEscritura := escritura.ValidarEscritura(solicitudEscritura, capacidades)
	if err != nil || errLector != nil || errEscritura != nil {
		causa := errors.Join(ErrResultadoBaremacionNoConfiable, err, errLector, errEscritura)
		if err == nil || escritura.Validar() == nil {
			return puertosbolsa.DocumentoFirmadoCustodiado{}, &ErrorCustodiaBaremacionIncompleta{
				DecisionRef: contenido.ID, DocumentoRef: artefacto.DocumentoFirmadoRef,
				Escritura: escritura, Causa: causa,
			}
		}
		return puertosbolsa.DocumentoFirmadoCustodiado{}, causa
	}
	falloTrasEscritura := func(causa error, retencion *puertosvec.ResultadoOperacionObjeto) error {
		return &ErrorCustodiaBaremacionIncompleta{
			DecisionRef: contenido.ID, DocumentoRef: artefacto.DocumentoFirmadoRef,
			Escritura: escritura, Retencion: retencion, Causa: causa,
		}
	}
	efectoRetencionRef, err := s.generador.NuevaReferenciaEfectoAlmacen()
	if err != nil || !referenciaAplicacionBaremacionValida(efectoRetencionRef) {
		return puertosbolsa.DocumentoFirmadoCustodiado{}, falloTrasEscritura(
			errors.Join(ErrResultadoBaremacionNoConfiable, err), nil,
		)
	}
	operacionRetencionRef := orden.OperacionCustodiaRef + ":retencion"
	huellaPlanRetencion, err := s.sellarPartesBaremacion(ctx, []string{
		"plan_retencion_documento_firmado_baremacion_v2", operacionRetencionRef,
		orden.CargaDocumentoFirmadoRef, seudonimo, efectoRetencionRef,
		artefacto.DocumentoFirmadoRef, artefacto.FirmaRef, artefacto.HuellaDocumentoSHA256,
		escritura.Objeto.Objeto.Referencia, escritura.Objeto.Objeto.Version,
		s.politicaRetencion, retenidoHasta.Format(time.RFC3339Nano), contenido.CorrelacionRef,
	})
	if err != nil {
		return puertosbolsa.DocumentoFirmadoCustodiado{}, falloTrasEscritura(err, nil)
	}
	vinculosRetencion := puertosvec.VinculosOperacionAlmacen{
		OperacionRef: operacionRetencionRef, CargaRef: orden.CargaDocumentoFirmadoRef,
		Clasificacion: s.clasificacion, SujetoSeudonimoHMAC: seudonimo,
		HuellaSolicitudHMAC: huellaPlanRetencion, EfectoRef: efectoRetencionRef,
		ObjetoVinculado: escritura.Objeto.Objeto,
	}
	recursoRetencion := recursoAlmacenBaremacion(
		artefacto.DocumentoFirmadoRef, puertosbolsa.ClaseRecursoDocumentoFirmado,
		contenido.SujetoRef, vinculosRetencion,
	)
	contextoRetencion, err := autorizarFirmaAlmacen(
		puertosbolsa.AccionRetenerDocumentoFirmadoBaremacion,
		puertosbolsa.ClaseRecursoDocumentoFirmado, artefacto.DocumentoFirmadoRef,
		"retener_binario", recursoRetencion,
	)
	if err != nil {
		return puertosbolsa.DocumentoFirmadoCustodiado{}, falloTrasEscritura(
			errors.Join(dominiovec.ErrAutorizacionDenegada, err), nil,
		)
	}
	contextoRetencionBase := contextoRetencion.ContextoOperacionBaremacion
	referenciaRetencion := contextoRetencion.Proyeccion().AutorizacionRef
	if !contextoRecuperacionBase.MismoVinculoAutenticacionQue(contextoRetencionBase) ||
		!contextoCustodiaBase.MismoVinculoAutenticacionQue(contextoRetencionBase) ||
		referenciaRecuperacion == referenciaRetencion || referenciaCustodia == referenciaRetencion {
		return puertosbolsa.DocumentoFirmadoCustodiado{}, falloTrasEscritura(
			errors.Join(dominiovec.ErrAutorizacionDenegada, ErrResultadoBaremacionNoConfiable), nil,
		)
	}
	contextoAlmacenRetencion, err := contextoRetencionBase.CrearContextoAlmacenRetenerDocumentoFirmado(
		vinculosRetencion,
	)
	if err != nil {
		return puertosbolsa.DocumentoFirmadoCustodiado{}, falloTrasEscritura(
			errors.Join(dominiovec.ErrAutorizacionDenegada, err), nil,
		)
	}
	solicitudRetencion := puertosvec.SolicitudRetenerObjeto{
		Contexto: contextoAlmacenRetencion,
		Objeto:   escritura.Objeto.Objeto, PoliticaRef: s.politicaRetencion, Hasta: retenidoHasta,
	}
	retencion, err := s.almacen.AplicarRetencion(ctx, solicitudRetencion)
	errRetencion := retencion.ValidarRetencion(solicitudRetencion, escritura.Objeto)
	if err != nil || errRetencion != nil {
		return puertosbolsa.DocumentoFirmadoCustodiado{}, falloTrasEscritura(
			errors.Join(ErrResultadoBaremacionNoConfiable, err, errRetencion), &retencion,
		)
	}
	documento := puertosbolsa.DocumentoFirmadoCustodiado{
		DocumentoFirmadoRef: artefacto.DocumentoFirmadoRef, FirmaRef: artefacto.FirmaRef,
		HuellaDocumentoSHA256: artefacto.HuellaDocumentoSHA256, Objeto: retencion.Objeto,
		EvidenciaEscritura: escritura.Evidencia, EvidenciaRetencion: retencion.Evidencia,
		EvidenciaRecuperacionRef:          binario.EvidenciaRecuperacionRef,
		HuellaEvidenciaRecuperacionSHA256: binario.HuellaEvidenciaSHA256,
		PoliticaRetencionRef:              s.politicaRetencion, RetenidoHasta: retenidoHasta,
	}
	if err := documento.ValidarPara(artefacto, escritura, retencion); err != nil {
		return puertosbolsa.DocumentoFirmadoCustodiado{}, falloTrasEscritura(err, &retencion)
	}
	return documento, nil
}
