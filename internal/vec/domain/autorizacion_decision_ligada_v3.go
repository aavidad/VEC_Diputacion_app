package domain

import (
	"errors"
	"sort"
	"time"
)

var (
	ErrEvidenciaEvaluacionAutorizacionV3Invalida = errors.New(
		"vec: evidencia de evaluacion de autorizacion V3 invalida",
	)
	ErrDecisionAutorizacionLigadaV3Invalida = errors.New(
		"vec: decision de autorizacion ligada V3 invalida",
	)
	ErrSerializacionEvidenciaEvaluacionAutorizacionV3Prohibida = errors.New(
		"vec: serializacion de evidencia de evaluacion de autorizacion V3 prohibida",
	)
	ErrSerializacionDecisionAutorizacionLigadaV3Prohibida = errors.New(
		"vec: serializacion de decision de autorizacion ligada V3 prohibida",
	)
)

const VersionDecisionAutorizacionLigadaV3 uint16 = 3

type evidenciaPoliticaAutorizacionV3 struct {
	referencia   string
	huellaSHA256 string
}

// datosEvidenciaEvaluacionAutorizacionV3 conserva solo el manifiesto
// minimizado de la instantanea evaluada. Los documentos vivos de asignacion,
// rol y politicas no forman parte de la capacidad.
type datosEvidenciaEvaluacionAutorizacionV3 struct {
	esquemaHuellaSolicitud                string
	solicitudHuellaSHA256                 string
	decisionRef                           string
	concedida                             bool
	codigo                                string
	contextoRecursoHuellaSHA256           string
	asignacionRef                         string
	asignacionHuellaSHA256                string
	versionRolRef                         string
	versionRolHuellaSHA256                string
	controlVigenciaVersionRolRef          string
	controlVigenciaVersionRolRevision     uint64
	controlVigenciaVersionRolHuellaSHA256 string
	revisionCatalogoPoliticas             uint64
	catalogoPoliticasHuellaSHA256         string
	politicasEvaluadas                    []evidenciaPoliticaAutorizacionV3
	politicasAplicables                   []evidenciaPoliticaAutorizacionV3
	garantiaMinima                        AuthAssurance
	camposPermitidos                      []string
	obligaciones                          []string
	emitidaEn                             time.Time
	validaHasta                           time.Time
	selloSHA256                           string
}

// EvidenciaEvaluacionAutorizacionV3 es una capacidad nominal opaca: no hay
// constructor desde DTO, bytes ni DecisionAutorizacion historica.
type EvidenciaEvaluacionAutorizacionV3 struct {
	bloqueoSerializacionEvidenciaEvaluacionAutorizacionV3
	datos *datosEvidenciaEvaluacionAutorizacionV3
}

