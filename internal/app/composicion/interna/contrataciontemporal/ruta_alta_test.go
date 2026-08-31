package contrataciontemporal

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync/atomic"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	contratacionapp "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/httpapi"
	"vec-diputacion-granada/internal/vec/adapters/memory"
	vecapp "vec-diputacion-granada/internal/vec/application"
)

const rutaAltaLiteralPrueba = "/api/vec/contratacion-temporal/solicitudes"

type manejadorRutaAltaEspiaPrueba struct {
	siguiente http.Handler
	llamadas  atomic.Int64
	secuencia *atomic.Int64
	orden     atomic.Int64
}

func (m *manejadorRutaAltaEspiaPrueba) ServeHTTP(
	respuesta http.ResponseWriter,
	peticion *http.Request,
) {
	m.llamadas.Add(1)
	if m.secuencia != nil {
		m.orden.Store(m.secuencia.Add(1))
	}
	m.siguiente.ServeHTTP(respuesta, peticion)
}

type autoridadExteriorRutaAltaEspiaPrueba struct {
	llamadas  atomic.Int64
	ruta      atomic.Value
	secuencia *atomic.Int64
	orden     atomic.Int64
}

func (a *autoridadExteriorRutaAltaEspiaPrueba) AutorizarRutaExacta(
	_ context.Context,
	ruta string,
) error {
	a.ruta.Store(ruta)
	a.llamadas.Add(1)
	if a.secuencia != nil {
		a.orden.Store(a.secuencia.Add(1))
	}
	return nil
}

func (a *autoridadExteriorRutaAltaEspiaPrueba) estado() (int64, string) {
	ruta, _ := a.ruta.Load().(string)
	return a.llamadas.Load(), ruta
}

type fronterasRutaAltaEspiaPrueba struct {
	autoridad   atomic.Int64
	ejecuciones atomic.Int64
}

func (f *fronterasRutaAltaEspiaPrueba) ResolverContextoCanalAlta(
	context.Context,
) (contratacionapp.SolicitudRegistrarExpediente, error) {
	f.autoridad.Add(1)
	return contratacionapp.SolicitudRegistrarExpediente{}, errors.New(
		"autoridad interior no esperada",
	)
}

func (f *fronterasRutaAltaEspiaPrueba) Registrar(
	context.Context,
	contratacionapp.SolicitudRegistrarExpediente,
) (ports.ReciboAlta, error) {
	f.ejecuciones.Add(1)
	return ports.ReciboAlta{}, errors.New("ejecucion no esperada")
}

func nuevaRutaAltaObservadaPrueba(
	t *testing.T,
	secuencia *atomic.Int64,
	fronteras *fronterasRutaAltaEspiaPrueba,
) (httpapi.RutaExacta, *manejadorRutaAltaEspiaPrueba) {
	t.Helper()
	ruta, err := NuevaRutaAlta(fronteras, fronteras, relojComposicionPrueba{})
	if err != nil {
		t.Fatalf("construir ruta de alta observada: %v", err)
	}
	espia := &manejadorRutaAltaEspiaPrueba{
		siguiente: ruta.Manejador,
		secuencia: secuencia,
	}
	ruta.Manejador = espia
	return ruta, espia
}

