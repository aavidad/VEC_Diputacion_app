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

const clavePropuestaFormalizacionHTTPPrueba = "" +
	"938f47a6-5d2b-4c10-aa11-1234567890ab"

type autoridadPropuestaFormalizacionHTTPPrueba struct {
	mu       sync.Mutex
	contexto ContextoServidorPropuestaFormalizacion
	err      error
	durante  func(context.Context)
	llamadas int
}

func (a *autoridadPropuestaFormalizacionHTTPPrueba) ResolverContextoPropuestaFormalizacion(
	ctx context.Context,
) (ContextoServidorPropuestaFormalizacion, error) {
	a.mu.Lock()
	a.llamadas++
	a.mu.Unlock()
	if a.durante != nil {
		a.durante(ctx)
	}
	return a.contexto, a.err
}

func (a *autoridadPropuestaFormalizacionHTTPPrueba) total() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.llamadas
}

type ejecutorPropuestaFormalizacionHTTPPrueba struct {
	mu        sync.Mutex
	responder func(
		context.Context,
		ports.SolicitudPropuestaFormalizacion,
	) (ports.ResultadoPropuestaFormalizacion, error)
	durante     func(context.Context)
	solicitudes []ports.SolicitudPropuestaFormalizacion
}

func (e *ejecutorPropuestaFormalizacionHTTPPrueba) PrepararYConfirmar(
	ctx context.Context,
	solicitud ports.SolicitudPropuestaFormalizacion,
) (ports.ResultadoPropuestaFormalizacion, error) {
	e.mu.Lock()
	e.solicitudes = append(e.solicitudes, solicitud.Clonar())
	e.mu.Unlock()
	if e.durante != nil {
		e.durante(ctx)
	}
	if e.responder == nil {
		return ports.ResultadoPropuestaFormalizacion{}, nil
	}
	return e.responder(ctx, solicitud.Clonar())
}

func (e *ejecutorPropuestaFormalizacionHTTPPrueba) total() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return len(e.solicitudes)
}

func (e *ejecutorPropuestaFormalizacionHTTPPrueba) ultima() (
	ports.SolicitudPropuestaFormalizacion,
	bool,
) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(e.solicitudes) == 0 {
		return ports.SolicitudPropuestaFormalizacion{}, false
	}
	return e.solicitudes[len(e.solicitudes)-1].Clonar(), true
}

func autoridadPropuestaFormalizacionHTTPValidaPrueba() *autoridadPropuestaFormalizacionHTTPPrueba {
	return &autoridadPropuestaFormalizacionHTTPPrueba{
		contexto: ContextoServidorPropuestaFormalizacion{
			OrganizacionRef: "organizacion:http-formalizacion",
		},
	}
}

func ejecutorPropuestaFormalizacionHTTPValidoPrueba() *ejecutorPropuestaFormalizacionHTTPPrueba {
	return &ejecutorPropuestaFormalizacionHTTPPrueba{
		responder: func(
			_ context.Context,
			solicitud ports.SolicitudPropuestaFormalizacion,
		) (ports.ResultadoPropuestaFormalizacion, error) {
			return resultadoPropuestaFormalizacionHTTPPrueba(
				solicitud,
				ports.ResultadoPropuestaFormalizacionConfirmado,
			), nil
		},
	}
}

func snapshotPropuestaFormalizacionHTTPPrueba(
	referencia string,
	digito string,
) snapshotPropuestaFormalizacionJSON {
	return snapshotPropuestaFormalizacionJSON{
		Referencia:   referencia,
		Version:      7,
		HuellaSHA256: strings.Repeat(digito, 64),
	}
}

func anexoPropuestaFormalizacionHTTPPrueba(
	referencia string,
	digito string,
	tamano uint64,
) anexoPropuestaFormalizacionJSON {
	return anexoPropuestaFormalizacionJSON{
		DocumentoRef: referencia,
		Version:      5,
		HuellaSHA256: strings.Repeat(digito, 64),
		TamanoBytes:  tamano,
	}
}

