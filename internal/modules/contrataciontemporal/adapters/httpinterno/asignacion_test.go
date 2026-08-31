package httpinterno

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type autoridadAsignacionPrueba struct {
	contexto ContextoCanalAsignacion
	err      error
	antes    func()
	llamadas int
}

func (a *autoridadAsignacionPrueba) ResolverContextoCanalAsignacion(
	context.Context,
) (ContextoCanalAsignacion, error) {
	a.llamadas++
	if a.antes != nil {
		a.antes()
	}
	return a.contexto, a.err
}

type ejecutorAsignacionPrueba struct {
	recibo         ports.ReciboAsignacion
	err            error
	antes          func()
	asignaciones   int
	reasignaciones int
	solicitud      application.SolicitudAsignarUnidad
	reasignacion   application.SolicitudReasignarUnidad
}

func (e *ejecutorAsignacionPrueba) Asignar(
	_ context.Context,
	solicitud application.SolicitudAsignarUnidad,
) (ports.ReciboAsignacion, error) {
	e.asignaciones++
	e.solicitud = solicitud
	if e.antes != nil {
		e.antes()
	}
	return e.recibo, e.err
}

func (e *ejecutorAsignacionPrueba) Reasignar(
	_ context.Context,
	solicitud application.SolicitudReasignarUnidad,
) (ports.ReciboAsignacion, error) {
	e.reasignaciones++
	e.reasignacion = solicitud
	if e.antes != nil {
		e.antes()
	}
	return e.recibo, e.err
}

func TestManejadorAsignacionDelegaUnaVezSinAutoridadHTTP(t *testing.T) {
	contexto := contextoCanalAsignacionPrueba()
	autoridad := &autoridadAsignacionPrueba{contexto: contexto}
	ejecutor := &ejecutorAsignacionPrueba{
		recibo: reciboAsignacionHTTPPrueba(
			ports.OperacionRegistrarAsignacion,
			3,
		),
	}
	manejador, err := NuevoManejadorAsignacion(autoridad, ejecutor)
	if err != nil {
		t.Fatal(err)
	}
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(
		respuesta,
		nuevaPeticionAsignacionPrueba(
			RutaAsignaciones,
			cuerpoAsignacionPrueba(3),
		),
	)

	if respuesta.Code != http.StatusCreated || autoridad.llamadas != 1 ||
		ejecutor.asignaciones != 1 || ejecutor.reasignaciones != 0 {
		t.Fatalf(
			"estado=%d autoridad=%d asignar=%d reasignar=%d cuerpo=%s",
			respuesta.Code,
			autoridad.llamadas,
			ejecutor.asignaciones,
			ejecutor.reasignaciones,
			respuesta.Body.String(),
		)
	}
	solicitud := ejecutor.solicitud
	if solicitud.AutenticacionRef != contexto.AutenticacionRef ||
		solicitud.SesionRef != contexto.SesionRef ||
		solicitud.PerfilRef != contexto.PerfilRef ||
		solicitud.OrganizacionRef != contexto.OrganizacionRef ||
		solicitud.ExpedienteRef != "expediente:asignacion:http:001" ||
		solicitud.VersionEsperada != 3 ||
		solicitud.ClaveIdempotencia !=
			"11111111-2222-4333-8444-555555555555" ||
		solicitud.UnidadRef != "unidad:destino:http:001" ||
		solicitud.ResponsableRef != "persona:responsable:http:001" {
		t.Fatalf("solicitud mal traducida: %#v", solicitud)
	}
	comprobarRespuestaAsignacionMinimizada(t, respuesta)
}

func TestManejadorAsignacionDelegaReasignacionUnaVez(t *testing.T) {
	contexto := contextoCanalAsignacionPrueba()
	autoridad := &autoridadAsignacionPrueba{contexto: contexto}
	ejecutor := &ejecutorAsignacionPrueba{
		recibo: reciboAsignacionHTTPPrueba(
			ports.OperacionRegistrarReasignacion,
			7,
		),
	}
	manejador, err := NuevoManejadorAsignacion(autoridad, ejecutor)
	if err != nil {
		t.Fatal(err)
	}
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(
		respuesta,
		nuevaPeticionAsignacionPrueba(
			RutaReasignaciones,
			cuerpoReasignacionPrueba(7),
		),
	)

	if respuesta.Code != http.StatusCreated || autoridad.llamadas != 1 ||
		ejecutor.asignaciones != 0 || ejecutor.reasignaciones != 1 {
		t.Fatalf("delegación inesperada: %d / %#v", respuesta.Code, ejecutor)
	}
	solicitud := ejecutor.reasignacion
	if solicitud.AutenticacionRef != contexto.AutenticacionRef ||
		solicitud.SesionRef != contexto.SesionRef ||
		solicitud.PerfilRef != contexto.PerfilRef ||
		solicitud.OrganizacionRef != contexto.OrganizacionRef ||
		solicitud.VersionEsperada != 7 ||
		solicitud.MotivoReasignacionClave != "necesidad_servicio" ||
		solicitud.Observaciones != "Cambio motivado de unidad responsable." {
		t.Fatalf("reasignación mal traducida: %#v", solicitud)
	}
}

