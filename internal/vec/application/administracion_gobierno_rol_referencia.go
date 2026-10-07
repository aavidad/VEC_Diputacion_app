package application

import (
	"context"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// ProponerGobiernoRolNuevo exige el puerto nominal tipado. La preparación y
// comprobación ADMIN son las del servicio existente; la autoridad señala
// replay sólo tras cotejar el material y auditar el acceso actual.
func (s *ServicioAdministracionPerfiles) ProponerGobiernoRolNuevo(ctx context.Context,
	solicitud domain.SolicitudPropuestaGobiernoPerfil) (ports.ResultadoPropuestaGobiernoRolNuevo, error) {
	if s == nil || solicitud.Intencion.Operacion != domain.OperacionCrearPerfilGobernado {
		return ports.ResultadoPropuestaGobiernoRolNuevo{}, domain.ErrPlanGobiernoPerfilInvalido
	}
	if _, ok := s.actos.(ports.AutoridadPropuestaGobiernoRolNuevoRecuperable); !ok {
		return ports.ResultadoPropuestaGobiernoRolNuevo{}, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	return s.proponerGobiernoPerfilConEstado(ctx, solicitud)
}

// CerrarGobiernoRolPorReferencia conserva el servicio ADMIN existente. La
// autoridad durable recupera proponente/objetivo en la transacción V3; el
// servicio comprueba después el resultado contra ese material original.
func (s *ServicioAdministracionPerfiles) CerrarGobiernoRolPorReferencia(ctx context.Context,
	solicitud domain.SolicitudCierreGobiernoRolPorReferencia) (domain.CierreGobiernoPerfil, error) {
	var vacio domain.CierreGobiernoPerfil
	if s == nil || s.catalogo == nil || s.actos == nil || s.reloj == nil {
		return vacio, ErrAdministracionPerfilesNoConfigurada
	}
	if ctx == nil || solicitud.Validar() != nil ||
		solicitud.Evidencia.ValidarEn(solicitud.Aprobador, s.reloj.Ahora()) != nil {
		return vacio, domain.ErrPlanGobiernoPerfilInvalido
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	if err := s.validarAdministrador(ctx, solicitud.InstantaneaAutorizacion); err != nil {
		return vacio, err
	}
	autoridad, ok := s.actos.(ports.AutoridadCierreGobiernoRolPorReferencia)
	if !ok {
		return vacio, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	cierre, err := autoridad.CerrarGobiernoRolPorReferencia(ctx, solicitud)
	if err != nil {
		return vacio, err
	}
	completa, err := solicitud.CompletarCierreGobiernoRolConMaterial(cierre.Material)
	if err != nil || cierre.ValidarPara(completa) != nil || cierre.ConfirmadoEn.After(s.reloj.Ahora()) {
		return vacio, domain.ErrPlanGobiernoPerfilInvalido
	}
	cierre.Material.Plan = cierre.Material.Plan.Copia()
	if cierre.Recibo != nil {
		copia := cierre.Recibo.Copia()
		cierre.Recibo = &copia
	}
	return cierre, nil
}
