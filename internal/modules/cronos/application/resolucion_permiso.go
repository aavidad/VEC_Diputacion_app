package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

// Contrato V3 de la resolución de permisos y de los avisos (AD3-57). La
// función durable de cronos_v1 000009 comprueba exactamente estos valores.
// Quien resuelve se autoriza sobre los ámbitos {persona_ref, paso_resolucion}:
// la concesión del perfil decide qué paso puede resolver; el circuito
// publicado en Cronos sólo restringe a qué personas.
const (
	AudienciaBandejaPermisos  = "vec_cronos_v1.permisos_bandeja.consultar.v1"
	AccionConsultarBandeja    = "cronos.permisos.bandeja.consultar"
	FinalidadConsultarBandeja = "consultar_bandeja_permisos"

	AudienciaResolucionPermiso = "vec_cronos_v1.permiso.resolver.v1"
	AccionResolverPermiso      = "cronos.permiso.resolver"
	FinalidadResolverPermiso   = "resolver_permiso"

	AudienciaConsultaAvisosPropios  = "vec_cronos_v1.avisos_propio.consultar.v1"
	AccionConsultarAvisosPropios    = "cronos.avisos.propio.consultar"
	FinalidadConsultarAvisosPropios = "consultar_avisos_propio"

	AudienciaArchivoAvisoPropio  = "vec_cronos_v1.aviso_propio.archivar.v1"
	AccionArchivarAvisoPropio    = "cronos.aviso.propio.archivar"
	FinalidadArchivarAvisoPropio = "archivar_aviso_propio"
)

const maximoFilasResolucion = 500

// recursoResolutor liga la operación de quien resuelve a su persona, al paso
// y a la huella del material exacto que recibirá la función durable.
func recursoResolutor(referencia, tipo, persona string, paso domain.PasoPermiso, canonico []byte) (vecdomain.RecursoAutorizable, error) {
	h := sha256.Sum256(canonico)
	r := vecdomain.RecursoAutorizable{
		Referencia: referencia, ModuloID: "cronos", Tipo: tipo,
		Ambitos:   map[string]string{"persona_ref": persona, "paso_resolucion": string(paso)},
		Atributos: map[string]string{"material_sha256": hex.EncodeToString(h[:])},
	}
	if r.Validar() != nil {
		return vecdomain.RecursoAutorizable{}, ErrContextoMarcajeNoAcreditado
	}
	return r, nil
}

func RecursoBandejaPermisos(m domain.MaterialBandejaPermisos) (vecdomain.RecursoAutorizable, error) {
	canonico, err := m.Canonico()
	if err != nil {
		return vecdomain.RecursoAutorizable{}, ErrContextoMarcajeNoAcreditado
	}
	return recursoResolutor(domain.BandejaPermisosRef(m.Paso), "bandeja_permisos", m.ActorRef, m.Paso, canonico)
}

func RecursoResolucionPermiso(m domain.MaterialResolucionPermiso) (vecdomain.RecursoAutorizable, error) {
	canonico, err := m.Canonico()
	if err != nil {
		return vecdomain.RecursoAutorizable{}, ErrContextoMarcajeNoAcreditado
	}
	return recursoResolutor(domain.ResolucionPermisoRef(m.ClaveOperacion), "resolucion_permiso", m.ActorRef, m.Paso, canonico)
}

func RecursoConsultaAvisosPropios(m domain.MaterialConsultaAvisosPropios) (vecdomain.RecursoAutorizable, error) {
	canonico, err := m.Canonico()
	if err != nil {
		return vecdomain.RecursoAutorizable{}, ErrContextoMarcajeNoAcreditado
	}
	return recursoEmpleado("avisos:cronos:"+m.EmpleadoRef, "avisos_propio", m.EmpleadoRef, canonico)
}

func RecursoArchivoAvisoPropio(m domain.MaterialArchivoAvisoPropio) (vecdomain.RecursoAutorizable, error) {
	canonico, err := m.Canonico()
	if err != nil {
		return vecdomain.RecursoAutorizable{}, ErrContextoMarcajeNoAcreditado
	}
	return recursoEmpleado(domain.ArchivoAvisoRef(m.ClaveOperacion), "archivo_aviso", m.EmpleadoRef, canonico)
}

// ServicioResolucionPermisos presenta la bandeja de un paso y registra la
// resolución de quien resuelve. No decide competencia: la acreditan la
// concesión V3 y el circuito publicado dentro de la función durable.
type ServicioResolucionPermisos struct {
	repositorio ports.RepositorioResolucionPermisos
	reloj       ports.Reloj
	zona        *time.Location
}

