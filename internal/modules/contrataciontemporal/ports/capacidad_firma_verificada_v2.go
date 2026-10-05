package ports

import (
	"bytes"
	vp "vec-diputacion-granada/internal/vec/ports"
)

type CapacidadFirmaVerificadaV2 struct {
	material         vp.ExportacionMaterialConsumoAutorizacionAtestadaV3
	descriptor       []byte
	descriptorSHA256 string
}

// El transporte antiguo no lleva descriptor para autorizar escritura V2.
// El consumidor rechaza esa ausencia.
func TransportarMaterialFirmaVerificadaV2(m vp.ExportacionMaterialConsumoAutorizacionAtestadaV3) CapacidadFirmaVerificadaV2 {
	return CapacidadFirmaVerificadaV2{material: m}
}

func TransportarMaterialFirmaVerificadaV2ConDescriptor(m vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, descriptor []byte, huella string) CapacidadFirmaVerificadaV2 {
	return CapacidadFirmaVerificadaV2{material: m, descriptor: bytes.Clone(descriptor), descriptorSHA256: huella}
}

func (c CapacidadFirmaVerificadaV2) ExportarMaterialParaConsumidor() vp.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	return c.material
}

func (c CapacidadFirmaVerificadaV2) ExportarDescriptorParaConsumidor() ([]byte, string) {
	return bytes.Clone(c.descriptor), c.descriptorSHA256
}
