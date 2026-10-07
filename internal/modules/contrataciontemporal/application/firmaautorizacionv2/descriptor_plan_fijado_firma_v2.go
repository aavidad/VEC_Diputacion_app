package firmaautorizacionv2

import (
	"bytes"
	"encoding/json"
	"io"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
)

const esquemaDescriptorPlanFijadoFirmaV2 = "ct.descriptor-plan-fijado-firma.v2"

type envoltorioDescriptorPlanFijado struct {
	Esquema    string                       `json:"esquema"`
	Descriptor json.RawMessage              `json:"descriptor"`
	Plan       vd.ReferenciaEntradaCatalogo `json:"plan"`
}

// CanonicoDescriptorPlanFijadoFirmaV2 liga el pin a los mismos bytes del
// descriptor nominal. Un JSON válido no acredita la publicación del plan;
// la fachada durable propietaria debe cotejarla antes del COMMIT.
func CanonicoDescriptorPlanFijadoFirmaV2(m ports.MaterialFirmaVerificadaV2, d ports.DescriptorPlanFijadoFirmaV2) ([]byte, error) {
	if d.Plan.Validar() != nil {
		return nil, ports.ErrCompetenciaFirmanteNoAcreditada
	}
	canon, err := CanonicoDescriptorFirmaVerificadaV2(m, d.Descriptor)
	if err != nil {
		return nil, err
	}
	return json.Marshal(envoltorioDescriptorPlanFijado{esquemaDescriptorPlanFijadoFirmaV2, canon, d.Plan})
}

// ValidarDescriptorPlanFijadoFirmaV2 coteja el pin esperado además del JSON
// canónico. El consumidor obtiene ese pin de la autoridad de publicación,
// nunca de una segunda lectura del propio envoltorio recibido.
func ValidarDescriptorPlanFijadoFirmaV2(m ports.MaterialFirmaVerificadaV2, esperado vd.ReferenciaEntradaCatalogo, datos []byte) error {
	if esperado.Validar() != nil || len(datos) < 2 || len(datos) > 65536 {
		return ports.ErrFirmaDocumentoDenegada
	}
	var e envoltorioDescriptorPlanFijado
	dec := json.NewDecoder(bytes.NewReader(datos))
	dec.DisallowUnknownFields()
	if dec.Decode(&e) != nil || dec.Decode(&struct{}{}) != io.EOF || e.Esquema != esquemaDescriptorPlanFijadoFirmaV2 || e.Plan != esperado {
		return ports.ErrFirmaDocumentoDenegada
	}
	if ValidarDescriptorFirmaVerificadaV2(m, e.Descriptor) != nil {
		return ports.ErrFirmaDocumentoDenegada
	}
	canon, err := json.Marshal(e)
	if err != nil || !bytes.Equal(canon, datos) {
		return ports.ErrFirmaDocumentoDenegada
	}
	return nil
}
