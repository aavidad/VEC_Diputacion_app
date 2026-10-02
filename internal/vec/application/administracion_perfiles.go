package application

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

var ErrAdministracionPerfilesNoConfigurada = errors.New("vec: administracion de perfiles no configurada")

// ServicioAdministracionPerfiles coordina un catalogo publicado y una sola
// autoridad durable. No construye decisiones de autorizacion ni publica
// concesiones; el puerto de actos ejecuta PDP V3, CAS, auditoria y recibo.
type ServicioAdministracionPerfiles struct {
	catalogo ports.CatalogoRolesAdministrables
	actos    ports.AutoridadActosAdministracionPerfiles
	reloj    ports.Reloj
}

func NuevoServicioAdministracionPerfiles(
	catalogo ports.CatalogoRolesAdministrables,
	actos ports.AutoridadActosAdministracionPerfiles,
	reloj ports.Reloj,
) (*ServicioAdministracionPerfiles, error) {
	if catalogo == nil || actos == nil || reloj == nil {
		return nil, ErrAdministracionPerfilesNoConfigurada
	}
	return &ServicioAdministracionPerfiles{catalogo: catalogo, actos: actos, reloj: reloj}, nil
}

func (s *ServicioAdministracionPerfiles) AplicarOrdinario(
	ctx context.Context, solicitud domain.SolicitudActoAdministracionPerfiles,
) (domain.ReciboAdministracionPerfiles, error) {
	if s == nil || s.catalogo == nil || s.actos == nil || s.reloj == nil {
		return domain.ReciboAdministracionPerfiles{}, ErrAdministracionPerfilesNoConfigurada
	}
	if ctx == nil || ctx.Err() != nil || solicitud.Validar() != nil ||
		solicitud.Clase != domain.ClaseControlPerfilOrdinario ||
		!domain.ReferenciaAdministracionPerfilesValida(solicitud.OperacionRef, "acto_admin:") {
		return domain.ReciboAdministracionPerfiles{}, domain.ErrActoAdministracionPerfilesInvalido
	}
	rol, err := s.catalogo.ResolverRolAdministrable(ctx, solicitud.RolVersionRef)
	if err != nil {
		return domain.ReciboAdministracionPerfiles{}, err
	}
	if rol.ValidarEn(s.reloj.Ahora()) != nil || rol.VersionRef != solicitud.RolVersionRef ||
		rol.Clase != solicitud.Clase || (rol.UnidadRequerida && solicitud.Objetivo.UnidadRef == "") {
		return domain.ReciboAdministracionPerfiles{}, domain.ErrActoAdministracionPerfilesInvalido
	}
	recibo, err := s.actos.AplicarActoOrdinario(ctx, solicitud)
	if err != nil {
		return domain.ReciboAdministracionPerfiles{}, err
	}
	if recibo.Validar() != nil || recibo.OperacionRef != solicitud.OperacionRef ||
		recibo.ObjetivoPersonaRef != solicitud.Objetivo.PersonaRef ||
		recibo.PerfilRef != solicitud.Objetivo.PerfilRef ||
		recibo.VinculoRef != solicitud.Objetivo.VinculoRef || recibo.UnidadRef != solicitud.Objetivo.UnidadRef ||
		recibo.ReferenciaActo != solicitud.ReferenciaActo {
		return domain.ReciboAdministracionPerfiles{}, domain.ErrActoAdministracionPerfilesInvalido
	}
	if solicitud.Operacion == domain.OperacionOtorgarPerfil &&
		recibo.EstadoPosterior != domain.EstadoVinculoContextoActorActivo {
		return domain.ReciboAdministracionPerfiles{}, domain.ErrActoAdministracionPerfilesInvalido
	}
	if solicitud.Operacion == domain.OperacionRevocarPerfil &&
		(recibo.EstadoPosterior != domain.EstadoVinculoContextoActorRevocado ||
			recibo.PerfilRef != solicitud.Objetivo.PerfilRef ||
			recibo.VinculoRef != solicitud.Objetivo.VinculoRef ||
			recibo.VersionPosterior <= solicitud.Objetivo.VinculoVersion) {
		return domain.ReciboAdministracionPerfiles{}, domain.ErrActoAdministracionPerfilesInvalido
	}
	return recibo, nil
}

func (s *ServicioAdministracionPerfiles) ProponerSensible(
	ctx context.Context, solicitud domain.SolicitudActoAdministracionPerfiles,
) (ports.PropuestaAdministracionPerfiles, error) {
	if s == nil || s.catalogo == nil || s.actos == nil || s.reloj == nil {
		return ports.PropuestaAdministracionPerfiles{}, ErrAdministracionPerfilesNoConfigurada
	}
	if ctx == nil || ctx.Err() != nil || solicitud.Validar() != nil ||
		!solicitud.Clase.RequiereDobleControl() ||
		!domain.ReferenciaAdministracionPerfilesValida(solicitud.OperacionRef, "propuesta_admin:") {
		return ports.PropuestaAdministracionPerfiles{}, domain.ErrControlAdministracionPerfilesInvalido
	}
	rol, err := s.catalogo.ResolverRolAdministrable(ctx, solicitud.RolVersionRef)
	if err != nil {
		return ports.PropuestaAdministracionPerfiles{}, err
	}
	if rol.ValidarEn(s.reloj.Ahora()) != nil || rol.VersionRef != solicitud.RolVersionRef ||
		rol.Clase != solicitud.Clase || (rol.UnidadRequerida && solicitud.Objetivo.UnidadRef == "") {
		return ports.PropuestaAdministracionPerfiles{}, domain.ErrControlAdministracionPerfilesInvalido
	}
	propuesta, err := s.actos.ProponerActoSensible(ctx, solicitud)
	if err != nil {
		return ports.PropuestaAdministracionPerfiles{}, err
	}
	if propuesta.ValidarPara(solicitud) != nil {
		return ports.PropuestaAdministracionPerfiles{}, domain.ErrControlAdministracionPerfilesInvalido
	}
	if !propuesta.CaducaEn.After(s.reloj.Ahora()) {
		return ports.PropuestaAdministracionPerfiles{}, domain.ErrControlAdministracionPerfilesInvalido
	}
	return propuesta, nil
}

func (s *ServicioAdministracionPerfiles) CerrarPropuestaSensible(
	ctx context.Context, solicitud domain.SolicitudCierrePropuestaAdministracionPerfiles,
) (ports.CierrePropuestaAdministracionPerfiles, error) {
	if s == nil || s.actos == nil {
		return ports.CierrePropuestaAdministracionPerfiles{}, ErrAdministracionPerfilesNoConfigurada
	}
	if ctx == nil || ctx.Err() != nil || solicitud.Validar() != nil {
		return ports.CierrePropuestaAdministracionPerfiles{}, domain.ErrControlAdministracionPerfilesInvalido
	}
	cierre, err := s.actos.CerrarPropuestaSensible(ctx, solicitud)
	if err != nil {
		return ports.CierrePropuestaAdministracionPerfiles{}, err
	}
	if cierre.ValidarPara(solicitud) != nil {
		return ports.CierrePropuestaAdministracionPerfiles{}, domain.ErrControlAdministracionPerfilesInvalido
	}
	return cierre, nil
}
