package postgres

import (
	"context"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

var _ ports.CatalogoRolesAdministrables = (*AutoridadVersionarRolBolsa)(nil)
var _ ports.AutoridadActosAdministracionPerfiles = (*AutoridadVersionarRolBolsa)(nil)

// ResolverRolAdministrable sólo comprueba la versión ADMIN que seleccionó la
// sesión. No deriva autoridad del nombre: el emisor y SQL cotejan el permiso.
func (a *AutoridadVersionarRolBolsa) ResolverRolAdministrable(ctx context.Context,
	ref string) (ports.RolAdministrable, error) {
	var vacio ports.RolAdministrable
	if err := a.disponible(ctx); err != nil {
		return vacio, err
	}
	if !domain.VersionRolAplicacionAdmitida(ref) {
		return vacio, ports.ConClaseVersionBolsa("rol_administrable_version", domain.ErrControlAdministracionPerfilesInvalido)
	}
	var b []byte
	if err := a.pool.QueryRow(ctx, catalogoSQL, ref).Scan(&b); err != nil {
		return vacio, ports.ConClaseVersionBolsa("rol_administrable_consulta", traducirGobiernoRol(ctx, err))
	}
	var x rolJSON
	if decodificar(b, &x) != nil || x.UnidadRequerida == nil || x.CategoriaAdmin == nil {
		return vacio, ports.ConClaseVersionBolsa("rol_administrable_respuesta", ports.ErrAutoridadAdministracionPerfilesNoDisponible)
	}
	r := ports.RolAdministrable{VersionRef: x.VersionRef, Clase: x.Clase, HuellaSHA256: x.HuellaSHA256,
		VigenteDesde: x.VigenteDesde, VigenteHasta: x.VigenteHasta, UnidadRequerida: *x.UnidadRequerida,
		CategoriaAdmin: *x.CategoriaAdmin}
	if r.VersionRef != ref || r.Clase != domain.ClaseControlPerfilAdministrador ||
		r.CategoriaAdmin != "aplicacion" || r.ValidarEn(a.reloj.Ahora()) != nil {
		return vacio, ports.ConClaseVersionBolsa("rol_administrable_incoherente", domain.ErrControlAdministracionPerfilesInvalido)
	}
	return r, nil
}

// La interfaz indivisible del servicio deniega expresamente AUT24: el LOGIN
// B1 carece de EXECUTE sobre esas funciones y la fachada no expone sus rutas.
func (*AutoridadVersionarRolBolsa) AplicarActoOrdinario(ctx context.Context,
	_ domain.SolicitudActoAdministracionPerfiles) (domain.ReciboAdministracionPerfiles, error) {
	return domain.ReciboAdministracionPerfiles{}, errorActoAjenoGobierno(ctx)
}

func (*AutoridadVersionarRolBolsa) ProponerActoSensible(ctx context.Context,
	_ domain.SolicitudActoAdministracionPerfiles) (ports.PropuestaAdministracionPerfiles, error) {
	return ports.PropuestaAdministracionPerfiles{}, errorActoAjenoGobierno(ctx)
}

func (*AutoridadVersionarRolBolsa) CerrarPropuestaSensible(ctx context.Context,
	_ domain.SolicitudCierrePropuestaAdministracionPerfiles) (ports.CierrePropuestaAdministracionPerfiles, error) {
	return ports.CierrePropuestaAdministracionPerfiles{}, errorActoAjenoGobierno(ctx)
}
