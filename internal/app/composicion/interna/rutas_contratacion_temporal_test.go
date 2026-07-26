package interna

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type autoridadAltaComposicionPrueba struct{}

func (autoridadAltaComposicionPrueba) ResolverContextoCanalAlta(
	context.Context,
) (application.SolicitudRegistrarExpediente, error) {
	return application.SolicitudRegistrarExpediente{}, errors.New(
		"no debe ejecutarse",
	)
}

type ejecutorAltaComposicionPrueba struct{}

func (ejecutorAltaComposicionPrueba) Registrar(
	context.Context,
	application.SolicitudRegistrarExpediente,
) (ports.ReciboAlta, error) {
	return ports.ReciboAlta{}, errors.New("no debe ejecutarse")
}

type relojComposicionPrueba struct{}

func (relojComposicionPrueba) Ahora() time.Time {
	return time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC)
}

type autoridadCoberturaComposicionPrueba struct{}

func (autoridadCoberturaComposicionPrueba) ResolverContextoCanalCobertura(
	context.Context,
) (httpinterno.ContextoCanalCobertura, error) {
	return httpinterno.ContextoCanalCobertura{}, errors.New("no debe ejecutarse")
}

type presentadorCoberturaComposicionPrueba struct{}

func (presentadorCoberturaComposicionPrueba) ProponerParaAdaptador(
	context.Context,
	application.SolicitudProponerCobertura,
) (application.ResultadoPropuestaCoberturaParaAdaptador, error) {
	return application.ResultadoPropuestaCoberturaParaAdaptador{},
		errors.New("no debe ejecutarse")
}

type decisorCoberturaComposicionPrueba struct{}

func (decisorCoberturaComposicionPrueba) DecidirParaAdaptador(
	context.Context,
	application.SolicitudDecidirCobertura,
) (application.ResultadoDecisionCoberturaParaAdaptador, error) {
	return application.ResultadoDecisionCoberturaParaAdaptador{},
		errors.New("no debe ejecutarse")
}

func (decisorCoberturaComposicionPrueba) RectificarParaAdaptador(
	context.Context,
	application.SolicitudRectificarCobertura,
) (application.ResultadoDecisionCoberturaParaAdaptador, error) {
	return application.ResultadoDecisionCoberturaParaAdaptador{},
		errors.New("no debe ejecutarse")
}

func TestRutasContratacionTemporalSeConstruyenJuntas(t *testing.T) {
	t.Parallel()
	rutas, err := nuevasRutasContratacionTemporal(
		dependenciasRutasContratacionTemporalPrueba(),
	)
	if err != nil {
		t.Fatalf("construir rutas: %v", err)
	}
	esperadas := []string{
		httpinterno.RutaAltaSolicitudes,
		httpinterno.RutaPropuestaCobertura,
		httpinterno.RutaDecisionCobertura,
		httpinterno.RutaRectificacionCobertura,
	}
	if len(rutas) != len(esperadas) {
		t.Fatalf("numero de rutas = %d", len(rutas))
	}
	for indice, esperada := range esperadas {
		if rutas[indice].Ruta != esperada || rutas[indice].Manejador == nil {
			t.Fatalf("ruta %d inesperada: %#v", indice, rutas[indice])
		}
	}
	if reflect.ValueOf(rutas[0].Manejador).Pointer() ==
		reflect.ValueOf(rutas[1].Manejador).Pointer() {
		t.Fatal("alta y cobertura comparten manejador")
	}
	for indice := 2; indice < len(rutas); indice++ {
		if reflect.ValueOf(rutas[indice].Manejador).Pointer() !=
			reflect.ValueOf(rutas[1].Manejador).Pointer() {
			t.Fatal("las rutas de cobertura no comparten el adaptador cerrado")
		}
	}
}

func TestRutasContratacionTemporalFallanSinConjuntoCompleto(t *testing.T) {
	t.Parallel()
	casos := []struct {
		nombre   string
		eliminar func(*dependenciasRutasContratacionTemporal)
	}{
		{"autoridad de alta", func(d *dependenciasRutasContratacionTemporal) {
			d.autoridadAlta = nil
		}},
		{"ejecutor de alta", func(d *dependenciasRutasContratacionTemporal) {
			d.ejecutorAlta = nil
		}},
		{"reloj", func(d *dependenciasRutasContratacionTemporal) {
			d.reloj = nil
		}},
		{"autoridad de cobertura", func(d *dependenciasRutasContratacionTemporal) {
			d.autoridadCobertura = nil
		}},
		{"presentador", func(d *dependenciasRutasContratacionTemporal) {
			d.presentador = nil
		}},
		{"decisor", func(d *dependenciasRutasContratacionTemporal) {
			d.decisor = nil
		}},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			dependencias := dependenciasRutasContratacionTemporalPrueba()
			caso.eliminar(&dependencias)
			rutas, err := nuevasRutasContratacionTemporal(dependencias)
			if rutas != nil ||
				!errors.Is(err, ErrRutasContratacionTemporalInvalidas) {
				t.Fatalf("resultado = (%#v, %v)", rutas, err)
			}
		})
	}
}

func TestRutasContratacionTemporalRechazanNuloTipado(t *testing.T) {
	t.Parallel()
	var autoridad *autoridadAltaComposicionPrueba
	dependencias := dependenciasRutasContratacionTemporalPrueba()
	dependencias.autoridadAlta = autoridad
	rutas, err := nuevasRutasContratacionTemporal(dependencias)
	if rutas != nil || !errors.Is(err, ErrRutasContratacionTemporalInvalidas) {
		t.Fatalf("resultado = (%#v, %v)", rutas, err)
	}
}

func dependenciasRutasContratacionTemporalPrueba() dependenciasRutasContratacionTemporal {
	return dependenciasRutasContratacionTemporal{
		autoridadAlta:      autoridadAltaComposicionPrueba{},
		ejecutorAlta:       ejecutorAltaComposicionPrueba{},
		reloj:              relojComposicionPrueba{},
		autoridadCobertura: autoridadCoberturaComposicionPrueba{},
		presentador:        presentadorCoberturaComposicionPrueba{},
		decisor:            decisorCoberturaComposicionPrueba{},
	}
}
