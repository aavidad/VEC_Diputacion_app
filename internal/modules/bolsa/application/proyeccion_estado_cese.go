package application

import (
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

// ProyectarSituacionConEstadoCese usa la fecha efectiva B45, que combina el
// último cese acreditado y el mayor límite de disponibilidad. Al vencer,
// vuelve al turno solo si B45 acredita que no queda otra relación abierta.
// Una pausa B2, exclusión o renuncia conservan su autoridad propia.
func ProyectarSituacionConEstadoCese(base ports.SituacionParticipacion, estado ports.EstadoCese, presente bool, corte time.Time) (ports.SituacionParticipacion, error) {
	if !presente {
		return base, nil
	}
	if estado.CesePendiente {
		if corte.IsZero() || estado.PendienteDesde.IsZero() || estado.PendienteDesde.After(corte) ||
			estado.FechaEfecto.IsZero() != estado.DisponibleDesde.IsZero() ||
			estado.FechaEfecto.IsZero() && (estado.EnRestriccion || estado.TrabajoCesado) ||
			!estado.FechaEfecto.IsZero() && (estado.FechaEfecto.After(corte) || estado.DisponibleDesde.Before(estado.FechaEfecto)) {
			return ports.SituacionParticipacion{}, ports.ErrConsultaEstadoCeseNoDisponible
		}
		switch base.Situacion {
		case "disponible", "trabajando", "disponible_desde":
			base.Situacion = "no_disponible"
			base.Desde = estado.PendienteDesde.UTC()
			base.FechaDisponible = nil
		}
		return base, nil
	}
	if corte.IsZero() || !estado.PendienteDesde.IsZero() || estado.FechaEfecto.IsZero() || estado.DisponibleDesde.IsZero() ||
		estado.FechaEfecto.After(corte) || estado.DisponibleDesde.Before(estado.FechaEfecto) ||
		estado.EnRestriccion && !estado.DisponibleDesde.After(corte) {
		return ports.SituacionParticipacion{}, ports.ErrConsultaEstadoCeseNoDisponible
	}
	switch base.Situacion {
	case "disponible", "trabajando", "disponible_desde":
	default:
		return base, nil
	}
	if estado.EnRestriccion {
		fecha := estado.DisponibleDesde.UTC()
		if base.FechaDisponible != nil && base.FechaDisponible.After(fecha) {
			fecha = base.FechaDisponible.UTC()
		} else {
			base.Desde = estado.FechaEfecto.UTC()
		}
		base.Situacion = "disponible_desde"
		base.FechaDisponible = &fecha
		return base, nil
	}
	if base.Situacion == "trabajando" && estado.TrabajoCesado {
		if base.FechaDisponible == nil || !base.FechaDisponible.After(corte) {
			base.Situacion = "disponible"
			base.Desde = estado.FechaEfecto.UTC()
			base.FechaDisponible = nil
		} else {
			base.Situacion = "disponible_desde"
		}
	}
	if base.Situacion == "disponible_desde" && base.FechaDisponible != nil && !base.FechaDisponible.After(corte) {
		base.Situacion = "disponible"
		base.Desde = base.FechaDisponible.UTC()
		base.FechaDisponible = nil
	}
	return base, nil
}
