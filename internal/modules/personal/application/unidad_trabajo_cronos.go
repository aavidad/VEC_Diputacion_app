package application

import (
	"bytes"
	"context"
	"errors"
	"regexp"
	"strconv"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type ServicioUnidadTrabajoCronos struct {
	autorizador ports.ProveedorAutorizacionUnidadTrabajoCronos
	repositorio ports.RepositorioUnidadTrabajoCronos
}

var referenciaUnidadCronos = regexp.MustCompile(`^[a-z][a-z0-9_:-]{2,127}$`)
var fuenteUnidadCronos = regexp.MustCompile(`^[a-z][a-z0-9_:-]{2,159}$`)
var reciboUnidadCronos = regexp.MustCompile(`^unidadcronos:[0-9a-f-]{36}$`)

func NuevoServicioUnidadTrabajoCronos(a ports.ProveedorAutorizacionUnidadTrabajoCronos, r ports.RepositorioUnidadTrabajoCronos) (*ServicioUnidadTrabajoCronos, error) {
	if nulo(a) || nulo(r) {
		return nil, domain.ErrUnidadTrabajoCronosNoDisponible
	}
	return &ServicioUnidadTrabajoCronos{a, r}, nil
}

func (s *ServicioUnidadTrabajoCronos) ConsultarUnidadTrabajoPropiaCronos(ctx context.Context, in ports.ConsultaUnidadTrabajoPropiaCronos) (ports.ResultadoUnidadTrabajoPropiaCronos, error) {
	var cero ports.ResultadoUnidadTrabajoPropiaCronos
	if s == nil || ctx == nil || nulo(s.autorizador) || nulo(s.repositorio) {
		return cero, domain.ErrUnidadTrabajoCronosNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	m, err := domain.NuevoMaterialUnidadTrabajoCronos(in.Actor, in.Fecha, in.ConocidoEn)
	if err != nil {
		return cero, err
	}
	a, err := s.autorizador.AutorizarUnidadTrabajoPropiaCronos(ctx, m)
	if err != nil {
		return cero, errorUnidadTrabajoCronos(ctx, err)
	}
	if !AutorizacionUnidadTrabajoCronosValida(m, a) {
		return cero, domain.ErrUnidadTrabajoCronosNoDisponible
	}
	r, err := s.repositorio.ConsultarUnidadTrabajoPropiaCronos(ctx, ports.OrdenUnidadTrabajoCronos{Material: m, Autorizacion: a})
	if err != nil {
		return cero, errorUnidadTrabajoCronos(ctx, err)
	}
	if !ResultadoUnidadTrabajoCronosValido(m, a, r) {
		return cero, domain.ErrUnidadTrabajoCronosNoDisponible
	}
	return r, nil
}

func AutorizacionUnidadTrabajoCronosValida(m domain.MaterialUnidadTrabajoCronos, a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) bool {
	actor := m.Actor()
	canon, err := actor.RepresentacionCanonicaVinculadaV2()
	huella, e := m.HuellaSHA256()
	x := a.ResumenCapacidad()
	return err == nil && e == nil && a.ValidarEstructura() == nil && bytes.Equal(canon, a.ContextoActorCanonico()) &&
		a.PersonaVersion() == actor.Instantanea.PersonaVersion && a.PerfilVersion() == actor.Instantanea.PerfilVersion &&
		x.Operacion() == ports.AccionUnidadTrabajoPropiaCronos && x.AudienciaConsumo() == ports.AudienciaUnidadTrabajoPropiaCronos &&
		x.EfectoRef() == m.EmpleadoRef() && x.EfectoHuellaSHA256() == huella
}

func ResultadoUnidadTrabajoCronosValido(m domain.MaterialUnidadTrabajoCronos, a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, r ports.ResultadoUnidadTrabajoPropiaCronos) bool {
	if !AutorizacionUnidadTrabajoCronosValida(m, a) || !evidenciaFichaPropiaValida(a, r.Evidencia) || r.Evidencia.ConsultadaEn.Before(m.ConocidoEn()) ||
		!reciboUnidadCronos.MatchString(r.Evidencia.ReciboRef) || r.Evidencia.AuditoriaRef != "aud_v3_"+r.Evidencia.ConsumoHuellaSHA256[:32] {
		return false
	}
	switch r.Cobertura {
	case "completa":
		if r.Unidad == nil {
			return false
		}
		u := r.Unidad
		fuenteVersion, err := strconv.ParseInt(u.FuenteVersionRef, 10, 64)
		return u.EmpleadoRef == m.EmpleadoRef() && domain.ReferenciaRelacionValida(u.RelacionRef) && referenciaUnidadCronos.MatchString(u.OrganismoRef) && referenciaUnidadCronos.MatchString(u.UnidadClave) &&
			u.Desde.Validar() == nil && !m.Fecha().AntesDe(u.Desde) && (u.HastaExclusivo == "" || (u.HastaExclusivo.Validar() == nil && m.Fecha().AntesDe(u.HastaExclusivo))) &&
			fuenteUnidadCronos.MatchString(u.FuenteRef) && err == nil && fuenteVersion > 0 && u.FuenteVersionRef == strconv.FormatInt(fuenteVersion, 10) && u.Version > 0
	case "sin_datos", "no_acreditada":
		return r.Unidad == nil
	default:
		return false
	}
}

func errorUnidadTrabajoCronos(ctx context.Context, err error) error {
	if e := ctx.Err(); e != nil {
		return e
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if errors.Is(err, domain.ErrUnidadTrabajoCronosDenegada) {
		return domain.ErrUnidadTrabajoCronosDenegada
	}
	return domain.ErrUnidadTrabajoCronosNoDisponible
}

var _ ports.LectorUnidadTrabajoPropiaCronos = (*ServicioUnidadTrabajoCronos)(nil)
