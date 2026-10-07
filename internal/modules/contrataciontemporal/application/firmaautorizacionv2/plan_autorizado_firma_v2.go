package firmaautorizacionv2

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
)

const esquemaPlanAutorizadoFirmaV2 = "ct.plan-autorizado-firma.v2"

type envoltorioPlanAutorizadoFirmaV2 struct {
	Esquema                string                       `json:"esquema"`
	Descriptor             json.RawMessage              `json:"descriptor"`
	Plan                   vd.ReferenciaEntradaCatalogo `json:"plan"`
	DecisionInteriorSHA256 string                       `json:"decision_interior_sha256"`
}

// CanonicoPlanAutorizadoFirmaV2 prepara el material de la autorización exterior.
// La huella interior debe proceder de la decisión exacta que consumirá CT172.
// El resultado por sí solo no acredita publicación ni autorización.
func CanonicoPlanAutorizadoFirmaV2(m ports.MaterialFirmaVerificadaV2, d ports.DescriptorPlanFijadoFirmaV2, decisionInteriorSHA256 string) ([]byte, error) {
	if d.Plan.Validar() != nil || !domain.HuellaSHA256FirmaValida(decisionInteriorSHA256) {
		return nil, ports.ErrSolicitudFirmaDocumentoInvalida
	}
	descriptor, err := CanonicoDescriptorFirmaVerificadaV2(m, d.Descriptor)
	if err != nil {
		return nil, err
	}
	return json.Marshal(envoltorioPlanAutorizadoFirmaV2{esquemaPlanAutorizadoFirmaV2, descriptor, d.Plan, decisionInteriorSHA256})
}

// ValidarPlanAutorizadoFirmaV2 coteja el pin obtenido del servidor y la decisión
// interior exacta. No se toma el valor esperado del propio JSON recibido.
func ValidarPlanAutorizadoFirmaV2(m ports.MaterialFirmaVerificadaV2, plan vd.ReferenciaEntradaCatalogo, decisionInteriorSHA256 string, datos []byte) error {
	if plan.Validar() != nil || !domain.HuellaSHA256FirmaValida(decisionInteriorSHA256) || len(datos) < 2 || len(datos) > 65536 {
		return ports.ErrFirmaDocumentoDenegada
	}
	var e envoltorioPlanAutorizadoFirmaV2
	d := json.NewDecoder(bytes.NewReader(datos))
	d.DisallowUnknownFields()
	if d.Decode(&e) != nil || d.Decode(&struct{}{}) != io.EOF || e.Esquema != esquemaPlanAutorizadoFirmaV2 ||
		e.Plan != plan || e.DecisionInteriorSHA256 != decisionInteriorSHA256 || ValidarDescriptorFirmaVerificadaV2(m, e.Descriptor) != nil {
		return ports.ErrFirmaDocumentoDenegada
	}
	canon, err := json.Marshal(e)
	if err != nil || !bytes.Equal(canon, datos) {
		return ports.ErrFirmaDocumentoDenegada
	}
	return nil
}

// RecursoPlanAutorizadoFirmaV2 mantiene acción, audiencia y recurso de firma;
// su contexto liga material, plan, descriptor y decisión interior. CT172 usa
// otro contexto: el consumidor debe confirmar ambos en una sola transacción.
// Lleva los mismos ámbitos que la decisión interior (los de la asignación de
// quien actúa), con las mismas reglas; AD209 los relee en el consumo.
func RecursoPlanAutorizadoFirmaV2(m ports.MaterialFirmaVerificadaV2, plan vd.ReferenciaEntradaCatalogo, decisionInteriorSHA256 string, envoltorio []byte, a ports.AmbitosOperadorFirmaV2) (vd.RecursoAutorizable, error) {
	if ValidarPlanAutorizadoFirmaV2(m, plan, decisionInteriorSHA256, envoltorio) != nil || a.OrganizacionRef != m.OrganizacionRef ||
		(a.UnidadRef != "" && (m.Via != ports.ViaFirmaCertificadoVEC || a.UnidadRef != m.UnidadFirmanteRef)) {
		return vd.RecursoAutorizable{}, ports.ErrSolicitudFirmaDocumentoInvalida
	}
	h, err := m.HuellaSHA256()
	if err != nil {
		return vd.RecursoAutorizable{}, err
	}
	tipo := ports.TipoRecursoFirmaExterna
	if m.Via == ports.ViaFirmaCertificadoVEC {
		tipo = ports.TipoRecursoFirmaVec
	}
	sha := sha256.Sum256(envoltorio)
	return vd.RecursoAutorizable{Referencia: m.RecursoRef(), ModuloID: ports.ModuloContratacion, Tipo: tipo,
		Ambitos:   a.Mapa(),
		Atributos: map[string]string{"material_sha256": h, "plan_firma_sha256": hex.EncodeToString(sha[:])}}, nil
}
