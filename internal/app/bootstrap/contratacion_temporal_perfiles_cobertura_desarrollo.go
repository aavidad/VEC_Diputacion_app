package bootstrap

import (
	cthttp "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// La composición, y no la petición, fija el perfil que ejecuta cada acción.
// Los cuatro casos de cobertura comparten su vínculo y su asignación propia.
var accionesPerfilCoberturaCTDesarrollo = map[string]string{
	cthttp.RutaPropuestaCobertura:     accionPropuestaCoberturaDesarrollo,
	cthttp.RutaDecisionCobertura:      string(ctdomain.AccionDecidirCoberturaGobernada),
	cthttp.RutaRectificacionCobertura: string(ctdomain.AccionRectificarCoberturaGobernada),
	cthttp.RutaResultadoCobertura:     string(ctports.AccionConsultarResultadoCobertura),
}

func rutaPerfilCoberturaCTDesarrollo(ruta string) bool {
	_, existe := accionesPerfilCoberturaCTDesarrollo[ruta]
	return existe
}

func asignarPerfilCoberturaCTDesarrollo(
	declaraciones []descriptorFronteraComunDesarrollo, perfilAlta, perfilCobertura string,
) ([]descriptorFronteraComunDesarrollo, error) {
	if !perfilActivoSeguridadComunValido(perfilAlta) ||
		!perfilActivoSeguridadComunValido(perfilCobertura) || perfilAlta == perfilCobertura {
		return nil, ErrActivacionDesarrolloInvalida
	}
	resultado := append([]descriptorFronteraComunDesarrollo(nil), declaraciones...)
	vistas := make(map[string]struct{}, len(accionesPerfilCoberturaCTDesarrollo))
	for i := range resultado {
		descriptor := &resultado[i]
		accion, esCobertura := accionesPerfilCoberturaCTDesarrollo[descriptor.Ruta]
		if !esCobertura {
			continue
		}
		if descriptor.ClaveCapacidad != accion || len(descriptor.PerfilesActivosRef) != 1 ||
			descriptor.PerfilesActivosRef[0] != perfilAlta {
			return nil, ErrActivacionDesarrolloInvalida
		}
		if _, repetida := vistas[descriptor.Ruta]; repetida {
			return nil, ErrActivacionDesarrolloInvalida
		}
		vistas[descriptor.Ruta] = struct{}{}
		descriptor.PerfilesActivosRef = []string{perfilCobertura}
	}
	if len(vistas) != len(accionesPerfilCoberturaCTDesarrollo) {
		return nil, ErrActivacionDesarrolloInvalida
	}
	return resultado, nil
}
