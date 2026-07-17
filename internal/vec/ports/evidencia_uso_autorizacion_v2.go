package ports

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"

	"vec-diputacion-granada/internal/vec/domain"
)

// decisionAutorizacionCanonicaV2 es un DTO congelado independiente de V1.
// Los cuatro campos nuevos no se incorporan nunca al esquema historico por
// reflexion ni mediante embedding: cualquier cambio posterior exige V3.
type decisionAutorizacionCanonicaV2 struct {
	Esquema                               string                                   `json:"esquema"`
	DecisionRef                           string                                   `json:"decision_ref"`
	Concedida                             bool                                     `json:"concedida"`
	Codigo                                string                                   `json:"codigo"`
	PrincipalID                           string                                   `json:"principal_id"`
	PerfilActivoRef                       string                                   `json:"perfil_activo_ref"`
	Accion                                string                                   `json:"accion"`
	RecursoRef                            string                                   `json:"recurso_ref"`
	ModuloID                              string                                   `json:"modulo_id"`
	TipoRecurso                           string                                   `json:"tipo_recurso"`
	ContextoRecursoHuellaSHA256           string                                   `json:"contexto_recurso_huella_sha256"`
	Finalidad                             string                                   `json:"finalidad"`
	CorrelacionRef                        string                                   `json:"correlacion_ref"`
	EsquemaHuellaSolicitud                string                                   `json:"esquema_huella_solicitud"`
	SolicitudHuellaSHA256                 string                                   `json:"solicitud_huella_sha256"`
	EsquemaHuellaMotivo                   string                                   `json:"esquema_huella_motivo"`
	MotivoHuellaSHA256                    string                                   `json:"motivo_huella_sha256"`
	VinculoAutenticacionActor             vinculoAutenticacionActorCanonicoV1      `json:"vinculo_autenticacion_actor"`
	AsignacionRef                         string                                   `json:"asignacion_ref"`
	AsignacionHuellaSHA256                string                                   `json:"asignacion_huella_sha256"`
	VersionRolRef                         string                                   `json:"version_rol_ref"`
	VersionRolHuellaSHA256                string                                   `json:"version_rol_huella_sha256"`
	ControlVigenciaVersionRolRef          string                                   `json:"control_vigencia_version_rol_ref"`
	ControlVigenciaVersionRolRevision     uint64                                   `json:"control_vigencia_version_rol_revision"`
	ControlVigenciaVersionRolHuellaSHA256 string                                   `json:"control_vigencia_version_rol_huella_sha256"`
	RevisionCatalogoPoliticas             uint64                                   `json:"revision_catalogo_politicas"`
	CatalogoPoliticasHuellaSHA256         string                                   `json:"catalogo_politicas_huella_sha256"`
	PoliticasEvaluadas                    []politicaDecisionAutorizacionCanonicaV1 `json:"politicas_evaluadas"`
	PoliticasAplicables                   []politicaDecisionAutorizacionCanonicaV1 `json:"politicas_aplicables"`
	GarantiaMinima                        domain.AuthAssurance                     `json:"garantia_minima"`
	CamposPermitidos                      []string                                 `json:"campos_permitidos"`
	Obligaciones                          []string                                 `json:"obligaciones"`
	EmitidaEn                             string                                   `json:"emitida_en"`
	ValidaHasta                           string                                   `json:"valida_hasta"`
}

// RepresentacionCanonicaDecisionAutorizacionReforzadaV2 devuelve la proyeccion
// apta para persistencia y cotejo. No contiene Motivo ni la referencia de
// catalogo en claro, sino sus compromisos de integridad. El consumidor durable
// debe releer la misma version publicada y cotejar su huella: este documento no
// acredita por si solo la procedencia del catalogo.
func RepresentacionCanonicaDecisionAutorizacionReforzadaV2(
	decision domain.DecisionAutorizacion,
) ([]byte, error) {
	return serializarDecisionAutorizacionReforzadaV2(decision)
}

