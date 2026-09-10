package incorporacionejercicio

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	dom "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	lector "vec-diputacion-granada/internal/modules/personal/adapters/lecturaincorporacion"
)

// Sólo fronteras IO dobles del fixture existente; no acredita PostgreSQL.
func casoPlanSinRelacionV2(t *testing.T) *casoPreparacionV2 {
	t.Helper()
	c := nuevoCasoPreparacionV2(t)
	c.plan.PublicacionInicial, c.plan.RelacionRef = c.pub, ""
	b := documentoPlanesV2Prueba(t, c.plan)
	f, err := NuevaFuentePlanesPreparacionV2(b, ternaPlanesV2Prueba(b))
	registroV2Exigir(t, err)
	c.p.c.Planes = f // parser y fuente sellada reales, sin otro proveedor.
	c.p.c.Inicial = inicialPreparacionDoble(func(context.Context, string, string, string) (dom.PublicacionDefinicionSeguimiento, dom.EstadoPersistidoSeguimiento, uint64, error) {
		t.Fatal("GET inicial intentó leer una relación futura")
		return dom.PublicacionDefinicionSeguimiento{}, dom.EstadoPersistidoSeguimiento{}, 0, nil
	})
	return c
}

func TestPreparacionInicialV2SinRelacionFutura(t *testing.T) {
	c := casoPlanSinRelacionV2(t)
	ctx := context.Background()
	uno, err := c.p.Consultar(ctx, c.plan.SolicitudPersonal.ExpedienteRef)
	registroV2Exigir(t, err)
	dos, err := c.p.Consultar(ctx, c.plan.SolicitudPersonal.ExpedienteRef)
	registroV2Exigir(t, err)
	if !reflect.DeepEqual(uno, dos) || uno.Recibo != nil || uno.Preparacion == nil || !uno.Preparacion.Disponible ||
		uno.Preparacion.VersionSolicitudPersonal != 7 || uno.VersionActualExpediente != 8 || uno.Preparacion.VersionSeguimientoEsperada != 0 {
		t.Fatal("preparación no estable 7/8/0")
	}
	preparada, err := c.p.Preparar(ctx, c.app.i)
	registroV2Exigir(t, err)
	if preparada.SolicitudPersonal != c.plan.SolicitudPersonal || preparada.VersionActualExpediente != 8 || preparada.Periodo != c.plan.Periodo {
		t.Fatal("intención no ligada al mismo plan")
	}
	// El original parcial se localiza, pero sólo se reutiliza tras su lectura
	// autorizada actual. La relación procede del original, nunca del plan.
	original, err := c.app.s.Confirmar(ctx, c.app.i) // fábrica nominal del fixture, no PG.
	registroV2Exigir(t, err)
	altas, confirmaciones := c.app.alta.n, c.app.ctTX.llamadas
	c.parcial = true
	lecturas := 0
	c.p.c.LectorPersonal = lectorPersonalPreparacionDoble(func(_ context.Context, s lector.Selector, u string, a ct.ContextoAutorizacionAltaV3) (lector.Resultado, error) {
		lecturas++
		o := c.app.alta.registro(t)
		if s.RelacionRef != o.Resultado.RelacionRef || u != c.plan.UnidadRef || !mismoContexto(a, c.a.a.contexto) {
			t.Fatal("original parcial cruzado")
		}
		return lector.Resultado{Registro: o, DecisionLecturaRef: "decision:lectura:actual", AuditoriaLecturaRef: "auditoria:lectura:actual", ConsumoHuellaSHA256: strings.Repeat("a", 64), LeidaEn: c.a.ahora}, nil
	})
	parcial, err := c.p.Consultar(ctx, c.plan.SolicitudPersonal.ExpedienteRef)
	registroV2Exigir(t, err)
	if lecturas != 1 || !reflect.DeepEqual(parcial, uno) {
		t.Fatal("la recuperación parcial cambió la preparación")
	}
	c.confirmada = true
	c.p.c.FuentePersonal = nil
	recuperada, err := c.p.Consultar(ctx, c.plan.SolicitudPersonal.ExpedienteRef)
	registroV2Exigir(t, err)
	if recuperada.Recibo == nil || !reflect.DeepEqual(*recuperada.Recibo, original) || recuperada.Preparacion != nil ||
		c.app.alta.n != altas || c.app.ctTX.llamadas != confirmaciones {
		t.Fatal("GET duplicó efectos o alteró original confirmado")
	}
}

