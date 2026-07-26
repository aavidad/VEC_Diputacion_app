package httpinterno

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
)

const (
	expedienteResultadoCoberturaPrueba = "expediente:ct:0001"
	claveResultadoCoberturaPrueba      = "" +
		"4d36e96e-e325-4f9b-bebc-291d91d6f732"
)

type consultorResultadoCoberturaPrueba struct {
	mu          sync.Mutex
	resultado   application.DatosConsultaResultadoCoberturaParaAdaptador
	err         error
	alConsultar func(context.Context)
	solicitudes []application.SolicitudConsultaResultadoCobertura
}

func (c *consultorResultadoCoberturaPrueba) ConsultarParaAdaptador(
	ctx context.Context,
	solicitud application.SolicitudConsultaResultadoCobertura,
) (application.DatosConsultaResultadoCoberturaParaAdaptador, error) {
	if c.alConsultar != nil {
		c.alConsultar(ctx)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.solicitudes = append(c.solicitudes, solicitud)
	return c.resultado, c.err
}

func (c *consultorResultadoCoberturaPrueba) total() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.solicitudes)
}

func (c *consultorResultadoCoberturaPrueba) ultima() (
	application.SolicitudConsultaResultadoCobertura,
	bool,
) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.solicitudes) == 0 {
		return application.SolicitudConsultaResultadoCobertura{}, false
	}
	return c.solicitudes[len(c.solicitudes)-1], true
}

func cuerpoResultadoCoberturaPrueba() string {
	return `{"expediente_ref":"` + expedienteResultadoCoberturaPrueba +
		`","clave_idempotencia":"` + claveResultadoCoberturaPrueba + `"}`
}

func nuevaPeticionResultadoCoberturaPrueba() *http.Request {
	peticion := httptest.NewRequest(
		http.MethodPost,
		RutaResultadoCobertura,
		bytes.NewBufferString(cuerpoResultadoCoberturaPrueba()),
	)
	peticion.Header.Set("Content-Type", "application/json; charset=utf-8")
	peticion.Header.Set("Accept", "application/json")
	return peticion
}

func reciboResultadoCoberturaPrueba(
	estado string,
) *application.DatosReciboDecisionCoberturaParaAdaptador {
	recibo := &application.DatosReciboDecisionCoberturaParaAdaptador{
		ReciboRef:    "recibo:ct:cobertura:0001",
		Estado:       estado,
		ConfirmadaEn: time.Date(2026, 7, 26, 9, 16, 0, 123000000, time.UTC),
	}
	if estado == "aplicada" {
		recibo.DecisionCoberturaRef = "decision-cobertura:confirmada:0001"
		recibo.VersionResultante = 2
	}
	return recibo
}

func TestManejadorResultadoCoberturaPublicaConfirmadoAplicadoExacto(
	t *testing.T,
) {
	t.Parallel()
	consultor := &consultorResultadoCoberturaPrueba{
		resultado: application.DatosConsultaResultadoCoberturaParaAdaptador{
			Estado: application.ResultadoCoberturaConfirmado,
			Recibo: reciboResultadoCoberturaPrueba("aplicada"),
		},
	}
	manejador, err := NuevoManejadorResultadoCobertura(consultor)
	if err != nil {
		t.Fatal(err)
	}
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, nuevaPeticionResultadoCoberturaPrueba())
	if respuesta.Code != http.StatusOK {
		t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body.String())
	}
	esperado := `{"data":{"esquema":"` +
		EsquemaResultadoConsultaCobertura +
		`","estado":"confirmado","recibo":{"esquema":` +
		`"vec.contratacion-temporal.recibo-cobertura.v1",` +
		`"recibo_ref":"recibo:ct:cobertura:0001","estado":"aplicada",` +
		`"decision_cobertura_ref":"decision-cobertura:confirmada:0001",` +
		`"version_resultante":2,` +
		`"confirmada_en":"2026-07-26T09:16:00.123Z"}}}`
	if respuesta.Body.String() != esperado {
		t.Fatalf("respuesta inesperada:\n%s\n!=\n%s", respuesta.Body, esperado)
	}
	solicitud, existe := consultor.ultima()
	if !existe ||
		solicitud.ExpedienteRef != expedienteResultadoCoberturaPrueba ||
		solicitud.ClaveIdempotencia != claveResultadoCoberturaPrueba {
		t.Fatalf("solicitud no mínima: %#v", solicitud)
	}
}

