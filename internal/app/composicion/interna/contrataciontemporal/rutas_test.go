package contrataciontemporal

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

type autoridadAnalisisComposicionPrueba struct{}

func (autoridadAnalisisComposicionPrueba) ResolverContextoCanalAnalisisRRHH(
	context.Context,
) (httpinterno.ContextoCanalAnalisisRRHH, error) {
	return httpinterno.ContextoCanalAnalisisRRHH{}, errors.New(
		"no debe ejecutarse",
	)
}

type ejecutorAnalisisComposicionPrueba struct{}

func (ejecutorAnalisisComposicionPrueba) Registrar(
	context.Context,
	application.SolicitudRegistrarAnalisis,
) (ports.ReciboOperacionAnalisis, error) {
	return ports.ReciboOperacionAnalisis{}, errors.New("no debe ejecutarse")
}

func (ejecutorAnalisisComposicionPrueba) Rectificar(
	context.Context,
	application.SolicitudRectificarAnalisis,
) (ports.ReciboOperacionAnalisis, error) {
	return ports.ReciboOperacionAnalisis{}, errors.New("no debe ejecutarse")
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

type ejecutorSeleccionComposicionPrueba struct {
	ejecuciones int
}

func (e *ejecutorSeleccionComposicionPrueba) SeleccionarYLlamar(
	context.Context,
	application.SolicitudSeleccionLlamamiento,
) (ports.ReciboSolicitudLlamamientoBolsa, error) {
	e.ejecuciones++
	return ports.ReciboSolicitudLlamamientoBolsa{
		PropuestaGenerada: true,
		ReciboRef:         "recibo:llamamiento:http:001",
		ConfirmadaEn: time.Date(
			2026, 8, 31, 10, 0, 0, 123000000, time.UTC,
		),
	}, nil
}

func TestRutasContratacionTemporalSeConstruyenJuntas(t *testing.T) {
	t.Parallel()
	rutas, err := NuevasRutas(
		dependenciasRutasPrueba(),
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
		httpinterno.RutaRegistroAnalisisRRHH,
		httpinterno.RutaRectificacionAnalisisRRHH,
		httpinterno.RutaSeleccionLlamamiento,
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
	if reflect.ValueOf(rutas[5].Manejador).Pointer() !=
		reflect.ValueOf(rutas[6].Manejador).Pointer() {
		t.Fatal("registro y rectificacion de analisis no comparten manejador")
	}
	if reflect.ValueOf(rutas[7].Manejador).Pointer() ==
		reflect.ValueOf(rutas[6].Manejador).Pointer() {
		t.Fatal("seleccion y analisis comparten manejador")
	}
}

func TestRutaResultadoCoberturaCompuestaUsaSoloElConsultor(t *testing.T) {
	t.Parallel()
	dependencias := dependenciasRutasPrueba()
	consultor := dependencias.ConsultorResultado.(*consultorResultadoCoberturaComposicionPrueba)
	rutas, err := NuevasRutas(dependencias)
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

func TestRutaSeleccionLlamamientoCompuestaDelegaUnaVez(t *testing.T) {
	t.Parallel()
	dependencias := dependenciasRutasPrueba()
	ejecutor := dependencias.EjecutorSeleccion.(*ejecutorSeleccionComposicionPrueba)
	rutas, err := NuevasRutas(dependencias)
	if err != nil {
		t.Fatal(err)
	}
	peticion := httptest.NewRequest(
		http.MethodPost,
		httpinterno.RutaSeleccionLlamamiento,
		strings.NewReader(
			`{"clave_idempotencia":"4d36e96e-e325-4f9b-bebc-291d91d6f732"}`,
		),
	)
	peticion.Header.Set("Content-Type", "application/json; charset=utf-8")
	peticion.Header.Set("Accept", "application/json")
	respuesta := httptest.NewRecorder()
	rutas[7].Manejador.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusOK || ejecutor.ejecuciones != 1 {
		t.Fatalf(
			"seleccion compuesta: estado=%d ejecuciones=%d cuerpo=%s",
			respuesta.Code,
			ejecutor.ejecuciones,
			respuesta.Body.String(),
		)
	}
}

func TestRutasContratacionTemporalFallanSinConjuntoCompleto(t *testing.T) {
	t.Parallel()
	casos := []struct {
		nombre   string
		eliminar func(*DependenciasRutas)
	}{
		{"autoridad de alta", func(d *DependenciasRutas) {
			d.AutoridadAlta = nil
		}},
		{"ejecutor de alta", func(d *DependenciasRutas) {
			d.EjecutorAlta = nil
		}},
		{"reloj", func(d *DependenciasRutas) {
			d.Reloj = nil
		}},
		{"autoridad de analisis", func(d *DependenciasRutas) {
			d.AutoridadAnalisis = nil
		}},
		{"ejecutor de analisis", func(d *DependenciasRutas) {
			d.EjecutorAnalisis = nil
		}},
		{"autoridad de cobertura", func(d *DependenciasRutas) {
			d.AutoridadCobertura = nil
		}},
		{"presentador", func(d *DependenciasRutas) {
			d.Presentador = nil
		}},
		{"decisor", func(d *DependenciasRutas) {
			d.Decisor = nil
		}},
		{"consultor de resultado", func(d *DependenciasRutas) {
			d.ConsultorResultado = nil
		}},
		{"ejecutor de seleccion", func(d *DependenciasRutas) {
			d.EjecutorSeleccion = nil
		}},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			dependencias := dependenciasRutasPrueba()
			caso.eliminar(&dependencias)
			rutas, err := NuevasRutas(dependencias)
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
		introducir func(*DependenciasRutas)
	}{
		{"autoridad de alta", func(d *DependenciasRutas) {
			var nulo *autoridadAltaComposicionPrueba
			d.AutoridadAlta = nulo
		}},
		{"ejecutor de alta", func(d *DependenciasRutas) {
			var nulo *ejecutorAltaComposicionPrueba
			d.EjecutorAlta = nulo
		}},
		{"reloj", func(d *DependenciasRutas) {
			var nulo *relojComposicionPrueba
			d.Reloj = nulo
		}},
		{"autoridad de analisis", func(d *DependenciasRutas) {
			var nulo *autoridadAnalisisComposicionPrueba
			d.AutoridadAnalisis = nulo
		}},
		{"ejecutor de analisis", func(d *DependenciasRutas) {
			var nulo *ejecutorAnalisisComposicionPrueba
			d.EjecutorAnalisis = nulo
		}},
		{"autoridad de cobertura", func(d *DependenciasRutas) {
			var nulo *autoridadCoberturaComposicionPrueba
			d.AutoridadCobertura = nulo
		}},
		{"presentador", func(d *DependenciasRutas) {
			var nulo *presentadorCoberturaComposicionPrueba
			d.Presentador = nulo
		}},
		{"decisor", func(d *DependenciasRutas) {
			var nulo *decisorCoberturaComposicionPrueba
			d.Decisor = nulo
		}},
		{"consultor de resultado", func(d *DependenciasRutas) {
			var nulo *consultorResultadoCoberturaComposicionPrueba
			d.ConsultorResultado = nulo
		}},
		{"ejecutor de seleccion", func(d *DependenciasRutas) {
			var nulo *ejecutorSeleccionComposicionPrueba
			d.EjecutorSeleccion = nulo
		}},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			dependencias := dependenciasRutasPrueba()
			caso.introducir(&dependencias)
			rutas, err := NuevasRutas(dependencias)
			if rutas != nil ||
				!errors.Is(err, ErrRutasContratacionTemporalInvalidas) {
				t.Fatalf("resultado = (%#v, %v)", rutas, err)
			}
		})
	}
}

func dependenciasRutasPrueba() DependenciasRutas {
	return DependenciasRutas{
		AutoridadAlta:      autoridadAltaComposicionPrueba{},
		EjecutorAlta:       ejecutorAltaComposicionPrueba{},
		Reloj:              relojComposicionPrueba{},
		AutoridadAnalisis:  autoridadAnalisisComposicionPrueba{},
		EjecutorAnalisis:   ejecutorAnalisisComposicionPrueba{},
		AutoridadCobertura: autoridadCoberturaComposicionPrueba{},
		Presentador:        presentadorCoberturaComposicionPrueba{},
		Decisor:            decisorCoberturaComposicionPrueba{},
		ConsultorResultado: &consultorResultadoCoberturaComposicionPrueba{},
		EjecutorSeleccion:  &ejecutorSeleccionComposicionPrueba{},
	}
}
