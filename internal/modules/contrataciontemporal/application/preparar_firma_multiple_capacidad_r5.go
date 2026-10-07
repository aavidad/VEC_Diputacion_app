package application

import (
	firma "vec-diputacion-granada/internal/modules/contrataciontemporal/application/firmaautorizacionv2"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
)

func RecursoFirmaVerificadaV2(m ports.MaterialFirmaVerificadaV2, descriptor []byte) (vd.RecursoAutorizable, error) {
	return firma.RecursoFirmaVerificadaV2(m, descriptor)
}
func RecursoFirmaVerificadaV2ConAmbitos(m ports.MaterialFirmaVerificadaV2, descriptor []byte, a ports.AmbitosOperadorFirmaV2) (vd.RecursoAutorizable, error) {
	return firma.RecursoFirmaVerificadaV2ConAmbitos(m, descriptor, a)
}
func ValidarCapacidadFirmaVerificadaV2(c ports.CapacidadFirmaVerificadaV2, m ports.MaterialFirmaVerificadaV2) error {
	return firma.ValidarCapacidadFirmaVerificadaV2(c, m)
}
func RecursoConsultaFirmasR5V2(m ports.MaterialConsultaFirmasR5V2) (vd.RecursoAutorizable, error) {
	return firma.RecursoConsultaFirmasR5V2(m)
}
func ValidarCapacidadConsultaFirmasR5V2(c ports.CapacidadConsultaFirmasR5V2, m ports.MaterialConsultaFirmasR5V2) error {
	return firma.ValidarCapacidadConsultaFirmasR5V2(c, m)
}
