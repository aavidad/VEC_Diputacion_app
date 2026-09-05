package httpinterno

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	claveRegistroComunicacionHTTPPrueba = "618f47a6-5d2b-4c10-8a11-1234567890ab"
	claveResolverComunicacionHTTPPrueba = "718f47a6-5d2b-4c10-9a11-1234567890ab"
)

type ejecutorComunicacionLlamamientoHTTPPrueba struct {
	mu                sync.Mutex
	comunicacion      ports.ComunicacionProbatoria
	errRegistro       error
	resolucion        ports.ResultadoResolucionLlamamiento
	errResolucion     error
	duranteRegistro   func(context.Context)
	duranteResolucion func(context.Context)
	registros         []ports.SolicitudRegistrarComunicacionLlamamiento
	resoluciones      []ports.SolicitudResolverLlamamiento
}

func (e *ejecutorComunicacionLlamamientoHTTPPrueba) Registrar(
	ctx context.Context,
	solicitud ports.SolicitudRegistrarComunicacionLlamamiento,
) (ports.ComunicacionProbatoria, error) {
	e.mu.Lock()
	e.registros = append(e.registros, solicitud)
	e.mu.Unlock()
	if e.duranteRegistro != nil {
		e.duranteRegistro(ctx)
	}
	return e.comunicacion, e.errRegistro
}

func (e *ejecutorComunicacionLlamamientoHTTPPrueba) Resolver(
	ctx context.Context,
	solicitud ports.SolicitudResolverLlamamiento,
) (ports.ResultadoResolucionLlamamiento, error) {
	e.mu.Lock()
	e.resoluciones = append(e.resoluciones, solicitud)
	e.mu.Unlock()
	if e.duranteResolucion != nil {
		e.duranteResolucion(ctx)
	}
	return e.resolucion, e.errResolucion
}

func (e *ejecutorComunicacionLlamamientoHTTPPrueba) totales() (int, int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return len(e.registros), len(e.resoluciones)
}

func (e *ejecutorComunicacionLlamamientoHTTPPrueba) ultimaRegistro() (
	ports.SolicitudRegistrarComunicacionLlamamiento,
	bool,
) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(e.registros) == 0 {
		return ports.SolicitudRegistrarComunicacionLlamamiento{}, false
	}
	return e.registros[len(e.registros)-1], true
}

func (e *ejecutorComunicacionLlamamientoHTTPPrueba) ultimaResolucion() (
	ports.SolicitudResolverLlamamiento,
	bool,
) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(e.resoluciones) == 0 {
		return ports.SolicitudResolverLlamamiento{}, false
	}
	return e.resoluciones[len(e.resoluciones)-1], true
}

func TestComunicacionLlamamientoHTTPRegistroLocalNoPublicaPlazo(t *testing.T) {
	s := solicitudRegistroComunicacionHTTPPrueba()
	r := comunicacionHTTPPrueba(s)
	r.RegistradaEn = r.EntregadaEn
	r.EntregadaEn = time.Time{}
	r.RespuestaHasta = time.Time{}
	r.IntencionEnvioRef = "outbox:local"
	for _, caso := range []struct {
		estado ports.EstadoResultadoComunicacionLlamamiento
		codigo int
	}{
		{ports.ResultadoComunicacionLlamamientoLocal, http.StatusCreated},
		{ports.ResultadoComunicacionLlamamientoReplayLocal, http.StatusOK},
	} {
		r.Estado = caso.estado
		salida, codigo, valido := proyectarRegistroComunicacionLlamamiento(s, r)
		if !valido || codigo != caso.codigo || salida.RegistradaEn == "" ||
			salida.IntencionEnvioRef != r.IntencionEnvioRef ||
			salida.EstadoLocal != string(caso.estado) {
			t.Fatal("proyección local divergente")
		}
		b, err := json.Marshal(salida)
		if err != nil || bytes.Contains(b, []byte("respuesta_hasta")) ||
			bytes.Contains(b, []byte("entregada_en")) || bytes.Contains(b, []byte("0001-")) {
			t.Fatal("entrega o plazo ficticios publicados")
		}
	}
}