// NuevaEvidenciaEvaluacionAutorizacionV3 evalua una instantanea completa y
// sella su proyeccion minimizada. validaHastaSolicitada es el limite superior
// configurado; el resultado se recorta por todas las fronteras conocidas.
func NuevaEvidenciaEvaluacionAutorizacionV3(
	solicitud SolicitudAutorizacionLigadaV3,
	instantanea InstantaneaAutorizacion,
	decisionRef string,
	emitidaEn time.Time,
	validaHastaSolicitada time.Time,
) (EvidenciaEvaluacionAutorizacionV3, error) {
	datosSolicitud, proyeccion, err := proyectarSolicitudAutorizacionLigadaV3ParaEvaluacion(solicitud)
	if err != nil || instantanea.Validar() != nil ||
		!textoAutorizacionSinComodinSeguro(decisionRef, 512, false) ||
		!instanteAutorizacionCanonico(emitidaEn) ||
		!instanteAutorizacionCanonico(validaHastaSolicitada) ||
		!validaHastaSolicitada.After(emitidaEn) ||
		validaHastaSolicitada.Sub(emitidaEn) > VigenciaMaximaDecisionAutorizacion {
		return EvidenciaEvaluacionAutorizacionV3{}, errorEvidenciaEvaluacionAutorizacionV3(err)
	}
	vinculo, err := datosSolicitud.VinculoAutenticacionActor.Datos()
	if err != nil || instantanea.AsignacionPerfil.PrincipalID != vinculo.PrincipalID ||
		instantanea.AsignacionPerfil.PerfilActivoRef != vinculo.PerfilActivoRef ||
		emitidaEn.Before(vinculo.SesionRevalidadaEn) || !emitidaEn.Before(vinculo.SesionValidaHasta) {
		return EvidenciaEvaluacionAutorizacionV3{}, errorEvidenciaEvaluacionAutorizacionV3(err)
	}

	huellaSolicitud, err := HuellaSHA256SolicitudAutorizacionV3(solicitud)
	if err != nil {
		return EvidenciaEvaluacionAutorizacionV3{}, errorEvidenciaEvaluacionAutorizacionV3(err)
	}
	huellaContexto, err := datosSolicitud.Recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		return EvidenciaEvaluacionAutorizacionV3{}, errorEvidenciaEvaluacionAutorizacionV3(err)
	}
	huellaAsignacion, err := instantanea.AsignacionPerfil.HuellaSHA256()
	if err != nil {
		return EvidenciaEvaluacionAutorizacionV3{}, errorEvidenciaEvaluacionAutorizacionV3(err)
	}
	huellaRol, err := instantanea.VersionRol.HuellaSHA256()
	if err != nil {
		return EvidenciaEvaluacionAutorizacionV3{}, errorEvidenciaEvaluacionAutorizacionV3(err)
	}
	huellaControl, err := instantanea.ControlVigenciaVersionRol.HuellaSHA256()
	if err != nil {
		return EvidenciaEvaluacionAutorizacionV3{}, errorEvidenciaEvaluacionAutorizacionV3(err)
	}

	politicasOrdenadas := append([]PoliticaRestrictiva(nil), instantanea.Politicas...)
	sort.Slice(politicasOrdenadas, func(i, j int) bool {
		return politicasOrdenadas[i].Referencia() < politicasOrdenadas[j].Referencia()
	})
	politicasEvaluadas := make([]evidenciaPoliticaAutorizacionV3, 0, len(politicasOrdenadas))
	politicasAplicables := make([]evidenciaPoliticaAutorizacionV3, 0, len(politicasOrdenadas))
	for _, politica := range politicasOrdenadas {
		huella, errHuella := politica.HuellaSHA256()
		if errHuella != nil {
			return EvidenciaEvaluacionAutorizacionV3{}, errorEvidenciaEvaluacionAutorizacionV3(errHuella)
		}
		entrada := evidenciaPoliticaAutorizacionV3{referencia: politica.Referencia(), huellaSHA256: huella}
		politicasEvaluadas = append(politicasEvaluadas, entrada)
		if politica.VigenteEn(emitidaEn) && politica.AplicaA(proyeccion) {
			politicasAplicables = append(politicasAplicables, entrada)
		}
	}

	concedida, codigo, garantia, campos, obligaciones, err := evaluarResultadoAutorizacionV3(
		proyeccion,
		vinculo.GarantiaObservada,
		instantanea,
		politicasOrdenadas,
		emitidaEn,
	)
	if err != nil {
		return EvidenciaEvaluacionAutorizacionV3{}, errorEvidenciaEvaluacionAutorizacionV3(err)
	}
	validaHasta := limitarVentanaEvaluacionAutorizacionV3(
		emitidaEn,
		validaHastaSolicitada,
		vinculo,
		instantanea,
	)
	if !validaHasta.After(emitidaEn) {
		return EvidenciaEvaluacionAutorizacionV3{}, errorEvidenciaEvaluacionAutorizacionV3(nil)
	}

	datos := &datosEvidenciaEvaluacionAutorizacionV3{
		esquemaHuellaSolicitud: EsquemaHuellaSolicitudAutorizacionV3,
		solicitudHuellaSHA256:  huellaSolicitud,
		decisionRef:            decisionRef, concedida: concedida, codigo: codigo,
		contextoRecursoHuellaSHA256: huellaContexto,
		asignacionRef:               instantanea.AsignacionPerfil.Referencia(), asignacionHuellaSHA256: huellaAsignacion,
		versionRolRef: instantanea.VersionRol.Referencia(), versionRolHuellaSHA256: huellaRol,
		controlVigenciaVersionRolRef:          instantanea.ControlVigenciaVersionRol.VersionRolRef,
		controlVigenciaVersionRolRevision:     instantanea.ControlVigenciaVersionRol.Revision,
		controlVigenciaVersionRolHuellaSHA256: huellaControl,
		revisionCatalogoPoliticas:             instantanea.RevisionCatalogoPoliticas,
		catalogoPoliticasHuellaSHA256:         instantanea.CatalogoPoliticasHuellaSHA256,
		politicasEvaluadas:                    clonarEvidenciasPoliticaAutorizacionV3(politicasEvaluadas),
		politicasAplicables:                   clonarEvidenciasPoliticaAutorizacionV3(politicasAplicables),
		garantiaMinima:                        garantia,
		camposPermitidos:                      normalizarListaAutorizacionV3(campos),
		obligaciones:                          normalizarListaAutorizacionV3(obligaciones),
		emitidaEn:                             emitidaEn, validaHasta: validaHasta,
	}
	datos.selloSHA256, err = huellaCanonicaEvidenciaEvaluacionAutorizacionV3(datos)
	if err != nil {
		return EvidenciaEvaluacionAutorizacionV3{}, errorEvidenciaEvaluacionAutorizacionV3(err)
	}
	evidencia := EvidenciaEvaluacionAutorizacionV3{datos: datos}
	if err := evidencia.ValidarPara(solicitud); err != nil {
		return EvidenciaEvaluacionAutorizacionV3{}, err
	}
	return evidencia, nil
}

func (e EvidenciaEvaluacionAutorizacionV3) Validar() error {
	if e.datos == nil || validarDatosEvidenciaEvaluacionAutorizacionV3(e.datos) != nil {
		return errorEvidenciaEvaluacionAutorizacionV3(nil)
	}
	huella, err := huellaCanonicaEvidenciaEvaluacionAutorizacionV3(e.datos)
	if err != nil || huella != e.datos.selloSHA256 {
		return errorEvidenciaEvaluacionAutorizacionV3(err)
	}
	return nil
}

