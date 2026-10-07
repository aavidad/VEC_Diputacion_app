package postgres

import (
	"context"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

var _ ports.CatalogoRolesAdministrables = (*AutoridadGobiernoRolNuevo)(nil)
var _ ports.AutoridadActosAdministracionPerfiles = (*AutoridadGobiernoRolNuevo)(nil)

// ResolverRolAdministrable lee sólo la versión del ADMIN Aplicación que
// seleccionó la sesión. AUT60 concede al LOGIN Gov esta única lectura del
// catálogo central; no infiere categoría a partir del nombre del rol.
func (a *AutoridadGobiernoRolNuevo) ResolverRolAdministrable(ctx context.Context,
	ref string) (ports.RolAdministrable, error) {
	var vacio ports.RolAdministrable
	if err := a.disponibleGobierno(ctx); err != nil {
		return vacio, err
	}
	if !domain.VersionRolAplicacionAdmitida(ref) {
		return vacio, domain.ErrControlAdministracionPerfilesInvalido
	}
	var b []byte
	if err := a.pool.QueryRow(ctx, catalogoSQL, ref).Scan(&b); err != nil {
		return vacio, traducirGobiernoRol(ctx, err)
	}
	var x rolJSON
	if decodificar(b, &x) != nil || x.UnidadRequerida == nil || x.CategoriaAdmin == nil {
		return vacio, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	r := ports.RolAdministrable{VersionRef: x.VersionRef, Clase: x.Clase, HuellaSHA256: x.HuellaSHA256,
		VigenteDesde: x.VigenteDesde, VigenteHasta: x.VigenteHasta, UnidadRequerida: *x.UnidadRequerida,
		CategoriaAdmin: *x.CategoriaAdmin}
	if r.VersionRef != ref || r.Clase != domain.ClaseControlPerfilAdministrador ||
		r.CategoriaAdmin != "aplicacion" || r.ValidarEn(a.reloj.Ahora()) != nil {
		return vacio, domain.ErrControlAdministracionPerfilesInvalido
	}
	return r, nil
}

// La interfaz indivisible del servicio se satisface con denegación explícita
// para los actos AUT24. El handler de gobierno no monta esas rutas y el LOGIN
// Gov carece de EXECUTE sobre sus fachadas; tampoco hay emisor para ellas.
func (*AutoridadGobiernoRolNuevo) AplicarActoOrdinario(ctx context.Context,
	_ domain.SolicitudActoAdministracionPerfiles) (domain.ReciboAdministracionPerfiles, error) {
	return domain.ReciboAdministracionPerfiles{}, errorActoAjenoGobierno(ctx)
}

func (*AutoridadGobiernoRolNuevo) ProponerActoSensible(ctx context.Context,
	_ domain.SolicitudActoAdministracionPerfiles) (ports.PropuestaAdministracionPerfiles, error) {
	return ports.PropuestaAdministracionPerfiles{}, errorActoAjenoGobierno(ctx)
}

func (*AutoridadGobiernoRolNuevo) CerrarPropuestaSensible(ctx context.Context,
	_ domain.SolicitudCierrePropuestaAdministracionPerfiles) (ports.CierrePropuestaAdministracionPerfiles, error) {
	return ports.CierrePropuestaAdministracionPerfiles{}, errorActoAjenoGobierno(ctx)
}

func errorActoAjenoGobierno(ctx context.Context) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	return ports.ErrAutoridadAdministracionPerfilesNoDisponible
}