func nuevoDispatcherRutaAltaPrueba(
	t *testing.T,
	rutas []httpapi.RutaExacta,
	autoridad httpapi.AutoridadRutasExactas,
) *httpapi.Handler {
	t.Helper()
	almacen := memory.NewStore()
	servicio, err := vecapp.NewService(almacen, almacen, almacen)
	if err != nil {
		t.Fatalf("construir servicio VEC: %v", err)
	}
	dispatcher, err := httpapi.NewHandlerWithOptions(
		servicio,
		httpapi.HandlerOptions{
			RutasExactas:          rutas,
			AutoridadRutasExactas: autoridad,
		},
	)
	if err != nil {
		t.Fatalf("construir dispatcher exacto: %v", err)
	}
	return dispatcher
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

func exigirPrecedenciaRutaAltaPrueba(
	t *testing.T,
	secuencia *atomic.Int64,
	autoridad *autoridadExteriorRutaAltaEspiaPrueba,
	manejador *manejadorRutaAltaEspiaPrueba,
	fronteras *fronterasRutaAltaEspiaPrueba,
) {
	t.Helper()
	llamadas, ruta := autoridad.estado()
	if llamadas != 1 || ruta != rutaAltaLiteralPrueba ||
		autoridad.orden.Load() != 1 || manejador.llamadas.Load() != 1 ||
		manejador.orden.Load() != 2 || secuencia.Load() != 2 {
		t.Fatalf(
			"precedencia exterior = llamadas:%d ruta:%q orden:%d; "+
				"manejador llamadas:%d orden:%d secuencia:%d",
			llamadas, ruta, autoridad.orden.Load(),
			manejador.llamadas.Load(), manejador.orden.Load(), secuencia.Load(),
		)
	}
	if fronteras.autoridad.Load() != 0 || fronteras.ejecuciones.Load() != 0 {
		t.Fatalf(
			"fronteras interiores invocadas: autoridad=%d ejecutor=%d",
			fronteras.autoridad.Load(), fronteras.ejecuciones.Load(),
		)
	}
}

func TestRutaAltaIntegradaAutorizaGETAntesDeResponder405(t *testing.T) {
	t.Parallel()
	var secuencia atomic.Int64
	fronteras := &fronterasRutaAltaEspiaPrueba{}
	ruta, manejador := nuevaRutaAltaObservadaPrueba(t, &secuencia, fronteras)
	autoridad := &autoridadExteriorRutaAltaEspiaPrueba{secuencia: &secuencia}
	dispatcher := nuevoDispatcherRutaAltaPrueba(
		t, []httpapi.RutaExacta{ruta}, autoridad,
	)

	respuesta := httptest.NewRecorder()
	dispatcher.ServeHTTP(respuesta, httptest.NewRequest(
		http.MethodGet, rutaAltaLiteralPrueba, nil,
	))
	if respuesta.Code != http.StatusMethodNotAllowed ||
		respuesta.Header().Get("Allow") != http.MethodPost {
		t.Fatalf(
			"GET integrado = estado:%d allow:%q",
			respuesta.Code, respuesta.Header().Get("Allow"),
		)
	}
	exigirPrecedenciaRutaAltaPrueba(
		t, &secuencia, autoridad, manejador, fronteras,
	)
}

func TestRutaAltaIntegradaAutorizaQueryExactaAntesDeResponder400(t *testing.T) {
	t.Parallel()
	var secuencia atomic.Int64
	fronteras := &fronterasRutaAltaEspiaPrueba{}
	ruta, manejador := nuevaRutaAltaObservadaPrueba(t, &secuencia, fronteras)
	autoridad := &autoridadExteriorRutaAltaEspiaPrueba{secuencia: &secuencia}
	dispatcher := nuevoDispatcherRutaAltaPrueba(
		t, []httpapi.RutaExacta{ruta}, autoridad,
	)
	peticion := nuevaPeticionContratacionErrorPrueba(rutaAltaLiteralPrueba)
	peticion.URL.RawQuery = "actor=inyectado"

	respuesta := httptest.NewRecorder()
	dispatcher.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusBadRequest {
		t.Fatalf(
			"query integrada = estado:%d cuerpo:%s",
			respuesta.Code, respuesta.Body.String(),
		)
	}
	exigirPrecedenciaRutaAltaPrueba(
		t, &secuencia, autoridad, manejador, fronteras,
	)
}

func TestRutaAltaIntegradaIgnoraRutaDistintaSinAutoridadNiEfectos(
	t *testing.T,
) {
	t.Parallel()
	var secuencia atomic.Int64
	fronteras := &fronterasRutaAltaEspiaPrueba{}
	ruta, manejador := nuevaRutaAltaObservadaPrueba(t, &secuencia, fronteras)
	autoridad := &autoridadExteriorRutaAltaEspiaPrueba{secuencia: &secuencia}
	dispatcher := nuevoDispatcherRutaAltaPrueba(
		t, []httpapi.RutaExacta{ruta}, autoridad,
	)

	respuesta := httptest.NewRecorder()
	dispatcher.ServeHTTP(
		respuesta,
		nuevaPeticionContratacionErrorPrueba(rutaAltaLiteralPrueba+"/otra"),
	)
	llamadas, seleccionada := autoridad.estado()
	if respuesta.Code != http.StatusUnauthorized || llamadas != 0 ||
		seleccionada != "" || secuencia.Load() != 0 ||
		manejador.llamadas.Load() != 0 || manejador.orden.Load() != 0 ||
		fronteras.autoridad.Load() != 0 || fronteras.ejecuciones.Load() != 0 {
		t.Fatalf(
			"ruta distinta produjo actividad: estado=%d exterior=(%d,%q) "+
				"secuencia=%d manejador=(%d,%d) interior=(%d,%d)",
			respuesta.Code, llamadas, seleccionada, secuencia.Load(),
			manejador.llamadas.Load(), manejador.orden.Load(),
			fronteras.autoridad.Load(), fronteras.ejecuciones.Load(),
		)
	}
}

func TestNuevaRutaAltaConNuevasRutasFallaCerradoPorDuplicado(t *testing.T) {
	t.Parallel()
	fronteras := &fronterasRutaAltaEspiaPrueba{}
	alta, err := NuevaRutaAlta(fronteras, fronteras, relojComposicionPrueba{})
	if err != nil {
		t.Fatalf("construir declaracion individual: %v", err)
	}
	dependencias := dependenciasRutasPrueba()
	dependencias.AutoridadAlta = fronteras
	dependencias.EjecutorAlta = fronteras
	rutas, err := NuevasRutas(dependencias)
	if err != nil {
		t.Fatalf("construir declaraciones conjuntas: %v", err)
	}
	rutas = append(rutas, alta)
	repeticiones := 0
	for _, ruta := range rutas {
		if ruta.Ruta == rutaAltaLiteralPrueba {
			repeticiones++
		}
	}

	var secuencia atomic.Int64
	autoridad := &autoridadExteriorRutaAltaEspiaPrueba{secuencia: &secuencia}
	almacen := memory.NewStore()
	servicio, err := vecapp.NewService(almacen, almacen, almacen)
	if err != nil {
		t.Fatalf("construir servicio VEC: %v", err)
	}
	handler, err := httpapi.NewHandlerWithOptions(servicio, httpapi.HandlerOptions{
		RutasExactas:          rutas,
		AutoridadRutasExactas: autoridad,
	})
	llamadas, seleccionada := autoridad.estado()
	if repeticiones != 2 || handler != nil ||
		!errors.Is(err, httpapi.ErrRutaExactaInvalida) || llamadas != 0 ||
		seleccionada != "" || secuencia.Load() != 0 ||
		fronteras.autoridad.Load() != 0 || fronteras.ejecuciones.Load() != 0 {
		t.Fatalf(
			"duplicado no cerrado: repeticiones=%d handler=%T error=%v "+
				"exterior=(%d,%q,%d) interior=(%d,%d)",
			repeticiones, handler, err, llamadas, seleccionada, secuencia.Load(),
			fronteras.autoridad.Load(), fronteras.ejecuciones.Load(),
		)
	}
}
