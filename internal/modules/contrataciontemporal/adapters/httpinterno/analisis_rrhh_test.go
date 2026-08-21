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

type autoridadAnalisisRRHHPrueba struct {
	contexto ContextoCanalAnalisisRRHH
	err      error
	antes    func()
	llamadas int
}

func (a *autoridadAnalisisRRHHPrueba) ResolverContextoCanalAnalisisRRHH(
	context.Context,
) (ContextoCanalAnalisisRRHH, error) {
	a.llamadas++
	if a.antes != nil {
		a.antes()
	}
	return a.contexto, a.err
}

type ejecutorAnalisisRRHHPrueba struct {
	recibo              ports.ReciboOperacionAnalisis
	err                 error
	antes               func()
	registros           int
	rectificaciones     int
	solicitudRegistro   application.SolicitudRegistrarAnalisis
	solicitudRectificar application.SolicitudRectificarAnalisis
}

func (e *ejecutorAnalisisRRHHPrueba) Registrar(
	_ context.Context,
	solicitud application.SolicitudRegistrarAnalisis,
) (ports.ReciboOperacionAnalisis, error) {
	e.registros++
	e.solicitudRegistro = solicitud
	if e.antes != nil {
		e.antes()
	}
	return e.recibo, e.err
}

func (e *ejecutorAnalisisRRHHPrueba) Rectificar(
	_ context.Context,
	solicitud application.SolicitudRectificarAnalisis,
) (ports.ReciboOperacionAnalisis, error) {
	e.rectificaciones++
	e.solicitudRectificar = solicitud
	if e.antes != nil {
		e.antes()
	}
	return e.recibo, e.err
}

func TestManejadorAnalisisRRHHTraduceRegistroSinAceptarAutoridadHTTP(
	t *testing.T,
) {
	contexto := contextoCanalAnalisisRRHHPrueba()
	autoridad := &autoridadAnalisisRRHHPrueba{contexto: contexto}
	ejecutor := &ejecutorAnalisisRRHHPrueba{
		recibo: reciboAnalisisRRHHPrueba(
			ports.OperacionRegistrarAnalisis,
			1,
		),
	}
	manejador, err := NuevoManejadorAnalisisRRHH(autoridad, ejecutor)
	if err != nil {
		t.Fatal(err)
	}
	peticion := nuevaPeticionAnalisisRRHHPrueba(
		RutaRegistroAnalisisRRHH,
		cuerpoRegistroAnalisisRRHHPrueba(),
	)
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, peticion)

	if respuesta.Code != http.StatusCreated || autoridad.llamadas != 1 ||
		ejecutor.registros != 1 || ejecutor.rectificaciones != 0 {
		t.Fatalf(
			"resultado inesperado: estado=%d autoridad=%d registro=%d rectificacion=%d cuerpo=%s",
			respuesta.Code,
			autoridad.llamadas,
			ejecutor.registros,
			ejecutor.rectificaciones,
			respuesta.Body.String(),
		)
	}
	solicitud := ejecutor.solicitudRegistro
	if solicitud.AutenticacionRef != contexto.AutenticacionRef ||
		solicitud.SesionRef != contexto.SesionRef ||
		solicitud.PerfilRef != contexto.PerfilRef ||
		solicitud.OrganizacionRef != contexto.OrganizacionRef ||
		solicitud.ExpedienteRef != "expediente:analisis:http:001" ||
		solicitud.VersionEsperada != 1 ||
		solicitud.ClaveIdempotencia !=
			"11111111-2222-4333-8444-555555555555" ||
		solicitud.ArtefactoRef != "artefacto:analisis:http:001" ||
		solicitud.DatosFuncionales.Validar() != nil {
		t.Fatalf("orden tipada incorrecta: %#v", solicitud)
	}
	if solicitud.DatosFuncionales.ModalidadClave != "interinidad" ||
		solicitud.DatosFuncionales.CategoriaRef != "categoria:tecnico:001" ||
		solicitud.DatosFuncionales.GrupoSubgrupo != "A2" ||
		solicitud.DatosFuncionales.CausaClave != "sustitucion" ||
		solicitud.DatosFuncionales.PorcentajeJornada != 7_500 ||
		solicitud.DatosFuncionales.EntradaRC.Referencia !=
			"entrada:rc:http:001" {
		t.Fatalf("datos funcionales alterados: %#v", solicitud.DatosFuncionales)
	}
	comprobarRespuestaSeguraAnalisisRRHH(t, respuesta)
	for _, privado := range []string{
		contexto.AutenticacionRef,
		contexto.SesionRef,
		contexto.PerfilRef,
		contexto.OrganizacionRef,
		"auditoria:analisis:http:001",
		strings.Repeat("a", 64),
	} {
		if strings.Contains(respuesta.Body.String(), privado) {
			t.Fatalf("la respuesta expone dato interno %q", privado)
		}
	}
	var salida envoltorioReciboAnalisisRRHH
	if err := json.Unmarshal(respuesta.Body.Bytes(), &salida); err != nil {
		t.Fatal(err)
	}
	if salida.Data.Esquema != esquemaReciboAnalisisRRHH ||
		salida.Data.Operacion != "registrar" ||
		salida.Data.ExpedienteRef != "expediente:analisis:http:001" ||
		salida.Data.VersionResultante != 2 ||
		salida.Data.ReciboRef != "recibo:analisis:http:001" {
		t.Fatalf("proyeccion opaca incorrecta: %#v", salida.Data)
	}
}