func NuevoServicioResolucionPermisos(repo ports.RepositorioResolucionPermisos, reloj ports.Reloj, zona *time.Location) (*ServicioResolucionPermisos, error) {
	if repo == nil || reloj == nil || zona == nil {
		return nil, ErrServiceDependencyRequired
	}
	return &ServicioResolucionPermisos{repositorio: repo, reloj: reloj, zona: zona}, nil
}

func (s *ServicioResolucionPermisos) contexto(ctx context.Context, orden ports.OrdenResolucionPermisos) (vecdomain.ContextoActor, string, error) {
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

func (s *ServicioResolucionPermisos) ConsultarBandeja(ctx context.Context, orden ports.OrdenResolucionPermisos, paso domain.PasoPermiso) (ports.BandejaPermisos, error) {
	actor, empleado, err := s.contexto(ctx, orden)
	if err != nil {
		return ports.BandejaPermisos{}, err
	}
	m := domain.MaterialBandejaPermisos{ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, EmpleadoRef: empleado, Paso: paso, ZonaHoraria: s.zona.String()}
	if m.Validar() != nil {
		return ports.BandejaPermisos{}, ports.ErrSolicitudCronosInvalida
	}
	b, err := s.repositorio.ConsultarBandeja(ctx, orden, m)
	if err != nil {
		return ports.BandejaPermisos{}, err
	}
	if !bandejaCoherente(b, paso, empleado) {
		return ports.BandejaPermisos{}, ports.ErrDependenciaNoDisponible
	}
	return b, nil
}

// bandejaCoherente rechaza una fuente que devuelva lo propio, un estado
// que no corresponda al paso o datos incompletos. Circuito es el aplicado
// (cronos_v1 000010): J-A por defecto y A sólo con marca directa; lo
// pendiente de asignación sólo aparece a RRHH, solicitado y en J-A.
func bandejaCoherente(b ports.BandejaPermisos, paso domain.PasoPermiso, propio string) bool {
	if b.Paso != paso || b.Pendientes == nil || len(b.Pendientes) > maximoFilasResolucion {
		return false
	}
	for _, p := range b.Pendientes {
		jefatura := p.Circuito == domain.CircuitoResponsableAdministracion
		pasoValido := (paso == domain.PasoResponsable && jefatura && p.Estado == domain.EstadoPermisoSolicitado && !p.PendienteAsignacion) ||
			(paso == domain.PasoAdministracion && ((p.Circuito == domain.CircuitoAdministracion && p.Estado == domain.EstadoPermisoSolicitado && !p.PendienteAsignacion) ||
				(jefatura && p.Estado == domain.EstadoPermisoPendienteAdministracion && !p.PendienteAsignacion) ||
				(jefatura && p.Estado == domain.EstadoPermisoSolicitado && p.PendienteAsignacion)))
		if !pasoValido || p.EmpleadoRef == propio || !domain.SolicitudPermisoRefValida(p.SolicitudRef) || p.Version < 1 || p.Cantidad < 1 ||
			(p.Unidad != domain.LeaveUnitDay && p.Unidad != domain.LeaveUnitHour) || p.Desde == "" || p.Hasta < p.Desde || p.SolicitadaEnUTC.IsZero() {
			return false
		}
	}
	return true
}

func (s *ServicioResolucionPermisos) ResolverPermiso(ctx context.Context, orden ports.OrdenResolucionPermisos, p ports.PeticionResolucionPermiso) (ports.ReciboResolucionPermiso, error) {
	actor, empleado, err := s.contexto(ctx, orden)
	if err != nil {
		return ports.ReciboResolucionPermiso{}, err
	}
	m := domain.MaterialResolucionPermiso{
		ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, EmpleadoRef: empleado,
		ClaveOperacion: p.ClaveOperacion, SolicitudRef: p.SolicitudRef, Paso: p.Paso, Decision: p.Decision,
		Motivo: strings.TrimSpace(p.Motivo), VersionEsperada: p.VersionEsperada, ZonaHoraria: s.zona.String(),
	}
	if m.Validar() != nil {
		return ports.ReciboResolucionPermiso{}, ports.ErrSolicitudCronosInvalida
	}
	r, err := s.repositorio.ResolverPermiso(ctx, orden, m)
	if err != nil {
		return ports.ReciboResolucionPermiso{}, err
	}
	esperado, _ := domain.EstadoTrasResolucion(m.Paso, m.Decision)
	if r.ResolucionRef != domain.ResolucionPermisoRef(m.ClaveOperacion) || r.SolicitudRef != m.SolicitudRef || r.Estado != esperado ||
		r.Version != m.VersionEsperada+1 || r.ReciboRef == "" || r.InstanteUTC.IsZero() || r.InstanteUTC.Location() != time.UTC {
		return ports.ReciboResolucionPermiso{}, ports.ErrDependenciaNoDisponible
	}
	return r, nil
}

// ServicioAvisosPropios lista los avisos de resolución de la persona y
// archiva uno. El texto visible se compone en la interfaz a partir de los
// datos de la resolución; nunca procede del cliente.
type ServicioAvisosPropios struct {
	repositorio ports.RepositorioAvisosPropios
	reloj       ports.Reloj
	zona        *time.Location
}

func NuevoServicioAvisosPropios(repo ports.RepositorioAvisosPropios, reloj ports.Reloj, zona *time.Location) (*ServicioAvisosPropios, error) {
	if repo == nil || reloj == nil || zona == nil {
		return nil, ErrServiceDependencyRequired
	}
	return &ServicioAvisosPropios{repositorio: repo, reloj: reloj, zona: zona}, nil
}

func (s *ServicioAvisosPropios) contexto(ctx context.Context, orden ports.OrdenAvisosPropios) (vecdomain.ContextoActor, string, error) {
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

func (s *ServicioAvisosPropios) ConsultarAvisos(ctx context.Context, orden ports.OrdenAvisosPropios) (ports.ConsultaAvisosPropios, error) {
	actor, empleado, err := s.contexto(ctx, orden)
	if err != nil {
		return ports.ConsultaAvisosPropios{}, err
	}
	m := domain.MaterialConsultaAvisosPropios{ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, EmpleadoRef: empleado, ZonaHoraria: s.zona.String()}
	if m.Validar() != nil {
		return ports.ConsultaAvisosPropios{}, ports.ErrDependenciaNoDisponible
	}
	c, err := s.repositorio.ConsultarAvisos(ctx, orden, m)
	if err != nil {
		return ports.ConsultaAvisosPropios{}, err
	}
	if c.Avisos == nil || len(c.Avisos) > maximoFilasResolucion {
		return ports.ConsultaAvisosPropios{}, ports.ErrDependenciaNoDisponible
	}
	for _, a := range c.Avisos {
		if !domain.AvisoRefValida(a.AvisoRef) || !domain.SolicitudPermisoRefValida(a.SolicitudRef) ||
			(a.Estado != domain.EstadoPermisoConcedido && a.Estado != domain.EstadoPermisoDenegado) ||
			(a.Estado == domain.EstadoPermisoDenegado && a.Motivo == "") || a.ResueltoEnUTC.IsZero() || a.Archivado != (a.ArchivadoEnUTC != nil) {
			return ports.ConsultaAvisosPropios{}, ports.ErrDependenciaNoDisponible
		}
	}
	return c, nil
}

func (s *ServicioAvisosPropios) ArchivarAviso(ctx context.Context, orden ports.OrdenAvisosPropios, p ports.PeticionArchivoAviso) (ports.ReciboArchivoAviso, error) {
	actor, empleado, err := s.contexto(ctx, orden)
	if err != nil {
		return ports.ReciboArchivoAviso{}, err
	}
	m := domain.MaterialArchivoAvisoPropio{ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, EmpleadoRef: empleado,
		ClaveOperacion: p.ClaveOperacion, AvisoRef: p.AvisoRef, ZonaHoraria: s.zona.String()}
	if m.Validar() != nil {
		return ports.ReciboArchivoAviso{}, ports.ErrSolicitudCronosInvalida
	}
	r, err := s.repositorio.ArchivarAviso(ctx, orden, m)
	if err != nil {
		return ports.ReciboArchivoAviso{}, err
	}
	if r.ArchivoRef != domain.ArchivoAvisoRef(m.ClaveOperacion) || r.AvisoRef != m.AvisoRef || r.ReciboRef == "" ||
		r.InstanteUTC.IsZero() || r.InstanteUTC.Location() != time.UTC {
		return ports.ReciboArchivoAviso{}, ports.ErrDependenciaNoDisponible
	}
	return r, nil
}

var (
	_ ports.CasoUsoResolucionPermisos = (*ServicioResolucionPermisos)(nil)
	_ ports.CasoUsoAvisosPropios      = (*ServicioAvisosPropios)(nil)
)
