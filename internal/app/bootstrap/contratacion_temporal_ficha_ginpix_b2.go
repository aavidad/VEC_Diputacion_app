package bootstrap

import (
	"context"
	"errors"
	"reflect"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/ginpixfichero"
	httpct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	dom "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	personal "vec-diputacion-granada/internal/modules/personal/domain"
	pp "vec-diputacion-granada/internal/modules/personal/ports"
	"vec-diputacion-granada/internal/vec/adapters/httpapi"
)

// La ficha sólo usa el origen confirmado de CT y los hechos revalidados por Personal.
// El formato es una ficha local de trazabilidad, no una confirmación de GINPIX.
type fichaGINPIXB2 struct {
	f             *fachadaIncorporacionPersonalB2
	m             mapeoFichaGINPIXDesarrollo
	consultarPlan func(context.Context, string, string) (pp.EstadoPlanIncorporacionCT, error)
}

func nuevoMapeoFichaGINPIXB2() (mapeoFichaGINPIXDesarrollo, error) {
	claves := []dom.ClaveCatalogo{"actuacion_ref", "expediente_ref", "recibo_ref", "relacion_ref", "seguimiento_ref", "solicitud_personal_ref"}
	reglas := make([]dom.ReglaMapeoGINPIX, 0, len(claves))
	for _, clave := range claves {
		reglas = append(reglas, dom.ReglaMapeoGINPIX{CampoCanonico: clave, CampoDestino: clave, Obligatorio: true})
	}
	m, err := dom.PublicarMapeoVersionadoGINPIX(dom.BorradorMapeoVersionadoGINPIX{
		Esquema: dom.EsquemaMapeoGINPIXV1, Referencia: "mapeo:ct:ginpix:ficha-personal-b2", Version: 1,
		ProcedenciaRef: "configuracion:ct:ginpix:ficha-trazabilidad-personal-b2:v1", Reglas: reglas,
	})
	return mapeoFichaGINPIXDesarrollo{m: m}, err
}

func nuevaRutaFichaGINPIXB2(m *montajeIncorporacionPersonalB2, soporte *soporteAltaContratacionTemporalDesarrollo, fronteras catalogoFronterasComunDesarrollo) (httpapi.RutaExacta, error) {
	if m == nil || m.fachada == nil || m.fachada.ct == nil || m.fachada.fuentes == nil || m.fachada.personal == nil || m.fachada.autoridad == nil || soporte == nil {
		return httpapi.RutaExacta{}, ct.ErrFichaGINPIXV2NoDisponible
	}
	mapa, err := nuevoMapeoFichaGINPIXB2()
	if err != nil {
		return httpapi.RutaExacta{}, err
	}
	f := &fichaGINPIXB2{f: m.fachada, m: mapa, consultarPlan: m.fachada.personal.ConsultarPersonalB2}
	h, err := httpct.NuevoManejadorFichaGINPIXV2(f, f)
	if err != nil {
		return httpapi.RutaExacta{}, err
	}
	return httpapi.RutaExacta{Ruta: httpct.RutaFichaGINPIXV2, Manejador: ligarContextoIncorporacionPersonalB2(h, soporte, fronteras)}, nil
}

func (x *fichaGINPIXB2) ResolverContextoFichaGINPIXV2(ctx context.Context) error {
	if x == nil || x.f == nil || x.f.autoridad == nil || ctx == nil {
		return ct.ErrFichaGINPIXV2NoDisponible
	}
	if _, err := x.f.autoridad.contexto(ctx, ct.AccionLeerPlanNominalB2); err != nil {
		return errorFichaGINPIXB2(ctx, err)
	}
	return nil
}

