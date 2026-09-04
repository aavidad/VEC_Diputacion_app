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

type autoridadInformeJuridicoPrueba struct {
	contexto ContextoCanalInformeJuridico
	err      error
	llamadas int
}

func (a *autoridadInformeJuridicoPrueba) ResolverContextoCanalInformeJuridico(
	context.Context,
) (ContextoCanalInformeJuridico, error) {
	a.llamadas++
	return a.contexto, a.err
}

type ejecutorInformeJuridicoPrueba struct {
	recibo    ports.ReciboInformeJuridico
	err       error
	llamadas  int
	solicitud application.SolicitudEmitirInformeJuridico
}

func (e *ejecutorInformeJuridicoPrueba) Emitir(
	_ context.Context,
	solicitud application.SolicitudEmitirInformeJuridico,
) (ports.ReciboInformeJuridico, error) {
	e.llamadas++
	e.solicitud = solicitud
	return e.recibo, e.err
}

func TestManejadorInformeJuridicoEmiteReciboVisibleConAutoridadConfiable(
	t *testing.T,
) {
	contexto := contextoCanalInformeJuridicoPrueba()
	autoridad := &autoridadInformeJuridicoPrueba{contexto: contexto}
	ejecutor := &ejecutorInformeJuridicoPrueba{
		recibo: reciboInformeJuridicoHTTPPrueba(),
	}
	manejador, err := NuevoManejadorInformeJuridico(autoridad, ejecutor)
	if err != nil {
		t.Fatal(err)
	}
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, nuevaPeticionInformeJuridicoPrueba(cuerpoInformeJuridicoPrueba()))

	if respuesta.Code != http.StatusCreated || autoridad.llamadas != 1 ||
		ejecutor.llamadas != 1 {
		t.Fatalf(
			"estado=%d autoridad=%d ejecutor=%d cuerpo=%s",
			respuesta.Code, autoridad.llamadas, ejecutor.llamadas,
			respuesta.Body.String(),
		)
	}
	solicitud := ejecutor.solicitud
	if solicitud.AutenticacionRef != contexto.AutenticacionRef ||
		solicitud.SesionRef != contexto.SesionRef ||
		solicitud.PerfilRef != contexto.PerfilRef ||
		solicitud.OrganizacionRef != contexto.OrganizacionRef ||
		solicitud.ExpedienteRef != "expediente:informe:http:001" ||
		solicitud.VersionEsperada != 5 ||
		solicitud.ClaveIdempotencia != "11111111-2222-4333-8444-555555555555" {
		t.Fatalf("solicitud mal traducida: %#v", solicitud)
	}
	var salida envoltorioReciboInformeJuridico
	if err := json.Unmarshal(respuesta.Body.Bytes(), &salida); err != nil {
		t.Fatalf("respuesta no JSON: %v", err)
	}
	if salida.Data.Esquema != esquemaReciboInformeJuridico ||
		salida.Data.Operacion != "preparar" ||
		salida.Data.VersionResultante != 6 ||
		salida.Data.ReciboRef != "recibo:informe:http:001" ||
		!strings.Contains(salida.Data.ContenidoDesarrollo, "DOCUMENTO DE DESARROLLO") {
		t.Fatalf("recibo no visible: %#v", salida.Data)
	}
	if respuesta.Header().Get("Content-Type") != "application/json; charset=utf-8" ||
		respuesta.Header().Get("Cache-Control") != "no-store, no-transform" ||
		respuesta.Header().Get("Set-Cookie") != "" {
		t.Fatalf("cabeceras inseguras: %#v", respuesta.Header())
	}
}

func TestManejadorInformeJuridicoRechazaJSONAbiertoAntesDeLaAutoridad(
	t *testing.T,
) {
	casos := map[string]string{
		"campo desconocido": strings.TrimSuffix(cuerpoInformeJuridicoPrueba(), "}") +
			`,"perfil_ref":"perfil:inyectado:001"}`,
		"clave duplicada": `{"expediente_ref":"expediente:informe:http:001",` +
			`"expediente_ref":"expediente:informe:http:002",` +
			`"version_esperada":5,` +
			`"clave_idempotencia":"11111111-2222-4333-8444-555555555555"}`,
	}
	for nombre, cuerpo := range casos {
		t.Run(nombre, func(t *testing.T) {
			autoridad := &autoridadInformeJuridicoPrueba{
				contexto: contextoCanalInformeJuridicoPrueba(),
			}
			ejecutor := &ejecutorInformeJuridicoPrueba{
				recibo: reciboInformeJuridicoHTTPPrueba(),
			}
			manejador, err := NuevoManejadorInformeJuridico(autoridad, ejecutor)
			if err != nil {
				t.Fatal(err)
			}
			respuesta := httptest.NewRecorder()
			manejador.ServeHTTP(respuesta, nuevaPeticionInformeJuridicoPrueba(cuerpo))
			if respuesta.Code != http.StatusBadRequest ||
				autoridad.llamadas != 0 || ejecutor.llamadas != 0 {
				t.Fatalf(
					"estado=%d autoridad=%d ejecutor=%d",
					respuesta.Code, autoridad.llamadas, ejecutor.llamadas,
				)
			}
		})
	}
}