func (e EvidenciaEvaluacionAutorizacionV3) ValidarPara(solicitud SolicitudAutorizacionLigadaV3) error {
	if e.Validar() != nil {
		return errorEvidenciaEvaluacionAutorizacionV3(nil)
	}
	datosSolicitud, _, err := proyectarSolicitudAutorizacionLigadaV3ParaEvaluacion(solicitud)
	if err != nil {
		return errorEvidenciaEvaluacionAutorizacionV3(err)
	}
	huellaSolicitud, err := HuellaSHA256SolicitudAutorizacionV3(solicitud)
	if err != nil {
		return errorEvidenciaEvaluacionAutorizacionV3(err)
	}
	huellaContexto, err := datosSolicitud.Recurso.HuellaContextoAutorizacionSHA256()
	vinculo, errVinculo := datosSolicitud.VinculoAutenticacionActor.Datos()
	if err != nil || errVinculo != nil ||
		e.datos.esquemaHuellaSolicitud != EsquemaHuellaSolicitudAutorizacionV3 ||
		e.datos.solicitudHuellaSHA256 != huellaSolicitud ||
		e.datos.contextoRecursoHuellaSHA256 != huellaContexto ||
		e.datos.emitidaEn.Before(vinculo.SesionRevalidadaEn) ||
		!e.datos.emitidaEn.Before(vinculo.SesionValidaHasta) ||
		e.datos.validaHasta.After(vinculo.SesionValidaHasta) ||
		(e.datos.concedida && !CumpleGarantiaAutenticacion(vinculo.GarantiaObservada, e.datos.garantiaMinima)) {
		return errorEvidenciaEvaluacionAutorizacionV3(errors.Join(err, errVinculo))
	}
	return nil
}

func validarDatosEvidenciaEvaluacionAutorizacionV3(d *datosEvidenciaEvaluacionAutorizacionV3) error {
	if d == nil || d.esquemaHuellaSolicitud != EsquemaHuellaSolicitudAutorizacionV3 ||
		!huellaSHA256AutorizacionV3NoNula(d.solicitudHuellaSHA256) ||
		!textoAutorizacionSinComodinSeguro(d.decisionRef, 512, false) ||
		!codigoResultadoEvaluacionAutorizacionV3Valido(d.codigo, d.concedida) ||
		!huellaSHA256AutorizacionValida(d.contextoRecursoHuellaSHA256) ||
		!textoAutorizacionSinComodinSeguro(d.asignacionRef, 512, false) ||
		!huellaSHA256AutorizacionValida(d.asignacionHuellaSHA256) ||
		!textoAutorizacionSinComodinSeguro(d.versionRolRef, 512, false) ||
		!huellaSHA256AutorizacionValida(d.versionRolHuellaSHA256) ||
		d.controlVigenciaVersionRolRef != d.versionRolRef ||
		d.controlVigenciaVersionRolRevision == 0 ||
		!huellaSHA256AutorizacionValida(d.controlVigenciaVersionRolHuellaSHA256) ||
		d.revisionCatalogoPoliticas == 0 ||
		!huellaSHA256AutorizacionValida(d.catalogoPoliticasHuellaSHA256) ||
		!evidenciasPoliticaAutorizacionV3Validas(d.politicasEvaluadas) ||
		!evidenciasPoliticaAutorizacionV3Validas(d.politicasAplicables) ||
		!subconjuntoPoliticasAutorizacionV3Valido(d.politicasEvaluadas, d.politicasAplicables) ||
		!listaAutorizacionValida(d.camposPermitidos, false, false) ||
		!listaAutorizacionValida(d.obligaciones, false, false) ||
		!listaOrdenadaUnicaAutorizacionV3(d.camposPermitidos) ||
		!listaOrdenadaUnicaAutorizacionV3(d.obligaciones) ||
		!instanteAutorizacionCanonico(d.emitidaEn) ||
		!instanteAutorizacionCanonico(d.validaHasta) ||
		!d.validaHasta.After(d.emitidaEn) ||
		d.validaHasta.Sub(d.emitidaEn) > VigenciaMaximaDecisionAutorizacion ||
		!huellaSHA256AutorizacionV3NoNula(d.selloSHA256) {
		return ErrEvidenciaEvaluacionAutorizacionV3Invalida
	}
	if d.garantiaMinima != "" && !d.garantiaMinima.Valida() {
		return ErrEvidenciaEvaluacionAutorizacionV3Invalida
	}
	if d.concedida && !d.garantiaMinima.Valida() {
		return ErrEvidenciaEvaluacionAutorizacionV3Invalida
	}
	referencias := make([]string, 0, len(d.politicasEvaluadas))
	huellas := make(map[string]string, len(d.politicasEvaluadas))
	for _, politica := range d.politicasEvaluadas {
		referencias = append(referencias, politica.referencia)
		huellas[politica.referencia] = politica.huellaSHA256
	}
	huellaCatalogo, err := HuellaEvidenciasCatalogoPoliticasAutorizacion(referencias, huellas)
	if err != nil || huellaCatalogo != d.catalogoPoliticasHuellaSHA256 {
		return ErrEvidenciaEvaluacionAutorizacionV3Invalida
	}
	return nil
}

