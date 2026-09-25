package application

import (
	"context"
	"errors"
	"regexp"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var reciboActoB2Valido = regexp.MustCompile(`^perrec_[0-9a-f]{32}$`)

type ServicioActosRegistroEmpleadoB2 struct {
	autorizador ports.ProveedorAutorizacionActosRegistroEmpleadoB2
	repositorio ports.RepositorioActosRegistroEmpleadoB2
}

func NuevoServicioActosRegistroEmpleadoB2(v ports.ProveedorAutorizacionActosRegistroEmpleadoB2, r ports.RepositorioActosRegistroEmpleadoB2) (*ServicioActosRegistroEmpleadoB2, error) {
	if nulo(v) || nulo(r) {
		return nil, domain.ErrRegistroEmpleadoB2NoDisponible
	}
	return &ServicioActosRegistroEmpleadoB2{v, r}, nil
}

func (s *ServicioActosRegistroEmpleadoB2) RegistrarEmpleado(ctx context.Context, solicitud domain.SolicitudAltaEmpleadoB2) (ports.ResultadoAltaEmpleadoB2, error) {
	var vacio ports.ResultadoAltaEmpleadoB2
	if s == nil || ctx == nil || nulo(s.autorizador) || nulo(s.repositorio) {
		return vacio, domain.ErrRegistroEmpleadoB2NoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	// La función SQL consume V3 y acredita la persona objetivo por B1 en la
	// misma transacción. No existe lectura runtime de terceros anterior a V3.
	material, err := domain.NuevoMaterialAltaEmpleadoB2(solicitud)
	if err != nil {
		return vacio, domain.ErrRegistroEmpleadoB2Invalido
	}
	autorizacion, err := s.autorizador.AutorizarActoRegistroEmpleadoB2(ctx, material)
	if err != nil {
		return vacio, errorRegistroB2Opaco(ctx, err)
	}
	if !autorizacionActoRegistroB2Valida(material, autorizacion) {
		return vacio, domain.ErrRegistroEmpleadoB2Denegado
	}
	resultado, err := s.repositorio.RegistrarEmpleadoRRHH(ctx, ports.OrdenAltaEmpleadoB2{Material: material, Autorizacion: autorizacion})
	if err != nil {
		return vacio, errorRegistroB2Opaco(ctx, err)
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	if !reciboActoRegistroB2ValidoPara(material, resultado.Recibo, "alta", "", 1) || !accesoActualRegistroB2Valido(material, autorizacion, resultado.AccesoActual) {
		return vacio, domain.ErrRegistroEmpleadoB2NoDisponible
	}
	return resultado, nil
}

func (s *ServicioActosRegistroEmpleadoB2) RegistrarHecho(ctx context.Context, solicitud domain.SolicitudHechoEmpleadoB2) (ports.ResultadoHechoEmpleadoB2, error) {
	var vacio ports.ResultadoHechoEmpleadoB2
	if s == nil || ctx == nil || nulo(s.autorizador) || nulo(s.repositorio) {
		return vacio, domain.ErrRegistroEmpleadoB2NoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	material, err := domain.NuevoMaterialHechoEmpleadoB2(solicitud)
	if err != nil {
		return vacio, domain.ErrRegistroEmpleadoB2Invalido
	}
	autorizacion, err := s.autorizador.AutorizarActoRegistroEmpleadoB2(ctx, material)
	if err != nil {
		return vacio, errorRegistroB2Opaco(ctx, err)
	}
	if !autorizacionActoRegistroB2Valida(material, autorizacion) {
		return vacio, domain.ErrRegistroEmpleadoB2Denegado
	}
	resultado, err := s.repositorio.RegistrarHechoEmpleadoRRHH(ctx, ports.OrdenHechoEmpleadoB2{Material: material, Autorizacion: autorizacion})
	if err != nil {
		return vacio, errorRegistroB2Opaco(ctx, err)
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	versionResultado := int64(1)
	if solicitud.Tipo == "relacion" && solicitud.RelacionRef != "" {
		versionResultado = solicitud.RelacionVersionEsperada + 1
	}
	if !reciboActoRegistroB2ValidoPara(material, resultado.Recibo, solicitud.Tipo, solicitud.RelacionRef, versionResultado) || !accesoActualRegistroB2Valido(material, autorizacion, resultado.AccesoActual) {
		return vacio, domain.ErrRegistroEmpleadoB2NoDisponible
	}
	return resultado, nil
}

func autorizacionActoRegistroB2Valida(m domain.MaterialActoRegistroEmpleadoB2, a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) bool {
	actor := m.Actor()
	r := m.Recurso()
	h, err := m.HuellaSHA256()
	x := a.ResumenCapacidad()
	accion, audiencia := domain.AccionAltaEmpleadoB2, domain.AudienciaAltaEmpleadoB2
	if m.Tipo() == "hecho" {
		accion, audiencia = domain.AccionHechoEmpleadoB2, domain.AudienciaHechoEmpleadoB2
	}
	return err == nil && a.ValidarEstructura() == nil && a.PersonaVersion() == actor.Instantanea.PersonaVersion && a.PerfilVersion() == actor.Instantanea.PerfilVersion && x.Operacion() == accion && x.AudienciaConsumo() == audiencia && x.EfectoRef() == r.Referencia && x.EfectoHuellaSHA256() == h
}

func reciboActoRegistroB2ValidoPara(m domain.MaterialActoRegistroEmpleadoB2, r ports.ReciboActoRegistroEmpleadoB2, tipo, relacionRef string, versionResultado int64) bool {
	_, offset := r.RegistradoEn.Zone()
	if !reciboActoB2Valido.MatchString(r.ReciboRef) || !domain.ReferenciaEmpleadoValida(r.EmpleadoRef) || !domain.ReferenciaRelacionValida(r.RelacionRef) || r.EficaciaAdministrativa || r.FirmaOficial ||
		r.Version < 1 || r.RegistradoEn.IsZero() || offset != 0 || r.RegistradoEn.Nanosecond()%1000 != 0 ||
		r.DecisionRef == "" || r.EfectoRef != m.Referencia() || !huellaRegistroB2.MatchString(r.ConsumoHuellaSHA256) || r.AuditoriaRef == "" {
		return false
	}
	if m.Tipo() == "alta" {
		return r.Tipo == "alta" && r.Version == 1 && regexp.MustCompile(`^pep_[A-Za-z0-9_-]{22,128}$`).MatchString(r.ProyeccionRef) && r.HechoRef == ""
	}
	return r.ProyeccionRef == "" && r.HechoRef != "" && r.Tipo == tipo && r.Version == versionResultado && r.EmpleadoRef == m.Referencia() && (relacionRef == "" || r.RelacionRef == relacionRef)
}

func accesoActualRegistroB2Valido(m domain.MaterialActoRegistroEmpleadoB2, a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, e ports.AccesoActualRegistroEmpleadoB2) bool {
	x := a.ResumenCapacidad()
	_, offset := e.ConsultadaEn.Zone()
	return (e.EstadoReplay == "registrado" || e.EstadoReplay == "replay") &&
		e.DecisionRef == x.DecisionRef() && e.EfectoRef == x.EfectoRef() && e.EfectoRef == m.Referencia() &&
		huellaRegistroB2.MatchString(e.ConsumoHuellaSHA256) && e.AuditoriaRef != "" &&
		!e.ConsultadaEn.IsZero() && offset == 0 && e.ConsultadaEn.Nanosecond()%1000 == 0 &&
		!e.ConsultadaEn.Before(x.EmitidaEn()) && e.ConsultadaEn.Before(x.ExpiraEn())
}

func errorActoRegistroB2Opaco(ctx context.Context, err error) error {
	if e := ctx.Err(); e != nil {
		return e
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return errorRegistroB2Opaco(ctx, err)
}
