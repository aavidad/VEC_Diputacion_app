package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"
)

const (
	// EsquemaHuellaDecisionAutorizacionV3 congela el documento de decision
	// ligado a solicitud V3 y actor registrado V2.
	EsquemaHuellaDecisionAutorizacionV3           = "vec.autorizacion.decision.v3.solicitud-ligada.actor-v2"
	esquemaSelloEvidenciaEvaluacionAutorizacionV3 = "vec.autorizacion.evidencia-evaluacion.v3.solicitud-ligada"
	formatoInstanteDecisionAutorizacionV3         = "2006-01-02T15:04:05.000000Z"
)

type politicaDecisionAutorizacionCanonicaV3 struct {
	Referencia   string `json:"referencia"`
	HuellaSHA256 string `json:"huella_sha256"`
}

type evidenciaEvaluacionAutorizacionCanonicaV3 struct {
	Esquema                               string                                   `json:"esquema"`
	EsquemaHuellaSolicitud                string                                   `json:"esquema_huella_solicitud"`
	SolicitudHuellaSHA256                 string                                   `json:"solicitud_huella_sha256"`
	DecisionRef                           string                                   `json:"decision_ref"`
	Concedida                             bool                                     `json:"concedida"`
	Codigo                                string                                   `json:"codigo"`
	ContextoRecursoHuellaSHA256           string                                   `json:"contexto_recurso_huella_sha256"`
	AsignacionRef                         string                                   `json:"asignacion_ref"`
	AsignacionHuellaSHA256                string                                   `json:"asignacion_huella_sha256"`
	VersionRolRef                         string                                   `json:"version_rol_ref"`
	VersionRolHuellaSHA256                string                                   `json:"version_rol_huella_sha256"`
	ControlVigenciaVersionRolRef          string                                   `json:"control_vigencia_version_rol_ref"`
	ControlVigenciaVersionRolRevision     uint64                                   `json:"control_vigencia_version_rol_revision"`
	ControlVigenciaVersionRolHuellaSHA256 string                                   `json:"control_vigencia_version_rol_huella_sha256"`
	RevisionCatalogoPoliticas             uint64                                   `json:"revision_catalogo_politicas"`
	CatalogoPoliticasHuellaSHA256         string                                   `json:"catalogo_politicas_huella_sha256"`
	PoliticasEvaluadas                    []politicaDecisionAutorizacionCanonicaV3 `json:"politicas_evaluadas"`
	PoliticasAplicables                   []politicaDecisionAutorizacionCanonicaV3 `json:"politicas_aplicables"`
	GarantiaMinima                        AuthAssurance                            `json:"garantia_minima"`
	CamposPermitidos                      []string                                 `json:"campos_permitidos"`
	Obligaciones                          []string                                 `json:"obligaciones"`
	EmitidaEn                             string                                   `json:"emitida_en"`
	ValidaHasta                           string                                   `json:"valida_hasta"`
}

// decisionAutorizacionCanonicaV3 enumera el contrato cerrado; nunca serializa
// por reflexion el tipo vivo ni una DecisionAutorizacion historica.
type decisionAutorizacionCanonicaV3 struct {
	Esquema                               string                                   `json:"esquema"`
	BloqueVersion                         uint16                                   `json:"bloque_version"`
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
	VinculoAutenticacionActor             vinculoSolicitudAutorizacionCanonicoV3   `json:"vinculo_autenticacion_actor"`
	AsignacionRef                         string                                   `json:"asignacion_ref"`
	AsignacionHuellaSHA256                string                                   `json:"asignacion_huella_sha256"`
	VersionRolRef                         string                                   `json:"version_rol_ref"`
	VersionRolHuellaSHA256                string                                   `json:"version_rol_huella_sha256"`
	ControlVigenciaVersionRolRef          string                                   `json:"control_vigencia_version_rol_ref"`
	ControlVigenciaVersionRolRevision     uint64                                   `json:"control_vigencia_version_rol_revision"`
	ControlVigenciaVersionRolHuellaSHA256 string                                   `json:"control_vigencia_version_rol_huella_sha256"`
	RevisionCatalogoPoliticas             uint64                                   `json:"revision_catalogo_politicas"`
	CatalogoPoliticasHuellaSHA256         string                                   `json:"catalogo_politicas_huella_sha256"`
	PoliticasEvaluadas                    []politicaDecisionAutorizacionCanonicaV3 `json:"politicas_evaluadas"`
	PoliticasAplicables                   []politicaDecisionAutorizacionCanonicaV3 `json:"politicas_aplicables"`
	GarantiaMinima                        AuthAssurance                            `json:"garantia_minima"`
	CamposPermitidos                      []string                                 `json:"campos_permitidos"`
	Obligaciones                          []string                                 `json:"obligaciones"`
	EmitidaEn                             string                                   `json:"emitida_en"`
	ValidaHasta                           string                                   `json:"valida_hasta"`
}