type datosDecisionAutorizacionLigadaV3 struct {
	bloqueVersion                         uint16
	decisionRef                           string
	concedida                             bool
	codigo                                string
	principalID                           string
	perfilActivoRef                       string
	accion                                string
	recursoRef                            string
	moduloID                              string
	tipoRecurso                           string
	contextoRecursoHuellaSHA256           string
	finalidad                             string
	correlacionRef                        string
	esquemaHuellaSolicitud                string
	solicitudHuellaSHA256                 string
	esquemaHuellaMotivo                   string
	motivoHuellaSHA256                    string
	vinculoAutenticacionActor             VinculoAutenticacionActorV2
	asignacionRef                         string
	asignacionHuellaSHA256                string
	versionRolRef                         string
	versionRolHuellaSHA256                string
	controlVigenciaVersionRolRef          string
	controlVigenciaVersionRolRevision     uint64
	controlVigenciaVersionRolHuellaSHA256 string
	revisionCatalogoPoliticas             uint64
	catalogoPoliticasHuellaSHA256         string
	politicasEvaluadas                    []evidenciaPoliticaAutorizacionV3
	politicasAplicables                   []evidenciaPoliticaAutorizacionV3
	garantiaMinima                        AuthAssurance
	camposPermitidos                      []string
	obligaciones                          []string
	emitidaEn                             time.Time
	validaHasta                           time.Time
	selloSHA256                           string
}

// DecisionAutorizacionLigadaV3 es un documento nominal opaco sin conversion
// ni downgrade a DecisionAutorizacion. Solo nace de solicitud y evidencia V3.
// La evaluacion por si sola no acredita el registro durable ni el CAS de la
// instantanea y, por tanto, este tipo no es una capacidad ejecutable.
type DecisionAutorizacionLigadaV3 struct {
	bloqueoSerializacionDecisionAutorizacionLigadaV3
	datos *datosDecisionAutorizacionLigadaV3
}

func NuevaDecisionAutorizacionLigadaV3(
	solicitud SolicitudAutorizacionLigadaV3,
	evidencia EvidenciaEvaluacionAutorizacionV3,
) (DecisionAutorizacionLigadaV3, error) {
	if err := evidencia.ValidarPara(solicitud); err != nil {
		return DecisionAutorizacionLigadaV3{}, errorDecisionAutorizacionLigadaV3(err)
	}
	s, _, err := proyectarSolicitudAutorizacionLigadaV3ParaEvaluacion(solicitud)
	if err != nil {
		return DecisionAutorizacionLigadaV3{}, errorDecisionAutorizacionLigadaV3(err)
	}
	vinculo, err := clonarVinculoAutenticacionActorV2(s.VinculoAutenticacionActor)
	if err != nil {
		return DecisionAutorizacionLigadaV3{}, errorDecisionAutorizacionLigadaV3(err)
	}
	vinculoDatos, _ := vinculo.Datos()
	correlacion, err := s.Correlacion.ValorCanonico()
	if err != nil {
		return DecisionAutorizacionLigadaV3{}, errorDecisionAutorizacionLigadaV3(err)
	}
	huellaMotivo, err := HuellaSHA256MotivoAutorizacionV2(s.ReferenciaMotivo)
	if err != nil {
		return DecisionAutorizacionLigadaV3{}, errorDecisionAutorizacionLigadaV3(err)
	}
	e := evidencia.datos
	datos := &datosDecisionAutorizacionLigadaV3{
		bloqueVersion: VersionDecisionAutorizacionLigadaV3,
		decisionRef:   e.decisionRef, concedida: e.concedida, codigo: e.codigo,
		principalID: vinculoDatos.PrincipalID, perfilActivoRef: vinculoDatos.PerfilActivoRef,
		accion: s.Accion, recursoRef: s.Recurso.Referencia, moduloID: s.Recurso.ModuloID,
		tipoRecurso: s.Recurso.Tipo, contextoRecursoHuellaSHA256: e.contextoRecursoHuellaSHA256,
		finalidad: s.Finalidad, correlacionRef: correlacion,
		esquemaHuellaSolicitud: e.esquemaHuellaSolicitud, solicitudHuellaSHA256: e.solicitudHuellaSHA256,
		esquemaHuellaMotivo: EsquemaHuellaMotivoAutorizacionV2, motivoHuellaSHA256: huellaMotivo,
		vinculoAutenticacionActor: vinculo,
		asignacionRef:             e.asignacionRef, asignacionHuellaSHA256: e.asignacionHuellaSHA256,
		versionRolRef: e.versionRolRef, versionRolHuellaSHA256: e.versionRolHuellaSHA256,
		controlVigenciaVersionRolRef:          e.controlVigenciaVersionRolRef,
		controlVigenciaVersionRolRevision:     e.controlVigenciaVersionRolRevision,
		controlVigenciaVersionRolHuellaSHA256: e.controlVigenciaVersionRolHuellaSHA256,
		revisionCatalogoPoliticas:             e.revisionCatalogoPoliticas,
		catalogoPoliticasHuellaSHA256:         e.catalogoPoliticasHuellaSHA256,
		politicasEvaluadas:                    clonarEvidenciasPoliticaAutorizacionV3(e.politicasEvaluadas),
		politicasAplicables:                   clonarEvidenciasPoliticaAutorizacionV3(e.politicasAplicables),
		garantiaMinima:                        e.garantiaMinima,
		camposPermitidos:                      append([]string(nil), e.camposPermitidos...),
		obligaciones:                          append([]string(nil), e.obligaciones...),
		emitidaEn:                             e.emitidaEn, validaHasta: e.validaHasta,
	}
	datos.selloSHA256, err = huellaCanonicaDecisionAutorizacionV3(datos)
	if err != nil {
		return DecisionAutorizacionLigadaV3{}, errorDecisionAutorizacionLigadaV3(err)
	}
	decision := DecisionAutorizacionLigadaV3{datos: datos}
	if err := decision.ValidarPara(solicitud); err != nil {
		return DecisionAutorizacionLigadaV3{}, err
	}
	return decision, nil
}

