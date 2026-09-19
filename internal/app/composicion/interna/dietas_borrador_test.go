package interna

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	dietaspg "vec-diputacion-granada/internal/modules/dietas/adapters/postgres"
	vp "vec-diputacion-granada/internal/vec/ports"

	dp "vec-diputacion-granada/internal/modules/dietas/ports"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
)

func TestBorradorDietasComposicionExigeAutoridadesReales(t *testing.T) {
	if m, e := NuevoManejadorBorradorDietas(DependenciasBorradorDietas{}); m != nil || !errors.Is(e, dp.ErrBorradorNoDisponible) {
		t.Fatal("composición vacía habilitada")
	}
}
func TestBorradorDietasSinCapsulaNoResuelveActor(t *testing.T) {
	a := &autoridadBorradorDietas{dependencias: DependenciasBorradorDietas{Identidad: &httpseguridad.ServicioIdentidad{}}}
	for _, ctx := range []context.Context{nil, context.Background()} {
		if _, e := a.ResolverContextoActor(ctx); !errors.Is(e, dp.ErrAccesoBorradorDenegado) {
			t.Fatal("contexto sin cápsula corporativa admitido")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := a.ResolverContextoActor(ctx); !errors.Is(e, dp.ErrAccesoBorradorDenegado) {
		t.Fatal("contexto cancelado admitido")
	}
}

type exportadorSalidaBorrador struct {
	err      error
	cancelar context.CancelFunc
}

func (exportadorSalidaBorrador) String() string       { return "[doble sintético]" }
func (exportadorSalidaBorrador) LogValue() slog.Value { return slog.StringValue("[doble sintético]") }
func (e exportadorSalidaBorrador) ExportarMaterialParaConsumidor() (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	if e.cancelar != nil {
		e.cancelar()
	}
	return vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, e.err
}

// Estas pruebas ejercitan fallos locales de exportación; no acreditan V3 ni PG.
func TestBorradorDietasRechazaSalidaIncompletaPosteriorAEmision(t *testing.T) {
	for _, caso := range []string{"emision", "exportador_nulo", "exportacion", "vinculo", "cancelada", "cancelada_exportacion"} {
		t.Run(caso, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			a := &autoridadBorradorDietas{}
			var exp vp.ExportadorMaterialConsumoAutorizacionAtestadaV3 = exportadorSalidaBorrador{}
			var emision error
			vigente := true
			switch caso {
			case "emision":
				emision = errors.New("fallo sintético")
			case "exportador_nulo":
				exp = nil
			case "exportacion":
				exp = exportadorSalidaBorrador{err: errors.New("fallo sintético")}
			case "vinculo":
				vigente = false
			case "cancelada":
				cancel()
			case "cancelada_exportacion":
				exp = exportadorSalidaBorrador{cancelar: cancel}
			}
			base := dietaspg.EnlaceAutorizacionBorrador{DecisionRef: "dec_sintetica", ContextoRef: "rca_sintetico", CorrelacionRef: "correlacion_sintetica"}
			_, e := a.finalizarEmisionBorrador(ctx, base, exp, emision, func() bool { return vigente })
			if !errors.Is(e, dp.ErrBorradorNoDisponible) || errors.Is(e, dp.ErrAccesoBorradorDenegado) {
				t.Fatal("salida inválida admitida o etiquetada como decisión PDP")
			}
		})
	}
}