func solicitudRegistroComunicacionHTTPPrueba() ports.SolicitudRegistrarComunicacionLlamamiento {
	return ports.SolicitudRegistrarComunicacionLlamamiento{
		ClaveIdempotencia: claveRegistroComunicacionHTTPPrueba,
		OrganizacionRef:   "organizacion:http-comunicacion",
		ExpedienteRef:     "expediente:http-comunicacion",
		LlamamientoRef:    "llamamiento:http-comunicacion",
		VersionEsperada:   7,
		PruebaEntregaRef:  "entrega:http-probatoria",
	}
}

func solicitudResolucionComunicacionHTTPPrueba(
	respuesta ports.RespuestaLlamamiento,
) ports.SolicitudResolverLlamamiento {
	prueba := "respuesta:http-probatoria"
	if respuesta == ports.RespuestaLlamamientoExpirada {
		prueba = ""
	}
	return ports.SolicitudResolverLlamamiento{
		ClaveIdempotencia:  claveResolverComunicacionHTTPPrueba,
		OrganizacionRef:    "organizacion:http-comunicacion",
		ExpedienteRef:      "expediente:http-comunicacion",
		LlamamientoRef:     "llamamiento:http-comunicacion",
		ComunicacionRef:    "comunicacion:http-probatoria",
		VersionEsperada:    8,
		Respuesta:          respuesta,
		PruebaRespuestaRef: prueba,
	}
}

func referenciaGobernadaComunicacionHTTPPrueba(
	referencia string,
	digito string,
) ports.ReferenciaGobernadaComunicacionLlamamiento {
	return ports.ReferenciaGobernadaComunicacionLlamamiento{
		Referencia:   "catalogo:" + referencia,
		Version:      3,
		HuellaSHA256: strings.Repeat(digito, 64),
	}
}

func comunicacionHTTPPrueba(
	solicitud ports.SolicitudRegistrarComunicacionLlamamiento,
) ports.ComunicacionProbatoria {
	return ports.ComunicacionProbatoria{
		Solicitud:       solicitud,
		ComunicacionRef: "comunicacion:http-probatoria",
		Canal: referenciaGobernadaComunicacionHTTPPrueba(
			"canal-comunicacion",
			"a",
		),
		Politica: referenciaGobernadaComunicacionHTTPPrueba(
			"politica-comunicacion",
			"b",
		),
		ReciboRef:         "recibo:http-comunicacion",
		AuditoriaRef:      "auditoria:http-comunicacion",
		VersionResultante: solicitud.VersionEsperada + 1,
		EntregadaEn: time.Date(
			2026, 8, 31, 10, 0, 0, 123000000, time.UTC,
		),
		RespuestaHasta: time.Date(
			2026, 9, 2, 10, 0, 0, 123000000, time.UTC,
		),
		Estado: ports.ResultadoComunicacionLlamamientoConfirmado,
	}
}

func resolucionComunicacionHTTPPrueba(
	solicitud ports.SolicitudResolverLlamamiento,
) ports.ResultadoResolucionLlamamiento {
	resuelta := time.Date(2026, 8, 31, 12, 0, 0, 456000000, time.UTC)
	resultado := ports.ResultadoResolucionLlamamiento{
		Solicitud: solicitud,
		Politica: referenciaGobernadaComunicacionHTTPPrueba(
			"politica-comunicacion",
			"b",
		),
		EvaluacionPlazoRef: "evaluacion:http-plazo",
		EstadoPlazo:        ports.PlazoLlamamientoVigente,
		ResolucionRef:      "resolucion:http-local",
		ReciboLocalRef:     "recibo:http-resolucion-local",
		AuditoriaRef:       "auditoria:http-resolucion",
		VersionResultante:  solicitud.VersionEsperada + 1,
		ResueltaEn:         resuelta,
		Estado:             ports.ResultadoComunicacionLlamamientoConfirmado,
	}
	if solicitud.Respuesta == ports.RespuestaLlamamientoAceptada {
		return resultado
	}
	if solicitud.Respuesta == ports.RespuestaLlamamientoExpirada {
		resultado.EstadoPlazo = ports.PlazoLlamamientoExpirado
	}
	resultado.IntencionSiguiente = ports.IntencionOutboxSiguienteCandidato{
		Solicitud:         solicitud,
		ResolucionRef:     resultado.ResolucionRef,
		LlamamientoRef:    solicitud.LlamamientoRef,
		ClaveIdempotencia: solicitud.ClaveIdempotencia,
		VersionEsperada:   solicitud.VersionEsperada,
		VersionResultante: resultado.VersionResultante,
		IntencionRef:      "outbox:http-siguiente",
		ComandoOpacoRef:   "comando:http-no-publicable",
		Estado:            ports.OutboxSiguienteCandidatoPendiente,
		ActualizadaEn:     resuelta,
	}
	return resultado
}

