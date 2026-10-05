package ports

import (
	"bytes"
	"context"

	vd "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

// CapacidadFirmaConPlanV2 transporta dos exportaciones distintas a una sola
// transacción. La interior conserva el descriptor nominal de CT172.
type CapacidadFirmaConPlanV2 struct {
	interior    CapacidadFirmaVerificadaV2
	exterior    vp.ExportacionMaterialConsumoAutorizacionAtestadaV3
	plan        vd.ReferenciaEntradaCatalogo
	envoltorio  []byte
	decisionSHA string
}

func TransportarFirmaConPlanV2(interior CapacidadFirmaVerificadaV2,
	exterior vp.ExportacionMaterialConsumoAutorizacionAtestadaV3,
	plan vd.ReferenciaEntradaCatalogo, envoltorio []byte, decisionSHA string,
) CapacidadFirmaConPlanV2 {
	return CapacidadFirmaConPlanV2{interior, exterior, plan, bytes.Clone(envoltorio), decisionSHA}
}

func (c CapacidadFirmaConPlanV2) ExportarParaConsumidor() (CapacidadFirmaVerificadaV2,
	vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, vd.ReferenciaEntradaCatalogo, []byte, string) {
	return c.interior, c.exterior, c.plan, bytes.Clone(c.envoltorio), c.decisionSHA
}

type RegistradorFirmaConPlanV2 interface {
	RegistrarFirmaConPlanV2(context.Context, MaterialFirmaVerificadaV2, CapacidadFirmaConPlanV2) (ReciboFirmaDocumento, error)
}
