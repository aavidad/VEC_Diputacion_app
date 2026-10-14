package composicion

import (
	"context"
	"net/http"

	"vec-diputacion-granada/internal/modules/cronos/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/cronos/application"
	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func (p proveedorPeticion) ProveerMaterialBandejaCorrecciones(ctx context.Context, m domain.MaterialBandejaCorrecciones) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	if p.autorizador == nil || !mismaPersona(p.identidad, m.ActorRef, m.PerfilRef, m.EmpleadoRef) {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrDependenciaNoDisponible
	}
	recurso, err := application.RecursoBandejaCorrecciones(m)
	if err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrDependenciaNoDisponible
	}
	return p.autorizador.emitir(ctx, p.identidad, contratoV3{
		application.AccionBandejaCorrecciones, application.FinalidadBandejaCorrecciones,
		application.AudienciaBandejaCorrecciones, p.autorizador.motivos.Movimientos,
	}, recurso)
}

func (p proveedorPeticion) ProveerMaterialActuacionCorreccion(ctx context.Context, m domain.MaterialActuacionCorreccion) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	if p.autorizador == nil || !mismaPersona(p.identidad, m.ActorRef, m.PerfilRef, m.EmpleadoRef) {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrDependenciaNoDisponible
	}
	accion, finalidad, audiencia, ok := application.ContratoActuacionCorreccion(m.Paso)
	if !ok {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrDependenciaNoDisponible
	}
	recurso, err := application.RecursoActuacionCorreccion(m)
	if err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrDependenciaNoDisponible
	}
	return p.autorizador.emitir(ctx, p.identidad, contratoV3{accion, finalidad, audiencia, p.autorizador.motivos.Correccion}, recurso)
}

func (p proveedorPeticion) ProveerMaterialHistoriaCorreccion(ctx context.Context, m domain.MaterialHistoriaCorreccion) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	if p.autorizador == nil || !mismaPersona(p.identidad, m.ActorRef, m.PerfilRef, m.EmpleadoRef) {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrDependenciaNoDisponible
	}
	accion, finalidad, audiencia, ok := application.ContratoHistoriaCorreccion(m)
	if !ok {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrDependenciaNoDisponible
	}
	recurso, err := application.RecursoHistoriaCorreccion(m)
	if err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrDependenciaNoDisponible
	}
	return p.autorizador.emitir(ctx, p.identidad, contratoV3{accion, finalidad, audiencia, p.autorizador.motivos.Movimientos}, recurso)
}

func (r *ResolutorPeticionCronos) ResolverDecisionCorrecciones(req *http.Request) (ports.OrdenConsumoCorreccion, error) {
	p, err := r.peticion(req)
	if err != nil {
		return ports.OrdenConsumoCorreccion{}, err
	}
	return ports.NuevaOrdenConsumoCorreccion(p.identidad.Contexto.Contexto, p)
}

var (
	_ ports.ProveedorMaterialDecisionCorreccion = proveedorPeticion{}
	_ httpinterno.ResolverDecisionCorrecciones  = (*ResolutorPeticionCronos)(nil)
)
