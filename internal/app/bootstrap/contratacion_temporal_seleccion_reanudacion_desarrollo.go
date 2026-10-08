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
	return e.reanudar(ctx, solicitud, ports.AccionReanudacionSeleccionLlamamiento)
}

func (e *ejecucionesSeleccionReanudablesDesarrollo) ReanudarSolicitudLlamamiento(
	ctx context.Context, solicitud ports.SolicitudReservaEjecucionSeleccionLlamamiento,
) (ports.EstadoEjecucionSeleccionLlamamiento, error) {
	return e.reanudar(ctx, solicitud, ports.AccionReanudacionSolicitudLlamamiento)
}

func (e *ejecucionesSeleccionReanudablesDesarrollo) reanudar(
	ctx context.Context, solicitud ports.SolicitudReservaEjecucionSeleccionLlamamiento, accion string,
) (ports.EstadoEjecucionSeleccionLlamamiento, error) {
	if ctx == nil || e == nil || e.EjecucionesSeleccionLlamamientoPostgreSQL == nil || e.autorizador == nil {
		return ports.EstadoEjecucionSeleccionLlamamiento{}, ports.ErrAutorizacionDenegada
	}
	p, presente := ctx.Value(clavePreparacionLlamamientoDesarrollo{}).(preparacionLlamamientoDesarrollo)
	if !presente || p.expediente.VersionActual != solicitud.VersionExpediente ||
		p.expediente.Fiscalizado.Version != solicitud.VersionExpediente || p.clave != solicitud.ClaveIdempotencia ||
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
	material, err := e.autorizador.AutorizarOperacion(ctx, accion, recurso)
	if err != nil {
		return ports.EstadoEjecucionSeleccionLlamamiento{}, err
	}
	if accion == ports.AccionReanudacionSeleccionLlamamiento {
		return e.EjecucionesSeleccionLlamamientoPostgreSQL.ReanudarPreparacionOrden(ctx, solicitud, material)
	}
	if accion == ports.AccionReanudacionSolicitudLlamamiento {
		return e.EjecucionesSeleccionLlamamientoPostgreSQL.ReanudarSolicitudLlamamiento(ctx, solicitud, material)
	}
	return ports.EstadoEjecucionSeleccionLlamamiento{}, ports.ErrAutorizacionDenegada
}
