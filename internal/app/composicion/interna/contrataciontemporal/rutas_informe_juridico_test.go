package contrataciontemporal

import (
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type autoridadInformeJuridicoComposicionPrueba struct{}

func (*autoridadInformeJuridicoComposicionPrueba) ResolverContextoCanalInformeJuridico(
	context.Context,
) (httpinterno.ContextoCanalInformeJuridico, error) {
	return httpinterno.ContextoCanalInformeJuridico{}, errors.New(
		"no debe ejecutarse",
	)
}

type ejecutorInformeJuridicoComposicionPrueba struct{}

func (*ejecutorInformeJuridicoComposicionPrueba) Emitir(
	context.Context,
	application.SolicitudEmitirInformeJuridico,
) (ports.ReciboInformeJuridico, error) {
	return ports.ReciboInformeJuridico{}, errors.New("no debe ejecutarse")
}

func TestNuevasRutasRegistraPreparacionInformeJuridicoUnaVez(t *testing.T) {
	t.Parallel()
	rutas, err := NuevasRutas(dependenciasRutasPrueba())
	if err != nil {
		t.Fatalf("construir rutas: %v", err)
	}
	registros := 0
	for _, ruta := range rutas {
		if ruta.Ruta == httpinterno.RutaPreparacionesInformeJuridico {
			registros++
			if ruta.Manejador == nil {
				t.Fatal("ruta sin manejador")
			}
		}
	}
	if registros != 1 {
		t.Fatalf("registros de informe juridico = %d", registros)
	}
}

func TestNuevasRutasInformeJuridicoRechazaDependenciasNulasAtomicamente(
	t *testing.T,
) {
	t.Parallel()
	casos := []struct {
		nombre   string
		preparar func(*DependenciasRutas)
	}{
		{"autoridad nil", func(d *DependenciasRutas) {
			d.AutoridadInformeJuridico = nil
		}},
		{"ejecutor nil", func(d *DependenciasRutas) {
			d.EjecutorInformeJuridico = nil
		}},
		{"autoridad nil tipada", func(d *DependenciasRutas) {
			var nula *autoridadInformeJuridicoComposicionPrueba
			d.AutoridadInformeJuridico = nula
		}},
		{"ejecutor nil tipado", func(d *DependenciasRutas) {
			var nulo *ejecutorInformeJuridicoComposicionPrueba
			d.EjecutorInformeJuridico = nulo
		}},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			dependencias := dependenciasRutasPrueba()
			caso.preparar(&dependencias)
			rutas, err := NuevasRutas(dependencias)
			if rutas != nil ||
				!errors.Is(err, ErrRutasContratacionTemporalInvalidas) {
				t.Fatalf("resultado = (%#v, %v)", rutas, err)
			}
		})
	}
}
