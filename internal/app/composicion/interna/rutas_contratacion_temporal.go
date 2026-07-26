package interna

import (
	"errors"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/httpapi"
)

var ErrRutasContratacionTemporalInvalidas = errors.New(
	"composicion interna: rutas de contratacion temporal invalidas",
)

// dependenciasRutasContratacionTemporal solo acepta casos de uso y autoridades
// ya constituidos. La identidad corporativa, PostgreSQL y los proveedores
// criptograficos pertenecen a fronteras anteriores de la raiz de composicion.
type dependenciasRutasContratacionTemporal struct {
	autoridadAlta      httpinterno.AutoridadContextoCanal
	ejecutorAlta       httpinterno.EjecutorAlta
	reloj              ports.Reloj
	autoridadCobertura httpinterno.AutoridadContextoCanalCobertura
	presentador        httpinterno.PresentadorPropuestaCobertura
	decisor            httpinterno.EjecutorDecisionCobertura
}

// nuevasRutasContratacionTemporal construye el conjunto de forma atomica. No
// devuelve una API parcial si falta una dependencia o falla un adaptador.
func nuevasRutasContratacionTemporal(
	dependencias dependenciasRutasContratacionTemporal,
) ([]httpapi.RutaExacta, error) {
	alta, err := httpinterno.NuevoManejadorAlta(
		dependencias.autoridadAlta,
		dependencias.ejecutorAlta,
		dependencias.reloj,
	)
	if err != nil {
		return nil, ErrRutasContratacionTemporalInvalidas
	}
	cobertura, err := httpinterno.NuevoManejadorCobertura(
		dependencias.autoridadCobertura,
		dependencias.presentador,
		dependencias.decisor,
	)
	if err != nil {
		return nil, ErrRutasContratacionTemporalInvalidas
	}
	return []httpapi.RutaExacta{
		{
			Ruta:      httpinterno.RutaAltaSolicitudes,
			Manejador: alta,
		},
		{
			Ruta:      httpinterno.RutaPropuestaCobertura,
			Manejador: cobertura,
		},
		{
			Ruta:      httpinterno.RutaDecisionCobertura,
			Manejador: cobertura,
		},
		{
			Ruta:      httpinterno.RutaRectificacionCobertura,
			Manejador: cobertura,
		},
	}, nil
}
