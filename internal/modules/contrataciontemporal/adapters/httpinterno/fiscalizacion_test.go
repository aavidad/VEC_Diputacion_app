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
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type autoridadFiscalizacionPrueba struct {
	contexto ContextoCanalFiscalizacion
	err      error
	llamadas int
}

func (a *autoridadFiscalizacionPrueba) ResolverContextoCanalFiscalizacion(
	context.Context,
) (ContextoCanalFiscalizacion, error) {
	a.llamadas++
	return a.contexto, a.err
}

type ejecutorFiscalizacionPrueba struct {
	recibo    ports.ReciboFiscalizacion
	err       error
	llamadas  int
	solicitud application.SolicitudRegistrarResultadoFiscalizacion
}

func (e *ejecutorFiscalizacionPrueba) RegistrarResultado(
	_ context.Context,
	solicitud application.SolicitudRegistrarResultadoFiscalizacion,
) (ports.ReciboFiscalizacion, error) {
	e.llamadas++
	e.solicitud = solicitud
	return e.recibo, e.err
}

func TestManejadorFiscalizacionRegistraYPublicaReciboTrazable(t *testing.T) {
	contexto := contextoCanalFiscalizacionPrueba()
	autoridad := &autoridadFiscalizacionPrueba{contexto: contexto}
	ejecutor := &ejecutorFiscalizacionPrueba{recibo: reciboFiscalizacionHTTPPrueba()}
	manejador, err := NuevoManejadorFiscalizacion(autoridad, ejecutor)
	if err != nil {
		t.Fatal(err)
	}
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(
		respuesta,
		nuevaPeticionFiscalizacionPrueba(cuerpoFiscalizacionPrueba("desfavorable", "Reparo sintetico.")),
	)

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
		solicitud.ExpedienteRef != "expediente:fiscalizacion:http:001" ||
		solicitud.VersionEsperada != 5 ||
		solicitud.ClaveIdempotencia != "11111111-2222-4333-8444-555555555555" ||
		string(solicitud.Resultado) != "desfavorable" ||
		solicitud.Observaciones != "Reparo sintetico." {
		t.Fatalf("solicitud mal traducida: %#v", solicitud)
	}
	var salida envoltorioReciboFiscalizacion
	if err := json.Unmarshal(respuesta.Body.Bytes(), &salida); err != nil {
		t.Fatalf("respuesta no JSON: %v", err)
	}
	if salida.Data.Esquema != esquemaReciboFiscalizacion ||
		salida.Data.Operacion != operacionFiscalizacion ||
		salida.Data.Resultado != "desfavorable" ||
		salida.Data.FaseResultante != "subsanacion_unidad" ||
		salida.Data.EstadoResultante != "incidencia" ||
		salida.Data.VersionResultante != 6 || salida.Data.ActorRef == "" ||
		salida.Data.UnidadRetornoRef == "" || salida.Data.ResponsableRetornoRef == "" {
		t.Fatalf("recibo no visible: %#v", salida.Data)
	}
	if respuesta.Header().Get("Content-Type") != "application/json; charset=utf-8" ||
		respuesta.Header().Get("Cache-Control") != "no-store, no-transform" ||
		respuesta.Header().Get("Set-Cookie") != "" {
		t.Fatalf("cabeceras inseguras: %#v", respuesta.Header())
	}
}

