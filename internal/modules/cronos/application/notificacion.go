package application

import (
	"context"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

// Contrato V3 de las notificaciones a RRHH (AD3-58). La función durable de
// cronos_v1 000010 comprueba exactamente estos valores. La persona se
// autoriza sobre su empleado propio; RRHH, como en su bandeja de permisos,
// sobre {persona_ref, paso_resolucion=administracion}, y el circuito
// publicado sólo restringe a qué personas atiende.
//
// Sustituye a un ServicioNotificaciones anterior que nunca se compuso: exigía
// adjunto sin huella, no tenía bandeja de RRHH ni función durable y usaba un
// proveedor compartido con los mensajes que ya no encaja con AD3-58.
const (
	AudienciaRegistroNotificacion  = "vec_cronos_v1.notificacion_propia.registrar.v1"
	AccionRegistrarNotificacion    = "cronos.notificacion.propia.registrar"
	FinalidadRegistrarNotificacion = "comunicar_incidencia_rrhh"

	AudienciaConsultaNotificacionesPropias  = "vec_cronos_v1.notificaciones_propio.consultar.v1"
	AccionConsultarNotificacionesPropias    = "cronos.notificaciones.propio.consultar"
	FinalidadConsultarNotificacionesPropias = "consultar_notificaciones_propio"

	AudienciaBandejaNotificaciones  = "vec_cronos_v1.notificaciones_bandeja.consultar.v1"
	AccionConsultarBandejaNotif     = "cronos.notificaciones.bandeja.consultar"
	FinalidadConsultarBandejaNotif  = "consultar_bandeja_notificaciones"
	AudienciaAtencionNotificacion   = "vec_cronos_v1.notificacion.atender.v1"
	AccionAtenderNotificacion       = "cronos.notificacion.atender"
	FinalidadAtenderNotificacion    = "atender_notificacion"
	maximoTiposNotificacion         = 100
	maximoFilasNotificaciones       = 500
	maximoEtiquetaEmpleadoNotificar = 120
)

func RecursoRegistroNotificacion(m domain.MaterialRegistroNotificacion) (vecdomain.RecursoAutorizable, error) {
	canonico, err := m.Canonico()
	if err != nil {
		return vecdomain.RecursoAutorizable{}, ErrContextoMarcajeNoAcreditado
	}
	return recursoEmpleado(domain.NotificacionPropiaRecursoRef(m.ClaveOperacion), "notificacion_propia", m.EmpleadoRef, canonico)
}

func RecursoConsultaNotificacionesPropias(m domain.MaterialConsultaNotificaciones) (vecdomain.RecursoAutorizable, error) {
	canonico, err := m.Canonico()
	if err != nil {
		return vecdomain.RecursoAutorizable{}, ErrContextoMarcajeNoAcreditado
	}
	return recursoEmpleado("notificaciones:cronos:"+m.EmpleadoRef, "notificaciones_propio", m.EmpleadoRef, canonico)
}

func RecursoBandejaNotificaciones(m domain.MaterialConsultaNotificaciones) (vecdomain.RecursoAutorizable, error) {
	canonico, err := m.Canonico()
	if err != nil {
		return vecdomain.RecursoAutorizable{}, ErrContextoMarcajeNoAcreditado
	}
	return recursoResolutor(domain.BandejaNotificacionesRef, "bandeja_notificaciones", m.ActorRef, domain.PasoAdministracion, canonico)
}

func RecursoAtencionNotificacion(m domain.MaterialAtencionNotificacion) (vecdomain.RecursoAutorizable, error) {
	canonico, err := m.Canonico()
	if err != nil {
		return vecdomain.RecursoAutorizable{}, ErrContextoMarcajeNoAcreditado
	}
	return recursoResolutor(domain.AtencionNotificacionRecursoRef(m.ClaveOperacion), "atencion_notificacion", m.ActorRef, domain.PasoAdministracion, canonico)
}

func instanteReciboValido(t time.Time) bool {
	return !t.IsZero() && t.Location() == time.UTC
}

// ServicioNotificacionesPropias: la persona consulta el catálogo vigente y
// sus notificaciones, y registra una nueva. No decide nada: la concesión V3
// y la función durable acreditan el efecto.
type ServicioNotificacionesPropias struct {
	repositorio ports.RepositorioNotificacionesPropias
	reloj       ports.Reloj
	zona        *time.Location
}

func NuevoServicioNotificacionesPropias(repo ports.RepositorioNotificacionesPropias, reloj ports.Reloj, zona *time.Location) (*ServicioNotificacionesPropias, error) {
	if repo == nil || reloj == nil || zona == nil {
		return nil, ErrServiceDependencyRequired
	}
	return &ServicioNotificacionesPropias{repositorio: repo, reloj: reloj, zona: zona}, nil
}

func (s *ServicioNotificacionesPropias) contexto(ctx context.Context, orden ports.OrdenNotificacionesPropias) (vecdomain.ContextoActor, string, error) {
	if s == nil || s.repositorio == nil || s.reloj == nil || s.zona == nil || ctx == nil {
		return vecdomain.ContextoActor{}, "", ports.ErrDependenciaNoDisponible
	}
	actor, err := orden.ContextoActor()
	if err != nil {
		return vecdomain.ContextoActor{}, "", err
	}
	empleado, _, err := empleadoVigente(actor, s.reloj)
	return actor, empleado, err
}

func (s *ServicioNotificacionesPropias) ConsultarPropias(ctx context.Context, orden ports.OrdenNotificacionesPropias) (ports.ConsultaNotificacionesPropias, error) {
	actor, empleado, err := s.contexto(ctx, orden)
	if err != nil {
		return ports.ConsultaNotificacionesPropias{}, err
	}
	m := domain.MaterialConsultaNotificaciones{ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, EmpleadoRef: empleado, ZonaHoraria: s.zona.String()}
	if m.Validar() != nil {
		return ports.ConsultaNotificacionesPropias{}, ports.ErrDependenciaNoDisponible
	}
	c, err := s.repositorio.ConsultarPropias(ctx, orden, m)
	if err != nil {
		return ports.ConsultaNotificacionesPropias{}, err
	}
	if !consultaPropiaCoherente(c) {
		return ports.ConsultaNotificacionesPropias{}, ports.ErrDependenciaNoDisponible
	}
	return c, nil
}

func consultaPropiaCoherente(c ports.ConsultaNotificacionesPropias) bool {
	if c.Tipos == nil || c.Notificaciones == nil || len(c.Tipos) > maximoTiposNotificacion || len(c.Notificaciones) > maximoFilasNotificaciones {
		return false
	}
	for _, t := range c.Tipos {
		if !domain.TipoNotificacionVersionValido(t.TipoVersionRef) || !domain.TipoNotificacionValido(t.TipoRef) || t.Nombre == "" {
			return false
		}
	}
	for _, n := range c.Notificaciones {
		atendida := n.Estado == ports.EstadoNotificacionAtendida
		if !domain.NotificacionRefValida(n.NotificacionRef) || !domain.TipoNotificacionValido(n.TipoRef) || n.TipoNombre == "" ||
			!domain.TextoNotificacionValido(n.Texto) || !domain.AdjuntoNotificacionValido(n.AdjuntoRef, n.AdjuntoSHA256) ||
			n.RegistradaEnUTC.IsZero() || (!atendida && n.Estado != ports.EstadoNotificacionRegistrada) || atendida != (n.AtendidaEnUTC != nil) {
			return false
		}
	}
	return true
}

func (s *ServicioNotificacionesPropias) RegistrarNotificacion(ctx context.Context, orden ports.OrdenNotificacionesPropias, p ports.PeticionRegistroNotificacion) (ports.ReciboNotificacion, error) {
	actor, empleado, err := s.contexto(ctx, orden)
	if err != nil {
		return ports.ReciboNotificacion{}, err
	}
	m := domain.MaterialRegistroNotificacion{ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, EmpleadoRef: empleado,
		ClaveOperacion: p.ClaveOperacion, TipoVersionRef: p.TipoVersionRef, FechaReferida: p.FechaReferida, Texto: p.Texto,
		AdjuntoRef: p.AdjuntoRef, AdjuntoSHA256: p.AdjuntoSHA256, ZonaHoraria: s.zona.String()}
	if m.Validar() != nil {
		return ports.ReciboNotificacion{}, ports.ErrSolicitudCronosInvalida
	}
	r, err := s.repositorio.RegistrarNotificacion(ctx, orden, m)
	if err != nil {
		return ports.ReciboNotificacion{}, err
	}
	if !domain.NotificacionRefValida(r.NotificacionRef) || r.ReciboRef == "" || !instanteReciboValido(r.InstanteUTC) {
		return ports.ReciboNotificacion{}, ports.ErrDependenciaNoDisponible
	}
	return r, nil
}

// ServicioBandejaNotificaciones: RRHH consulta lo que le han comunicado las
// personas de su circuito y marca una notificación atendida.
type ServicioBandejaNotificaciones struct {
	repositorio ports.RepositorioBandejaNotificaciones
	reloj       ports.Reloj
	zona        *time.Location
}

func NuevoServicioBandejaNotificaciones(repo ports.RepositorioBandejaNotificaciones, reloj ports.Reloj, zona *time.Location) (*ServicioBandejaNotificaciones, error) {
	if repo == nil || reloj == nil || zona == nil {
		return nil, ErrServiceDependencyRequired
	}
	return &ServicioBandejaNotificaciones{repositorio: repo, reloj: reloj, zona: zona}, nil
}

func (s *ServicioBandejaNotificaciones) contexto(ctx context.Context, orden ports.OrdenBandejaNotificaciones) (vecdomain.ContextoActor, string, error) {
	if s == nil || s.repositorio == nil || s.reloj == nil || s.zona == nil || ctx == nil {
		return vecdomain.ContextoActor{}, "", ports.ErrDependenciaNoDisponible
	}
	actor, err := orden.ContextoActor()
	if err != nil {
		return vecdomain.ContextoActor{}, "", err
	}
	empleado, _, err := empleadoVigente(actor, s.reloj)
	return actor, empleado, err
}

func (s *ServicioBandejaNotificaciones) ConsultarBandeja(ctx context.Context, orden ports.OrdenBandejaNotificaciones) (ports.BandejaNotificaciones, error) {
	actor, empleado, err := s.contexto(ctx, orden)
	if err != nil {
		return ports.BandejaNotificaciones{}, err
	}
	m := domain.MaterialConsultaNotificaciones{ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, EmpleadoRef: empleado, ZonaHoraria: s.zona.String()}
	if m.Validar() != nil {
		return ports.BandejaNotificaciones{}, ports.ErrDependenciaNoDisponible
	}
	b, err := s.repositorio.ConsultarBandeja(ctx, orden, m)
	if err != nil {
		return ports.BandejaNotificaciones{}, err
	}
	if b.Notificaciones == nil || len(b.Notificaciones) > maximoFilasNotificaciones {
		return ports.BandejaNotificaciones{}, ports.ErrDependenciaNoDisponible
	}
	for _, n := range b.Notificaciones {
		if n.EmpleadoRef == empleado || !domain.NotificacionRefValida(n.NotificacionRef) || !domain.TipoNotificacionValido(n.TipoRef) || n.TipoNombre == "" ||
			!domain.TextoNotificacionValido(n.Texto) || !domain.AdjuntoNotificacionValido(n.AdjuntoRef, n.AdjuntoSHA256) ||
			len([]rune(n.EmpleadoEtiqueta)) > maximoEtiquetaEmpleadoNotificar || n.RegistradaEnUTC.IsZero() || n.Atendida != (n.AtendidaEnUTC != nil) {
			return ports.BandejaNotificaciones{}, ports.ErrDependenciaNoDisponible
		}
	}
	return b, nil
}

func (s *ServicioBandejaNotificaciones) AtenderNotificacion(ctx context.Context, orden ports.OrdenBandejaNotificaciones, p ports.PeticionAtencionNotificacion) (ports.ReciboAtencionNotificacion, error) {
	actor, empleado, err := s.contexto(ctx, orden)
	if err != nil {
		return ports.ReciboAtencionNotificacion{}, err
	}
	m := domain.MaterialAtencionNotificacion{ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, EmpleadoRef: empleado,
		ClaveOperacion: p.ClaveOperacion, NotificacionRef: p.NotificacionRef, ZonaHoraria: s.zona.String()}
	if m.Validar() != nil {
		return ports.ReciboAtencionNotificacion{}, ports.ErrSolicitudCronosInvalida
	}
	r, err := s.repositorio.AtenderNotificacion(ctx, orden, m)
	if err != nil {
		return ports.ReciboAtencionNotificacion{}, err
	}
	if !domain.AtencionRefValida(r.AtencionRef) || r.NotificacionRef != m.NotificacionRef || r.ReciboRef == "" || !instanteReciboValido(r.InstanteUTC) {
		return ports.ReciboAtencionNotificacion{}, ports.ErrDependenciaNoDisponible
	}
	return r, nil
}

var (
	_ ports.CasoUsoNotificacionesPropias = (*ServicioNotificacionesPropias)(nil)
	_ ports.CasoUsoBandejaNotificaciones = (*ServicioBandejaNotificaciones)(nil)
)
