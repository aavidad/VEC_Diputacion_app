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

type consultorCuadroRRHHComposicionPrueba struct {
	consultas        int
	antesDeConsultar func()
}

func (c *consultorCuadroRRHHComposicionPrueba) Consultar(
	context.Context,
	ports.SolicitudCuadroRRHH,
) (ports.PaginaCuadroRRHH, error) {
	c.consultas++
	if c.antesDeConsultar != nil {
		c.antesDeConsultar()
	}
	return ports.PaginaCuadroRRHH{}, application.ErrConsultaRRHHNoDisponible
}

type consultorDetalleRRHHComposicionPrueba struct {
	consultas        int
	antesDeConsultar func()
}

func (c *consultorDetalleRRHHComposicionPrueba) Consultar(
	context.Context,
	ports.SolicitudDetalleRRHH,
) (ports.DetalleExpedienteRRHH, error) {
	c.consultas++
	if c.antesDeConsultar != nil {
		c.antesDeConsultar()
	}
	return ports.DetalleExpedienteRRHH{}, application.ErrConsultaRRHHNoDisponible
}

type ejecutorSeleccionComposicionPrueba struct {
	ejecuciones int
}

func (e *ejecutorSeleccionComposicionPrueba) SeleccionarYLlamarParaAdaptador(
	context.Context,
	application.SolicitudSeleccionLlamamiento,
) (application.DatosReciboSeleccionLlamamientoParaAdaptador, error) {
	e.ejecuciones++
	return application.DatosReciboSeleccionLlamamientoParaAdaptador{
		ReciboRef: "recibo:llamamiento:http:001",
		ConfirmadaEn: time.Date(
			2026, 8, 31, 10, 0, 0, 123000000, time.UTC,
		),
	}, nil
}

type autoridadPropuestaFormalizacionComposicionPrueba struct {
	resoluciones int
}

func (a *autoridadPropuestaFormalizacionComposicionPrueba) ResolverContextoPropuestaFormalizacion(
	context.Context,
) (httpinterno.ContextoServidorPropuestaFormalizacion, error) {
	a.resoluciones++
	return httpinterno.ContextoServidorPropuestaFormalizacion{
		OrganizacionRef: "organizacion:composicion-formalizacion",
	}, nil
}

type ejecutorPropuestaFormalizacionComposicionPrueba struct {
	ejecuciones int
}

func (e *ejecutorPropuestaFormalizacionComposicionPrueba) PrepararYConfirmar(
	_ context.Context,
	solicitud ports.SolicitudPropuestaFormalizacion,
) (ports.ResultadoPropuestaFormalizacion, error) {
	e.ejecuciones++
	return ports.ResultadoPropuestaFormalizacion{
		Solicitud:         solicitud.Clonar(),
		PropuestaRef:      "propuesta:composicion-formalizacion",
		ReciboLocalRef:    "recibo:composicion-formalizacion",
		AuditoriaRef:      "auditoria:composicion-formalizacion",
		VersionResultante: solicitud.VersionEsperada + 1,
		ConfirmadaEn: time.Date(
			2026, 8, 31, 16, 0, 0, 123000000, time.UTC,
		),
		Estado: ports.ResultadoPropuestaFormalizacionConfirmado,
	}, nil
}

type autoridadCierreAdministrativoComposicionPrueba struct {
	organizacionRef string
	resoluciones    int
}

func (a *autoridadCierreAdministrativoComposicionPrueba) ResolverOrganizacionCierreAdministrativo(
	context.Context,
) (string, error) {
	a.resoluciones++
	return a.organizacionRef, nil
}

type ejecutorCierreAdministrativoComposicionPrueba struct {
	cierres     int
	reaperturas int
}

func (e *ejecutorCierreAdministrativoComposicionPrueba) Cerrar(
	_ context.Context,
	solicitud application.SolicitudCerrarAdministrativamente,
) (ports.ResultadoCierreAdministrativo, error) {
	e.cierres++
	return resultadoCierreAdministrativoComposicionPrueba(
		solicitudPuertoCierreAdministrativoComposicionPrueba(solicitud),
	)
}

func (e *ejecutorCierreAdministrativoComposicionPrueba) ReabrirExcepcionalmente(
	_ context.Context,
	solicitud application.SolicitudReabrirExcepcionalmente,
) (ports.ResultadoCierreAdministrativo, error) {
	e.reaperturas++
	return resultadoCierreAdministrativoComposicionPrueba(
		solicitudPuertoReaperturaAdministrativaComposicionPrueba(solicitud),
	)
}

