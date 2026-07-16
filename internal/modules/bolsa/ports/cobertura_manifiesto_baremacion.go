package ports

import dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"

// disposicionManifiestoBaremacion fija las posiciones canonicas del flujo. Los
// indices se calculan a partir del numero de evidencias de merito y de las dos
// capas opcionales gobernadas por la politica de firma.
type disposicionManifiestoBaremacion struct {
	meritos int
	sello   bool
	aumento bool

	autorizacionAdopcion            int
	autorizacionPolitica            int
	autorizacionConsultaFirma       int
	autorizacionValidacionInicial   int
	autorizacionSello               int
	autorizacionValidacionTrasSello int
	autorizacionAumento             int
	autorizacionValidacionFinal     int
	autorizacionRecuperacion        int
	autorizacionCustodiaFirmado     int
	autorizacionRetencion           int
	autorizacionReserva             int
	autorizacionConfirmacion        int

	evidenciaContenido           int
	evidenciaPolitica            int
	evidenciaDocumentoCanonico   int
	evidenciaCustodiaFirmable    int
	evidenciaPreparacionFirma    int
	evidenciaConsultaFirma       int
	evidenciaValidacionInicial   int
	evidenciaSello               int
	evidenciaValidacionTrasSello int
	evidenciaAumento             int
	evidenciaValidacionFinal     int
	evidenciaRecuperacion        int
	evidenciaCustodiaFirmado     int
	evidenciaRetencion           int
}

