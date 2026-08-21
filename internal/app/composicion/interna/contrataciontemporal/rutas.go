package contrataciontemporal

import (
	"errors"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/httpapi"
)

var ErrRutasContratacionTemporalInvalidas = errors.New(
	"composicion interna: rutas de contratacion temporal invalidas",
)

// DependenciasRutas solo acepta casos de uso y autoridades ya constituidos.
// La identidad corporativa, PostgreSQL y los proveedores criptograficos
// pertenecen a fronteras anteriores de la raiz de composicion.
type DependenciasRutas struct {
	AutoridadAlta      httpinterno.AutoridadContextoCanal
	EjecutorAlta       httpinterno.EjecutorAlta
	Reloj              ports.Reloj
	AutoridadAnalisis  httpinterno.AutoridadContextoCanalAnalisisRRHH
	EjecutorAnalisis   httpinterno.EjecutorAnalisisRRHH
	AutoridadCobertura httpinterno.AutoridadContextoCanalCobertura
	Presentador        httpinterno.PresentadorPropuestaCobertura
	Decisor            httpinterno.EjecutorDecisionCobertura
	ConsultorResultado httpinterno.ConsultorResultadoCobertura
}

// NuevasRutas construye el conjunto de forma atomica. No devuelve una API
// parcial si falta una dependencia o falla un adaptador.
func NuevasRutas(
	dependencias DependenciasRutas,
) ([]httpapi.RutaExacta, error) {
	alta, err := httpinterno.NuevoManejadorAlta(
		dependencias.AutoridadAlta,
		dependencias.EjecutorAlta,
		dependencias.Reloj,
	)
	if err != nil {
		return nil, ErrRutasContratacionTemporalInvalidas
	}
	cobertura, err := httpinterno.NuevoManejadorCobertura(
		dependencias.AutoridadCobertura,
		dependencias.Presentador,
		dependencias.Decisor,
	)
	if err != nil {
		return nil, ErrRutasContratacionTemporalInvalidas
	}
	resultado, err := httpinterno.NuevoManejadorResultadoCobertura(
		dependencias.ConsultorResultado,
	)
	if err != nil {
		return nil, ErrRutasContratacionTemporalInvalidas
	}
	analisis, err := httpinterno.NuevoManejadorAnalisisRRHH(
		dependencias.AutoridadAnalisis,
		dependencias.EjecutorAnalisis,
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
		{
			Ruta:      httpinterno.RutaResultadoCobertura,
			Manejador: resultado,
		},
		{
			Ruta:      httpinterno.RutaRegistroAnalisisRRHH,
			Manejador: analisis,
		},
		{
			Ruta:      httpinterno.RutaRectificacionAnalisisRRHH,
			Manejador: analisis,
		},
	}, nil
}
