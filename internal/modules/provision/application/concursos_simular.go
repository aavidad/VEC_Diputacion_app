package application

import "vec-diputacion-granada/internal/modules/provision/domain"

// Simular es el único caso de uso de valoración local para CLI y web. No
// dispone de puertos institucionales ni acepta persistencia como efecto.
func Simular(c domain.Configuracion, e domain.Entrada) (domain.Resultado, error) {
	return domain.Calcular(c, e)
}