func (d DecisionAutorizacionLigadaV3) Validar() error {
	if d.datos == nil || validarDatosDecisionAutorizacionLigadaV3(d.datos) != nil {
		return errorDecisionAutorizacionLigadaV3(nil)
	}
	huella, err := huellaCanonicaDecisionAutorizacionV3(d.datos)
	if err != nil || huella != d.datos.selloSHA256 {
		return errorDecisionAutorizacionLigadaV3(err)
	}
	return nil
}

func (d DecisionAutorizacionLigadaV3) ValidarPara(solicitud SolicitudAutorizacionLigadaV3) error {
	if d.Validar() != nil {
		return errorDecisionAutorizacionLigadaV3(nil)
	}
	s, _, err := proyectarSolicitudAutorizacionLigadaV3ParaEvaluacion(solicitud)
	if err != nil {
		return errorDecisionAutorizacionLigadaV3(err)
	}
	huellaSolicitud, err := HuellaSHA256SolicitudAutorizacionV3(solicitud)
	huellaMotivo, errMotivo := HuellaSHA256MotivoAutorizacionV2(s.ReferenciaMotivo)
	huellaContexto, errContexto := s.Recurso.HuellaContextoAutorizacionSHA256()
	correlacion, errCorrelacion := s.Correlacion.ValorCanonico()
	vinculo, errVinculo := s.VinculoAutenticacionActor.Datos()
	if err != nil || errMotivo != nil || errContexto != nil || errCorrelacion != nil || errVinculo != nil ||
		d.datos.principalID != vinculo.PrincipalID || d.datos.perfilActivoRef != vinculo.PerfilActivoRef ||
		d.datos.accion != s.Accion || d.datos.recursoRef != s.Recurso.Referencia ||
		d.datos.moduloID != s.Recurso.ModuloID || d.datos.tipoRecurso != s.Recurso.Tipo ||
		d.datos.contextoRecursoHuellaSHA256 != huellaContexto || d.datos.finalidad != s.Finalidad ||
		d.datos.correlacionRef != correlacion ||
		d.datos.esquemaHuellaSolicitud != EsquemaHuellaSolicitudAutorizacionV3 ||
		d.datos.solicitudHuellaSHA256 != huellaSolicitud ||
		d.datos.esquemaHuellaMotivo != EsquemaHuellaMotivoAutorizacionV2 ||
		d.datos.motivoHuellaSHA256 != huellaMotivo ||
		!d.datos.vinculoAutenticacionActor.CoincideExactamenteCon(s.VinculoAutenticacionActor) {
		return errorDecisionAutorizacionLigadaV3(errors.Join(
			err, errMotivo, errContexto, errCorrelacion, errVinculo,
		))
	}
	return nil
}

