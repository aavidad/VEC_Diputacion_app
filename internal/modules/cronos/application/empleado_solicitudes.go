package application

import (
	"context"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

// empleadoVigente exige el único empleado canónico del contexto registrado
// y que la instantánea siga vigente en el reloj del servidor.
func empleadoVigente(actor vecdomain.ContextoActor, reloj ports.Reloj) (string, time.Time, error) {
	empleados, err := actor.Referencias(vecdomain.TipoReferenciaContextoActorEmpleado)
	if err != nil || len(empleados) != 1 {
		return "", time.Time{}, ports.ErrDependenciaNoDisponible
	}
	ahora := reloj.AhoraUTC().UTC().Truncate(time.Microsecond)
	if ahora.IsZero() || !actor.Instantanea.VigenteEn(ahora) {
		return "", time.Time{}, ports.ErrDependenciaNoDisponible
	}
	return empleados[0], ahora, nil
}

// ServicioConsultaMovimientos resuelve el periodo como el saldo propio y lee
// calendario, absentismos y correcciones de la persona.
type ServicioConsultaMovimientos struct {
	repositorio ports.RepositorioConsultaMovimientos
	reloj       ports.Reloj
	zona        *time.Location
}

func NuevoServicioConsultaMovimientos(repo ports.RepositorioConsultaMovimientos, reloj ports.Reloj, zona *time.Location) (*ServicioConsultaMovimientos, error) {
	if repo == nil || reloj == nil || zona == nil {
		return nil, ErrServiceDependencyRequired
	}
	return &ServicioConsultaMovimientos{repositorio: repo, reloj: reloj, zona: zona}, nil
}

func (s *ServicioConsultaMovimientos) ConsultarMovimientos(ctx context.Context, orden ports.OrdenConsultaMovimientos, tipo ports.PeriodoSaldo, desdeArg, hastaArg string) (ports.ConsultaMovimientos, error) {
	if s == nil || s.repositorio == nil || s.reloj == nil || s.zona == nil || ctx == nil {
		return ports.ConsultaMovimientos{}, ports.ErrDependenciaNoDisponible
	}
	actor, err := orden.ContextoActor()
	if err != nil {
		return ports.ConsultaMovimientos{}, err
	}
	empleado, ahora, err := empleadoVigente(actor, s.reloj)
	if err != nil {
		return ports.ConsultaMovimientos{}, err
	}
	desde, hasta, err := resolverPeriodoSaldo(tipo, desdeArg, hastaArg, ahora.In(s.zona), s.zona)
	if err != nil {
		return ports.ConsultaMovimientos{}, ports.ErrConsultaSaldoInvalida
	}
	d, h := desde.Format(time.DateOnly), hasta.Format(time.DateOnly)
	r, err := s.repositorio.ConsultarMovimientos(ctx, orden, empleado, d, h, s.zona.String())
	if err != nil {
		return ports.ConsultaMovimientos{}, err
	}
	if r.Periodo.Desde != d || r.Periodo.Hasta != h || !movimientosCoherentes(r, d, h) {
		return ports.ConsultaMovimientos{}, ports.ErrDependenciaNoDisponible
	}
	r.Periodo.Tipo = tipo
	return r, nil
}

// movimientosCoherentes rechaza una fuente que devuelva hechos fuera del
// periodo pedido o un calendario no disponible con días.
func movimientosCoherentes(r ports.ConsultaMovimientos, desde, hasta string) bool {
	if !r.Calendario.Disponible && len(r.Calendario.Dias) > 0 {
		return false
	}
	for _, d := range r.Calendario.Dias {
		if d.Fecha < desde || d.Fecha > hasta || (d.Tipo != "festivo" && d.Tipo != "no_laborable") || d.Nombre == "" {
			return false
		}
	}
	for _, m := range r.MarcajesPorDia {
		if m.Fecha < desde || m.Fecha > hasta || m.Marcajes < 1 {
			return false
		}
	}
	for _, a := range r.Absentismos {
		if a.Hasta < desde || a.Desde > hasta || a.Desde > a.Hasta || a.Cantidad < 1 || (a.Unidad != domain.LeaveUnitDay && a.Unidad != domain.LeaveUnitHour) {
			return false
		}
	}
	for _, c := range r.Correcciones {
		if c.FechaCivil < desde || c.FechaCivil > hasta || c.Version < 1 {
			return false
		}
	}
	return true
}

// ServicioPermisosPropios presenta el listado anual del catálogo versionado
// con lo solicitado, concedido, pendiente de justificar y la resta, y crea
// solicitudes que quedan pendientes de conceder. No concede ni deniega.
type ServicioPermisosPropios struct {
	repositorio ports.RepositorioPermisosPropios
	reloj       ports.Reloj
	zona        *time.Location
}

func NuevoServicioPermisosPropios(repo ports.RepositorioPermisosPropios, reloj ports.Reloj, zona *time.Location) (*ServicioPermisosPropios, error) {
	if repo == nil || reloj == nil || zona == nil {
		return nil, ErrServiceDependencyRequired
	}
	return &ServicioPermisosPropios{repositorio: repo, reloj: reloj, zona: zona}, nil
}

func (s *ServicioPermisosPropios) contexto(ctx context.Context, orden ports.OrdenPermisosPropios) (vecdomain.ContextoActor, string, time.Time, error) {
	if s == nil || s.repositorio == nil || s.reloj == nil || s.zona == nil || ctx == nil {
		return vecdomain.ContextoActor{}, "", time.Time{}, ports.ErrDependenciaNoDisponible
	}
	actor, err := orden.ContextoActor()
	if err != nil {
		return vecdomain.ContextoActor{}, "", time.Time{}, err
	}
	empleado, ahora, err := empleadoVigente(actor, s.reloj)
	return actor, empleado, ahora, err
}

// ConsultarPermisosPropios: anio 0 significa el año civil en curso.
func (s *ServicioPermisosPropios) ConsultarPermisosPropios(ctx context.Context, orden ports.OrdenPermisosPropios, anio int) (ports.ConsultaPermisosPropios, error) {
	_, empleado, ahora, err := s.contexto(ctx, orden)
	if err != nil {
		return ports.ConsultaPermisosPropios{}, err
	}
	if anio == 0 {
		anio = ahora.In(s.zona).Year()
	}
	if anio < 2000 || anio > 2100 {
		return ports.ConsultaPermisosPropios{}, ports.ErrSolicitudCronosInvalida
	}
	f, err := s.repositorio.ConsultarPermisosPropios(ctx, orden, empleado, anio, s.zona.String())
	if err != nil {
		return ports.ConsultaPermisosPropios{}, err
	}
	if f.EmpleadoRef != empleado || f.Anio != anio || len(f.Catalogo) > 500 || len(f.Solicitudes) > 5000 {
		return ports.ConsultaPermisosPropios{}, ports.ErrDependenciaNoDisponible
	}
	return proyectarPermisosPropios(f)
}

func proyectarPermisosPropios(f ports.FuentePermisosPropios) (ports.ConsultaPermisosPropios, error) {
	hechos := make(map[string][]domain.HechoResumenPermiso)
	anio := time.Date(f.Anio, 1, 1, 0, 0, 0, 0, time.UTC).Format("2006")
	for _, s := range f.Solicitudes {
		if len(s.Desde) != 10 || s.Desde[:4] != anio {
			return ports.ConsultaPermisosPropios{}, ports.ErrDependenciaNoDisponible
		}
		hechos[s.PermisoRef] = append(hechos[s.PermisoRef], domain.HechoResumenPermiso{
			SolicitudRef: s.SolicitudRef, CatalogoVersionRef: s.CatalogoVersionRef, Unidad: s.Unidad,
			Cantidad: s.Cantidad, Estado: s.Estado, PendienteJustificar: s.PendienteJustificar,
		})
	}
	r := ports.ConsultaPermisosPropios{Anio: f.Anio, Permisos: make([]ports.PermisoAnualPropio, 0, len(f.Catalogo)), Solicitudes: f.Solicitudes}
	if r.Solicitudes == nil {
		r.Solicitudes = []ports.SolicitudPermisoPropia{}
	}
	vistos := make(map[string]bool, len(f.Catalogo))
	for _, e := range f.Catalogo {
		c := e.Version
		if vistos[c.PermisoRef] {
			return ports.ConsultaPermisosPropios{}, ports.ErrDependenciaNoDisponible
		}
		vistos[c.PermisoRef] = true
		p, err := domain.ProyectarPermisoAnual(c, f.Anio, hechos[c.PermisoRef])
		if err != nil {
			return ports.ConsultaPermisosPropios{}, ports.ErrDependenciaNoDisponible
		}
		r.Permisos = append(r.Permisos, ports.PermisoAnualPropio{
			PermisoRef: c.PermisoRef, VersionRef: c.VersionRef, Nombre: c.Nombre, Unidad: c.Unidad, Computo: c.Computo,
			Circuito: c.Circuito, Minimo: c.Minimo, MaximoSolicitud: c.MaximoSolicitud, MaximoMensual: c.MaximoMensual,
			MaximoAnual: c.MaximoAnual, JustificanteExigido: c.JustificanteExigido, Solicitable: e.Solicitable, Sintetico: e.Sintetico,
			Solicitado: p.Solicitado, Concedido: p.Concedido, PendienteJustificar: p.PendienteJustificar, Resta: p.Resta, SinConciliar: p.SinConciliar,
		})
	}
	return r, nil
}

func (s *ServicioPermisosPropios) SolicitarPermisoPropio(ctx context.Context, orden ports.OrdenPermisosPropios, p ports.PeticionPermisoPropio) (ports.ReciboPermisoPropio, error) {
	actor, empleado, _, err := s.contexto(ctx, orden)
	if err != nil {
		return ports.ReciboPermisoPropio{}, err
	}
	m := domain.MaterialSolicitudPermisoPropio{
		ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, EmpleadoRef: empleado,
		ClaveOperacion: p.ClaveOperacion, PermisoRef: p.PermisoRef, Desde: p.Desde, Hasta: p.Hasta,
		HoraInicio: p.HoraInicio, HoraFin: p.HoraFin, ZonaHoraria: s.zona.String(),
	}
	if m.Validar() != nil {
		return ports.ReciboPermisoPropio{}, ports.ErrSolicitudCronosInvalida
	}
	r, err := s.repositorio.SolicitarPermisoPropio(ctx, orden, m)
	if err != nil {
		return ports.ReciboPermisoPropio{}, err
	}
	if r.SolicitudRef != domain.SolicitudPermisoPropioRef(p.ClaveOperacion) || r.Version != 1 || r.Estado != domain.EstadoPermisoSolicitado ||
		r.Cantidad < 1 || (r.Unidad != domain.LeaveUnitDay && r.Unidad != domain.LeaveUnitHour) || r.ReciboRef == "" || r.CatalogoVersionRef == "" ||
		r.InstanteUTC.IsZero() || r.InstanteUTC.Location() != time.UTC {
		return ports.ReciboPermisoPropio{}, ports.ErrDependenciaNoDisponible
	}
	return r, nil
}

var (
	_ ports.CasoUsoConsultarMovimientos = (*ServicioConsultaMovimientos)(nil)
	_ ports.CasoUsoPermisosPropios      = (*ServicioPermisosPropios)(nil)
)