func (m ManifiestoProbatorioBaremacion) validarCoberturaCanonica() (disposicionManifiestoBaremacion, error) {
	fallo := func() (disposicionManifiestoBaremacion, error) {
		return disposicionManifiestoBaremacion{}, ErrSolicitudBaremacionInvalida
	}
	if len(m.Autorizaciones) < 17 || len(m.Evidencias) < 16 {
		return fallo()
	}

	indice := 3
	meritos := 0
	for indice < len(m.Evidencias) && m.Evidencias[indice].Tipo == EvidenciaDocumentoMeritoBaremacion {
		if indice+1 >= len(m.Evidencias) || m.Evidencias[indice+1].Tipo != EvidenciaRepresentacionBaremacion {
			return fallo()
		}
		meritos++
		indice += 2
	}
	if meritos < 1 {
		return fallo()
	}

	baseEvidencias := 3 + 2*meritos
	if len(m.Evidencias) < baseEvidencias+11 {
		return fallo()
	}
	disposicion := disposicionManifiestoBaremacion{
		meritos:                         meritos,
		autorizacionSello:               -1,
		autorizacionValidacionTrasSello: -1,
		autorizacionAumento:             -1,
		autorizacionValidacionFinal:     -1,
		evidenciaSello:                  -1,
		evidenciaValidacionTrasSello:    -1,
		evidenciaAumento:                -1,
		evidenciaContenido:              baseEvidencias,
		evidenciaPolitica:               baseEvidencias + 1,
		evidenciaDocumentoCanonico:      baseEvidencias + 2,
		evidenciaCustodiaFirmable:       baseEvidencias + 3,
		evidenciaPreparacionFirma:       baseEvidencias + 4,
		evidenciaConsultaFirma:          baseEvidencias + 5,
		evidenciaValidacionInicial:      baseEvidencias + 6,
	}
	cursorEvidencias := baseEvidencias + 7
	if cursorEvidencias < len(m.Evidencias) && m.Evidencias[cursorEvidencias].Tipo == EvidenciaSelloTiempoBaremacion {
		disposicion.sello = true
		disposicion.evidenciaSello = cursorEvidencias
		cursorEvidencias++
	}
	if cursorEvidencias < len(m.Evidencias) && m.Evidencias[cursorEvidencias].Tipo == EvidenciaValidacionDocumentoSelladoBaremacion {
		disposicion.evidenciaValidacionTrasSello = cursorEvidencias
		cursorEvidencias++
	}
	if cursorEvidencias < len(m.Evidencias) && m.Evidencias[cursorEvidencias].Tipo == EvidenciaAumentoLongevidadBaremacion {
		disposicion.aumento = true
		disposicion.evidenciaAumento = cursorEvidencias
		cursorEvidencias++
	}
	if disposicion.aumento && (!disposicion.sello || disposicion.evidenciaValidacionTrasSello < 0) ||
		!disposicion.aumento && disposicion.evidenciaValidacionTrasSello >= 0 {
		return fallo()
	}
	disposicion.evidenciaValidacionFinal = cursorEvidencias
	disposicion.evidenciaRecuperacion = cursorEvidencias + 1
	disposicion.evidenciaCustodiaFirmado = cursorEvidencias + 2
	disposicion.evidenciaRetencion = cursorEvidencias + 3

	tiposEsperados := tiposEvidenciaCanonicos(meritos, disposicion.sello, disposicion.aumento)
	if len(m.Evidencias) != len(tiposEsperados) {
		return fallo()
	}
	for i, esperado := range tiposEsperados {
		if m.Evidencias[i].Tipo != esperado {
			return fallo()
		}
	}

	baseAutorizaciones := 3 + 2*meritos
	if baseAutorizaciones >= len(m.Autorizaciones) || !accionAdopcionManifiestoValida(m.Autorizaciones[baseAutorizaciones].Accion) {
		return fallo()
	}
	disposicion.autorizacionAdopcion = baseAutorizaciones
	disposicion.autorizacionPolitica = baseAutorizaciones + 1
	disposicion.autorizacionConsultaFirma = baseAutorizaciones + 5
	disposicion.autorizacionValidacionInicial = baseAutorizaciones + 6
	cursorAutorizaciones := baseAutorizaciones + 7
	if disposicion.sello {
		disposicion.autorizacionSello = cursorAutorizaciones
		cursorAutorizaciones++
	}
	if disposicion.aumento {
		disposicion.autorizacionValidacionTrasSello = cursorAutorizaciones
		disposicion.autorizacionAumento = cursorAutorizaciones + 1
		cursorAutorizaciones += 2
	}
	if disposicion.sello {
		disposicion.autorizacionValidacionFinal = cursorAutorizaciones
		cursorAutorizaciones++
	}
	disposicion.autorizacionRecuperacion = cursorAutorizaciones
	disposicion.autorizacionCustodiaFirmado = cursorAutorizaciones + 1
	disposicion.autorizacionRetencion = cursorAutorizaciones + 2
	disposicion.autorizacionReserva = cursorAutorizaciones + 3
	disposicion.autorizacionConfirmacion = cursorAutorizaciones + 4

	accionesEsperadas := accionesManifiestoCanonicas(
		meritos, m.Autorizaciones[baseAutorizaciones].Accion, disposicion.sello, disposicion.aumento,
	)
	if len(m.Autorizaciones) != len(accionesEsperadas) {
		return fallo()
	}
	for i, esperada := range accionesEsperadas {
		if m.Autorizaciones[i].Accion != esperada {
			return fallo()
		}
	}
	if !m.coberturaReferenciasEstructuralesValida(disposicion) {
		return fallo()
	}
	return disposicion, nil
}

func tiposEvidenciaCanonicos(meritos int, sello, aumento bool) []TipoEvidenciaProbatoriaBaremacion {
	tipos := []TipoEvidenciaProbatoriaBaremacion{
		EvidenciaEstadoBaseBaremacion,
		EvidenciaCalculoOficialBaremacion,
		EvidenciaCriterioPublicadoBaremacion,
	}
	for i := 0; i < meritos; i++ {
		tipos = append(tipos, EvidenciaDocumentoMeritoBaremacion, EvidenciaRepresentacionBaremacion)
	}
	tipos = append(tipos,
		EvidenciaContenidoDecisionBaremacion,
		EvidenciaPoliticaFirmaBaremacion,
		EvidenciaDocumentoCanonicoBaremacion,
		EvidenciaCustodiaFirmableBaremacion,
		EvidenciaPreparacionFirmaBaremacion,
		EvidenciaConsultaFirmaBaremacion,
		EvidenciaValidacionInicialBaremacion,
	)
	if sello {
		tipos = append(tipos, EvidenciaSelloTiempoBaremacion)
	}
	if aumento {
		tipos = append(tipos, EvidenciaValidacionDocumentoSelladoBaremacion, EvidenciaAumentoLongevidadBaremacion)
	}
	return append(tipos,
		EvidenciaValidacionFinalBaremacion,
		EvidenciaRecuperacionFirmadoBaremacion,
		EvidenciaCustodiaFirmadoBaremacion,
		EvidenciaRetencionFirmadoBaremacion,
	)
}