func TestManejadorResultadoCoberturaPublicaConfirmadoDenegadoMinimo(
	t *testing.T,
) {
	t.Parallel()
	consultor := &consultorResultadoCoberturaPrueba{
		resultado: application.DatosConsultaResultadoCoberturaParaAdaptador{
			Estado: application.ResultadoCoberturaConfirmado,
			Recibo: reciboResultadoCoberturaPrueba("denegada"),
		},
	}
	manejador, err := NuevoManejadorResultadoCobertura(consultor)
	if err != nil {
		t.Fatal(err)
	}
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, nuevaPeticionResultadoCoberturaPrueba())
	if respuesta.Code != http.StatusOK {
		t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body.String())
	}
	var envoltorio map[string]map[string]any
	if err := json.Unmarshal(respuesta.Body.Bytes(), &envoltorio); err != nil {
		t.Fatal(err)
	}
	recibo, ok := envoltorio["data"]["recibo"].(map[string]any)
	if !ok || len(recibo) != 4 || recibo["estado"] != "denegada" {
		t.Fatalf("recibo no mínimo: %#v", envoltorio)
	}
	for _, prohibido := range []string{
		"decision_cobertura_ref",
		"version_resultante",
	} {
		if _, existe := recibo[prohibido]; existe {
			t.Fatalf("campo %q publicado", prohibido)
		}
	}
}

func TestManejadorResultadoCoberturaPublicaNoObservableExacto(
	t *testing.T,
) {
	t.Parallel()
	consultor := &consultorResultadoCoberturaPrueba{
		resultado: application.DatosConsultaResultadoCoberturaParaAdaptador{
			Estado: application.ResultadoCoberturaNoObservable,
		},
	}
	manejador, err := NuevoManejadorResultadoCobertura(consultor)
	if err != nil {
		t.Fatal(err)
	}
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, nuevaPeticionResultadoCoberturaPrueba())
	if respuesta.Code != http.StatusAccepted {
		t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body.String())
	}
	esperado := `{"data":{"esquema":"` +
		EsquemaResultadoConsultaCobertura +
		`","estado":"no_observable"}}`
	if respuesta.Body.String() != esperado {
		t.Fatalf("respuesta inesperada: %s", respuesta.Body)
	}
}

