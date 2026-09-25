package application

import (
	"context"
	"errors"
	"regexp"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var huellaRegistroB2 = regexp.MustCompile(`^[a-f0-9]{64}$`)

type ServicioRegistroEmpleadoB2 struct {
	autorizador ports.ProveedorAutorizacionRegistroEmpleadoB2
	repositorio ports.RepositorioRegistroEmpleadoB2
}

func NuevoServicioRegistroEmpleadoB2(a ports.ProveedorAutorizacionRegistroEmpleadoB2, r ports.RepositorioRegistroEmpleadoB2) (*ServicioRegistroEmpleadoB2, error) {
	if nulo(a) || nulo(r) {
		return nil, domain.ErrRegistroEmpleadoB2NoDisponible
	}
	return &ServicioRegistroEmpleadoB2{a, r}, nil
}

func (s *ServicioRegistroEmpleadoB2) ConsultarFicha(ctx context.Context, solicitud domain.SolicitudFichaEmpleadoB2) (ports.ResultadoFichaEmpleadoB2, error) {
	var vacio ports.ResultadoFichaEmpleadoB2
	if s == nil || ctx == nil || nulo(s.autorizador) || nulo(s.repositorio) {
		return vacio, domain.ErrRegistroEmpleadoB2NoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	material, err := domain.NuevoMaterialFichaEmpleadoB2(solicitud)
	if err != nil {
		return vacio, domain.ErrRegistroEmpleadoB2Invalido
	}
	autorizacion, err := s.autorizador.AutorizarConsultaRegistroEmpleadoB2(ctx, material)
	if err != nil {
		return vacio, errorRegistroB2Opaco(ctx, err)
	}
	if !autorizacionRegistroB2Valida(material, autorizacion) {
		return vacio, domain.ErrRegistroEmpleadoB2Denegado
	}
	resultado, err := s.repositorio.ConsultarFichaRRHH(ctx, ports.OrdenFichaEmpleadoB2{Material: material, Autorizacion: autorizacion})
	if err != nil {
		return vacio, errorRegistroB2Opaco(ctx, err)
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	if resultado.Ficha.ValidarPara(material) != nil || !evidenciaRegistroB2Valida(material, autorizacion, resultado.Evidencia) {
		return vacio, domain.ErrRegistroEmpleadoB2NoDisponible
	}
	return resultado, nil
}

func (s *ServicioRegistroEmpleadoB2) ConsultarVacantes(ctx context.Context, solicitud domain.SolicitudVacantesB2) (ports.ResultadoVacantesB2, error) {
	var vacio ports.ResultadoVacantesB2
	if s == nil || ctx == nil || nulo(s.autorizador) || nulo(s.repositorio) {
		return vacio, domain.ErrRegistroEmpleadoB2NoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	material, err := domain.NuevoMaterialVacantesB2(solicitud)
	if err != nil {
		return vacio, domain.ErrRegistroEmpleadoB2Invalido
	}
	autorizacion, err := s.autorizador.AutorizarConsultaRegistroEmpleadoB2(ctx, material)
	if err != nil {
		return vacio, errorRegistroB2Opaco(ctx, err)
	}
	if !autorizacionRegistroB2Valida(material, autorizacion) {
		return vacio, domain.ErrRegistroEmpleadoB2Denegado
	}
	resultado, err := s.repositorio.ListarVacantesRRHH(ctx, ports.OrdenVacantesB2{Material: material, Autorizacion: autorizacion})
	if err != nil {
		return vacio, errorRegistroB2Opaco(ctx, err)
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	if resultado.Pagina.ValidarPara(material) != nil || !evidenciaRegistroB2Valida(material, autorizacion, resultado.Evidencia) {
		return vacio, domain.ErrRegistroEmpleadoB2NoDisponible
	}
	return resultado, nil
}

func autorizacionRegistroB2Valida(m domain.MaterialConsultaRegistroEmpleadoB2, a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) bool {
	actor := m.Actor()
	r := m.Recurso()
	h, err := m.HuellaSHA256()
	x := a.ResumenCapacidad()
	accion, audiencia := domain.AccionFichaEmpleadoB2, domain.AudienciaFichaEmpleadoB2
	switch m.Operacion() {
	case "vacantes":
		accion, audiencia = domain.AccionVacantesB2, domain.AudienciaVacantesB2
	case "empleados":
		accion, audiencia = domain.AccionEmpleadosB2, domain.AudienciaEmpleadosB2
	}
	return err == nil && a.ValidarEstructura() == nil &&
		a.PersonaVersion() == actor.Instantanea.PersonaVersion && a.PerfilVersion() == actor.Instantanea.PerfilVersion &&
		x.Operacion() == accion && x.AudienciaConsumo() == audiencia && x.EfectoRef() == r.Referencia && x.EfectoHuellaSHA256() == h
}

func evidenciaRegistroB2Valida(m domain.MaterialConsultaRegistroEmpleadoB2, a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, e ports.EvidenciaRegistroEmpleadoB2) bool {
	x := a.ResumenCapacidad()
	_, offset := e.ConsultadaEn.Zone()
	return e.ReciboRef != "" && len(e.ReciboRef) <= 160 && e.AuditoriaRef != "" && len(e.AuditoriaRef) <= 160 &&
		huellaRegistroB2.MatchString(e.ConsumoHuellaSHA256) && e.DecisionRef == x.DecisionRef() && e.EfectoRef == x.EfectoRef() &&
		!e.ConsultadaEn.IsZero() && offset == 0 && e.ConsultadaEn.Nanosecond()%1000 == 0 &&
		!e.ConsultadaEn.Before(x.EmitidaEn()) && e.ConsultadaEn.Before(x.ExpiraEn())
}

func errorRegistroB2Opaco(ctx context.Context, err error) error {
	if e := ctx.Err(); e != nil {
		return e
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if errors.Is(err, domain.ErrRegistroEmpleadoB2Denegado) {
		return domain.ErrRegistroEmpleadoB2Denegado
	}
	if errors.Is(err, domain.ErrRegistroEmpleadoB2NoEncontrado) {
		return domain.ErrRegistroEmpleadoB2NoEncontrado
	}
	if errors.Is(err, domain.ErrCoberturaVacantesB2NoAcreditada) {
		return domain.ErrCoberturaVacantesB2NoAcreditada
	}
	if errors.Is(err, domain.ErrRegistroEmpleadoB2Conflicto) {
		return domain.ErrRegistroEmpleadoB2Conflicto
	}
	return domain.ErrRegistroEmpleadoB2NoDisponible
}