func huellaDecisionAutorizacionReforzadaV2(decision domain.DecisionAutorizacion) (string, error) {
	contenido, err := serializarDecisionAutorizacionReforzadaV2(decision)
	if err != nil {
		return "", err
	}
	suma := sha256.Sum256(contenido)
	return hex.EncodeToString(suma[:]), nil
}

func serializarDecisionAutorizacionReforzadaV2(decision domain.DecisionAutorizacion) ([]byte, error) {
	if decision.ValidarEvidenciaInstantaneaSolicitudLigadaV2() != nil ||
		contieneComodinDecisionAutorizacion(decision) {
		return nil, ErrEvidenciaUsoDecisionAutorizacionInvalida
	}
	politicasEvaluadas, err := politicasDecisionAutorizacionCanonicas(
		decision.PoliticasEvaluadasRefs,
		decision.PoliticasEvaluadasHuellasSHA256,
	)
	if err != nil {
		return nil, err
	}
	politicasAplicables, err := politicasDecisionAutorizacionCanonicas(
		decision.PoliticasRefs,
		decision.PoliticasHuellasSHA256,
	)
	if err != nil {
		return nil, err
	}
	campos := append([]string{}, decision.CamposPermitidos...)
	obligaciones := append([]string{}, decision.Obligaciones...)
	vinculo, err := vinculoAutenticacionActorDecisionCanonicoV1(decision.VinculoAutenticacionActor)
	if err != nil {
		return nil, err
	}
	sort.Strings(campos)
	sort.Strings(obligaciones)
	canonica := decisionAutorizacionCanonicaV2{
		Esquema:     EsquemaHuellaDecisionAutorizacionReforzadaV2,
		DecisionRef: decision.DecisionRef, Concedida: decision.Concedida, Codigo: decision.Codigo,
		PrincipalID: decision.PrincipalID, PerfilActivoRef: decision.PerfilActivoRef,
		Accion: decision.Accion, RecursoRef: decision.RecursoRef, ModuloID: decision.ModuloID,
		TipoRecurso: decision.TipoRecurso, ContextoRecursoHuellaSHA256: decision.ContextoRecursoHuellaSHA256,
		Finalidad: decision.Finalidad, CorrelacionRef: decision.CorrelacionRef,
		EsquemaHuellaSolicitud: decision.EsquemaHuellaSolicitud,
		SolicitudHuellaSHA256:  decision.SolicitudHuellaSHA256,
		EsquemaHuellaMotivo:    decision.EsquemaHuellaMotivo, MotivoHuellaSHA256: decision.MotivoHuellaSHA256,
		VinculoAutenticacionActor: vinculo,
		AsignacionRef:             decision.AsignacionRef, AsignacionHuellaSHA256: decision.AsignacionHuellaSHA256,
		VersionRolRef: decision.VersionRolRef, VersionRolHuellaSHA256: decision.VersionRolHuellaSHA256,
		ControlVigenciaVersionRolRef:          decision.ControlVigenciaVersionRolRef,
		ControlVigenciaVersionRolRevision:     decision.ControlVigenciaVersionRolRevision,
		ControlVigenciaVersionRolHuellaSHA256: decision.ControlVigenciaVersionRolHuellaSHA256,
		RevisionCatalogoPoliticas:             decision.RevisionCatalogoPoliticas,
		CatalogoPoliticasHuellaSHA256:         decision.CatalogoPoliticasHuellaSHA256,
		PoliticasEvaluadas:                    politicasEvaluadas, PoliticasAplicables: politicasAplicables,
		GarantiaMinima: decision.GarantiaMinima, CamposPermitidos: campos, Obligaciones: obligaciones,
		EmitidaEn:   decision.EmitidaEn.UTC().Format(formatoInstanteDecisionAutorizacionV1),
		ValidaHasta: decision.ValidaHasta.UTC().Format(formatoInstanteDecisionAutorizacionV1),
	}
	contenido, err := json.Marshal(canonica)
	if err != nil {
		return nil, ErrEvidenciaUsoDecisionAutorizacionInvalida
	}
	return contenido, nil
}
