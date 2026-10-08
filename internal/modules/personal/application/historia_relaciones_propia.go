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

type ServicioHistoriaRelacionesPropia struct {
	autorizador ports.ProveedorAutorizacionHistoriaRelacionesPropia
	repositorio ports.RepositorioHistoriaRelacionesPropia
	intentos    ports.RegistroIntentosHistoriaRelacionesPropia
}

func NuevoServicioHistoriaRelacionesPropia(a ports.ProveedorAutorizacionHistoriaRelacionesPropia, r ports.RepositorioHistoriaRelacionesPropia, i ports.RegistroIntentosHistoriaRelacionesPropia) (*ServicioHistoriaRelacionesPropia, error) {
	if nulo(a) || nulo(r) || nulo(i) {
		return nil, domain.ErrHistoriaRelacionesPropiaNoDisponible
	}
	return &ServicioHistoriaRelacionesPropia{a, r, i}, nil
}
func (s *ServicioHistoriaRelacionesPropia) Consultar(ctx context.Context, in domain.SolicitudHistoriaRelacionesPropia) (ports.ResultadoHistoriaRelacionesPropia, error) {
	var cero ports.ResultadoHistoriaRelacionesPropia
	if s == nil || ctx == nil || nulo(s.autorizador) || nulo(s.repositorio) || nulo(s.intentos) {
		return cero, domain.ErrHistoriaRelacionesPropiaNoDisponible
	}
	fallar := func(err error) (ports.ResultadoHistoriaRelacionesPropia, error) {
		return cero, s.registrarFallo(ctx, err)
	}
	if err := ctx.Err(); err != nil {
		return fallar(err)
	}
	if s.intentos.VerificarRegistroHistoriaRelacionesPropia(ctx) != nil {
		return fallar(domain.ErrHistoriaRelacionesPropiaNoDisponible)
	}
	m, err := domain.NuevoMaterialHistoriaRelacionesPropia(in)
	if err != nil {
		return fallar(err)
	}
	a, err := s.autorizador.AutorizarHistoriaRelacionesPropia(ctx, m)
	if err != nil {
		return fallar(err)
	}
	if !AutorizacionHistoriaRelacionesPropiaValida(m, a) {
		return fallar(domain.ErrHistoriaRelacionesPropiaNoDisponible)
	}
	r, err := s.repositorio.ConsultarHistoriaRelacionesPropia(ctx, ports.OrdenHistoriaRelacionesPropia{Material: m, Autorizacion: a})
	if err != nil {
		return fallar(err)
	}
	if err = ctx.Err(); err != nil {
		return fallar(err)
	}
	if err = r.Historia.ValidarPara(m); err != nil {
		if errors.Is(err, domain.ErrHistoriaRelacionesPropiaExcedeLimite) {
			return fallar(err)
		}
		return fallar(domain.ErrHistoriaRelacionesPropiaNoDisponible)
	}
	if !ResultadoHistoriaRelacionesPropiaValido(m, a, r) {
		return fallar(domain.ErrHistoriaRelacionesPropiaNoDisponible)
	}
	return r, nil
}

// Estas comprobaciones ligan la respuesta; no reemplazan firma, gobierno,
// revocación ni consumo transaccional que debe verificar el adaptador real.
func AutorizacionHistoriaRelacionesPropiaValida(m domain.MaterialHistoriaRelacionesPropia, a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) bool {
	actor := m.Actor()
	canon, err := actor.RepresentacionCanonicaVinculadaV2()
	h, e := m.HuellaSHA256()
	x := a.ResumenCapacidad()
	return err == nil && e == nil && a.ValidarEstructura() == nil && bytes.Equal(canon, a.ContextoActorCanonico()) && a.PersonaVersion() == actor.Instantanea.PersonaVersion && a.PerfilVersion() == actor.Instantanea.PerfilVersion && x.Operacion() == domain.AccionHistoriaRelacionesPropia && x.AudienciaConsumo() == domain.AudienciaHistoriaRelacionesPropia && x.EfectoRef() == m.EmpleadoRef() && x.EfectoHuellaSHA256() == h
}
func ResultadoHistoriaRelacionesPropiaValido(m domain.MaterialHistoriaRelacionesPropia, a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, r ports.ResultadoHistoriaRelacionesPropia) bool {
	return AutorizacionHistoriaRelacionesPropiaValida(m, a) && r.Historia.ValidarPara(m) == nil && domain.ReciboHistoriaRelacionesPropiaLigado(r.Evidencia.ReciboRef, r.Evidencia.AuditoriaRef, r.Evidencia.ConsumoHuellaSHA256) && evidenciaFichaPropiaValida(a, r.Evidencia)
}
func errorHistoriaRelacionesPropia(ctx context.Context, e error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	for _, nominal := range []error{context.Canceled, context.DeadlineExceeded, domain.ErrHistoriaRelacionesPropiaInvalida, domain.ErrHistoriaRelacionesPropiaDenegada, domain.ErrHistoriaRelacionesPropiaExcedeLimite} {
		if errors.Is(e, nominal) {
			return nominal
		}
	}
	return domain.ErrHistoriaRelacionesPropiaNoDisponible
}
func (s *ServicioHistoriaRelacionesPropia) registrarFallo(ctx context.Context, e error) error {
	nominal := errorHistoriaRelacionesPropia(ctx, e)
	motivo := "no_disponible"
	if errors.Is(nominal, domain.ErrHistoriaRelacionesPropiaInvalida) {
		motivo = "entrada_invalida"
	}
	if errors.Is(nominal, domain.ErrHistoriaRelacionesPropiaDenegada) {
		motivo = "denegado"
	}
	// El puerto de lectura ya ha cerrado su TX. La cancelación no borra el
	// intento; WithoutCancel conserva la captura de identidad del servidor.
	auditCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()
	if s.intentos.RegistrarIntentoHistoriaRelacionesPropia(auditCtx, ports.IntentoHistoriaRelacionesPropia{Motivo: motivo}) != nil {
		return domain.ErrHistoriaRelacionesPropiaNoDisponible
	}
	return nominal
}