// RepresentacionCanonicaDecisionAutorizacionV3 es la unica salida de bytes
// autorizada. No existe la operacion inversa ni un parser de capacidades.
func RepresentacionCanonicaDecisionAutorizacionV3(
	decision DecisionAutorizacionLigadaV3,
) ([]byte, error) {
	if decision.Validar() != nil {
		return nil, ErrDecisionAutorizacionLigadaV3Invalida
	}
	contenido, err := representacionCanonicaDecisionAutorizacionV3DesdeDatos(decision.datos)
	if err != nil {
		return nil, ErrDecisionAutorizacionLigadaV3Invalida
	}
	suma := sha256.Sum256(contenido)
	if hex.EncodeToString(suma[:]) != decision.datos.selloSHA256 {
		return nil, ErrDecisionAutorizacionLigadaV3Invalida
	}
	return append([]byte(nil), contenido...), nil
}

func HuellaSHA256DecisionAutorizacionV3(decision DecisionAutorizacionLigadaV3) (string, error) {
	contenido, err := RepresentacionCanonicaDecisionAutorizacionV3(decision)
	if err != nil {
		return "", err
	}
	suma := sha256.Sum256(contenido)
	return hex.EncodeToString(suma[:]), nil
}

func huellaCanonicaDecisionAutorizacionV3(d *datosDecisionAutorizacionLigadaV3) (string, error) {
	contenido, err := representacionCanonicaDecisionAutorizacionV3DesdeDatos(d)
	if err != nil {
		return "", err
	}
	suma := sha256.Sum256(contenido)
	return hex.EncodeToString(suma[:]), nil
}

func representacionCanonicaDecisionAutorizacionV3DesdeDatos(
	d *datosDecisionAutorizacionLigadaV3,
) ([]byte, error) {
	if d == nil {
		return nil, ErrDecisionAutorizacionLigadaV3Invalida
	}
	vinculo, err := d.vinculoAutenticacionActor.Datos()
	if err != nil {
		return nil, ErrDecisionAutorizacionLigadaV3Invalida
	}
	documento := decisionAutorizacionCanonicaV3{
		Esquema: EsquemaHuellaDecisionAutorizacionV3, BloqueVersion: d.bloqueVersion,
		DecisionRef: d.decisionRef, Concedida: d.concedida, Codigo: d.codigo,
		PrincipalID: d.principalID, PerfilActivoRef: d.perfilActivoRef,
		Accion: d.accion, RecursoRef: d.recursoRef, ModuloID: d.moduloID, TipoRecurso: d.tipoRecurso,
		ContextoRecursoHuellaSHA256: d.contextoRecursoHuellaSHA256,
		Finalidad:                   d.finalidad, CorrelacionRef: d.correlacionRef,
		EsquemaHuellaSolicitud: d.esquemaHuellaSolicitud, SolicitudHuellaSHA256: d.solicitudHuellaSHA256,
		EsquemaHuellaMotivo: d.esquemaHuellaMotivo, MotivoHuellaSHA256: d.motivoHuellaSHA256,
		VinculoAutenticacionActor: vinculoSolicitudAutorizacionCanonicoV3Desde(vinculo),
		AsignacionRef:             d.asignacionRef, AsignacionHuellaSHA256: d.asignacionHuellaSHA256,
		VersionRolRef: d.versionRolRef, VersionRolHuellaSHA256: d.versionRolHuellaSHA256,
		ControlVigenciaVersionRolRef:          d.controlVigenciaVersionRolRef,
		ControlVigenciaVersionRolRevision:     d.controlVigenciaVersionRolRevision,
		ControlVigenciaVersionRolHuellaSHA256: d.controlVigenciaVersionRolHuellaSHA256,
		RevisionCatalogoPoliticas:             d.revisionCatalogoPoliticas,
		CatalogoPoliticasHuellaSHA256:         d.catalogoPoliticasHuellaSHA256,
		PoliticasEvaluadas:                    politicasDecisionAutorizacionCanonicasV3(d.politicasEvaluadas),
		PoliticasAplicables:                   politicasDecisionAutorizacionCanonicasV3(d.politicasAplicables),
		GarantiaMinima:                        d.garantiaMinima,
		CamposPermitidos:                      append([]string{}, d.camposPermitidos...),
		Obligaciones:                          append([]string{}, d.obligaciones...),
		EmitidaEn:                             instanteDecisionAutorizacionCanonicoV3(d.emitidaEn),
		ValidaHasta:                           instanteDecisionAutorizacionCanonicoV3(d.validaHasta),
	}
	contenido, err := json.Marshal(documento)
	if err != nil {
		return nil, ErrDecisionAutorizacionLigadaV3Invalida
	}
	return contenido, nil
}

