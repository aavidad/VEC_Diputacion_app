package incorporacionejercicio

import (
	"bytes"
	"context"
	"errors"
	"time"
	appct "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	dom "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/modules/personal/adapters/fuenteejercicio"
	core "vec-diputacion-granada/internal/vec/domain"
)

type PreparadorDurableV2 struct {
	c ConfiguracionPreparacionDurableV2
}

func NuevoPreparadorDurableV2(c ConfiguracionPreparacionDurableV2) (*PreparadorDurableV2, error) {
	for _, d := range []any{c.Autoridad, c.Detalle, c.Planes, c.Inicial, c.LocalizadorCT, c.Restaurador, c.LocalizadorPersonal, c.LectorPersonal, c.Reloj} {
		if nulo(d) {
			return nil, ct.ErrComposicionIncorporacionAplicacion
		}
	}
	// La recuperación CT original no depende de que siga disponible una fuente
	// de altas actual. Su sello se comprueba sólo al preparar un nuevo intento.
	c.FuentePersonal = bytes.Clone(c.FuentePersonal)
	return &PreparadorDurableV2{c: c}, nil
}

type lecturaPreparacionV2 struct {
	plan        PlanPreparacionDurableV2
	detalle     ct.DetalleExpedienteRRHH
	preparacion ct.PreparacionIncorporacionAplicacionV2
	recibo      *ct.ReciboIncorporacionAplicacionV2
	ultimo      time.Time
}