func validarDatosDecisionAutorizacionLigadaV3(d *datosDecisionAutorizacionLigadaV3) error {
	if d == nil || d.bloqueVersion != VersionDecisionAutorizacionLigadaV3 ||
		!textoAutorizacionSinComodinSeguro(d.decisionRef, 512, false) ||
		!codigoResultadoEvaluacionAutorizacionV3Valido(d.codigo, d.concedida) ||
		!textoAutorizacionSinComodinSeguro(d.principalID, 512, false) ||
		!textoAutorizacionSinComodinSeguro(d.perfilActivoRef, 512, false) ||
		!textoAutorizacionSinComodinSeguro(d.accion, 256, false) ||
		!textoAutorizacionSinComodinSeguro(d.recursoRef, 512, false) ||
		!textoAutorizacionSinComodinSeguro(d.moduloID, 128, false) ||
		!textoAutorizacionSinComodinSeguro(d.tipoRecurso, 128, false) ||
		!huellaSHA256AutorizacionValida(d.contextoRecursoHuellaSHA256) ||
		!textoAutorizacionSinComodinSeguro(d.finalidad, 512, false) ||
		!ReferenciaCorrelacionAutorizacionV2Valida(d.correlacionRef) ||
		d.esquemaHuellaSolicitud != EsquemaHuellaSolicitudAutorizacionV3 ||
		!huellaSHA256AutorizacionV3NoNula(d.solicitudHuellaSHA256) ||
		d.esquemaHuellaMotivo != EsquemaHuellaMotivoAutorizacionV2 ||
		!huellaSHA256AutorizacionV3NoNula(d.motivoHuellaSHA256) ||
		d.vinculoAutenticacionActor.Validar() != nil ||
		!textoAutorizacionSinComodinSeguro(d.asignacionRef, 512, false) ||
		!huellaSHA256AutorizacionValida(d.asignacionHuellaSHA256) ||
		!textoAutorizacionSinComodinSeguro(d.versionRolRef, 512, false) ||
		!huellaSHA256AutorizacionValida(d.versionRolHuellaSHA256) ||
		d.controlVigenciaVersionRolRef != d.versionRolRef ||
		d.controlVigenciaVersionRolRevision == 0 ||
		!huellaSHA256AutorizacionValida(d.controlVigenciaVersionRolHuellaSHA256) ||
		d.revisionCatalogoPoliticas == 0 ||
		!huellaSHA256AutorizacionValida(d.catalogoPoliticasHuellaSHA256) ||
		!evidenciasPoliticaAutorizacionV3Validas(d.politicasEvaluadas) ||
		!evidenciasPoliticaAutorizacionV3Validas(d.politicasAplicables) ||
		!subconjuntoPoliticasAutorizacionV3Valido(d.politicasEvaluadas, d.politicasAplicables) ||
		!listaAutorizacionValida(d.camposPermitidos, false, false) ||
		!listaAutorizacionValida(d.obligaciones, false, false) ||
		!listaOrdenadaUnicaAutorizacionV3(d.camposPermitidos) ||
		!listaOrdenadaUnicaAutorizacionV3(d.obligaciones) ||
		!instanteAutorizacionCanonico(d.emitidaEn) || !instanteAutorizacionCanonico(d.validaHasta) ||
		!d.validaHasta.After(d.emitidaEn) || d.validaHasta.Sub(d.emitidaEn) > VigenciaMaximaDecisionAutorizacion ||
		!huellaSHA256AutorizacionV3NoNula(d.selloSHA256) {
		return ErrDecisionAutorizacionLigadaV3Invalida
	}
	vinculo, err := d.vinculoAutenticacionActor.Datos()
	if err != nil || d.principalID != vinculo.PrincipalID || d.perfilActivoRef != vinculo.PerfilActivoRef ||
		d.emitidaEn.Before(vinculo.SesionRevalidadaEn) || !d.emitidaEn.Before(vinculo.SesionValidaHasta) ||
		d.validaHasta.After(vinculo.SesionValidaHasta) ||
		(d.garantiaMinima != "" && !d.garantiaMinima.Valida()) ||
		(d.concedida && (!d.garantiaMinima.Valida() ||
			!CumpleGarantiaAutenticacion(vinculo.GarantiaObservada, d.garantiaMinima))) {
		return ErrDecisionAutorizacionLigadaV3Invalida
	}
	referencias := make([]string, 0, len(d.politicasEvaluadas))
	huellas := make(map[string]string, len(d.politicasEvaluadas))
	for _, politica := range d.politicasEvaluadas {
		referencias = append(referencias, politica.referencia)
		huellas[politica.referencia] = politica.huellaSHA256
	}
	huellaCatalogo, err := HuellaEvidenciasCatalogoPoliticasAutorizacion(referencias, huellas)
	if err != nil || huellaCatalogo != d.catalogoPoliticasHuellaSHA256 {
		return ErrDecisionAutorizacionLigadaV3Invalida
	}
	return nil
}

// Resultado devuelve el resultado minimo sin exponer la instantanea sellada.
func (d DecisionAutorizacionLigadaV3) Resultado() (bool, string, error) {
	if err := d.Validar(); err != nil {
		return false, "", err
	}
	return d.datos.concedida, d.datos.codigo, nil
}

func (d DecisionAutorizacionLigadaV3) VentanaValidez() (time.Time, time.Time, error) {
	if err := d.Validar(); err != nil {
		return time.Time{}, time.Time{}, err
	}
	return d.datos.emitidaEn, d.datos.validaHasta, nil
}

// VigenteEn falla cerrado porque NuevaDecisionAutorizacionLigadaV3 solo prueba
// la evaluacion en memoria. Un futuro flujo productivo debe devolver otro tipo
// nominal creado exclusivamente tras COMMIT/CAS durable; reutilizar este tipo
// permitiria ejecutar una concesion que nunca llego a registrarse.
func (d DecisionAutorizacionLigadaV3) VigenteEn(time.Time) bool {
	return false
}

