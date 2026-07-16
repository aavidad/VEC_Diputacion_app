package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"hash"
	"io"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

func (s *ServicioBaremacion) construirManifiestoProbatorio(
	ctx context.Context,
	firmaPreparada FirmaBaremacionPreparada,
	contenido dominiobolsa.ContenidoDecisionTecnica,
	consulta puertosbolsa.ConsultaFirmaInteractiva,
	validacionInicial puertosbolsa.ValidacionFirmaServidor,
	sello *puertosbolsa.SelloTiempoFirma,
	validacionTrasSello *puertosbolsa.ValidacionFirmaServidor,
	aumento *puertosbolsa.ResultadoAumentoFirma,
	validacionFinal puertosbolsa.ValidacionFirmaServidor,
	documentoFirmado puertosbolsa.DocumentoFirmadoCustodiado,
	proyeccionesFinales []puertosbolsa.ProyeccionAutorizacionBaremacion,
	creadoEn time.Time,
) (puertosbolsa.ManifiestoProbatorioBaremacion, error) {
	autorizaciones, err := autorizacionesPreviasManifiestoBaremacion(firmaPreparada)
	if err != nil {
		return puertosbolsa.ManifiestoProbatorioBaremacion{}, err
	}
	vistas := make(map[string]struct{}, len(autorizaciones)+len(proyeccionesFinales))
	for _, autorizacion := range autorizaciones {
		vistas[autorizacion.AutorizacionRef] = struct{}{}
	}
	for _, proyeccion := range proyeccionesFinales {
		if _, repetida := vistas[proyeccion.AutorizacionRef]; repetida {
			return puertosbolsa.ManifiestoProbatorioBaremacion{}, ErrResultadoBaremacionNoConfiable
		}
		vistas[proyeccion.AutorizacionRef] = struct{}{}
		autorizaciones = append(autorizaciones, puertosbolsa.AutorizacionProbatoriaBaremacion{
			Secuencia: uint32(len(autorizaciones) + 1), Accion: proyeccion.Accion,
			ClaseRecurso: proyeccion.ClaseRecurso, RecursoRef: proyeccion.RecursoRef,
			AutorizacionRef: proyeccion.AutorizacionRef,
		})
	}
	evidencias, err := evidenciasManifiestoBaremacion(
		firmaPreparada, contenido, consulta, validacionInicial, sello, validacionTrasSello,
		aumento, validacionFinal, documentoFirmado,
	)
	if err != nil {
		return puertosbolsa.ManifiestoProbatorioBaremacion{}, err
	}
	referencia, err := s.generador.NuevaReferenciaManifiestoProbatorio()
	if err != nil || !referenciaAplicacionBaremacionValida(referencia) {
		return puertosbolsa.ManifiestoProbatorioBaremacion{}, errors.Join(ErrResultadoBaremacionNoConfiable, err)
	}
	base := firmaPreparada.decision.decision.revision.revision.version.Referencia
	manifiesto := puertosbolsa.ManifiestoProbatorioBaremacion{
		Esquema:        puertosbolsa.EsquemaManifiestoProbatorioBaremacion,
		Finalidad:      puertosbolsa.FinalidadManifiestoProbatorioBaremacion,
		VersionEsquema: puertosbolsa.VersionManifiestoProbatorioBaremacion,
		Referencia:     referencia, ProcesoRef: contenido.ProcesoRef, SolicitudRef: contenido.SolicitudRef,
		SujetoRef: contenido.SujetoRef, BaremacionMeritoRef: contenido.BaremacionMeritoRef,
		DecisionRef: contenido.ID, VersionBase: base.Numero, HuellaVersionBaseSHA256: base.HuellaEstadoSHA256,
		Autorizaciones: autorizaciones, Evidencias: evidencias, CreadoEn: creadoEn.UTC(),
	}
	preparado, carga, err := manifiesto.PrepararSellado()
	if err != nil {
		return puertosbolsa.ManifiestoProbatorioBaremacion{}, err
	}
	selloManifiesto, err := s.selladorSolicitud.SellarSelloBaremacion(ctx, puertosbolsa.SolicitudSellarSelloBaremacion{
		Finalidad:              puertosbolsa.FinalidadSelloManifiestoProbatorioBaremacionV2,
		RepresentacionCanonica: carga,
	})
	if err != nil || !selloGeneradoBaremacionValido(selloManifiesto) {
		return puertosbolsa.ManifiestoProbatorioBaremacion{},
			errors.Join(ErrResultadoBaremacionNoConfiable, err)
	}
	return preparado.IncorporarSello(selloManifiesto)
}