func entradaPropuestaFormalizacionHTTPPrueba() propuestaFormalizacionEntradaJSON {
	anexos := []anexoPropuestaFormalizacionJSON{
		anexoPropuestaFormalizacionHTTPPrueba("anexo:http-a", "7", 4096),
		anexoPropuestaFormalizacionHTTPPrueba("anexo:http-b", "8", 8192),
	}
	return propuestaFormalizacionEntradaJSON{
		ClaveIdempotencia:                clavePropuestaFormalizacionHTTPPrueba,
		ExpedienteRef:                    "expediente:http-formalizacion",
		LlamamientoRef:                   "llamamiento:http-formalizacion",
		ResolucionLlamamientoAceptadaRef: "resolucion:http-aceptada",
		ReciboResolucionAceptadaRef:      "recibo:http-aceptacion",
		VersionEsperada:                  13,
		TipoFormalizacion: snapshotPropuestaFormalizacionHTTPPrueba(
			"tipo:http-formalizacion",
			"1",
		),
		Plantilla: snapshotPropuestaFormalizacionHTTPPrueba(
			"plantilla:http-formalizacion",
			"2",
		),
		Anexos: &anexos,
		PoliticaFirma: snapshotPropuestaFormalizacionHTTPPrueba(
			"politica:http-firma",
			"3",
		),
		PlanFirma: snapshotPropuestaFormalizacionHTTPPrueba(
			"plan:http-firma",
			"4",
		),
	}
}

func cuerpoPropuestaFormalizacionHTTPPrueba(t *testing.T) string {
	t.Helper()
	contenido, err := json.Marshal(entradaPropuestaFormalizacionHTTPPrueba())
	if err != nil {
		t.Fatalf("codificar entrada: %v", err)
	}
	return string(contenido)
}

func peticionPropuestaFormalizacionHTTPPrueba(t *testing.T) *http.Request {
	t.Helper()
	peticion := httptest.NewRequest(
		http.MethodPost,
		RutaPropuestaFormalizacion,
		strings.NewReader(cuerpoPropuestaFormalizacionHTTPPrueba(t)),
	)
	peticion.Header.Set("Content-Type", "application/json; charset=utf-8")
	peticion.Header.Set("Accept", "application/json")
	return peticion
}

func nuevoManejadorPropuestaFormalizacionHTTPPrueba(
	t *testing.T,
	autoridad AutoridadServidorPropuestaFormalizacion,
	ejecutor EjecutorPropuestaFormalizacion,
) http.Handler {
	t.Helper()
	manejador, err := NuevoManejadorPropuestaFormalizacion(autoridad, ejecutor)
	if err != nil {
		t.Fatalf("construir manejador: %v", err)
	}
	return manejador
}

func resultadoPropuestaFormalizacionHTTPPrueba(
	solicitud ports.SolicitudPropuestaFormalizacion,
	estado ports.EstadoResultadoPropuestaFormalizacion,
) ports.ResultadoPropuestaFormalizacion {
	return ports.ResultadoPropuestaFormalizacion{
		Solicitud:         solicitud.Clonar(),
		PropuestaRef:      "propuesta:http-local",
		ReciboLocalRef:    "recibo:http-local",
		AuditoriaRef:      "auditoria:http-interna",
		VersionResultante: solicitud.VersionEsperada + 1,
		ConfirmadaEn: time.Date(
			2026, 8, 31, 16, 0, 0, 123000000, time.UTC,
		),
		Estado: estado,
	}
}

func TestManejadorPropuestaFormalizacionConfirmaConAutoridadSeparadaYSalidaMinima(
	t *testing.T,
) {
	t.Parallel()
	autoridad := autoridadPropuestaFormalizacionHTTPValidaPrueba()
	ejecutor := ejecutorPropuestaFormalizacionHTTPValidoPrueba()
	manejador := nuevoManejadorPropuestaFormalizacionHTTPPrueba(
		t,
		autoridad,
		ejecutor,
	)
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, peticionPropuestaFormalizacionHTTPPrueba(t))
	esperado := `{"data":{"esquema":"` + EsquemaPropuestaFormalizacion +
		`","estado_local":"confirmado","propuesta_ref":"propuesta:http-local",` +
		`"recibo_local_ref":"recibo:http-local","version_resultante":14,` +
		`"confirmada_en":"2026-08-31T16:00:00.123Z"}}`
	solicitud, existe := ejecutor.ultima()
	if respuesta.Code != http.StatusCreated || respuesta.Body.String() != esperado ||
		!existe || solicitud.OrganizacionRef != "organizacion:http-formalizacion" ||
		solicitud.Validar() != nil || autoridad.total() != 1 || ejecutor.total() != 1 {
		t.Fatalf(
			"estado=%d cuerpo=%s solicitud=%+v llamadas=%d/%d",
			respuesta.Code,
			respuesta.Body,
			solicitud,
			autoridad.total(),
			ejecutor.total(),
		)
	}
	for _, prohibido := range []string{
		`"organizacion_ref"`, `"expediente_ref"`, `"llamamiento_ref"`,
		`"auditoria_ref"`, `"documento_ref"`, `"firma_ref"`,
		`"registro_ref"`, `"descarga_ref"`, `"custodia_ref"`,
		`"actor_ref"`, `"persona_ref"`,
	} {
		if strings.Contains(strings.ToLower(respuesta.Body.String()), prohibido) {
			t.Fatalf("afirmacion o dato no publicable %q: %s", prohibido, respuesta.Body)
		}
	}
}