// leer siempre empieza por consulta nominal ACTUAL. Ni CT81 ni el restaurador
// histórico conceden acceso vigente; no hay caché de órdenes o recibos.
func (p *PreparadorDurableV2) leer(ctx context.Context, exp string, version uint64) (lecturaPreparacionV2, error) {
	var z lecturaPreparacionV2
	if ctx == nil || p == nil {
		return z, ct.ErrComposicionIncorporacionAplicacion
	}
	if err := ctx.Err(); err != nil {
		return z, err
	}
	s, err := ct.NuevaSolicitudDetalleRRHH(exp, version)
	if err != nil {
		return z, ct.ErrIntencionIncorporacionAplicacion
	}
	t, err := p.c.Autoridad.revalidar(ctx, time.Time{})
	if err != nil {
		return z, errorAutoridadPreparacion(ctx, err)
	}
	d, err := p.c.Detalle.Consultar(ctx, s)
	if ctx.Err() != nil {
		return z, ctx.Err()
	}
	if err != nil {
		if errors.Is(err, appct.ErrConsultaRRHHNoObservable) || errors.Is(err, ct.ErrAutorizacionDenegada) {
			return z, ct.ErrDenegadaIncorporacionAplicacion
		}
		return z, fallo(ctx, err)
	}
	a := p.c.Autoridad.PreparacionAutoridadCT()
	if d.ValidarContenidoPublicablePara(s) != nil || d.Resumen.OrganizacionRef != a.OrganizacionRef || d.Resumen.UnidadRef != a.UnidadRef {
		return z, ct.ErrConflictoIncorporacionAplicacion
	}
	plan, err := p.c.Planes.ResolverPlan(ctx, a.OrganizacionRef, exp)
	if ctx.Err() != nil {
		return z, ctx.Err()
	}
	if err != nil {
		return z, fallo(ctx, err)
	}
	if plan.Validar() != nil || plan.OrganizacionRef != a.OrganizacionRef || plan.UnidadRef != a.UnidadRef || plan.SolicitudPersonal.ExpedienteRef != exp {
		return z, ct.ErrConflictoIncorporacionAplicacion
	}
	z = lecturaPreparacionV2{plan: plan.Copia(), detalle: d.Clonar(), ultimo: t}
	sel, encontrado, err := p.c.LocalizadorCT.Localizar(ctx, a.OrganizacionRef, exp, plan.SolicitudPersonal.SolicitudRef)
	if ctx.Err() != nil {
		return lecturaPreparacionV2{}, ctx.Err()
	}
	if err != nil {
		return lecturaPreparacionV2{}, fallo(ctx, err)
	}
	if encontrado {
		h, err := p.c.Restaurador.Restaurar(ctx, sel)
		if ctx.Err() != nil {
			return lecturaPreparacionV2{}, ctx.Err()
		}
		if err != nil {
			return lecturaPreparacionV2{}, fallo(ctx, err)
		}
		z.preparacion, z.recibo, err = preparacionOriginalRestaurada(h, sel, a.OrganizacionRef, exp, plan.SolicitudPersonal.SolicitudRef)
		if err != nil {
			return lecturaPreparacionV2{}, err
		}
		// La versión del recibo sigue siendo la del commit original, incluso
		// si el expediente observado ha continuado por otro trámite.
		return z, nil
	}
	if sel.ReciboRef != "" || sel.MaterialSHA256 != "" || sel.IntencionSHA256 != "" {
		return lecturaPreparacionV2{}, ct.ErrComposicionIncorporacionAplicacion
	}
	if plan.SolicitudPersonal.VersionExpediente > d.Resumen.Version || plan.VersionExpedienteRaiz > d.Resumen.Version {
		return lecturaPreparacionV2{}, ct.ErrConflictoIncorporacionAplicacion
	}
	// El localizador Personal sólo aporta selectores. El consumidor V2 relee
	// y acredita el original con su permiso propio antes de reutilizarlo.
	local, parcial, err := p.c.LocalizadorPersonal.Localizar(ctx, a.OrganizacionRef, exp, plan.SolicitudPersonal.SolicitudRef)
	if ctx.Err() != nil {
		return lecturaPreparacionV2{}, ctx.Err()
	}
	if err != nil {
		return lecturaPreparacionV2{}, fallo(ctx, err)
	}
	if parcial {
		if local.Solicitud != plan.SolicitudPersonal || !selectorPersonalDelPlan(local.Selector, plan) {
			return lecturaPreparacionV2{}, ct.ErrConflictoIncorporacionAplicacion
		}
		contexto, err := p.c.Autoridad.ContextoAutoridad()
		if err != nil {
			return lecturaPreparacionV2{}, errorAutoridadPreparacion(ctx, err)
		}
		original, err := p.c.LectorPersonal.Leer(ctx, local.Selector, plan.UnidadRef, contexto)
		if ctx.Err() != nil {
			return lecturaPreparacionV2{}, ctx.Err()
		}
		if err != nil {
			return lecturaPreparacionV2{}, fallo(ctx, err)
		}
		ahora := p.c.Reloj.Ahora()
		if ctx.Err() != nil {
			return lecturaPreparacionV2{}, ctx.Err()
		}
		if !dom.InstanteUTCCanonico(ahora) || ahora.Before(t) || !originalPersonalDelPlan(original, local.Selector, plan, ahora) {
			return lecturaPreparacionV2{}, ct.ErrComposicionIncorporacionAplicacion
		}
		z.ultimo = ahora
	} else if local.Solicitud != (ct.SolicitudAltaPersonalRPT{}) || local.Selector.SolicitudRef != "" {
		return lecturaPreparacionV2{}, ct.ErrComposicionIncorporacionAplicacion
	}
	if plan.FuentePersonal != p.c.TernaPersonal {
		return lecturaPreparacionV2{}, ct.ErrConflictoIncorporacionAplicacion
	}
	fuente, err := fuenteejercicio.NuevaFuenteEjercicio(p.c.FuentePersonal, p.c.TernaPersonal)
	if err != nil {
		return lecturaPreparacionV2{}, fallo(ctx, err)
	}
	vinculo, err := fuente.Resolver(ctx, plan.SolicitudPersonal)
	if ctx.Err() != nil {
		return lecturaPreparacionV2{}, ctx.Err()
	}
	if err != nil {
		return lecturaPreparacionV2{}, fallo(ctx, err)
	}
	// No convertir fechas civiles en instantes: el plan declara ambos límites
	// UTC. Sólo se cotejan los días de la fuente propietaria ya resuelta.
	if vinculo.CentroRef != d.Resumen.CentroRef || plan.Periodo.Desde.Format("2006-01-02") != vinculo.Desde || plan.Periodo.Hasta.Format("2006-01-02") != vinculo.Hasta {
		return lecturaPreparacionV2{}, ct.ErrConflictoIncorporacionAplicacion
	}
	pub, estado, versionRaiz, err := p.c.Inicial.LeerPreparacionInicial(ctx, plan.OrganizacionRef, exp, plan.RelacionRef)
	if ctx.Err() != nil {
		return lecturaPreparacionV2{}, ctx.Err()
	}
	if err != nil {
		return lecturaPreparacionV2{}, fallo(ctx, err)
	}
	ahora := p.c.Reloj.Ahora()
	if ctx.Err() != nil {
		return lecturaPreparacionV2{}, ctx.Err()
	}
	if !dom.InstanteUTCCanonico(ahora) || ahora.Before(z.ultimo) {
		return lecturaPreparacionV2{}, ct.ErrComposicionIncorporacionAplicacion
	}
	z.ultimo = ahora
	if err = validarInicialPreparacion(plan, pub, estado, versionRaiz, ahora); err != nil {
		return lecturaPreparacionV2{}, err
	}
	z.preparacion = ct.PreparacionIncorporacionAplicacionV2{SolicitudPersonal: plan.SolicitudPersonal,
		VersionActualExpediente: d.Resumen.Version, VersionSeguimientoEsperada: estado.Version,
		Periodo: plan.Periodo, MotivoClave: plan.MotivoClave, Documentos: append([]dom.DocumentoSeguimiento(nil), plan.Documentos...), MotivoV3: plan.MotivoV3}
	return z, nil
}

