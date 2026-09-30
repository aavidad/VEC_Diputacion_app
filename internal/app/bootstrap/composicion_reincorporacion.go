package bootstrap

import (
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
)

// ComponerRutasReincorporacionTitular reutiliza CT130 y CT134 con los puertos
// recibidos. La composición no publica permisos ni cambia la relación de
// Personal o la disponibilidad de Bolsa.
func ComponerRutasReincorporacionTitular(
	autoridad httpinterno.AutoridadCanalSeguimiento,
	dependencias application.DependenciasReincorporacionTitular,
	capacidad httpinterno.ComprobadorCapacidadReincorporacionTitular,
) ([]vechttp.RutaExacta, error) {
	servicio, err := application.NuevoServicioReincorporacionTitular(dependencias)
	if err != nil {
		return nil, err
	}
	registro, err := httpinterno.NuevoManejadorReincorporacionTitular(autoridad, servicio)
	if err != nil {
		return nil, err
	}
	consulta, err := httpinterno.NuevoManejadorCapacidadReincorporacionTitular(autoridad, capacidad)
	if err != nil {
		return nil, err
	}
	return []vechttp.RutaExacta{
		{Ruta: httpinterno.RutaReincorporacionesTitular, Manejador: registro},
		{Ruta: httpinterno.RutaCapacidadReincorporacionTitular, Manejador: consulta},
	}, nil
}
