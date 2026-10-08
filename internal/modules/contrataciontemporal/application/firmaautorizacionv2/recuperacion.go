package firmaautorizacionv2

import (
	"slices"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func ValidarCapacidadRecuperacionFirmasV2(c ports.CapacidadRecuperacionFirmasV2, m ports.MaterialConsultaFirmasR5V2) error {
	r, err := RecursoConsultaFirmasR5V2(m)
	if err != nil {
		return ports.ErrFirmaDocumentoDenegada
	}
	h, err := r.HuellaContextoAutorizacionSHA256()
	a := c.ExportarMaterialParaConsumidor()
	s := a.ResumenCapacidad()
	campos, obligaciones := c.RestriccionesParaConsumidor()
	if err != nil || a.ValidarEstructura() != nil || s.Operacion() != ports.AccionRecuperarFirmasR5V2 ||
		s.AudienciaConsumo() != ports.AudienciaRecuperacionFirmasR5V2 || s.EfectoRef() != r.Referencia || s.EfectoHuellaSHA256() != h ||
		!slices.Equal(campos, ports.CamposRecuperacionFirmasV2()) || len(obligaciones) != 0 {
		return ports.ErrFirmaDocumentoDenegada
	}
	return nil
}
