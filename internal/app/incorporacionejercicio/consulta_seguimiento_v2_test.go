package incorporacionejercicio

import (
	"context"
	"errors"
	"testing"

	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestConsultarSeguimientoIncorporacionV2EntregaHitoOriginal(t *testing.T) {
	c := nuevoCasoPreparacionV2(t)
	recibo, err := c.app.s.Confirmar(context.Background(), c.app.i)
	if err != nil {
		t.Fatal(err)
	}
	c.confirmada = true
	p := &PeticionV2PostgreSQL{preparador: c.p}
	vista, err := p.ConsultarSeguimientoIncorporacionV2(context.Background(), c.plan.SolicitudPersonal.ExpedienteRef)
	if err != nil {
		t.Fatal(err)
	}
	hito := vista.Actuaciones[len(vista.Actuaciones)-1]
	if vista.Alcance != "original_incorporacion" || vista.ExpedienteRef != recibo.ExpedienteRef ||
		vista.ReciboIncorporacionRef != recibo.ReciboRef || vista.SeguimientoRef != recibo.SeguimientoRef ||
		vista.VersionSeguimiento != recibo.VersionSeguimientoResultante || hito.ActuacionRef != recibo.ActuacionRef ||
		hito.RegistradaEn != recibo.RegistradaEn {
		t.Fatal("la consulta no conserva el hito original de incorporacion")
	}
}

func TestConsultarSeguimientoIncorporacionV2DeniegaAntesDeExportarYSinReciboConflicto(t *testing.T) {
	t.Run("denegada_antes_de_exportar", func(t *testing.T) {
		c := nuevoCasoPreparacionV2(t)
		c.p.c.Detalle = detallePreparacionDoble(func(context.Context, ct.SolicitudDetalleRRHH) (ct.DetalleExpedienteRRHH, error) {
			return ct.DetalleExpedienteRRHH{}, ct.ErrAutorizacionDenegada
		})
		p := &PeticionV2PostgreSQL{preparador: c.p}
		_, err := p.ConsultarSeguimientoIncorporacionV2(context.Background(), c.plan.SolicitudPersonal.ExpedienteRef)
		if !errors.Is(err, ct.ErrDenegadaIncorporacionAplicacion) || c.app.ctTX.llamadas != 0 {
			t.Fatalf("denegacion no detenida antes de exportar: %v", err)
		}
	})
	t.Run("sin_recibo", func(t *testing.T) {
		c := nuevoCasoPreparacionV2(t)
		p := &PeticionV2PostgreSQL{preparador: c.p}
		_, err := p.ConsultarSeguimientoIncorporacionV2(context.Background(), c.plan.SolicitudPersonal.ExpedienteRef)
		if !errors.Is(err, ct.ErrConflictoIncorporacionAplicacion) {
			t.Fatalf("sin recibo no devuelve conflicto: %v", err)
		}
	})
}