func accionesManifiestoCanonicas(
	meritos int,
	adopcion AccionOperacionBaremacion,
	sello, aumento bool,
) []AccionOperacionBaremacion {
	acciones := []AccionOperacionBaremacion{
		AccionConsultarBaremacionVigente,
		AccionRecuperarCalculoBaremacion,
		AccionConsultarCriterioBaremacion,
	}
	for i := 0; i < meritos; i++ {
		acciones = append(acciones, AccionConsultarEvidenciaBaremacion, AccionConsultarRepresentacionBaremacion)
	}
	acciones = append(acciones,
		adopcion,
		AccionConsultarPoliticaFirmaBaremacion,
		AccionCodificarDecisionBaremacion,
		AccionCustodiarDecisionBaremacion,
		AccionPrepararFirmaDecisionBaremacion,
		AccionConsultarFirmaDecisionBaremacion,
		AccionValidarFirmaDecisionBaremacion,
	)
	if sello {
		acciones = append(acciones, AccionSellarTiempoDecisionBaremacion)
	}
	if aumento {
		acciones = append(acciones, AccionValidarFirmaDecisionBaremacion, AccionAumentarFirmaDecisionBaremacion)
	}
	if sello {
		acciones = append(acciones, AccionValidarFirmaDecisionBaremacion)
	}
	return append(acciones,
		AccionRecuperarBinarioFirmadoBaremacion,
		AccionCustodiarDocumentoFirmadoBaremacion,
		AccionRetenerDocumentoFirmadoBaremacion,
		AccionReservarDecisionBaremacion,
		AccionConfirmarDecisionBaremacion,
	)
}

func accionAdopcionManifiestoValida(accion AccionOperacionBaremacion) bool {
	switch accion {
	case AccionAdoptarDecisionInicialBaremacion, AccionRectificarDecisionBaremacion,
		AccionRevocarDecisionBaremacion, AccionRehabilitarDecisionBaremacion:
		return true
	default:
		return false
	}
}

