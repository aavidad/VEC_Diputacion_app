package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/cronos/application"
	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

const (
	consultaBandejaCorrecciones = `SELECT vec_cronos_v1.consultar_bandeja_correcciones_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`
	consultaActuacionCorreccion = `SELECT vec_cronos_v1.registrar_actuacion_correccion_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`
	consultaHistoriaCorreccion  = `SELECT vec_cronos_v1.consultar_historia_correccion_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`
)

// RepositorioCorreccionesDecisiones compone la solicitud y recuperación C5a
// con las decisiones C5b. No cambia los métodos ni la historia de C5a.
type RepositorioCorreccionesDecisiones struct {
	*RepositorioCorreccionesPropias
	db iniciadorMarcaje
}

func NuevoRepositorioCorreccionesDecisiones(pool *pgxpool.Pool, zona string) (*RepositorioCorreccionesDecisiones, error) {
	propias, err := NuevoRepositorioCorreccionesPropias(pool, zona)
	if err != nil {
		return nil, err
	}
	return &RepositorioCorreccionesDecisiones{RepositorioCorreccionesPropias: propias, db: pool}, nil
}

func proveedorDecision(orden ports.OrdenConsumoCorreccion, actor vecdomain.ContextoActor, empleado string) (ports.ProveedorMaterialDecisionCorreccion, error) {
	proveedor, ok := orden.ProveedorMaterial().(ports.ProveedorMaterialDecisionCorreccion)
	if !ok || dependenciaPostgresNula(proveedor) || !empleadoDeOrden(actor, empleado) {
		return nil, ports.ErrDependenciaNoDisponible
	}
	return proveedor, nil
}

func (r *RepositorioCorreccionesDecisiones) ConsultarBandejaCorrecciones(ctx context.Context, orden ports.OrdenConsumoCorreccion, m domain.MaterialBandejaCorrecciones) (ports.BandejaCorrecciones, error) {
	if r == nil || r.db == nil || ctx == nil || ctx.Err() != nil {
		return ports.BandejaCorrecciones{}, ports.ErrDependenciaNoDisponible
	}
	actor, err := orden.ContextoActor()
	if err != nil || actor.PersonaRef != m.ActorRef || actor.PerfilActivoRef != m.PerfilRef {
		return ports.BandejaCorrecciones{}, ports.ErrDependenciaNoDisponible
	}
	proveedor, err := proveedorDecision(orden, actor, m.EmpleadoRef)
	if err != nil {
		return ports.BandejaCorrecciones{}, err
	}
	canonico, err := m.Canonico()
	if err != nil {
		return ports.BandejaCorrecciones{}, err
	}
	defer clear(canonico)
	recurso, err := application.RecursoBandejaCorrecciones(m)
	if err != nil {
		return ports.BandejaCorrecciones{}, ports.ErrDependenciaNoDisponible
	}
	v3, err := proveedor.ProveerMaterialBandejaCorrecciones(ctx, m)
	if err != nil {
		return ports.BandejaCorrecciones{}, errorProveedorV3(ctx, err)
	}
	if !resumenV3Ligado(v3, application.AudienciaBandejaCorrecciones, application.AccionBandejaCorrecciones, recurso) {
		return ports.BandejaCorrecciones{}, ports.ErrDependenciaNoDisponible
	}
	bruto, err := ejecutarFuncionV3(ctx, r.db, consultaBandejaCorrecciones, canonico, v3, errorDecisionCorreccion)
	if err != nil {
		return ports.BandejaCorrecciones{}, err
	}
	defer clear(bruto)
	var b ports.BandejaCorrecciones
	if decodificarEstricto(bruto, &b) != nil || b.Paso != m.Paso || b.Pendientes == nil {
		return ports.BandejaCorrecciones{}, ports.ErrDependenciaNoDisponible
	}
	return b, nil
}

func (r *RepositorioCorreccionesDecisiones) RegistrarDecisionCorreccion(ctx context.Context, orden ports.OrdenConsumoCorreccion, m domain.MaterialActuacionCorreccion) (ports.ReciboCorreccion, error) {
	if r == nil || r.db == nil || ctx == nil || ctx.Err() != nil {
		return ports.ReciboCorreccion{}, ports.ErrDependenciaNoDisponible
	}
	actor, err := orden.ContextoActor()
	if err != nil || actor.PersonaRef != m.ActorRef || actor.PerfilActivoRef != m.PerfilRef {
		return ports.ReciboCorreccion{}, ports.ErrDependenciaNoDisponible
	}
	proveedor, err := proveedorDecision(orden, actor, m.EmpleadoRef)
	if err != nil {
		return ports.ReciboCorreccion{}, err
	}
	canonico, err := m.Canonico()
	if err != nil {
		return ports.ReciboCorreccion{}, err
	}
	defer clear(canonico)
	recurso, err := application.RecursoActuacionCorreccion(m)
	if err != nil {
		return ports.ReciboCorreccion{}, ports.ErrDependenciaNoDisponible
	}
	accion, _, audiencia, ok := application.ContratoActuacionCorreccion(m.Paso)
	if !ok {
		return ports.ReciboCorreccion{}, ports.ErrDependenciaNoDisponible
	}
	v3, err := proveedor.ProveerMaterialActuacionCorreccion(ctx, m)
	if err != nil {
		return ports.ReciboCorreccion{}, errorProveedorV3(ctx, err)
	}
	if !resumenV3Ligado(v3, audiencia, accion, recurso) {
		return ports.ReciboCorreccion{}, ports.ErrDependenciaNoDisponible
	}
	bruto, err := ejecutarFuncionV3(ctx, r.db, consultaActuacionCorreccion, canonico, v3, errorDecisionCorreccion)
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
	if decodificarEstricto(bruto, &sql) != nil || sql.Replay == nil || sql.SolicitudRef != m.SolicitudRef ||
		!referenciaReciboSolicitud.MatchString(sql.ReciboRef) || !referenciaActuacionCorreccion.MatchString(sql.ActuacionRef) {
		return ports.ReciboCorreccion{}, ports.ErrDependenciaNoDisponible
	}
	return ports.ReciboCorreccion{SolicitudRef: sql.SolicitudRef, ActuacionRef: sql.ActuacionRef, ReciboRef: sql.ReciboRef,
		Estado: sql.Estado, Version: sql.Version, InstanteUTC: sql.InstanteUTC.UTC(), Replay: *sql.Replay}, nil
}

