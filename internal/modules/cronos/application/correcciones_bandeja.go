package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

type RegistroDenegacionCorreccion interface {
	RegistrarDenegacionCorreccion(context.Context, vecdomain.ContextoActor, string, string) error
}

func rechazoCorreccionAuditado(ctx context.Context, actor vecdomain.ContextoActor, accion, finalidad string, err error) error {
	if errors.Is(err, ports.ErrResolucionNoCompetente) || errors.Is(err, ports.ErrCorreccionNoAutorizada) {
		registro, ok := ctx.Value(claveRegistroDenegacionVinculo{}).(RegistroDenegacionVinculo)
		if !ok || registroDenegacionVinculoNulo(registro) || accion == "" || finalidad == "" {
			return ports.ErrDependenciaNoDisponible
		}
		correccion, ok := registro.(RegistroDenegacionCorreccion)
		if !ok || correccion == nil {
			return ports.ErrDependenciaNoDisponible
		}
		if registroErr := correccion.RegistrarDenegacionCorreccion(ctx, actor, accion, finalidad); registroErr != nil {
			return errors.Join(ports.ErrDependenciaNoDisponible, registroErr)
		}
	}
	return err
}

const (
	AudienciaBandejaCorrecciones           = "vec_cronos_v1.correccion.bandeja.v1"
	AccionBandejaCorrecciones              = "cronos.correccion.bandeja.consultar"
	FinalidadBandejaCorrecciones           = "consultar_bandeja_correcciones"
	AudienciaDecisionResponsableCorreccion = "vec_cronos_v1.correccion.responsable.decidir.v1"
	FinalidadDecisionResponsableCorreccion = "decidir_correccion_responsable"
	AudienciaResolucionRRHHCorreccion      = "vec_cronos_v1.correccion.rrhh.resolver.v1"
	FinalidadResolucionRRHHCorreccion      = "resolver_correccion_rrhh"
	AudienciaAplicacionCorreccion          = "vec_cronos_v1.correccion.aplicar.v1"
	FinalidadAplicacionCorreccion          = "aplicar_correccion"
	AudienciaHistoriaCorreccionPropia      = "vec_cronos_v1.correccion.historia.propia.v1"
	AccionHistoriaCorreccionPropia         = "cronos.correccion.historia.propia.consultar"
	FinalidadHistoriaCorreccionPropia      = "consultar_historia_correccion_propia"
	AudienciaHistoriaCorreccionEquipo      = "vec_cronos_v1.correccion.historia.equipo.v1"
	AccionHistoriaCorreccionEquipo         = "cronos.correccion.historia.equipo.consultar"
	FinalidadHistoriaCorreccionEquipo      = "consultar_historia_correccion_equipo"
)

func ContratoActuacionCorreccion(paso domain.PasoCorreccion) (accion, finalidad, audiencia string, ok bool) {
	switch paso {
	case domain.PasoDecisionResponsable:
		return AccionDecidirCorreccion, FinalidadDecisionResponsableCorreccion, AudienciaDecisionResponsableCorreccion, true
	case domain.PasoResolucionRRHH:
		return AccionResolverCorreccionRRHH, FinalidadResolucionRRHHCorreccion, AudienciaResolucionRRHHCorreccion, true
	case domain.PasoAplicacion:
		return AccionAplicarCorreccion, FinalidadAplicacionCorreccion, AudienciaAplicacionCorreccion, true
	default:
		return "", "", "", false
	}
}

func recursoDecisionCorreccion(ref, tipo, actor, paso string, canonico []byte) (vecdomain.RecursoAutorizable, error) {
	h := sha256.Sum256(canonico)
	r := vecdomain.RecursoAutorizable{Referencia: ref, ModuloID: "cronos", Tipo: tipo,
		Ambitos:   map[string]string{"persona_ref": actor, "paso_resolucion": paso},
		Atributos: map[string]string{"material_sha256": hex.EncodeToString(h[:])}}
	if r.Validar() != nil {
		return vecdomain.RecursoAutorizable{}, ErrContextoMarcajeNoAcreditado
	}
	return r, nil
}

func RecursoBandejaCorrecciones(m domain.MaterialBandejaCorrecciones) (vecdomain.RecursoAutorizable, error) {
	b, err := m.Canonico()
	if err != nil {
		return vecdomain.RecursoAutorizable{}, err
	}
	defer clear(b)
	return recursoDecisionCorreccion("bandeja:cronos:correcciones:"+m.Paso, "bandeja_correcciones", m.ActorRef, m.Paso, b)
}

