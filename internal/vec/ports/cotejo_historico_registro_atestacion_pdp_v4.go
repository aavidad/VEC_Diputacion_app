package ports

import (
	"bytes"

	"vec-diputacion-granada/internal/vec/domain"
)

// CotejarConDecisionHistoricaAtestacionPDPV1 comprueba que la solicitud opaca
// viva, la preimagen durable y la proyeccion nominal extraida del payload
// VEC-AD-1 describen exactamente la misma aplicacion.
//
// Este cotejo no verifica COSE, no consulta una raiz de confianza y no concede
// autoridad. Solo debe invocarlo el caso de uso interno despues de verificar la
// firma y antes de entregar la solicitud al registrador tecnico aislado.
// Obligaciones permanece cerrado: exige lista y mapa de cumplimientos vacios,
// con sus huellas canonicas. No se habilitaran obligaciones no vacias hasta que
// el registro persista el mapa, evidencia tipada de cumplimiento y revocacion.
func (s SolicitudAplicacionAutorizacionEjecucionDocumentalV4) CotejarConDecisionHistoricaAtestacionPDPV1(
	datos domain.DatosDecisionHistoricaAtestacionAutorizacionV1,
	preimagen PreimagenRecursoAutorizacionEjecucionDocumentalV4,
) error {
	if s.validarEstructura() != nil || preimagen.Validar() != nil || s.datos == nil ||
		s.datos.vinculo.datos == nil {
		return errorAutorizacionEjecucionDocumentalV4()
	}

	proyeccion, err := s.ProyeccionParaTransaccion()
	preimagenViva, errPreimagenViva := s.PreimagenRecursoParaEvidenciaDurable()
	serializacion, errSerializacion := preimagen.SerializacionCanonicaParaPersistencia()
	serializacionViva, errSerializacionViva := preimagenViva.SerializacionCanonicaParaPersistencia()
	recurso, errRecurso := preimagen.RecursoCanonico()
	huellaContexto, errHuellaContexto := preimagen.HuellaContextoRecursoSHA256()
	huellaAmbitos, errHuellaAmbitos := preimagen.HuellaAmbitosSHA256()
	d := s.datos.vinculo.datos
	datosEvidencia, errEvidencia := d.evidencia.Datos()
	vinculo := datos.VinculoAutenticacionActor

	if err != nil || errPreimagenViva != nil || errSerializacion != nil ||
		errSerializacionViva != nil || errRecurso != nil || errHuellaContexto != nil ||
		errHuellaAmbitos != nil || errEvidencia != nil || vinculo.Validar() != nil ||
		!decisionHistoricaAtestacionPDPV1CoincideExactamente(
			datos, datosEvidencia.Decision,
		) ||
		!bytes.Equal(serializacion, serializacionViva) ||
		!datos.Concedida || datos.Codigo != "concedida" ||
		datos.DecisionRef != proyeccion.Clave.DecisionRef || datos.DecisionRef != d.decisionRef ||
		datos.PrincipalID != d.principalID || datos.PrincipalID != vinculo.PrincipalID ||
		datos.PerfilActivoRef != proyeccion.PerfilActivoRef ||
		datos.PerfilActivoRef != d.perfilActivoRef ||
		datos.PerfilActivoRef != vinculo.PerfilActivoRef ||
		vinculo.AutenticacionRef != d.autenticacionRef || vinculo.SesionRef != d.sesionRef ||
		vinculo.ControlSesionRef != d.controlSesionRef ||
		vinculo.ControlSesionRevision != d.controlSesionRevision ||
		vinculo.ControlSesionHuellaSHA256 != d.controlSesionHuellaSHA256 ||
		vinculo.ContextoActorRef != d.contextoActorRef ||
		vinculo.ContextoActorVersion != d.contextoActorVersion ||
		vinculo.ContextoActorHuellaSHA256 != proyeccion.ContextoActorHuellaSHA256 ||
		vinculo.ContextoActorHuellaSHA256 != d.contextoActorHuellaSHA256 ||
		datos.Accion != proyeccion.Accion || datos.Accion != d.accion ||
		datos.RecursoRef != proyeccion.RecursoRef || datos.RecursoRef != recurso.Referencia ||
		datos.ModuloID != proyeccion.ModuloID || datos.ModuloID != recurso.ModuloID ||
		datos.TipoRecurso != proyeccion.TipoRecurso || datos.TipoRecurso != recurso.Tipo ||
		datos.ContextoRecursoHuellaSHA256 != proyeccion.HuellaRecursoSHA256 ||
		datos.ContextoRecursoHuellaSHA256 != huellaContexto ||
		huellaAmbitos != proyeccion.HuellaAmbitosSHA256 ||
		datos.Finalidad != proyeccion.Finalidad || datos.Finalidad != d.finalidad ||
		datos.CorrelacionRef != proyeccion.CorrelacionRef ||
		datos.CorrelacionRef != d.correlacionRef ||
		!listaExactaAutorizacionEjecucionDocumentalV4(datos.CamposPermitidos, d.camposPermitidos) ||
		!listaExactaAutorizacionEjecucionDocumentalV4(datos.Obligaciones, d.obligaciones) ||
		len(datos.Obligaciones) != 0 || len(d.obligaciones) != 0 ||
		len(d.cumplimientosObligacionesPorRef) != 0 ||
		huellaListaAutorizacionEjecucionDocumentalV4(
			"vec.documentos.autorizacion-ejecucion.campos.v4", datos.CamposPermitidos,
		) != proyeccion.HuellaCamposPermitidosSHA256 ||
		huellaListaAutorizacionEjecucionDocumentalV4(
			"vec.documentos.autorizacion-ejecucion.obligaciones.v4", datos.Obligaciones,
		) != proyeccion.HuellaObligacionesSHA256 ||
		proyeccion.HuellaObligacionesSHA256 !=
			HuellaObligacionesVaciasAutorizacionEjecucionDocumentalV4() ||
		proyeccion.HuellaCumplimientosSHA256 !=
			HuellaCumplimientosVaciosAutorizacionEjecucionDocumentalV4() ||
		!datos.EmitidaEn.Equal(datosEvidencia.Decision.EmitidaEn) ||
		datos.EmitidaEn.After(proyeccion.VerificadaEn) ||
		!proyeccion.VerificadaEn.Before(datos.ValidaHasta) ||
		!datos.ValidaHasta.Equal(proyeccion.ValidaHasta) ||
		!datos.ValidaHasta.Equal(datosEvidencia.Decision.ValidaHasta) ||
		recurso.Atributos[AtributoAutorizacionDocumentalHuellaPlanSHA256] !=
			proyeccion.Clave.HuellaPlanSHA256 ||
		recurso.Atributos[AtributoAutorizacionDocumentalEfectoRef] !=
			proyeccion.Clave.EfectoRef {
		return errorAutorizacionEjecucionDocumentalV4()
	}
	return nil
}

