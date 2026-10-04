package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const accionLoteOrdinario = "administracion.perfiles.aplicar_lote_ordinario"
const audienciaLoteOrdinario = "vec_autorizacion.administracion_perfiles.lote_ordinario.v1"
const aplicarLoteOrdinarioSQL = `SELECT vec_autorizacion.aplicar_lote_ordinario_admin_v1` + argumentosV3

// NuevaAutoridadLoteOrdinario no depende de las fachadas singulares heredadas.
// El LOGIN propio se acredita en SQL antes de recibir la primera orden.
func NuevaAutoridadLoteOrdinario(ctx context.Context, pool *pgxpool.Pool, emisor Emisor, reloj ports.Reloj) (*Autoridad, error) {
	return nuevaAutoridadLoteOrdinario(ctx, pool, emisor, reloj)
}

func nuevaAutoridadLoteOrdinario(ctx context.Context, pool conexion, emisor Emisor, reloj ports.Reloj) (*Autoridad, error) {
	if ctx == nil || ausente(pool) || ausente(emisor) || ausente(reloj) || ctx.Err() != nil {
		return nil, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	var acreditado bool
	if err := pool.QueryRow(ctx, `SELECT vec_autorizacion.acreditar_login_lote_ordinario_admin_v1()`).Scan(&acreditado); err != nil || !acreditado {
		return nil, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	return &Autoridad{pool: pool, emisor: emisor, reloj: reloj}, nil
}

type reciboLoteOrdinarioJSON struct {
	OperacionRef          string       `json:"operacion_ref"`
	ActoRef               string       `json:"acto_ref"`
	ReciboRef             string       `json:"recibo_ref"`
	AuditoriaRef          string       `json:"auditoria_ref"`
	HuellaSolicitudSHA256 string       `json:"huella_solicitud_sha256"`
	ConfirmadoEn          time.Time    `json:"confirmado_en"`
	Cambios               []reciboJSON `json:"cambios"`
}

func (x reciboLoteOrdinarioJSON) dominio() domain.ReciboLoteAdministracionPerfiles {
	r := domain.ReciboLoteAdministracionPerfiles{
		OperacionRef: x.OperacionRef, ActoRef: x.ActoRef, ReciboRef: x.ReciboRef,
		AuditoriaRef: x.AuditoriaRef, HuellaSolicitudSHA256: x.HuellaSolicitudSHA256,
		ConfirmadoEn: x.ConfirmadoEn, Cambios: make([]domain.ReciboAdministracionPerfiles, 0, len(x.Cambios)),
	}
	for _, c := range x.Cambios {
		r.Cambios = append(r.Cambios, c.dominio())
	}
	return r
}

func (a *Autoridad) aplicarLoteOrdinario(ctx context.Context, s domain.SolicitudLoteAdministracionPerfiles) (domain.ReciboLoteAdministracionPerfiles, error) {
	var vacio domain.ReciboLoteAdministracionPerfiles
	if err := a.disponible(ctx); err != nil {
		return vacio, err
	}
	if s.Validar() != nil || s.Evidencia.ValidarEn(s.Actor, a.reloj.Ahora()) != nil ||
		s.InstantaneaAutorizacion.VersionRol.RolID != "administracion_perfiles" ||
		s.InstantaneaAutorizacion.VersionRol.Version != 6 ||
		s.InstantaneaAutorizacion.VersionRol.Estado != domain.EstadoVersionRolPublicada ||
		s.InstantaneaAutorizacion.ControlVigenciaVersionRol.Estado != domain.EstadoControlVigenciaVersionRolHabilitada ||
		!s.InstantaneaAutorizacion.AsignacionPerfil.VigenteEn(a.reloj.Ahora()) {
		return vacio, domain.ErrActoAdministracionPerfilesInvalido
	}
	for _, cambio := range s.Cambios {
		rol, err := a.ResolverRolAdministrable(ctx, cambio.RolVersionRef)
		if err != nil {
			return vacio, err
		}
		if rol.VersionRef != cambio.RolVersionRef || rol.Clase != domain.ClaseControlPerfilOrdinario ||
			(rol.UnidadRequerida && cambio.Objetivo.UnidadRef == "") ||
			(cambio.Operacion == domain.OperacionOtorgarPerfil &&
				(cambio.Objetivo.VigenteDesde.Before(rol.VigenteDesde) || cambio.Objetivo.VigenteHasta.After(rol.VigenteHasta))) {
			return vacio, domain.ErrActoAdministracionPerfilesInvalido
		}
	}
	canonico, huella, err := s.CanonicoYHuella()
	if err != nil || huella != s.HuellaSolicitudSHA256 || len(canonico) > 65536 {
		return vacio, domain.ErrActoAdministracionPerfilesInvalido
	}
	actor, err := s.Actor.Clonar()
	if err != nil {
		return vacio, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	resultado, err := s.Evidencia.ResultadoContexto.Clonar()
	if err != nil {
		return vacio, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	evidencia := domain.EvidenciaSesionAdministracionPerfiles{ResultadoContexto: resultado, Vinculo: s.Evidencia.Vinculo}
	if evidencia.ValidarPara(actor) != nil {
		return vacio, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	var instantanea domain.InstantaneaAutorizacion
	b, err := json.Marshal(s.InstantaneaAutorizacion)
	if err != nil || json.Unmarshal(b, &instantanea) != nil || instantanea.Validar() != nil {
		return vacio, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	efecto := Efecto{Accion: accionLoteOrdinario, Audiencia: audienciaLoteOrdinario,
		Referencia: s.Cambios[0].Objetivo.PersonaRef, Material: canonico, CorrelacionAccesoRef: s.CorrelacionRef}
	var recibo domain.ReciboLoteAdministracionPerfiles
	err = a.ejecutar(ctx, actor, evidencia, instantanea, efecto, aplicarLoteOrdinarioSQL, func(bruto []byte) error {
		var x reciboLoteOrdinarioJSON
		if decodificar(bruto, &x) != nil || len(x.Cambios) != len(s.Cambios) {
			return ports.ErrAutoridadAdministracionPerfilesNoDisponible
		}
		for _, c := range x.Cambios {
			if !c.vigenciaHistoricaCompleta() {
				return ports.ErrAutoridadAdministracionPerfilesNoDisponible
			}
		}
		recibo = x.dominio()
		if recibo.ValidarPara(s) != nil {
			return ports.ErrAutoridadAdministracionPerfilesNoDisponible
		}
		return nil
	})
	if err != nil {
		return vacio, err
	}
	return recibo, nil
}
