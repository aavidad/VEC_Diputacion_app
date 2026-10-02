package application

import (
	"bytes"
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type ServicioVinculoPropioCRN11 struct {
	autorizador ports.ProveedorAutorizacionVinculoCRN11
	repositorio ports.RepositorioVinculoPropioCRN11
	ahora       func() time.Time
}

func NuevoServicioVinculoPropioCRN11(a ports.ProveedorAutorizacionVinculoCRN11, r ports.RepositorioVinculoPropioCRN11, ahora func() time.Time) (*ServicioVinculoPropioCRN11, error) {
	if nulo(a) || nulo(r) || ahora == nil {
		return nil, domain.ErrVinculoCRN11NoDisponible
	}
	return &ServicioVinculoPropioCRN11{a, r, ahora}, nil
}
func (s *ServicioVinculoPropioCRN11) ConsultarVinculoPropioCRN11(ctx context.Context, in domain.SolicitudVinculoPropioCRN11) (ports.ResultadoVinculoPropioCRN11, error) {
	var cero ports.ResultadoVinculoPropioCRN11
	if s == nil || ctx == nil || nulo(s.autorizador) || nulo(s.repositorio) || s.ahora == nil {
		return cero, domain.ErrVinculoCRN11NoDisponible
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	m, err := domain.NuevoMaterialVinculoPropioCRN11(in)
	if err != nil {
		return cero, err
	}
	instante := s.ahora().UTC().Truncate(time.Microsecond)
	if !m.VigenteParaLectura(instante) {
		return cero, domain.ErrVinculoCRN11Denegado
	}
	a, err := s.autorizador.AutorizarVinculoPropioCRN11(ctx, m)
	if err != nil {
		return cero, errorVinculoCRN11Opaco(ctx, err)
	}
	if !autorizacionVinculoCRN11Valida(m, a) {
		return cero, domain.ErrVinculoCRN11NoDisponible
	}
	instante = s.ahora().UTC().Truncate(time.Microsecond)
	x := a.ResumenCapacidad()
	if !m.VigenteParaLectura(instante) || instante.Before(x.EmitidaEn()) || !instante.Before(x.ExpiraEn()) {
		return cero, domain.ErrVinculoCRN11Denegado
	}
	r, err := s.repositorio.ConsultarVinculoPropioCRN11(ctx, ports.OrdenVinculoPropioCRN11{Material: m, Autorizacion: a})
	if err != nil {
		return cero, errorVinculoCRN11Opaco(ctx, err)
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	instante = s.ahora().UTC().Truncate(time.Microsecond)
	if !m.VigenteParaLectura(instante) || !instante.Before(x.ExpiraEn()) {
		return cero, domain.ErrVinculoCRN11Denegado
	}
	if r.Vinculo.ValidarPara(m) != nil || !evidenciaFichaPropiaValida(a, r.Evidencia) || r.Evidencia.ConsultadaEn.After(instante) {
		return cero, domain.ErrVinculoCRN11NoDisponible
	}
	return r, nil
}
func autorizacionVinculoCRN11Valida(m domain.MaterialVinculoPropioCRN11, a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) bool {
	actor := m.Actor()
	canon, err := actor.RepresentacionCanonicaVinculadaV2()
	h, errH := m.HuellaSHA256()
	x := a.ResumenCapacidad()
	return err == nil && errH == nil && a.ValidarEstructura() == nil && bytes.Equal(canon, a.ContextoActorCanonico()) &&
		a.PersonaVersion() == actor.Instantanea.PersonaVersion && a.PerfilVersion() == actor.Instantanea.PerfilVersion &&
		x.Operacion() == domain.AccionVinculoPropioCRN11 && x.AudienciaConsumo() == domain.AudienciaVinculoPropioCRN11 && x.EfectoRef() == m.EmpleadoRef() && x.EfectoHuellaSHA256() == h
}
func errorVinculoCRN11Opaco(ctx context.Context, err error) error {
	if e := ctx.Err(); e != nil {
		return e
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if errors.Is(err, domain.ErrVinculoCRN11Denegado) {
		return domain.ErrVinculoCRN11Denegado
	}
	return domain.ErrVinculoCRN11NoDisponible
}

var _ ports.LectorVinculoPropioCRN11 = (*ServicioVinculoPropioCRN11)(nil)
