package incorporacionejercicio

import (
	"context"
	"errors"
	"math"
	"reflect"

	dom "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	personal "vec-diputacion-granada/internal/modules/personal/domain"
	pp "vec-diputacion-granada/internal/modules/personal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

// Cada operación obtiene el perfil nominal vigente desde la frontera confiable.
// No se conserva el actor de la preparación para ejecutar o recuperar sus actos.
type ActoresConsumidorPersonalB2 interface {
	ActorPreparacionPlanB2(context.Context) (core.ContextoActor, error)
	ActorConsultaPlanB2(context.Context) (core.ContextoActor, error)
	ActorEjecucionPlanB2(context.Context) (core.ContextoActor, error)
	ActorLecturaHechosB2(context.Context) (core.ContextoActor, error)
}

// La autoridad produce una concesión nueva para el material y perfil RPT fijos.
// Calcula su huella mediante la misma representación PostgreSQL que AD3 consume.
type AutoridadUsoRPTPersonalB2 interface {
	AutorizarReservaPlanB2(context.Context, pp.PlanIncorporacionCT, vp.MaterialReservaUsoCategoriaRPT) (vp.OrdenReservaUsoCategoriaRPT, error)
	AutorizarConfirmacionPlanB2(context.Context, pp.PlanIncorporacionCT, vp.MaterialTerminalUsoCategoriaRPT) (vp.OrdenConfirmacionUsoCategoriaRPT, error)
}

type ConfiguracionConsumidorPersonalB2 struct {
	Personal     pp.ServicioPlanIncorporacionCT
	Ficha        pp.FuenteFichaIncorporacionCT
	Hechos       pp.ConsultaHechosIncorporacionCT
	RPT          vp.GestorUsosCategoriaRPT
	AutoridadRPT AutoridadUsoRPTPersonalB2
	Actores      ActoresConsumidorPersonalB2
	Reloj        ct.Reloj
}

type ConsumidorPersonalB2 struct {
	c ConfiguracionConsumidorPersonalB2
}

type ResultadoConsumidorPersonalB2 struct {
	Estado pp.EstadoPlanIncorporacionCT
	Hechos pp.HechosIncorporacionCT
	Uso    vp.ResultadoUsoCategoriaRPT
}

func NuevoConsumidorPersonalB2(c ConfiguracionConsumidorPersonalB2) (*ConsumidorPersonalB2, error) {
	for _, d := range []any{c.Personal, c.Ficha, c.Hechos, c.RPT, c.AutoridadRPT, c.Actores, c.Reloj} {
		if nuloPlanNominal(d) {
			return nil, ct.ErrComposicionIncorporacionAplicacion
		}
	}
	return &ConsumidorPersonalB2{c}, nil
}

// ReservarPersonalB2 prepara la intención propietaria de Personal. Las claves y
// referencias RPT proceden del plan guardado, incluso después de un reinicio.
func (c *ConsumidorPersonalB2) ReservarPersonalB2(ctx context.Context, s SolicitudReservaPersonalB2) (ReservaPersonalB2, error) {
	var cero ReservaPersonalB2
	if c == nil || ctx == nil || !contratoBasicoValido(s.Contrato, s.Contrato.OrganizacionRef, s.Contrato.ExpedienteRef) ||
		!contratoB2Valido(s.Contrato) || s.Contrato.Protocolo != ProtocoloPersonalB2V1 {
		return cero, ct.ErrIntencionIncorporacionAplicacion
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	actor, err := c.c.Actores.ActorPreparacionPlanB2(ctx)
	if err != nil {
		return cero, errorConsumidorPersonalB2(ctx, err)
	}
	d, err := datosPlanPropietarioB2(s)
	if err != nil {
		return cero, err
	}
	consulta := pp.SolicitudPlanIncorporacionCT{DatosPlanIncorporacionCT: d, Actor: actor}
	estado, err := c.c.Personal.PrepararPlan(ctx, consulta)
	if err != nil {
		return cero, errorConsumidorPersonalB2(ctx, err)
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	plan := estado.Plan
	if plan.Validar() != nil || plan.Version < 1 || !reflect.DeepEqual(plan.Datos, d) {
		return cero, ct.ErrConflictoIncorporacionAplicacion
	}
	contrato := s.Contrato
	return ReservaPersonalB2{
		OrganizacionRef: contrato.OrganizacionRef, ExpedienteRef: contrato.ExpedienteRef,
		SolicitudRef: contrato.SolicitudRef, IdempotenciaRef: contrato.ReservaIdempotente,
		PersonaRef: s.Persona.PersonaRef, PersonaVersion: s.Persona.PersonaVersion,
		ContratoReciboRef: contrato.ContratoReciboRef, ContratoSHA256: contrato.ContratoSHA256,
		FuenteRPT: contrato.FuenteRPT, PuestoRef: contrato.PuestoRef, PlazaRef: contrato.PlazaRef,
		ReservaRef: plan.PlanRef, ReciboRef: plan.ReciboRef, VersionReserva: uint64(plan.Version),
		EmpleadoRef: plan.EmpleadoExistenteRef, EjercicioSintetico: contrato.EjercicioSintetico, Plan: plan,
	}, nil
}

func datosPlanPropietarioB2(s SolicitudReservaPersonalB2) (pp.DatosPlanIncorporacionCT, error) {
	x, p, d := s.Contrato, s.Persona, s.Contrato.DatosPersonal
	if p.OrganizacionRef != x.OrganizacionRef || p.ExpedienteRef != x.ExpedienteRef ||
		p.AceptacionRef != x.AceptacionRef || p.LlamamientoRef != x.LlamamientoRef ||
		p.SeleccionRef != x.SeleccionRef || p.VersionSeleccion != x.VersionSeleccion || p.ReciboRef != x.SeleccionReciboRef ||
		s.Puesto.OrganizacionRef != x.OrganizacionRef || s.Puesto.ExpedienteRef != x.ExpedienteRef ||
		s.Puesto.Fuente != x.FuenteRPT || s.Puesto.PuestoRef != x.PuestoRef || s.Puesto.PlazaRef != x.PlazaRef ||
		s.Puesto.CategoriaRef != x.CategoriaRef || s.Puesto.VinculoRevision != x.VinculoRevision ||
		s.Puesto.VinculoReciboRef != x.VinculoReciboRef ||
		!s.Puesto.Prospectivo || s.Puesto.AcreditaProcedenciaHistorica ||
		x.VersionExpediente > math.MaxInt64 || p.PersonaVersion > math.MaxInt64 || d.CatalogoRPTVersion > math.MaxInt64 {
		return pp.DatosPlanIncorporacionCT{}, ct.ErrConflictoIncorporacionAplicacion
	}
	r := pp.DatosPlanIncorporacionCT{
		IdempotenciaRef: x.ReservaIdempotente, OrigenCTRef: x.ContratoRef,
		OrigenCTReciboRef: x.ContratoReciboRef, OrigenCTHuellaSHA256: x.ContratoSHA256,
		ExpedienteRef: x.ExpedienteRef, ExpedienteVersion: int64(x.VersionExpediente),
		OrganismoRef: d.OrganismoRef, UnidadRef: d.UnidadRef, PersonaRef: p.PersonaRef, PersonaVersion: int64(p.PersonaVersion),
		FuenteBolsaRef: p.FuenteRef, FuenteBolsaVersion: p.FuenteVersion, FuenteBolsaReciboRef: p.ReciboRef, FuenteBolsaHuellaSHA256: p.FuenteSHA256,
		Regimen: d.Regimen, Modalidad: d.Modalidad, Desde: d.Desde, Hasta: d.Hasta, ClaseOcupacion: d.ClaseOcupacion,
		PlazaRef: x.PlazaRef, PuestoRef: x.PuestoRef, VersionPlantillaRef: d.VersionPlantillaRef, VersionRPTRef: d.VersionRPTRef,
		RevisionPlaza: d.RevisionPlaza, RevisionPuesto: d.RevisionPuesto,
		FuenteOrganizacionRef: d.FuenteOrganizacionRef, FuenteOrganizacionHuellaSHA256: d.FuenteOrganizacionSHA256,
		CatalogoRPTID: d.CatalogoRPTID, CatalogoRPTModulo: d.ModuloRPTID, CatalogoRPTCategoria: d.CategoriaID,
		CatalogoRPTVersion: int64(d.CatalogoRPTVersion), CatalogoRPTHuellaSHA256: d.CatalogoRPTHuellaSHA256,
		VinculoCTReciboRef: x.VinculoReciboRef, Procedencia: d.Procedencia,
	}
	if r.Validar() != nil {
		return pp.DatosPlanIncorporacionCT{}, ct.ErrIntencionIncorporacionAplicacion
	}
	return r, nil
}

// ConsultarPersonalB2 sólo consulta el plan y sus recibos con permiso actual.
// No prepara planes, reserva RPT ni ejecuta actos aunque la preparación falte.
func (c *ConsumidorPersonalB2) ConsultarPersonalB2(ctx context.Context, planRef, organismo string) (pp.EstadoPlanIncorporacionCT, error) {
	var cero pp.EstadoPlanIncorporacionCT
	if c == nil || ctx == nil {
		return cero, ct.ErrComposicionIncorporacionAplicacion
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	actor, err := c.c.Actores.ActorConsultaPlanB2(ctx)
	if err != nil {
		return cero, errorConsumidorPersonalB2(ctx, err)
	}
	r, err := c.c.Personal.ConsultarPlan(ctx, pp.ConsultaPlanIncorporacionCT{PlanRef: planRef, OrganismoRef: organismo, Actor: actor})
	if err != nil {
		return cero, errorConsumidorPersonalB2(ctx, err)
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if r.Plan.Validar() != nil || r.Plan.PlanRef != planRef || r.Plan.Datos.OrganismoRef != organismo {
		return cero, ct.ErrConflictoIncorporacionAplicacion
	}
	return r, nil
}

// ConfirmarPersonalB2 retoma el plan propietario. Personal ejecuta sólo los
// actos ausentes y comprueba la reserva RPT con su dependencia confiable.
func (c *ConsumidorPersonalB2) ConfirmarPersonalB2(ctx context.Context, planRef, organismo string) (ResultadoConsumidorPersonalB2, error) {
	var cero ResultadoConsumidorPersonalB2
	estado, err := c.ConsultarPersonalB2(ctx, planRef, organismo)
	if err != nil {
		return cero, err
	}
	p := estado.Plan
	m := materialReservaPlanB2(p)
	orden, err := c.c.AutoridadRPT.AutorizarReservaPlanB2(ctx, p, m)
	if err != nil {
		return cero, errorConsumidorPersonalB2(ctx, err)
	}
	if orden.Material != m {
		return cero, ct.ErrDenegadaIncorporacionAplicacion
	}
	uso, err := c.c.RPT.ReservarUsoCategoriaRPT(ctx, orden)
	if err != nil {
		return cero, errorConsumidorPersonalB2(ctx, err)
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if !usoPlanB2Valido(uso, p, false) || (uso.Uso.Estado == "confirmado" && !ejecucionPlanB2Completa(estado)) {
		return cero, ct.ErrConflictoIncorporacionAplicacion
	}
	if !ejecucionPlanB2Completa(estado) {
		actor, err := c.c.Actores.ActorEjecucionPlanB2(ctx)
		if err != nil {
			return cero, errorConsumidorPersonalB2(ctx, err)
		}
		estado, err = c.c.Personal.EjecutarPlan(ctx, pp.ConsultaPlanIncorporacionCT{PlanRef: planRef, OrganismoRef: organismo, Actor: actor})
		if err != nil {
			return cero, errorConsumidorPersonalB2(ctx, err)
		}
		if estado.Plan != p || !ejecucionPlanB2Completa(estado) {
			return cero, ct.ErrConflictoIncorporacionAplicacion
		}
	}
	hechos, err := c.leerHechos(ctx, estado)
	if err != nil {
		return cero, err
	}
	terminal := vp.MaterialTerminalUsoCategoriaRPT{Reserva: m, TerminalReciboRef: p.ConfirmacionRPTRef,
		EvidenciaRef: estado.EjecucionReciboRef, EvidenciaSHA256: estado.EjecucionHuellaSHA256}
	confirmacion, err := c.c.AutoridadRPT.AutorizarConfirmacionPlanB2(ctx, p, terminal)
	if err != nil {
		return cero, errorConsumidorPersonalB2(ctx, err)
	}
	if confirmacion.Material != terminal {
		return cero, ct.ErrDenegadaIncorporacionAplicacion
	}
	uso, err = c.c.RPT.ConfirmarUsoCategoriaRPT(ctx, confirmacion)
	if err != nil {
		return cero, errorConsumidorPersonalB2(ctx, err)
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if !usoPlanB2Valido(uso, p, true) {
		return cero, ct.ErrConflictoIncorporacionAplicacion
	}
	return ResultadoConsumidorPersonalB2{Estado: estado, Hechos: hechos, Uso: uso}, nil
}

func materialReservaPlanB2(p pp.PlanIncorporacionCT) vp.MaterialReservaUsoCategoriaRPT {
	d := p.Datos
	return vp.MaterialReservaUsoCategoriaRPT{Consumidor: "personal", UsoRef: p.UsoRPTRef, CategoriaID: d.CatalogoRPTCategoria,
		Publicacion: vp.ReferenciaPublicacionRPT{CatalogoID: d.CatalogoRPTID, Version: int(d.CatalogoRPTVersion), HuellaSHA256: d.CatalogoRPTHuellaSHA256}, ReservaReciboRef: p.ReservaRPTRef}
}

func usoPlanB2Valido(r vp.ResultadoUsoCategoriaRPT, p pp.PlanIncorporacionCT, confirmado bool) bool {
	u, m := r.Uso, materialReservaPlanB2(p)
	if !r.Encontrado || u == nil || u.Consumidor != m.Consumidor || u.UsoRef != m.UsoRef ||
		u.CategoriaID != m.CategoriaID || u.Publicacion != m.Publicacion || u.ReservaReciboRef != m.ReservaReciboRef || u.Revision < 1 {
		return false
	}
	if u.Estado == "reservado" {
		return !confirmado && u.TerminalReciboRef == nil && u.TerminalEn == nil
	}
	return u.Estado == "confirmado" && u.TerminalReciboRef != nil && *u.TerminalReciboRef == p.ConfirmacionRPTRef && u.TerminalEn != nil
}

func ejecucionPlanB2Completa(e pp.EstadoPlanIncorporacionCT) bool {
	a, o := e.ReciboAltaRelacion, e.ReciboOcupacion
	if e.Estado != "ejecutado" || a == nil || o == nil || !huellaPlanNominalValida(e.EjecucionHuellaSHA256) ||
		!referenciaPlanB2(e.EjecucionReciboRef) || !personal.ReferenciaEmpleadoValida(a.EmpleadoRef) ||
		!personal.ReferenciaRelacionValida(a.RelacionRef) || a.RelacionRef != o.RelacionRef || a.EmpleadoRef != o.EmpleadoRef ||
		a.Version < 1 || o.Version < 1 || o.Tipo != "ocupacion" || !referenciaPlanB2(o.HechoRef) ||
		!referenciaPlanB2(a.ReciboRef) || !referenciaPlanB2(o.ReciboRef) || a.ReciboRef == o.ReciboRef ||
		!dom.InstanteUTCCanonico(a.RegistradoEn) || !dom.InstanteUTCCanonico(o.RegistradoEn) || o.EfectoRef != o.EmpleadoRef ||
		a.FirmaOficial || a.EficaciaAdministrativa || o.FirmaOficial || o.EficaciaAdministrativa {
		return false
	}
	if e.Plan.Modo == "alta_empleado" {
		return a.Tipo == "alta" && a.EfectoRef == e.Plan.Datos.PersonaRef
	}
	return e.Plan.Modo == "nueva_relacion" && a.Tipo == "relacion" && a.EmpleadoRef == e.Plan.EmpleadoExistenteRef && a.EfectoRef == a.EmpleadoRef
}

func referenciaPlanB2(s string) bool { return s != "" && len(s) <= 160 }

func (c *ConsumidorPersonalB2) leerHechos(ctx context.Context, e pp.EstadoPlanIncorporacionCT) (pp.HechosIncorporacionCT, error) {
	var cero pp.HechosIncorporacionCT
	actor, err := c.c.Actores.ActorLecturaHechosB2(ctx)
	if err != nil {
		return cero, errorConsumidorPersonalB2(ctx, err)
	}
	d, a, o := e.Plan.Datos, e.ReciboAltaRelacion, e.ReciboOcupacion
	corte := personal.CorteEmpleadoB2{VigenteEn: d.Desde, ConocidoEn: c.c.Reloj.Ahora()}
	ficha, err := c.c.Ficha.ConsultarFicha(ctx, personal.SolicitudFichaEmpleadoB2{EmpleadoRef: a.EmpleadoRef, OrganismoRef: d.OrganismoRef, Corte: corte, Actor: actor})
	if err != nil {
		return cero, errorConsumidorPersonalB2(ctx, err)
	}
	if ficha.Ficha.PersonaRef != d.PersonaRef || ficha.Ficha.EmpleadoRef != a.EmpleadoRef || ficha.Ficha.OrganismoRef != d.OrganismoRef || ficha.Ficha.Version < 1 {
		return cero, ct.ErrConflictoIncorporacionAplicacion
	}
	if !catalogosHechosPlanB2(ficha.Ficha, e) {
		return cero, ct.ErrConflictoIncorporacionAplicacion
	}
	s := pp.SolicitudHechosIncorporacionCT{Actor: actor, Seleccion: pp.SeleccionHechosIncorporacionCT{
		OrganismoRef: d.OrganismoRef, PersonaRef: d.PersonaRef, EmpleadoRef: a.EmpleadoRef,
		RelacionRef: a.RelacionRef, OcupacionRef: o.HechoRef, UnidadRef: d.UnidadRef, PuestoRef: d.PuestoRef, PlazaRef: d.PlazaRef,
		VersionEmpleado: ficha.Ficha.Version, VersionRelacion: a.Version, VersionOcupacion: o.Version, Corte: corte}}
	r, err := c.c.Hechos.ConsultarHechosIncorporacionCT(ctx, s)
	if err != nil {
		return cero, errorConsumidorPersonalB2(ctx, err)
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if r.Seleccion != s.Seleccion || r.RegimenRef != d.Regimen.Ref || r.ModalidadRef != d.Modalidad.Ref ||
		r.ClaseOcupacion != d.ClaseOcupacion ||
		r.Relacion.Desde != d.Desde || r.Relacion.Hasta != d.Hasta || r.Ocupacion.Desde != d.Desde || r.Ocupacion.Hasta != d.Hasta ||
		r.Relacion.ActoRef != d.Procedencia.ActoRef || r.Ocupacion.ActoRef != d.Procedencia.ActoRef ||
		r.Relacion.FuenteRef != d.Procedencia.FuenteRef || r.Ocupacion.FuenteRef != d.Procedencia.FuenteRef ||
		r.Relacion.FuenteVersion != d.Procedencia.FuenteVersion || r.Ocupacion.FuenteVersion != d.Procedencia.FuenteVersion ||
		r.FirmaOficial || r.EficaciaAdministrativa {
		return cero, ct.ErrConflictoIncorporacionAplicacion
	}
	return r, nil
}

func catalogosHechosPlanB2(f personal.FichaEmpleadoB2, e pp.EstadoPlanIncorporacionCT) bool {
	d, a, o := e.Plan.Datos, e.ReciboAltaRelacion, e.ReciboOcupacion
	var relaciones, ocupaciones int
	for _, r := range f.Relaciones {
		if r.RelacionRef == a.RelacionRef && r.Traza.Version == a.Version {
			reg, mod := r.CatalogoSnapshot.Regimen, r.CatalogoSnapshot.Modalidad
			if reg == nil || mod == nil || reg.Ref != d.Regimen.Ref || reg.Version != d.Regimen.Version ||
				mod.Ref != d.Modalidad.Ref || mod.Version != d.Modalidad.Version {
				return false
			}
			relaciones++
		}
	}
	for _, r := range f.Ocupaciones {
		if r.OcupacionRef == o.HechoRef && r.Traza.Version == o.Version {
			mod := r.CatalogoSnapshot.Modalidad
			if mod == nil || mod.Ref != d.Modalidad.Ref || mod.Version != d.Modalidad.Version {
				return false
			}
			ocupaciones++
		}
	}
	return relaciones == 1 && ocupaciones == 1
}

func errorConsumidorPersonalB2(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	switch {
	case errors.Is(err, personal.ErrRegistroEmpleadoB2Denegado), errors.Is(err, vp.ErrUsoCategoriaRPTDenegado), errors.Is(err, ct.ErrAutorizacionDenegada), errors.Is(err, ct.ErrPlanNominalB2Denegado), errors.Is(err, vp.ErrDenegacionExplicitaAutorizacionLigadaV3):
		return ct.ErrDenegadaIncorporacionAplicacion
	case errors.Is(err, personal.ErrRegistroEmpleadoB2NoEncontrado), errors.Is(err, ct.ErrPlanNominalB2NoEncontrado):
		return ct.ErrPreparacionIncorporacionPendiente
	case errors.Is(err, personal.ErrRegistroEmpleadoB2Conflicto), errors.Is(err, vp.ErrUsoCategoriaRPTConflicto), errors.Is(err, ct.ErrPlanNominalB2Conflicto):
		return ct.ErrConflictoIncorporacionAplicacion
	case errors.Is(err, personal.ErrRegistroEmpleadoB2Invalido), errors.Is(err, vp.ErrUsoCategoriaRPTInvalido), errors.Is(err, ct.ErrPlanNominalB2Invalido):
		return ct.ErrIntencionIncorporacionAplicacion
	}
	return errorFuentePlanNominal(ctx, err)
}

var _ ReservadorPersonalB2 = (*ConsumidorPersonalB2)(nil)
