package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
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
	if decodificar(b, &x) != nil {
		return cero, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	rol := ports.RolAdministrable{VersionRef: x.VersionRef, Clase: x.Clase, HuellaSHA256: x.HuellaSHA256,
		VigenteDesde: x.VigenteDesde, VigenteHasta: x.VigenteHasta, UnidadRequerida: x.UnidadRequerida}
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
	err = a.ejecutar(ctx, s.Actor, s.InstantaneaAutorizacion, e, aplicarSQL, func(b []byte) error {
		var x reciboJSON
		if decodificar(b, &x) != nil {
			return ports.ErrAutoridadAdministracionPerfilesNoDisponible
		}
		recibo = x.dominio()
		if recibo.Validar() != nil || recibo.OperacionRef != s.OperacionRef || recibo.ObjetivoPersonaRef != s.Objetivo.PersonaRef ||
			recibo.PerfilRef != s.Objetivo.PerfilRef || recibo.VinculoRef != s.Objetivo.VinculoRef || recibo.UnidadRef != s.Objetivo.UnidadRef || recibo.ReferenciaActo != s.ReferenciaActo || recibo.PropuestaRef != "" {
			return domain.ErrActoAdministracionPerfilesInvalido
		}
		if s.Operacion == domain.OperacionOtorgarPerfil && (recibo.EstadoPosterior != domain.EstadoVinculoContextoActorActivo || recibo.VersionPosterior != 1) ||
			s.Operacion == domain.OperacionRevocarPerfil && (recibo.EstadoPosterior != domain.EstadoVinculoContextoActorRevocado || recibo.VersionPosterior <= s.Objetivo.VinculoVersion) {
			return domain.ErrActoAdministracionPerfilesInvalido
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
	err = a.ejecutar(ctx, s.Actor, s.InstantaneaAutorizacion, e, proponerSQL, func(b []byte) error {
		var x propuestaJSON
		if decodificar(b, &x) != nil {
			return ports.ErrAutoridadAdministracionPerfilesNoDisponible
		}
		resultado = ports.PropuestaAdministracionPerfiles{x.OperacionRef, x.PropuestaRef, x.HuellaSHA256, x.ProponentePersonaRef, x.ObjetivoPersonaRef, x.CaducaEn}
		if resultado.ValidarPara(s) != nil || !resultado.CaducaEn.After(a.reloj.Ahora()) {
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
	if s.Validar() != nil {
		return resultado, domain.ErrControlAdministracionPerfilesInvalido
	}
	e, err := materialCierre(s)
	if err != nil {
		return resultado, err
	}
	err = a.ejecutar(ctx, s.Aprobador, s.InstantaneaAutorizacion, e, cerrarSQL, func(b []byte) error {
		var x cierreResultadoJSON
		if decodificar(b, &x) != nil {
			return ports.ErrAutoridadAdministracionPerfilesNoDisponible
		}
		resultado = ports.CierrePropuestaAdministracionPerfiles{OperacionRef: x.OperacionRef, PropuestaRef: x.PropuestaRef,
			Decision: x.Decision, HuellaCierreSHA256: x.HuellaCierreSHA256, ConfirmadoEn: x.ConfirmadoEn}
		if x.Recibo != nil {
			r := x.Recibo.dominio()
			resultado.Recibo = &r
		}
		return resultado.ValidarPara(s)
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
	if s.Validar() != nil || s.Clase.RequiereDobleControl() != sensible {
		return cero, domain.ErrActoAdministracionPerfilesInvalido
	}
	rol, err := a.ResolverRolAdministrable(ctx, s.RolVersionRef)
	if err != nil {
		return cero, err
	}
	if rol.Clase != s.Clase || rol.UnidadRequerida && s.Objetivo.UnidadRef == "" {
		return cero, domain.ErrActoAdministracionPerfilesInvalido
	}
	return rol, nil
}

func (a *Autoridad) ejecutar(ctx context.Context, actor domain.ContextoActor, instantanea domain.InstantaneaAutorizacion, e Efecto, consulta string, validar func([]byte) error) error {
	// El emisor recibe copia: no puede alterar el material ya ligado al efecto.
	entrega := e
	entrega.Material = append([]byte(nil), e.Material...)
	m, err := a.emisor.EmitirAdministracionPerfiles(ctx, actor, instantanea, entrega)
	clear(entrega.Material)
	if err != nil {
		return traducir(ctx, err)
	}
	huella := sha256.Sum256(e.Material)
	r := m.ResumenCapacidad()
	ahora := a.reloj.Ahora()
	if m.ValidarEstructura() != nil || r.AudienciaConsumo() != e.Audiencia || r.Operacion() != e.Accion ||
		r.EfectoRef() != e.Referencia || r.EfectoHuellaSHA256() != hex.EncodeToString(huella[:]) || ahora.Before(r.EmitidaEn()) || !ahora.Before(r.ExpiraEn()) ||
		m.PersonaVersion() != actor.Instantanea.PersonaVersion || m.PerfilVersion() != actor.Instantanea.PerfilVersion {
		return ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	args := []any{string(e.Material), m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(),
		strconv.FormatUint(m.PersonaVersion(), 10), strconv.FormatUint(m.PerfilVersion(), 10), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI()}
	defer func() {
		for _, arg := range args {
			if b, ok := arg.([]byte); ok {
				clear(b)
			}
		}
	}()
	tx, err := a.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return traducir(ctx, err)
	}
	if ausente(tx) {
		return ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	defer func() {
		rollbackCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = tx.Rollback(rollbackCtx)
	}()
	var bruto []byte
	if err := tx.QueryRow(ctx, consulta, args...).Scan(&bruto); err != nil {
		return traducir(ctx, err)
	}
	if err := validar(bruto); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return traducir(ctx, tx.Commit(ctx))
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