func TestManejadorFiscalizacionRechazaContratoAbiertoYResultadosInvalidos(t *testing.T) {
	base := cuerpoFiscalizacionPrueba("favorable", "")
	casos := map[string]string{
		"campo desconocido": strings.TrimSuffix(base, "}") +
			`,"actor_ref":"actor:inyectado:001"}`,
		"clave duplicada": `{"expediente_ref":"expediente:fiscalizacion:http:001",` +
			`"expediente_ref":"expediente:fiscalizacion:http:002",` +
			`"version_esperada":5,` +
			`"clave_idempotencia":"11111111-2222-4333-8444-555555555555",` +
			`"resultado":"favorable","observaciones":""}`,
		"resultado ajeno":   cuerpoFiscalizacionPrueba("reparo", "Reparo."),
		"sin observaciones": cuerpoFiscalizacionPrueba("favorable_con_observaciones", ""),
	}
	for nombre, cuerpo := range casos {
		t.Run(nombre, func(t *testing.T) {
			autoridad := &autoridadFiscalizacionPrueba{
				contexto: contextoCanalFiscalizacionPrueba(),
			}
			ejecutor := &ejecutorFiscalizacionPrueba{recibo: reciboFiscalizacionHTTPPrueba()}
			manejador, err := NuevoManejadorFiscalizacion(autoridad, ejecutor)
			if err != nil {
				t.Fatal(err)
			}
			respuesta := httptest.NewRecorder()
			manejador.ServeHTTP(respuesta, nuevaPeticionFiscalizacionPrueba(cuerpo))
			if (respuesta.Code != http.StatusBadRequest &&
				respuesta.Code != http.StatusUnprocessableEntity) ||
				autoridad.llamadas != 0 || ejecutor.llamadas != 0 {
				t.Fatalf(
					"estado=%d autoridad=%d ejecutor=%d",
					respuesta.Code, autoridad.llamadas, ejecutor.llamadas,
				)
			}
		})
	}
}

func TestManejadorFiscalizacionRechazaAutoridadEnCabecera(t *testing.T) {
	autoridad := &autoridadFiscalizacionPrueba{contexto: contextoCanalFiscalizacionPrueba()}
	ejecutor := &ejecutorFiscalizacionPrueba{recibo: reciboFiscalizacionHTTPPrueba()}
	manejador, err := NuevoManejadorFiscalizacion(autoridad, ejecutor)
	if err != nil {
		t.Fatal(err)
	}
	peticion := nuevaPeticionFiscalizacionPrueba(cuerpoFiscalizacionPrueba("favorable", ""))
	peticion.Header.Set("Cookie", "sesion=no-confiable")
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusBadRequest || autoridad.llamadas != 0 ||
		ejecutor.llamadas != 0 {
		t.Fatalf(
			"estado=%d autoridad=%d ejecutor=%d",
			respuesta.Code, autoridad.llamadas, ejecutor.llamadas,
		)
	}
}

func TestManejadorFiscalizacionRechazaFavorableConObservacionesComoContenido(t *testing.T) {
	autoridad := &autoridadFiscalizacionPrueba{contexto: contextoCanalFiscalizacionPrueba()}
	ejecutor := &ejecutorFiscalizacionPrueba{recibo: reciboFiscalizacionHTTPPrueba()}
	manejador, err := NuevoManejadorFiscalizacion(autoridad, ejecutor)
	if err != nil {
		t.Fatal(err)
	}
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(
		respuesta,
		nuevaPeticionFiscalizacionPrueba(cuerpoFiscalizacionPrueba("favorable", "Sobra.")),
	)
	if respuesta.Code != http.StatusUnprocessableEntity ||
		autoridad.llamadas != 0 || ejecutor.llamadas != 0 {
		t.Fatalf(
			"estado=%d autoridad=%d ejecutor=%d",
			respuesta.Code, autoridad.llamadas, ejecutor.llamadas,
		)
	}
}

func TestManejadorFiscalizacionNoPublicaReciboNoLigado(t *testing.T) {
	recibo := reciboFiscalizacionHTTPPrueba()
	recibo.Resultado = domain.ResultadoFiscalizacion("favorable")
	autoridad := &autoridadFiscalizacionPrueba{contexto: contextoCanalFiscalizacionPrueba()}
	ejecutor := &ejecutorFiscalizacionPrueba{recibo: recibo}
	manejador, err := NuevoManejadorFiscalizacion(autoridad, ejecutor)
	if err != nil {
		t.Fatal(err)
	}
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(
		respuesta,
		nuevaPeticionFiscalizacionPrueba(cuerpoFiscalizacionPrueba("desfavorable", "Reparo sintetico.")),
	)
	if respuesta.Code != http.StatusBadGateway {
		t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body.String())
	}
}

