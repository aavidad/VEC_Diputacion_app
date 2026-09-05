package bootstrap

import (
	"context"

	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// Conserva el único repositorio de ejecuciones y habilita únicamente la
// recuperación acotada del proveedor Bolsa idempotente compuesto en desarrollo.
// No restablece estados por SQL administrativo ni reutiliza una autorización.
type ejecucionesSeleccionReanudablesDesarrollo struct {
	*postgresct.EjecucionesSeleccionLlamamientoPostgreSQL
	autorizador *autorizadorLlamamientoDesarrollo
}

func (e *ejecucionesSeleccionReanudablesDesarrollo) ReanudarPreparacionOrden(
	ctx context.Context, solicitud ports.SolicitudReservaEjecucionSeleccionLlamamiento,
) (ports.EstadoEjecucionSeleccionLlamamiento, error) {
	if ctx == nil || e == nil || e.EjecucionesSeleccionLlamamientoPostgreSQL == nil || e.autorizador == nil {
		return ports.EstadoEjecucionSeleccionLlamamiento{}, ports.ErrAutorizacionDenegada
	}
	p, presente := ctx.Value(clavePreparacionLlamamientoDesarrollo{}).(preparacionLlamamientoDesarrollo)
	if !presente || p.expediente.VersionActual != 6 || p.clave != solicitud.ClaveIdempotencia ||
		p.expediente.Fiscalizado.Referencia != solicitud.ExpedienteRef ||
		p.expediente.Fiscalizado.OrganizacionRef != solicitud.OrganizacionRef ||
		p.necesidad != solicitud.Necesidad.Referencia {
		return ports.EstadoEjecucionSeleccionLlamamiento{}, ports.ErrAutorizacionDenegada
	}
	recurso, err := ports.NuevoRecursoReanudacionSeleccionLlamamiento(solicitud)
	if err != nil {
		return ports.EstadoEjecucionSeleccionLlamamiento{}, err
	}
	ctx = context.WithValue(ctx, claveReanudacionSeleccionDesarrollo{}, solicitud)
	material, err := e.autorizador.AutorizarOperacion(ctx, ports.AccionReanudacionSeleccionLlamamiento, recurso)
	if err != nil {
		return ports.EstadoEjecucionSeleccionLlamamiento{}, err
	}
	return e.EjecucionesSeleccionLlamamientoPostgreSQL.ReanudarPreparacionOrden(ctx, solicitud, material)
}
