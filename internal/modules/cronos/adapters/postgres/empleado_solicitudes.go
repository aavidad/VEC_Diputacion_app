package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/cronos/application"
	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

// Segundo corte de la persona empleada sobre cronos_v1 000008. Cada llamada
// obtiene una decisión V3 nueva de su audiencia (AD3-70) que la función
// durable consume, audita y usa para fijar la RLS en la misma transacción.
const (
	consultaMovimientosPropios  = `SELECT vec_cronos_v1.consultar_movimientos_propio_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`
	consultaSolicitarCorreccion = `SELECT vec_cronos_v1.solicitar_correccion_propia_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`
	consultaPermisosPropios     = `SELECT vec_cronos_v1.consultar_permisos_propio_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`
	consultaSolicitarPermiso    = `SELECT vec_cronos_v1.solicitar_permiso_propio_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`
)

var (
	referenciaReciboSolicitud     = regexp.MustCompile(`^recibo:cronos:[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	referenciaActuacionCorreccion = regexp.MustCompile(`^correccion:actuacion:[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
)

// errorSolicitud traduce los rechazos nominales de 000008; el resto sigue la
// política común: nada que no sea nominal se presenta como denegación.
func errorSolicitud(ctx context.Context, err error, conflicto error) error {
	var pg *pgconn.PgError
	if (ctx == nil || ctx.Err() == nil) && errors.As(err, &pg) {
		switch pg.Code {
		case "PC001":
			return ports.ErrSolicitudCronosInvalida
		case "PC002":
			return conflicto
		case "PC007":
			return ports.ErrPermisoFueraDeLimites
		case "PC008":
			return ports.ErrCalendarioNoPublicado
		case "PC009":
			return ports.ErrPermisoNoSolicitable
		case "PC010":
			return ports.ErrPermisoSolapado
		}
	}
	return errorSeguro(ctx, err)
}

func traductor(conflicto error) func(context.Context, error) error {
	return func(ctx context.Context, err error) error { return errorSolicitud(ctx, err, conflicto) }
}

func sha256Hex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func decodificarEstricto(bruto []byte, destino any) error {
	if len(bruto) == 0 || len(bruto) > 4*1024*1024 {
		return ports.ErrDependenciaNoDisponible
	}
	dec := json.NewDecoder(bytes.NewReader(bruto))
	dec.DisallowUnknownFields()
	if dec.Decode(destino) != nil || dec.Decode(&struct{}{}) != io.EOF {
		return ports.ErrDependenciaNoDisponible
	}
	return nil
}

func empleadoDeOrden(actor vecdomain.ContextoActor, empleado string) bool {
	empleados, err := actor.Referencias(vecdomain.TipoReferenciaContextoActorEmpleado)
	return err == nil && len(empleados) == 1 && empleados[0] == empleado
}

// ---- Movimientos ----

type RepositorioConsultaMovimientos struct{ db iniciadorMarcaje }

func NuevoRepositorioConsultaMovimientos(pool *pgxpool.Pool) (*RepositorioConsultaMovimientos, error) {
	if pool == nil {
		return nil, ports.ErrDependenciaNoDisponible
	}
	return &RepositorioConsultaMovimientos{db: pool}, nil
}

func (r *RepositorioConsultaMovimientos) ConsultarMovimientos(ctx context.Context, orden ports.OrdenConsultaMovimientos, empleado, desde, hasta, zona string) (ports.ConsultaMovimientos, error) {
	if r == nil || r.db == nil || ctx == nil {
		return ports.ConsultaMovimientos{}, ports.ErrDependenciaNoDisponible
	}
	actor, err := orden.ContextoActor()
	proveedor := orden.ProveedorMaterial()
	if err != nil || proveedor == nil || !empleadoDeOrden(actor, empleado) {
		return ports.ConsultaMovimientos{}, ports.ErrDependenciaNoDisponible
	}
	m := domain.MaterialConsultaMovimientosPropios{ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, EmpleadoRef: empleado, Desde: desde, Hasta: hasta, ZonaHoraria: zona}
	canonico, err := m.Canonico()
	if err != nil {
		return ports.ConsultaMovimientos{}, ports.ErrConsultaSaldoInvalida
	}
	recurso, err := application.RecursoConsultaMovimientosPropios(m)
	if err != nil {
		return ports.ConsultaMovimientos{}, ports.ErrDependenciaNoDisponible
	}
	v3, err := proveedor.ProveerMaterialConsultaMovimientosPropios(ctx, m)
	if err != nil {
		return ports.ConsultaMovimientos{}, errorProveedorV3(ctx, err)
	}
	if !resumenV3Ligado(v3, application.AudienciaConsultaMovimientosPropios, application.AccionConsultarMovimientosPropios, recurso) {
		return ports.ConsultaMovimientos{}, ports.ErrDependenciaNoDisponible
	}
	bruto, err := ejecutarFuncionV3(ctx, r.db, consultaMovimientosPropios, canonico, v3, traductor(ports.ErrDependenciaNoDisponible))
	if err != nil {
		return ports.ConsultaMovimientos{}, err
	}
	defer clear(bruto)
	var sql struct {
		EmpleadoRef    string                       `json:"empleado_ref"`
		Desde          string                       `json:"desde"`
		Hasta          string                       `json:"hasta"`
		ZonaHoraria    string                       `json:"zona_horaria"`
		Calendario     *ports.CalendarioMovimientos `json:"calendario"`
		MarcajesPorDia []ports.MarcajesDia          `json:"marcajes_por_dia"`
		Absentismos    []ports.Absentismo           `json:"absentismos"`
		Correcciones   []ports.CorreccionPropia     `json:"correcciones"`
	}
	if decodificarEstricto(bruto, &sql) != nil || sql.EmpleadoRef != empleado || sql.ZonaHoraria != zona || sql.Calendario == nil ||
		sql.MarcajesPorDia == nil || sql.Absentismos == nil || sql.Correcciones == nil || sql.Calendario.Dias == nil {
		return ports.ConsultaMovimientos{}, ports.ErrDependenciaNoDisponible
	}
	for i := range sql.Correcciones {
		sql.Correcciones[i].SolicitadaEnUTC = sql.Correcciones[i].SolicitadaEnUTC.UTC()
	}
	return ports.ConsultaMovimientos{
		Periodo:    ports.PeriodoConsultaSaldo{Desde: sql.Desde, Hasta: sql.Hasta},
		Calendario: *sql.Calendario, MarcajesPorDia: sql.MarcajesPorDia, Absentismos: sql.Absentismos, Correcciones: sql.Correcciones,
	}, nil
}

// ---- Corrección de un olvido ----

// RepositorioCorreccionesPropias implementa sólo la solicitud de la persona
// empleada. Las decisiones de jefatura y RRHH, la aplicación y la
// recuperación del recibo pertenecen a otro corte y fallan cerradas.
type RepositorioCorreccionesPropias struct {
	db   iniciadorMarcaje
	zona string
}

func NuevoRepositorioCorreccionesPropias(pool *pgxpool.Pool, zona string) (*RepositorioCorreccionesPropias, error) {
	if pool == nil || (zona != domain.ZonaSaldoPeninsula && zona != domain.ZonaSaldoCanarias) {
		return nil, ports.ErrDependenciaNoDisponible
	}
	return &RepositorioCorreccionesPropias{db: pool, zona: zona}, nil
}

func (r *RepositorioCorreccionesPropias) SolicitarOlvido(ctx context.Context, s domain.SolicitudCorreccion, orden ports.OrdenConsumoCorreccion) (ports.ReciboCorreccion, error) {
	if r == nil || r.db == nil || ctx == nil {
		return ports.ReciboCorreccion{}, ports.ErrDependenciaNoDisponible
	}
	actor, err := orden.ContextoActor()
	proveedor := orden.ProveedorMaterial()
	if err != nil || dependenciaPostgresNula(proveedor) || !empleadoDeOrden(actor, s.EmpleadoRef) ||
		actor.PersonaRef != s.ActorRef || actor.PerfilActivoRef != s.PerfilRef {
		return ports.ReciboCorreccion{}, ports.ErrDependenciaNoDisponible
	}
	canonico, err := domain.MaterialSolicitudCorreccionPropia(s, r.zona)
	if err != nil {
		return ports.ReciboCorreccion{}, ports.ErrSolicitudCronosInvalida
	}
	h := sha256Hex(canonico)
	material := domain.MaterialAutorizacionCorreccion{
		ActorRef: s.ActorRef, PerfilRef: s.PerfilRef, EmpleadoRef: s.EmpleadoRef,
		SolicitudRef: "correccion:cronos:" + s.ClaveOperacion, ClaveOperacion: s.ClaveOperacion,
		Paso: domain.PasoSolicitudCorreccion, ComandoSHA256: h, InstanteUTC: s.SolicitadaEnUTC,
	}
	recurso, err := application.RecursoSolicitudCorreccionPropia(material)
	if err != nil {
		return ports.ReciboCorreccion{}, ports.ErrDependenciaNoDisponible
	}
	v3, err := proveedor.ProveerMaterialCorreccion(ctx, material)
	if err != nil {
		return ports.ReciboCorreccion{}, errorProveedorV3(ctx, err)
	}
	if !resumenV3Ligado(v3, application.AudienciaSolicitudCorreccionPropia, application.AccionSolicitarCorreccion, recurso) {
		return ports.ReciboCorreccion{}, ports.ErrDependenciaNoDisponible
	}
	bruto, err := ejecutarFuncionV3(ctx, r.db, consultaSolicitarCorreccion, canonico, v3, traductor(ports.ErrCorreccionEnConflicto))
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
	if decodificarEstricto(bruto, &sql) != nil || sql.Replay == nil || sql.SolicitudRef != material.SolicitudRef ||
		!referenciaReciboSolicitud.MatchString(sql.ReciboRef) || !referenciaActuacionCorreccion.MatchString(sql.ActuacionRef) {
		return ports.ReciboCorreccion{}, ports.ErrDependenciaNoDisponible
	}
	return ports.ReciboCorreccion{SolicitudRef: sql.SolicitudRef, ActuacionRef: sql.ActuacionRef, ReciboRef: sql.ReciboRef,
		Estado: sql.Estado, Version: sql.Version, InstanteUTC: sql.InstanteUTC.UTC(), Replay: *sql.Replay}, nil
}

func (r *RepositorioCorreccionesPropias) RegistrarActuacion(context.Context, domain.ActuacionCorreccion, ports.OrdenConsumoCorreccion) (ports.ReciboCorreccion, error) {
	return ports.ReciboCorreccion{}, ports.ErrDependenciaNoDisponible
}

func (r *RepositorioCorreccionesPropias) RecuperarRecibo(context.Context, ports.ClaveRecuperacionCorreccion, ports.OrdenConsumoCorreccion) (ports.ReciboCorreccion, error) {
	return ports.ReciboCorreccion{}, ports.ErrDependenciaNoDisponible
}

// ---- Permisos ----

type RepositorioPermisosPropios struct{ db iniciadorMarcaje }

func NuevoRepositorioPermisosPropios(pool *pgxpool.Pool) (*RepositorioPermisosPropios, error) {
	if pool == nil {
		return nil, ports.ErrDependenciaNoDisponible
	}
	return &RepositorioPermisosPropios{db: pool}, nil
}

type catalogoSQL struct {
	PermisoRef          string     `json:"permiso_ref"`
	VersionRef          string     `json:"version_ref"`
	Nombre              string     `json:"nombre"`
	FuenteRef           string     `json:"fuente_ref"`
	VigenteDesde        time.Time  `json:"vigente_desde"`
	VigenteHasta        *time.Time `json:"vigente_hasta"`
	Unidad              string     `json:"unidad"`
	Computo             string     `json:"computo"`
	Circuito            string     `json:"circuito"`
	Minimo              int64      `json:"minimo"`
	MaximoSolicitud     *int64     `json:"maximo_solicitud"`
	MaximoMensual       *int64     `json:"maximo_mensual"`
	MaximoAnual         *int64     `json:"maximo_anual"`
	JustificanteExigido bool       `json:"justificante_exigido"`
	Solicitable         *bool      `json:"solicitable"`
	Sintetico           *bool      `json:"sintetico"`
}

type solicitudSQL struct {
	SolicitudRef        string     `json:"solicitud_ref"`
	CatalogoVersionRef  string     `json:"catalogo_version_ref"`
	PermisoRef          string     `json:"permiso_ref"`
	Desde               string     `json:"desde"`
	Hasta               string     `json:"hasta"`
	HoraInicio          *string    `json:"hora_inicio"`
	HoraFin             *string    `json:"hora_fin"`
	Cantidad            int64      `json:"cantidad"`
	Unidad              string     `json:"unidad"`
	Estado              string     `json:"estado"`
	Version             int        `json:"version"`
	PendienteJustificar bool       `json:"pendiente_justificar"`
	SolicitadaEn        *time.Time `json:"solicitada_en"`
	// Los añade cronos_v1 000010; sin ella faltan los dos.
	Circuito            *string `json:"circuito"`
	PendienteAsignacion *bool   `json:"pendiente_asignacion"`
}

// circuitoSolicitudPropia valida el circuito aplicado que devuelve 000010:
// ausente del todo antes de 000010; si no, J-A o A, obligatorio mientras la
// solicitud está viva (pendiente de RRHH sólo tras la jefatura) y pendiente
// de asignación sólo en lo solicitado por J-A.
func circuitoSolicitudPropia(s solicitudSQL) (domain.CircuitoPermiso, bool, bool) {
	if s.PendienteAsignacion == nil {
		return "", false, s.Circuito == nil
	}
	var c domain.CircuitoPermiso
	if s.Circuito != nil {
		c = domain.CircuitoPermiso(*s.Circuito)
		if c != domain.CircuitoAdministracion && c != domain.CircuitoResponsableAdministracion {
			return "", false, false
		}
	}
	switch domain.EstadoSolicitudPermiso(s.Estado) {
	case domain.EstadoPermisoSolicitado:
		if c == "" || (*s.PendienteAsignacion && c != domain.CircuitoResponsableAdministracion) {
			return "", false, false
		}
	case domain.EstadoPermisoPendienteAdministracion:
		if c != domain.CircuitoResponsableAdministracion || *s.PendienteAsignacion {
			return "", false, false
		}
	default:
		if *s.PendienteAsignacion {
			return "", false, false
		}
	}
	return c, *s.PendienteAsignacion, true
}

func (r *RepositorioPermisosPropios) ConsultarPermisosPropios(ctx context.Context, orden ports.OrdenPermisosPropios, empleado string, anio int, zona string) (ports.FuentePermisosPropios, error) {
	if r == nil || r.db == nil || ctx == nil {
		return ports.FuentePermisosPropios{}, ports.ErrDependenciaNoDisponible
	}
	actor, err := orden.ContextoActor()
	proveedor := orden.ProveedorMaterial()
	if err != nil || proveedor == nil || !empleadoDeOrden(actor, empleado) {
		return ports.FuentePermisosPropios{}, ports.ErrDependenciaNoDisponible
	}
	m := domain.MaterialConsultaPermisosPropios{ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, EmpleadoRef: empleado, Anio: anio, ZonaHoraria: zona}
	canonico, err := m.Canonico()
	if err != nil {
		return ports.FuentePermisosPropios{}, ports.ErrSolicitudCronosInvalida
	}
	recurso, err := application.RecursoConsultaPermisosPropios(m)
	if err != nil {
		return ports.FuentePermisosPropios{}, ports.ErrDependenciaNoDisponible
	}
	v3, err := proveedor.ProveerMaterialConsultaPermisosPropios(ctx, m)
	if err != nil {
		return ports.FuentePermisosPropios{}, errorProveedorV3(ctx, err)
	}
	if !resumenV3Ligado(v3, application.AudienciaConsultaPermisosPropios, application.AccionConsultarPermisosPropios, recurso) {
		return ports.FuentePermisosPropios{}, ports.ErrDependenciaNoDisponible
	}
	bruto, err := ejecutarFuncionV3(ctx, r.db, consultaPermisosPropios, canonico, v3, traductor(ports.ErrDependenciaNoDisponible))
	if err != nil {
		return ports.FuentePermisosPropios{}, err
	}
	defer clear(bruto)
	var sql struct {
		EmpleadoRef string         `json:"empleado_ref"`
		Anio        int            `json:"anio"`
		Catalogo    []catalogoSQL  `json:"catalogo"`
		Solicitudes []solicitudSQL `json:"solicitudes"`
	}
	if decodificarEstricto(bruto, &sql) != nil || sql.Catalogo == nil || sql.Solicitudes == nil {
		return ports.FuentePermisosPropios{}, ports.ErrDependenciaNoDisponible
	}
	f := ports.FuentePermisosPropios{EmpleadoRef: sql.EmpleadoRef, Anio: sql.Anio,
		Catalogo: make([]ports.EntradaCatalogoPropio, 0, len(sql.Catalogo)), Solicitudes: make([]ports.SolicitudPermisoPropia, 0, len(sql.Solicitudes))}
	for _, c := range sql.Catalogo {
		if c.Solicitable == nil || c.Sintetico == nil {
			return ports.FuentePermisosPropios{}, ports.ErrDependenciaNoDisponible
		}
		v := domain.CatalogoPermisoVersion{
			PermisoRef: c.PermisoRef, VersionRef: c.VersionRef, FuenteRef: c.FuenteRef, Nombre: c.Nombre,
			VigenteDesde: c.VigenteDesde.UTC(), Unidad: domain.LeaveUnit(c.Unidad), Computo: domain.ComputoPermiso(c.Computo),
			Circuito: domain.CircuitoPermiso(c.Circuito), Minimo: c.Minimo, MaximoSolicitud: c.MaximoSolicitud,
			MaximoMensual: c.MaximoMensual, MaximoAnual: c.MaximoAnual, JustificanteExigido: c.JustificanteExigido,
		}
		if c.VigenteHasta != nil {
			v.VigenteHasta = c.VigenteHasta.UTC()
		}
		if v.Validar() != nil {
			return ports.FuentePermisosPropios{}, ports.ErrDependenciaNoDisponible
		}
		f.Catalogo = append(f.Catalogo, ports.EntradaCatalogoPropio{Version: v, Solicitable: *c.Solicitable, Sintetico: *c.Sintetico})
	}
	for _, s := range sql.Solicitudes {
		circuito, pendienteAsignacion, ok := circuitoSolicitudPropia(s)
		if s.SolicitadaEn == nil || s.Version < 1 || !ok {
			return ports.FuentePermisosPropios{}, ports.ErrDependenciaNoDisponible
		}
		p := ports.SolicitudPermisoPropia{SolicitudRef: s.SolicitudRef, CatalogoVersionRef: s.CatalogoVersionRef, PermisoRef: s.PermisoRef,
			Desde: s.Desde, Hasta: s.Hasta, Cantidad: s.Cantidad, Unidad: domain.LeaveUnit(s.Unidad), Estado: domain.EstadoSolicitudPermiso(s.Estado),
			Version: s.Version, PendienteJustificar: s.PendienteJustificar, SolicitadaEnUTC: s.SolicitadaEn.UTC(),
			Circuito: circuito, PendienteAsignacion: pendienteAsignacion}
		if s.HoraInicio != nil && s.HoraFin != nil {
			p.HoraInicio, p.HoraFin = *s.HoraInicio, *s.HoraFin
		}
		f.Solicitudes = append(f.Solicitudes, p)
	}
	return f, nil
}

func (r *RepositorioPermisosPropios) SolicitarPermisoPropio(ctx context.Context, orden ports.OrdenPermisosPropios, m domain.MaterialSolicitudPermisoPropio) (ports.ReciboPermisoPropio, error) {
	if r == nil || r.db == nil || ctx == nil {
		return ports.ReciboPermisoPropio{}, ports.ErrDependenciaNoDisponible
	}
	actor, err := orden.ContextoActor()
	proveedor := orden.ProveedorMaterial()
	if err != nil || proveedor == nil || !empleadoDeOrden(actor, m.EmpleadoRef) || actor.PersonaRef != m.ActorRef || actor.PerfilActivoRef != m.PerfilRef {
		return ports.ReciboPermisoPropio{}, ports.ErrDependenciaNoDisponible
	}
	canonico, err := m.Canonico()
	if err != nil {
		return ports.ReciboPermisoPropio{}, ports.ErrSolicitudCronosInvalida
	}
	recurso, err := application.RecursoSolicitudPermisoPropio(m)
	if err != nil {
		return ports.ReciboPermisoPropio{}, ports.ErrDependenciaNoDisponible
	}
	v3, err := proveedor.ProveerMaterialSolicitudPermisoPropio(ctx, m)
	if err != nil {
		return ports.ReciboPermisoPropio{}, errorProveedorV3(ctx, err)
	}
	if !resumenV3Ligado(v3, application.AudienciaSolicitudPermisoPropio, application.AccionSolicitarPermisoPropio, recurso) {
		return ports.ReciboPermisoPropio{}, ports.ErrDependenciaNoDisponible
	}
	bruto, err := ejecutarFuncionV3(ctx, r.db, consultaSolicitarPermiso, canonico, v3, traductor(ports.ErrClaveOperacionEnConflicto))
	if err != nil {
		return ports.ReciboPermisoPropio{}, err
	}
	defer clear(bruto)
	var sql struct {
		ports.ReciboPermisoPropio
		Replay *bool `json:"replay"`
	}
	if decodificarEstricto(bruto, &sql) != nil || sql.Replay == nil || !referenciaReciboSolicitud.MatchString(sql.ReciboRef) {
		return ports.ReciboPermisoPropio{}, ports.ErrDependenciaNoDisponible
	}
	recibo := sql.ReciboPermisoPropio
	recibo.Replay = *sql.Replay
	recibo.InstanteUTC = recibo.InstanteUTC.UTC()
	return recibo, nil
}

var (
	_ ports.RepositorioConsultaMovimientos = (*RepositorioConsultaMovimientos)(nil)
	_ ports.RepositorioCorrecciones        = (*RepositorioCorreccionesPropias)(nil)
	_ ports.RepositorioPermisosPropios     = (*RepositorioPermisosPropios)(nil)
)
