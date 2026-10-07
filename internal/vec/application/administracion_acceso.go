package application

import (
	"context"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// ProponerCambioAccesoPersona reutiliza el servicio y su catálogo central.
// No acepta cuentas enumeradas por el solicitante y no cambia ningún acceso.
func (s *ServicioAdministracionPerfiles) ProponerCambioAccesoPersona(ctx context.Context, solicitud domain.SolicitudPropuestaAdministracionAcceso) (domain.PropuestaAdministracionAcceso, error) {
	if s == nil || s.catalogo == nil || s.actos == nil || s.reloj == nil {
		return domain.PropuestaAdministracionAcceso{}, ErrAdministracionPerfilesNoConfigurada
	}
	if ctx == nil || solicitud.Validar() != nil || solicitud.Evidencia.ValidarEn(solicitud.Actor, s.reloj.Ahora()) != nil {
		return domain.PropuestaAdministracionAcceso{}, domain.ErrAdministracionAccesoInvalida
	}
	if err := ctx.Err(); err != nil {
		return domain.PropuestaAdministracionAcceso{}, err
	}
	if err := s.validarAdministrador(ctx, solicitud.InstantaneaAutorizacion); err != nil {
		return domain.PropuestaAdministracionAcceso{}, err
	}
	autoridad, ok := s.actos.(ports.AutoridadAdministracionAcceso)
	if !ok {
		return domain.PropuestaAdministracionAcceso{}, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	conjunto, err := autoridad.ResolverConjuntoCuentasPersona(ctx, solicitud)
	if err != nil {
		return domain.PropuestaAdministracionAcceso{}, err
	}
	orden, err := domain.PrepararPropuestaAdministracionAcceso(solicitud, conjunto)
	if err != nil {
		return domain.PropuestaAdministracionAcceso{}, err
	}
	if err := ctx.Err(); err != nil {
		return domain.PropuestaAdministracionAcceso{}, err
	}
	paraPuerto := orden
	paraPuerto.Material.Conjunto.Cuentas = append([]domain.PreimagenCuentaAdministracionAcceso(nil), orden.Material.Conjunto.Cuentas...)
	propuesta, err := autoridad.ProponerCambioAccesoPersona(ctx, paraPuerto)
	if err != nil {
		return domain.PropuestaAdministracionAcceso{}, err
	}
	if propuesta.ValidarPara(orden) != nil || !propuesta.CaducaEn.After(s.reloj.Ahora()) {
		return domain.PropuestaAdministracionAcceso{}, domain.ErrAdministracionAccesoInvalida
	}
	propuesta.Material.Conjunto.Cuentas = append([]domain.PreimagenCuentaAdministracionAcceso(nil), propuesta.Material.Conjunto.Cuentas...)
	return propuesta, nil
}

// CerrarCambioAccesoPersona reutiliza el cierre de doble control existente.
// La autoridad reconstruye el conjunto y el estado solicitado de la propuesta,
// revalida bajo su barrera y aplica todos los efectos con el recibo indivisible.
func (s *ServicioAdministracionPerfiles) CerrarCambioAccesoPersona(ctx context.Context, solicitud domain.SolicitudCierrePropuestaAdministracionPerfiles) (domain.CierreAdministracionAcceso, error) {
	if s == nil || s.catalogo == nil || s.actos == nil || s.reloj == nil {
		return domain.CierreAdministracionAcceso{}, ErrAdministracionPerfilesNoConfigurada
	}
	if ctx == nil || solicitud.Validar() != nil || !domain.ReferenciaMotivoAutorizacionV2Valida(solicitud.Motivo) ||
		solicitud.Evidencia.ValidarEn(solicitud.Aprobador, s.reloj.Ahora()) != nil {
		return domain.CierreAdministracionAcceso{}, domain.ErrAdministracionAccesoInvalida
	}
	if err := ctx.Err(); err != nil {
		return domain.CierreAdministracionAcceso{}, err
	}
	if err := s.validarAdministrador(ctx, solicitud.InstantaneaAutorizacion); err != nil {
		return domain.CierreAdministracionAcceso{}, err
	}
	autoridad, ok := s.actos.(ports.AutoridadAdministracionAcceso)
	if !ok {
		return domain.CierreAdministracionAcceso{}, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return domain.CierreAdministracionAcceso{}, err
	}
	cierre, err := autoridad.CerrarCambioAccesoPersona(ctx, solicitud)
	if err != nil {
		return domain.CierreAdministracionAcceso{}, err
	}
	if cierre.ValidarPara(solicitud) != nil {
		return domain.CierreAdministracionAcceso{}, domain.ErrAdministracionAccesoInvalida
	}
	cierre.Material.Conjunto.Cuentas = append([]domain.PreimagenCuentaAdministracionAcceso(nil), cierre.Material.Conjunto.Cuentas...)
	if cierre.Recibo != nil {
		r := *cierre.Recibo
		r.Cuentas = append([]domain.ResultadoCuentaAdministracionAcceso(nil), r.Cuentas...)
		cierre.Recibo = &r
	}
	return cierre, nil
}
