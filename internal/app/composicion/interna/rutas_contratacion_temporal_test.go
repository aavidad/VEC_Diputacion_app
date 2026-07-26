package interna

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
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

type consultorResultadoCoberturaComposicionPrueba struct {
	consultas int
}

func (c *consultorResultadoCoberturaComposicionPrueba) ConsultarParaAdaptador(
	context.Context,
	application.SolicitudConsultaResultadoCobertura,
) (application.DatosConsultaResultadoCoberturaParaAdaptador, error) {
	c.consultas++
	return application.DatosConsultaResultadoCoberturaParaAdaptador{
		Estado: application.ResultadoCoberturaNoObservable,
	}, nil
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
		httpinterno.RutaResultadoCobertura,
	}
	if len(rutas) != len(esperadas) {
		t.Fatalf("numero de rutas = %d", len(rutas))
	}
	vistas := make(map[string]struct{}, len(rutas))
	for indice, esperada := range esperadas {
		if rutas[indice].Ruta != esperada || rutas[indice].Manejador == nil {
			t.Fatalf("ruta %d inesperada: %#v", indice, rutas[indice])
		}
		if _, repetida := vistas[rutas[indice].Ruta]; repetida {
			t.Fatalf("ruta repetida: %q", rutas[indice].Ruta)
		}
		vistas[rutas[indice].Ruta] = struct{}{}
	}
	if reflect.ValueOf(rutas[0].Manejador).Pointer() ==
		reflect.ValueOf(rutas[1].Manejador).Pointer() {
		t.Fatal("alta y cobertura comparten manejador")
	}
	for indice := 2; indice < 4; indice++ {
		if reflect.ValueOf(rutas[indice].Manejador).Pointer() !=
			reflect.ValueOf(rutas[1].Manejador).Pointer() {
			t.Fatal("las rutas de cobertura no comparten el adaptador cerrado")
		}
	}
	if reflect.ValueOf(rutas[4].Manejador).Pointer() ==
		reflect.ValueOf(rutas[1].Manejador).Pointer() {
		t.Fatal("la lectura comparte el manejador de efectos")
	}
}

func TestRutaResultadoCoberturaCompuestaUsaSoloElConsultor(t *testing.T) {
	t.Parallel()
	dependencias := dependenciasRutasContratacionTemporalPrueba()
	consultor := dependencias.consultorResultado.(*consultorResultadoCoberturaComposicionPrueba)
	rutas, err := nuevasRutasContratacionTemporal(dependencias)
	if err != nil {
		t.Fatal(err)
	}
	peticion := httptest.NewRequest(
		http.MethodPost,
		httpinterno.RutaResultadoCobertura,
		strings.NewReader(
			`{"expediente_ref":"expediente:ct:0001",`+
				`"clave_idempotencia":`+
				`"4d36e96e-e325-4f9b-bebc-291d91d6f732"}`,
		),
	)
	peticion.Header.Set("Content-Type", "application/json; charset=utf-8")
	peticion.Header.Set("Accept", "application/json")
	respuesta := httptest.NewRecorder()
	rutas[4].Manejador.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusAccepted || consultor.consultas != 1 {
		t.Fatalf(
			"lectura compuesta: estado=%d consultas=%d cuerpo=%s",
			respuesta.Code,
			consultor.consultas,
			respuesta.Body.String(),
		)
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
		{"consultor de resultado", func(d *dependenciasRutasContratacionTemporal) {
			d.consultorResultado = nil
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
	casos := []struct {
		nombre     string
		introducir func(*dependenciasRutasContratacionTemporal)
	}{
		{"autoridad de alta", func(d *dependenciasRutasContratacionTemporal) {
			var nulo *autoridadAltaComposicionPrueba
			d.autoridadAlta = nulo
		}},
		{"ejecutor de alta", func(d *dependenciasRutasContratacionTemporal) {
			var nulo *ejecutorAltaComposicionPrueba
			d.ejecutorAlta = nulo
		}},
		{"reloj", func(d *dependenciasRutasContratacionTemporal) {
			var nulo *relojComposicionPrueba
			d.reloj = nulo
		}},
		{"autoridad de cobertura", func(d *dependenciasRutasContratacionTemporal) {
			var nulo *autoridadCoberturaComposicionPrueba
			d.autoridadCobertura = nulo
		}},
		{"presentador", func(d *dependenciasRutasContratacionTemporal) {
			var nulo *presentadorCoberturaComposicionPrueba
			d.presentador = nulo
		}},
		{"decisor", func(d *dependenciasRutasContratacionTemporal) {
			var nulo *decisorCoberturaComposicionPrueba
			d.decisor = nulo
		}},
		{"consultor de resultado", func(d *dependenciasRutasContratacionTemporal) {
			var nulo *consultorResultadoCoberturaComposicionPrueba
			d.consultorResultado = nulo
		}},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			dependencias := dependenciasRutasContratacionTemporalPrueba()
			caso.introducir(&dependencias)
			rutas, err := nuevasRutasContratacionTemporal(dependencias)
			if rutas != nil ||
				!errors.Is(err, ErrRutasContratacionTemporalInvalidas) {
				t.Fatalf("resultado = (%#v, %v)", rutas, err)
			}
		})
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
		consultorResultado: &consultorResultadoCoberturaComposicionPrueba{},
	}
}
