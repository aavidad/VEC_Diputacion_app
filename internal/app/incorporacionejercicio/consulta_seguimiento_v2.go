package incorporacionejercicio

import (
	"context"
	"errors"
	"time"

	appct "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// leerSeguimientoV2 no recibe el detalle nominal del expediente. Mantiene la
// consulta V3 actual, coteja el plan y restaura el recibo e historia originales.
func (p *PreparadorDurableV2) leerSeguimientoV2(ctx context.Context, exp string) (lecturaPreparacionV2, error) {
	var z lecturaPreparacionV2
	if p == nil || ctx == nil {
		return z, ct.ErrComposicionIncorporacionAplicacion
	}
	if err := ctx.Err(); err != nil {
		return z, err
	}
	s, err := ct.NuevaSolicitudDetalleRRHH(exp, 0)
	if err != nil {
		return z, ct.ErrIntencionIncorporacionAplicacion
	}
	t, err := p.c.Autoridad.revalidar(ctx, time.Time{})
	if err != nil {
		return z, errorAutoridadPreparacion(ctx, err)
	}
	a := p.c.Autoridad.PreparacionAutoridadCT()
	resumen, err := p.c.Detalle.ConsultarResumenSeguimiento(ctx, s, a.OrganizacionRef, a.UnidadRef)
	if ctx.Err() != nil {
		return z, ctx.Err()
	}
	if err != nil {
		if errors.Is(err, appct.ErrConsultaRRHHNoObservable) || errors.Is(err, ct.ErrAutorizacionDenegada) {
			return z, ct.ErrDenegadaIncorporacionAplicacion
		}
		return z, fallo(ctx, err)
	}
	if resumen.ExpedienteRef != exp || resumen.VersionExpediente == 0 || resumen.VersionExpediente > ct.MaximoEnteroSeguroOperacionAnalisis {
		return z, ct.ErrComposicionIncorporacionAplicacion
	}
	plan, err := p.c.Planes.ResolverPlan(ctx, a.OrganizacionRef, exp)
	if ctx.Err() != nil {
		return z, ctx.Err()
	}
	if err != nil {
		if errors.Is(err, ct.ErrPreparacionIncorporacionPendiente) {
			return z, ct.ErrPreparacionIncorporacionPendiente
		}
		return z, fallo(ctx, err)
	}
	if plan.Validar() != nil || plan.OrganizacionRef != a.OrganizacionRef || plan.UnidadRef != a.UnidadRef || plan.SolicitudPersonal.ExpedienteRef != exp {
		return z, ct.ErrConflictoIncorporacionAplicacion
	}
	sel, encontrado, err := p.c.LocalizadorCT.Localizar(ctx, a.OrganizacionRef, exp, plan.SolicitudPersonal.SolicitudRef)
	if ctx.Err() != nil {
		return z, ctx.Err()
	}
	if err != nil {
		return z, fallo(ctx, err)
	}
	if !encontrado {
		if sel.ReciboRef != "" || sel.MaterialSHA256 != "" || sel.IntencionSHA256 != "" {
			return z, ct.ErrComposicionIncorporacionAplicacion
		}
		return z, ct.ErrConflictoIncorporacionAplicacion
	}
	h, err := p.c.Restaurador.Restaurar(ctx, sel)
	if ctx.Err() != nil {
		return z, ctx.Err()
	}
	if err != nil {
		return z, fallo(ctx, err)
	}
	z.preparacion, z.recibo, err = preparacionOriginalRestaurada(h, sel, a.OrganizacionRef, exp, plan.SolicitudPersonal.SolicitudRef)
	if err != nil {
		return lecturaPreparacionV2{}, err
	}
	z.publicacion, _, z.posterior = h.Historia.EvidenciaSeguimiento()
	z.ultimo = t
	return z, nil
}

// ConsultarSeguimientoIncorporacionV2 entrega únicamente la proyección del
// seguimiento que dejó la incorporación original ya restaurada y validada.
func (p *PeticionV2PostgreSQL) ConsultarSeguimientoIncorporacionV2(ctx context.Context, exp string) (ct.VistaSeguimientoIncorporacionV2, error) {
	var cero ct.VistaSeguimientoIncorporacionV2
	if p == nil || p.preparador == nil || ctx == nil {
		return cero, ct.ErrComposicionIncorporacionAplicacion
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	l, err := p.preparador.leerSeguimientoV2(ctx, exp)
	if err != nil {
		return cero, err
	}
	if l.recibo == nil {
		return cero, ct.ErrConflictoIncorporacionAplicacion
	}
	t, err := p.preparador.finalizar(ctx, l.ultimo)
	if err != nil {
		return cero, err
	}
	if !reciboVisibleValido(*l.recibo, exp, t) {
		return cero, ct.ErrComposicionIncorporacionAplicacion
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	vista, err := appct.ProyectarSeguimientoIncorporacionV2(*l.recibo, l.publicacion, l.posterior)
	if err != nil {
		return cero, err
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	return vista, nil
}
