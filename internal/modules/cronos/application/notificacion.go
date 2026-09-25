package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

const (
	AudienciaComunicacionesCronos = "vec_cronos_v1.comunicaciones.v1"
	AccionEnviarNotificacion      = "cronos.notificacion.propia.enviar"
	FinalidadEnviarNotificacion   = "comunicar_incidencia_rrhh"
	AccionLeerNotificacion        = "cronos.notificacion.propia.leer"
	AccionLeerMensaje             = "cronos.mensaje.propio.leer"
)

// RecursoEnvioNotificacion fija la preimagen V3 sin exponer el texto en el
// recurso ni en logs. SQL debe reconstruir la misma huella semántica.
func RecursoEnvioNotificacion(m domain.MaterialAutorizacionNotificacion) (vecdomain.RecursoAutorizable, error) {
	b, err := m.Canonico()
	if err != nil {
		return vecdomain.RecursoAutorizable{}, ports.ErrComunicacionNoAcreditada
	}
	h := sha256.Sum256(b)
	r := vecdomain.RecursoAutorizable{Referencia: "notificacion:cronos:" + m.Notificacion.ClaveOperacion, ModuloID: "cronos", Tipo: "notificacion_propia", Ambitos: map[string]string{"empleado_ref": m.Notificacion.EmpleadoRef}, Atributos: map[string]string{"material_sha256": hex.EncodeToString(h[:])}}
	if r.Validar() != nil {
		return vecdomain.RecursoAutorizable{}, ports.ErrComunicacionNoAcreditada
	}
	return r, nil
}

type ServicioNotificaciones struct {
	repositorio ports.RepositorioNotificaciones
	reloj       ports.Reloj
}

func NuevoServicioNotificaciones(r ports.RepositorioNotificaciones, reloj ports.Reloj) (*ServicioNotificaciones, error) {
	if r == nil || reloj == nil {
		return nil, ports.ErrDependenciaNoDisponible
	}
	return &ServicioNotificaciones{repositorio: r, reloj: reloj}, nil
}

func contextoComunicaciones(ctx context.Context, orden ports.OrdenComunicaciones, reloj ports.Reloj) (vecdomain.ContextoActor, string, time.Time, error) {
	actor, ahora, err := contextoActorComunicaciones(ctx, orden, reloj)
	if err != nil {
		return vecdomain.ContextoActor{}, "", time.Time{}, err
	}
	empleados, err := actor.Referencias(vecdomain.TipoReferenciaContextoActorEmpleado)
	if err != nil || len(empleados) != 1 {
		return vecdomain.ContextoActor{}, "", time.Time{}, ports.ErrComunicacionNoAcreditada
	}
	return actor, empleados[0], ahora, nil
}

func contextoActorComunicaciones(ctx context.Context, orden ports.OrdenComunicaciones, reloj ports.Reloj) (vecdomain.ContextoActor, time.Time, error) {
	if ctx == nil || reloj == nil || orden.Proveedor() == nil {
		return vecdomain.ContextoActor{}, time.Time{}, ports.ErrComunicacionNoAcreditada
	}
	actor, err := orden.Actor()
	if err != nil {
		return vecdomain.ContextoActor{}, time.Time{}, ports.ErrComunicacionNoAcreditada
	}
	ahora := reloj.AhoraUTC().UTC().Truncate(time.Microsecond)
	if ahora.IsZero() || !actor.Instantanea.VigenteEn(ahora) {
		return vecdomain.ContextoActor{}, time.Time{}, ports.ErrComunicacionNoAcreditada
	}
	return actor, ahora, nil
}

func (s *ServicioNotificaciones) EnviarNotificacion(ctx context.Context, orden ports.OrdenComunicaciones, sol ports.SolicitudNotificacion) (ports.ReciboNotificacion, error) {
	if s == nil || s.repositorio == nil {
		return ports.ReciboNotificacion{}, ports.ErrDependenciaNoDisponible
	}
	actor, empleado, ahora, err := contextoComunicaciones(ctx, orden, s.reloj)
	if err != nil {
		return ports.ReciboNotificacion{}, err
	}
	n := domain.Notificacion{EmpleadoRef: empleado, TipoRef: sol.TipoRef, TipoVersion: sol.TipoVersion, FechaReferida: sol.FechaReferida, Texto: sol.Texto, AdjuntoRef: sol.AdjuntoRef, ClaveOperacion: sol.ClaveOperacion, InstanteUTC: ahora}
	m := domain.MaterialAutorizacionNotificacion{ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, Notificacion: n}
	if m.Validar() != nil {
		return ports.ReciboNotificacion{}, domain.ErrNotificacionInvalida
	}
	if _, err := RecursoEnvioNotificacion(m); err != nil {
		return ports.ReciboNotificacion{}, ports.ErrComunicacionNoAcreditada
	}
	v3, err := orden.Proveedor().ProveerEnvioNotificacion(ctx, m)
	if err != nil || v3.ValidarEstructura() != nil {
		return ports.ReciboNotificacion{}, ports.ErrComunicacionNoAcreditada
	}
	recibo, err := s.repositorio.RegistrarAutorizada(ctx, m, v3)
	if err != nil {
		return ports.ReciboNotificacion{}, err
	}
	if !domain.ReferenciaMensajeValida(recibo.Referencia) || !domain.ReferenciaMensajeValida(recibo.NotificacionRef) ||
		recibo.RegistradaUTC.IsZero() || recibo.RegistradaUTC.Location() != time.UTC || recibo.RegistradaUTC.Nanosecond()%1000 != 0 ||
		(recibo.Estado != ports.NotificacionRegistrada && recibo.Estado != ports.NotificacionPendienteEnvio) {
		return ports.ReciboNotificacion{}, ports.ErrDependenciaNoDisponible
	}
	return recibo, nil
}

func (s *ServicioNotificaciones) ListarTiposNotificacion(ctx context.Context, orden ports.OrdenComunicaciones) ([]ports.TipoNotificacionDisponible, error) {
	if s == nil || s.repositorio == nil {
		return nil, ports.ErrDependenciaNoDisponible
	}
	actor, empleado, _, err := contextoComunicaciones(ctx, orden, s.reloj)
	if err != nil {
		return nil, err
	}
	return s.repositorio.TiposDisponibles(ctx, actor, empleado)
}

func (s *ServicioNotificaciones) ListarNotificaciones(ctx context.Context, orden ports.OrdenComunicaciones, pendientes bool) ([]ports.NotificacionGuardada, error) {
	if s == nil || s.repositorio == nil {
		return nil, ports.ErrDependenciaNoDisponible
	}
	actor, empleado, _, err := contextoComunicaciones(ctx, orden, s.reloj)
	if err != nil {
		return nil, err
	}
	return s.repositorio.ListarPropias(ctx, actor, empleado, pendientes)
}

func (s *ServicioNotificaciones) ConsultarNotificacion(ctx context.Context, orden ports.OrdenComunicaciones, referencia string) (ports.NotificacionGuardada, error) {
	if s == nil || s.repositorio == nil {
		return ports.NotificacionGuardada{}, ports.ErrDependenciaNoDisponible
	}
	actor, empleado, _, err := contextoComunicaciones(ctx, orden, s.reloj)
	if err != nil {
		return ports.NotificacionGuardada{}, err
	}
	if !domain.ReferenciaMensajeValida(referencia) {
		return ports.NotificacionGuardada{}, domain.ErrNotificacionInvalida
	}
	return s.repositorio.ConsultarPropia(ctx, actor, empleado, referencia)
}