func TestNuevoManejadorAsignacionRechazaNilYNilTipado(t *testing.T) {
	contexto := contextoCanalAsignacionPrueba()
	autoridad := &autoridadAsignacionPrueba{contexto: contexto}
	ejecutor := &ejecutorAsignacionPrueba{}
	var autoridadNil *autoridadAsignacionPrueba
	var ejecutorNil *ejecutorAsignacionPrueba
	casos := []struct {
		nombre    string
		autoridad AutoridadContextoCanalAsignacion
		ejecutor  EjecutorAsignacion
	}{
		{"autoridad nil", nil, ejecutor},
		{"ejecutor nil", autoridad, nil},
		{"autoridad nil tipado", autoridadNil, ejecutor},
		{"ejecutor nil tipado", autoridad, ejecutorNil},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			manejador, err := NuevoManejadorAsignacion(
				caso.autoridad,
				caso.ejecutor,
			)
			if manejador != nil ||
				!errors.Is(err, ErrManejadorAsignacionInvalido) {
				t.Fatalf("manejador=%#v err=%v", manejador, err)
			}
		})
	}
}

func contextoCanalAsignacionPrueba() ContextoCanalAsignacion {
	return ContextoCanalAsignacion{
		AutenticacionRef: "aut_aaaaaaaaaaaaaaaaaaaaaaaa",
		SesionRef:        "ses_bbbbbbbbbbbbbbbbbbbbbbbb",
		PerfilRef:        "prf_cccccccccccccccccccccccc",
		OrganizacionRef:  "organizacion:dipgra:http:001",
	}
}

func cuerpoAsignacionPrueba(version uint64) string {
	return `{"expediente_ref":"expediente:asignacion:http:001",` +
		`"version_esperada":` + numeroJSONAsignacionPrueba(version) + `,` +
		`"clave_idempotencia":"11111111-2222-4333-8444-555555555555",` +
		`"unidad_ref":"unidad:destino:http:001",` +
		`"responsable_ref":"persona:responsable:http:001"}`
}

func cuerpoReasignacionPrueba(version uint64) string {
	return `{"expediente_ref":"expediente:asignacion:http:001",` +
		`"version_esperada":` + numeroJSONAsignacionPrueba(version) + `,` +
		`"clave_idempotencia":"11111111-2222-4333-8444-555555555555",` +
		`"unidad_ref":"unidad:destino:http:001",` +
		`"responsable_ref":"persona:responsable:http:001",` +
		`"motivo_reasignacion_clave":"necesidad_servicio",` +
		`"observaciones":"Cambio motivado de unidad responsable."}`
}

func numeroJSONAsignacionPrueba(valor uint64) string {
	contenido, _ := json.Marshal(valor)
	return string(contenido)
}

func nuevaPeticionAsignacionPrueba(ruta, cuerpo string) *http.Request {
	peticion := httptest.NewRequest(http.MethodPost, ruta, strings.NewReader(cuerpo))
	peticion.Header.Set("Content-Type", "application/json")
	peticion.Header.Set("Accept", "application/json")
	return peticion
}

func reciboAsignacionHTTPPrueba(
	operacion ports.TipoOperacionAsignacion,
	version uint64,
) ports.ReciboAsignacion {
	return ports.ReciboAsignacion{
		Operacion:              operacion,
		OrganizacionRef:        contextoCanalAsignacionPrueba().OrganizacionRef,
		ExpedienteRef:          "expediente:asignacion:http:001",
		VersionAnterior:        version,
		VersionResultante:      version + 1,
		UnidadRef:              "unidad:destino:http:001",
		ResponsableRef:         "persona:responsable:http:001",
		ReciboRef:              "recibo:asignacion:http:001",
		NotificacionRef:        "notificacion:asignacion:http:001",
		BandejaRef:             "bandeja:asignacion:http:001",
		AuditoriaRef:           "auditoria:asignacion:http:001",
		EventoRef:              "evento:asignacion:http:001",
		ConcesionV3DecisionRef: "decision:asignacion:http:001",
		AmbitoIdempotenciaHMAC: "hmac-sha256:asignacion.ambito/v1:" +
			strings.Repeat("a", 64),
		HuellaPeticionHMAC: "hmac-sha256:asignacion.peticion/v1:" +
			strings.Repeat("b", 64),
		ConfirmadaEn: time.Date(2026, 8, 31, 12, 0, 0, 123000000, time.UTC),
	}
}

func comprobarRespuestaAsignacionMinimizada(
	t *testing.T,
	respuesta *httptest.ResponseRecorder,
) {
	t.Helper()
	if respuesta.Header().Get("Content-Type") !=
		"application/json; charset=utf-8" ||
		respuesta.Header().Get("Cache-Control") != "no-store, no-transform" ||
		respuesta.Header().Get("Set-Cookie") != "" {
		t.Fatalf("cabeceras inseguras: %#v", respuesta.Header())
	}
	cuerpo := respuesta.Body.String()
	for _, privado := range []string{
		"organizacion_ref", "unidad_ref", "responsable_ref",
		"notificacion_ref", "auditoria_ref", "evento_ref", "hmac-sha256",
	} {
		if strings.Contains(cuerpo, privado) {
			t.Fatalf("la respuesta expone %q: %s", privado, cuerpo)
		}
	}
}