type especificacionAutorizacionManifiestoBaremacion struct {
	accion  puertosbolsa.AccionOperacionBaremacion
	clase   puertosbolsa.ClaseRecursoOperacionBaremacion
	recurso string
}

func autorizacionesPreviasManifiestoBaremacion(
	firma FirmaBaremacionPreparada,
) ([]puertosbolsa.AutorizacionProbatoriaBaremacion, error) {
	contenido := firma.decision.decision.revision.contenido
	calculo := firma.decision.decision.revision.calculo.Calculo
	accionAdopcion, existe := puertosbolsa.AccionAdopcionParaClase(contenido.Clase)
	if !existe {
		return nil, ErrResultadoBaremacionNoConfiable
	}
	especificaciones := []especificacionAutorizacionManifiestoBaremacion{
		{puertosbolsa.AccionConsultarBaremacionVigente, puertosbolsa.ClaseRecursoBaremacion, contenido.BaremacionMeritoRef},
		{puertosbolsa.AccionRecuperarCalculoBaremacion, puertosbolsa.ClaseRecursoCalculo, calculo.CalculoRef},
		{puertosbolsa.AccionConsultarCriterioBaremacion, puertosbolsa.ClaseRecursoProceso, calculo.Criterio.ProcesoRef},
	}
	for _, evidencia := range calculo.Evidencias {
		especificaciones = append(especificaciones,
			especificacionAutorizacionManifiestoBaremacion{puertosbolsa.AccionConsultarEvidenciaBaremacion, puertosbolsa.ClaseRecursoEvidencia, evidencia.Referencia.DocumentoRef},
			especificacionAutorizacionManifiestoBaremacion{puertosbolsa.AccionConsultarRepresentacionBaremacion, puertosbolsa.ClaseRecursoRepresentacion, evidencia.Referencia.RepresentacionRef},
		)
	}
	especificaciones = append(especificaciones,
		especificacionAutorizacionManifiestoBaremacion{accionAdopcion, puertosbolsa.ClaseRecursoBaremacion, contenido.BaremacionMeritoRef},
		especificacionAutorizacionManifiestoBaremacion{puertosbolsa.AccionConsultarPoliticaFirmaBaremacion, puertosbolsa.ClaseRecursoPoliticaFirma, firma.decision.decision.politica.Referencia},
		especificacionAutorizacionManifiestoBaremacion{puertosbolsa.AccionCodificarDecisionBaremacion, puertosbolsa.ClaseRecursoDecision, contenido.ID},
		especificacionAutorizacionManifiestoBaremacion{puertosbolsa.AccionCustodiarDecisionBaremacion, puertosbolsa.ClaseRecursoDecision, contenido.ID},
		especificacionAutorizacionManifiestoBaremacion{puertosbolsa.AccionPrepararFirmaDecisionBaremacion, puertosbolsa.ClaseRecursoDecision, contenido.ID},
	)
	if len(especificaciones) != len(firma.autorizacionesRefs) {
		return nil, ErrResultadoBaremacionNoConfiable
	}
	resultado := make([]puertosbolsa.AutorizacionProbatoriaBaremacion, len(especificaciones))
	for indice, especificacion := range especificaciones {
		resultado[indice] = puertosbolsa.AutorizacionProbatoriaBaremacion{
			Secuencia: uint32(indice + 1), Accion: especificacion.accion, ClaseRecurso: especificacion.clase,
			RecursoRef: especificacion.recurso, AutorizacionRef: firma.autorizacionesRefs[indice],
		}
		if resultado[indice].Validar() != nil {
			return nil, ErrResultadoBaremacionNoConfiable
		}
	}
	return resultado, nil
}

