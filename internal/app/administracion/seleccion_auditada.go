package administracion

import (
	"context"
	"strings"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad/adminperfiles"
	"vec-diputacion-granada/internal/vec/ports"
)

// Este seam aún necesita una implementación de identidad y auditoría común.
// La fuente conserva lectura/selección, revalidación y recibo en la misma
// transacción. No se admite auditar después de confirmar otro repositorio.
type FuenteSeleccionAuditadaADMIN = adminperfiles.FuenteSeleccionAuditadaADMIN
type LecturaPropiosAuditadaADMIN = adminperfiles.LecturaPropiosAuditadaADMIN
type SeleccionAuditadaADMIN = adminperfiles.SeleccionAuditadaADMIN

func referenciaAuditoriaComun(ref string) bool {
	return ref != "" && len(ref) <= 256 && !strings.ContainsAny(ref, "* \t\r\n")
}

type seleccionAuditadaADMIN struct{ fuente FuenteSeleccionAuditadaADMIN }

func (s seleccionAuditadaADMIN) ListarPropiosADMIN(ctx context.Context, o adminperfiles.ObservacionADMIN) (adminperfiles.PerfilesPropios, error) {
	if s.fuente == nil {
		return adminperfiles.PerfilesPropios{}, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	r, err := s.fuente.ListarPropiosAuditadosADMIN(ctx, o)
	if err != nil {
		return adminperfiles.PerfilesPropios{}, err
	}
	if !r.Propios.Validos() || !referenciaAuditoriaComun(r.AuditoriaComunRef) {
		return adminperfiles.PerfilesPropios{}, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	return r.Propios, nil
}
func (s seleccionAuditadaADMIN) SeleccionarPerfilADMIN(ctx context.Context, o adminperfiles.ObservacionADMIN, ref string, version uint64) (adminperfiles.SeleccionPerfil, error) {
	if s.fuente == nil {
		return adminperfiles.SeleccionPerfil{}, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	r, err := s.fuente.SeleccionarPerfilAuditadoADMIN(ctx, o, ref, version)
	if err != nil {
		return adminperfiles.SeleccionPerfil{}, err
	}
	if !r.Seleccion.Valida() || r.Seleccion.PerfilActivoRef != ref || !referenciaAuditoriaComun(r.AuditoriaComunRef) {
		return adminperfiles.SeleccionPerfil{}, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	return r.Seleccion, nil
}
