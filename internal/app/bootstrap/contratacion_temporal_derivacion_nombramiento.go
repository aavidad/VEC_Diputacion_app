package bootstrap

import (
	"context"
	"net/http"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// La composición aporta el caso de uso con su autoridad propia. Mientras no
// esté disponible, la ruta permanece cerrada sin tomar permisos del padre.
func nuevaRutaDerivacionNombramientoDesarrollo(e httpinterno.EjecutorDerivacionNombramiento) (http.Handler, error) {
	if dependenciaEsNulaContratacionTemporalDesarrollo(e) {
		e = derivacionNombramientoNoCompuesta{}
	}
	return httpinterno.NuevoManejadorDerivacionNombramiento(e)
}

type derivacionNombramientoNoCompuesta struct{}

func (derivacionNombramientoNoCompuesta) Derivar(context.Context,
	ports.EntradaDerivacionNombramiento) (ports.ResultadoDerivacionNombramiento, error) {
	return ports.ResultadoDerivacionNombramiento{}, ports.ErrDerivacionNombramientoNoDisponible
}
