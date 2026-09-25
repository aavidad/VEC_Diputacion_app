package application

import (
	"context"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

const (
	AudienciaCorreccionMarcaje      = "vec_cronos_v1.correccion_marcaje.v1"
	AccionSolicitarCorreccion       = "cronos.correccion.solicitar"
	AccionDecidirCorreccion         = "cronos.correccion.responsable.decidir"
	AccionResolverCorreccionRRHH    = "cronos.correccion.rrhh.resolver"
	AccionAplicarCorreccion         = "cronos.correccion.aplicar"
	AccionRecuperarReciboCorreccion = "cronos.correccion.recibo.consultar"
)

type ServicioCorrecciones struct {
	repositorio ports.RepositorioCorrecciones
	reloj       ports.Reloj
}

func NuevoServicioCorrecciones(r ports.RepositorioCorrecciones, reloj ports.Reloj) (*ServicioCorrecciones, error) {
	if r == nil || reloj == nil {
		return nil, ports.ErrDependenciaNoDisponible
	}
	return &ServicioCorrecciones{repositorio: r, reloj: reloj}, nil
}

func (s *ServicioCorrecciones) SolicitarOlvido(ctx context.Context, orden ports.OrdenConsumoCorreccion, entrada ports.SolicitudOlvidoMarcaje) (ports.ReciboCorreccion, error) {
	actor, instante, err := s.actorVigente(ctx, orden)
	if err != nil {
		return ports.ReciboCorreccion{}, err
	}
	empleados, err := actor.Referencias(vecdomain.TipoReferenciaContextoActorEmpleado)
	if err != nil || len(empleados) != 1 {
		return ports.ReciboCorreccion{}, ports.ErrCorreccionNoAutorizada
	}
	solicitud := domain.SolicitudCorreccion{
		EmpleadoRef: empleados[0], ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef,
		ClaveOperacion: entrada.ClaveOperacion, MarcajeOriginalRef: entrada.MarcajeOriginalRef,
		HuecoDeclarado: entrada.HuecoDeclarado, Movimiento: entrada.Movimiento,
		FechaCivil: entrada.FechaCivil, HoraPretendida: entrada.HoraPretendida,
		MotivoCodigo: domain.MotivoOlvidoMarcaje, SolicitadaEnUTC: instante,
	}
	if err := solicitud.Validar(); err != nil {
		return ports.ReciboCorreccion{}, err
	}
	recibo, err := s.repositorio.SolicitarOlvido(ctx, solicitud, orden)
	if err != nil {
		return ports.ReciboCorreccion{}, err
	}
	if !reciboCorreccionValido(recibo, "correccion:cronos:"+entrada.ClaveOperacion, domain.CorreccionPendienteResponsable, 1) {
		return ports.ReciboCorreccion{}, ports.ErrDependenciaNoDisponible
	}
	return recibo, nil
}

func (s *ServicioCorrecciones) DecidirResponsable(ctx context.Context, orden ports.OrdenConsumoCorreccion, entrada ports.DecisionResponsableCorreccion) (ports.ReciboCorreccion, error) {
	return s.actuar(ctx, orden, entrada.SolicitudRef, entrada.ClaveOperacion, domain.PasoDecisionResponsable, entrada.Resultado, entrada.VersionEsperada)
}

func (s *ServicioCorrecciones) ResolverRRHH(ctx context.Context, orden ports.OrdenConsumoCorreccion, entrada ports.ResolucionRRHHCorreccion) (ports.ReciboCorreccion, error) {
	return s.actuar(ctx, orden, entrada.SolicitudRef, entrada.ClaveOperacion, domain.PasoResolucionRRHH, entrada.Resultado, entrada.VersionEsperada)
}

func (s *ServicioCorrecciones) AplicarResolucion(ctx context.Context, orden ports.OrdenConsumoCorreccion, entrada ports.AplicacionCorreccion) (ports.ReciboCorreccion, error) {
	return s.actuar(ctx, orden, entrada.SolicitudRef, entrada.ClaveOperacion, domain.PasoAplicacion, "", entrada.VersionEsperada)
}

func (s *ServicioCorrecciones) actuar(ctx context.Context, orden ports.OrdenConsumoCorreccion, solicitudRef, clave string, paso domain.PasoCorreccion, resultado domain.ResultadoCorreccion, version uint64) (ports.ReciboCorreccion, error) {
	actor, instante, err := s.actorVigente(ctx, orden)
	if err != nil {
		return ports.ReciboCorreccion{}, err
	}
	actuacion := domain.ActuacionCorreccion{
		SolicitudRef: solicitudRef, ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef,
		ClaveOperacion: clave, Paso: paso, Resultado: resultado,
		VersionEsperada: version, RegistradaEnUTC: instante,
	}
	if err := actuacion.Validar(); err != nil {
		return ports.ReciboCorreccion{}, err
	}
	recibo, err := s.repositorio.RegistrarActuacion(ctx, actuacion, orden)
	if err != nil {
		return ports.ReciboCorreccion{}, err
	}
	estado := estadoResultanteCorreccion(paso, resultado)
	if !reciboCorreccionValido(recibo, solicitudRef, estado, version+1) {
		return ports.ReciboCorreccion{}, ports.ErrDependenciaNoDisponible
	}
	return recibo, nil
}

func (s *ServicioCorrecciones) RecuperarRecibo(ctx context.Context, orden ports.OrdenConsumoCorreccion, clave ports.ClaveRecuperacionCorreccion) (ports.ReciboCorreccion, error) {
	if _, _, err := s.actorVigente(ctx, orden); err != nil {
		return ports.ReciboCorreccion{}, err
	}
	if err := (domain.ClaveRecuperacionCorreccion{SolicitudRef: clave.SolicitudRef, ClaveOperacion: clave.ClaveOperacion, Paso: clave.Paso}).Validar(); err != nil {
		return ports.ReciboCorreccion{}, err
	}
	recibo, err := s.repositorio.RecuperarRecibo(ctx, clave, orden)
	if err != nil {
		return ports.ReciboCorreccion{}, err
	}
	if !reciboRecuperadoValido(recibo, clave) {
		return ports.ReciboCorreccion{}, ports.ErrDependenciaNoDisponible
	}
	return recibo, nil
}

func (s *ServicioCorrecciones) actorVigente(ctx context.Context, orden ports.OrdenConsumoCorreccion) (vecdomain.ContextoActor, time.Time, error) {
	if s == nil || s.repositorio == nil || s.reloj == nil || ctx == nil || orden.ProveedorMaterial() == nil {
		return vecdomain.ContextoActor{}, time.Time{}, ports.ErrCorreccionNoAutorizada
	}
	actor, err := orden.ContextoActor()
	if err != nil {
		return vecdomain.ContextoActor{}, time.Time{}, ports.ErrCorreccionNoAutorizada
	}
	instante := s.reloj.AhoraUTC().UTC().Truncate(time.Microsecond)
	if instante.IsZero() || !actor.Instantanea.VigenteEn(instante) {
		return vecdomain.ContextoActor{}, time.Time{}, ports.ErrCorreccionNoAutorizada
	}
	return actor, instante, nil
}

func estadoResultanteCorreccion(paso domain.PasoCorreccion, resultado domain.ResultadoCorreccion) domain.EstadoCorreccion {
	switch paso {
	case domain.PasoDecisionResponsable:
		if resultado == domain.ResultadoFavorable {
			return domain.CorreccionPendienteRRHH
		}
		return domain.CorreccionDenegadaResponsable
	case domain.PasoResolucionRRHH:
		if resultado == domain.ResultadoFavorable {
			return domain.CorreccionPendienteAplicacion
		}
		return domain.CorreccionDenegadaRRHH
	case domain.PasoAplicacion:
		return domain.CorreccionAplicada
	default:
		return ""
	}
}

func reciboCorreccionValido(r ports.ReciboCorreccion, solicitudRef string, estado domain.EstadoCorreccion, version uint64) bool {
	return r.SolicitudRef == solicitudRef && r.Estado == estado && r.Version == version &&
		strings.HasPrefix(r.ActuacionRef, "correccion:actuacion:") &&
		strings.HasPrefix(r.ReciboRef, "recibo:cronos:") && instanteCorreccionReciboValido(r.InstanteUTC)
}

func reciboRecuperadoValido(r ports.ReciboCorreccion, clave ports.ClaveRecuperacionCorreccion) bool {
	if r.SolicitudRef != clave.SolicitudRef || !strings.HasPrefix(r.ActuacionRef, "correccion:actuacion:") ||
		!strings.HasPrefix(r.ReciboRef, "recibo:cronos:") || !instanteCorreccionReciboValido(r.InstanteUTC) {
		return false
	}
	switch clave.Paso {
	case domain.PasoSolicitudCorreccion:
		return r.Version == 1 && r.Estado == domain.CorreccionPendienteResponsable
	case domain.PasoDecisionResponsable:
		return r.Version == 2 && (r.Estado == domain.CorreccionPendienteRRHH || r.Estado == domain.CorreccionDenegadaResponsable)
	case domain.PasoResolucionRRHH:
		return r.Version == 3 && (r.Estado == domain.CorreccionPendienteAplicacion || r.Estado == domain.CorreccionDenegadaRRHH)
	case domain.PasoAplicacion:
		return r.Version == 4 && r.Estado == domain.CorreccionAplicada
	default:
		return false
	}
}

func instanteCorreccionReciboValido(i time.Time) bool {
	return !i.IsZero() && i.Location() == time.UTC && i.Nanosecond()%1000 == 0
}