func cuerpoRegistroComunicacionHTTPPrueba() string {
	return `{"clave_idempotencia":"` + claveRegistroComunicacionHTTPPrueba +
		`","organizacion_ref":"organizacion:http-comunicacion",` +
		`"expediente_ref":"expediente:http-comunicacion",` +
		`"llamamiento_ref":"llamamiento:http-comunicacion",` +
		`"version_esperada":7,"prueba_entrega_ref":"entrega:http-probatoria"}`
}

func cuerpoResolucionComunicacionHTTPPrueba(
	respuesta ports.RespuestaLlamamiento,
) string {
	prueba := "respuesta:http-probatoria"
	if respuesta == ports.RespuestaLlamamientoExpirada {
		prueba = ""
	}
	return `{"clave_idempotencia":"` + claveResolverComunicacionHTTPPrueba +
		`","organizacion_ref":"organizacion:http-comunicacion",` +
		`"expediente_ref":"expediente:http-comunicacion",` +
		`"llamamiento_ref":"llamamiento:http-comunicacion",` +
		`"comunicacion_ref":"comunicacion:http-probatoria",` +
		`"version_esperada":8,"respuesta":"` + string(respuesta) +
		`","prueba_respuesta_ref":"` + prueba + `"}`
}

func peticionComunicacionHTTPPrueba(ruta string, cuerpo string) *http.Request {
	peticion := httptest.NewRequest(http.MethodPost, ruta, strings.NewReader(cuerpo))
	peticion.Header.Set("Content-Type", "application/json; charset=utf-8")
	peticion.Header.Set("Accept", "application/json")
	return peticion
}

func nuevoManejadorComunicacionHTTPPrueba(
	t *testing.T,
	ejecutor EjecutorComunicacionLlamamiento,
) http.Handler {
	t.Helper()
	manejador, err := NuevoManejadorComunicacionLlamamiento(ejecutor)
	if err != nil {
		t.Fatalf("construir manejador: %v", err)
	}
	return manejador
}

func TestManejadorComunicacionLlamamientoRegistraYProyectaMinimoLocal(
	t *testing.T,
) {
	t.Parallel()
	solicitud := solicitudRegistroComunicacionHTTPPrueba()
	ejecutor := &ejecutorComunicacionLlamamientoHTTPPrueba{
		comunicacion: comunicacionHTTPPrueba(solicitud),
	}
	manejador := nuevoManejadorComunicacionHTTPPrueba(t, ejecutor)
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(
		respuesta,
		peticionComunicacionHTTPPrueba(
			RutaRegistroComunicacionLlamamiento,
			cuerpoRegistroComunicacionHTTPPrueba(),
		),
	)
	esperado := `{"data":{"esquema":"` + EsquemaRegistroComunicacionLlamamiento +
		`","estado_local":"confirmado",` +
		`"comunicacion_ref":"comunicacion:http-probatoria",` +
		`"recibo_ref":"recibo:http-comunicacion",` +
		`"auditoria_ref":"auditoria:http-comunicacion",` +
		`"version_resultante":8,` +
		`"respuesta_hasta":"2026-09-02T10:00:00.123Z"}}`
	obtenida, existe := ejecutor.ultimaRegistro()
	registros, resoluciones := ejecutor.totales()
	if respuesta.Code != http.StatusCreated || respuesta.Body.String() != esperado ||
		!existe || obtenida != solicitud || registros != 1 || resoluciones != 0 {
		t.Fatalf(
			"estado=%d cuerpo=%s entrada=%+v llamadas=%d/%d",
			respuesta.Code,
			respuesta.Body,
			obtenida,
			registros,
			resoluciones,
		)
	}
}

