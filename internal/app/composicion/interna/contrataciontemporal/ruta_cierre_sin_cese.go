package contrataciontemporal

import (
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/vec/adapters/httpapi"
)

// NuevaRutaCierreAdministrativoSinCese declara solo la operación nominal. La
// autoridad del canal conserva la organización fuera del navegador.
func NuevaRutaCierreAdministrativoSinCese(a httpinterno.AutoridadServidorCierreAdministrativo, e httpinterno.EjecutorCierreAdministrativo) (httpapi.RutaExacta, error) {
	h, err := httpinterno.NuevoManejadorCierreAdministrativo(a, e)
	if err != nil {
		return httpapi.RutaExacta{}, ErrRutasContratacionTemporalInvalidas
	}
	if _, ok := e.(httpinterno.EjecutorCierreAdministrativoSinCese); !ok {
		return httpapi.RutaExacta{}, ErrRutasContratacionTemporalInvalidas
	}
	return httpapi.RutaExacta{Ruta: httpinterno.RutaCerrarAdministrativamenteSinCese, Manejador: h}, nil
}