func proyectarSolicitudAutorizacionLigadaV3ParaEvaluacion(
	s SolicitudAutorizacionLigadaV3,
) (DatosSolicitudAutorizacionLigadaV3, SolicitudAutorizacion, error) {
	datos, err := s.Datos()
	if err != nil {
		return DatosSolicitudAutorizacionLigadaV3{}, SolicitudAutorizacion{}, err
	}
	vinculo, err := datos.VinculoAutenticacionActor.Datos()
	if err != nil {
		return DatosSolicitudAutorizacionLigadaV3{}, SolicitudAutorizacion{}, err
	}
	correlacion, err := datos.Correlacion.ValorCanonico()
	if err != nil {
		return DatosSolicitudAutorizacionLigadaV3{}, SolicitudAutorizacion{}, err
	}
	proyeccion := SolicitudAutorizacion{
		Principal:       Principal{ID: vinculo.PrincipalID, AuthMethod: vinculo.MetodoObservado, AuthAssurance: vinculo.GarantiaObservada},
		PerfilActivoRef: vinculo.PerfilActivoRef,
		Accion:          datos.Accion, Recurso: datos.Recurso, Finalidad: datos.Finalidad,
		CorrelacionRef: correlacion, Motivo: datos.ReferenciaMotivo.EntradaClave,
		ReferenciaMotivo: datos.ReferenciaMotivo,
	}
	if proyeccion.Validar() != nil {
		return DatosSolicitudAutorizacionLigadaV3{}, SolicitudAutorizacion{}, ErrSolicitudAutorizacionLigadaV3Invalida
	}
	return datos, proyeccion, nil
}

func evaluarResultadoAutorizacionV3(
	solicitud SolicitudAutorizacion,
	garantiaObservada AuthAssurance,
	instantanea InstantaneaAutorizacion,
	politicasOrdenadas []PoliticaRestrictiva,
	instante time.Time,
) (bool, string, AuthAssurance, []string, []string, error) {
	asignacion := instantanea.AsignacionPerfil
	version := instantanea.VersionRol
	control := instantanea.ControlVigenciaVersionRol
	if !asignacion.VigenteEn(instante) {
		return false, "perfil_no_vigente", "", nil, nil, nil
	}
	if !asignacion.Cubre(solicitud.Recurso) {
		return false, "ambito_no_autorizado", "", nil, nil, nil
	}
	if version.Estado != EstadoVersionRolPublicada || version.PublicadaEn.After(instante) ||
		version.Referencia() != asignacion.VersionRolRef {
		return false, "rol_no_publicado", "", nil, nil, nil
	}
	if control.Estado != EstadoControlVigenciaVersionRolHabilitada {
		return false, "rol_retirado", "", nil, nil, nil
	}
	var concesion ConcesionRol
	encontrada := false
	for _, candidata := range version.Concesiones {
		if candidata.Accion == solicitud.Accion && candidata.ModuloID == solicitud.Recurso.ModuloID &&
			candidata.TipoRecurso == solicitud.Recurso.Tipo {
			concesion, encontrada = candidata, true
			break
		}
	}
	if !encontrada {
		return false, "accion_no_concedida", "", nil, nil, nil
	}
	if !concesion.AdmiteFinalidad(solicitud.Finalidad) {
		return false, "finalidad_no_autorizada", "", nil, nil, nil
	}
	garantia := concesion.GarantiaMinima
	campos := append([]string(nil), concesion.CamposPermitidos...)
	obligaciones := append([]string(nil), concesion.Obligaciones...)
	deniegaPolitica := false
	incumpleRestriccion := false
	for _, politica := range politicasOrdenadas {
		if !politica.VigenteEn(instante) || !politica.AplicaA(solicitud) {
			continue
		}
		if politica.Efecto == EfectoPoliticaDenegar {
			deniegaPolitica = true
		}
		if politica.Efecto == EfectoPoliticaRestringir && !politica.Cumple(solicitud) {
			incumpleRestriccion = true
		}
		if politica.GarantiaMinima != "" {
			var err error
			garantia, err = GarantiaAutenticacionMasAlta(garantia, politica.GarantiaMinima)
			if err != nil {
				return false, "", "", nil, nil, err
			}
		}
		if politica.RestringeCampos {
			campos = interseccionCamposAutorizacionV3(campos, politica.CamposPermitidos)
		}
		obligaciones = append(obligaciones, politica.Obligaciones...)
	}
	campos = normalizarListaAutorizacionV3(campos)
	obligaciones = normalizarListaAutorizacionV3(obligaciones)
	if !listaAutorizacionValida(campos, false, false) || !listaAutorizacionValida(obligaciones, false, false) {
		return false, "", "", nil, nil, ErrConfiguracionAccesoInvalida
	}
	if deniegaPolitica {
		return false, "denegada_por_politica", garantia, campos, obligaciones, nil
	}
	if incumpleRestriccion {
		return false, "restriccion_abac_incumplida", garantia, campos, obligaciones, nil
	}
	if !CumpleGarantiaAutenticacion(garantiaObservada, garantia) {
		return false, "garantia_insuficiente", garantia, campos, obligaciones, nil
	}
	return true, "concedida", garantia, campos, obligaciones, nil
}