func TestManejadorComunicacionLlamamientoResuelveLasTresRespuestas(
	t *testing.T,
) {
	t.Parallel()
	casos := []struct {
		nombre    string
		respuesta ports.RespuestaLlamamiento
		plazo     string
		intencion bool
	}{
		{"aceptada", ports.RespuestaLlamamientoAceptada, "vigente", false},
		{"renunciada", ports.RespuestaLlamamientoRenunciada, "vigente", true},
		{"expirada", ports.RespuestaLlamamientoExpirada, "expirado", true},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			solicitud := solicitudResolucionComunicacionHTTPPrueba(caso.respuesta)
			ejecutor := &ejecutorComunicacionLlamamientoHTTPPrueba{
				resolucion: resolucionComunicacionHTTPPrueba(solicitud),
			}
			manejador := nuevoManejadorComunicacionHTTPPrueba(t, ejecutor)
			respuesta := httptest.NewRecorder()
			manejador.ServeHTTP(
				respuesta,
				peticionComunicacionHTTPPrueba(
					RutaResolucionComunicacionLlamamiento,
					cuerpoResolucionComunicacionHTTPPrueba(caso.respuesta),
				),
			)
			var salida struct {
				Data struct {
					Esquema            string          `json:"esquema"`
					Respuesta          string          `json:"respuesta"`
					EstadoPlazo        string          `json:"estado_plazo"`
					EstadoLocal        string          `json:"estado_local"`
					ResolucionRef      string          `json:"resolucion_ref"`
					ReciboLocalRef     string          `json:"recibo_local_ref"`
					AuditoriaRef       string          `json:"auditoria_ref"`
					IntencionSiguiente json.RawMessage `json:"intencion_siguiente"`
					VersionResultante  uint64          `json:"version_resultante"`
					ResueltaEn         string          `json:"resuelta_en"`
				} `json:"data"`
			}
			err := json.Unmarshal(respuesta.Body.Bytes(), &salida)
			obtenida, existe := ejecutor.ultimaResolucion()
			registros, resoluciones := ejecutor.totales()
			tieneIntencion := len(salida.Data.IntencionSiguiente) != 0
			if respuesta.Code != http.StatusCreated || err != nil || !existe ||
				obtenida != solicitud || registros != 0 || resoluciones != 1 ||
				salida.Data.Esquema != EsquemaResolucionComunicacionLlamamiento ||
				salida.Data.Respuesta != string(caso.respuesta) ||
				salida.Data.EstadoPlazo != caso.plazo ||
				salida.Data.EstadoLocal != "confirmado" ||
				salida.Data.ResolucionRef != "resolucion:http-local" ||
				salida.Data.ReciboLocalRef != "recibo:http-resolucion-local" ||
				salida.Data.AuditoriaRef != "auditoria:http-resolucion" ||
				salida.Data.VersionResultante != 9 ||
				salida.Data.ResueltaEn != "2026-08-31T12:00:00.456Z" ||
				tieneIntencion != caso.intencion {
				t.Fatalf("respuesta incoherente: estado=%d salida=%+v err=%v", respuesta.Code, salida, err)
			}
			for _, prohibido := range []string{
				"comando:http-no-publicable", "comando_opaco", "prueba_respuesta",
				"organizacion:http", "expediente:http", "llamamiento:http",
			} {
				if bytes.Contains(respuesta.Body.Bytes(), []byte(prohibido)) {
					t.Fatalf("dato no publicable %q filtrado: %s", prohibido, respuesta.Body)
				}
			}
		})
	}
}

