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

type ServicioLectorRelacionRPT struct {
	autorizador ports.ProveedorAutorizacionLectorRelacionRPT
	repositorio ports.RepositorioLectorRelacionRPT
	intentos    ports.RegistroIntentosLectorRelacionRPT
	ahora       func() time.Time
}

func NuevoServicioLectorRelacionRPT(a ports.ProveedorAutorizacionLectorRelacionRPT, r ports.RepositorioLectorRelacionRPT, i ports.RegistroIntentosLectorRelacionRPT, ahora func() time.Time) (*ServicioLectorRelacionRPT, error) {
	if nulo(a) || nulo(r) || nulo(i) || ahora == nil {
		return nil, domain.ErrLectorRelacionRPTNoDisponible
	}
	return &ServicioLectorRelacionRPT{a, r, i, ahora}, nil
}
func (s *ServicioLectorRelacionRPT) ConsultarRelacionParaRPT(ctx context.Context, in ports.ConsultaRelacionParaRPTV1) (ports.ResultadoRelacionParaRPTV1, error) {
	var cero ports.ResultadoRelacionParaRPTV1
	if s == nil || ctx == nil || nulo(s.autorizador) || nulo(s.repositorio) || nulo(s.intentos) || s.ahora == nil {
		return cero, domain.ErrLectorRelacionRPTNoDisponible
	}
	fallar := func(err error) (ports.ResultadoRelacionParaRPTV1, error) { return cero, s.registrarFallo(ctx, in, err) }
	if err := ctx.Err(); err != nil {
		return fallar(err)
	}
	if err := s.intentos.VerificarRegistroRelacionRPT(ctx); err != nil {
		return fallar(domain.ErrLectorRelacionRPTNoDisponible)
	}
	m, err := domain.NuevoMaterialLectorRelacionRPT(domain.SolicitudLectorRelacionRPT{Actor: in.Actor, EmpleadoRef: in.EmpleadoRef, RelacionRef: in.RelacionRef, OrganismoRef: in.OrganismoRef, VersionEsperada: in.VersionEsperada, Corte: in.Corte})
	if err != nil {
		return fallar(err)
	}
	instante := s.ahora().UTC().Truncate(time.Microsecond)
	if !m.ActorVigenteEn(instante) || m.Corte().ConocidoEn.After(instante) {
		return fallar(domain.ErrLectorRelacionRPTDenegado)
	}
	a, err := s.autorizador.AutorizarRelacionParaRPT(ctx, m)
	if err != nil {
		return fallar(errorLectorRPTNominal(ctx, err))
	}
	if !AutorizacionLectorRelacionRPTValida(m, a) {
		return fallar(domain.ErrLectorRelacionRPTNoDisponible)
	}
	instante = s.ahora().UTC().Truncate(time.Microsecond)
	x := a.ResumenCapacidad()
	if !m.ActorVigenteEn(instante) || instante.Before(x.EmitidaEn()) || !instante.Before(x.ExpiraEn()) {
		return fallar(domain.ErrLectorRelacionRPTDenegado)
	}
	r, err := s.repositorio.ConsultarRelacionParaRPT(ctx, ports.OrdenLectorRelacionRPT{Material: m, Autorizacion: a})
	if err != nil {
		return fallar(errorLectorRPTNominal(ctx, err))
	}
	if err := ctx.Err(); err != nil {
		return fallar(err)
	}
	instante = s.ahora().UTC().Truncate(time.Microsecond)
	if !m.ActorVigenteEn(instante) || !instante.Before(x.ExpiraEn()) {
		return fallar(domain.ErrLectorRelacionRPTDenegado)
	}
	if !ResultadoLectorRelacionRPTValido(m, a, r) || r.Evidencia.ConsultadaEn.After(instante) {
		return fallar(domain.ErrLectorRelacionRPTNoDisponible)
	}
	return r, nil
}

// Estas comprobaciones también las invoca el adaptador antes de confirmar.
// No verifican COSE ni conceden acceso: la fachada SQL revalida y consume V3.
func AutorizacionLectorRelacionRPTValida(m domain.MaterialLectorRelacionRPT, a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) bool {
	actor := m.Actor()
	canon, err := actor.RepresentacionCanonicaVinculadaV2()
	h, errH := m.HuellaSHA256()
	x := a.ResumenCapacidad()
	return err == nil && errH == nil && a.ValidarEstructura() == nil && bytes.Equal(canon, a.ContextoActorCanonico()) &&
		a.PersonaVersion() == actor.Instantanea.PersonaVersion && a.PerfilVersion() == actor.Instantanea.PerfilVersion &&
		x.Operacion() == ports.AccionRelacionParaRPTV1 && x.AudienciaConsumo() == ports.AudienciaRelacionParaRPTV1 && x.EfectoRef() == m.Recurso().Referencia && x.EfectoHuellaSHA256() == h
}
func ResultadoLectorRelacionRPTValido(m domain.MaterialLectorRelacionRPT, a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, r ports.ResultadoRelacionParaRPTV1) bool {
	v := r.Relacion
	return m.CoincideObjetivo(v.EmpleadoRef, v.RelacionRef, v.OrganismoRef, v.Version) && m.CoincideCorte(r.Corte) &&
		domain.ValidarEstadoPeriodoLectorRelacionRPT(v.Estado, v.Periodo.Desde, v.Periodo.Hasta) == nil &&
		domain.ValidarProcedenciaLectorRelacionRPT(v.Procedencia.ActoRef, v.Procedencia.FuenteRef, v.Procedencia.FuenteVersion) == nil &&
		v.Procedencia.Certeza == ports.CertezaPersonalNoAcreditadaV1 && r.Cobertura == ports.CoberturaPersonalNoAcreditadaV1 &&
		evidenciaFichaPropiaValida(a, r.Evidencia) && m.ActorVigenteEn(r.Evidencia.ConsultadaEn) && !r.Evidencia.ConsultadaEn.Before(m.Corte().ConocidoEn)
}
func (s *ServicioLectorRelacionRPT) registrarFallo(ctx context.Context, in ports.ConsultaRelacionParaRPTV1, err error) error {
	nominal := errorLectorRPTNominal(ctx, err)
	motivo := "no_disponible"
	if errors.Is(nominal, domain.ErrLectorRelacionRPTInvalido) {
		motivo = "entrada_invalida"
	}
	if errors.Is(nominal, domain.ErrLectorRelacionRPTDenegado) {
		motivo = "denegado"
	}
	auditCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()
	if e := s.intentos.RegistrarIntentoRelacionRPT(auditCtx, ports.IntentoLectorRelacionRPT{Actor: in.Actor, RelacionRef: in.RelacionRef, Motivo: motivo}); e != nil {
		return domain.ErrLectorRelacionRPTNoDisponible
	}
	return nominal
}
func errorLectorRPTNominal(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}
	if errors.Is(err, domain.ErrLectorRelacionRPTDenegado) {
		return domain.ErrLectorRelacionRPTDenegado
	}
	if errors.Is(err, domain.ErrLectorRelacionRPTInvalido) {
		return domain.ErrLectorRelacionRPTInvalido
	}
	return domain.ErrLectorRelacionRPTNoDisponible
}

var _ ports.LectorRelacionParaRPTV1 = (*ServicioLectorRelacionRPT)(nil)
