package bootstrap

import (
	"context"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// resolutorAlcanceEstadisticasRRHHDesarrollo obtiene el ámbito de las
// estadísticas (C18) del lector RRHH acreditado por mTLS en la petición: el
// mismo lector, organización y ámbito con los que consulta el cuadro. Nada
// procede de la URL ni de cabeceras.
type resolutorAlcanceEstadisticasRRHHDesarrollo struct {
	sello      *selloConsultasContratacionTemporalDesarrollo
	resolvedor *resolvedorIdentidadDesarrollo
}

func (r *resolutorAlcanceEstadisticasRRHHDesarrollo) AlcanceEstadisticasRRHH(ctx context.Context) (ports.AlcanceEstadisticasRRHH, error) {
	if r == nil || r.sello == nil || r.resolvedor == nil || contextoInterfazNulo(ctx) || ctx.Err() != nil {
		return ports.AlcanceEstadisticasRRHH{}, ports.ErrEstadisticasRRHHNoDisponible
	}
	capacidad, existe := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	if !existe || capacidad.sello != r.sello || capacidad.ruta != httpinterno.RutaEstadisticasRRHH {
		return ports.AlcanceEstadisticasRRHH{}, ports.ErrEstadisticasRRHHNoDisponible
	}
	lector, ok := r.resolvedor.lectorConsultaRRHH(capacidad.principal)
	if !ok {
		return ports.AlcanceEstadisticasRRHH{}, ports.ErrEstadisticasRRHHNoDisponible
	}
	alcance := ports.AlcanceEstadisticasRRHH{OrganizacionRef: lector.organizacionRef, ClaseAmbito: lector.clase, AmbitoRef: lector.ambitoRef}
	if err := alcance.Validar(); err != nil {
		return ports.AlcanceEstadisticasRRHH{}, err
	}
	return alcance, nil
}
