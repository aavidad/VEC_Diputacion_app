package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const funcionReanudarPreparacionOrdenSeleccion = "vec_contratacion_temporal.reanudar_preparacion_orden_seleccion_v1"
const funcionReanudarSolicitudLlamamiento = "vec_contratacion_temporal.reanudar_solicitud_llamamiento_v1"

// ReanudarPreparacionOrden conserva la intención y las ventanas. La función
// nominal consume autorización fresca, cambia el propietario y añade historia
// y outbox en esta misma transacción. Nunca reintenta ni recupera un token viejo.
func (e *EjecucionesSeleccionLlamamientoPostgreSQL) ReanudarPreparacionOrden(
	ctx context.Context,
	solicitud ports.SolicitudReservaEjecucionSeleccionLlamamiento,
	material puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3,
) (ports.EstadoEjecucionSeleccionLlamamiento, error) {
	funcion := funcionReanudarPreparacionOrdenSeleccion
	if solicitud.VersionExpediente != 6 {
		funcion = "vec_contratacion_temporal.reanudar_preparacion_orden_seleccion_v2"
	}
	return e.reanudarEfecto(ctx, solicitud, material, ports.AccionReanudacionSeleccionLlamamiento,
		ports.EfectoPrepararOrdenSeleccionLlamamiento, funcion)
}

// ReanudarSolicitudLlamamiento renueva únicamente la ventana ya abierta de
// solicitud. La decisión nominal es distinta de la usada para la orden.
func (e *EjecucionesSeleccionLlamamientoPostgreSQL) ReanudarSolicitudLlamamiento(
	ctx context.Context,
	solicitud ports.SolicitudReservaEjecucionSeleccionLlamamiento,
	material puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3,
) (ports.EstadoEjecucionSeleccionLlamamiento, error) {
	return e.reanudarEfecto(ctx, solicitud, material, ports.AccionReanudacionSolicitudLlamamiento,
		ports.EfectoSolicitarSeleccionLlamamiento, funcionReanudarSolicitudLlamamiento)
}

func (e *EjecucionesSeleccionLlamamientoPostgreSQL) reanudarEfecto(
	ctx context.Context,
	solicitud ports.SolicitudReservaEjecucionSeleccionLlamamiento,
	material puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3,
	accion string,
	efecto ports.EfectoSeleccionLlamamiento,
	funcion string,
) (ports.EstadoEjecucionSeleccionLlamamiento, error) {
	vacio := ports.EstadoEjecucionSeleccionLlamamiento{}
	if !e.valido(ctx) {
		return vacio, errEjecucionesSeleccionLlamamientoPostgreSQL
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	recurso, err := ports.NuevoRecursoReanudacionSeleccionLlamamiento(solicitud)
	contenido, errContenido := codificarSolicitudSeleccionO6(solicitud)
	if err != nil || errContenido != nil || material.ValidarEstructura() != nil {
		return vacio, errEjecucionesSeleccionLlamamientoPostgreSQL
	}
	h, err := recurso.HuellaContextoAutorizacionSHA256()
	c := material.ResumenCapacidad()
	if err != nil || c.Operacion() != accion ||
		c.AudienciaConsumo() != ports.AudienciaReanudacionSeleccionLlamamiento ||
		c.EfectoRef() != solicitud.ExpedienteRef || c.EfectoHuellaSHA256() != h {
		return vacio, errEjecucionesSeleccionLlamamientoPostgreSQL
	}
	tx, err := e.iniciar(ctx, pgx.ReadWrite)
	if err != nil {
		return vacio, errorEjecucionesSeleccionO6(ctx)
	}
	defer revertirTransaccion(tx)
	var fila filaEjecucionSeleccionO6
	err = tx.QueryRow(ctx, `SELECT situacion, solicitud_json, reserva_ref,
		efecto, recibo_json, artefacto_json FROM `+funcion+`(
		$1::text,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`,
		string(contenido), material.CapacidadCanonica(), material.DecisionCanonica(),
		material.MotivoCanonico(), material.ContextoActorCanonico(), material.PersonaVersion(),
		material.PerfilVersion(), material.PayloadVECAD3(), material.SobreCOSESign1(),
		material.EvidenciaVerificacion(), material.RaizPublicaSPKI(),
	).Scan(&fila.Situacion, &fila.SolicitudJSON, &fila.ReservaRef,
		&fila.Efecto, &fila.ReciboJSON, &fila.ArtefactoJSON)
	if err != nil {
		return vacio, errorReanudacionSeleccionO6(ctx, err)
	}
	estado, err := estadoReanudacionSeleccionDesdeFilaParaEfecto(fila, solicitud, efecto)
	if err != nil {
		return vacio, err
	}
	// Validar antes del COMMIT: ninguna respuesta divergente confirma efectos.
	if err := tx.Commit(ctx); err != nil {
		return vacio, errorEjecucionesSeleccionO6(ctx)
	}
	return estado, nil
}

func errorReanudacionSeleccionO6(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "42501" {
		return ports.ErrAutorizacionDenegada
	}
	return errEjecucionesSeleccionLlamamientoPostgreSQL
}

func estadoReanudacionSeleccionDesdeFila(f filaEjecucionSeleccionO6, s ports.SolicitudReservaEjecucionSeleccionLlamamiento) (ports.EstadoEjecucionSeleccionLlamamiento, error) {
	return estadoReanudacionSeleccionDesdeFilaParaEfecto(f, s, ports.EfectoPrepararOrdenSeleccionLlamamiento)
}

func estadoReanudacionSeleccionDesdeFilaParaEfecto(f filaEjecucionSeleccionO6, s ports.SolicitudReservaEjecucionSeleccionLlamamiento, efecto ports.EfectoSeleccionLlamamiento) (ports.EstadoEjecucionSeleccionLlamamiento, error) {
	var dto solicitudEjecucionSeleccionO6
	if s.Validar() != nil || s.VersionExpediente < 6 ||
		f.Situacion != string(ports.EjecucionSeleccionLlamamientoPropietaria) ||
		f.Efecto != string(efecto) ||
		!referenciaReservaSeleccionO6Valida(f.ReservaRef) ||
		f.ReciboJSON != "" || f.ArtefactoJSON != "" || len(f.SolicitudJSON) > maximoCargaSeleccionO6 ||
		decodificarJSONEstricto([]byte(f.SolicitudJSON), &dto) != nil ||
		!dto.valida() || dto.puertos() != s {
		return ports.EstadoEjecucionSeleccionLlamamiento{}, errEjecucionesSeleccionLlamamientoPostgreSQL
	}
	return ports.EstadoEjecucionSeleccionLlamamiento{
		Solicitud: s, Situacion: ports.EjecucionSeleccionLlamamientoPropietaria,
		ReservaRef: f.ReservaRef, EfectoPosible: efecto,
	}, nil
}