func limitarVentanaEvaluacionAutorizacionV3(
	instante time.Time,
	limite time.Time,
	vinculo DatosVinculoAutenticacionActorV2,
	instantanea InstantaneaAutorizacion,
) time.Time {
	fronteras := []time.Time{
		vinculo.SesionValidaHasta,
		instantanea.AsignacionPerfil.VigenteDesde,
		instantanea.AsignacionPerfil.VigenteHasta,
		instantanea.VersionRol.PublicadaEn,
	}
	if !instantanea.VersionRol.RetiradaEn.IsZero() {
		fronteras = append(fronteras, instantanea.VersionRol.RetiradaEn)
	}
	for _, politica := range instantanea.Politicas {
		if politica.Estado == EstadoPoliticaRestrictivaPublicada {
			fronteras = append(fronteras, politica.VigenteDesde, politica.VigenteHasta)
		}
	}
	for _, frontera := range fronteras {
		if frontera.After(instante) && frontera.Before(limite) {
			limite = frontera
		}
	}
	return limite
}

func interseccionCamposAutorizacionV3(actuales, restriccion []string) []string {
	if contieneAutorizacionExacta(restriccion, comodinAutorizacion) {
		return append([]string(nil), actuales...)
	}
	permitidos := make(map[string]struct{}, len(restriccion))
	for _, campo := range restriccion {
		permitidos[campo] = struct{}{}
	}
	resultado := make([]string, 0, len(actuales))
	for _, campo := range actuales {
		if _, permitido := permitidos[campo]; permitido {
			resultado = append(resultado, campo)
		}
	}
	return resultado
}

func normalizarListaAutorizacionV3(valores []string) []string {
	unicos := make(map[string]struct{}, len(valores))
	for _, valor := range valores {
		unicos[valor] = struct{}{}
	}
	resultado := make([]string, 0, len(unicos))
	for valor := range unicos {
		resultado = append(resultado, valor)
	}
	sort.Strings(resultado)
	return resultado
}

func listaOrdenadaUnicaAutorizacionV3(valores []string) bool {
	return sort.StringsAreSorted(valores) && len(normalizarListaAutorizacionV3(valores)) == len(valores)
}

func evidenciasPoliticaAutorizacionV3Validas(valores []evidenciaPoliticaAutorizacionV3) bool {
	if len(valores) > maximoElementosAutorizacion {
		return false
	}
	anterior := ""
	for indice, valor := range valores {
		if !textoAutorizacionSinComodinSeguro(valor.referencia, 512, false) ||
			!huellaSHA256AutorizacionValida(valor.huellaSHA256) ||
			(indice > 0 && valor.referencia <= anterior) {
			return false
		}
		anterior = valor.referencia
	}
	return true
}

func subconjuntoPoliticasAutorizacionV3Valido(
	evaluadas []evidenciaPoliticaAutorizacionV3,
	aplicables []evidenciaPoliticaAutorizacionV3,
) bool {
	porReferencia := make(map[string]string, len(evaluadas))
	for _, politica := range evaluadas {
		porReferencia[politica.referencia] = politica.huellaSHA256
	}
	for _, politica := range aplicables {
		if porReferencia[politica.referencia] != politica.huellaSHA256 {
			return false
		}
	}
	return true
}

func clonarEvidenciasPoliticaAutorizacionV3(
	valores []evidenciaPoliticaAutorizacionV3,
) []evidenciaPoliticaAutorizacionV3 {
	return append([]evidenciaPoliticaAutorizacionV3(nil), valores...)
}

func clonarVinculoAutenticacionActorV2(v VinculoAutenticacionActorV2) (VinculoAutenticacionActorV2, error) {
	datos, err := v.Datos()
	if err != nil {
		return VinculoAutenticacionActorV2{}, err
	}
	return VinculoAutenticacionActorV2{datos: &datos}, nil
}

func codigoResultadoEvaluacionAutorizacionV3Valido(codigo string, concedida bool) bool {
	if concedida {
		return codigo == "concedida"
	}
	switch codigo {
	case "perfil_no_vigente", "ambito_no_autorizado", "rol_no_publicado", "rol_retirado",
		"accion_no_concedida", "finalidad_no_autorizada", "denegada_por_politica",
		"restriccion_abac_incumplida", "garantia_insuficiente":
		return true
	default:
		return false
	}
}

// CodigoResultadoEvaluacionAutorizacionV3Valido es la única clasificación
// nominal autoritativa de resultados V3. Los módulos consumidores deben
// reutilizarla y no mantener listas divergentes de códigos.
func CodigoResultadoEvaluacionAutorizacionV3Valido(
	codigo string,
	concedida bool,
) bool {
	return codigoResultadoEvaluacionAutorizacionV3Valido(codigo, concedida)
}

func errorEvidenciaEvaluacionAutorizacionV3(causa error) error {
	return errors.Join(ErrEvidenciaEvaluacionAutorizacionV3Invalida, causa)
}

func errorDecisionAutorizacionLigadaV3(causa error) error {
	return errors.Join(ErrDecisionAutorizacionInvalida, ErrDecisionAutorizacionLigadaV3Invalida, causa)
}
