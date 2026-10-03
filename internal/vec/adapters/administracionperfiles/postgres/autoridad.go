package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

func (a *Autoridad) disponible(ctx context.Context) error {
	if ctx == nil || a == nil || ausente(a.pool) || ausente(a.emisor) || ausente(a.reloj) {
		return ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	return ctx.Err()
}

func (a *Autoridad) ResolverRolAdministrable(ctx context.Context, ref string) (ports.RolAdministrable, error) {
	var cero ports.RolAdministrable
	if err := a.disponible(ctx); err != nil {
		return cero, err
	}
	if !domain.RolVersionAdministracionPerfilesValido(ref) {
		return cero, domain.ErrActoAdministracionPerfilesInvalido
	}
	var b []byte
	if err := a.pool.QueryRow(ctx, catalogoSQL, ref).Scan(&b); err != nil {
		return cero, traducir(ctx, err)
	}
	var x rolJSON
	if decodificar(b, &x) != nil || x.UnidadRequerida == nil {
		return cero, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	if x.Clase == domain.ClaseControlPerfilAdministrador && x.CategoriaAdmin == nil {
		return cero, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	rol := ports.RolAdministrable{VersionRef: x.VersionRef, Clase: x.Clase, HuellaSHA256: x.HuellaSHA256,
		VigenteDesde: x.VigenteDesde, VigenteHasta: x.VigenteHasta, UnidadRequerida: *x.UnidadRequerida}
	if x.CategoriaAdmin != nil {
		rol.CategoriaAdmin = *x.CategoriaAdmin
	}
	if rol.VersionRef != ref || rol.ValidarEn(a.reloj.Ahora()) != nil {
		return cero, domain.ErrActoAdministracionPerfilesInvalido
	}
	return rol, nil
}

func (a *Autoridad) AplicarActoOrdinario(ctx context.Context, s domain.SolicitudActoAdministracionPerfiles) (domain.ReciboAdministracionPerfiles, error) {
	var recibo domain.ReciboAdministracionPerfiles
	rol, err := a.validarActo(ctx, s, false)
	if err != nil {
		return recibo, err
	}
	e, err := materialActo(s, rol, false)
	if err != nil {
		return recibo, err
	}
	err = a.ejecutar(ctx, s.Actor, s.Evidencia, s.InstantaneaAutorizacion, e, aplicarSQL, func(b []byte) error {
		var x reciboJSON
		if decodificar(b, &x) != nil || !x.vigenciaHistoricaCompleta() {
			return ports.ErrAutoridadAdministracionPerfilesNoDisponible
		}
		recibo = x.dominio()
		if recibo.ValidarPara(s) != nil {
			return ports.ErrAutoridadAdministracionPerfilesNoDisponible
		}
		return nil
	})
	if err != nil {
		return domain.ReciboAdministracionPerfiles{}, err
	}
	return recibo, nil
}

func (a *Autoridad) ProponerActoSensible(ctx context.Context, s domain.SolicitudActoAdministracionPerfiles) (ports.PropuestaAdministracionPerfiles, error) {
	var resultado ports.PropuestaAdministracionPerfiles
	rol, err := a.validarActo(ctx, s, true)
	if err != nil {
		return resultado, err
	}
	e, err := materialActo(s, rol, true)
	if err != nil {
		return resultado, err
	}
	err = a.ejecutar(ctx, s.Actor, s.Evidencia, s.InstantaneaAutorizacion, e, proponerSQL, func(b []byte) error {
		var x propuestaJSON
		if decodificar(b, &x) != nil {
			return ports.ErrAutoridadAdministracionPerfilesNoDisponible
		}
		resultado = ports.PropuestaAdministracionPerfiles{OperacionRef: x.OperacionRef, PropuestaRef: x.PropuestaRef,
			HuellaSHA256: x.HuellaSHA256, ProponentePersonaRef: x.ProponentePersonaRef,
			ObjetivoPersonaRef: x.ObjetivoPersonaRef, CaducaEn: x.CaducaEn}
		if resultado.ValidarPara(s) != nil || !resultado.CaducaEn.After(a.reloj.Ahora()) || !instantePersistible(resultado.CaducaEn) {
			return domain.ErrControlAdministracionPerfilesInvalido
		}
		return nil
	})
	if err != nil {
		return ports.PropuestaAdministracionPerfiles{}, err
	}
	return resultado, nil
}

func (a *Autoridad) CerrarPropuestaSensible(ctx context.Context, s domain.SolicitudCierrePropuestaAdministracionPerfiles) (ports.CierrePropuestaAdministracionPerfiles, error) {
	var resultado ports.CierrePropuestaAdministracionPerfiles
	if err := a.disponible(ctx); err != nil {
		return resultado, err
	}
	if s.Validar() != nil || s.Evidencia.ValidarEn(s.Aprobador, a.reloj.Ahora()) != nil {
		return resultado, domain.ErrControlAdministracionPerfilesInvalido
	}
	if err := a.validarAdministrador(ctx, s.InstantaneaAutorizacion); err != nil {
		return resultado, err
	}
	e, err := materialCierre(s)
	if err != nil {
		return resultado, err
	}
	err = a.ejecutar(ctx, s.Aprobador, s.Evidencia, s.InstantaneaAutorizacion, e, cerrarSQL, func(b []byte) error {
		var x cierreResultadoJSON
		if decodificar(b, &x) != nil {
			return ports.ErrAutoridadAdministracionPerfilesNoDisponible
		}
		resultado = ports.CierrePropuestaAdministracionPerfiles{OperacionRef: x.OperacionRef, PropuestaRef: x.PropuestaRef,
			PropuestaHuellaSHA256: x.PropuestaHuellaSHA256, Decision: x.Decision, HuellaCierreSHA256: x.HuellaCierreSHA256, ConfirmadoEn: x.ConfirmadoEn}
		if x.Recibo != nil {
			if !x.Recibo.vigenciaHistoricaCompleta() {
				return ports.ErrAutoridadAdministracionPerfilesNoDisponible
			}
			r := x.Recibo.dominio()
			resultado.Recibo = &r
		}
		if !instantePersistible(resultado.ConfirmadoEn) || resultado.ValidarPara(s) != nil {
			return ports.ErrAutoridadAdministracionPerfilesNoDisponible
		}
		return nil
	})
	if err != nil {
		return ports.CierrePropuestaAdministracionPerfiles{}, err
	}
	return resultado, nil
}

func (a *Autoridad) validarActo(ctx context.Context, s domain.SolicitudActoAdministracionPerfiles, sensible bool) (ports.RolAdministrable, error) {
	var cero ports.RolAdministrable
	if err := a.disponible(ctx); err != nil {
		return cero, err
	}
	if s.Validar() != nil || s.Clase.RequiereDobleControl() != sensible ||
		s.Evidencia.ValidarEn(s.Actor, a.reloj.Ahora()) != nil {
		return cero, domain.ErrActoAdministracionPerfilesInvalido
	}
	if err := a.validarAdministrador(ctx, s.InstantaneaAutorizacion); err != nil {
		return cero, err
	}
	rol, err := a.ResolverRolAdministrable(ctx, s.RolVersionRef)
	if err != nil {
		return cero, err
	}
	if rol.Clase != s.Clase || rol.UnidadRequerida && s.Objetivo.UnidadRef == "" ||
		(s.Operacion == domain.OperacionOtorgarPerfil &&
			(s.Objetivo.VigenteDesde.Before(rol.VigenteDesde) || s.Objetivo.VigenteHasta.After(rol.VigenteHasta))) {
		return cero, domain.ErrActoAdministracionPerfilesInvalido
	}
	return rol, nil
}

func (a *Autoridad) validarAdministrador(ctx context.Context, instantanea domain.InstantaneaAutorizacion) error {
	if instantanea.Validar() != nil || instantanea.VersionRol.RolID != "administracion_perfiles" ||
		!instantanea.AsignacionPerfil.VigenteEn(a.reloj.Ahora()) ||
		instantanea.ControlVigenciaVersionRol.Estado != domain.EstadoControlVigenciaVersionRolHabilitada {
		return domain.ErrControlAdministracionPerfilesInvalido
	}
	rol, err := a.ResolverRolAdministrable(ctx, instantanea.VersionRol.Referencia())
	if err != nil {
		return err
	}
	huella, err := instantanea.VersionRol.HuellaSHA256()
	if err != nil || rol.HuellaSHA256 != huella || rol.Clase != domain.ClaseControlPerfilAdministrador || rol.CategoriaAdmin != "aplicacion" {
		return domain.ErrControlAdministracionPerfilesInvalido
	}
	return nil
}

// AplicarLoteOrdinario conserva cerrado el efecto hasta disponer de una fachada
// central de lote aprobada. Nunca transforma la orden en actos singulares.
func (a *Autoridad) AplicarLoteOrdinario(ctx context.Context, _ domain.SolicitudLoteAdministracionPerfiles) (domain.ReciboLoteAdministracionPerfiles, error) {
	if ctx != nil && ctx.Err() != nil {
		return domain.ReciboLoteAdministracionPerfiles{}, ctx.Err()
	}
	return domain.ReciboLoteAdministracionPerfiles{}, ports.ErrAutoridadAdministracionPerfilesNoDisponible
}

func (a *Autoridad) ejecutar(ctx context.Context, actor domain.ContextoActor, evidencia domain.EvidenciaSesionAdministracionPerfiles, instantanea domain.InstantaneaAutorizacion, e Efecto, consulta string, validar func([]byte) error) error {
	if err := a.disponible(ctx); err != nil {
		return err
	}
	return ejecutarConsumoADMIN(ctx, a.pool, a.emisor, a.reloj, actor, evidencia, instantanea, e, consulta, validar)
}

func traducir(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "22023", "23514":
			return domain.ErrActoAdministracionPerfilesInvalido
		case "40001", "40P01":
			return domain.ErrControlAdministracionPerfilesInvalido
		}
	}
	return ports.ErrAutoridadAdministracionPerfilesNoDisponible
}

func instantePersistible(t time.Time) bool {
	return !t.IsZero() && t.Equal(t.Truncate(time.Microsecond))
}