func TestManejadorAnalisisRRHHTraduceRectificacionNominal(t *testing.T) {
	contexto := contextoCanalAnalisisRRHHPrueba()
	autoridad := &autoridadAnalisisRRHHPrueba{contexto: contexto}
	ejecutor := &ejecutorAnalisisRRHHPrueba{
		recibo: reciboAnalisisRRHHPrueba(
			ports.OperacionRectificarAnalisis,
			2,
		),
	}
	manejador, err := NuevoManejadorAnalisisRRHH(autoridad, ejecutor)
	if err != nil {
		t.Fatal(err)
	}
	peticion := nuevaPeticionAnalisisRRHHPrueba(
		RutaRectificacionAnalisisRRHH,
		cuerpoRectificacionAnalisisRRHHPrueba(),
	)
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, peticion)

	if respuesta.Code != http.StatusCreated || ejecutor.registros != 0 ||
		ejecutor.rectificaciones != 1 {
		t.Fatalf("rectificacion inesperada: %d %s", respuesta.Code, respuesta.Body)
	}
	solicitud := ejecutor.solicitudRectificar
	if solicitud.AutenticacionRef != contexto.AutenticacionRef ||
		solicitud.OrganizacionRef != contexto.OrganizacionRef ||
		solicitud.VersionEsperada != 2 ||
		solicitud.MotivoRectificacionClave != "ajuste_jornada" {
		t.Fatalf("orden de rectificacion incorrecta: %#v", solicitud)
	}
	comprobarRespuestaSeguraAnalisisRRHH(t, respuesta)
}

