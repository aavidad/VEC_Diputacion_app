package application

import (
	"context"
	"reflect"

	"vec-diputacion-granada/internal/vec/ports"
)

// ResolverPoliticaConservacionDocumental coordina la frontera documental y
// exige una unica politica exacta, aprobada y vigente en el instante resuelto.
func ResolverPoliticaConservacionDocumental(
	ctx context.Context,
	resolutor ports.ResolutorPoliticaConservacionDocumental,
	reloj ports.Reloj,
	solicitud ports.SolicitudPoliticaConservacionDocumental,
) (ports.ResultadoPoliticaConservacionDocumental, error) {
	denegar := func() (ports.ResultadoPoliticaConservacionDocumental, error) {
		return ports.ResultadoPoliticaConservacionDocumental{},
			ports.ErrPoliticaConservacionDocumentalNoResuelta
	}
	if ctx == nil || ctx.Err() != nil || solicitud.Validar() != nil ||
		dependenciaPoliticaConservacionDocumentalNula(resolutor) ||
		dependenciaPoliticaConservacionDocumentalNula(reloj) {
		return denegar()
	}
	politicas, err := resolutor.BuscarPoliticasConservacionDocumental(ctx, solicitud)
	if err != nil || ctx.Err() != nil || len(politicas) != 1 {
		return denegar()
	}
	resueltaEn := reloj.Ahora()
	if ctx.Err() != nil {
		return denegar()
	}
	resultado, err := ports.NuevoResultadoPoliticaConservacionDocumental(
		politicas[0], solicitud, resueltaEn,
	)
	if err != nil {
		return denegar()
	}
	return resultado, nil
}

func dependenciaPoliticaConservacionDocumentalNula(dependencia any) bool {
	if dependencia == nil {
		return true
	}
	valor := reflect.ValueOf(dependencia)
	switch valor.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return valor.IsNil()
	default:
		return false
	}
}
