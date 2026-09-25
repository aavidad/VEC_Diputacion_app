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
)

// Notificaciones de la persona a RRHH sobre cronos_v1 000010. Cada llamada
// obtiene una decisión V3 nueva de su audiencia (AD3-58) que la función
// durable consume, audita y usa para fijar la RLS en la misma transacción.
const (
	consultaNotificacionesPropias = `SELECT vec_cronos_v1.consultar_notificaciones_propio_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`
	consultaRegistrarNotificacion = `SELECT vec_cronos_v1.registrar_notificacion_propia_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`
	consultaBandejaNotificaciones = `SELECT vec_cronos_v1.consultar_bandeja_notificaciones_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`
	consultaAtenderNotificacion   = `SELECT vec_cronos_v1.atender_notificacion_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`
)

// errorRegistroNotificacion: PC011 en el registro es un tipo ya no vigente.
func errorRegistroNotificacion(ctx context.Context, err error) error {
	var pg *pgconn.PgError
	if (ctx == nil || ctx.Err() == nil) && errors.As(err, &pg) && pg.Code == "PC011" {
		return ports.ErrTipoNotificacionNoVigente
	}
	return errorSolicitud(ctx, err, ports.ErrClaveOperacionEnConflicto)
}

// ---- La persona ----

type RepositorioNotificacionesPropias struct{ db iniciadorMarcaje }

func NuevoRepositorioNotificacionesPropias(pool *pgxpool.Pool) (*RepositorioNotificacionesPropias, error) {
	if pool == nil {
		return nil, ports.ErrDependenciaNoDisponible
	}
	return &RepositorioNotificacionesPropias{db: pool}, nil
}

type notificacionPropiaSQL struct {
	NotificacionRef string     `json:"notificacion_ref"`
	TipoRef         string     `json:"tipo_ref"`
	TipoNombre      string     `json:"tipo_nombre"`
	FechaReferida   string     `json:"fecha_referida"`
	Texto           string     `json:"texto"`
	AdjuntoRef      *string    `json:"adjunto_ref"`
	AdjuntoSHA256   *string    `json:"adjunto_sha256"`
	RegistradaEn    *time.Time `json:"registrada_en"`
	Estado          string     `json:"estado"`
	AtendidaEn      *time.Time `json:"atendida_en"`
}

func instanteOpcional(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	u := t.UTC()
	return &u
}

func (r *RepositorioNotificacionesPropias) ConsultarPropias(ctx context.Context, orden ports.OrdenNotificacionesPropias, m domain.MaterialConsultaNotificaciones) (ports.ConsultaNotificacionesPropias, error) {
	if r == nil || r.db == nil || ctx == nil {
		return ports.ConsultaNotificacionesPropias{}, ports.ErrDependenciaNoDisponible
	}
	actor, err := orden.ContextoActor()
	proveedor := orden.ProveedorMaterial()
	if err != nil || proveedor == nil || !mismoActor(actor, m.ActorRef, m.PerfilRef, m.EmpleadoRef) {
		return ports.ConsultaNotificacionesPropias{}, ports.ErrDependenciaNoDisponible
	}
	canonico, err := m.Canonico()
	if err != nil {
		return ports.ConsultaNotificacionesPropias{}, ports.ErrDependenciaNoDisponible
	}
	recurso, err := application.RecursoConsultaNotificacionesPropias(m)
	if err != nil {
		return ports.ConsultaNotificacionesPropias{}, ports.ErrDependenciaNoDisponible
	}
	v3, err := proveedor.ProveerMaterialConsultaNotificacionesPropias(ctx, m)
	if err != nil {
		return ports.ConsultaNotificacionesPropias{}, errorProveedorV3(ctx, err)
	}
	if !resumenV3Ligado(v3, application.AudienciaConsultaNotificacionesPropias, application.AccionConsultarNotificacionesPropias, recurso) {
		return ports.ConsultaNotificacionesPropias{}, ports.ErrDependenciaNoDisponible
	}
	bruto, err := ejecutarFuncionV3(ctx, r.db, consultaNotificacionesPropias, canonico, v3, errorResolucion)
	if err != nil {
		return ports.ConsultaNotificacionesPropias{}, err
	}
	defer clear(bruto)
	var sql struct {
		EmpleadoRef    string                   `json:"empleado_ref"`
		Tipos          []ports.TipoNotificacion `json:"tipos"`
		Notificaciones []notificacionPropiaSQL  `json:"notificaciones"`
	}
	if decodificarEstricto(bruto, &sql) != nil || sql.Tipos == nil || sql.Notificaciones == nil || sql.EmpleadoRef != m.EmpleadoRef {
		return ports.ConsultaNotificacionesPropias{}, ports.ErrDependenciaNoDisponible
	}
	c := ports.ConsultaNotificacionesPropias{Tipos: sql.Tipos, Notificaciones: make([]ports.NotificacionPropia, 0, len(sql.Notificaciones))}
	for _, n := range sql.Notificaciones {
		if n.RegistradaEn == nil {
			return ports.ConsultaNotificacionesPropias{}, ports.ErrDependenciaNoDisponible
		}
		c.Notificaciones = append(c.Notificaciones, ports.NotificacionPropia{NotificacionRef: n.NotificacionRef, TipoRef: n.TipoRef, TipoNombre: n.TipoNombre,
			FechaReferida: n.FechaReferida, Texto: n.Texto, AdjuntoRef: textoOpcional(n.AdjuntoRef), AdjuntoSHA256: textoOpcional(n.AdjuntoSHA256),
			RegistradaEnUTC: n.RegistradaEn.UTC(), Estado: n.Estado, AtendidaEnUTC: instanteOpcional(n.AtendidaEn)})
	}
	return c, nil
}

