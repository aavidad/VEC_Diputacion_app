package application

import (
	"context"

	aplicacionvec "vec-diputacion-granada/internal/vec/application"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

// ServicioConsultaPoliticaConservacionDocumental permite que Contratacion
// temporal consuma la resolucion gobernada por la capacidad documental comun.
// No selecciona politicas ni interpreta la conservacion como permiso de
// borrado o expurgo.
type ServicioConsultaPoliticaConservacionDocumental struct {
	resolutor puertosvec.ResolutorPoliticaConservacionDocumental
	reloj     puertosvec.Reloj
}

func NuevoServicioConsultaPoliticaConservacionDocumental(
	resolutor puertosvec.ResolutorPoliticaConservacionDocumental,
	reloj puertosvec.Reloj,
) (*ServicioConsultaPoliticaConservacionDocumental, error) {
	if dependenciaNula(resolutor) || dependenciaNula(reloj) {
		return nil, puertosvec.ErrPoliticaConservacionDocumentalNoResuelta
	}
	return &ServicioConsultaPoliticaConservacionDocumental{
		resolutor: resolutor,
		reloj:     reloj,
	}, nil
}

// Obtener delega sin rutas alternativas en el coordinador documental de VEC.
// Cualquier fallo conserva el resultado cero y el error publico opaco comun.
func (s *ServicioConsultaPoliticaConservacionDocumental) Obtener(
	ctx context.Context,
	solicitud puertosvec.SolicitudPoliticaConservacionDocumental,
) (puertosvec.ResultadoPoliticaConservacionDocumental, error) {
	if s == nil {
		return puertosvec.ResultadoPoliticaConservacionDocumental{},
			puertosvec.ErrPoliticaConservacionDocumentalNoResuelta
	}
	return aplicacionvec.ResolverPoliticaConservacionDocumental(
		ctx,
		s.resolutor,
		s.reloj,
		solicitud,
	)
}