func TestNuevasRutasContratacionTemporalSeConstruyenJuntas(t *testing.T) {
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
		httpinterno.RutaPropuestaFormalizacion,
		httpinterno.RutaCerrarAdministrativamente,
		httpinterno.RutaReabrirExcepcionalmente,
		httpinterno.RutaPreparacionesInformeJuridico,
		httpinterno.RutaConsultaCuadroRRHH,
		httpinterno.RutaConsultaDetalleRRHH,
		httpinterno.RutaAsignaciones,
		httpinterno.RutaReasignaciones,
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
	if reflect.ValueOf(rutas[8].Manejador).Pointer() ==
		reflect.ValueOf(rutas[7].Manejador).Pointer() {
		t.Fatal("propuesta de formalizacion y seleccion comparten manejador")
	}
	if reflect.ValueOf(rutas[9].Manejador).Pointer() !=
		reflect.ValueOf(rutas[10].Manejador).Pointer() {
		t.Fatal("cierre y reapertura no comparten un unico manejador")
	}
	if reflect.ValueOf(rutas[9].Manejador).Pointer() ==
		reflect.ValueOf(rutas[8].Manejador).Pointer() {
		t.Fatal("cierre administrativo comparte un manejador previo")
	}
	if reflect.ValueOf(rutas[11].Manejador).Pointer() ==
		reflect.ValueOf(rutas[12].Manejador).Pointer() {
		t.Fatal("cuadro y detalle RRHH comparten manejador")
	}
	if reflect.ValueOf(rutas[11].Manejador).Pointer() ==
		reflect.ValueOf(rutas[10].Manejador).Pointer() {
		t.Fatal("las consultas RRHH comparten el manejador anterior")
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

func TestNuevasRutasContratacionTemporalFallanSinConjuntoCompleto(t *testing.T) {
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
		{"consultor de cuadro RRHH", func(d *DependenciasRutas) {
			d.ConsultorCuadroRRHH = nil
		}},
		{"consultor de detalle RRHH", func(d *DependenciasRutas) {
			d.ConsultorDetalleRRHH = nil
		}},
		{"ejecutor de seleccion", func(d *DependenciasRutas) {
			d.EjecutorSeleccion = nil
		}},
		{"autoridad de propuesta", func(d *DependenciasRutas) {
			d.AutoridadPropuestaFormalizacion = nil
		}},
		{"ejecutor de propuesta", func(d *DependenciasRutas) {
			d.EjecutorPropuestaFormalizacion = nil
		}},
		{"autoridad de cierre", func(d *DependenciasRutas) {
			d.AutoridadCierreAdministrativo = nil
		}},
		{"ejecutor de cierre", func(d *DependenciasRutas) {
			d.EjecutorCierreAdministrativo = nil
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

func TestNuevasRutasContratacionTemporalRechazanNuloTipado(t *testing.T) {
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
		{"consultor de cuadro RRHH", func(d *DependenciasRutas) {
			var nulo *consultorCuadroRRHHComposicionPrueba
			d.ConsultorCuadroRRHH = nulo
		}},
		{"consultor de detalle RRHH", func(d *DependenciasRutas) {
			var nulo *consultorDetalleRRHHComposicionPrueba
			d.ConsultorDetalleRRHH = nulo
		}},
		{"ejecutor de seleccion", func(d *DependenciasRutas) {
			var nulo *ejecutorSeleccionComposicionPrueba
			d.EjecutorSeleccion = nulo
		}},
		{"autoridad de propuesta", func(d *DependenciasRutas) {
			var nulo *autoridadPropuestaFormalizacionComposicionPrueba
			d.AutoridadPropuestaFormalizacion = nulo
		}},
		{"ejecutor de propuesta", func(d *DependenciasRutas) {
			var nulo *ejecutorPropuestaFormalizacionComposicionPrueba
			d.EjecutorPropuestaFormalizacion = nulo
		}},
		{"autoridad de cierre", func(d *DependenciasRutas) {
			var nulo *autoridadCierreAdministrativoComposicionPrueba
			d.AutoridadCierreAdministrativo = nulo
		}},
		{"ejecutor de cierre", func(d *DependenciasRutas) {
			var nulo *ejecutorCierreAdministrativoComposicionPrueba
			d.EjecutorCierreAdministrativo = nulo
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

func TestRutasContratacionTemporalNoDevuelvenParcialSiFallaConstructorPropuesta(
	t *testing.T,
) {
	t.Parallel()
	dependencias := dependenciasRutasPrueba()
	dependencias.EjecutorPropuestaFormalizacion = nil
	rutas, err := NuevasRutas(dependencias)
	if rutas != nil || !errors.Is(err, ErrRutasContratacionTemporalInvalidas) {
		t.Fatalf("resultado = (%#v, %v)", rutas, err)
	}
}

func TestNuevasRutasNoDevuelvenParcialSiFallaConstructorConsultasRRHH(
	t *testing.T,
) {
	t.Parallel()
	casos := []struct {
		nombre   string
		eliminar func(*DependenciasRutas)
	}{
		{"cuadro", func(d *DependenciasRutas) { d.ConsultorCuadroRRHH = nil }},
		{"detalle", func(d *DependenciasRutas) { d.ConsultorDetalleRRHH = nil }},
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

func TestNuevasRutasNoDevuelvenParcialSiFallaConstructorCierreAdministrativo(
	t *testing.T,
) {
	t.Parallel()
	dependencias := dependenciasRutasPrueba()
	dependencias.EjecutorCierreAdministrativo = nil
	rutas, err := NuevasRutas(dependencias)
	if rutas != nil || !errors.Is(err, ErrRutasContratacionTemporalInvalidas) {
		t.Fatalf("resultado = (%#v, %v)", rutas, err)
	}
}

func nuevaPeticionPropuestaFormalizacionComposicionPrueba() *http.Request {
	cuerpo := `{"clave_idempotencia":"938f47a6-5d2b-4c10-aa11-1234567890ab",` +
		`"expediente_ref":"expediente:http-formalizacion",` +
		`"llamamiento_ref":"llamamiento:http-formalizacion",` +
		`"resolucion_llamamiento_aceptada_ref":"resolucion:http-aceptada",` +
		`"recibo_resolucion_aceptada_ref":"recibo:http-aceptacion",` +
		`"version_esperada":13,` +
		`"tipo_formalizacion":{"referencia":"tipo:http-formalizacion",` +
		`"version":7,"huella_sha256":"` + strings.Repeat("1", 64) + `"},` +
		`"plantilla":{"referencia":"plantilla:http-formalizacion",` +
		`"version":7,"huella_sha256":"` + strings.Repeat("2", 64) + `"},` +
		`"anexos":[],` +
		`"politica_firma":{"referencia":"politica:http-firma",` +
		`"version":7,"huella_sha256":"` + strings.Repeat("3", 64) + `"},` +
		`"plan_firma":{"referencia":"plan:http-firma",` +
		`"version":7,"huella_sha256":"` + strings.Repeat("4", 64) + `"}}`
	peticion := httptest.NewRequest(
		http.MethodPost,
		httpinterno.RutaPropuestaFormalizacion,
		strings.NewReader(cuerpo),
	)
	peticion.Header.Set("Content-Type", "application/json; charset=utf-8")
	peticion.Header.Set("Accept", "application/json")
	return peticion
}

func nuevaPeticionCierreAdministrativoComposicionPrueba(
	ruta string,
) *http.Request {
	cuerpo := `{"expediente_ref":"` +
		referenciaCierreAdministrativoComposicionPrueba("b") + `",` +
		`"seguimiento_ref":"` +
		referenciaCierreAdministrativoComposicionPrueba("c") + `",` +
		`"version_esperada":7,` +
		`"clave_idempotencia":"12345678-1234-4567-8abc-123456789abc",` +
		`"transicion_clave":"cierre_administrativo",` +
		`"motivo_clave":"fin_relacion_confirmado"}`
	peticion := httptest.NewRequest(
		http.MethodPost,
		ruta,
		strings.NewReader(cuerpo),
	)
	peticion.Header.Set("Content-Type", "application/json; charset=utf-8")
	peticion.Header.Set("Accept", "application/json")
	return peticion
}

func nuevaPeticionConsultaRRHHComposicionPrueba(ruta string) *http.Request {
	cuerpo := `{"filtros":{"texto":"","estado_clave":"",` +
		`"fase_clave":""},"paginacion":{"limite":1,"cursor":""}}`
	if ruta == httpinterno.RutaConsultaDetalleRRHH {
		cuerpo = `{"expediente_ref":"expediente:ct:0001",` +
			`"version_observada":1}`
	}
	peticion := httptest.NewRequest(http.MethodPost, ruta, strings.NewReader(cuerpo))
	peticion.Header.Set("Content-Type", "application/json")
	peticion.Header.Set("Accept", "application/json")
	return peticion
}

func referenciaCierreAdministrativoComposicionPrueba(digito string) string {
	return "ref:" + strings.Repeat(digito, 64)
}

func solicitudPuertoCierreAdministrativoComposicionPrueba(
	s application.SolicitudCerrarAdministrativamente,
) ports.SolicitudTransaccionCierreAdministrativo {
	return ports.SolicitudTransaccionCierreAdministrativo{
		Operacion:         ports.OperacionCerrarAdministrativamente,
		OrganizacionRef:   s.OrganizacionRef,
		ExpedienteRef:     s.ExpedienteRef,
		SeguimientoRef:    s.SeguimientoRef,
		VersionEsperada:   s.VersionEsperada,
		ClaveIdempotencia: s.ClaveIdempotencia,
		TransicionClave:   s.TransicionClave,
		MotivoClave:       s.MotivoClave,
	}
}

func solicitudPuertoReaperturaAdministrativaComposicionPrueba(
	s application.SolicitudReabrirExcepcionalmente,
) ports.SolicitudTransaccionCierreAdministrativo {
	return ports.SolicitudTransaccionCierreAdministrativo{
		Operacion:         ports.OperacionReabrirExcepcionalmente,
		OrganizacionRef:   s.OrganizacionRef,
		ExpedienteRef:     s.ExpedienteRef,
		SeguimientoRef:    s.SeguimientoRef,
		VersionEsperada:   s.VersionEsperada,
		ClaveIdempotencia: s.ClaveIdempotencia,
		TransicionClave:   s.TransicionClave,
		MotivoClave:       s.MotivoClave,
	}
}

func resultadoCierreAdministrativoComposicionPrueba(
	solicitud ports.SolicitudTransaccionCierreAdministrativo,
) (ports.ResultadoCierreAdministrativo, error) {
	return ports.NuevoResultadoCierreAdministrativo(
		ports.DatosResultadoCierreAdministrativo{
			Solicitud:         solicitud,
			VersionResultante: solicitud.VersionEsperada + 1,
			ActuacionRef: referenciaCierreAdministrativoComposicionPrueba(
				"d",
			),
			ReciboRef: referenciaCierreAdministrativoComposicionPrueba(
				"e",
			),
			ActorRef: referenciaCierreAdministrativoComposicionPrueba(
				"f",
			),
			CorrelacionRef: referenciaCierreAdministrativoComposicionPrueba(
				"1",
			),
			Estado: ports.EstadoResultadoCierreAdministrativoConfirmado,
		},
	)
}

func dependenciasRutasPrueba() DependenciasRutas {
	return DependenciasRutas{
		AutoridadAlta:                   autoridadAltaComposicionPrueba{},
		EjecutorAlta:                    ejecutorAltaComposicionPrueba{},
		Reloj:                           relojComposicionPrueba{},
		AutoridadAnalisis:               autoridadAnalisisComposicionPrueba{},
		EjecutorAnalisis:                ejecutorAnalisisComposicionPrueba{},
		AutoridadCobertura:              autoridadCoberturaComposicionPrueba{},
		Presentador:                     presentadorCoberturaComposicionPrueba{},
		Decisor:                         decisorCoberturaComposicionPrueba{},
		ConsultorResultado:              &consultorResultadoCoberturaComposicionPrueba{},
		ConsultorCuadroRRHH:             &consultorCuadroRRHHComposicionPrueba{},
		ConsultorDetalleRRHH:            &consultorDetalleRRHHComposicionPrueba{},
		EjecutorSeleccion:               &ejecutorSeleccionComposicionPrueba{},
		AutoridadPropuestaFormalizacion: &autoridadPropuestaFormalizacionComposicionPrueba{},
		EjecutorPropuestaFormalizacion:  &ejecutorPropuestaFormalizacionComposicionPrueba{},
		AutoridadCierreAdministrativo: &autoridadCierreAdministrativoComposicionPrueba{
			organizacionRef: referenciaCierreAdministrativoComposicionPrueba("a"),
		},
		EjecutorCierreAdministrativo: &ejecutorCierreAdministrativoComposicionPrueba{},
		AutoridadAsignacion:          &autoridadAsignacionComposicionPrueba{},
		EjecutorAsignacion:           &ejecutorAsignacionComposicionPrueba{},
		AutoridadInformeJuridico:     &autoridadInformeJuridicoComposicionPrueba{},
		EjecutorInformeJuridico:      &ejecutorInformeJuridicoComposicionPrueba{},
	}
}
