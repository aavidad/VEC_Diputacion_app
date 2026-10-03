package firmaautorizacionv2

import "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"

func ValidarCapacidadRecuperacionFirmasV2(c ports.CapacidadRecuperacionFirmasV2, m ports.MaterialConsultaFirmasR5V2) error {
	r, err := RecursoConsultaFirmasR5V2(m)
	if err != nil { return ports.ErrFirmaDocumentoDenegada }
	h, err := r.HuellaContextoAutorizacionSHA256()
	a := c.ExportarMaterialParaConsumidor()
	s := a.ResumenCapacidad()
	if err != nil || a.ValidarEstructura() != nil || s.Operacion() != ports.AccionRecuperarFirmasR5V2 ||
		s.AudienciaConsumo() != ports.AudienciaRecuperacionFirmasR5V2 || s.EfectoRef() != r.Referencia || s.EfectoHuellaSHA256() != h {
		return ports.ErrFirmaDocumentoDenegada
	}
	return nil
}
