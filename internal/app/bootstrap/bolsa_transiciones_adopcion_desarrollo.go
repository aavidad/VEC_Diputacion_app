package bootstrap

import (
	"time"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// Una versión gobernada puede sumar la lectura S6 y, si existe, su edición.
// Se adopta intacta; esta comprobación no construye ni publica concesiones.
func instantaneaBolsaS6Compatible(publicada dominiovec.InstantaneaAutorizacion,
	vinculo dominiovec.DatosVinculoAutenticacionActorV2,
	soporte *soporteSesionBorradorBolsaDesarrollo, ahora time.Time, versionBase int) bool {
	return instantaneaBolsaGobernadaCompatible(publicada, vinculo, soporte, ahora,
		versionBase, concesionesAdicionalesS6Exactas)
}