func (m ManifiestoProbatorioBaremacion) coberturaReferenciasEstructuralesValida(
	d disposicionManifiestoBaremacion,
) bool {
	autorizaciones := m.Autorizaciones
	evidencias := m.Evidencias
	if autorizaciones[0].RecursoRef != m.BaremacionMeritoRef || autorizaciones[2].RecursoRef != m.ProcesoRef ||
		autorizaciones[d.autorizacionAdopcion].RecursoRef != m.BaremacionMeritoRef ||
		autorizaciones[d.autorizacionAdopcion+2].RecursoRef != m.DecisionRef ||
		autorizaciones[d.autorizacionAdopcion+3].RecursoRef != m.DecisionRef ||
		autorizaciones[d.autorizacionAdopcion+4].RecursoRef != m.DecisionRef ||
		autorizaciones[d.autorizacionReserva].RecursoRef != m.BaremacionMeritoRef ||
		autorizaciones[d.autorizacionConfirmacion].RecursoRef != m.BaremacionMeritoRef {
		return false
	}
	for i := 0; i < d.meritos; i++ {
		indice := 3 + 2*i
		if autorizaciones[indice].RecursoRef != evidencias[indice].Referencia ||
			autorizaciones[indice+1].RecursoRef != evidencias[indice+1].Referencia ||
			evidencias[indice].HuellaEvidenciaSHA256 != evidencias[indice+1].HuellaEvidenciaSHA256 {
			return false
		}
	}
	if evidencias[0].Referencia != m.BaremacionMeritoRef ||
		evidencias[0].HuellaEvidenciaSHA256 != m.HuellaVersionBaseSHA256 ||
		evidencias[2].Referencia != m.ProcesoRef ||
		evidencias[d.evidenciaContenido].Referencia != m.DecisionRef ||
		evidencias[d.evidenciaDocumentoCanonico].Referencia != m.DecisionRef ||
		evidencias[d.evidenciaDocumentoCanonico].HuellaEvidenciaSHA256 !=
			evidencias[d.evidenciaCustodiaFirmable].HuellaEvidenciaSHA256 ||
		evidencias[d.evidenciaPreparacionFirma].Referencia == evidencias[d.evidenciaConsultaFirma].Referencia ||
		evidencias[d.evidenciaCustodiaFirmado].Referencia == evidencias[d.evidenciaRetencion].Referencia ||
		evidencias[d.evidenciaCustodiaFirmado].HuellaEvidenciaSHA256 !=
			evidencias[d.evidenciaRetencion].HuellaEvidenciaSHA256 {
		return false
	}
	recursoFirma := autorizaciones[d.autorizacionValidacionInicial].RecursoRef
	if autorizaciones[d.autorizacionConsultaFirma].RecursoRef == recursoFirma {
		return false
	}
	if d.sello && autorizaciones[d.autorizacionSello].RecursoRef != recursoFirma {
		return false
	}
	if d.sello && (autorizaciones[d.autorizacionValidacionFinal].RecursoRef != recursoFirma ||
		evidencias[d.evidenciaValidacionInicial].Referencia == evidencias[d.evidenciaValidacionFinal].Referencia) {
		return false
	}
	if d.aumento && (autorizaciones[d.autorizacionValidacionTrasSello].RecursoRef != recursoFirma ||
		autorizaciones[d.autorizacionAumento].RecursoRef != recursoFirma ||
		evidencias[d.evidenciaValidacionInicial].Referencia == evidencias[d.evidenciaValidacionTrasSello].Referencia ||
		evidencias[d.evidenciaValidacionTrasSello].Referencia == evidencias[d.evidenciaValidacionFinal].Referencia) {
		return false
	}
	if !d.sello && (evidencias[d.evidenciaValidacionInicial].Referencia != evidencias[d.evidenciaValidacionFinal].Referencia ||
		evidencias[d.evidenciaValidacionInicial].HuellaEvidenciaSHA256 != evidencias[d.evidenciaValidacionFinal].HuellaEvidenciaSHA256) {
		return false
	}
	recursoFirmado := autorizaciones[d.autorizacionRecuperacion].RecursoRef
	return autorizaciones[d.autorizacionCustodiaFirmado].RecursoRef == recursoFirmado &&
		autorizaciones[d.autorizacionRetencion].RecursoRef == recursoFirmado
}

