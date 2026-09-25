package incorporacionejercicio

import (
	"context"
	"errors"
	"testing"

	appct "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type consultaSoloSeguimientoDoble struct {
	resumen appct.ResumenConsultaSeguimientoRRHH
	err     error
}

func (d consultaSoloSeguimientoDoble) Consultar(context.Context, ct.SolicitudDetalleRRHH) (ct.DetalleExpedienteRRHH, error) {
	panic("la consulta de seguimiento intento exportar el detalle completo")
}

func (d consultaSoloSeguimientoDoble) ConsultarResumenSeguimiento(_ context.Context, _ ct.SolicitudDetalleRRHH, _, _ string) (appct.ResumenConsultaSeguimientoRRHH, error) {
	return d.resumen, d.err
}

func TestConsultarSeguimientoIncorporacionV2EntregaHitoOriginal(t *testing.T) {
	c := nuevoCasoPreparacionV2(t)
	recibo, err := c.app.s.Confirmar(context.Background(), c.app.i)
	if err != nil {
		t.Fatal(err)
	}
	c.confirmada = true
	c.p.c.Detalle = consultaSoloSeguimientoDoble{resumen: appct.ResumenConsultaSeguimientoRRHH{
		ExpedienteRef: c.plan.SolicitudPersonal.ExpedienteRef, VersionExpediente: c.detalle.Resumen.Version,
	}}
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

func TestConsultarSeguimientoIncorporacionV2RechazaProyeccionCruzada(t *testing.T) {
	c := nuevoCasoPreparacionV2(t)
	c.confirmada = true
	c.p.c.Detalle = consultaSoloSeguimientoDoble{resumen: appct.ResumenConsultaSeguimientoRRHH{
		ExpedienteRef: "expediente:ajeno", VersionExpediente: c.detalle.Resumen.Version,
	}}
	p := &PeticionV2PostgreSQL{preparador: c.p}
	_, err := p.ConsultarSeguimientoIncorporacionV2(context.Background(), c.plan.SolicitudPersonal.ExpedienteRef)
	if !errors.Is(err, ct.ErrComposicionIncorporacionAplicacion) || len(c.pasos) != 0 {
		t.Fatalf("proyeccion cruzada continuo a la restauracion: %v, %v", err, c.pasos)
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