func TestManejadorResultadoCoberturaClasificaErroresSinFiltrarlos(
	t *testing.T,
) {
	t.Parallel()
	casos := []struct {
		nombre string
		err    error
		estado int
		codigo string
	}{
		{
			"denegado",
			application.ErrConsultaResultadoCoberturaDenegada,
			http.StatusForbidden,
			"acceso_denegado",
		},
		{
			"conflicto",
			application.ErrConsultaResultadoCoberturaConflicto,
			http.StatusConflict,
			"conflicto",
		},
		{
			"no confiable",
			application.ErrConsultaResultadoCoberturaNoConfiable,
			http.StatusServiceUnavailable,
			"servicio_no_disponible",
		},
		{
			"no disponible",
			application.ErrConsultaResultadoCoberturaNoDisponible,
			http.StatusServiceUnavailable,
			"servicio_no_disponible",
		},
		{
			"cancelado",
			context.Canceled,
			http.StatusServiceUnavailable,
			"servicio_no_disponible",
		},
		{
			"plazo",
			context.DeadlineExceeded,
			http.StatusServiceUnavailable,
			"servicio_no_disponible",
		},
		{
			"desconocido",
			errors.New("postgres://usuario:secreto@interno/bd"),
			http.StatusServiceUnavailable,
			"servicio_no_disponible",
		},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			consultor := &consultorResultadoCoberturaPrueba{err: caso.err}
			manejador, err := NuevoManejadorResultadoCobertura(consultor)
			if err != nil {
				t.Fatal(err)
			}
			respuesta := httptest.NewRecorder()
			manejador.ServeHTTP(
				respuesta,
				nuevaPeticionResultadoCoberturaPrueba(),
			)
			if respuesta.Code != caso.estado {
				t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body)
			}
			var envoltorio struct {
				Error struct {
					Codigo string `json:"codigo"`
				} `json:"error"`
			}
			if err := json.Unmarshal(respuesta.Body.Bytes(), &envoltorio); err != nil {
				t.Fatal(err)
			}
			if envoltorio.Error.Codigo != caso.codigo {
				t.Fatalf("error público=%#v", envoltorio)
			}
			if bytes.Contains(respuesta.Body.Bytes(), []byte("postgres")) ||
				bytes.Contains(respuesta.Body.Bytes(), []byte("secreto")) {
				t.Fatalf("causa filtrada: %s", respuesta.Body)
			}
		})
	}
}

func TestManejadorResultadoCoberturaRechazaUnionNoNominal(
	t *testing.T,
) {
	t.Parallel()
	casos := []struct {
		nombre    string
		resultado application.DatosConsultaResultadoCoberturaParaAdaptador
	}{
		{"cero", application.DatosConsultaResultadoCoberturaParaAdaptador{}},
		{
			"confirmado sin recibo",
			application.DatosConsultaResultadoCoberturaParaAdaptador{
				Estado: application.ResultadoCoberturaConfirmado,
			},
		},
		{
			"ausente con recibo",
			application.DatosConsultaResultadoCoberturaParaAdaptador{
				Estado: application.ResultadoCoberturaNoObservable,
				Recibo: reciboResultadoCoberturaPrueba("aplicada"),
			},
		},
		{
			"estado desconocido",
			application.DatosConsultaResultadoCoberturaParaAdaptador{
				Estado: "inventado",
			},
		},
		{
			"aplicado sin versión",
			application.DatosConsultaResultadoCoberturaParaAdaptador{
				Estado: application.ResultadoCoberturaConfirmado,
				Recibo: func() *application.
					DatosReciboDecisionCoberturaParaAdaptador {
					recibo := reciboResultadoCoberturaPrueba("aplicada")
					recibo.VersionResultante = 0
					return recibo
				}(),
			},
		},
		{
			"denegado con decisión",
			application.DatosConsultaResultadoCoberturaParaAdaptador{
				Estado: application.ResultadoCoberturaConfirmado,
				Recibo: func() *application.
					DatosReciboDecisionCoberturaParaAdaptador {
					recibo := reciboResultadoCoberturaPrueba("denegada")
					recibo.DecisionCoberturaRef = "decision:prohibida:001"
					return recibo
				}(),
			},
		},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			consultor := &consultorResultadoCoberturaPrueba{
				resultado: caso.resultado,
			}
			manejador, err := NuevoManejadorResultadoCobertura(consultor)
			if err != nil {
				t.Fatal(err)
			}
			respuesta := httptest.NewRecorder()
			manejador.ServeHTTP(
				respuesta,
				nuevaPeticionResultadoCoberturaPrueba(),
			)
			if respuesta.Code != http.StatusServiceUnavailable {
				t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body)
			}
		})
	}
}

