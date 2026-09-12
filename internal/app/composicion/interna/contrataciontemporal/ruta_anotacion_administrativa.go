package contrataciontemporal

import (
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/vec/adapters/httpapi"
)

// NuevasRutasAnotacionAdministrativa declara el POST y su recuperación GET.
// La raíz decide si las publica después de verificar configuración y autoridad.
func NuevasRutasAnotacionAdministrativa(a httpinterno.AutoridadContextoCanalAnotacionAdministrativa, e httpinterno.EjecutorAnotacionAdministrativa) ([]httpapi.RutaExacta, error) {
	h, err := httpinterno.NuevoManejadorAnotacionAdministrativa(a, e)
	if err != nil {
		return nil, ErrRutasContratacionTemporalInvalidas
	}
	return []httpapi.RutaExacta{
		{Ruta: httpinterno.RutaAnotacionesAdministrativas, Manejador: h},
		{Ruta: httpinterno.RutaRecuperacionAnotacionesAdministrativas, Manejador: h},
	}, nil
}