func (r *RepositorioCorreccionesDecisiones) ConsultarHistoriaCorreccion(ctx context.Context, orden ports.OrdenConsumoCorreccion, m domain.MaterialHistoriaCorreccion) (ports.HistoriaCorreccion, error) {
	if r == nil || r.db == nil || ctx == nil || ctx.Err() != nil {
		return ports.HistoriaCorreccion{}, ports.ErrDependenciaNoDisponible
	}
	actor, err := orden.ContextoActor()
	if err != nil || actor.PersonaRef != m.ActorRef || actor.PerfilActivoRef != m.PerfilRef {
		return ports.HistoriaCorreccion{}, ports.ErrDependenciaNoDisponible
	}
	proveedor, err := proveedorDecision(orden, actor, m.EmpleadoRef)
	if err != nil {
		return ports.HistoriaCorreccion{}, err
	}
	canonico, err := m.Canonico()
	if err != nil {
		return ports.HistoriaCorreccion{}, err
	}
	defer clear(canonico)
	recurso, err := application.RecursoHistoriaCorreccion(m)
	if err != nil {
		return ports.HistoriaCorreccion{}, ports.ErrDependenciaNoDisponible
	}
	accion, _, audiencia, ok := application.ContratoHistoriaCorreccion(m)
	if !ok {
		return ports.HistoriaCorreccion{}, ports.ErrDependenciaNoDisponible
	}
	v3, err := proveedor.ProveerMaterialHistoriaCorreccion(ctx, m)
	if err != nil {
		return ports.HistoriaCorreccion{}, errorProveedorV3(ctx, err)
	}
	if !resumenV3Ligado(v3, audiencia, accion, recurso) {
		return ports.HistoriaCorreccion{}, ports.ErrDependenciaNoDisponible
	}
	bruto, err := ejecutarFuncionV3(ctx, r.db, consultaHistoriaCorreccion, canonico, v3, errorDecisionCorreccion)
	if err != nil {
		return ports.HistoriaCorreccion{}, err
	}
	defer clear(bruto)
	var h ports.HistoriaCorreccion
	if decodificarEstricto(bruto, &h) != nil || h.SolicitudRef != m.SolicitudRef || h.EmpleadoRef == "" || h.Actuaciones == nil || h.Asientos == nil {
		return ports.HistoriaCorreccion{}, ports.ErrDependenciaNoDisponible
	}
	for _, a := range h.Actuaciones {
		if !referenciaReciboSolicitud.MatchString(a.ReciboRef) || a.InstanteUTC.IsZero() || a.InstanteUTC.Nanosecond()%1000 != 0 {
			return ports.HistoriaCorreccion{}, ports.ErrDependenciaNoDisponible
		}
	}
	return h, nil
}

// La interfaz C5 sigue disponible para la solicitud y su recuperación. Los
// pasos con motivo se envían mediante RegistrarDecisionCorreccion.
func (r *RepositorioCorreccionesDecisiones) RegistrarActuacion(context.Context, domain.ActuacionCorreccion, ports.OrdenConsumoCorreccion) (ports.ReciboCorreccion, error) {
	return ports.ReciboCorreccion{}, ports.ErrDependenciaNoDisponible
}

func errorDecisionCorreccion(ctx context.Context, err error) error {
	var pg *pgconn.PgError
	if (ctx == nil || ctx.Err() == nil) && errors.As(err, &pg) {
		switch pg.Code {
		case "PC001":
			return domain.ErrCorreccionInvalida
		case "PC002":
			return ports.ErrCorreccionEnConflicto
		case "PC011":
			return ports.ErrResolucionEstadoCambiado
		case "PC012":
			return ports.ErrResolucionNoCompetente
		case "PC004":
			return ports.ErrCorreccionNoAutorizada
		}
	}
	return errorSeguro(ctx, err)
}

var (
	_ ports.RepositorioCorrecciones         = (*RepositorioCorreccionesDecisiones)(nil)
	_ ports.RepositorioDecisionCorrecciones = (*RepositorioCorreccionesDecisiones)(nil)
)
