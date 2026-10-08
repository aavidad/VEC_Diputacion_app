package catalogoalta

import (
	"context"
	"crypto/subtle"
	"reflect"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type RecuperadorInstantaneaConfirmada interface {
	RecuperarInstantaneaNecesidadConfirmada(context.Context, ports.ConsultaInstantaneaNecesidadConfirmada) (ports.ResultadoInstantaneaNecesidadAlta, error)
}

// Fuente sólo admite la publicación local declarada y la instantánea de una
// confirmación anterior. Una versión retirada no se sustituye por la vigente.
type Fuente struct {
	ruta        string
	recuperador RecuperadorInstantaneaConfirmada
}

func NuevaFuente(ruta string, recuperador RecuperadorInstantaneaConfirmada) (*Fuente, error) {
	if recuperador == nil || (reflect.ValueOf(recuperador).Kind() == reflect.Pointer && reflect.ValueOf(recuperador).IsNil()) {
		return nil, ports.ErrFuenteNecesidadesAltaNoDisponible
	}
	if _, err := CargarNecesidades(ruta); err != nil {
		return nil, ports.ErrFuenteNecesidadesAltaNoDisponible
	}
	return &Fuente{ruta: ruta, recuperador: recuperador}, nil
}

func (f *Fuente) ResolverCatalogoNecesidadesAlta(
	ctx context.Context, referencia string, version uint64, huella string,
) (domain.CatalogoNecesidadesAlta, error) {
	if ctx == nil || f == nil || ctx.Err() != nil {
		return domain.CatalogoNecesidadesAlta{}, ports.ErrFuenteNecesidadesAltaNoDisponible
	}
	catalogo, err := CargarNecesidades(f.ruta)
	if err != nil || catalogo.Referencia != referencia || catalogo.Version != version ||
		subtle.ConstantTimeCompare([]byte(catalogo.HuellaSHA256), []byte(huella)) != 1 {
		return domain.CatalogoNecesidadesAlta{}, ports.ErrFuenteNecesidadesAltaNoDisponible
	}
	return catalogo, nil
}

func (f *Fuente) RecuperarInstantaneaNecesidadConfirmada(
	ctx context.Context, consulta ports.ConsultaInstantaneaNecesidadConfirmada,
) (ports.ResultadoInstantaneaNecesidadAlta, error) {
	if ctx == nil || f == nil || consulta.Validar() != nil || ctx.Err() != nil {
		return ports.ResultadoInstantaneaNecesidadAlta{}, ports.ErrFuenteNecesidadesAltaNoDisponible
	}
	resultado, err := f.recuperador.RecuperarInstantaneaNecesidadConfirmada(ctx, consulta)
	if err != nil || resultado.Validar() != nil {
		return ports.ResultadoInstantaneaNecesidadAlta{}, ports.ErrFuenteNecesidadesAltaNoDisponible
	}
	if resultado.Estado == ports.InstantaneaNecesidadConfirmadaV3 {
		if _, err := domain.RestaurarCatalogoNecesidadesAlta(resultado.Instantanea); err != nil {
			return ports.ResultadoInstantaneaNecesidadAlta{}, ports.ErrFuenteNecesidadesAltaNoDisponible
		}
	}
	resultado.Instantanea = append([]byte(nil), resultado.Instantanea...)
	return resultado, nil
}