func RecursoActuacionCorreccion(m domain.MaterialActuacionCorreccion) (vecdomain.RecursoAutorizable, error) {
	b, err := m.Canonico()
	if err != nil {
		return vecdomain.RecursoAutorizable{}, err
	}
	defer clear(b)
	return recursoDecisionCorreccion("correccion:cronos:actuacion:"+m.ClaveOperacion, "actuacion_correccion", m.ActorRef, m.PasoAmbito(), b)
}

func ContratoHistoriaCorreccion(m domain.MaterialHistoriaCorreccion) (accion, finalidad, audiencia string, ok bool) {
	switch m.Vista {
	case "propia":
		return AccionHistoriaCorreccionPropia, FinalidadHistoriaCorreccionPropia, AudienciaHistoriaCorreccionPropia, true
	case "equipo":
		return AccionHistoriaCorreccionEquipo, FinalidadHistoriaCorreccionEquipo, AudienciaHistoriaCorreccionEquipo, true
	default:
		return "", "", "", false
	}
}

func RecursoHistoriaCorreccion(m domain.MaterialHistoriaCorreccion) (vecdomain.RecursoAutorizable, error) {
	b, err := m.Canonico()
	if err != nil {
		return vecdomain.RecursoAutorizable{}, err
	}
	defer clear(b)
	ref := "historia:cronos:" + m.SolicitudRef
	if m.Vista == "propia" {
		return recursoEmpleado(ref, "historia_correccion_propia", m.EmpleadoRef, b)
	}
	return recursoDecisionCorreccion(ref, "historia_correccion_equipo", m.ActorRef, m.Paso, b)
}

type ServicioDecisionCorrecciones struct {
	repo  ports.RepositorioDecisionCorrecciones
	reloj ports.Reloj
	zona  *time.Location
}

func NuevoServicioDecisionCorrecciones(repo ports.RepositorioDecisionCorrecciones, reloj ports.Reloj, zona *time.Location) (*ServicioDecisionCorrecciones, error) {
	if repo == nil || reloj == nil || zona == nil {
		return nil, ports.ErrDependenciaNoDisponible
	}
	if zona.String() != domain.ZonaSaldoPeninsula && zona.String() != domain.ZonaSaldoCanarias {
		return nil, ports.ErrDependenciaNoDisponible
	}
	return &ServicioDecisionCorrecciones{repo: repo, reloj: reloj, zona: zona}, nil
}

func (s *ServicioDecisionCorrecciones) contexto(ctx context.Context, orden ports.OrdenConsumoCorreccion) (vecdomain.ContextoActor, string, error) {
	if s == nil || s.repo == nil || s.reloj == nil || s.zona == nil || ctx == nil || ctx.Err() != nil {
		return vecdomain.ContextoActor{}, "", ports.ErrDependenciaNoDisponible
	}
	actor, err := orden.ContextoActor()
	if err != nil {
		return vecdomain.ContextoActor{}, "", ports.ErrCorreccionNoAutorizada
	}
	empleado, _, err := empleadoVigente(ctx, actor, s.reloj)
	return actor, empleado, err
}

func (s *ServicioDecisionCorrecciones) ConsultarBandejaCorrecciones(ctx context.Context, orden ports.OrdenConsumoCorreccion, paso string) (ports.BandejaCorrecciones, error) {
	actor, empleado, err := s.contexto(ctx, orden)
	if err != nil {
		return ports.BandejaCorrecciones{}, err
	}
	m := domain.MaterialBandejaCorrecciones{ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, EmpleadoRef: empleado, Paso: paso, ZonaHoraria: s.zona.String()}
	if m.Validar() != nil {
		return ports.BandejaCorrecciones{}, domain.ErrCorreccionInvalida
	}
	b, err := s.repo.ConsultarBandejaCorrecciones(ctx, orden, m)
	if err != nil {
		return ports.BandejaCorrecciones{}, rechazoCorreccionAuditado(ctx, actor, AccionBandejaCorrecciones, FinalidadBandejaCorrecciones, err)
	}
	if b.Paso != paso || b.Pendientes == nil || len(b.Pendientes) > 500 {
		return ports.BandejaCorrecciones{}, ports.ErrDependenciaNoDisponible
	}
	for _, p := range b.Pendientes {
		if p.EmpleadoRef == empleado || p.SolicitudRef == "" || p.EmpleadoRef == "" || p.Version == 0 || p.SolicitadaEnUTC.IsZero() ||
			(paso == "responsable" && (p.Estado != domain.CorreccionPendienteResponsable || p.Version != 1)) ||
			(paso == "administracion" && p.Estado != domain.CorreccionPendienteRRHH && p.Estado != domain.CorreccionPendienteAplicacion) {
			return ports.BandejaCorrecciones{}, ports.ErrDependenciaNoDisponible
		}
	}
	return b, nil
}