func (m ManifiestoProbatorioBaremacion) validarCoberturaPara(
	version ReferenciaVersionBaremacion,
	contenido dominiobolsa.ContenidoDecisionTecnica,
) (disposicionManifiestoBaremacion, error) {
	d, err := m.validarCoberturaCanonica()
	if err != nil || m.Validar() != nil || version.Validar() != nil || contenido.Validar() != nil ||
		m.ProcesoRef != contenido.ProcesoRef || m.SolicitudRef != contenido.SolicitudRef ||
		m.SujetoRef != contenido.SujetoRef || m.BaremacionMeritoRef != contenido.BaremacionMeritoRef ||
		version.BaremacionMeritoRef != contenido.BaremacionMeritoRef || m.DecisionRef != contenido.ID ||
		m.VersionBase != version.Numero || contenido.VersionAnteriorBaremacion != version.Numero ||
		m.HuellaVersionBaseSHA256 != version.HuellaEstadoSHA256 ||
		contenido.VersionBaremacion != version.Numero+1 || m.CreadoEn.Before(contenido.DecididaEn) {
		return disposicionManifiestoBaremacion{}, ErrSolicitudBaremacionInvalida
	}
	accionAdopcion, existe := AccionAdopcionParaClase(contenido.Clase)
	evidenciasMerito, err := contenido.CalculoOficial.EvidenciasCanonicas()
	if err != nil || !existe || len(evidenciasMerito) != d.meritos ||
		m.Autorizaciones[1].RecursoRef != contenido.CalculoOficial.CalculoRef ||
		m.Autorizaciones[2].RecursoRef != contenido.Criterio.ProcesoRef ||
		m.Autorizaciones[d.autorizacionAdopcion].Accion != accionAdopcion ||
		m.Autorizaciones[d.autorizacionAdopcion].AutorizacionRef != contenido.AutorizacionRef ||
		m.Evidencias[2].Referencia != contenido.Criterio.ProcesoRef ||
		m.Evidencias[2].HuellaEvidenciaSHA256 != contenido.Criterio.HuellaSHA256 {
		return disposicionManifiestoBaremacion{}, ErrSolicitudBaremacionInvalida
	}
	for i, evidencia := range evidenciasMerito {
		indice := 3 + 2*i
		referencia := evidencia.Referencia
		if m.Autorizaciones[indice].RecursoRef != referencia.DocumentoRef ||
			m.Autorizaciones[indice+1].RecursoRef != referencia.RepresentacionRef ||
			m.Evidencias[indice].Referencia != referencia.DocumentoRef ||
			m.Evidencias[indice+1].Referencia != referencia.RepresentacionRef ||
			m.Evidencias[indice].HuellaEvidenciaSHA256 != referencia.HuellaSHA256 ||
			m.Evidencias[indice+1].HuellaEvidenciaSHA256 != referencia.HuellaSHA256 {
			return disposicionManifiestoBaremacion{}, ErrSolicitudBaremacionInvalida
		}
	}
	huellaContenido, err := contenido.HuellaContenidoSHA256()
	if err != nil || m.Evidencias[d.evidenciaContenido].Referencia != contenido.ID ||
		m.Evidencias[d.evidenciaContenido].HuellaEvidenciaSHA256 != huellaContenido {
		return disposicionManifiestoBaremacion{}, ErrSolicitudBaremacionInvalida
	}
	return d, nil
}