func TestManejadorComunicacionLlamamientoReplayEs200YSoloEstadoLocal(
	t *testing.T,
) {
	t.Parallel()
	solicitudRegistro := solicitudRegistroComunicacionHTTPPrueba()
	comunicacion := comunicacionHTTPPrueba(solicitudRegistro)
	comunicacion.Estado = ports.ResultadoComunicacionLlamamientoReplay
	solicitudResolucion := solicitudResolucionComunicacionHTTPPrueba(
		ports.RespuestaLlamamientoRenunciada,
	)
	resolucion := resolucionComunicacionHTTPPrueba(solicitudResolucion)
	resolucion.Estado = ports.ResultadoComunicacionLlamamientoReplay
	resolucion.IntencionSiguiente.Estado = ports.OutboxSiguienteCandidatoDespachada
	ejecutor := &ejecutorComunicacionLlamamientoHTTPPrueba{
		comunicacion: comunicacion,
		resolucion:   resolucion,
	}
	manejador := nuevoManejadorComunicacionHTTPPrueba(t, ejecutor)
	peticiones := []*http.Request{
		peticionComunicacionHTTPPrueba(
			RutaRegistroComunicacionLlamamiento,
			cuerpoRegistroComunicacionHTTPPrueba(),
		),
		peticionComunicacionHTTPPrueba(
			RutaResolucionComunicacionLlamamiento,
			cuerpoResolucionComunicacionHTTPPrueba(
				ports.RespuestaLlamamientoRenunciada,
			),
		),
	}
	for indice, peticion := range peticiones {
		respuesta := httptest.NewRecorder()
		manejador.ServeHTTP(respuesta, peticion)
		if respuesta.Code != http.StatusOK ||
			!bytes.Contains(respuesta.Body.Bytes(), []byte(`"estado_local":"replay_confirmado"`)) {
			t.Fatalf("peticion %d: estado=%d cuerpo=%s", indice, respuesta.Code, respuesta.Body)
		}
		if bytes.Contains(respuesta.Body.Bytes(), []byte("comando:http")) {
			t.Fatalf("comando opaco publicado: %s", respuesta.Body)
		}
	}
	if !bytes.Contains(
		[]byte(func() string {
			respuesta := httptest.NewRecorder()
			manejador.ServeHTTP(
				respuesta,
				peticionComunicacionHTTPPrueba(
					RutaResolucionComunicacionLlamamiento,
					cuerpoResolucionComunicacionHTTPPrueba(
						ports.RespuestaLlamamientoRenunciada,
					),
				),
			)
			return respuesta.Body.String()
		}()),
		[]byte(`"estado_local":"despachada"`),
	) {
		t.Fatal("el replay no proyecto el estado estrictamente local de la intencion")
	}
}

