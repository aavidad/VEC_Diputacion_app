package composicion

import (
	"net/http"

	"vec-diputacion-granada/internal/modules/cronos/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

func (r *ResolutorPeticionCronos) ResolverConsultaMovimientosPropios(req *http.Request) (ports.OrdenConsultaMovimientos, error) {
	p, err := r.peticion(req)
	if err != nil {
		return ports.OrdenConsultaMovimientos{}, err
	}
	return ports.NuevaOrdenConsultaMovimientos(p.identidad.Contexto.Contexto, p)
}

func (r *ResolutorPeticionCronos) ResolverSolicitudCorreccionPropia(req *http.Request) (ports.OrdenConsumoCorreccion, error) {
	p, err := r.peticion(req)
	if err != nil {
		return ports.OrdenConsumoCorreccion{}, err
	}
	return ports.NuevaOrdenConsumoCorreccion(p.identidad.Contexto.Contexto, p)
}

func (r *ResolutorPeticionCronos) ResolverPermisosPropios(req *http.Request) (ports.OrdenPermisosPropios, error) {
	p, err := r.peticion(req)
	if err != nil {
		return ports.OrdenPermisosPropios{}, err
	}
	return ports.NuevaOrdenPermisosPropios(p.identidad.Contexto.Contexto, p)
}

var (
	_ httpinterno.ResolverConsultaMovimientosPropios = (*ResolutorPeticionCronos)(nil)
	_ httpinterno.ResolverSolicitudCorreccionPropia  = (*ResolutorPeticionCronos)(nil)
	_ httpinterno.ResolverPermisosPropios            = (*ResolutorPeticionCronos)(nil)
)
