package firmaautorizacionv2

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vp "vec-diputacion-granada/internal/vec/ports"
)

type decisionFirmaPlanV2 struct {
	DecisionRef                 string `json:"decision_ref"`
	PrincipalID                 string `json:"principal_id"`
	PerfilActivoRef             string `json:"perfil_activo_ref"`
	VersionRolRef               string `json:"version_rol_ref"`
	Accion                      string `json:"accion"`
	RecursoRef                  string `json:"recurso_ref"`
	ModuloID                    string `json:"modulo_id"`
	TipoRecurso                 string `json:"tipo_recurso"`
	Finalidad                   string `json:"finalidad"`
	ContextoRecursoHuellaSHA256 string `json:"contexto_recurso_huella_sha256"`
}

func decisionTransportadaFirmaPlanV2(x vp.ExportacionMaterialConsumoAutorizacionAtestadaV3) (decisionFirmaPlanV2, error) {
	var d decisionFirmaPlanV2
	b := x.DecisionCanonica()
	h := sha256.Sum256(b)
	s := x.ResumenCapacidad()
	if err := json.Unmarshal(b, &d); err != nil {
		return decisionFirmaPlanV2{}, errors.Join(ports.ErrFirmaDocumentoDenegada, err)
	}
	if d.DecisionRef == "" ||
		d.DecisionRef != s.DecisionRef() || hex.EncodeToString(h[:]) != s.DecisionHuellaSHA256() ||
		d.ContextoRecursoHuellaSHA256 != s.EfectoHuellaSHA256() ||
		d.Accion != s.Operacion() || d.RecursoRef != s.EfectoRef() {
		return decisionFirmaPlanV2{}, ports.ErrFirmaDocumentoDenegada
	}
	return d, nil
}

// ValidarCapacidadFirmaConPlanV2 coteja las dos declaraciones transportadas.
// AD177 vuelve a validar las decisiones canónicas y su consumo dentro de SQL.
func ValidarCapacidadFirmaConPlanV2(c ports.CapacidadFirmaConPlanV2, m ports.MaterialFirmaVerificadaV2) error {
	interior, exterior, plan, envoltorio, decisionSHA := c.ExportarParaConsumidor()
	defer clear(envoltorio)
	ambitos, err := AmbitosCapacidadFirmaVerificadaV2(interior, m)
	if err != nil || exterior.ValidarEstructura() != nil || plan.Validar() != nil {
		return ports.ErrFirmaDocumentoDenegada
	}
	i := interior.ExportarMaterialParaConsumidor()
	e := exterior
	di, errI := decisionTransportadaFirmaPlanV2(i)
	de, errE := decisionTransportadaFirmaPlanV2(e)
	if errI != nil {
		return errI
	}
	if errE != nil {
		return errE
	}
	h := sha256.Sum256(i.DecisionCanonica())
	if decisionSHA != hex.EncodeToString(h[:]) ||
		di.DecisionRef == de.DecisionRef || di.PrincipalID == "" || di.PerfilActivoRef == "" ||
		di.VersionRolRef == "" || di.Accion == "" || di.RecursoRef == "" || di.ModuloID == "" ||
		di.TipoRecurso == "" || di.Finalidad == "" ||
		di.PrincipalID != de.PrincipalID || di.PerfilActivoRef != de.PerfilActivoRef ||
		di.VersionRolRef != de.VersionRolRef || di.Accion != de.Accion || di.RecursoRef != de.RecursoRef ||
		di.ModuloID != de.ModuloID || di.TipoRecurso != de.TipoRecurso || di.Finalidad != de.Finalidad ||
		!bytes.Equal(i.ContextoActorCanonico(), e.ContextoActorCanonico()) ||
		i.PersonaVersion() != e.PersonaVersion() || i.PerfilVersion() != e.PerfilVersion() {
		return ports.ErrFirmaDocumentoDenegada
	}
	if ValidarPlanAutorizadoFirmaV2(m, plan, decisionSHA, envoltorio) != nil {
		return ports.ErrFirmaDocumentoDenegada
	}
	r, err := RecursoPlanAutorizadoFirmaV2(m, plan, decisionSHA, envoltorio, ambitos)
	if err != nil {
		return ports.ErrFirmaDocumentoDenegada
	}
	recursoSHA, err := r.HuellaContextoAutorizacionSHA256()
	s := e.ResumenCapacidad()
	if err != nil || s.Operacion() != i.ResumenCapacidad().Operacion() ||
		s.AudienciaConsumo() != i.ResumenCapacidad().AudienciaConsumo() ||
		s.EfectoRef() != r.Referencia || s.EfectoHuellaSHA256() != recursoSHA ||
		de.ContextoRecursoHuellaSHA256 != recursoSHA ||
		di.PerfilActivoRef != m.PerfilActivoOperadorRef ||
		di.RecursoRef != m.RecursoRef() || di.ModuloID != ports.ModuloContratacion {
		return ports.ErrFirmaDocumentoDenegada
	}
	if m.Via == ports.ViaFirmaCertificadoVEC && di.PrincipalID != m.FirmantePrincipalRef ||
		m.Via == ports.ViaFirmaExternaPortafirmas && di.PrincipalID == m.FirmantePrincipalRef {
		return ports.ErrFirmaDocumentoDenegada
	}
	return nil
}