func TestManejadorComunicacionLlamamientoClasificaErroresOpacos(t *testing.T) {
	t.Parallel()
	casos := []struct {
		nombre string
		err    error
		estado int
		codigo string
	}{
		{"clave", application.ErrClaveComunicacionLlamamientoEnColision, http.StatusConflict, "clave_idempotencia_reutilizada"},
		{"version", application.ErrVersionComunicacionLlamamientoEnConflicto, http.StatusConflict, "version_en_conflicto"},
		{"denegada", application.ErrComunicacionLlamamientoDenegada, http.StatusForbidden, "acceso_denegado"},
		{"cancelada", context.Canceled, http.StatusRequestTimeout, "peticion_cancelada"},
		{"plazo", context.DeadlineExceeded, http.StatusGatewayTimeout, "plazo_agotado"},
		{"solicitud", application.ErrSolicitudComunicacionLlamamientoInvalida, http.StatusUnprocessableEntity, "contenido_no_valido"},
		{"resultado", application.ErrResultadoComunicacionLlamamientoNoConfiable, http.StatusBadGateway, "resultado_no_confiable"},
		{"indisponible", application.ErrComunicacionLlamamientoNoDisponible, http.StatusServiceUnavailable, "servicio_no_disponible"},
		{"privado", errors.New("postgres://persona:secreto@privado/rrhh"), http.StatusServiceUnavailable, "servicio_no_disponible"},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			ejecutor := &ejecutorComunicacionLlamamientoHTTPPrueba{errRegistro: caso.err}
			manejador := nuevoManejadorComunicacionHTTPPrueba(t, ejecutor)
			respuesta := httptest.NewRecorder()
			manejador.ServeHTTP(
				respuesta,
				peticionComunicacionHTTPPrueba(
					RutaRegistroComunicacionLlamamiento,
					cuerpoRegistroComunicacionHTTPPrueba(),
				),
			)
			if respuesta.Code != caso.estado || !bytes.Contains(
				respuesta.Body.Bytes(),
				[]byte(`"codigo":"`+caso.codigo+`"`),
			) {
				t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body)
			}
			for _, privado := range []string{"postgres", "persona", "secreto", "privado"} {
				if bytes.Contains(respuesta.Body.Bytes(), []byte(privado)) {
					t.Fatalf("detalle privado filtrado: %s", respuesta.Body)
				}
			}
		})
	}
}

func TestManejadorComunicacionLlamamientoRevalidaResultados(t *testing.T) {
	t.Parallel()
	solicitudRegistro := solicitudRegistroComunicacionHTTPPrueba()
	comunicacion := comunicacionHTTPPrueba(solicitudRegistro)
	comunicacion.VersionResultante++
	solicitudResolucion := solicitudResolucionComunicacionHTTPPrueba(
		ports.RespuestaLlamamientoRenunciada,
	)
	resolucion := resolucionComunicacionHTTPPrueba(solicitudResolucion)
	resolucion.IntencionSiguiente.ComandoOpacoRef = ""
	casos := []struct {
		ruta     string
		cuerpo   string
		ejecutor *ejecutorComunicacionLlamamientoHTTPPrueba
	}{
		{RutaRegistroComunicacionLlamamiento, cuerpoRegistroComunicacionHTTPPrueba(), &ejecutorComunicacionLlamamientoHTTPPrueba{comunicacion: comunicacion}},
		{RutaResolucionComunicacionLlamamiento, cuerpoResolucionComunicacionHTTPPrueba(ports.RespuestaLlamamientoRenunciada), &ejecutorComunicacionLlamamientoHTTPPrueba{resolucion: resolucion}},
	}
	for indice, caso := range casos {
		manejador := nuevoManejadorComunicacionHTTPPrueba(t, caso.ejecutor)
		respuesta := httptest.NewRecorder()
		manejador.ServeHTTP(
			respuesta,
			peticionComunicacionHTTPPrueba(caso.ruta, caso.cuerpo),
		)
		if respuesta.Code != http.StatusBadGateway || !bytes.Contains(
			respuesta.Body.Bytes(),
			[]byte(`"codigo":"resultado_no_confiable"`),
		) {
			t.Fatalf("caso %d: estado=%d cuerpo=%s", indice, respuesta.Code, respuesta.Body)
		}
	}
}

func TestManejadorComunicacionLlamamientoRechazaDependenciasNulas(t *testing.T) {
	t.Parallel()
	var nulo *ejecutorComunicacionLlamamientoHTTPPrueba
	for nombre, dependencia := range map[string]EjecutorComunicacionLlamamiento{
		"nula":        nil,
		"nula tipada": nulo,
	} {
		manejador, err := NuevoManejadorComunicacionLlamamiento(dependencia)
		if manejador != nil || !errors.Is(err, ErrManejadorComunicacionLlamamientoInvalido) {
			t.Fatalf("%s aceptada: manejador=%#v err=%v", nombre, manejador, err)
		}
	}
	tipo := reflect.TypeOf((*EjecutorComunicacionLlamamiento)(nil)).Elem()
	if tipo.NumMethod() != 2 {
		t.Fatalf("HTTP incorpora capacidades ajenas: %v", tipo)
	}
}