func evidenciasManifiestoBaremacion(
	firma FirmaBaremacionPreparada,
	contenido dominiobolsa.ContenidoDecisionTecnica,
	consulta puertosbolsa.ConsultaFirmaInteractiva,
	validacionInicial puertosbolsa.ValidacionFirmaServidor,
	sello *puertosbolsa.SelloTiempoFirma,
	validacionTrasSello *puertosbolsa.ValidacionFirmaServidor,
	aumento *puertosbolsa.ResultadoAumentoFirma,
	validacionFinal puertosbolsa.ValidacionFirmaServidor,
	documentoFirmado puertosbolsa.DocumentoFirmadoCustodiado,
) ([]puertosbolsa.EvidenciaProbatoriaBaremacion, error) {
	base := firma.decision.decision.revision.revision.version
	calculo := firma.decision.decision.revision.calculo
	politica := firma.decision.decision.politica
	codificacion := firma.decision.decision.codificacion
	documentoFirmable := firma.decision.documento
	contenidoHuella, err := contenido.HuellaContenidoSHA256()
	if err != nil {
		return nil, err
	}
	resultado := []puertosbolsa.EvidenciaProbatoriaBaremacion{
		{Tipo: puertosbolsa.EvidenciaEstadoBaseBaremacion, Referencia: base.Referencia.BaremacionMeritoRef, HuellaEvidenciaSHA256: base.Referencia.HuellaEstadoSHA256},
		{Tipo: puertosbolsa.EvidenciaCalculoOficialBaremacion, Referencia: calculo.EvidenciaGobiernoRef, HuellaEvidenciaSHA256: calculo.HuellaEvidenciaSHA256},
		{Tipo: puertosbolsa.EvidenciaCriterioPublicadoBaremacion, Referencia: calculo.Calculo.Criterio.ProcesoRef, HuellaEvidenciaSHA256: calculo.Calculo.Criterio.HuellaSHA256},
	}
	for _, evidencia := range calculo.Calculo.Evidencias {
		resultado = append(resultado,
			puertosbolsa.EvidenciaProbatoriaBaremacion{Tipo: puertosbolsa.EvidenciaDocumentoMeritoBaremacion, Referencia: evidencia.Referencia.DocumentoRef, HuellaEvidenciaSHA256: evidencia.Referencia.HuellaSHA256},
			puertosbolsa.EvidenciaProbatoriaBaremacion{Tipo: puertosbolsa.EvidenciaRepresentacionBaremacion, Referencia: evidencia.Referencia.RepresentacionRef, HuellaEvidenciaSHA256: evidencia.Referencia.HuellaSHA256},
		)
	}
	resultado = append(resultado,
		puertosbolsa.EvidenciaProbatoriaBaremacion{Tipo: puertosbolsa.EvidenciaContenidoDecisionBaremacion, Referencia: contenido.ID, HuellaEvidenciaSHA256: contenidoHuella},
		puertosbolsa.EvidenciaProbatoriaBaremacion{Tipo: puertosbolsa.EvidenciaPoliticaFirmaBaremacion, Referencia: politica.AprobacionRef, HuellaEvidenciaSHA256: politica.HuellaAprobacionSHA256},
		puertosbolsa.EvidenciaProbatoriaBaremacion{Tipo: puertosbolsa.EvidenciaDocumentoCanonicoBaremacion, Referencia: codificacion.DecisionRef, HuellaEvidenciaSHA256: codificacion.HuellaDocumentoSHA256},
		puertosbolsa.EvidenciaProbatoriaBaremacion{Tipo: puertosbolsa.EvidenciaCustodiaFirmableBaremacion, Referencia: documentoFirmable.EvidenciaCustodia.Referencia, HuellaEvidenciaSHA256: documentoFirmable.HuellaDocumentoSHA256},
		puertosbolsa.EvidenciaProbatoriaBaremacion{Tipo: puertosbolsa.EvidenciaPreparacionFirmaBaremacion, Referencia: firma.sesion.EvidenciaPreparacionRef, HuellaEvidenciaSHA256: firma.sesion.HuellaEvidenciaSHA256},
		puertosbolsa.EvidenciaProbatoriaBaremacion{Tipo: puertosbolsa.EvidenciaConsultaFirmaBaremacion, Referencia: consulta.EvidenciaConsultaRef, HuellaEvidenciaSHA256: consulta.HuellaEvidenciaSHA256},
		puertosbolsa.EvidenciaProbatoriaBaremacion{Tipo: puertosbolsa.EvidenciaValidacionInicialBaremacion, Referencia: validacionInicial.ValidacionRef, HuellaEvidenciaSHA256: validacionInicial.HuellaValidacionSHA256},
	)
	if sello != nil {
		if validacionTrasSello == nil {
			return nil, ErrResultadoBaremacionNoConfiable
		}
		vinculo, err := puertosbolsa.NuevoVinculoRevisionSelladaPAdES(*sello, *validacionTrasSello)
		if err != nil {
			return nil, ErrResultadoBaremacionNoConfiable
		}
		resultado = append(resultado,
			puertosbolsa.EvidenciaProbatoriaBaremacion{
				Tipo: puertosbolsa.EvidenciaSelloTiempoBaremacion, Referencia: sello.SelloTiempoRef,
				HuellaEvidenciaSHA256: sello.HuellaSelloTiempoSHA256,
			},
			puertosbolsa.EvidenciaProbatoriaBaremacion{
				Tipo:       puertosbolsa.EvidenciaVinculoRevisionSelladaBaremacion,
				Referencia: vinculo.Referencia, HuellaEvidenciaSHA256: vinculo.HuellaSHA256,
			},
		)
	}
	if aumento != nil {
		if sello == nil || validacionTrasSello == nil {
			return nil, ErrResultadoBaremacionNoConfiable
		}
		resultado = append(resultado, puertosbolsa.EvidenciaProbatoriaBaremacion{
			Tipo:                  puertosbolsa.EvidenciaValidacionDocumentoSelladoBaremacion,
			Referencia:            validacionTrasSello.ValidacionRef,
			HuellaEvidenciaSHA256: validacionTrasSello.HuellaValidacionSHA256,
		})
		resultado = append(resultado, puertosbolsa.EvidenciaProbatoriaBaremacion{
			Tipo: puertosbolsa.EvidenciaAumentoLongevidadBaremacion, Referencia: aumento.EvidenciaAumentoRef,
			HuellaEvidenciaSHA256: aumento.HuellaEvidenciaSHA256,
		})
		vinculo, err := puertosbolsa.NuevoVinculoRevisionLongevaPAdES(
			*sello, *validacionTrasSello, *aumento, validacionFinal,
		)
		if err != nil {
			return nil, ErrResultadoBaremacionNoConfiable
		}
		resultado = append(resultado, puertosbolsa.EvidenciaProbatoriaBaremacion{
			Tipo:       puertosbolsa.EvidenciaVinculoRevisionLongevaBaremacion,
			Referencia: vinculo.Referencia, HuellaEvidenciaSHA256: vinculo.HuellaSHA256,
		})
	}
	resultado = append(resultado,
		puertosbolsa.EvidenciaProbatoriaBaremacion{Tipo: puertosbolsa.EvidenciaValidacionFinalBaremacion, Referencia: validacionFinal.ValidacionRef, HuellaEvidenciaSHA256: validacionFinal.HuellaValidacionSHA256},
		puertosbolsa.EvidenciaProbatoriaBaremacion{Tipo: puertosbolsa.EvidenciaRecuperacionFirmadoBaremacion, Referencia: documentoFirmado.EvidenciaRecuperacionRef, HuellaEvidenciaSHA256: documentoFirmado.HuellaEvidenciaRecuperacionSHA256},
		puertosbolsa.EvidenciaProbatoriaBaremacion{Tipo: puertosbolsa.EvidenciaCustodiaFirmadoBaremacion, Referencia: documentoFirmado.EvidenciaEscritura.Referencia, HuellaEvidenciaSHA256: documentoFirmado.Objeto.HuellaSHA256},
		puertosbolsa.EvidenciaProbatoriaBaremacion{Tipo: puertosbolsa.EvidenciaRetencionFirmadoBaremacion, Referencia: documentoFirmado.EvidenciaRetencion.Referencia, HuellaEvidenciaSHA256: documentoFirmado.Objeto.HuellaSHA256},
	)
	for indice := range resultado {
		resultado[indice].Secuencia = uint32(indice + 1)
		if resultado[indice].Validar() != nil {
			return nil, ErrResultadoBaremacionNoConfiable
		}
	}
	return resultado, nil
}

