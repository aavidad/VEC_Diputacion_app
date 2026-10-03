package ports

import (
	"context"
	"slices"
	"vec-diputacion-granada/internal/vec/ports"
)

const (
	AccionRecuperarFirmasR5V2       = "contratacion_temporal.documento.firmas_r5_v2.recuperar"
	AudienciaRecuperacionFirmasR5V2 = "vec_contratacion_temporal.firmas_r5.recuperar.v2"
)

// CanonNominal conserva el UTF-8 original de la autoridad de competencia.
// No se reconstruye desde el descriptor CT ni desde fuentes actuales.
type RecuperacionFirmaV2 struct {
	FirmaRef           string
	MaterialRootSHA256 string
	CanonNominal       string
	CanonNominalSHA256 string
	CanonNominalRef    string
}

type LecturaRecuperacionFirmasV2 struct {
	LecturaFirmasR5V2
	Recuperaciones []RecuperacionFirmaV2
}

type CapacidadRecuperacionFirmasV2 struct {
	material     ports.ExportacionMaterialConsumoAutorizacionAtestadaV3
	campos       []string
	obligaciones []string
}

// Campos y obligaciones proceden de RestriccionesProyeccionPara de la decisión
// nominal ligada. El consumidor durable debe volver a verificar la decisión.
func TransportarMaterialRecuperacionFirmasV2(m ports.ExportacionMaterialConsumoAutorizacionAtestadaV3, campos, obligaciones []string) CapacidadRecuperacionFirmasV2 {
	return CapacidadRecuperacionFirmasV2{m, slices.Clone(campos), slices.Clone(obligaciones)}
}
func (c CapacidadRecuperacionFirmasV2) ExportarMaterialParaConsumidor() ports.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	return c.material
}
func (c CapacidadRecuperacionFirmasV2) RestriccionesParaConsumidor() ([]string, []string) {
	return slices.Clone(c.campos), slices.Clone(c.obligaciones)
}

type AutorizadorRecuperacionFirmasV2 interface {
	AutorizarRecuperacionFirmasV2(context.Context, MaterialConsultaFirmasR5V2) (CapacidadRecuperacionFirmasV2, error)
}
type LectorRecuperacionFirmasV2 interface {
	RecuperarFirmasAutorizadasV2(context.Context, MaterialConsultaFirmasR5V2, CapacidadRecuperacionFirmasV2) (LecturaRecuperacionFirmasV2, error)
}

// La nueva acción exige los 48 campos exactos; la consulta anterior mantiene 44.
func CamposRecuperacionFirmasV2() []string {
	c := append(CamposConsultaFirmasR5V2(), "CanonNominal", "CanonNominalRef", "CanonNominalSHA256", "MaterialRootSHA256")
	slices.Sort(c)
	return c
}
