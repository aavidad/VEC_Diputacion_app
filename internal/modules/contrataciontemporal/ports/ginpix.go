package ports

import (
	"errors"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

var ErrSolicitudMapeoGINPIXInvalida = errors.New(
	"contratacion temporal: solicitud de mapeo ginpix invalida",
)

// SolicitudMapeoGINPIX retiene copias validadas del modelo y del mapeo. No es
// una orden de envío ni concede autoridad sobre ningún sistema externo.
type SolicitudMapeoGINPIX struct {
	modelo domain.ModeloCanonicoGINPIX
	mapeo  domain.MapeoVersionadoGINPIX
}

func NuevaSolicitudMapeoGINPIX(
	modelo domain.ModeloCanonicoGINPIX,
	mapeo domain.MapeoVersionadoGINPIX,
) (SolicitudMapeoGINPIX, error) {
	modeloClonado, errModelo := domain.RestaurarModeloCanonicoGINPIX(
		modelo.Publicacion(),
	)
	mapeoClonado, errMapeo := domain.RestaurarMapeoVersionadoGINPIX(
		mapeo.Publicacion(),
	)
	if errModelo != nil || errMapeo != nil {
		return SolicitudMapeoGINPIX{}, ErrSolicitudMapeoGINPIXInvalida
	}
	return SolicitudMapeoGINPIX{
		modelo: modeloClonado,
		mapeo:  mapeoClonado,
	}, nil
}

func (s SolicitudMapeoGINPIX) Validar() error {
	if s.modelo.Validar() != nil || s.mapeo.Validar() != nil {
		return ErrSolicitudMapeoGINPIXInvalida
	}
	return nil
}

func (s SolicitudMapeoGINPIX) Modelo() (domain.ModeloCanonicoGINPIX, error) {
	if s.Validar() != nil {
		return domain.ModeloCanonicoGINPIX{}, ErrSolicitudMapeoGINPIXInvalida
	}
	return domain.RestaurarModeloCanonicoGINPIX(s.modelo.Publicacion())
}

func (s SolicitudMapeoGINPIX) Mapeo() (domain.MapeoVersionadoGINPIX, error) {
	if s.Validar() != nil {
		return domain.MapeoVersionadoGINPIX{}, ErrSolicitudMapeoGINPIXInvalida
	}
	return domain.RestaurarMapeoVersionadoGINPIX(s.mapeo.Publicacion())
}

// MapeadorGINPIX define únicamente la transformación neutral. Los futuros
// adaptadores API y fichero tendrán contratos de efecto separados.
type MapeadorGINPIX interface {
	Mapear(SolicitudMapeoGINPIX) (domain.CargaMapeadaGINPIX, error)
}
