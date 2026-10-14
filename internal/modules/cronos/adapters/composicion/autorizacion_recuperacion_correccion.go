package composicion

import (
	"context"

	"vec-diputacion-granada/internal/modules/cronos/application"
	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func (p proveedorPeticion) ProveerMaterialRecuperacionCorreccionPropia(ctx context.Context, m domain.MaterialRecuperacionCorreccionPropia) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	if !mismaPersona(p.identidad, m.ActorRef, m.PerfilRef, m.EmpleadoRef) {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrDependenciaNoDisponible
	}
	recurso, err := application.RecursoRecuperarReciboCorreccionPropia(m)
	if err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrDependenciaNoDisponible
	}
	return p.autorizador.emitir(ctx, p.identidad, contratoV3{
		application.AccionRecuperarReciboCorreccion,
		application.FinalidadRecuperarReciboCorreccionPropia,
		application.AudienciaRecuperarReciboCorreccionPropia,
		p.autorizador.motivos.Correccion,
	}, recurso)
}

var _ ports.ProveedorMaterialRecuperacionCorreccionPropia = proveedorPeticion{}
