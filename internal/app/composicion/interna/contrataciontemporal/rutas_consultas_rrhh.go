package contrataciontemporal

import (
	"reflect"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/vec/adapters/httpapi"
)

// DependenciasConsultasRRHH limita este montaje a las dos lecturas que ya
// consume la web. Las escrituras y los borradores requieren otros cortes.
type DependenciasConsultasRRHH struct {
	Cuadro  httpinterno.ConsultorCuadroRRHH
	Detalle httpinterno.ConsultorDetalleRRHH
}

// NuevasRutasConsultasRRHH conserva las rutas y manejadores HTTP canónicos
// sin exigir las dependencias de las restantes operaciones CT.
func NuevasRutasConsultasRRHH(d DependenciasConsultasRRHH) ([]httpapi.RutaExacta, error) {
	if nulaConsultaRRHH(d.Cuadro) || nulaConsultaRRHH(d.Detalle) {
		return nil, ErrRutasContratacionTemporalInvalidas
	}
	cuadro, err := httpinterno.NuevoManejadorConsultaCuadroRRHH(d.Cuadro)
	if err != nil {
		return nil, ErrRutasContratacionTemporalInvalidas
	}
	detalle, err := httpinterno.NuevoManejadorConsultaDetalleRRHH(d.Detalle)
	if err != nil {
		return nil, ErrRutasContratacionTemporalInvalidas
	}
	return []httpapi.RutaExacta{
		{Ruta: httpinterno.RutaConsultaCuadroRRHH, Manejador: cuadro},
		{Ruta: httpinterno.RutaConsultaDetalleRRHH, Manejador: detalle},
	}, nil
}

func nulaConsultaRRHH(valor any) bool {
	if valor == nil {
		return true
	}
	v := reflect.ValueOf(valor)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}
