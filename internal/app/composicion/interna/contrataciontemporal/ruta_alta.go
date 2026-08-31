package contrataciontemporal

import (
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/httpapi"
)

// NuevaRutaAlta construye exclusivamente la declaración que la autoridad
// exterior de rutas exactas podrá consumir. No registra ni publica la ruta.
func NuevaRutaAlta(
	autoridad httpinterno.AutoridadContextoCanal,
	ejecutor httpinterno.EjecutorAlta,
	reloj ports.Reloj,
) (httpapi.RutaExacta, error) {
	manejador, err := httpinterno.NuevoManejadorAlta(
		autoridad,
		ejecutor,
		reloj,
	)
	if err != nil {
		return httpapi.RutaExacta{}, ErrRutasContratacionTemporalInvalidas
	}
	return httpapi.RutaExacta{
		Ruta:      httpinterno.RutaAltaSolicitudes,
		Manejador: manejador,
	}, nil
}
