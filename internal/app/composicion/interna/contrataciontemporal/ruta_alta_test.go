package contrataciontemporal

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync/atomic"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/httpapi"
	"vec-diputacion-granada/internal/vec/adapters/memory"
	vecapp "vec-diputacion-granada/internal/vec/application"
)

const rutaAltaLiteralPrueba = "/api/vec/contratacion-temporal/solicitudes"

type manejadorRutaAltaEspiaPrueba struct {
	siguiente http.Handler
	llamadas  atomic.Int64
}

func (m *manejadorRutaAltaEspiaPrueba) ServeHTTP(
	respuesta http.ResponseWriter,
	peticion *http.Request,
) {
	m.llamadas.Add(1)
	m.siguiente.ServeHTTP(respuesta, peticion)
}

func TestNuevaRutaAltaConstruyeUnaDeclaracionExactaSinEstadoCompartido(
	t *testing.T,
) {
	t.Parallel()
	ruta, err := NuevaRutaAlta(
		autoridadAltaComposicionPrueba{},
		ejecutorAltaComposicionPrueba{},
		relojComposicionPrueba{},
	)
	if err != nil {
		t.Fatalf("construir ruta de alta: %v", err)
	}
	otra, err := NuevaRutaAlta(
		autoridadAltaComposicionPrueba{},
		ejecutorAltaComposicionPrueba{},
		relojComposicionPrueba{},
	)
	if err != nil {
		t.Fatalf("construir segunda ruta de alta: %v", err)
	}
	if ruta.Ruta != rutaAltaLiteralPrueba ||
		httpinterno.RutaAltaSolicitudes != rutaAltaLiteralPrueba ||
		ruta.Manejador == nil {
		t.Fatalf("declaracion inesperada: %#v", ruta)
	}
	if _, esMux := ruta.Manejador.(*http.ServeMux); esMux {
		t.Fatal("la ruta de alta creo una segunda autoridad de despacho")
	}
	identidad := reflect.ValueOf(ruta.Manejador).Pointer()
	identidadOtra := reflect.ValueOf(otra.Manejador).Pointer()
	if identidad == 0 || identidadOtra == 0 || identidad == identidadOtra {
		t.Fatalf(
			"los constructores compartieron manejador: %x/%x",
			identidad,
			identidadOtra,
		)
	}

	peticionGET := httptest.NewRequest(
		http.MethodGet,
		rutaAltaLiteralPrueba,
		nil,
	)
	respuestaGET := httptest.NewRecorder()
	ruta.Manejador.ServeHTTP(respuestaGET, peticionGET)
	if respuestaGET.Code != http.StatusMethodNotAllowed ||
		respuestaGET.Header().Get("Allow") != http.MethodPost {
		t.Fatalf(
			"metodo no cerrado: estado=%d allow=%q",
			respuestaGET.Code,
			respuestaGET.Header().Get("Allow"),
		)
	}

	peticionRutaDistinta := httptest.NewRequest(
		http.MethodPost,
		rutaAltaLiteralPrueba+"/",
		nil,
	)
	respuestaRutaDistinta := httptest.NewRecorder()
	ruta.Manejador.ServeHTTP(respuestaRutaDistinta, peticionRutaDistinta)
	if respuestaRutaDistinta.Code != http.StatusNotFound {
		t.Fatalf(
			"ruta no exacta aceptada: estado=%d",
			respuestaRutaDistinta.Code,
		)
	}
}

func TestNuevaRutaAltaFallaCerradoConNilYNilTipado(t *testing.T) {
	t.Parallel()
	autoridad := autoridadAltaComposicionPrueba{}
	ejecutor := ejecutorAltaComposicionPrueba{}
	reloj := relojComposicionPrueba{}
	var autoridadNula *autoridadAltaComposicionPrueba
	var ejecutorNulo *ejecutorAltaComposicionPrueba
	var relojNulo *relojComposicionPrueba

	casos := []struct {
		nombre    string
		autoridad httpinterno.AutoridadContextoCanal
		ejecutor  httpinterno.EjecutorAlta
		reloj     ports.Reloj
	}{
		{"sin autoridad", nil, ejecutor, reloj},
		{"sin ejecutor", autoridad, nil, reloj},
		{"sin reloj", autoridad, ejecutor, nil},
		{"autoridad nula tipada", autoridadNula, ejecutor, reloj},
		{"ejecutor nulo tipado", autoridad, ejecutorNulo, reloj},
		{"reloj nulo tipado", autoridad, ejecutor, relojNulo},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			ruta, err := NuevaRutaAlta(
				caso.autoridad,
				caso.ejecutor,
				caso.reloj,
			)
			if !reflect.DeepEqual(ruta, httpapi.RutaExacta{}) ||
				!errors.Is(err, ErrRutasContratacionTemporalInvalidas) ||
				errors.Is(err, httpinterno.ErrManejadorAltaInvalido) {
				t.Fatalf("resultado no opaco: ruta=%#v error=%v", ruta, err)
			}
		})
	}
}

func TestRutaAltaEsCompatibleConDispatcherExactoYDeniegaAntesDelManejador(
	t *testing.T,
) {
	t.Parallel()
	negocio := &negocioContratacionNoInvocablePrueba{}
	ruta, err := NuevaRutaAlta(
		autoridadAltaComposicionPrueba{},
		negocio,
		relojComposicionPrueba{},
	)
	if err != nil {
		t.Fatalf("construir ruta de alta: %v", err)
	}
	espia := &manejadorRutaAltaEspiaPrueba{siguiente: ruta.Manejador}
	ruta.Manejador = espia

	almacen := memory.NewStore()
	servicio, err := vecapp.NewService(almacen, almacen, almacen)
	if err != nil {
		t.Fatalf("construir servicio VEC: %v", err)
	}
	autoridadExterior := &autoridadDespachoContratacionEspiaPrueba{
		err: httpapi.ErrAccesoRutaExactaDenegado,
	}
	dispatcher, err := httpapi.NewHandlerWithOptions(
		servicio,
		httpapi.HandlerOptions{
			RutasExactas:          []httpapi.RutaExacta{ruta},
			AutoridadRutasExactas: autoridadExterior,
		},
	)
	if err != nil {
		t.Fatalf("construir dispatcher exacto: %v", err)
	}

	respuesta := httptest.NewRecorder()
	dispatcher.ServeHTTP(
		respuesta,
		nuevaPeticionContratacionErrorPrueba(rutaAltaLiteralPrueba),
	)
	if respuesta.Code != http.StatusForbidden {
		t.Fatalf(
			"denegacion exterior: estado=%d cuerpo=%s",
			respuesta.Code,
			respuesta.Body.String(),
		)
	}
	if llamadas, recibida := autoridadExterior.estado(); llamadas != 1 || recibida != rutaAltaLiteralPrueba {
		t.Fatalf("autoridad exterior = (%d, %q)", llamadas, recibida)
	}
	if espia.llamadas.Load() != 0 || negocio.altas.Load() != 0 {
		t.Fatalf(
			"la denegacion invoco dependencias: manejador=%d negocio=%d",
			espia.llamadas.Load(),
			negocio.altas.Load(),
		)
	}
}