func (r *RepositorioNotificacionesPropias) RegistrarNotificacion(ctx context.Context, orden ports.OrdenNotificacionesPropias, m domain.MaterialRegistroNotificacion) (ports.ReciboNotificacion, error) {
	if r == nil || r.db == nil || ctx == nil {
		return ports.ReciboNotificacion{}, ports.ErrDependenciaNoDisponible
	}
	actor, err := orden.ContextoActor()
	proveedor := orden.ProveedorMaterial()
	if err != nil || proveedor == nil || !mismoActor(actor, m.ActorRef, m.PerfilRef, m.EmpleadoRef) {
		return ports.ReciboNotificacion{}, ports.ErrDependenciaNoDisponible
	}
	canonico, err := m.Canonico()
	if err != nil {
		return ports.ReciboNotificacion{}, ports.ErrSolicitudCronosInvalida
	}
	recurso, err := application.RecursoRegistroNotificacion(m)
	if err != nil {
		return ports.ReciboNotificacion{}, ports.ErrDependenciaNoDisponible
	}
	v3, err := proveedor.ProveerMaterialRegistroNotificacion(ctx, m)
	if err != nil {
		return ports.ReciboNotificacion{}, errorProveedorV3(ctx, err)
	}
	if !resumenV3Ligado(v3, application.AudienciaRegistroNotificacion, application.AccionRegistrarNotificacion, recurso) {
		return ports.ReciboNotificacion{}, ports.ErrDependenciaNoDisponible
	}
	bruto, err := ejecutarFuncionV3(ctx, r.db, consultaRegistrarNotificacion, canonico, v3, errorRegistroNotificacion)
	if err != nil {
		return ports.ReciboNotificacion{}, err
	}
	defer clear(bruto)
	var sql struct {
		ports.ReciboNotificacion
		Replay *bool `json:"replay"`
	}
	if decodificarEstricto(bruto, &sql) != nil || sql.Replay == nil || !referenciaReciboSolicitud.MatchString(sql.ReciboRef) {
		return ports.ReciboNotificacion{}, ports.ErrDependenciaNoDisponible
	}
	recibo := sql.ReciboNotificacion
	recibo.Replay = *sql.Replay
	recibo.InstanteUTC = recibo.InstanteUTC.UTC()
	return recibo, nil
}

// ---- RRHH ----

type RepositorioBandejaNotificaciones struct{ db iniciadorMarcaje }

func NuevoRepositorioBandejaNotificaciones(pool *pgxpool.Pool) (*RepositorioBandejaNotificaciones, error) {
	if pool == nil {
		return nil, ports.ErrDependenciaNoDisponible
	}
	return &RepositorioBandejaNotificaciones{db: pool}, nil
}

type notificacionRecibidaSQL struct {
	NotificacionRef  string     `json:"notificacion_ref"`
	EmpleadoRef      string     `json:"empleado_ref"`
	EmpleadoEtiqueta *string    `json:"empleado_etiqueta"`
	TipoRef          string     `json:"tipo_ref"`
	TipoNombre       string     `json:"tipo_nombre"`
	FechaReferida    string     `json:"fecha_referida"`
	Texto            string     `json:"texto"`
	AdjuntoRef       *string    `json:"adjunto_ref"`
	AdjuntoSHA256    *string    `json:"adjunto_sha256"`
	RegistradaEn     *time.Time `json:"registrada_en"`
	Atendida         *bool      `json:"atendida"`
	AtendidaEn       *time.Time `json:"atendida_en"`
}