// decisionHistoricaAtestacionPDPV1CoincideExactamente compara los treinta
// campos superiores de DecisionAutorizacion y los veinticinco del vinculo. No
// usa una huella parcial ni semantica de conjuntos: listas conservan orden,
// mapas exigen las mismas parejas y los instantes deben ser identicos.
func decisionHistoricaAtestacionPDPV1CoincideExactamente(
	recibida domain.DatosDecisionHistoricaAtestacionAutorizacionV1,
	esperada domain.DecisionAutorizacion,
) bool {
	vinculoEsperado, err := esperada.VinculoAutenticacionActor.Datos()
	return err == nil && recibida.DecisionRef == esperada.DecisionRef &&
		recibida.Concedida == esperada.Concedida && recibida.Codigo == esperada.Codigo &&
		recibida.PrincipalID == esperada.PrincipalID &&
		recibida.PerfilActivoRef == esperada.PerfilActivoRef &&
		recibida.Accion == esperada.Accion && recibida.RecursoRef == esperada.RecursoRef &&
		recibida.ModuloID == esperada.ModuloID && recibida.TipoRecurso == esperada.TipoRecurso &&
		recibida.ContextoRecursoHuellaSHA256 == esperada.ContextoRecursoHuellaSHA256 &&
		recibida.Finalidad == esperada.Finalidad && recibida.CorrelacionRef == esperada.CorrelacionRef &&
		datosVinculoHistoricoAtestacionPDPV1CoincidenExactamente(
			recibida.VinculoAutenticacionActor, vinculoEsperado,
		) &&
		recibida.AsignacionRef == esperada.AsignacionRef &&
		recibida.AsignacionHuellaSHA256 == esperada.AsignacionHuellaSHA256 &&
		recibida.VersionRolRef == esperada.VersionRolRef &&
		recibida.VersionRolHuellaSHA256 == esperada.VersionRolHuellaSHA256 &&
		recibida.ControlVigenciaVersionRolRef == esperada.ControlVigenciaVersionRolRef &&
		recibida.ControlVigenciaVersionRolRevision == esperada.ControlVigenciaVersionRolRevision &&
		recibida.ControlVigenciaVersionRolHuellaSHA256 == esperada.ControlVigenciaVersionRolHuellaSHA256 &&
		recibida.RevisionCatalogoPoliticas == esperada.RevisionCatalogoPoliticas &&
		recibida.CatalogoPoliticasHuellaSHA256 == esperada.CatalogoPoliticasHuellaSHA256 &&
		listasHistoricasAtestacionPDPV1Iguales(
			recibida.PoliticasEvaluadasRefs, esperada.PoliticasEvaluadasRefs,
		) &&
		mapasHistoricosAtestacionPDPV1Iguales(
			recibida.PoliticasEvaluadasHuellasSHA256,
			esperada.PoliticasEvaluadasHuellasSHA256,
		) &&
		listasHistoricasAtestacionPDPV1Iguales(recibida.PoliticasRefs, esperada.PoliticasRefs) &&
		mapasHistoricosAtestacionPDPV1Iguales(
			recibida.PoliticasHuellasSHA256, esperada.PoliticasHuellasSHA256,
		) &&
		recibida.GarantiaMinima == esperada.GarantiaMinima &&
		listasHistoricasAtestacionPDPV1Iguales(recibida.CamposPermitidos, esperada.CamposPermitidos) &&
		listasHistoricasAtestacionPDPV1Iguales(recibida.Obligaciones, esperada.Obligaciones) &&
		recibida.EmitidaEn.Equal(esperada.EmitidaEn) &&
		recibida.ValidaHasta.Equal(esperada.ValidaHasta)
}

