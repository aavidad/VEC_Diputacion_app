package application

import (
	firma "vec-diputacion-granada/internal/modules/contrataciontemporal/application/firmaautorizacionv2"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func CanonicoDescriptorFirmaVerificadaV2(m ports.MaterialFirmaVerificadaV2, d ports.DescriptorConstructorFirmaV2) ([]byte, error) {
	return firma.CanonicoDescriptorFirmaVerificadaV2(m, d)
}
func ValidarDescriptorFirmaVerificadaV2(m ports.MaterialFirmaVerificadaV2, datos []byte) error {
	return firma.ValidarDescriptorFirmaVerificadaV2(m, datos)
}
