package application

import (
	"context"
	"errors"
	"reflect"
	"regexp"
	"time"
	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type ServicioConsultaRelacionEmpleado struct {
	proveedor   personalports.ProveedorAutorizacionConsultaRelacionPropia
	repositorio personalports.RepositorioRelacionesEmpleado
}

func NuevoServicioConsultaRelacionEmpleado(p personalports.ProveedorAutorizacionConsultaRelacionPropia, r personalports.RepositorioRelacionesEmpleado) (*ServicioConsultaRelacionEmpleado, error) {
	if nulo(p) || nulo(r) {
		return nil, personalports.ErrRelacionEmpleadoNoDisponible
	}
	return &ServicioConsultaRelacionEmpleado{p, r}, nil
}
func (s *ServicioConsultaRelacionEmpleado) ConsultarPropiasParaDietas(ctx context.Context, c personaldomain.SolicitudConsultaRelacionPropia) (personalports.ResultadoConsultaRelacionPropia, error) {
	var z personalports.ResultadoConsultaRelacionPropia
	if s == nil || ctx == nil || nulo(s.proveedor) || nulo(s.repositorio) {
		return z, personalports.ErrRelacionEmpleadoNoDisponible
	}
	if e := ctx.Err(); e != nil {
		return z, e
	}
	m, e := personaldomain.NuevoMaterialConsultaRelacionPropia(c)
	if e != nil {
		return z, personalports.ErrConsultaRelacionEmpleadoInvalida
	}
	a, e := s.proveedor.AutorizarConsultaRelacionPropia(ctx, m)
	if e != nil {
		return z, opaco(ctx, e)
	}
	if !autorizacionValida(m, a) {
		return z, personalports.ErrRelacionEmpleadoNoDisponible
	}
	r, e := s.repositorio.ConsultarRelacionesPropiasDietas(ctx, personalports.OrdenConsultaRelacionPropia{Material: m, Autorizacion: a})
	if e != nil {
		return z, opaco(ctx, e)
	}
	if !resultadoValido(m, r, a) {
		return z, personalports.ErrRelacionEmpleadoNoDisponible
	}
	return r, nil
}
func autorizacionValida(m personaldomain.MaterialConsultaRelacionPropia, a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) bool {
	c := m.Solicitud()
	r := m.Recurso()
	h, e := m.HuellaSHA256()
	x := a.ResumenCapacidad()
	return e == nil && a.ValidarEstructura() == nil && a.PersonaVersion() == c.Actor.Instantanea.PersonaVersion && a.PerfilVersion() == c.Actor.Instantanea.PerfilVersion && x.Operacion() == "personal.relacion.propia.consultar_dietas" && x.AudienciaConsumo() == "vec_personal.relacion_propia.consultar_dietas.v1" && x.EfectoRef() == r.Referencia && x.EfectoHuellaSHA256() == h
}
func resultadoValido(m personaldomain.MaterialConsultaRelacionPropia, r personalports.ResultadoConsultaRelacionPropia, a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) bool {
	c := m.Solicitud()
	e, er := c.Actor.Referencias("empleado")
	resumen := a.ResumenCapacidad()
	_, offset := r.Evidencia.ConsultadaEn.Zone()
	if er != nil || len(e) != 1 || len(r.Relaciones) > 100 || !reciboConsultaValido.MatchString(r.Evidencia.ReciboRef) || !huellaConsultaValida.MatchString(r.Evidencia.ConsumoHuellaSHA256) || r.Evidencia.AuditoriaRef == "" || r.Evidencia.ConsultadaEn.IsZero() || r.Evidencia.ConsultadaEn.Location() != time.UTC || offset != 0 || r.Evidencia.ConsultadaEn.Nanosecond()%1000 != 0 || r.Evidencia.ConsultadaEn.Before(resumen.EmitidaEn()) || !r.Evidencia.ConsultadaEn.Before(resumen.ExpiraEn()) || r.Evidencia.DecisionRef != resumen.DecisionRef() || r.Evidencia.EfectoRef != resumen.EfectoRef() {
		return false
	}
	for _, x := range r.Relaciones {
		if x.Validar() != nil || x.PersonaRef != c.Actor.PersonaRef || x.EmpleadoRef != e[0] || !x.VigenteEn(c.FechaReferencia) || (c.Operacion == personaldomain.OperacionDetalleRelacionPropia && x.RelacionRef != c.RelacionRef) {
			return false
		}
	}
	return true
}

var reciboConsultaValido = regexp.MustCompile(`^rpd_[0-9a-f]{32}$`)
var huellaConsultaValida = regexp.MustCompile(`^[a-f0-9]{64}$`)

func opaco(ctx context.Context, e error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(e, context.Canceled) || errors.Is(e, context.DeadlineExceeded) {
		return e
	}
	return personalports.ErrRelacionEmpleadoNoDisponible
}
func nulo(v any) bool {
	if v == nil {
		return true
	}
	x := reflect.ValueOf(v)
	return (x.Kind() == reflect.Ptr || x.Kind() == reflect.Interface || x.Kind() == reflect.Func || x.Kind() == reflect.Map || x.Kind() == reflect.Slice) && x.IsNil()
}
