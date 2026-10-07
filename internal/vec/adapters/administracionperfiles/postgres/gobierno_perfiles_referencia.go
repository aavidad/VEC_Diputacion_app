package postgres

import (
	"context"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

var _ ports.AutoridadCierreGobiernoRolPorReferencia = (*AutoridadGobiernoRolNuevo)(nil)

// CerrarGobiernoRolPorReferencia conserva la propuesta original como fuente
// de proponente y RolID. AUT60 resuelve ambos bajo el cerrojo/CAS de la misma
// transacción donde consume la aprobación V3. Se valida respuesta antes de
// COMMIT; un resultado incompatible revierte el efecto y su recibo.
func (a *AutoridadGobiernoRolNuevo) CerrarGobiernoRolPorReferencia(ctx context.Context,
	s domain.SolicitudCierreGobiernoRolPorReferencia) (domain.CierreGobiernoPerfil, error) {
	var vacio domain.CierreGobiernoPerfil
	if err := a.disponibleGobierno(ctx); err != nil {
		return vacio, err
	}
	e, err := materialCierreGobiernoRolPorReferencia(s)
	if err != nil {
		return vacio, err
	}
	var r cierreGobiernoRolRespuesta
	err = a.ejecutarGobierno(ctx, s.Aprobador, s.Evidencia, s.InstantaneaAutorizacion,
		e, cerrarGobiernoRolSQL, func(b []byte) error {
			if decodificarGobiernoRol(b, &r) != nil || r.Estado != "permitido" ||
				r.Decision != domain.DecisionAprobarPropuestaPerfil || r.OperacionRef != s.OperacionRef {
				return ports.ErrAutoridadAdministracionPerfilesNoDisponible
			}
			var m domain.MaterialPropuestaGobiernoPerfil
			if decodificarGobiernoRol([]byte(r.MaterialCanon), &m) != nil {
				return ports.ErrAutoridadAdministracionPerfilesNoDisponible
			}
			completa, err := s.CompletarCierreGobiernoRolConMaterial(m)
			if err != nil {
				return ports.ErrAutoridadAdministracionPerfilesNoDisponible
			}
			r.Cierre = domain.CierreGobiernoPerfil{OperacionRef: r.OperacionRef, Material: m,
				PropuestaHuellaSHA256: r.PropuestaHuellaSHA256, Decision: r.Decision,
				ConfirmadoEn: r.ConfirmadoEn, AuditoriaAccesoRef: r.AuditoriaAccesoRef,
				Recibo: r.Recibo.Dominio()}
			if r.Cierre.ValidarPara(completa) != nil || r.Cierre.ConfirmadoEn.After(a.reloj.Ahora()) {
				return ports.ErrAutoridadAdministracionPerfilesNoDisponible
			}
			return nil
		})
	if err != nil {
		return vacio, err
	}
	r.Cierre.Material.Plan = r.Cierre.Material.Plan.Copia()
	if r.Cierre.Recibo != nil {
		copia := r.Cierre.Recibo.Copia()
		r.Cierre.Recibo = &copia
	}
	return r.Cierre, nil
}