func TestManejadorPropuestaFormalizacionReplayEs200YLocal(t *testing.T) {
	t.Parallel()
	ejecutor := ejecutorPropuestaFormalizacionHTTPValidoPrueba()
	ejecutor.responder = func(
		_ context.Context,
		solicitud ports.SolicitudPropuestaFormalizacion,
	) (ports.ResultadoPropuestaFormalizacion, error) {
		return resultadoPropuestaFormalizacionHTTPPrueba(
			solicitud,
			ports.ResultadoPropuestaFormalizacionReplay,
		), nil
	}
	manejador := nuevoManejadorPropuestaFormalizacionHTTPPrueba(
		t,
		autoridadPropuestaFormalizacionHTTPValidaPrueba(),
		ejecutor,
	)
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, peticionPropuestaFormalizacionHTTPPrueba(t))
	if respuesta.Code != http.StatusOK || !bytes.Contains(
		respuesta.Body.Bytes(),
		[]byte(`"estado_local":"replay_confirmado"`),
	) || bytes.Contains(respuesta.Body.Bytes(), []byte(`"firma_ref"`)) {
		t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body)
	}
}

func TestManejadorPropuestaFormalizacionClasificaOCCIdempotenciaYDenegacion(
	t *testing.T,
) {
	t.Parallel()
	casos := []struct {
		nombre string
		err    error
		estado int
		codigo string
	}{
		{"occ", application.ErrVersionPropuestaFormalizacionEnConflicto, http.StatusConflict, "version_en_conflicto"},
		{"idempotencia_divergente", application.ErrClavePropuestaFormalizacionEnColision, http.StatusConflict, "clave_idempotencia_reutilizada"},
		{"resolucion_no_aceptada", application.ErrResolucionFormalizacionNoAceptada, http.StatusConflict, "resolucion_no_aceptada"},
		{"denegada", application.ErrPropuestaFormalizacionDenegada, http.StatusForbidden, "acceso_denegado"},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			ejecutor := &ejecutorPropuestaFormalizacionHTTPPrueba{
				responder: func(
					context.Context,
					ports.SolicitudPropuestaFormalizacion,
				) (ports.ResultadoPropuestaFormalizacion, error) {
					return ports.ResultadoPropuestaFormalizacion{}, caso.err
				},
			}
			manejador := nuevoManejadorPropuestaFormalizacionHTTPPrueba(
				t,
				autoridadPropuestaFormalizacionHTTPValidaPrueba(),
				ejecutor,
			)
			respuesta := httptest.NewRecorder()
			manejador.ServeHTTP(
				respuesta,
				peticionPropuestaFormalizacionHTTPPrueba(t),
			)
			if respuesta.Code != caso.estado || !bytes.Contains(
				respuesta.Body.Bytes(),
				[]byte(`"codigo":"`+caso.codigo+`"`),
			) || ejecutor.total() != 1 {
				t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body)
			}
		})
	}
}

func TestManejadorPropuestaFormalizacionRechazaResultadoMasError(
	t *testing.T,
) {
	t.Parallel()
	ejecutor := &ejecutorPropuestaFormalizacionHTTPPrueba{
		responder: func(
			_ context.Context,
			solicitud ports.SolicitudPropuestaFormalizacion,
		) (ports.ResultadoPropuestaFormalizacion, error) {
			return resultadoPropuestaFormalizacionHTTPPrueba(
				solicitud,
				ports.ResultadoPropuestaFormalizacionConfirmado,
			), application.ErrVersionPropuestaFormalizacionEnConflicto
		},
	}
	manejador := nuevoManejadorPropuestaFormalizacionHTTPPrueba(
		t,
		autoridadPropuestaFormalizacionHTTPValidaPrueba(),
		ejecutor,
	)
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, peticionPropuestaFormalizacionHTTPPrueba(t))
	if respuesta.Code != http.StatusBadGateway || !bytes.Contains(
		respuesta.Body.Bytes(),
		[]byte(`"codigo":"resultado_no_confiable"`),
	) || bytes.Contains(respuesta.Body.Bytes(), []byte("propuesta:http-local")) {
		t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body)
	}
}