func (x *fichaGINPIXB2) Preparar(ctx context.Context, exp string) (ginpixfichero.PreparacionExportacion, error) {
	var cero ginpixfichero.PreparacionExportacion
	if x == nil || x.f == nil || x.f.ct == nil || x.f.fuentes == nil || x.consultarPlan == nil || ctx == nil || !dom.ReferenciaOpacaValida(exp) {
		return cero, ct.ErrFichaGINPIXV2NoDisponible
	}
	if ctx.Err() != nil {
		return cero, ct.ErrFichaGINPIXV2NoDisponible
	}
	f := x.f
	c, err := f.ct.LeerContratoPlanNominal(ctx, f.organizacionRef, exp)
	if err != nil {
		return cero, errorFichaGINPIXB2(ctx, err)
	}
	o, ok, err := f.ct.LeerOrigenIncorporacionB2(ctx, f.organizacionRef, exp)
	if err != nil {
		return cero, errorFichaGINPIXB2(ctx, err)
	}
	if !ok {
		return cero, ct.ErrFichaGINPIXV2Conflicto
	}
	p, err := f.consultarConfirmada(ctx, exp, c, o)
	if err != nil {
		return cero, errorFichaGINPIXB2(ctx, err)
	}
	h := o.Confirmacion.Hechos
	if h.ModoPersonal != "alta_empleado" || h.AltaReciboRef == "" || h.AltaReciboRef != h.RelacionReciboRef {
		return cero, ct.ErrFichaGINPIXV2Conflicto
	}
	estado, err := x.consultarPlan(ctx, h.PersonalPlanRef, f.fuentes.organismoRef)
	if err != nil {
		return cero, errorFichaGINPIXB2(ctx, err)
	}
	if !estadoPlanFichaGINPIXB2Valido(estado, c, o, exp, f.fuentes.organismoRef) {
		return cero, ct.ErrFichaGINPIXV2Conflicto
	}
	// Releer el origen y la versión CT tras la verificación propietaria.
	actual, existe, err := f.ct.LeerOrigenIncorporacionB2(ctx, f.organizacionRef, exp)
	if err != nil {
		return cero, errorFichaGINPIXB2(ctx, err)
	}
	detalle, err := f.fuentes.detalleActual(ctx, exp)
	if err != nil {
		return cero, errorFichaGINPIXB2(ctx, err)
	}
	if ctx.Err() != nil {
		return cero, ct.ErrFichaGINPIXV2NoDisponible
	}
	if !existe || !reflect.DeepEqual(actual, o) || detalle.Resumen.Validar() != nil || detalle.Resumen.ExpedienteRef != exp || detalle.Resumen.Version != p.VersionExpedienteActual || detalle.Resumen.OrganizacionRef != f.organizacionRef {
		return cero, ct.ErrFichaGINPIXV2Conflicto
	}
	if !dom.ReferenciaOpacaValida(c.SolicitudPersonalRef) || !dom.ReferenciaOpacaValida(c.IdempotenciaPersonalUUID) {
		return cero, ct.ErrFichaGINPIXV2NoDisponible
	}
	valores := []struct {
		clave dom.ClaveCatalogo
		valor string
	}{
		{"actuacion_ref", o.ReciboRef}, {"expediente_ref", exp}, {"recibo_ref", h.AltaReciboRef},
		{"relacion_ref", h.RelacionRef}, {"seguimiento_ref", h.SeguimientoRef}, {"solicitud_personal_ref", c.SolicitudPersonalRef},
	}
	datos := make([]dom.DatoCanonicoGINPIX, 0, len(valores))
	for _, v := range valores {
		campo, e := dom.CampoValorGINPIX(v.valor)
		if e != nil {
			return cero, ct.ErrFichaGINPIXV2NoDisponible
		}
		datos = append(datos, dom.DatoCanonicoGINPIX{Clave: v.clave, Campo: campo})
	}
	modelo, err := dom.NuevoModeloCanonicoGINPIX(dom.BorradorModeloCanonicoGINPIX{
		Esquema: dom.EsquemaModeloCanonicoGINPIXV1, VersionExpediente: p.VersionExpedienteActual,
		ExpedienteRef: exp, IncorporacionRef: o.ReciboRef, ProcedenciaRef: o.ReciboRef,
		CorrelacionRef: c.SolicitudPersonalRef, IdempotenciaRef: c.IdempotenciaPersonalUUID, Datos: datos,
	})
	if err != nil {
		return cero, ct.ErrFichaGINPIXV2NoDisponible
	}
	mapa, err := dom.RestaurarMapeoVersionadoGINPIX(x.m.m.Publicacion())
	if err != nil {
		return cero, ct.ErrFichaGINPIXV2NoDisponible
	}
	carga, err := dom.AplicarMapeoGINPIX(modelo, mapa)
	if err != nil {
		return cero, ct.ErrFichaGINPIXV2NoDisponible
	}
	return ginpixfichero.PrepararExportacion(carga)
}

