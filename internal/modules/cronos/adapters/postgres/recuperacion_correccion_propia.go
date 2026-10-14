package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/cronos/application"
	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

const consultaRecuperarCorreccionPropia = `SELECT vec_cronos_v1.recuperar_recibo_correccion_propia_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`

func (r *RepositorioCorreccionesPropias) RecuperarRecibo(ctx context.Context, clave ports.ClaveRecuperacionCorreccion, orden ports.OrdenConsumoCorreccion) (ports.ReciboCorreccion, error) {
	if r == nil || r.db == nil || ctx == nil || ctx.Err() != nil {
		return ports.ReciboCorreccion{}, ports.ErrDependenciaNoDisponible
	}
	actor, err := orden.ContextoActor()
	proveedor, ok := orden.ProveedorMaterial().(ports.ProveedorMaterialRecuperacionCorreccionPropia)
	if err != nil || !ok || dependenciaPostgresNula(proveedor) || clave.Paso != domain.PasoSolicitudCorreccion ||
		clave.SolicitudRef != "correccion:cronos:"+clave.ClaveOperacion {
		return ports.ReciboCorreccion{}, ports.ErrDependenciaNoDisponible
	}
	empleados, err := actor.Referencias(vecdomain.TipoReferenciaContextoActorEmpleado)
	if err != nil || len(empleados) != 1 || !empleadoDeOrden(actor, empleados[0]) {
		return ports.ReciboCorreccion{}, ports.ErrCorreccionNoAutorizada
	}
	vinculo, existe := orden.VinculoPropioHistoricoCRN11()
	if !existe || vinculo.PersonaRef != actor.PersonaRef || vinculo.EmpleadoRef != empleados[0] ||
		vinculo.Version == 0 || vinculo.VinculoRef == "" || vinculo.FuenteRef == "" ||
		vinculo.Evidencia.ReciboRef == "" || vinculo.Evidencia.DecisionRef == "" ||
		vinculo.Evidencia.AuditoriaRef == "" || vinculo.Evidencia.ConsultadaEn.IsZero() {
		return ports.ReciboCorreccion{}, ports.ErrDependenciaNoDisponible
	}
	m := domain.MaterialRecuperacionCorreccionPropia{
		ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef,
		EmpleadoRef: empleados[0], SolicitudRef: clave.SolicitudRef, ClaveOperacion: clave.ClaveOperacion,
	}
	canonico, err := m.Canonico()
	if err != nil {
		return ports.ReciboCorreccion{}, ports.ErrDependenciaNoDisponible
	}
	defer clear(canonico)
	recurso, err := application.RecursoRecuperarReciboCorreccionPropia(m)
	if err != nil {
		return ports.ReciboCorreccion{}, ports.ErrDependenciaNoDisponible
	}
	v3, err := proveedor.ProveerMaterialRecuperacionCorreccionPropia(ctx, m)
	if err != nil {
		return ports.ReciboCorreccion{}, errorProveedorV3(ctx, err)
	}
	if !resumenV3Ligado(v3, application.AudienciaRecuperarReciboCorreccionPropia, application.AccionRecuperarReciboCorreccion, recurso) {
		return ports.ReciboCorreccion{}, ports.ErrDependenciaNoDisponible
	}
	bruto, err := ejecutarFuncionV3(ctx, r.db, consultaRecuperarCorreccionPropia, canonico, v3, errorRecuperacionCorreccion)
	if err != nil {
		return ports.ReciboCorreccion{}, err
	}
	defer clear(bruto)
	var sql struct {
		SolicitudRef string                  `json:"solicitud_ref"`
		ActuacionRef string                  `json:"actuacion_ref"`
		ReciboRef    string                  `json:"recibo_ref"`
		Estado       domain.EstadoCorreccion `json:"estado"`
		Version      uint64                  `json:"version"`
		InstanteUTC  time.Time               `json:"instante_utc"`
		Replay       *bool                   `json:"replay"`
	}
	if decodificarEstricto(bruto, &sql) != nil || sql.Replay == nil || !*sql.Replay ||
		sql.SolicitudRef != clave.SolicitudRef || sql.Estado != domain.CorreccionPendienteResponsable ||
		sql.Version != 1 || !referenciaReciboSolicitud.MatchString(sql.ReciboRef) ||
		!referenciaActuacionCorreccion.MatchString(sql.ActuacionRef) || sql.InstanteUTC.IsZero() ||
		sql.InstanteUTC.Nanosecond()%1000 != 0 {
		return ports.ReciboCorreccion{}, ports.ErrDependenciaNoDisponible
	}
	return ports.ReciboCorreccion{SolicitudRef: sql.SolicitudRef, ActuacionRef: sql.ActuacionRef,
		ReciboRef: sql.ReciboRef, Estado: sql.Estado, Version: sql.Version,
		InstanteUTC: sql.InstanteUTC.UTC(), Replay: true}, nil
}

func errorRecuperacionCorreccion(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "PC004" {
		return ports.ErrCorreccionNoAutorizada
	}
	return ports.ErrDependenciaNoDisponible
}
