package domain

import (
	"encoding/binary"
	"encoding/hex"
	"sort"
	"time"
)

const esquemaInstantaneaLecturaActualV3 = "vec.autorizacion.lectura-actual.v3.instantanea"

// ReferenciaHuellaPoliticaLecturaActualV3 conserva la identidad y el contenido
// de cada política del catálogo evaluado, incluidas las no aplicables.
type ReferenciaHuellaPoliticaLecturaActualV3 struct {
	Referencia   string `json:"referencia"`
	HuellaSHA256 string `json:"huella_sha256"`
}

type instantaneaLecturaActualCanonicaV3 struct {
	Esquema                       string                                    `json:"esquema"`
	AsignacionRef                 string                                    `json:"asignacion_ref"`
	AsignacionVersion             int                                       `json:"asignacion_version"`
	AsignacionHuellaSHA256        string                                    `json:"asignacion_huella_sha256"`
	VersionRolRef                 string                                    `json:"version_rol_ref"`
	VersionRolVersion             int                                       `json:"version_rol_version"`
	VersionRolHuellaSHA256        string                                    `json:"version_rol_huella_sha256"`
	ControlVersionRolRef          string                                    `json:"control_version_rol_ref"`
	RevisionControlVersionRol     uint64                                    `json:"revision_control_version_rol"`
	ControlVersionRolHuellaSHA256 string                                    `json:"control_version_rol_huella_sha256"`
	RevisionCatalogoPoliticas     uint64                                    `json:"revision_catalogo_politicas"`
	CatalogoPoliticasHuellaSHA256 string                                    `json:"catalogo_politicas_huella_sha256"`
	PoliticasEvaluadas            []ReferenciaHuellaPoliticaLecturaActualV3 `json:"politicas_evaluadas"`
}

// ResultadoLecturaActualAutorizacionV3 describe una evaluación puntual para
// construir una captura de lectura. Sus revisiones identifican la instantánea
// evaluada; el consumidor debe obtenerla de FuenteAutorizacion en cada lectura.
// Este resultado no es una decisión ni una concesión ejecutable.
type ResultadoLecturaActualAutorizacionV3 struct {
	Permitido                     bool
	AsignacionRef                 string
	AsignacionVersion             int
	AsignacionHuellaSHA256        string
	VersionRolRef                 string
	VersionRolVersion             int
	VersionRolHuellaSHA256        string
	ControlVersionRolRef          string
	RevisionControlVersionRol     uint64
	ControlVersionRolHuellaSHA256 string
	RevisionCatalogoPoliticas     uint64
	CatalogoPoliticasHuellaSHA256 string
	PoliticasEvaluadas            []ReferenciaHuellaPoliticaLecturaActualV3
	HuellaInstantaneaSHA256       string
	RevisionPermisos              uint64
	EvaluadaEn                    time.Time
	ValidaHasta                   time.Time
	CamposPermitidos              []string
	Obligaciones                  []string
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
	politicasEvaluadas := make([]ReferenciaHuellaPoliticaLecturaActualV3, 0, len(politicas))
	for _, politica := range politicas {
		huella, errHuella := politica.HuellaSHA256()
		if errHuella != nil || !huellaSHA256AutorizacionV3NoNula(huella) {
			return denegado, ErrAutorizacionDenegada
		}
		politicasEvaluadas = append(politicasEvaluadas, ReferenciaHuellaPoliticaLecturaActualV3{
			Referencia: politica.Referencia(), HuellaSHA256: huella,
		})
	}
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
	huellaAsignacion, err := instantanea.AsignacionPerfil.HuellaSHA256()
	if err != nil || !huellaSHA256AutorizacionV3NoNula(huellaAsignacion) {
		return denegado, ErrAutorizacionDenegada
	}
	huellaRol, err := instantanea.VersionRol.HuellaSHA256()
	if err != nil || !huellaSHA256AutorizacionV3NoNula(huellaRol) {
		return denegado, ErrAutorizacionDenegada
	}
	huellaControl, err := instantanea.ControlVigenciaVersionRol.HuellaSHA256()
	if err != nil || !huellaSHA256AutorizacionV3NoNula(huellaControl) {
		return denegado, ErrAutorizacionDenegada
	}
	manifiesto := instantaneaLecturaActualCanonicaV3{
		Esquema:                       esquemaInstantaneaLecturaActualV3,
		AsignacionRef:                 instantanea.AsignacionPerfil.Referencia(),
		AsignacionVersion:             instantanea.AsignacionPerfil.Version,
		AsignacionHuellaSHA256:        huellaAsignacion,
		VersionRolRef:                 instantanea.VersionRol.Referencia(),
		VersionRolVersion:             instantanea.VersionRol.Version,
		VersionRolHuellaSHA256:        huellaRol,
		ControlVersionRolRef:          instantanea.ControlVigenciaVersionRol.VersionRolRef,
		RevisionControlVersionRol:     instantanea.ControlVigenciaVersionRol.Revision,
		ControlVersionRolHuellaSHA256: huellaControl,
		RevisionCatalogoPoliticas:     instantanea.RevisionCatalogoPoliticas,
		CatalogoPoliticasHuellaSHA256: instantanea.CatalogoPoliticasHuellaSHA256,
		PoliticasEvaluadas:            politicasEvaluadas,
	}
	huellaInstantanea, err := huellaAutorizacion(manifiesto)
	if err != nil || !huellaSHA256AutorizacionV3NoNula(huellaInstantanea) {
		return denegado, ErrAutorizacionDenegada
	}
	bytesHuella, err := hex.DecodeString(huellaInstantanea)
	if err != nil {
		return denegado, ErrAutorizacionDenegada
	}
	revisionPermisos := binary.BigEndian.Uint64(bytesHuella[:8])
	if revisionPermisos == 0 {
		return denegado, ErrAutorizacionDenegada
	}
	campos := append([]string(nil), camposSolicitados...)
	sort.Strings(campos)
	return ResultadoLecturaActualAutorizacionV3{
		Permitido:                     true,
		AsignacionRef:                 instantanea.AsignacionPerfil.Referencia(),
		AsignacionVersion:             instantanea.AsignacionPerfil.Version,
		AsignacionHuellaSHA256:        huellaAsignacion,
		VersionRolRef:                 instantanea.VersionRol.Referencia(),
		VersionRolVersion:             instantanea.VersionRol.Version,
		VersionRolHuellaSHA256:        huellaRol,
		ControlVersionRolRef:          instantanea.ControlVigenciaVersionRol.VersionRolRef,
		RevisionControlVersionRol:     instantanea.ControlVigenciaVersionRol.Revision,
		ControlVersionRolHuellaSHA256: huellaControl,
		RevisionCatalogoPoliticas:     instantanea.RevisionCatalogoPoliticas,
		CatalogoPoliticasHuellaSHA256: instantanea.CatalogoPoliticasHuellaSHA256,
		PoliticasEvaluadas:            append([]ReferenciaHuellaPoliticaLecturaActualV3(nil), politicasEvaluadas...),
		HuellaInstantaneaSHA256:       huellaInstantanea,
		RevisionPermisos:              revisionPermisos,
		EvaluadaEn:                    instante,
		ValidaHasta:                   limite,
		CamposPermitidos:              campos,
		Obligaciones:                  append([]string(nil), obligaciones...),
	}, nil
}