func (r *RepositorioBandejaNotificaciones) ConsultarBandeja(ctx context.Context, orden ports.OrdenBandejaNotificaciones, m domain.MaterialConsultaNotificaciones) (ports.BandejaNotificaciones, error) {
	if r == nil || r.db == nil || ctx == nil {
		return ports.BandejaNotificaciones{}, ports.ErrDependenciaNoDisponible
	}
	actor, err := orden.ContextoActor()
	proveedor := orden.ProveedorMaterial()
	if err != nil || proveedor == nil || !mismoActor(actor, m.ActorRef, m.PerfilRef, m.EmpleadoRef) {
		return ports.BandejaNotificaciones{}, ports.ErrDependenciaNoDisponible
	}
	canonico, err := m.Canonico()
	if err != nil {
		return ports.BandejaNotificaciones{}, ports.ErrDependenciaNoDisponible
	}
	recurso, err := application.RecursoBandejaNotificaciones(m)
	if err != nil {
		return ports.BandejaNotificaciones{}, ports.ErrDependenciaNoDisponible
	}
	v3, err := proveedor.ProveerMaterialBandejaNotificaciones(ctx, m)
	if err != nil {
		return ports.BandejaNotificaciones{}, errorProveedorV3(ctx, err)
	}
	if !resumenV3Ligado(v3, application.AudienciaBandejaNotificaciones, application.AccionConsultarBandejaNotif, recurso) {
		return ports.BandejaNotificaciones{}, ports.ErrDependenciaNoDisponible
	}
	bruto, err := ejecutarFuncionV3(ctx, r.db, consultaBandejaNotificaciones, canonico, v3, errorResolucion)
	if err != nil {
		return ports.BandejaNotificaciones{}, err
	}
	defer clear(bruto)
	var sql struct {
		Notificaciones []notificacionRecibidaSQL `json:"notificaciones"`
	}
	if decodificarEstricto(bruto, &sql) != nil || sql.Notificaciones == nil {
		return ports.BandejaNotificaciones{}, ports.ErrDependenciaNoDisponible
	}
	b := ports.BandejaNotificaciones{Notificaciones: make([]ports.NotificacionRecibida, 0, len(sql.Notificaciones))}
	for _, n := range sql.Notificaciones {
		if n.RegistradaEn == nil || n.Atendida == nil {
			return ports.BandejaNotificaciones{}, ports.ErrDependenciaNoDisponible
		}
		b.Notificaciones = append(b.Notificaciones, ports.NotificacionRecibida{NotificacionRef: n.NotificacionRef, EmpleadoRef: n.EmpleadoRef,
			EmpleadoEtiqueta: textoOpcional(n.EmpleadoEtiqueta), TipoRef: n.TipoRef, TipoNombre: n.TipoNombre, FechaReferida: n.FechaReferida,
			Texto: n.Texto, AdjuntoRef: textoOpcional(n.AdjuntoRef), AdjuntoSHA256: textoOpcional(n.AdjuntoSHA256), RegistradaEnUTC: n.RegistradaEn.UTC(),
			Atendida: *n.Atendida, AtendidaEnUTC: instanteOpcional(n.AtendidaEn)})
	}
	return b, nil
}

func (r *RepositorioBandejaNotificaciones) AtenderNotificacion(ctx context.Context, orden ports.OrdenBandejaNotificaciones, m domain.MaterialAtencionNotificacion) (ports.ReciboAtencionNotificacion, error) {
	if r == nil || r.db == nil || ctx == nil {
		return ports.ReciboAtencionNotificacion{}, ports.ErrDependenciaNoDisponible
	}
	actor, err := orden.ContextoActor()
	proveedor := orden.ProveedorMaterial()
	if err != nil || proveedor == nil || !mismoActor(actor, m.ActorRef, m.PerfilRef, m.EmpleadoRef) {
		return ports.ReciboAtencionNotificacion{}, ports.ErrDependenciaNoDisponible
	}
	canonico, err := m.Canonico()
	if err != nil {
		return ports.ReciboAtencionNotificacion{}, ports.ErrSolicitudCronosInvalida
	}
	recurso, err := application.RecursoAtencionNotificacion(m)
	if err != nil {
		return ports.ReciboAtencionNotificacion{}, ports.ErrDependenciaNoDisponible
	}
	v3, err := proveedor.ProveerMaterialAtencionNotificacion(ctx, m)
	if err != nil {
		return ports.ReciboAtencionNotificacion{}, errorProveedorV3(ctx, err)
	}
	if !resumenV3Ligado(v3, application.AudienciaAtencionNotificacion, application.AccionAtenderNotificacion, recurso) {
		return ports.ReciboAtencionNotificacion{}, ports.ErrDependenciaNoDisponible
	}
	bruto, err := ejecutarFuncionV3(ctx, r.db, consultaAtenderNotificacion, canonico, v3, errorResolucion)
	if err != nil {
		return ports.ReciboAtencionNotificacion{}, err
	}
	defer clear(bruto)
	var sql struct {
		ports.ReciboAtencionNotificacion
		Replay *bool `json:"replay"`
	}
	if decodificarEstricto(bruto, &sql) != nil || sql.Replay == nil || !referenciaReciboSolicitud.MatchString(sql.ReciboRef) {
		return ports.ReciboAtencionNotificacion{}, ports.ErrDependenciaNoDisponible
	}
	recibo := sql.ReciboAtencionNotificacion
	recibo.Replay = *sql.Replay
	recibo.InstanteUTC = recibo.InstanteUTC.UTC()
	return recibo, nil
}

var (
	_ ports.RepositorioNotificacionesPropias = (*RepositorioNotificacionesPropias)(nil)
	_ ports.RepositorioBandejaNotificaciones = (*RepositorioBandejaNotificaciones)(nil)
)
