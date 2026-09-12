package ports

import "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"

// DTO neutrales para la frontera de lectura. La validación e integridad viven
// en domain, que no depende de puertos ni de transportes.
const EsquemaPresentacionFlujoRRHH = domain.EsquemaPresentacionFlujoRRHH

type FasePresentacionFlujoRRHH = domain.FasePresentacionFlujoRRHH
type PresentacionFlujoRRHH = domain.PresentacionFlujoRRHH

var (
	ErrPresentacionFlujoRRHHInvalida    = domain.ErrPresentacionFlujoRRHHInvalida
	CalcularHuellaPresentacionFlujoRRHH = domain.CalcularHuellaPresentacionFlujoRRHH
)
