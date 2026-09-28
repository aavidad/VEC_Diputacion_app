package application

import (
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

// ProyectarRestriccionCese conserva el estado propio cuando ya es más
// restrictivo. Solo estados que podían llegar al turno se presentan a RRHH
// como disponibles desde la fecha B45. B10 reduce ese estado a no_disponible.
func ProyectarRestriccionCese(situacion ports.SituacionParticipacion, restriccion ports.RestriccionCese, activa bool) ports.SituacionParticipacion {
	if !activa {
		return situacion
	}
	switch situacion.Situacion {
	case "disponible", "trabajando", "disponible_desde":
		if restriccion.DisponibleDesde.IsZero() || restriccion.FechaEfecto.IsZero() {
			return situacion
		}
		// Una fecha ordinaria posterior también debe conservar su límite.
		fecha := restriccion.DisponibleDesde.UTC()
		if situacion.FechaDisponible != nil && situacion.FechaDisponible.After(fecha) {
			fecha = situacion.FechaDisponible.UTC()
		} else {
			situacion.Desde = restriccion.FechaEfecto.UTC()
		}
		situacion.Situacion = "disponible_desde"
		situacion.FechaDisponible = &fecha
	}
	return situacion
}

// EstadoPublicoRestriccionCese mantiene el catálogo B10 reducido. Ni fecha,
// recibo ni versión de política forman parte de una posición pública.
func EstadoPublicoRestriccionCese(situacion ports.SituacionParticipacion, corte time.Time) string {
	switch situacion.Situacion {
	case "trabajando", "pendiente_incorporacion":
		return "ocupado"
	case "renuncia":
		return "renuncia_pendiente"
	case "disponible_desde":
		if situacion.FechaDisponible != nil && !situacion.FechaDisponible.After(corte) {
			return "disponible"
		}
		return "no_disponible"
	case "disponible", "no_disponible", "excluido":
		return situacion.Situacion
	default:
		return "no_disponible"
	}
}