func datosVinculoHistoricoAtestacionPDPV1CoincidenExactamente(
	recibidos, esperados domain.DatosVinculoAutenticacionActorV1,
) bool {
	return recibidos.BloqueVersion == esperados.BloqueVersion &&
		recibidos.AutenticacionRef == esperados.AutenticacionRef &&
		recibidos.AutenticacionHuellaSHA256 == esperados.AutenticacionHuellaSHA256 &&
		recibidos.AsercionRef == esperados.AsercionRef &&
		recibidos.SesionRef == esperados.SesionRef &&
		recibidos.ControlSesionRef == esperados.ControlSesionRef &&
		recibidos.ControlSesionRevision == esperados.ControlSesionRevision &&
		recibidos.ControlSesionHuellaSHA256 == esperados.ControlSesionHuellaSHA256 &&
		recibidos.CuentaRef == esperados.CuentaRef &&
		recibidos.CuentaOrdinariaRef == esperados.CuentaOrdinariaRef &&
		recibidos.PrincipalID == esperados.PrincipalID &&
		recibidos.PerfilActivoRef == esperados.PerfilActivoRef &&
		recibidos.CuentaPrivilegiada == esperados.CuentaPrivilegiada &&
		recibidos.Superficie == esperados.Superficie &&
		recibidos.MetodoObservado == esperados.MetodoObservado &&
		recibidos.GarantiaObservada == esperados.GarantiaObservada &&
		recibidos.PoliticaGarantiaRef == esperados.PoliticaGarantiaRef &&
		recibidos.PoliticaGarantiaHuellaSHA256 == esperados.PoliticaGarantiaHuellaSHA256 &&
		recibidos.AutenticacionVerificadaEn.Equal(esperados.AutenticacionVerificadaEn) &&
		recibidos.SesionEmitidaEn.Equal(esperados.SesionEmitidaEn) &&
		recibidos.SesionValidaHasta.Equal(esperados.SesionValidaHasta) &&
		recibidos.SesionRevalidadaEn.Equal(esperados.SesionRevalidadaEn) &&
		recibidos.ContextoActorRef == esperados.ContextoActorRef &&
		recibidos.ContextoActorVersion == esperados.ContextoActorVersion &&
		recibidos.ContextoActorHuellaSHA256 == esperados.ContextoActorHuellaSHA256
}

func listasHistoricasAtestacionPDPV1Iguales(recibida, esperada []string) bool {
	if len(recibida) != len(esperada) {
		return false
	}
	for indice := range recibida {
		if recibida[indice] != esperada[indice] {
			return false
		}
	}
	return true
}

func mapasHistoricosAtestacionPDPV1Iguales(recibido, esperado map[string]string) bool {
	if len(recibido) != len(esperado) {
		return false
	}
	for clave, valorEsperado := range esperado {
		if valorRecibido, existe := recibido[clave]; !existe || valorRecibido != valorEsperado {
			return false
		}
	}
	return true
}
