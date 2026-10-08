package domain

import (
	"sort"
	"time"
)

// ResultadoLecturaActualAutorizacionV3 describe una evaluación puntual para
// construir una captura de lectura. Sus revisiones identifican la instantánea
// evaluada; el consumidor debe obtenerla de FuenteAutorizacion en cada lectura.
// Este resultado no es una decisión ni una concesión ejecutable.
type ResultadoLecturaActualAutorizacionV3 struct {
	Permitido                 bool
	AsignacionRef             string
	AsignacionVersion         int
	VersionRolRef             string
	RevisionControlVersionRol uint64
	RevisionCatalogoPoliticas uint64
	EvaluadaEn                time.Time
	ValidaHasta               time.Time
	CamposPermitidos          []string
	Obligaciones              []string
}

// EvaluarLecturaActualAutorizacionV3 exige el vínculo y el resultado V2 de la
// misma sesión, una instantánea central coherente y todos los campos exactos
// de la lectura. El recurso, sus ámbitos y atributos deben ser resueltos por el
// servidor. La función es pura y no registra ni consume una decisión V3.
func EvaluarLecturaActualAutorizacionV3(
	vinculo VinculoAutenticacionActorV2,
	resultado ResultadoContextoActorRegistradoV2,
	instantanea InstantaneaAutorizacion,
	accion string,
	recurso RecursoAutorizable,
	finalidad string,
	camposSolicitados []string,
	instante time.Time,
) (ResultadoLecturaActualAutorizacionV3, error) {
	denegado := ResultadoLecturaActualAutorizacionV3{}
	if resultado.Validar() != nil || vinculo.ValidarPara(resultado) != nil ||
		instantanea.Validar() != nil || !instanteAutorizacionCanonico(instante) ||
		!textoAutorizacionSinComodinSeguro(accion, 256, false) ||
		!textoAutorizacionSinComodinSeguro(finalidad, 512, false) ||
		recurso.Validar() != nil || !listaAutorizacionValida(camposSolicitados, false, true) {
		return denegado, ErrAutorizacionDenegada
	}
	datos, err := vinculo.Datos()
	if err != nil || !vinculo.VigenteEn(instante, resultado) ||
		instantanea.AsignacionPerfil.PrincipalID != datos.PrincipalID ||
		instantanea.AsignacionPerfil.PerfilActivoRef != datos.PerfilActivoRef ||
		instantanea.ControlVigenciaVersionRol.ActualizadoEn.After(instante) {
		return denegado, ErrAutorizacionDenegada
	}

	politicas := append([]PoliticaRestrictiva(nil), instantanea.Politicas...)
	sort.Slice(politicas, func(i, j int) bool {
		return politicas[i].Referencia() < politicas[j].Referencia()
	})
	concedida, _, _, camposPermitidos, obligaciones, err := evaluarResultadoAutorizacionV3(
		SolicitudAutorizacion{Accion: accion, Recurso: recurso, Finalidad: finalidad},
		datos.GarantiaObservada, instantanea, politicas, instante,
	)
	if err != nil || !concedida {
		return denegado, ErrAutorizacionDenegada
	}
	for _, campo := range camposSolicitados {
		if !contieneAutorizacionExacta(camposPermitidos, campo) {
			return denegado, ErrAutorizacionDenegada
		}
	}
	limite := limitarVentanaEvaluacionAutorizacionV3(
		instante, instante.Add(VigenciaMaximaDecisionAutorizacion), datos, instantanea,
	)
	limite = limiteContextoCapacidadInformativaV3(resultado.Contexto.Instantanea, limite)
	if !limite.After(instante) {
		return denegado, ErrAutorizacionDenegada
	}
	campos := append([]string(nil), camposSolicitados...)
	sort.Strings(campos)
	return ResultadoLecturaActualAutorizacionV3{
		Permitido:                 true,
		AsignacionRef:             instantanea.AsignacionPerfil.Referencia(),
		AsignacionVersion:         instantanea.AsignacionPerfil.Version,
		VersionRolRef:             instantanea.VersionRol.Referencia(),
		RevisionControlVersionRol: instantanea.ControlVigenciaVersionRol.Revision,
		RevisionCatalogoPoliticas: instantanea.RevisionCatalogoPoliticas,
		EvaluadaEn:                instante,
		ValidaHasta:               limite,
		CamposPermitidos:          campos,
		Obligaciones:              append([]string(nil), obligaciones...),
	}, nil
}
