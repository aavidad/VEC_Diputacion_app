package postgres

import (
	"context"
	"errors"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/cronos/application"
	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

// Resolución de permisos y avisos sobre cronos_v1 000009. Cada llamada
// obtiene una decisión V3 nueva de su audiencia (AD3-57) que la función
// durable consume, audita y usa para fijar la RLS en la misma transacción.
const (
	consultaBandejaPermisos = `SELECT vec_cronos_v1.consultar_bandeja_permisos_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`
	consultaResolverPermiso = `SELECT vec_cronos_v1.resolver_permiso_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`
	consultaAvisosPropios   = `SELECT vec_cronos_v1.consultar_avisos_propio_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`
	consultaArchivarAviso   = `SELECT vec_cronos_v1.archivar_aviso_propio_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`
)

var referenciaAviso = regexp.MustCompile(`^aviso:cronos:[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

// errorResolucion traduce los rechazos nominales de 000009; el resto sigue
// la política común de la solicitud y, después, la de dependencia.
func errorResolucion(ctx context.Context, err error) error {
	var pg *pgconn.PgError
	if (ctx == nil || ctx.Err() == nil) && errors.As(err, &pg) {
		switch pg.Code {
		case "PC011":
			return ports.ErrResolucionEstadoCambiado
		case "PC012":
			return ports.ErrResolucionNoCompetente
		}
	}
	return errorSolicitud(ctx, err, ports.ErrClaveOperacionEnConflicto)
}

func mismoActor(actor vecdomain.ContextoActor, actorRef, perfilRef, empleado string) bool {
	return empleadoDeOrden(actor, empleado) && actor.PersonaRef == actorRef && actor.PerfilActivoRef == perfilRef
}

func textoOpcional(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// ---- Quien resuelve ----

type RepositorioResolucionPermisos struct{ db iniciadorMarcaje }

func NuevoRepositorioResolucionPermisos(pool *pgxpool.Pool) (*RepositorioResolucionPermisos, error) {
	if pool == nil {
		return nil, ports.ErrDependenciaNoDisponible
	}
	return &RepositorioResolucionPermisos{db: pool}, nil
}

type pendienteSQL struct {
	SolicitudRef        string     `json:"solicitud_ref"`
	EmpleadoRef         string     `json:"empleado_ref"`
	EmpleadoEtiqueta    *string    `json:"empleado_etiqueta"`
	PermisoRef          string     `json:"permiso_ref"`
	Nombre              string     `json:"nombre"`
	Circuito            string     `json:"circuito"`
	JustificanteExigido bool       `json:"justificante_exigido"`
	Desde               string     `json:"desde"`
	Hasta               string     `json:"hasta"`
	HoraInicio          *string    `json:"hora_inicio"`
	HoraFin             *string    `json:"hora_fin"`
	Cantidad            int64      `json:"cantidad"`
	Unidad              string     `json:"unidad"`
	Estado              string     `json:"estado"`
	Version             int        `json:"version"`
	SolicitadaEn        *time.Time `json:"solicitada_en"`
}

func (r *RepositorioResolucionPermisos) ConsultarBandeja(ctx context.Context, orden ports.OrdenResolucionPermisos, m domain.MaterialBandejaPermisos) (ports.BandejaPermisos, error) {
	if r == nil || r.db == nil || ctx == nil {
		return ports.BandejaPermisos{}, ports.ErrDependenciaNoDisponible
	}
	actor, err := orden.ContextoActor()
	proveedor := orden.ProveedorMaterial()
	if err != nil || proveedor == nil || !mismoActor(actor, m.ActorRef, m.PerfilRef, m.EmpleadoRef) {
		return ports.BandejaPermisos{}, ports.ErrDependenciaNoDisponible
	}
	canonico, err := m.Canonico()
	if err != nil {
		return ports.BandejaPermisos{}, ports.ErrSolicitudCronosInvalida
	}
	recurso, err := application.RecursoBandejaPermisos(m)
	if err != nil {
		return ports.BandejaPermisos{}, ports.ErrDependenciaNoDisponible
	}
	v3, err := proveedor.ProveerMaterialBandejaPermisos(ctx, m)
	if err != nil {
		return ports.BandejaPermisos{}, errorProveedorV3(ctx, err)
	}
	if !resumenV3Ligado(v3, application.AudienciaBandejaPermisos, application.AccionConsultarBandeja, recurso) {
		return ports.BandejaPermisos{}, ports.ErrDependenciaNoDisponible
	}
	bruto, err := ejecutarFuncionV3(ctx, r.db, consultaBandejaPermisos, canonico, v3, errorResolucion)
	if err != nil {
		return ports.BandejaPermisos{}, err
	}
	defer clear(bruto)
	var sql struct {
		Paso       string         `json:"paso"`
		Pendientes []pendienteSQL `json:"pendientes"`
	}
	if decodificarEstricto(bruto, &sql) != nil || sql.Pendientes == nil {
		return ports.BandejaPermisos{}, ports.ErrDependenciaNoDisponible
	}
	b := ports.BandejaPermisos{Paso: domain.PasoPermiso(sql.Paso), Pendientes: make([]ports.SolicitudPendiente, 0, len(sql.Pendientes))}
	for _, p := range sql.Pendientes {
		if p.SolicitadaEn == nil {
			return ports.BandejaPermisos{}, ports.ErrDependenciaNoDisponible
		}
		b.Pendientes = append(b.Pendientes, ports.SolicitudPendiente{
			SolicitudRef: p.SolicitudRef, EmpleadoRef: p.EmpleadoRef, EmpleadoEtiqueta: textoOpcional(p.EmpleadoEtiqueta),
			PermisoRef: p.PermisoRef, Nombre: p.Nombre, Circuito: domain.CircuitoPermiso(p.Circuito), JustificanteExigido: p.JustificanteExigido,
			Desde: p.Desde, Hasta: p.Hasta, HoraInicio: textoOpcional(p.HoraInicio), HoraFin: textoOpcional(p.HoraFin),
			Cantidad: p.Cantidad, Unidad: domain.LeaveUnit(p.Unidad), Estado: domain.EstadoSolicitudPermiso(p.Estado),
			Version: p.Version, SolicitadaEnUTC: p.SolicitadaEn.UTC(),
		})
	}
	return b, nil
}

func (r *RepositorioResolucionPermisos) ResolverPermiso(ctx context.Context, orden ports.OrdenResolucionPermisos, m domain.MaterialResolucionPermiso) (ports.ReciboResolucionPermiso, error) {
	if r == nil || r.db == nil || ctx == nil {
		return ports.ReciboResolucionPermiso{}, ports.ErrDependenciaNoDisponible
	}
	actor, err := orden.ContextoActor()
	proveedor := orden.ProveedorMaterial()
	if err != nil || proveedor == nil || !mismoActor(actor, m.ActorRef, m.PerfilRef, m.EmpleadoRef) {
		return ports.ReciboResolucionPermiso{}, ports.ErrDependenciaNoDisponible
	}
	canonico, err := m.Canonico()
	if err != nil {
		return ports.ReciboResolucionPermiso{}, ports.ErrSolicitudCronosInvalida
	}
	recurso, err := application.RecursoResolucionPermiso(m)
	if err != nil {
		return ports.ReciboResolucionPermiso{}, ports.ErrDependenciaNoDisponible
	}
	v3, err := proveedor.ProveerMaterialResolucionPermiso(ctx, m)
	if err != nil {
		return ports.ReciboResolucionPermiso{}, errorProveedorV3(ctx, err)
	}
	if !resumenV3Ligado(v3, application.AudienciaResolucionPermiso, application.AccionResolverPermiso, recurso) {
		return ports.ReciboResolucionPermiso{}, ports.ErrDependenciaNoDisponible
	}
	bruto, err := ejecutarFuncionV3(ctx, r.db, consultaResolverPermiso, canonico, v3, errorResolucion)
	if err != nil {
		return ports.ReciboResolucionPermiso{}, err
	}
	defer clear(bruto)
	var sql struct {
		ports.ReciboResolucionPermiso
		Replay *bool `json:"replay"`
	}
	if decodificarEstricto(bruto, &sql) != nil || sql.Replay == nil || !referenciaReciboSolicitud.MatchString(sql.ReciboRef) {
		return ports.ReciboResolucionPermiso{}, ports.ErrDependenciaNoDisponible
	}
	recibo := sql.ReciboResolucionPermiso
	recibo.Replay = *sql.Replay
	recibo.InstanteUTC = recibo.InstanteUTC.UTC()
	return recibo, nil
}

// ---- La persona: avisos ----

type RepositorioAvisosPropios struct{ db iniciadorMarcaje }

func NuevoRepositorioAvisosPropios(pool *pgxpool.Pool) (*RepositorioAvisosPropios, error) {
	if pool == nil {
		return nil, ports.ErrDependenciaNoDisponible
	}
	return &RepositorioAvisosPropios{db: pool}, nil
}

type avisoSQL struct {
	AvisoRef     string     `json:"aviso_ref"`
	SolicitudRef string     `json:"solicitud_ref"`
	Estado       string     `json:"estado"`
	Motivo       *string    `json:"motivo"`
	ResueltoEn   *time.Time `json:"resuelto_en"`
	PermisoRef   string     `json:"permiso_ref"`
	Nombre       string     `json:"nombre"`
	Desde        string     `json:"desde"`
	Hasta        string     `json:"hasta"`
	HoraInicio   *string    `json:"hora_inicio"`
	HoraFin      *string    `json:"hora_fin"`
	Cantidad     int64      `json:"cantidad"`
	Unidad       string     `json:"unidad"`
	Archivado    *bool      `json:"archivado"`
	ArchivadoEn  *time.Time `json:"archivado_en"`
}

func (r *RepositorioAvisosPropios) ConsultarAvisos(ctx context.Context, orden ports.OrdenAvisosPropios, m domain.MaterialConsultaAvisosPropios) (ports.ConsultaAvisosPropios, error) {
	if r == nil || r.db == nil || ctx == nil {
		return ports.ConsultaAvisosPropios{}, ports.ErrDependenciaNoDisponible
	}
	actor, err := orden.ContextoActor()
	proveedor := orden.ProveedorMaterial()
	if err != nil || proveedor == nil || !mismoActor(actor, m.ActorRef, m.PerfilRef, m.EmpleadoRef) {
		return ports.ConsultaAvisosPropios{}, ports.ErrDependenciaNoDisponible
	}
	canonico, err := m.Canonico()
	if err != nil {
		return ports.ConsultaAvisosPropios{}, ports.ErrDependenciaNoDisponible
	}
	recurso, err := application.RecursoConsultaAvisosPropios(m)
	if err != nil {
		return ports.ConsultaAvisosPropios{}, ports.ErrDependenciaNoDisponible
	}
	v3, err := proveedor.ProveerMaterialConsultaAvisosPropios(ctx, m)
	if err != nil {
		return ports.ConsultaAvisosPropios{}, errorProveedorV3(ctx, err)
	}
	if !resumenV3Ligado(v3, application.AudienciaConsultaAvisosPropios, application.AccionConsultarAvisosPropios, recurso) {
		return ports.ConsultaAvisosPropios{}, ports.ErrDependenciaNoDisponible
	}
	bruto, err := ejecutarFuncionV3(ctx, r.db, consultaAvisosPropios, canonico, v3, errorResolucion)
	if err != nil {
		return ports.ConsultaAvisosPropios{}, err
	}
	defer clear(bruto)
	var sql struct {
		EmpleadoRef string     `json:"empleado_ref"`
		Avisos      []avisoSQL `json:"avisos"`
	}
	if decodificarEstricto(bruto, &sql) != nil || sql.Avisos == nil || sql.EmpleadoRef != m.EmpleadoRef {
		return ports.ConsultaAvisosPropios{}, ports.ErrDependenciaNoDisponible
	}
	c := ports.ConsultaAvisosPropios{Avisos: make([]ports.AvisoPropio, 0, len(sql.Avisos))}
	for _, a := range sql.Avisos {
		if a.ResueltoEn == nil || a.Archivado == nil || !referenciaAviso.MatchString(a.AvisoRef) {
			return ports.ConsultaAvisosPropios{}, ports.ErrDependenciaNoDisponible
		}
		aviso := ports.AvisoPropio{AvisoRef: a.AvisoRef, SolicitudRef: a.SolicitudRef, Estado: domain.EstadoSolicitudPermiso(a.Estado),
			Motivo: textoOpcional(a.Motivo), ResueltoEnUTC: a.ResueltoEn.UTC(), PermisoRef: a.PermisoRef, Nombre: a.Nombre,
			Desde: a.Desde, Hasta: a.Hasta, HoraInicio: textoOpcional(a.HoraInicio), HoraFin: textoOpcional(a.HoraFin),
			Cantidad: a.Cantidad, Unidad: domain.LeaveUnit(a.Unidad), Archivado: *a.Archivado}
		if a.ArchivadoEn != nil {
			en := a.ArchivadoEn.UTC()
			aviso.ArchivadoEnUTC = &en
		}
		c.Avisos = append(c.Avisos, aviso)
	}
	return c, nil
}

func (r *RepositorioAvisosPropios) ArchivarAviso(ctx context.Context, orden ports.OrdenAvisosPropios, m domain.MaterialArchivoAvisoPropio) (ports.ReciboArchivoAviso, error) {
	if r == nil || r.db == nil || ctx == nil {
		return ports.ReciboArchivoAviso{}, ports.ErrDependenciaNoDisponible
	}
	actor, err := orden.ContextoActor()
	proveedor := orden.ProveedorMaterial()
	if err != nil || proveedor == nil || !mismoActor(actor, m.ActorRef, m.PerfilRef, m.EmpleadoRef) {
		return ports.ReciboArchivoAviso{}, ports.ErrDependenciaNoDisponible
	}
	canonico, err := m.Canonico()
	if err != nil {
		return ports.ReciboArchivoAviso{}, ports.ErrSolicitudCronosInvalida
	}
	recurso, err := application.RecursoArchivoAvisoPropio(m)
	if err != nil {
		return ports.ReciboArchivoAviso{}, ports.ErrDependenciaNoDisponible
	}
	v3, err := proveedor.ProveerMaterialArchivoAvisoPropio(ctx, m)
	if err != nil {
		return ports.ReciboArchivoAviso{}, errorProveedorV3(ctx, err)
	}
	if !resumenV3Ligado(v3, application.AudienciaArchivoAvisoPropio, application.AccionArchivarAvisoPropio, recurso) {
		return ports.ReciboArchivoAviso{}, ports.ErrDependenciaNoDisponible
	}
	bruto, err := ejecutarFuncionV3(ctx, r.db, consultaArchivarAviso, canonico, v3, errorResolucion)
	if err != nil {
		return ports.ReciboArchivoAviso{}, err
	}
	defer clear(bruto)
	var sql struct {
		ports.ReciboArchivoAviso
		Replay *bool `json:"replay"`
	}
	if decodificarEstricto(bruto, &sql) != nil || sql.Replay == nil || !referenciaReciboSolicitud.MatchString(sql.ReciboRef) {
		return ports.ReciboArchivoAviso{}, ports.ErrDependenciaNoDisponible
	}
	recibo := sql.ReciboArchivoAviso
	recibo.Replay = *sql.Replay
	recibo.InstanteUTC = recibo.InstanteUTC.UTC()
	return recibo, nil
}

var (
	_ ports.RepositorioResolucionPermisos = (*RepositorioResolucionPermisos)(nil)
	_ ports.RepositorioAvisosPropios      = (*RepositorioAvisosPropios)(nil)
)
