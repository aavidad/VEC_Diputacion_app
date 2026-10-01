package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"regexp"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type ServicioPlanIncorporacionCT struct {
	autorizador ports.ProveedorAutorizacionPlanIncorporacionCT
	repositorio ports.RepositorioPlanIncorporacionCT
	actos       ports.ActosPlanIncorporacionCT
	rpt         ports.FuenteReservaRPTPlanCT
}

var _ ports.ServicioPlanIncorporacionCT = (*ServicioPlanIncorporacionCT)(nil)

func NuevoServicioPlanIncorporacionCT(a ports.ProveedorAutorizacionPlanIncorporacionCT, r ports.RepositorioPlanIncorporacionCT, actos ports.ActosPlanIncorporacionCT, rpt ports.FuenteReservaRPTPlanCT) (*ServicioPlanIncorporacionCT, error) {
	if nulo(a) || nulo(r) || nulo(actos) || nulo(rpt) {
		return nil, domain.ErrRegistroEmpleadoB2NoDisponible
	}
	return &ServicioPlanIncorporacionCT{a, r, actos, rpt}, nil
}
func (s *ServicioPlanIncorporacionCT) PrepararPlan(ctx context.Context, sol ports.SolicitudPlanIncorporacionCT) (ports.EstadoPlanIncorporacionCT, error) {
	m, err := domain.NuevoMaterialPrepararPlanIncorporacionCT(sol)
	if err != nil {
		return ports.EstadoPlanIncorporacionCT{}, err
	}
	return s.operar(ctx, m)
}
func (s *ServicioPlanIncorporacionCT) ConsultarPlan(ctx context.Context, sol ports.ConsultaPlanIncorporacionCT) (ports.EstadoPlanIncorporacionCT, error) {
	return s.consultar(ctx, sol, "consultar")
}
func (s *ServicioPlanIncorporacionCT) consultar(ctx context.Context, sol ports.ConsultaPlanIncorporacionCT, op string) (ports.EstadoPlanIncorporacionCT, error) {
	m, err := domain.NuevoMaterialConsultarPlanIncorporacionCT(sol, op)
	if err != nil {
		return ports.EstadoPlanIncorporacionCT{}, err
	}
	return s.operar(ctx, m)
}
func (s *ServicioPlanIncorporacionCT) operar(ctx context.Context, m domain.MaterialPlanIncorporacionCT) (ports.EstadoPlanIncorporacionCT, error) {
	var vacio ports.EstadoPlanIncorporacionCT
	if s == nil || ctx == nil || nulo(s.autorizador) || nulo(s.repositorio) {
		return vacio, domain.ErrRegistroEmpleadoB2NoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	a, err := s.autorizador.AutorizarPlanIncorporacionCT(ctx, m)
	if err != nil {
		return vacio, errorRegistroB2Opaco(ctx, err)
	}
	if !autorizacionPlanCTValida(m, a) {
		return vacio, domain.ErrRegistroEmpleadoB2Denegado
	}
	o := ports.OrdenPlanIncorporacionCT{Material: m, Autorizacion: a}
	var estado ports.EstadoPlanIncorporacionCT
	switch m.Operacion() {
	case "preparar":
		estado, err = s.repositorio.PrepararPlan(ctx, o)
	case "confirmar":
		estado, err = s.repositorio.ConfirmarPlan(ctx, o)
	default:
		estado, err = s.repositorio.ConsultarPlan(ctx, o)
	}
	if err != nil {
		return vacio, errorRegistroB2Opaco(ctx, err)
	}
	if err = ctx.Err(); err != nil {
		return vacio, err
	}
	if !estadoPlanCTValido(estado, m, a) {
		return vacio, domain.ErrRegistroEmpleadoB2NoDisponible
	}
	return estado, nil
}

// Cada recuperación consulta primero los actos con permiso vigente. Un recibo
// histórico impide reconstruir el material con otro contexto de identidad.
func (s *ServicioPlanIncorporacionCT) EjecutarPlan(ctx context.Context, sol ports.ConsultaPlanIncorporacionCT) (ports.EstadoPlanIncorporacionCT, error) {
	var vacio ports.EstadoPlanIncorporacionCT
	if s == nil || nulo(s.actos) || nulo(s.rpt) {
		return vacio, domain.ErrRegistroEmpleadoB2NoDisponible
	}
	estado, err := s.consultar(ctx, sol, "ejecutar")
	if err != nil {
		return vacio, err
	}
	p := estado.Plan
	actorRPT, _ := sol.Actor.Clonar()
	evidencia, err := s.rpt.AcreditarReservaPlanCT(ctx, p, actorRPT)
	if err != nil {
		return vacio, errorRegistroB2Opaco(ctx, err)
	}
	if !reservaPlanCTValida(p, evidencia) {
		return vacio, domain.ErrRegistroEmpleadoB2Denegado
	}
	if err = ctx.Err(); err != nil {
		return vacio, err
	}
	if estado.Estado == "ejecutado" {
		return estado, nil
	}
	d := p.Datos
	if estado.ReciboAltaRelacion == nil {
		actorActo, _ := sol.Actor.Clonar()
		proc := d.Procedencia
		proc.IdempotenciaRef = p.ClaveAltaRelacion
		var recibo ports.ReciboActoRegistroEmpleadoB2
		if p.Modo == "alta_empleado" {
			r, e := s.actos.RegistrarEmpleado(ctx, domain.SolicitudAltaEmpleadoB2{PersonaRef: d.PersonaRef, OrganismoRef: d.OrganismoRef, UnidadRef: d.UnidadRef, Regimen: d.Regimen, Modalidad: d.Modalidad, VigenteDesde: d.Desde, VigenteHasta: d.Hasta, Procedencia: proc, Actor: actorActo})
			err = e
			recibo = r.Recibo
		} else {
			r, e := s.actos.RegistrarHecho(ctx, domain.SolicitudHechoEmpleadoB2{Tipo: "relacion", EmpleadoRef: p.EmpleadoExistenteRef, OrganismoRef: d.OrganismoRef, RevisionEsperada: 1, UnidadRef: d.UnidadRef, Regimen: d.Regimen, Modalidad: d.Modalidad, Estado: "vigente", VigenteDesde: d.Desde, VigenteHasta: d.Hasta, Procedencia: proc, Actor: actorActo})
			err = e
			recibo = r.Recibo
		}
		if err != nil {
			return vacio, errorRegistroB2Opaco(ctx, err)
		}
		estado, err = s.consultar(ctx, sol, "ejecutar")
		if err != nil {
			return vacio, err
		}
		if estado.Plan.HuellaSHA256 != p.HuellaSHA256 || estado.ReciboAltaRelacion == nil || estado.ReciboAltaRelacion.ReciboRef != recibo.ReciboRef {
			return vacio, domain.ErrRegistroEmpleadoB2NoDisponible
		}
	}
	if estado.ReciboOcupacion == nil {
		actorActo, _ := sol.Actor.Clonar()
		proc := d.Procedencia
		proc.IdempotenciaRef = p.ClaveOcupacion
		relacion := estado.ReciboAltaRelacion
		r, e := s.actos.RegistrarHecho(ctx, domain.SolicitudHechoEmpleadoB2{Tipo: "ocupacion", EmpleadoRef: relacion.EmpleadoRef, OrganismoRef: d.OrganismoRef, RelacionRef: relacion.RelacionRef, RevisionEsperada: 1, RelacionVersionEsperada: relacion.Version, UnidadRef: d.UnidadRef, Modalidad: d.Modalidad, ClaseOcupacion: d.ClaseOcupacion, Estado: "vigente", PlazaRef: d.PlazaRef, PuestoRef: d.PuestoRef, VersionPlazaRef: d.VersionPlantillaRef, VersionPuestoRef: d.VersionRPTRef, VigenteDesde: d.Desde, VigenteHasta: d.Hasta, Procedencia: proc, Actor: actorActo})
		if e != nil {
			return vacio, errorRegistroB2Opaco(ctx, e)
		}
		estado, err = s.consultar(ctx, sol, "ejecutar")
		if err != nil {
			return vacio, err
		}
		if estado.Plan.HuellaSHA256 != p.HuellaSHA256 || estado.ReciboOcupacion == nil || estado.ReciboOcupacion.ReciboRef != r.Recibo.ReciboRef {
			return vacio, domain.ErrRegistroEmpleadoB2NoDisponible
		}
	}
	// La función propietaria vuelve a cotejar ambos actos y fija el recibo terminal.
	estado, err = s.consultar(ctx, sol, "confirmar")
	if err != nil {
		return vacio, err
	}
	if estado.Estado != "ejecutado" || estado.Plan.HuellaSHA256 != p.HuellaSHA256 {
		return vacio, domain.ErrRegistroEmpleadoB2NoDisponible
	}
	return estado, nil
}
func autorizacionPlanCTValida(m domain.MaterialPlanIncorporacionCT, a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) bool {
	h, e := m.HuellaSHA256()
	x := a.ResumenCapacidad()
	actor := m.Actor()
	return e == nil && a.ValidarEstructura() == nil && a.PersonaVersion() == actor.Instantanea.PersonaVersion && a.PerfilVersion() == actor.Instantanea.PerfilVersion && x.Operacion() == m.Accion() && x.AudienciaConsumo() == domain.AudienciaPlanIncorporacionCT && x.EfectoRef() == m.Recurso().Referencia && x.EfectoHuellaSHA256() == h
}
func reservaPlanCTValida(p ports.PlanIncorporacionCT, e ports.EvidenciaReservaRPTPlanCT) bool {
	d := p.Datos
	return e.UsoRPTRef == p.UsoRPTRef && e.ReservaRPTRef == p.ReservaRPTRef && e.PlanHuellaSHA256 == p.HuellaSHA256 && e.PlazaRef == d.PlazaRef && e.PuestoRef == d.PuestoRef && e.CatalogoID == d.CatalogoRPTID && e.Modulo == d.CatalogoRPTModulo && e.Categoria == d.CatalogoRPTCategoria && e.Version == d.CatalogoRPTVersion && e.HuellaSHA256 == d.CatalogoRPTHuellaSHA256 && e.ReciboRef != ""
}

var refEstadoPlanCT = regexp.MustCompile(`^[a-z][a-z0-9_:-]{2,159}$`)

func estadoPlanCTValido(s ports.EstadoPlanIncorporacionCT, m domain.MaterialPlanIncorporacionCT, a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) bool {
	p := s.Plan
	d := p.Datos
	e := s.Evidencia
	x := a.ResumenCapacidad()
	_, offset := e.ConsultadaEn.Zone()
	if p.Validar() != nil || d.OrganismoRef != m.OrganismoRef() || (!refEstadoPlanCT.MatchString(e.ReciboRef)) || e.DecisionRef != x.DecisionRef() || e.EfectoRef != m.Recurso().Referencia || !huellaRegistroB2.MatchString(e.ConsumoHuellaSHA256) || e.AuditoriaRef == "" || e.ConsultadaEn.IsZero() || offset != 0 || e.ConsultadaEn.Nanosecond()%1000 != 0 || e.ConsultadaEn.Before(x.EmitidaEn()) || !e.ConsultadaEn.Before(x.ExpiraEn()) {
		return false
	}
	if m.Operacion() == "preparar" {
		if d.HuellaSHA256() != m.Datos().HuellaSHA256() {
			return false
		}
	} else if p.PlanRef != m.PlanRef() {
		return false
	}
	r := s.ReciboAltaRelacion
	o := s.ReciboOcupacion
	if r != nil {
		if !reciboPlanCTBasico(*r) || r.RegistradoEn.After(e.ConsultadaEn) || r.Version != 1 {
			return false
		}
		if p.Modo == "alta_empleado" {
			if r.Tipo != "alta" || r.EfectoRef != d.PersonaRef || r.ProyeccionRef == "" || r.HechoRef != "" {
				return false
			}
		} else if r.Tipo != "relacion" || r.EmpleadoRef != p.EmpleadoExistenteRef || r.EfectoRef != p.EmpleadoExistenteRef || r.ProyeccionRef != "" || r.HechoRef != r.RelacionRef {
			return false
		}
	}
	if o != nil && (r == nil || !reciboPlanCTBasico(*o) || o.RegistradoEn.After(e.ConsultadaEn) || o.Tipo != "ocupacion" || o.Version != 1 || o.EmpleadoRef != r.EmpleadoRef || o.RelacionRef != r.RelacionRef || o.EfectoRef != r.EmpleadoRef || o.HechoRef == "" || o.ProyeccionRef != "") {
		return false
	}
	switch s.Estado {
	case "preparado":
		if r != nil || o != nil {
			return false
		}
	case "relacion_registrada":
		if r == nil || o != nil {
			return false
		}
	case "ocupacion_registrada":
		if r == nil || o == nil {
			return false
		}
	case "ejecutado":
		if r == nil || o == nil || !refEstadoPlanCT.MatchString(s.EjecucionReciboRef) {
			return false
		}
		h := sha256.Sum256([]byte(p.HuellaSHA256 + "|" + r.ReciboRef + "|" + o.ReciboRef))
		return s.EjecucionHuellaSHA256 == hex.EncodeToString(h[:])
	default:
		return false
	}
	return s.EjecucionReciboRef == "" && s.EjecucionHuellaSHA256 == ""
}
func reciboPlanCTBasico(r ports.ReciboActoRegistroEmpleadoB2) bool {
	_, off := r.RegistradoEn.Zone()
	return reciboActoB2Valido.MatchString(r.ReciboRef) && domain.ReferenciaEmpleadoValida(r.EmpleadoRef) && domain.ReferenciaRelacionValida(r.RelacionRef) && !r.EficaciaAdministrativa && !r.FirmaOficial && !r.RegistradoEn.IsZero() && off == 0 && r.RegistradoEn.Nanosecond()%1000 == 0 && r.DecisionRef != "" && r.AuditoriaRef != "" && huellaRegistroB2.MatchString(r.ConsumoHuellaSHA256)
}

func (s *ServicioPlanIncorporacionCT) ResolverSeleccion(ctx context.Context, q ports.SeleccionPlanIncorporacionCT) (ports.ResultadoSeleccionPlanIncorporacionCT, error) {
	var vacio ports.ResultadoSeleccionPlanIncorporacionCT
	if s == nil || ctx == nil || nulo(s.autorizador) || nulo(s.repositorio) {
		return vacio, domain.ErrRegistroEmpleadoB2NoDisponible
	}
	if e := ctx.Err(); e != nil {
		return vacio, e
	}
	m, e := domain.NuevoMaterialSeleccionPlanIncorporacionCT(q)
	if e != nil {
		return vacio, e
	}
	a, e := s.autorizador.AutorizarPlanIncorporacionCT(ctx, m)
	if e != nil {
		return vacio, errorRegistroB2Opaco(ctx, e)
	}
	if !autorizacionPlanCTValida(m, a) {
		return vacio, domain.ErrRegistroEmpleadoB2Denegado
	}
	r, e := s.repositorio.ResolverSeleccion(ctx, ports.OrdenPlanIncorporacionCT{Material: m, Autorizacion: a})
	if e != nil {
		return vacio, errorRegistroB2Opaco(ctx, e)
	}
	if e = ctx.Err(); e != nil {
		return vacio, e
	}
	x := a.ResumenCapacidad()
	ev := r.Evidencia
	_, off := ev.ConsultadaEn.Zone()
	if r.Seleccion.ValidarPara(m) != nil || !refEstadoPlanCT.MatchString(ev.ReciboRef) || ev.DecisionRef != x.DecisionRef() || ev.EfectoRef != m.Recurso().Referencia || !huellaRegistroB2.MatchString(ev.ConsumoHuellaSHA256) || ev.AuditoriaRef == "" || ev.ConsultadaEn.IsZero() || off != 0 || ev.ConsultadaEn.Nanosecond()%1000 != 0 || ev.ConsultadaEn.Before(x.EmitidaEn()) || !ev.ConsultadaEn.Before(x.ExpiraEn()) {
		return vacio, domain.ErrRegistroEmpleadoB2NoDisponible
	}
	return r, nil
}

var _ ports.ServicioClasesOcupacionCT = (*ServicioPlanIncorporacionCT)(nil)

func (s *ServicioPlanIncorporacionCT) ConsultarClasesOcupacion(ctx context.Context, q ports.ConsultaClasesOcupacionCT) (ports.ResultadoClasesOcupacionCT, error) {
	var vacio ports.ResultadoClasesOcupacionCT
	if s == nil || ctx == nil || nulo(s.autorizador) || nulo(s.repositorio) {
		return vacio, domain.ErrRegistroEmpleadoB2NoDisponible
	}
	if e := ctx.Err(); e != nil {
		return vacio, e
	}
	m, e := domain.NuevoMaterialClasesOcupacionCT(q)
	if e != nil {
		return vacio, e
	}
	a, e := s.autorizador.AutorizarPlanIncorporacionCT(ctx, m)
	if e != nil {
		return vacio, errorRegistroB2Opaco(ctx, e)
	}
	if !autorizacionPlanCTValida(m, a) {
		return vacio, domain.ErrRegistroEmpleadoB2Denegado
	}
	r, e := s.repositorio.ConsultarClasesOcupacion(ctx, ports.OrdenPlanIncorporacionCT{Material: m, Autorizacion: a})
	if e != nil {
		return vacio, errorRegistroB2Opaco(ctx, e)
	}
	if e = ctx.Err(); e != nil {
		return vacio, e
	}
	x := a.ResumenCapacidad()
	ev := r.Evidencia
	_, off := ev.ConsultadaEn.Zone()
	if r.Catalogo.Validar() != nil || !refEstadoPlanCT.MatchString(ev.ReciboRef) || ev.DecisionRef != x.DecisionRef() || ev.EfectoRef != m.Recurso().Referencia || !huellaRegistroB2.MatchString(ev.ConsumoHuellaSHA256) || ev.AuditoriaRef == "" || ev.ConsultadaEn.IsZero() || off != 0 || ev.ConsultadaEn.Nanosecond()%1000 != 0 || ev.ConsultadaEn.Before(x.EmitidaEn()) || !ev.ConsultadaEn.Before(x.ExpiraEn()) {
		return vacio, domain.ErrRegistroEmpleadoB2NoDisponible
	}
	return r, nil
}
