package incorporacionejercicio

import (
	"context"

	pgct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	dom "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// Misma fuente sellada y mismo lector CT78. El plan inicial no consulta una
// relación que todavía no existe. El candidato nace de la orden posterior al
// alta y a la acreditación Personal; CT75 vuelve a cotejar y consumir todo.
type resolutorRaizPlanV2 struct {
	planes FuentePlanesPreparacionV2
	previo pgct.ResolverRaizIncorporacionV2
	reloj  ct.Reloj
}

func (r *resolutorRaizPlanV2) ResolverSeguimientoIncorporacionV2(ctx context.Context, o ct.OrdenConfirmacionIncorporacionV2) (string, error) {
	return r.previo.ResolverSeguimientoIncorporacionV2(ctx, o)
}

func (r *resolutorRaizPlanV2) ResolverRaizInicialIncorporacionV2(ctx context.Context, o ct.OrdenConfirmacionIncorporacionV2) (pgct.PreparacionRaizIncorporacionV2, error) {
	z := pgct.PreparacionRaizIncorporacionV2{}
	if r == nil || ctx == nil || nulo(r.planes) || nulo(r.reloj) || nulo(r.previo) {
		return z, ct.ErrRegistroIncorporacionV2
	}
	if err := ctx.Err(); err != nil {
		return z, err
	}
	t := r.reloj.Ahora()
	if o.ValidarEn(t) != nil {
		return z, ct.ErrRegistroIncorporacionV2
	}
	m, err := o.Material().Datos()
	if err != nil {
		return z, err
	}
	p, err := r.planes.ResolverPlan(ctx, m.Preparacion.OrganizacionRef, m.Personal.Solicitud.ExpedienteRef)
	if err != nil {
		return z, fallo(ctx, err)
	}
	if p.Validar() != nil || p.SolicitudPersonal != m.Personal.Solicitud || p.UnidadRef != m.Preparacion.UnidadRef || p.OrganizacionRef != m.Preparacion.OrganizacionRef {
		return z, ct.ErrConflictoIncorporacionAplicacion
	}
	if p.PublicacionInicial.Referencia == "" {
		ref, err := r.previo.ResolverSeguimientoIncorporacionV2(ctx, o)
		if err != nil {
			return z, err
		}
		return pgct.RaizIncorporacionV2Existente(ref)
	}
	def, err := dom.RestaurarDefinicionSeguimiento(p.PublicacionInicial)
	if err != nil || p.VersionExpedienteRaiz != m.VersionActualExpediente || p.MotivoClave != m.Confirmacion.MotivoClave || !documentosRaizV2Exactos(p.Documentos, m.Confirmacion.Documentos) {
		return z, ct.ErrConflictoIncorporacionAplicacion
	}
	return pgct.NuevaPreparacionRaizIncorporacionV2(def, p.SeguimientoRef, p.Periodo, o, t)
}

func documentosRaizV2Exactos(plan, material []dom.DocumentoSeguimiento) bool {
	if len(plan) != len(material) {
		return false
	}
	vistos := make(map[dom.DocumentoSeguimiento]bool, len(plan))
	for _, d := range plan {
		vistos[d] = true
	}
	for _, d := range material {
		if !vistos[d] {
			return false
		}
		delete(vistos, d)
	}
	return len(vistos) == 0
}
