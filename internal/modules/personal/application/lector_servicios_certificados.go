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

// ServicioLectorServiciosCertificados implementa los contratos V1 y V2 que
// Personal publica para Certificados. Sólo autoservicio en este corte.
type ServicioLectorServiciosCertificados struct {
	autorizador ports.ProveedorAutorizacionLectorServiciosCertificados
	repositorio ports.RepositorioLectorServiciosCertificados
	intentos    ports.RegistroIntentosLectorServiciosCertificados
	ahora       func() time.Time
}

func NuevoServicioLectorServiciosCertificados(a ports.ProveedorAutorizacionLectorServiciosCertificados, r ports.RepositorioLectorServiciosCertificados, i ports.RegistroIntentosLectorServiciosCertificados, ahora func() time.Time) (*ServicioLectorServiciosCertificados, error) {
	if nulo(a) || nulo(r) || nulo(i) || ahora == nil {
		return nil, domain.ErrLectorServiciosCertificadosNoDisponible
	}
	return &ServicioLectorServiciosCertificados{a, r, i, ahora}, nil
}

// ConsultarServiciosParaCertificados devuelve V1: la misma lectura V2, sin días.
func (s *ServicioLectorServiciosCertificados) ConsultarServiciosParaCertificados(ctx context.Context, in ports.ConsultaServiciosParaCertificadosV1) (ports.ResultadoServiciosParaCertificadosV1, error) {
	r, err := s.ConsultarServiciosParaCertificadosV2(ctx, in)
	if err != nil {
		return ports.ResultadoServiciosParaCertificadosV1{}, err
	}
	return r.V1(), nil
}

func (s *ServicioLectorServiciosCertificados) ConsultarServiciosParaCertificadosV2(ctx context.Context, in ports.ConsultaServiciosParaCertificadosV1) (ports.ResultadoServiciosParaCertificadosV2, error) {
	var cero ports.ResultadoServiciosParaCertificadosV2
	if s == nil || ctx == nil || nulo(s.autorizador) || nulo(s.repositorio) || nulo(s.intentos) || s.ahora == nil {
		return cero, domain.ErrLectorServiciosCertificadosNoDisponible
	}
	fallar := func(err error) (ports.ResultadoServiciosParaCertificadosV2, error) {
		return cero, s.registrarFallo(ctx, err)
	}
	if err := ctx.Err(); err != nil {
		return fallar(err)
	}
	if s.intentos.VerificarRegistroServiciosCertificados(ctx) != nil {
		return fallar(domain.ErrLectorServiciosCertificadosNoDisponible)
	}
	m, err := domain.NuevoMaterialLectorServiciosCertificados(domain.SolicitudLectorServiciosCertificados{Actor: in.Actor, EmpleadoRef: in.EmpleadoRef, OrganismoRef: in.OrganismoRef, Corte: in.Corte})
	if err != nil {
		return fallar(err)
	}
	instante := s.ahora().UTC().Truncate(time.Microsecond)
	if !m.ActorVigenteEn(instante) || m.Corte().ConocidoEn.After(instante) {
		return fallar(domain.ErrLectorServiciosCertificadosDenegado)
	}
	a, err := s.autorizador.AutorizarServiciosParaCertificados(ctx, m)
	if err != nil {
		return fallar(err)
	}
	if !AutorizacionLectorServiciosCertificadosValida(m, a) {
		return fallar(domain.ErrLectorServiciosCertificadosNoDisponible)
	}
	instante = s.ahora().UTC().Truncate(time.Microsecond)
	x := a.ResumenCapacidad()
	if !m.ActorVigenteEn(instante) || instante.Before(x.EmitidaEn()) || !instante.Before(x.ExpiraEn()) {
		return fallar(domain.ErrLectorServiciosCertificadosDenegado)
	}
	r, err := s.repositorio.ConsultarServiciosParaCertificados(ctx, ports.OrdenLectorServiciosCertificados{Material: m, Autorizacion: a})
	if err != nil {
		return fallar(err)
	}
	if err := ctx.Err(); err != nil {
		return fallar(err)
	}
	instante = s.ahora().UTC().Truncate(time.Microsecond)
	if !m.ActorVigenteEn(instante) || !instante.Before(x.ExpiraEn()) {
		return fallar(domain.ErrLectorServiciosCertificadosDenegado)
	}
	if !ResultadoLectorServiciosCertificadosValido(m, a, r) || r.Evidencia.ConsultadaEn.After(instante) {
		return fallar(domain.ErrLectorServiciosCertificadosNoDisponible)
	}
	return copiarResultadoServiciosCertificados(r), nil
}

// Estas comprobaciones también las invoca el adaptador antes de confirmar.
// No verifican COSE ni conceden acceso: la fachada SQL revalida y consume V3.
func AutorizacionLectorServiciosCertificadosValida(m domain.MaterialLectorServiciosCertificados, a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) bool {
	actor := m.Actor()
	canon, err := actor.RepresentacionCanonicaVinculadaV2()
	h, errH := m.HuellaSHA256()
	x := a.ResumenCapacidad()
	return err == nil && errH == nil && a.ValidarEstructura() == nil && bytes.Equal(canon, a.ContextoActorCanonico()) &&
		a.PersonaVersion() == actor.Instantanea.PersonaVersion && a.PerfilVersion() == actor.Instantanea.PerfilVersion &&
		x.Operacion() == ports.AccionServiciosParaCertificadosV1 && x.AudienciaConsumo() == ports.AudienciaServiciosParaCertificadosV1 &&
		x.EfectoRef() == m.EmpleadoRef() && x.EfectoHuellaSHA256() == h
}