func (p *PreparadorDurableV2) finalizar(ctx context.Context, previo time.Time) (time.Time, error) {
	t, err := p.c.Autoridad.revalidar(ctx, previo)
	if err != nil {
		return time.Time{}, errorAutoridadPreparacion(ctx, err)
	}
	f := p.c.Reloj.Ahora()
	if ctx.Err() != nil {
		return time.Time{}, ctx.Err()
	}
	if !dom.InstanteUTCCanonico(f) || f.Before(t) {
		return time.Time{}, ct.ErrComposicionIncorporacionAplicacion
	}
	return f, nil
}

func (p *PreparadorDurableV2) Consultar(ctx context.Context, exp string) (ct.ProyeccionIncorporacionAplicacionV2, error) {
	var z ct.ProyeccionIncorporacionAplicacionV2
	l, err := p.leer(ctx, exp, 0)
	if err != nil {
		return z, err
	}
	out := ct.ProyeccionIncorporacionAplicacionV2{Esquema: "vec.contratacion-temporal.incorporacion-ejercicio.preparacion.v2", ExpedienteRef: exp, VersionActualExpediente: l.detalle.Resumen.Version, Recibo: l.recibo}
	if l.recibo == nil {
		x := l.preparacion
		refs := make([]string, 0, len(x.Documentos))
		for _, d := range x.Documentos {
			refs = append(refs, d.Referencia)
		}
		out.Preparacion = &ct.PreparacionVisibleIncorporacionV2{SolicitudPersonalRef: x.SolicitudPersonal.SolicitudRef, VersionSolicitudPersonal: x.SolicitudPersonal.VersionExpediente,
			VersionSeguimientoEsperada: x.VersionSeguimientoEsperada, Periodo: x.Periodo, Motivos: []dom.ClaveCatalogo{x.MotivoClave}, DocumentosRefs: refs, Disponible: true}
	}
	t, err := p.finalizar(ctx, l.ultimo)
	if err != nil {
		return z, err
	}
	if !proyeccionValida(out, exp, t) {
		return z, ct.ErrComposicionIncorporacionAplicacion
	}
	return out.Copia(), nil
}

func (p *PreparadorDurableV2) Preparar(ctx context.Context, i ct.IntencionIncorporacionAplicacionV2) (ct.PreparacionIncorporacionAplicacionV2, error) {
	var z ct.PreparacionIncorporacionAplicacionV2
	if ctx == nil {
		return z, ct.ErrComposicionIncorporacionAplicacion
	}
	if ctx.Err() != nil {
		return z, ctx.Err()
	}
	if i.Validar() != nil {
		return z, ct.ErrIntencionIncorporacionAplicacion
	}
	i = i.Copia()
	l, err := p.leer(ctx, i.ExpedienteRef, i.VersionActualExpedienteObservada)
	if err != nil {
		return z, err
	}
	x := l.preparacion
	// Nunca actualizar la intención original para ocultar un conflicto.
	if x.SolicitudPersonal.SolicitudRef != i.SolicitudPersonalRef || x.VersionActualExpediente != i.VersionActualExpedienteObservada || x.MotivoClave != i.MotivoClave || !documentosIntencionExactos(x.Documentos, i.DocumentosRefs) {
		return z, ct.ErrConflictoIncorporacionAplicacion
	}
	x.Preparacion = p.c.Autoridad.PreparacionAutoridadCT()
	x.SolicitudContexto = p.c.Autoridad.solicitudContexto()
	x.Contexto, err = p.c.Autoridad.ContextoAutoridad()
	if err != nil {
		return z, errorAutoridadPreparacion(ctx, err)
	}
	// Sólo la correlación de autorización es fresca. Solicitud/idempotencia y
	// datos de transición permanecen inmutables y salen del plan/original.
	x.CorrelacionV3, err = core.GenerarReferenciaCorrelacionAutorizacionV2(ctx, p.c.Autoridad.correlador)
	if ctx.Err() != nil {
		return z, ctx.Err()
	}
	if err != nil {
		return z, fallo(ctx, err)
	}
	t, err := p.finalizar(ctx, l.ultimo)
	if err != nil {
		return z, err
	}
	if err = validarPreparacion(x, i, t); err != nil {
		return z, err
	}
	return clonarPreparacion(x)
}

func errorAutoridadPreparacion(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return ct.ErrDenegadaIncorporacionAplicacion
}

var _ ct.ProveedorPreparacionIncorporacionAplicacionV2 = (*PreparadorDurableV2)(nil)
