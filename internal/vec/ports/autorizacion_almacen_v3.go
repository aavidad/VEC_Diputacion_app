package ports

import (
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

// Contexto de almacén desde una decisión V3 registrada.
//
// La aplicación autoriza con V3 y DecisionAutorizacionLigadaV3 no se convierte
// al formato V1/V2. Estas fábricas derivan la capacidad de almacén
// directamente de tres piezas nominales que el PDP V3 ya produce:
//
//   - la solicitud exacta evaluada (acción, recurso, finalidad, correlación,
//     motivo y vínculo de actor);
//   - la decisión V3 sellada, que debe ser una concesión y estar ligada a esa
//     solicitud (ValidarPara);
//   - la confirmación de registro durable de esa misma decisión (misma
//     referencia, huella y ventana), sin la cual no hay capacidad.
//
// Además se exige la proyección exacta: la decisión debe conceder solo los
// campos del plan y ninguna obligación. El recurso evaluado debe vincular la
// operación (vínculos de almacén) igual que en V1, y la acción de la solicitud
// debe ser la acción de negocio del plan. Solo hay fábricas V3 para las
// operaciones que la composición necesita: custodiar el documento firmado de
// un expediente y leer el original de un documento generado.
//
// La capacidad V3 no expone EvidenciaAutorizacion (esa evidencia es V1) y la
// concesión V3 de almacén NO se consume de forma única por sí misma: vale
// dentro de su ventana registrada. Por eso:
//   - la lectura del original depende además de la descarga SQL de Documentos,
//     que consume en su transacción la concesión documentos.original.descargar;
//   - la custodia del firmado es un efecto y no puede componerse hasta que
//     exista un consumidor durable propio de documentos.firmado.custodiar
//     (migración AD3 nueva) que consuma la concesión en la misma transacción
//     que registra el documento. Sin él, la composición no debe publicarla.

// marcaPlanDecisionV3 separa las huellas de plan de contextos V3 de las V1.
const marcaPlanDecisionV3 = "decision:v3"

type evidenciaAlmacenV3 struct {
	solicitud    domain.SolicitudAutorizacionLigadaV3
	decision     domain.DecisionAutorizacionLigadaV3
	confirmacion ConfirmacionRegistroConcesionAutorizacionLigadaV3
}

// NuevoContextoCustodiarDocumentoFirmadoExpedienteAlmacenV3 deriva la
// escritura del documento firmado de un expediente desde una concesión V3
// registrada de documentos.firmado.custodiar.
func NuevoContextoCustodiarDocumentoFirmadoExpedienteAlmacenV3(
	solicitud domain.SolicitudAutorizacionLigadaV3,
	decision domain.DecisionAutorizacionLigadaV3,
	confirmacion ConfirmacionRegistroConcesionAutorizacionLigadaV3,
	vinculos VinculosOperacionAlmacen,
	verificadaEn time.Time,
) (ContextoOperacionAlmacen, error) {
	return nuevoContextoOperacionAlmacenV3(solicitud, decision, confirmacion, vinculos, verificadaEn,
		especificacionCustodiarDocumentoFirmadoExpediente())
}

// NuevoContextoEscribirOriginalFirmableAlmacenV3 deriva solo la escritura
// del objeto del intento reservado; exige una concesión PDP registrada propia.
func NuevoContextoEscribirOriginalFirmableAlmacenV3(
	solicitud domain.SolicitudAutorizacionLigadaV3,
	decision domain.DecisionAutorizacionLigadaV3,
	confirmacion ConfirmacionRegistroConcesionAutorizacionLigadaV3,
	vinculos VinculosOperacionAlmacen,
	verificadaEn time.Time,
) (ContextoOperacionAlmacen, error) {
	return nuevoContextoOperacionAlmacenV3(solicitud, decision, confirmacion, vinculos, verificadaEn,
		especificacionEscribirOriginalFirmable())
}

// NuevoContextoLeerDocumentoGeneradoAlmacenV3 deriva la lectura del objeto
// exacto de un original desde una concesión V3 registrada de
// documentos.original.descargar.
func NuevoContextoLeerDocumentoGeneradoAlmacenV3(
	solicitud domain.SolicitudAutorizacionLigadaV3,
	decision domain.DecisionAutorizacionLigadaV3,
	confirmacion ConfirmacionRegistroConcesionAutorizacionLigadaV3,
	vinculos VinculosOperacionAlmacen,
	verificadaEn time.Time,
) (ContextoOperacionAlmacen, error) {
	return nuevoContextoOperacionAlmacenV3(solicitud, decision, confirmacion, vinculos, verificadaEn,
		especificacionLeerOriginalDocumentoGenerado())
}

func nuevoContextoOperacionAlmacenV3(
	solicitud domain.SolicitudAutorizacionLigadaV3,
	decision domain.DecisionAutorizacionLigadaV3,
	confirmacion ConfirmacionRegistroConcesionAutorizacionLigadaV3,
	vinculos VinculosOperacionAlmacen,
	verificadaEn time.Time,
	especificacion especificacionAutorizacionAlmacen,
) (ContextoOperacionAlmacen, error) {
	if verificadaEn.IsZero() || !especificacion.valida() || !vinculos.validosPara(especificacion) {
		return ContextoOperacionAlmacen{}, errorAutorizacionAlmacen()
	}
	instante := verificadaEn.UTC().Truncate(time.Microsecond)
	evidencia := &evidenciaAlmacenV3{solicitud: solicitud, decision: decision, confirmacion: confirmacion}
	ligada, err := evidencia.validarPara(especificacion, vinculos)
	if err != nil || !evidencia.confirmacion.DentroDeVentanaEn(instante) {
		return ContextoOperacionAlmacen{}, errorAutorizacionAlmacen()
	}
	pasos := clonarPasosOperacionAlmacen(especificacion.pasos)
	primerPaso := pasos[0]
	datos := &datosContextoOperacionAlmacen{
		esquema:      EsquemaContextoOperacionAlmacenV1,
		operacionRef: vinculos.OperacionRef, correlacionRef: ligada.correlacionRef,
		autorizacionRef: ligada.decisionRef, finalidad: ligada.finalidad,
		clasificacion: vinculos.Clasificacion, accionNegocio: especificacion.accionNegocio,
		accionTecnica: primerPaso.accion, cargaRef: vinculos.CargaRef,
		sujetoSeudonimoHMAC: vinculos.SujetoSeudonimoHMAC,
		recursoRef:          ligada.recurso.Referencia, moduloID: ligada.recurso.ModuloID, tipoRecurso: ligada.recurso.Tipo,
		huellaRecursoSHA256: ligada.huellaRecurso, huellaSolicitudHMAC: vinculos.HuellaSolicitudHMAC,
		efectoRef:              vinculos.EfectoRef,
		huellaPlanEfectoSHA256: ligada.huellaPlan(vinculos, especificacion),
		huellaManifiestoSHA256: especificacion.huellaManifiestoSHA256,
		huellaPasoSHA256:       primerPaso.huellaPasoSHA256, pasoRef: primerPaso.referencia,
		objetoVinculado:      vinculos.ObjetoVinculado,
		huellaDecisionSHA256: ligada.huellaDecision,
		verificadaEn:         instante, validaHasta: ligada.validaHasta,
		decisionV3: evidencia, pasos: pasos,
	}
	contexto := ContextoOperacionAlmacen{datos: datos}
	if contexto.validarEstructura() != nil {
		return ContextoOperacionAlmacen{}, errorAutorizacionAlmacen()
	}
	return contexto, nil
}

// decisionAlmacenV3Ligada son los datos ya cotejados entre solicitud,
// decisión y confirmación.
type decisionAlmacenV3Ligada struct {
	decisionRef, huellaDecision, correlacionRef, finalidad, accion, huellaRecurso string
	recurso                                                                       domain.RecursoAutorizable
	camposPermitidos                                                              []string
	validaHasta                                                                   time.Time
}

// validarPara coteja las tres piezas V3 con el plan y los vínculos. Falla
// cerrado ante una decisión denegada, no ligada a la solicitud, de otra
// acción, sin registro o con otro registro, con campos distintos de los del
// plan, con obligaciones o con un recurso que no vincula la operación.
func (e *evidenciaAlmacenV3) validarPara(
	especificacion especificacionAutorizacionAlmacen,
	vinculos VinculosOperacionAlmacen,
) (decisionAlmacenV3Ligada, error) {
	var cero decisionAlmacenV3Ligada
	if e == nil || e.decision.ValidarPara(e.solicitud) != nil || e.confirmacion.Validar() != nil {
		return cero, errorAutorizacionAlmacen()
	}
	concedida, _, err := e.decision.Resultado()
	if err != nil || !concedida {
		return cero, errorAutorizacionAlmacen()
	}
	datos, err := e.solicitud.Datos()
	if err != nil || datos.Accion != especificacion.accionNegocio ||
		datos.Recurso.Validar() != nil || contieneComodinRecursoAlmacen(datos.Recurso) ||
		contieneComodinContextoAlmacen(datos.Accion, datos.Finalidad, datos.Recurso.Referencia,
			datos.Recurso.ModuloID, datos.Recurso.Tipo) ||
		!recursoVinculaOperacionAlmacen(datos.Recurso, vinculos, especificacion) {
		return cero, errorAutorizacionAlmacen()
	}
	if e.decision.ExigirProyeccionPara(e.solicitud, especificacion.camposExactos, nil) != nil {
		return cero, errorAutorizacionAlmacen()
	}
	restricciones, err := e.decision.RestriccionesProyeccionPara(e.solicitud)
	if err != nil || len(restricciones.Obligaciones) != 0 ||
		!camposAutorizacionExactos(restricciones.CamposPermitidos, especificacion.camposExactos) {
		return cero, errorAutorizacionAlmacen()
	}
	resumen, err := resumenDecisionAutorizacionLigadaV3(e.decision)
	huella, errHuella := domain.HuellaSHA256DecisionAutorizacionV3(e.decision)
	registro, errRegistro := e.confirmacion.Datos()
	emitidaEn, validaHasta, errVentana := e.decision.VentanaValidez()
	huellaRecurso, errRecurso := datos.Recurso.HuellaContextoAutorizacionSHA256()
	correlacion, errCorrelacion := datos.Correlacion.ValorCanonico()
	if err != nil || errHuella != nil || errRegistro != nil || errVentana != nil || errRecurso != nil ||
		errCorrelacion != nil || !resumen.Concedida ||
		registro.DecisionRef != resumen.DecisionRef || registro.DecisionHuellaSHA256 != huella ||
		!registro.EmitidaEn.Equal(emitidaEn) || !registro.ValidaHasta.Equal(validaHasta) ||
		!esSHA256Hexadecimal(huella) || !esSHA256Hexadecimal(huellaRecurso) ||
		contieneComodinContextoAlmacen(resumen.DecisionRef, correlacion) {
		return cero, errorAutorizacionAlmacen()
	}
	return decisionAlmacenV3Ligada{
		decisionRef: resumen.DecisionRef, huellaDecision: huella, correlacionRef: correlacion,
		finalidad: datos.Finalidad, accion: datos.Accion, huellaRecurso: huellaRecurso,
		recurso: datos.Recurso, camposPermitidos: restricciones.CamposPermitidos,
		validaHasta: validaHasta.UTC(),
	}, nil
}

func (l decisionAlmacenV3Ligada) huellaPlan(v VinculosOperacionAlmacen, e especificacionAutorizacionAlmacen) string {
	return huellaPlanOperacionAlmacenCampos(marcaPlanDecisionV3, l.decisionRef, l.huellaDecision, l.accion,
		l.recurso.Referencia, l.huellaRecurso, l.finalidad, l.correlacionRef, v, e)
}

// validarEstructuraV3 repite todos los cotejos sobre los datos guardados:
// una copia alterada del contexto nunca se acepta.
func (d *datosContextoOperacionAlmacen) validarEstructuraV3() error {
	especificacion, existe := especificacionAlmacenV3(d.accionNegocio)
	if !existe || d.huellaManifiestoSHA256 != "" || !pasosIgualesAlmacen(d.pasos, especificacion.pasos) ||
		especificacion.requiereObjeto != (d.objetoVinculado != (ReferenciaObjetoAlmacen{})) {
		return ErrAutorizacionAlmacenInvalida
	}
	vinculos := VinculosOperacionAlmacen{
		OperacionRef: d.operacionRef, CargaRef: d.cargaRef, Clasificacion: d.clasificacion,
		SujetoSeudonimoHMAC: d.sujetoSeudonimoHMAC, HuellaSolicitudHMAC: d.huellaSolicitudHMAC,
		EfectoRef: d.efectoRef, ObjetoVinculado: d.objetoVinculado,
	}
	ligada, err := d.decisionV3.validarPara(especificacion, vinculos)
	if err != nil || !especificacion.valida() || !vinculos.validosPara(especificacion) ||
		ligada.decisionRef != d.autorizacionRef || ligada.huellaDecision != d.huellaDecisionSHA256 ||
		ligada.correlacionRef != d.correlacionRef || ligada.finalidad != d.finalidad ||
		ligada.recurso.Referencia != d.recursoRef || ligada.recurso.ModuloID != d.moduloID ||
		ligada.recurso.Tipo != d.tipoRecurso || ligada.huellaRecurso != d.huellaRecursoSHA256 ||
		!ligada.validaHasta.Equal(d.validaHasta) ||
		!d.decisionV3.confirmacion.DentroDeVentanaEn(d.verificadaEn) ||
		ligada.huellaPlan(vinculos, especificacion) != d.huellaPlanEfectoSHA256 {
		return ErrAutorizacionAlmacenInvalida
	}
	return nil
}

// especificacionAlmacenV3 es la lista cerrada de planes que admiten una
// decisión V3; los campos exactos salen del plan, nunca de la decisión.
func especificacionAlmacenV3(accionNegocio string) (especificacionAutorizacionAlmacen, bool) {
	switch accionNegocio {
	case AccionNegocioCustodiarDocumentoFirmadoExpediente:
		return especificacionCustodiarDocumentoFirmadoExpediente(), true
	case AccionNegocioEscribirOriginalFirmable:
		return especificacionEscribirOriginalFirmable(), true
	case AccionNegocioLeerOriginalDocumentoGenerado:
		return especificacionLeerOriginalDocumentoGenerado(), true
	default:
		return especificacionAutorizacionAlmacen{}, false
	}
}

func pasosIgualesAlmacen(a, b []pasoPlanOperacionAlmacen) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].referencia != b[i].referencia || a[i].accion != b[i].accion ||
			a[i].huellaPasoSHA256 != b[i].huellaPasoSHA256 || a[i].pasoDocumental != nil || b[i].pasoDocumental != nil {
			return false
		}
	}
	return true
}