func TestManejadorInformeJuridicoRechazaAutoridadEnCabecera(
	t *testing.T,
) {
	autoridad := &autoridadInformeJuridicoPrueba{
		contexto: contextoCanalInformeJuridicoPrueba(),
	}
	ejecutor := &ejecutorInformeJuridicoPrueba{
		recibo: reciboInformeJuridicoHTTPPrueba(),
	}
	manejador, err := NuevoManejadorInformeJuridico(autoridad, ejecutor)
	if err != nil {
		t.Fatal(err)
	}
	peticion := nuevaPeticionInformeJuridicoPrueba(cuerpoInformeJuridicoPrueba())
	peticion.Header.Set("Cookie", "sesion=no-confiable")
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusBadRequest ||
		autoridad.llamadas != 0 || ejecutor.llamadas != 0 {
		t.Fatalf(
			"estado=%d autoridad=%d ejecutor=%d",
			respuesta.Code, autoridad.llamadas, ejecutor.llamadas,
		)
	}
}

func TestManejadorInformeJuridicoNoPublicaReciboNoConfiable(t *testing.T) {
	recibo := reciboInformeJuridicoHTTPPrueba()
	recibo.ContenidoDesarrollo = "Informe definitivo sin marca."
	autoridad := &autoridadInformeJuridicoPrueba{
		contexto: contextoCanalInformeJuridicoPrueba(),
	}
	ejecutor := &ejecutorInformeJuridicoPrueba{recibo: recibo}
	manejador, err := NuevoManejadorInformeJuridico(autoridad, ejecutor)
	if err != nil {
		t.Fatal(err)
	}
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, nuevaPeticionInformeJuridicoPrueba(cuerpoInformeJuridicoPrueba()))
	if respuesta.Code != http.StatusBadGateway {
		t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body.String())
	}
}

func TestNuevoManejadorInformeJuridicoRechazaNilYNilTipado(t *testing.T) {
	autoridad := &autoridadInformeJuridicoPrueba{
		contexto: contextoCanalInformeJuridicoPrueba(),
	}
	ejecutor := &ejecutorInformeJuridicoPrueba{}
	var autoridadNil *autoridadInformeJuridicoPrueba
	var ejecutorNil *ejecutorInformeJuridicoPrueba
	casos := []struct {
		nombre    string
		autoridad AutoridadContextoCanalInformeJuridico
		ejecutor  EjecutorInformeJuridico
	}{
		{"autoridad nil", nil, ejecutor},
		{"ejecutor nil", autoridad, nil},
		{"autoridad nil tipado", autoridadNil, ejecutor},
		{"ejecutor nil tipado", autoridad, ejecutorNil},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			manejador, err := NuevoManejadorInformeJuridico(
				caso.autoridad,
				caso.ejecutor,
			)
			if manejador != nil ||
				!errors.Is(err, ErrManejadorInformeJuridicoInvalido) {
				t.Fatalf("manejador=%#v err=%v", manejador, err)
			}
		})
	}
}

func contextoCanalInformeJuridicoPrueba() ContextoCanalInformeJuridico {
	return ContextoCanalInformeJuridico{
		AutenticacionRef: "aut_aaaaaaaaaaaaaaaaaaaaaaaa",
		SesionRef:        "ses_bbbbbbbbbbbbbbbbbbbbbbbb",
		PerfilRef:        "prf_cccccccccccccccccccccccc",
		OrganizacionRef:  "organizacion:dipgra:http:001",
	}
}

func cuerpoInformeJuridicoPrueba() string {
	return `{"expediente_ref":"expediente:informe:http:001",` +
		`"version_esperada":5,` +
		`"clave_idempotencia":"11111111-2222-4333-8444-555555555555"}`
}

func nuevaPeticionInformeJuridicoPrueba(cuerpo string) *http.Request {
	peticion := httptest.NewRequest(
		http.MethodPost,
		RutaPreparacionesInformeJuridico,
		strings.NewReader(cuerpo),
	)
	peticion.Header.Set("Content-Type", "application/json")
	peticion.Header.Set("Accept", "application/json")
	return peticion
}

func reciboInformeJuridicoHTTPPrueba() ports.ReciboInformeJuridico {
	return ports.ReciboInformeJuridico{
		Operacion:              "preparar",
		OrganizacionRef:        contextoCanalInformeJuridicoPrueba().OrganizacionRef,
		ExpedienteRef:          "expediente:informe:http:001",
		VersionAnterior:        5,
		VersionResultante:      6,
		InformeRef:             "informe:juridico:http:001",
		DocumentoRef:           "documento:informe:http:001",
		VersionDocumento:       1,
		Formato:                ports.FormatoInformeJuridicoDesarrollo,
		Nombre:                 "informe-juridico-desarrollo.txt",
		HuellaDocumentoSHA256:  strings.Repeat("a", 64),
		HuellaBorradorSHA256:   strings.Repeat("b", 64),
		ReciboRef:              "recibo:informe:http:001",
		AuditoriaRef:           "auditoria:informe:http:001",
		EventoRef:              "evento:informe:http:001",
		ConcesionV3DecisionRef: "decision:informe:http:001",
		AmbitoIdempotenciaHMAC: "hmac-sha256:" +
			ports.DominioAmbitoIdempotenciaInformeJuridico + "/v1:" +
			strings.Repeat("c", 64),
		HuellaPeticionHMAC: "hmac-sha256:" +
			ports.DominioHuellaPeticionInformeJuridico + "/v1:" +
			strings.Repeat("d", 64),
		ContenidoDesarrollo: "DOCUMENTO DE DESARROLLO - SIN FIRMA NI VALIDEZ JURIDICA\n" +
			"Contenido sintetico del informe.",
		ConfirmadaEn: time.Date(2026, 9, 4, 10, 0, 0, 123000000, time.UTC),
	}
}
