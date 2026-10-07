package application

import (
	"context"
	"reflect"
	"time"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type ServicioPlanNominalB2 struct {
	fuentes     ports.FuentePreparacionPlanNominalB2
	autoridad   ports.AutoridadPlanNominalB2
	repositorio ports.RepositorioPlanNominalB2
}

func NuevoServicioPlanNominalB2(f ports.FuentePreparacionPlanNominalB2, a ports.AutoridadPlanNominalB2, r ports.RepositorioPlanNominalB2) (*ServicioPlanNominalB2, error) {
	if dependenciaNula(f) || dependenciaNula(a) || dependenciaNula(r) {
		return nil, ports.ErrPlanNominalB2NoDisponible
	}
	return &ServicioPlanNominalB2{f, a, r}, nil
}
func validarSolicitudPlanB2(s ports.SolicitudPlanNominalB2) bool {
	for _, r := range []string{s.OrganizacionRef, s.ExpedienteRef, s.PuestoRef, s.PlazaRef, s.VersionPlantillaRef, s.VersionRPTRef, s.Regimen.Ref, s.Modalidad.Ref, s.DocumentoRef} {
		if !domain.ReferenciaOpacaValida(r) {
			return false
		}
	}
	d, e := time.Parse(time.DateOnly, s.Desde)
	h, eh := time.Parse(time.DateOnly, s.Hasta)
	return domain.ClaseOcupacionPlanPersonalB2Valida(s.ClaseOcupacion) && e == nil && d.Year() > 0 && d.Format(time.DateOnly) == s.Desde && (s.Hasta == "" || eh == nil && h.Year() > 0 && h.Format(time.DateOnly) == s.Hasta && h.After(d)) && domain.VersionPlanPersonalB2Valida(s.VersionExpediente) && domain.VersionPlanPersonalB2Valida(s.Regimen.Version) && domain.VersionPlanPersonalB2Valida(s.Modalidad.Version) && domain.ClaveCatalogo(s.MotivoClave).Valida() && domain.HuellaPlanPersonalB2Valida(s.DocumentoSHA256) && domain.UUIDPlanPersonalB2Valido(s.ClaveIdempotencia)
}
func planCoincideSeleccionB2(p domain.PlanIncorporacionPersonalB2, s ports.SolicitudPlanNominalB2) bool {
	return p.Validar() == nil && p.OrganizacionRef == s.OrganizacionRef && p.ExpedienteRef == s.ExpedienteRef && p.VersionExpediente == s.VersionExpediente && p.PuestoRef == s.PuestoRef && p.PlazaRef == s.PlazaRef && p.VersionPlazaRef == s.VersionPlantillaRef && p.VersionPuestoRef == s.VersionRPTRef && p.ClaseOcupacion == s.ClaseOcupacion && p.Regimen == s.Regimen && p.Modalidad == s.Modalidad && p.Desde == s.Desde && p.Hasta == s.Hasta && p.MotivoClave == s.MotivoClave && p.DocumentoRef == s.DocumentoRef && p.DocumentoSHA256 == s.DocumentoSHA256
}
func contratoPlanB2Valido(c ports.ContratoPlanNominalB2) bool {
	if c.Protocolo != ports.ProtocoloIncorporacionPersonalB2 || c.PlanVersion != 1 || c.IntencionVersion != 1 || c.Material.Validar() != nil || !domain.UUIDPlanPersonalB2Valido(c.IdempotenciaPersonalUUID) || !domain.InstanteUTCCanonico(c.RegistradoEn) {
		return false
	}
	for _, r := range []string{c.PlanRef, c.PlanReciboRef, c.IntencionRef, c.IntencionReciboRef, c.SolicitudPersonalRef} {
		if !domain.ReferenciaOpacaValida(r) {
			return false
		}
	}
	// La huella se fija sobre la intención y el material, antes de los refs que
	// PostgreSQL asigna al asiento. El propietario devuelve la misma intención.
	h, e := domain.SHA256PlanPersonalB2(ports.RegistroPlanNominalB2{Solicitud: c.Solicitud, Material: c.Material})
	return e == nil && h == c.PlanSHA256 && planCoincideSeleccionB2(c.Material, c.Solicitud)
}
func (s *ServicioPlanNominalB2) RegistrarPlanNominalB2(ctx context.Context, sol ports.SolicitudPlanNominalB2, actor ports.ActorIncorporacionPersonalB2) (ports.ContratoPlanNominalB2, error) {
	var cero ports.ContratoPlanNominalB2
	if s == nil || ctx == nil || !validarSolicitudPlanB2(sol) || actor.Validar() != nil {
		return cero, ports.ErrPlanNominalB2Invalido
	}
	if e := ctx.Err(); e != nil {
		return cero, e
	}
	p, e := s.fuentes.ResolverPlanNominalB2(ctx, sol, actor)
	if e != nil {
		return cero, e
	}
	if e := ctx.Err(); e != nil {
		return cero, e
	}
	if !planCoincideSeleccionB2(p, sol) {
		return cero, ports.ErrPlanNominalB2Conflicto
	}
	m := ports.RegistroPlanNominalB2{Solicitud: sol, Material: p}
	b, e := domain.CanonicoPlanPersonalB2(m)
	if e != nil {
		return cero, ports.ErrPlanNominalB2Invalido
	}
	a, e := s.autoridad.AutorizarPlanNominalB2(ctx, ports.AccionRegistrarPlanNominalB2, b, actor)
	if e != nil {
		return cero, e
	}
	if e := ctx.Err(); e != nil {
		return cero, e
	}
	c, e := s.repositorio.RegistrarPlanNominalB2(ctx, m, a)
	if e != nil {
		return cero, e
	}
	sha, _ := domain.SHA256PlanPersonalB2(m)
	if !contratoPlanB2Valido(c) || c.Material != p || c.PlanSHA256 != sha {
		return cero, ports.ErrPlanNominalB2NoDisponible
	}
	return c, nil
}

// La autoridad de servidor vuelve a resolver el contexto actor actual también
// al consultar y al recuperar. La lectura no modifica ni ejecuta Personal.
func (s *ServicioPlanNominalB2) LeerContratoPlanNominal(ctx context.Context, org, exp string) (ports.ContratoPlanNominalB2, error) {
	var cero ports.ContratoPlanNominalB2
	if s == nil || ctx == nil || !domain.ReferenciaOpacaValida(org) || !domain.ReferenciaOpacaValida(exp) {
		return cero, ports.ErrPlanNominalB2Invalido
	}
	if e := ctx.Err(); e != nil {
		return cero, e
	}
	fu, ok := s.fuentes.(ports.FuenteUnidadPlanNominalB2)
	if !ok {
		return cero, ports.ErrPlanNominalB2NoDisponible
	}
	unidad, e := fu.ResolverUnidadPlanNominalB2(ctx, org, exp)
	if e != nil {
		return cero, e
	}
	if e := ctx.Err(); e != nil {
		return cero, e
	}
	if !domain.ReferenciaOpacaValida(unidad) {
		return cero, ports.ErrPlanNominalB2NoDisponible
	}
	m := map[string]string{"organizacion_ref": org, "expediente_ref": exp, "unidad_ref": unidad}
	b, _ := domain.CanonicoPlanPersonalB2(m)
	a, e := s.autoridad.AutorizarPlanNominalB2(ctx, ports.AccionLeerPlanNominalB2, b, ports.ActorIncorporacionPersonalB2{})
	if e != nil {
		return cero, e
	}
	if e := ctx.Err(); e != nil {
		return cero, e
	}
	c, e := s.repositorio.LeerContratoPlanNominal(ctx, org, exp, a, unidad)
	if e != nil {
		return cero, e
	}
	if !contratoPlanB2Valido(c) || c.Material.OrganizacionRef != org || c.Material.ExpedienteRef != exp {
		return cero, ports.ErrPlanNominalB2NoDisponible
	}
	return c, nil
}
func (s *ServicioPlanNominalB2) ConfirmarOrigenIncorporacionB2(ctx context.Context, m ports.ConfirmacionOrigenIncorporacionB2, actor ports.ActorIncorporacionPersonalB2) (ports.OrigenIncorporacionPersonalB2, error) {
	var cero ports.OrigenIncorporacionPersonalB2
	if s == nil || ctx == nil || actor.Validar() != nil || !domain.ReferenciaOpacaValida(m.PlanRef) || m.PlanVersion != 1 || !domain.HuellaPlanPersonalB2Valida(m.PlanSHA256) {
		return cero, ports.ErrPlanNominalB2Invalido
	}
	if e := ctx.Err(); e != nil {
		return cero, e
	}
	c, e := s.LeerContratoPlanNominal(ctx, m.OrganizacionRef, m.ExpedienteRef)
	if e != nil {
		return cero, e
	}
	if e := ctx.Err(); e != nil {
		return cero, e
	}
	if m.UnidadCTRef != "" && m.UnidadCTRef != c.Material.UnidadCTRef {
		return cero, ports.ErrPlanNominalB2Conflicto
	}
	m.UnidadCTRef = c.Material.UnidadCTRef
	if c.PlanRef != m.PlanRef || c.PlanVersion != m.PlanVersion || c.PlanSHA256 != m.PlanSHA256 {
		return cero, ports.ErrPlanNominalB2Conflicto
	}
	h, e := s.fuentes.VerificarHechosPersonalB2(ctx, c, m.Hechos, actor)
	if e != nil {
		return cero, e
	}
	if e := ctx.Err(); e != nil {
		return cero, e
	}
	if !reflect.DeepEqual(h, m.Hechos) {
		return cero, ports.ErrPlanNominalB2Conflicto
	}
	b, e := domain.CanonicoPlanPersonalB2(m)
	if e != nil {
		return cero, ports.ErrPlanNominalB2Invalido
	}
	a, e := s.autoridad.AutorizarPlanNominalB2(ctx, ports.AccionConfirmarOrigenB2, b, actor)
	if e != nil {
		return cero, e
	}
	if e := ctx.Err(); e != nil {
		return cero, e
	}
	o, e := s.repositorio.ConfirmarOrigenIncorporacionB2(ctx, m, a)
	if e != nil {
		return cero, e
	}
	if o.Protocolo != ports.ProtocoloIncorporacionPersonalB2 || o.Confirmacion != m || o.FirmaOficial || o.EficaciaAdministrativa || !domain.InstanteUTCCanonico(o.RegistradoEn) || !domain.ReferenciaOpacaValida(o.ReciboRef) || !domain.ReferenciaOpacaValida(o.AuditoriaRef) || !domain.ReferenciaOpacaValida(o.OutboxRef) {
		return cero, ports.ErrPlanNominalB2NoDisponible
	}
	return o, nil
}

func (s *ServicioPlanNominalB2) LeerOrigenIncorporacionB2(ctx context.Context, org, exp string) (ports.OrigenIncorporacionPersonalB2, bool, error) {
	var cero ports.OrigenIncorporacionPersonalB2
	if s == nil || ctx == nil || !domain.ReferenciaOpacaValida(org) || !domain.ReferenciaOpacaValida(exp) {
		return cero, false, ports.ErrPlanNominalB2Invalido
	}
	if e := ctx.Err(); e != nil {
		return cero, false, e
	}
	r, ok := s.repositorio.(ports.LectorOrigenIncorporacionB2)
	if !ok {
		return cero, false, ports.ErrPlanNominalB2NoDisponible
	}
	fu, ok := s.fuentes.(ports.FuenteUnidadPlanNominalB2)
	if !ok {
		return cero, false, ports.ErrPlanNominalB2NoDisponible
	}
	unidad, e := fu.ResolverUnidadPlanNominalB2(ctx, org, exp)
	if e != nil {
		return cero, false, e
	}
	if e := ctx.Err(); e != nil {
		return cero, false, e
	}
	if !domain.ReferenciaOpacaValida(unidad) {
		return cero, false, ports.ErrPlanNominalB2NoDisponible
	}
	b, _ := domain.CanonicoPlanPersonalB2(map[string]string{"organizacion_ref": org, "expediente_ref": exp, "unidad_ref": unidad})
	a, e := s.autoridad.AutorizarPlanNominalB2(ctx, ports.AccionLeerPlanNominalB2, b, ports.ActorIncorporacionPersonalB2{})
	if e != nil {
		return cero, false, e
	}
	if e := ctx.Err(); e != nil {
		return cero, false, e
	}
	o, encontrado, e := r.LeerOrigenIncorporacionB2(ctx, org, exp, a, unidad)
	if e != nil || !encontrado {
		return cero, encontrado, e
	}
	if o.Protocolo != ports.ProtocoloIncorporacionPersonalB2 || o.Confirmacion.OrganizacionRef != org || o.Confirmacion.ExpedienteRef != exp || o.FirmaOficial || o.EficaciaAdministrativa || !domain.InstanteUTCCanonico(o.RegistradoEn) {
		return cero, false, ports.ErrPlanNominalB2NoDisponible
	}
	return o, true, nil
}
