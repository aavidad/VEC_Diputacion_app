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

type ServicioHistoriaServiciosPropia struct {
	autorizador ports.ProveedorAutorizacionHistoriaServiciosPropia
	repositorio ports.RepositorioHistoriaServiciosPropia
	intentos    ports.RegistroIntentosHistoriaServiciosPropia
}

func NuevoServicioHistoriaServiciosPropia(a ports.ProveedorAutorizacionHistoriaServiciosPropia, r ports.RepositorioHistoriaServiciosPropia, i ports.RegistroIntentosHistoriaServiciosPropia) (*ServicioHistoriaServiciosPropia, error) {
	if nulo(a) || nulo(r) || nulo(i) {
		return nil, domain.ErrHistoriaServiciosPropiaNoDisponible
	}
	return &ServicioHistoriaServiciosPropia{a, r, i}, nil
}
func (s *ServicioHistoriaServiciosPropia) Consultar(ctx context.Context, in domain.SolicitudHistoriaServiciosPropia) (ports.ResultadoHistoriaServiciosPropia, error) {
	var cero ports.ResultadoHistoriaServiciosPropia
	if s == nil || ctx == nil || nulo(s.autorizador) || nulo(s.repositorio) || nulo(s.intentos) {
		return cero, domain.ErrHistoriaServiciosPropiaNoDisponible
	}
	fallar := func(err error) (ports.ResultadoHistoriaServiciosPropia, error) {
		return cero, s.registrarFallo(ctx, err)
	}
	if err := ctx.Err(); err != nil {
		return fallar(err)
	}
	if s.intentos.VerificarRegistroHistoriaServiciosPropia(ctx) != nil {
		return fallar(domain.ErrHistoriaServiciosPropiaNoDisponible)
	}
	m, err := domain.NuevoMaterialHistoriaServiciosPropia(in)
	if err != nil {
		return fallar(err)
	}
	a, err := s.autorizador.AutorizarHistoriaServiciosPropia(ctx, m)
	if err != nil {
		return fallar(err)
	}
	if !AutorizacionHistoriaServiciosPropiaValida(m, a) {
		return fallar(domain.ErrHistoriaServiciosPropiaNoDisponible)
	}
	r, err := s.repositorio.ConsultarHistoriaServiciosPropia(ctx, ports.OrdenHistoriaServiciosPropia{Material: m, Autorizacion: a})
	if err != nil {
		return fallar(err)
	}
	if err = ctx.Err(); err != nil {
		return fallar(err)
	}
	if err = r.Historia.ValidarPara(m); err != nil {
		if errors.Is(err, domain.ErrHistoriaServiciosPropiaExcedeLimite) {
			return fallar(err)
		}
		return fallar(domain.ErrHistoriaServiciosPropiaNoDisponible)
	}
	if !ResultadoHistoriaServiciosPropiaValido(m, a, r) {
		return fallar(domain.ErrHistoriaServiciosPropiaNoDisponible)
	}
	return r, nil
}

// Estas comprobaciones ligan la respuesta; no reemplazan firma, gobierno,
// revocación ni consumo transaccional que debe verificar el adaptador real.
func AutorizacionHistoriaServiciosPropiaValida(m domain.MaterialHistoriaServiciosPropia, a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) bool {
	actor := m.Actor()
	canon, err := actor.RepresentacionCanonicaVinculadaV2()
	h, e := m.HuellaSHA256()
	x := a.ResumenCapacidad()
	return err == nil && e == nil && a.ValidarEstructura() == nil && bytes.Equal(canon, a.ContextoActorCanonico()) && a.PersonaVersion() == actor.Instantanea.PersonaVersion && a.PerfilVersion() == actor.Instantanea.PerfilVersion && x.Operacion() == domain.AccionHistoriaServiciosPropia && x.AudienciaConsumo() == domain.AudienciaHistoriaServiciosPropia && x.EfectoRef() == m.EmpleadoRef() && x.EfectoHuellaSHA256() == h
}
func ResultadoHistoriaServiciosPropiaValido(m domain.MaterialHistoriaServiciosPropia, a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, r ports.ResultadoHistoriaServiciosPropia) bool {
	return AutorizacionHistoriaServiciosPropiaValida(m, a) && r.Historia.ValidarPara(m) == nil && domain.ReciboHistoriaServiciosPropiaLigado(r.Evidencia.ReciboRef, r.Evidencia.AuditoriaRef, r.Evidencia.ConsumoHuellaSHA256) && evidenciaFichaPropiaValida(a, r.Evidencia)
}
func errorHistoriaServiciosPropia(ctx context.Context, e error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	for _, nominal := range []error{context.Canceled, context.DeadlineExceeded, domain.ErrHistoriaServiciosPropiaInvalida, domain.ErrHistoriaServiciosPropiaDenegada, domain.ErrHistoriaServiciosPropiaExcedeLimite} {
		if errors.Is(e, nominal) {
			return nominal
		}
	}
	return domain.ErrHistoriaServiciosPropiaNoDisponible
}
func (s *ServicioHistoriaServiciosPropia) registrarFallo(ctx context.Context, e error) error {
	nominal := errorHistoriaServiciosPropia(ctx, e)
	motivo := "no_disponible"
	if errors.Is(nominal, domain.ErrHistoriaServiciosPropiaInvalida) {
		motivo = "entrada_invalida"
	}
	if errors.Is(nominal, domain.ErrHistoriaServiciosPropiaDenegada) {
		motivo = "denegado"
	}
	// El puerto de lectura ya ha cerrado su TX. La cancelación no borra el
	// intento; WithoutCancel conserva la captura de identidad del servidor.
	auditCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()
	if s.intentos.RegistrarIntentoHistoriaServiciosPropia(auditCtx, ports.IntentoHistoriaServiciosPropia{Motivo: motivo}) != nil {
		return domain.ErrHistoriaServiciosPropiaNoDisponible
	}
	return nominal
}