// Personal valida y audita esta lectura con V3. Aquí se coteja el plan
// ejecutado y sus recibos históricos con el origen confirmado de CT.
func estadoPlanFichaGINPIXB2Valido(e pp.EstadoPlanIncorporacionCT, c ct.ContratoPlanNominalB2, o ct.OrigenIncorporacionPersonalB2, exp, organismo string) bool {
	h, p := o.Confirmacion.Hechos, e.Plan
	a, ocupacion := e.ReciboAltaRelacion, e.ReciboOcupacion
	return e.Estado == "ejecutado" && p.Validar() == nil && p.Modo == "alta_empleado" &&
		p.PlanRef == h.PersonalPlanRef && uint64(p.Version) == h.PersonalPlanVersion &&
		p.ReciboRef == h.PersonalPlanReciboRef && p.HuellaSHA256 == h.PersonalPlanSHA256 &&
		p.Datos.ExpedienteRef == exp && p.Datos.OrganismoRef == organismo &&
		p.Datos.OrigenCTRef == c.PlanRef && p.Datos.OrigenCTReciboRef == c.PlanReciboRef &&
		p.Datos.OrigenCTHuellaSHA256 == c.PlanSHA256 &&
		a != nil && a.Tipo == "alta" && a.ReciboRef == h.AltaReciboRef &&
		a.ReciboRef == h.RelacionReciboRef && a.EmpleadoRef == h.EmpleadoRef &&
		a.RelacionRef == h.RelacionRef && uint64(a.Version) == h.RelacionVersion &&
		ocupacion != nil && ocupacion.Tipo == "ocupacion" &&
		ocupacion.ReciboRef == h.OcupacionReciboRef && ocupacion.EmpleadoRef == h.EmpleadoRef &&
		ocupacion.RelacionRef == h.RelacionRef && ocupacion.HechoRef == h.OcupacionRef &&
		uint64(ocupacion.Version) == h.OcupacionVersion && e.EjecucionReciboRef == h.SeguimientoRef
}

func errorFichaGINPIXB2(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ct.ErrFichaGINPIXV2NoDisponible
	}
	if errors.Is(err, ct.ErrPlanNominalB2NoEncontrado) || errors.Is(err, ct.ErrPlanNominalB2Conflicto) ||
		errors.Is(err, ct.ErrConflictoIncorporacionAplicacion) || errors.Is(err, ct.ErrPreparacionIncorporacionPendiente) ||
		errors.Is(err, personal.ErrRegistroEmpleadoB2Conflicto) || errors.Is(err, personal.ErrRegistroEmpleadoB2NoEncontrado) ||
		errors.Is(err, httpct.ErrConflictoIncorporacionPersonalB2) || errors.Is(err, httpct.ErrPreparacionPendienteIncorporacionPersonalB2) {
		return ct.ErrFichaGINPIXV2Conflicto
	}
	if errors.Is(err, ct.ErrAutorizacionDenegada) || errors.Is(err, ct.ErrPlanNominalB2Denegado) ||
		errors.Is(err, ct.ErrDenegadaIncorporacionAplicacion) ||
		errors.Is(err, personal.ErrRegistroEmpleadoB2Denegado) || errors.Is(err, httpct.ErrDenegadaIncorporacionPersonalB2) {
		return ct.ErrFichaGINPIXV2Denegada
	}
	return ct.ErrFichaGINPIXV2NoDisponible
}

var _ httpct.AutoridadFichaGINPIXV2 = (*fichaGINPIXB2)(nil)
var _ httpct.PreparadorFichaGINPIXV2 = (*fichaGINPIXB2)(nil)