func TestManejadorResultadoCoberturaCancelacionConcurrentePrevalece(
	t *testing.T,
) {
	t.Parallel()
	resultados := []struct {
		nombre    string
		resultado application.DatosConsultaResultadoCoberturaParaAdaptador
	}{
		{
			"confirmado",
			application.DatosConsultaResultadoCoberturaParaAdaptador{
				Estado: application.ResultadoCoberturaConfirmado,
				Recibo: reciboResultadoCoberturaPrueba("aplicada"),
			},
		},
		{
			"no observable",
			application.DatosConsultaResultadoCoberturaParaAdaptador{
				Estado: application.ResultadoCoberturaNoObservable,
			},
		},
	}
	causas := []struct {
		nombre   string
		contexto func() (
			context.Context,
			context.CancelFunc,
			func(context.Context),
		)
	}{
		{
			"cancelacion",
			func() (
				context.Context,
				context.CancelFunc,
				func(context.Context),
			) {
				ctx, cancelar := context.WithCancel(context.Background())
				return ctx, cancelar, func(context.Context) {
					cancelar()
				}
			},
		},
		{
			"plazo",
			func() (
				context.Context,
				context.CancelFunc,
				func(context.Context),
			) {
				ctx, cancelar := context.WithTimeout(
					context.Background(),
					100*time.Millisecond,
				)
				return ctx, cancelar, func(contexto context.Context) {
					<-contexto.Done()
				}
			},
		},
	}
	for _, causa := range causas {
		causa := causa
		for _, rama := range resultados {
			rama := rama
			t.Run(causa.nombre+"/"+rama.nombre, func(t *testing.T) {
				t.Parallel()
				ctx, cancelar, duranteConsulta := causa.contexto()
				defer cancelar()
				consultor := &consultorResultadoCoberturaPrueba{
					resultado:   rama.resultado,
					alConsultar: duranteConsulta,
				}
				manejador, err := NuevoManejadorResultadoCobertura(consultor)
				if err != nil {
					t.Fatal(err)
				}
				peticion := nuevaPeticionResultadoCoberturaPrueba().
					WithContext(ctx)
				respuesta := httptest.NewRecorder()
				manejador.ServeHTTP(respuesta, peticion)
				if respuesta.Code != http.StatusServiceUnavailable {
					t.Fatalf(
						"estado=%d cuerpo=%s",
						respuesta.Code,
						respuesta.Body,
					)
				}
				for _, funcional := range []string{
					"confirmado",
					"no_observable",
					"recibo:ct:cobertura",
				} {
					if bytes.Contains(
						respuesta.Body.Bytes(),
						[]byte(funcional),
					) {
						t.Fatalf(
							"respuesta funcional tras %s: %s",
							causa.nombre,
							respuesta.Body,
						)
					}
				}
				if consultor.total() != 1 {
					t.Fatalf("consultas=%d", consultor.total())
				}
			})
		}
	}
}

func TestManejadorResultadoCoberturaTienePuertoYDependenciaMinimos(
	t *testing.T,
) {
	t.Parallel()
	var nulo *consultorResultadoCoberturaPrueba
	if _, err := NuevoManejadorResultadoCobertura(nil); !errors.Is(
		err,
		ErrManejadorResultadoCoberturaInvalido,
	) {
		t.Fatalf("nulo aceptado: %v", err)
	}
	if _, err := NuevoManejadorResultadoCobertura(nulo); !errors.Is(
		err,
		ErrManejadorResultadoCoberturaInvalido,
	) {
		t.Fatalf("nulo tipado aceptado: %v", err)
	}
	tipoPuerto := reflect.TypeOf((*ConsultorResultadoCobertura)(nil)).Elem()
	if tipoPuerto.NumMethod() != 1 ||
		tipoPuerto.Method(0).Name != "ConsultarParaAdaptador" {
		t.Fatalf("puerto demasiado amplio: %v", tipoPuerto)
	}
	tipoSolicitud := reflect.TypeOf(
		application.SolicitudConsultaResultadoCobertura{},
	)
	if tipoSolicitud.NumField() != 2 ||
		tipoSolicitud.Field(0).Name != "ClaveIdempotencia" ||
		tipoSolicitud.Field(1).Name != "ExpedienteRef" {
		t.Fatalf("solicitud acepta autoridad: %v", tipoSolicitud)
	}
}