func (s *ServicioDecisionCorrecciones) RegistrarDecisionCorreccion(ctx context.Context, orden ports.OrdenConsumoCorreccion, p ports.PeticionActuacionCorreccion) (ports.ReciboCorreccion, error) {
	actor, empleado, err := s.contexto(ctx, orden)
	if err != nil {
		return ports.ReciboCorreccion{}, err
	}
	m := domain.MaterialActuacionCorreccion{ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, EmpleadoRef: empleado,
		SolicitudRef: p.SolicitudRef, ClaveOperacion: p.ClaveOperacion, Paso: p.Paso, Resultado: p.Resultado,
		Motivo: p.Motivo, VersionEsperada: p.VersionEsperada, ZonaHoraria: s.zona.String()}
	if m.Validar() != nil {
		return ports.ReciboCorreccion{}, domain.ErrCorreccionInvalida
	}
	r, err := s.repo.RegistrarDecisionCorreccion(ctx, orden, m)
	if err != nil {
		accion, finalidad, _, ok := ContratoActuacionCorreccion(m.Paso)
		if !ok {
			return ports.ReciboCorreccion{}, ports.ErrDependenciaNoDisponible
		}
		return ports.ReciboCorreccion{}, rechazoCorreccionAuditado(ctx, actor, accion, finalidad, err)
	}
	esperado := estadoResultanteCorreccion(m.Paso, m.Resultado)
	if !reciboCorreccionValido(r, m.SolicitudRef, esperado, m.VersionEsperada+1) {
		return ports.ReciboCorreccion{}, ports.ErrDependenciaNoDisponible
	}
	return r, nil
}

func (s *ServicioDecisionCorrecciones) ConsultarHistoriaCorreccion(ctx context.Context, orden ports.OrdenConsumoCorreccion, p ports.PeticionHistoriaCorreccion) (ports.HistoriaCorreccion, error) {
	actor, empleado, err := s.contexto(ctx, orden)
	if err != nil {
		return ports.HistoriaCorreccion{}, err
	}
	m := domain.MaterialHistoriaCorreccion{ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, EmpleadoRef: empleado,
		SolicitudRef: p.SolicitudRef, Vista: p.Vista, Paso: p.Paso, ZonaHoraria: s.zona.String()}
	if m.Validar() != nil {
		return ports.HistoriaCorreccion{}, domain.ErrCorreccionInvalida
	}
	h, err := s.repo.ConsultarHistoriaCorreccion(ctx, orden, m)
	if err != nil {
		accion, finalidad, _, ok := ContratoHistoriaCorreccion(m)
		if !ok {
			return ports.HistoriaCorreccion{}, ports.ErrDependenciaNoDisponible
		}
		return ports.HistoriaCorreccion{}, rechazoCorreccionAuditado(ctx, actor, accion, finalidad, err)
	}
	if h.SolicitudRef != m.SolicitudRef || h.EmpleadoRef == "" || h.Version == 0 || h.Actuaciones == nil || h.Asientos == nil ||
		(m.Vista == "propia" && h.EmpleadoRef != empleado) || (m.Vista == "equipo" && h.EmpleadoRef == empleado) ||
		len(h.Actuaciones) != int(h.Version) || len(h.Asientos) > 2 {
		return ports.HistoriaCorreccion{}, ports.ErrDependenciaNoDisponible
	}
	for i, a := range h.Actuaciones {
		if a.Version != uint64(i+1) || (m.Vista == "propia" && (a.ActorRef != "" || a.PerfilRef != "")) ||
			(m.Vista == "equipo" && (a.ActorRef == "" || a.PerfilRef == "")) {
			return ports.HistoriaCorreccion{}, ports.ErrDependenciaNoDisponible
		}
	}
	return h, nil
}