type lectorFirmadoVerificado struct {
	origen         io.Reader
	huella         hash.Hash
	leidos         int64
	tamanoEsperado int64
	huellaEsperada string
}

func nuevoLectorFirmadoVerificado(origen io.Reader, tamano int64, huellaEsperada string) *lectorFirmadoVerificado {
	return &lectorFirmadoVerificado{origen: origen, huella: sha256.New(), tamanoEsperado: tamano, huellaEsperada: huellaEsperada}
}

func (l *lectorFirmadoVerificado) Read(destino []byte) (int, error) {
	n, err := l.origen.Read(destino)
	if n > 0 {
		l.leidos += int64(n)
		_, _ = l.huella.Write(destino[:n])
		if l.leidos > l.tamanoEsperado {
			return n, ErrResultadoBaremacionNoConfiable
		}
	}
	return n, err
}

func (l *lectorFirmadoVerificado) Verificar() error {
	if l == nil || l.leidos != l.tamanoEsperado ||
		hex.EncodeToString(l.huella.Sum(nil)) != l.huellaEsperada {
		return ErrResultadoBaremacionNoConfiable
	}
	return nil
}

func recursoAlmacenBaremacion(
	recursoRef string,
	clase puertosbolsa.ClaseRecursoOperacionBaremacion,
	sujetoRef string,
	vinculos puertosvec.VinculosOperacionAlmacen,
) dominiovec.RecursoAutorizable {
	atributos := map[string]string{
		puertosvec.AtributoAlmacenOperacionRef:        vinculos.OperacionRef,
		puertosvec.AtributoAlmacenCargaRef:            vinculos.CargaRef,
		puertosvec.AtributoAlmacenClasificacion:       vinculos.Clasificacion,
		puertosvec.AtributoAlmacenSujetoSeudonimoHMAC: vinculos.SujetoSeudonimoHMAC,
		puertosvec.AtributoAlmacenHuellaSolicitudHMAC: vinculos.HuellaSolicitudHMAC,
		puertosvec.AtributoAlmacenEfectoRef:           vinculos.EfectoRef,
	}
	if vinculos.ObjetoVinculado.Validar() == nil {
		atributos[puertosvec.AtributoAlmacenObjetoRef] = vinculos.ObjetoVinculado.Referencia
		atributos[puertosvec.AtributoAlmacenObjetoVersion] = vinculos.ObjetoVinculado.Version
	}
	return dominiovec.RecursoAutorizable{
		Referencia: recursoRef, ModuloID: "bolsa", Tipo: string(clase),
		Ambitos: map[string]string{"sujeto_ref": sujetoRef}, Atributos: atributos,
	}
}