func TestManejadorFiscalizacionNoPublicaTransicionImposible(t *testing.T) {
	recibo := reciboFiscalizacionHTTPPrueba()
	recibo.FaseResultante = domain.FaseFiscalizacion
	recibo.EstadoResultante = domain.EstadoEnCurso
	autoridad := &autoridadFiscalizacionPrueba{contexto: contextoCanalFiscalizacionPrueba()}
	ejecutor := &ejecutorFiscalizacionPrueba{recibo: recibo}
	manejador, err := NuevoManejadorFiscalizacion(autoridad, ejecutor)
	if err != nil {
		t.Fatal(err)
	}
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(
		respuesta,
		nuevaPeticionFiscalizacionPrueba(
			cuerpoFiscalizacionPrueba("desfavorable", "Reparo sintetico."),
		),
	)
	if respuesta.Code != http.StatusBadGateway {
		t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body.String())
	}
}

func TestNuevoManejadorFiscalizacionRechazaNilYNilTipado(t *testing.T) {
	autoridad := &autoridadFiscalizacionPrueba{contexto: contextoCanalFiscalizacionPrueba()}
	ejecutor := &ejecutorFiscalizacionPrueba{}
	var autoridadNil *autoridadFiscalizacionPrueba
	var ejecutorNil *ejecutorFiscalizacionPrueba
	casos := []struct {
		nombre    string
		autoridad AutoridadContextoCanalFiscalizacion
		ejecutor  EjecutorFiscalizacion
	}{
		{"autoridad nil", nil, ejecutor},
		{"ejecutor nil", autoridad, nil},
		{"autoridad nil tipado", autoridadNil, ejecutor},
		{"ejecutor nil tipado", autoridad, ejecutorNil},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			manejador, err := NuevoManejadorFiscalizacion(caso.autoridad, caso.ejecutor)
			if manejador != nil || !errors.Is(err, ErrManejadorFiscalizacionInvalido) {
				t.Fatalf("manejador=%#v err=%v", manejador, err)
			}
		})
	}
}

func contextoCanalFiscalizacionPrueba() ContextoCanalFiscalizacion {
	return ContextoCanalFiscalizacion{
		AutenticacionRef: "aut_aaaaaaaaaaaaaaaaaaaaaaaa",
		SesionRef:        "ses_bbbbbbbbbbbbbbbbbbbbbbbb",
		PerfilRef:        "prf_cccccccccccccccccccccccc",
		OrganizacionRef:  "organizacion:dipgra:fiscalizacion:001",
	}
}

func cuerpoFiscalizacionPrueba(resultado, observaciones string) string {
	contenido, err := json.Marshal(fiscalizacionEntradaJSON{
		ExpedienteRef:     "expediente:fiscalizacion:http:001",
		VersionEsperada:   punteroUint64FiscalizacionPrueba(5),
		ClaveIdempotencia: "11111111-2222-4333-8444-555555555555",
		Resultado:         resultado,
		Observaciones:     observaciones,
	})
	if err != nil {
		panic(err)
	}
	return string(contenido)
}

func nuevaPeticionFiscalizacionPrueba(cuerpo string) *http.Request {
	peticion := httptest.NewRequest(
		http.MethodPost,
		RutaResultadosFiscalizacion,
		strings.NewReader(cuerpo),
	)
	peticion.Header.Set("Content-Type", "application/json; charset=utf-8")
	peticion.Header.Set("Accept", "application/json")
	return peticion
}

func reciboFiscalizacionHTTPPrueba() ports.ReciboFiscalizacion {
	return ports.ReciboFiscalizacion{
		Operacion:             operacionFiscalizacion,
		OrganizacionRef:       contextoCanalFiscalizacionPrueba().OrganizacionRef,
		ExpedienteRef:         "expediente:fiscalizacion:http:001",
		VersionAnterior:       5,
		VersionResultante:     6,
		Resultado:             domain.ResultadoFiscalizacion("desfavorable"),
		FaseResultante:        domain.FaseSubsanacionUnidad,
		EstadoResultante:      domain.EstadoIncidencia,
		ReciboRef:             "recibo:fiscalizacion:http:001",
		AuditoriaRef:          "auditoria:fiscalizacion:http:001",
		EventoRef:             "evento:fiscalizacion:http:001",
		ActorRef:              "actor:intervencion:http:001",
		UnidadRetornoRef:      "unidad:retorno:http:001",
		ResponsableRetornoRef: "responsable:retorno:http:001",
		RegistradaEn: time.Date(
			2026, 9, 4, 19, 0, 0, 123000000, time.UTC,
		),
	}
}

func punteroUint64FiscalizacionPrueba(valor uint64) *uint64 {
	return &valor
}
