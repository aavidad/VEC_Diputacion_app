package bootstrap

import (
	contextopg "vec-diputacion-granada/internal/vec/adapters/contextoactor/postgres"
	vecapp "vec-diputacion-granada/internal/vec/application"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

// La autoridad F1 del firmante resuelve el mismo ContextoActor que las demás
// rutas, pero exige la proyección canónica de empleado de Personal. El
// constructor productivo rechaza una proyección ausente o ambigua.
func nuevaAutoridadContextoEmpleadoFirmaVecR5(resolutor vp.ResolutorRegistroContextoActorV2,
	reloj vp.Reloj) (core.ResolutorContextoActorRegistradoV2, error) {
	alcance, err := core.NuevoAlcanceProyeccionesContextoActor(core.ProyeccionContextoActorEmpleado)
	if err != nil {
		return nil, err
	}
	servicio, err := vecapp.NuevoServicioContextoActorProductivoV2ConAlcance(
		resolutor, contextopg.NuevoGeneradorOperacionContextoActorV2Criptografico(), reloj, alcance)
	if err != nil {
		return nil, err
	}
	return vecapp.NuevaAutoridadContextoActorRegistradoV2(servicio)
}