func TestNuevoManejadorAnalisisRRHHFallaCerrado(t *testing.T) {
	var autoridadNula *autoridadAnalisisRRHHPrueba
	var ejecutorNulo *ejecutorAnalisisRRHHPrueba
	validaAutoridad := &autoridadAnalisisRRHHPrueba{
		contexto: contextoCanalAnalisisRRHHPrueba(),
	}
	validoEjecutor := &ejecutorAnalisisRRHHPrueba{}
	casos := []struct {
		nombre    string
		autoridad AutoridadContextoCanalAnalisisRRHH
		ejecutor  EjecutorAnalisisRRHH
	}{
		{"autoridad ausente", nil, validoEjecutor},
		{"autoridad nula tipada", autoridadNula, validoEjecutor},
		{"ejecutor ausente", validaAutoridad, nil},
		{"ejecutor nulo tipado", validaAutoridad, ejecutorNulo},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if _, err := NuevoManejadorAnalisisRRHH(
				caso.autoridad,
				caso.ejecutor,
			); !errors.Is(err, ErrManejadorAnalisisRRHHInvalido) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func contextoCanalAnalisisRRHHPrueba() ContextoCanalAnalisisRRHH {
	return ContextoCanalAnalisisRRHH{
		AutenticacionRef: "aut_aaaaaaaaaaaaaaaaaaaaaaaa",
		SesionRef:        "ses_bbbbbbbbbbbbbbbbbbbbbbbb",
		PerfilRef:        "prf_cccccccccccccccccccccccc",
		OrganizacionRef:  "organizacion:dipgra:http:001",
	}
}

func cuerpoRegistroAnalisisRRHHPrueba() string {
	return `{
 "expediente_ref":"expediente:analisis:http:001",
 "version_esperada":1,
 "clave_idempotencia":"11111111-2222-4333-8444-555555555555",
 "artefacto_ref":"artefacto:analisis:http:001",
 "analisis":{
  "modalidad_clave":"interinidad",
  "categoria_ref":"categoria:tecnico:001",
  "grupo_subgrupo":"A2",
  "causa_clave":"sustitucion",
  "periodo":{"inicio":"2026-09-01T00:00:00Z","fin":"2027-02-28T00:00:00Z"},
  "porcentaje_jornada":7500,
  "entrada_rc":{"referencia":"entrada:rc:http:001","huella_sha256":"` +
		strings.Repeat("9", 64) + `"}
 }
}`
}

func cuerpoRectificacionAnalisisRRHHPrueba() string {
	return strings.Replace(
		strings.Replace(
			cuerpoRegistroAnalisisRRHHPrueba(),
			`"version_esperada":1`,
			`"version_esperada":2`,
			1,
		),
		`"analisis":`,
		`"motivo_rectificacion_clave":"ajuste_jornada","analisis":`,
		1,
	)
}

func nuevaPeticionAnalisisRRHHPrueba(ruta, cuerpo string) *http.Request {
	peticion := httptest.NewRequest(http.MethodPost, ruta, strings.NewReader(cuerpo))
	peticion.Header.Set("Content-Type", "application/json; charset=utf-8")
	peticion.Header.Set("Accept", "application/json")
	return peticion
}

func reciboAnalisisRRHHPrueba(
	operacion ports.TipoOperacionAnalisis,
	version uint64,
) ports.ReciboOperacionAnalisis {
	return ports.ReciboOperacionAnalisis{
		Operacion:              operacion,
		OrganizacionRef:        "organizacion:dipgra:http:001",
		ExpedienteRef:          "expediente:analisis:http:001",
		VersionAnterior:        version,
		VersionResultante:      version + 1,
		SecuenciaActuacion:     version + 1,
		ArtefactoRef:           "artefacto:analisis:http:001",
		ArtefactoHuellaSHA256:  strings.Repeat("a", 64),
		ReciboRef:              "recibo:analisis:http:001",
		AuditoriaRef:           "auditoria:analisis:http:001",
		EventoRef:              "evento:analisis:http:001",
		ConsumoFuentesRef:      "consumo:fuentes:http:001",
		HuellaConsumoFuentes:   strings.Repeat("b", 64),
		ConcesionV3DecisionRef: "decision:v3:analisis:http:001",
		HuellaSemanticaHMAC: "hmac-sha256:analisis/v1:" +
			strings.Repeat("c", 64),
		AmbitoConsultaHMAC: "hmac-sha256:analisis/v1:" +
			strings.Repeat("d", 64),
		HuellaConsultaHMAC: "hmac-sha256:analisis/v1:" +
			strings.Repeat("e", 64),
		ConfirmadaEn: time.Date(
			2026,
			time.August,
			21,
			10,
			0,
			0,
			123_000_000,
			time.UTC,
		),
	}
}

func comprobarRespuestaSeguraAnalisisRRHH(
	t *testing.T,
	respuesta *httptest.ResponseRecorder,
) {
	t.Helper()
	if respuesta.Header().Get("Cache-Control") != "no-store, no-transform" ||
		respuesta.Header().Get("Content-Type") !=
			"application/json; charset=utf-8" ||
		respuesta.Header().Values("Set-Cookie") != nil {
		t.Fatalf("cabeceras inseguras: %#v", respuesta.Header())
	}
}