// ValidarCoberturaFirmaPara prueba que el manifiesto contiene exactamente las
// capacidades y recibos que sostienen la firma incorporada a la decision. No
// basta con que cada entrada sea valida de forma aislada: recursos, huellas,
// capas opcionales y orden deben coincidir con esa decision concreta.
func (m ManifiestoProbatorioBaremacion) ValidarCoberturaFirmaPara(
	version ReferenciaVersionBaremacion,
	contenido dominiobolsa.ContenidoDecisionTecnica,
	firma dominiobolsa.FirmaDecisionTecnica,
) error {
	d, err := m.validarCoberturaPara(version, contenido)
	if err != nil {
		return err
	}
	if _, err := dominiobolsa.ConstituirDecisionFirmada(contenido, firma); err != nil ||
		firma.ManifiestoProbatorioRef != m.Referencia ||
		firma.HuellaManifiestoProbatorioSHA256 != m.HuellaManifiestoSHA256 ||
		firma.SelloManifiestoProbatorioHMACSHA256 != m.SelloManifiestoHMACSHA256 ||
		firma.RequiereSelloTiempo != d.sello || firma.RequiereAumentoLongevidad != d.aumento ||
		m.CreadoEn.Before(firma.ValidadaEn) {
		return ErrSolicitudBaremacionInvalida
	}
	autorizaciones := m.Autorizaciones
	evidencias := m.Evidencias
	if autorizaciones[d.autorizacionPolitica].RecursoRef != firma.PoliticaFirmaRef ||
		autorizaciones[d.autorizacionConsultaFirma].RecursoRef != firma.SesionFirmaInteractivaRef ||
		autorizaciones[d.autorizacionValidacionInicial].RecursoRef != firma.FirmaRef ||
		autorizaciones[d.autorizacionRecuperacion].RecursoRef != firma.DocumentoFirmadoRef ||
		autorizaciones[d.autorizacionCustodiaFirmado].RecursoRef != firma.DocumentoFirmadoRef ||
		autorizaciones[d.autorizacionRetencion].RecursoRef != firma.DocumentoFirmadoRef ||
		!evidenciaManifiestoCoincide(evidencias[d.evidenciaDocumentoCanonico], contenido.ID, firma.HuellaDocumentoFirmableSHA256) ||
		!evidenciaManifiestoCoincide(evidencias[d.evidenciaCustodiaFirmable], firma.EvidenciaCustodiaRef, firma.HuellaDocumentoFirmableSHA256) ||
		!evidenciaManifiestoCoincide(evidencias[d.evidenciaValidacionInicial], firma.ValidacionInicialFirmaRef, firma.HuellaValidacionInicialSHA256) ||
		!evidenciaManifiestoCoincide(evidencias[d.evidenciaValidacionFinal], firma.ValidacionFirmaRef, firma.HuellaValidacionSHA256) ||
		!evidenciaManifiestoCoincide(evidencias[d.evidenciaRecuperacion], firma.EvidenciaRecuperacionFirmadoRef, firma.HuellaEvidenciaRecuperacionSHA256) ||
		!evidenciaManifiestoCoincide(evidencias[d.evidenciaCustodiaFirmado], firma.EvidenciaCustodiaDocumentoFirmadoRef, firma.HuellaDocumentoSHA256) ||
		!evidenciaManifiestoCoincide(evidencias[d.evidenciaRetencion], firma.EvidenciaRetencionDocumentoFirmadoRef, firma.HuellaDocumentoSHA256) {
		return ErrSolicitudBaremacionInvalida
	}
	if d.sello && (autorizaciones[d.autorizacionSello].RecursoRef != firma.FirmaRef ||
		autorizaciones[d.autorizacionValidacionFinal].RecursoRef != firma.FirmaRef ||
		!evidenciaManifiestoCoincide(evidencias[d.evidenciaSello], firma.SelloTiempoRef, firma.HuellaSelloTiempoSHA256)) {
		return ErrSolicitudBaremacionInvalida
	}
	if d.aumento && (autorizaciones[d.autorizacionValidacionTrasSello].RecursoRef != firma.FirmaRef ||
		autorizaciones[d.autorizacionAumento].RecursoRef != firma.FirmaRef ||
		!evidenciaManifiestoCoincide(evidencias[d.evidenciaValidacionTrasSello],
			firma.ValidacionDocumentoSelladoRef, firma.HuellaValidacionDocumentoSelladoSHA256) ||
		!evidenciaManifiestoCoincide(evidencias[d.evidenciaAumento], firma.AumentoLongevidadRef, firma.HuellaAumentoLongevidadSHA256)) {
		return ErrSolicitudBaremacionInvalida
	}
	return nil
}