func huellaCanonicaEvidenciaEvaluacionAutorizacionV3(
	d *datosEvidenciaEvaluacionAutorizacionV3,
) (string, error) {
	if d == nil {
		return "", ErrEvidenciaEvaluacionAutorizacionV3Invalida
	}
	documento := evidenciaEvaluacionAutorizacionCanonicaV3{
		Esquema:                esquemaSelloEvidenciaEvaluacionAutorizacionV3,
		EsquemaHuellaSolicitud: d.esquemaHuellaSolicitud, SolicitudHuellaSHA256: d.solicitudHuellaSHA256,
		DecisionRef: d.decisionRef, Concedida: d.concedida, Codigo: d.codigo,
		ContextoRecursoHuellaSHA256: d.contextoRecursoHuellaSHA256,
		AsignacionRef:               d.asignacionRef, AsignacionHuellaSHA256: d.asignacionHuellaSHA256,
		VersionRolRef: d.versionRolRef, VersionRolHuellaSHA256: d.versionRolHuellaSHA256,
		ControlVigenciaVersionRolRef:          d.controlVigenciaVersionRolRef,
		ControlVigenciaVersionRolRevision:     d.controlVigenciaVersionRolRevision,
		ControlVigenciaVersionRolHuellaSHA256: d.controlVigenciaVersionRolHuellaSHA256,
		RevisionCatalogoPoliticas:             d.revisionCatalogoPoliticas,
		CatalogoPoliticasHuellaSHA256:         d.catalogoPoliticasHuellaSHA256,
		PoliticasEvaluadas:                    politicasDecisionAutorizacionCanonicasV3(d.politicasEvaluadas),
		PoliticasAplicables:                   politicasDecisionAutorizacionCanonicasV3(d.politicasAplicables),
		GarantiaMinima:                        d.garantiaMinima,
		CamposPermitidos:                      append([]string{}, d.camposPermitidos...),
		Obligaciones:                          append([]string{}, d.obligaciones...),
		EmitidaEn:                             instanteDecisionAutorizacionCanonicoV3(d.emitidaEn),
		ValidaHasta:                           instanteDecisionAutorizacionCanonicoV3(d.validaHasta),
	}
	contenido, err := json.Marshal(documento)
	if err != nil {
		return "", ErrEvidenciaEvaluacionAutorizacionV3Invalida
	}
	suma := sha256.Sum256(contenido)
	return hex.EncodeToString(suma[:]), nil
}

func politicasDecisionAutorizacionCanonicasV3(
	politicas []evidenciaPoliticaAutorizacionV3,
) []politicaDecisionAutorizacionCanonicaV3 {
	resultado := make([]politicaDecisionAutorizacionCanonicaV3, 0, len(politicas))
	for _, politica := range politicas {
		resultado = append(resultado, politicaDecisionAutorizacionCanonicaV3{
			Referencia:   politica.referencia,
			HuellaSHA256: politica.huellaSHA256,
		})
	}
	return resultado
}

func instanteDecisionAutorizacionCanonicoV3(instante time.Time) string {
	return instante.UTC().Format(formatoInstanteDecisionAutorizacionV3)
}

func huellaSHA256AutorizacionV3NoNula(valor string) bool {
	return huellaSHA256AutorizacionValida(valor) && valor != strings.Repeat("0", sha256.Size*2)
}