func TestManejadorPropuestaFormalizacionRevalidaResultado(t *testing.T) {
	t.Parallel()
	ejecutor := ejecutorPropuestaFormalizacionHTTPValidoPrueba()
	ejecutor.responder = func(
		_ context.Context,
		solicitud ports.SolicitudPropuestaFormalizacion,
	) (ports.ResultadoPropuestaFormalizacion, error) {
		resultado := resultadoPropuestaFormalizacionHTTPPrueba(
			solicitud,
			ports.ResultadoPropuestaFormalizacionConfirmado,
		)
		resultado.Solicitud.PlanFirma.Version++
		return resultado, nil
	}
	manejador := nuevoManejadorPropuestaFormalizacionHTTPPrueba(
		t,
		autoridadPropuestaFormalizacionHTTPValidaPrueba(),
		ejecutor,
	)
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, peticionPropuestaFormalizacionHTTPPrueba(t))
	if respuesta.Code != http.StatusBadGateway || !bytes.Contains(
		respuesta.Body.Bytes(),
		[]byte(`"codigo":"resultado_no_confiable"`),
	) {
		t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body)
	}
}

func TestManejadorPropuestaFormalizacionSaneaErrorPrivado(t *testing.T) {
	t.Parallel()
	ejecutor := &ejecutorPropuestaFormalizacionHTTPPrueba{
		responder: func(
			context.Context,
			ports.SolicitudPropuestaFormalizacion,
		) (ports.ResultadoPropuestaFormalizacion, error) {
			return ports.ResultadoPropuestaFormalizacion{},
				errors.New("postgres://persona:secreto@privado/formalizacion")
		},
	}
	manejador := nuevoManejadorPropuestaFormalizacionHTTPPrueba(
		t,
		autoridadPropuestaFormalizacionHTTPValidaPrueba(),
		ejecutor,
	)
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, peticionPropuestaFormalizacionHTTPPrueba(t))
	if respuesta.Code != http.StatusServiceUnavailable {
		t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body)
	}
	for _, privado := range []string{"postgres", "persona", "secreto", "privado"} {
		if bytes.Contains(respuesta.Body.Bytes(), []byte(privado)) {
			t.Fatalf("detalle privado filtrado: %s", respuesta.Body)
		}
	}
}

func TestManejadorPropuestaFormalizacionRechazaDependenciasNulas(t *testing.T) {
	t.Parallel()
	var autoridadNula *autoridadPropuestaFormalizacionHTTPPrueba
	var ejecutorNulo *ejecutorPropuestaFormalizacionHTTPPrueba
	casos := []struct {
		nombre    string
		autoridad AutoridadServidorPropuestaFormalizacion
		ejecutor  EjecutorPropuestaFormalizacion
	}{
		{"autoridad nula", nil, ejecutorPropuestaFormalizacionHTTPValidoPrueba()},
		{"autoridad nula tipada", autoridadNula, ejecutorPropuestaFormalizacionHTTPValidoPrueba()},
		{"ejecutor nulo", autoridadPropuestaFormalizacionHTTPValidaPrueba(), nil},
		{"ejecutor nulo tipado", autoridadPropuestaFormalizacionHTTPValidaPrueba(), ejecutorNulo},
	}
	for _, caso := range casos {
		manejador, err := NuevoManejadorPropuestaFormalizacion(
			caso.autoridad,
			caso.ejecutor,
		)
		if manejador != nil || !errors.Is(err, ErrManejadorPropuestaFormalizacionInvalido) {
			t.Fatalf("%s aceptado: manejador=%#v err=%v", caso.nombre, manejador, err)
		}
	}
}

func TestContratoHTTPPropuestaFormalizacionNoAceptaAutoridadNiEfectoExterno(
	t *testing.T,
) {
	t.Parallel()
	entrada := reflect.TypeOf(propuestaFormalizacionEntradaJSON{})
	for _, campo := range []string{
		"OrganizacionRef", "ActorRef", "PerfilRef", "SesionRef", "AutenticacionRef",
	} {
		if _, existe := entrada.FieldByName(campo); existe {
			t.Fatalf("entrada incorpora autoridad %s", campo)
		}
	}
	salida := reflect.TypeOf(propuestaFormalizacionSalidaJSON{})
	esperados := []string{
		"Esquema", "EstadoLocal", "PropuestaRef", "ReciboLocalRef",
		"VersionResultante", "ConfirmadaEn",
	}
	if salida.NumField() != len(esperados) {
		t.Fatalf("salida no minima: %v", salida)
	}
	for indice, esperado := range esperados {
		if salida.Field(indice).Name != esperado {
			t.Fatalf("campo %d=%s, esperado %s", indice, salida.Field(indice).Name, esperado)
		}
	}
	for _, prohibido := range []string{
		"Documento", "Firma", "Registro", "Descarga", "Custodia", "Auditoria",
	} {
		if _, existe := salida.FieldByName(prohibido + "Ref"); existe {
			t.Fatalf("salida afirma efecto %s", prohibido)
		}
	}
}