func TestPlanInicialV2ConflictosYCopias(t *testing.T) {
	c := casoPlanSinRelacionV2(t)
	for nombre, mutar := range map[string]func(*PlanPreparacionDurableV2){
		"relacion_anticipada":  func(p *PlanPreparacionDurableV2) { p.RelacionRef = c.estado.RelacionRef },
		"publicacion_cruzada":  func(p *PlanPreparacionDurableV2) { p.PublicacionInicial.Referencia = registroV2Ref("otra:definicion") },
		"publicacion_truncada": func(p *PlanPreparacionDurableV2) { p.PublicacionInicial.Referencia = "" },
		"raiz_no_tipificada":   func(p *PlanPreparacionDurableV2) { p.SeguimientoRef = "seguimiento:sin-canon" },
	} {
		t.Run(nombre, func(t *testing.T) {
			p := c.plan.Copia()
			mutar(&p)
			if p.Validar() == nil {
				t.Fatal("plan ambiguo aceptado")
			}
		})
	}
	p, err := c.p.c.Planes.ResolverPlan(context.Background(), c.plan.OrganizacionRef, c.plan.SolicitudPersonal.ExpedienteRef)
	registroV2Exigir(t, err)
	p.PublicacionInicial.Transiciones[0].MotivosPermitidos[0] = "alterado"
	releido, err := c.p.c.Planes.ResolverPlan(context.Background(), c.plan.OrganizacionRef, c.plan.SolicitudPersonal.ExpedienteRef)
	registroV2Exigir(t, err)
	if releido.Validar() != nil || !reflect.DeepEqual(releido, c.plan) {
		t.Fatal("fuente sellada mutable")
	}
	// Una versión actual distinta del plan no habilita la actuación.
	c.detalle.Resumen.Version = 9
	if _, err = c.p.Consultar(context.Background(), c.plan.SolicitudPersonal.ExpedienteRef); err == nil {
		t.Fatal("expediente avanzado habilitado")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err = c.p.Consultar(ctx, c.plan.SolicitudPersonal.ExpedienteRef); !errors.Is(err, context.Canceled) {
		t.Fatal("cancelación no conservada")
	}
}

type raizAnteriorV2Doble struct{}

func (raizAnteriorV2Doble) ResolverSeguimientoIncorporacionV2(context.Context, ct.OrdenConfirmacionIncorporacionV2) (string, error) {
	panic("resolver inicial no debe leer la relación futura")
}

func TestResolutorPlanInicialV2UsaOrdenPersonal(t *testing.T) {
	c := casoPlanSinRelacionV2(t)
	c.app.p.p.Periodo = c.plan.Periodo // el doble del servicio recibe el mismo plan.
	_, err := c.app.s.Confirmar(context.Background(), c.app.i)
	registroV2Exigir(t, err)
	r := &resolutorRaizPlanV2{planes: c.p.c.Planes, previo: raizAnteriorV2Doble{}, reloj: c.app.s.c.Reloj}
	_, err = r.ResolverRaizInicialIncorporacionV2(context.Background(), c.app.ctTX.original)
	registroV2Exigir(t, err)
	r.planes = planPreparacionDoble(func(context.Context, string, string) (PlanPreparacionDurableV2, error) {
		p := c.plan.Copia()
		p.Documentos[0].TipoClave = "tipo_cruzado"
		return p, nil
	})
	if _, err = r.ResolverRaizInicialIncorporacionV2(context.Background(), c.app.ctTX.original); err == nil {
		t.Fatal("documento del plan ajeno al material aceptado")
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err = r.ResolverRaizInicialIncorporacionV2(ctx, c.app.ctTX.original); !errors.Is(err, context.Canceled) {
		t.Fatal("cancelación del resolutor no conservada")
	}
}