func (m ManifiestoProbatorioBaremacion) validarCoberturaArtefactosFirmaPara(
	politica PoliticaFirmaBaremacion,
	artefacto ArtefactoFirma,
	validacionInicial ValidacionFirmaServidor,
	sello *SelloTiempoFirma,
	validacionTrasSello *ValidacionFirmaServidor,
	aumento *ResultadoAumentoFirma,
	validacionFinal ValidacionFirmaServidor,
	documento DocumentoFirmadoCustodiado,
) error {
	d, err := m.validarCoberturaCanonica()
	if err != nil || m.Validar() != nil || politica.Validar() != nil || artefacto.Validar() != nil ||
		validacionInicial.Validar() != nil || validacionFinal.Validar() != nil ||
		(sello != nil) != d.sello || (aumento != nil) != d.aumento || (validacionTrasSello != nil) != d.sello {
		return ErrSolicitudBaremacionInvalida
	}
	if validacionTrasSello != nil && validacionTrasSello.Validar() != nil {
		return ErrSolicitudBaremacionInvalida
	}
	artefactoFinal := artefacto
	if sello != nil {
		artefactoFinal = sello.ArtefactoSellado
	}
	if aumento != nil {
		artefactoFinal = aumento.Artefacto
	}
	autorizaciones := m.Autorizaciones
	evidencias := m.Evidencias
	if autorizaciones[d.autorizacionPolitica].RecursoRef != politica.Referencia ||
		autorizaciones[d.autorizacionConsultaFirma].RecursoRef != artefacto.SesionFirmaRef ||
		autorizaciones[d.autorizacionValidacionInicial].RecursoRef != artefacto.FirmaRef ||
		!evidenciaManifiestoCoincide(evidencias[d.evidenciaPolitica], politica.AprobacionRef, politica.HuellaAprobacionSHA256) ||
		!evidenciaManifiestoCoincide(evidencias[d.evidenciaDocumentoCanonico], artefacto.DecisionRef, artefacto.HuellaDocumentoFirmableSHA256) ||
		!evidenciaManifiestoCoincide(evidencias[d.evidenciaCustodiaFirmable], artefacto.EvidenciaCustodiaRef, artefacto.HuellaDocumentoFirmableSHA256) ||
		!evidenciaManifiestoCoincide(evidencias[d.evidenciaValidacionInicial], validacionInicial.ValidacionRef, validacionInicial.HuellaValidacionSHA256) ||
		!evidenciaManifiestoCoincide(evidencias[d.evidenciaValidacionFinal], validacionFinal.ValidacionRef, validacionFinal.HuellaValidacionSHA256) ||
		autorizaciones[d.autorizacionRecuperacion].RecursoRef != artefactoFinal.DocumentoFirmadoRef ||
		autorizaciones[d.autorizacionCustodiaFirmado].RecursoRef != artefactoFinal.DocumentoFirmadoRef ||
		autorizaciones[d.autorizacionRetencion].RecursoRef != artefactoFinal.DocumentoFirmadoRef ||
		!evidenciaManifiestoCoincide(evidencias[d.evidenciaRecuperacion], documento.EvidenciaRecuperacionRef, documento.HuellaEvidenciaRecuperacionSHA256) ||
		!evidenciaManifiestoCoincide(evidencias[d.evidenciaCustodiaFirmado], documento.EvidenciaEscritura.Referencia, documento.HuellaDocumentoSHA256) ||
		!evidenciaManifiestoCoincide(evidencias[d.evidenciaRetencion], documento.EvidenciaRetencion.Referencia, documento.HuellaDocumentoSHA256) {
		return ErrSolicitudBaremacionInvalida
	}
	if sello != nil && (sello.Validar() != nil ||
		autorizaciones[d.autorizacionSello].RecursoRef != artefacto.FirmaRef ||
		autorizaciones[d.autorizacionValidacionFinal].RecursoRef != artefacto.FirmaRef ||
		!evidenciaManifiestoCoincide(evidencias[d.evidenciaSello], sello.SelloTiempoRef, sello.HuellaSelloTiempoSHA256)) {
		return ErrSolicitudBaremacionInvalida
	}
	if aumento != nil && (aumento.Validar() != nil ||
		autorizaciones[d.autorizacionValidacionTrasSello].RecursoRef != artefacto.FirmaRef ||
		autorizaciones[d.autorizacionAumento].RecursoRef != artefacto.FirmaRef ||
		!evidenciaManifiestoCoincide(evidencias[d.evidenciaValidacionTrasSello],
			validacionTrasSello.ValidacionRef, validacionTrasSello.HuellaValidacionSHA256) ||
		!evidenciaManifiestoCoincide(evidencias[d.evidenciaAumento], aumento.EvidenciaAumentoRef, aumento.HuellaEvidenciaSHA256)) {
		return ErrSolicitudBaremacionInvalida
	}
	return nil
}

func evidenciaManifiestoCoincide(evidencia EvidenciaProbatoriaBaremacion, referencia, huella string) bool {
	return evidencia.Referencia == referencia && evidencia.HuellaEvidenciaSHA256 == huella
}

func (m ManifiestoProbatorioBaremacion) autorizacionConfirmacionCoincide(
	contexto ContextoOperacionBaremacion,
) bool {
	d, err := m.validarCoberturaCanonica()
	if err != nil || contexto.ValidarPara(
		AccionConfirmarDecisionBaremacion, ClaseRecursoBaremacion, m.BaremacionMeritoRef,
	) != nil {
		return false
	}
	return m.Autorizaciones[d.autorizacionConfirmacion].AutorizacionRef ==
		contexto.Proyeccion().AutorizacionRef
}