// ResultadoLectorServiciosCertificadosValido liga la respuesta a la consulta:
// mismo empleado, organismo y corte; cobertura y certeza no acreditadas (la
// fuente carece de eficacia administrativa); servicios únicos, ordenados por
// inicio y referencia y ya comenzados al corte; recibo igual a la auditoría común.
func ResultadoLectorServiciosCertificadosValido(m domain.MaterialLectorServiciosCertificados, a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, r ports.ResultadoServiciosParaCertificadosV2) bool {
	if !AutorizacionLectorServiciosCertificadosValida(m, a) || r.EmpleadoRef != m.EmpleadoRef() || r.OrganismoRef != m.OrganismoRef() ||
		!m.CoincideCorte(r.Corte) || r.Cobertura != ports.CoberturaPersonalNoAcreditadaV1 || r.Version < int64(len(r.Servicios)) ||
		r.Servicios == nil || len(r.Servicios) > domain.LimiteLectorServiciosCertificados ||
		!domain.ReciboHistoriaServiciosPropiaLigado(r.Evidencia.ReciboRef, r.Evidencia.AuditoriaRef, r.Evidencia.ConsumoHuellaSHA256) ||
		!evidenciaFichaPropiaValida(a, r.Evidencia) || !m.ActorVigenteEn(r.Evidencia.ConsultadaEn) || r.Evidencia.ConsultadaEn.Before(m.Corte().ConocidoEn) {
		return false
	}
	vistos := make(map[string]struct{}, len(r.Servicios))
	for i, s := range r.Servicios {
		p := s.Procedencia
		if domain.ValidarServicioLeidoCertificados(domain.ServicioLeidoCertificados{ServicioRef: s.ServicioRef, RelacionRef: s.RelacionRef, Estado: s.Estado,
			ClaseRef: s.ClaseRef, Version: s.Version, DiasReconocidos: s.DiasReconocidos, ClaseVersion: s.ClaseVersion,
			Desde: s.Periodo.Desde, Hasta: s.Periodo.Hasta, ActoRef: p.ActoRef, FuenteRef: p.FuenteRef, FuenteVersion: p.FuenteVersion}, r.Corte) != nil ||
			p.Certeza != ports.CertezaPersonalNoAcreditadaV1 {
			return false
		}
		if _, repetido := vistos[s.ServicioRef]; repetido {
			return false
		}
		vistos[s.ServicioRef] = struct{}{}
		if i > 0 {
			ant := r.Servicios[i-1]
			if s.Periodo.Desde.AntesDe(ant.Periodo.Desde) || (s.Periodo.Desde == ant.Periodo.Desde && s.ServicioRef < ant.ServicioRef) {
				return false
			}
		}
	}
	return true
}

func copiarResultadoServiciosCertificados(r ports.ResultadoServiciosParaCertificadosV2) ports.ResultadoServiciosParaCertificadosV2 {
	r.Servicios = append(make([]ports.ServicioParaCertificadosV2, 0, len(r.Servicios)), r.Servicios...)
	return r
}

func (s *ServicioLectorServiciosCertificados) registrarFallo(ctx context.Context, err error) error {
	nominal := errorLectorServiciosCertificadosNominal(ctx, err)
	motivo := "no_disponible"
	if errors.Is(nominal, domain.ErrLectorServiciosCertificadosInvalido) {
		motivo = "entrada_invalida"
	}
	if errors.Is(nominal, domain.ErrLectorServiciosCertificadosDenegado) {
		motivo = "denegado"
	}
	// El repositorio ya cerró su transacción. Una cancelación no borra el
	// intento: WithoutCancel conserva la identidad capturada en la frontera.
	auditCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()
	if s.intentos.RegistrarIntentoServiciosCertificados(auditCtx, ports.IntentoLectorServiciosCertificados{Motivo: motivo}) != nil {
		return domain.ErrLectorServiciosCertificadosNoDisponible
	}
	return nominal
}

func errorLectorServiciosCertificadosNominal(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	for _, nominal := range []error{context.Canceled, context.DeadlineExceeded, domain.ErrLectorServiciosCertificadosDenegado,
		domain.ErrLectorServiciosCertificadosInvalido, domain.ErrLectorServiciosCertificadosExcedeLimite} {
		if errors.Is(err, nominal) {
			return nominal
		}
	}
	return domain.ErrLectorServiciosCertificadosNoDisponible
}

var (
	_ ports.LectorServiciosParaCertificadosV1 = (*ServicioLectorServiciosCertificados)(nil)
	_ ports.LectorServiciosParaCertificadosV2 = (*ServicioLectorServiciosCertificados)(nil)
)
