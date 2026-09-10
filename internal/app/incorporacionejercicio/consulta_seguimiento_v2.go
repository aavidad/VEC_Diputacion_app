package incorporacionejercicio

import (
	"context"

	appct "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

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
	l, err := p.preparador.leer(ctx, exp, 0)
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
