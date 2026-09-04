package contrataciontemporal

import (
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type autoridadFiscalizacionComposicionPrueba struct{}

func (*autoridadFiscalizacionComposicionPrueba) ResolverContextoCanalFiscalizacion(
	context.Context,
) (httpinterno.ContextoCanalFiscalizacion, error) {
	return httpinterno.ContextoCanalFiscalizacion{}, errors.New("no debe ejecutarse")
}

type ejecutorFiscalizacionComposicionPrueba struct{}

func (*ejecutorFiscalizacionComposicionPrueba) RegistrarResultado(
	context.Context,
	application.SolicitudRegistrarResultadoFiscalizacion,
) (ports.ReciboFiscalizacion, error) {
	return ports.ReciboFiscalizacion{}, errors.New("no debe ejecutarse")
}

func TestNuevasRutasRegistraResultadoFiscalizacionUnaVez(t *testing.T) {
	t.Parallel()
	rutas, err := NuevasRutas(dependenciasRutasPrueba())
	if err != nil {
		t.Fatalf("construir rutas: %v", err)
	}
	registros := 0
	for _, ruta := range rutas {
		if ruta.Ruta == httpinterno.RutaResultadosFiscalizacion {
			registros++
			if ruta.Manejador == nil {
				t.Fatal("ruta sin manejador")
			}
		}
	}
	if registros != 1 {
		t.Fatalf("registros de fiscalizacion = %d", registros)
	}
}

func TestNuevasRutasFiscalizacionRechazaDependenciasNulasAtomicamente(t *testing.T) {
	t.Parallel()
	casos := []struct {
		nombre   string
		preparar func(*DependenciasRutas)
	}{
		{"autoridad nil", func(d *DependenciasRutas) {
			d.AutoridadFiscalizacion = nil
		}},
		{"ejecutor nil", func(d *DependenciasRutas) {
			d.EjecutorFiscalizacion = nil
		}},
		{"autoridad nil tipada", func(d *DependenciasRutas) {
			var nula *autoridadFiscalizacionComposicionPrueba
			d.AutoridadFiscalizacion = nula
		}},
		{"ejecutor nil tipado", func(d *DependenciasRutas) {
			var nulo *ejecutorFiscalizacionComposicionPrueba
			d.EjecutorFiscalizacion = nulo
		}},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			dependencias := dependenciasRutasPrueba()
			caso.preparar(&dependencias)
			rutas, err := NuevasRutas(dependencias)
			if rutas != nil || !errors.Is(err, ErrRutasContratacionTemporalInvalidas) {
				t.Fatalf("resultado = (%#v, %v)", rutas, err)
			}
		})
	}
}
