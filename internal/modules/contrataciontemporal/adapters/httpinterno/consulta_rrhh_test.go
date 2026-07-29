package httpinterno

import (
	"bytes"
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

type consultorCuadroRRHHPrueba struct {
	pagina      ports.PaginaCuadroRRHH
	err         error
	llamadas    int
	solicitud   ports.SolicitudCuadroRRHH
	alConsultar func(context.Context)
}

func (c *consultorCuadroRRHHPrueba) Consultar(
	ctx context.Context,
	solicitud ports.SolicitudCuadroRRHH,
) (ports.PaginaCuadroRRHH, error) {
	c.llamadas++
	c.solicitud = solicitud
	if c.alConsultar != nil {
		c.alConsultar(ctx)
	}
	return c.pagina, c.err
}

type consultorDetalleRRHHPrueba struct {
	detalle     ports.DetalleExpedienteRRHH
	err         error
	llamadas    int
	solicitud   ports.SolicitudDetalleRRHH
	alConsultar func(context.Context)
}

func (c *consultorDetalleRRHHPrueba) Consultar(
	ctx context.Context,
	solicitud ports.SolicitudDetalleRRHH,
) (ports.DetalleExpedienteRRHH, error) {
	c.llamadas++
	c.solicitud = solicitud
	if c.alConsultar != nil {
		c.alConsultar(ctx)
	}
	return c.detalle, c.err
}

func cuerpoCuadroRRHHPrueba() string {
	return `{"filtros":{"texto":"2026/CT","estado_clave":"en_curso",` +
		`"fase_clave":"analisis"},"paginacion":{"limite":50,"cursor":""}}`
}

func cuerpoDetalleRRHHPrueba() string {
	return `{"expediente_ref":"expediente:ct:0001","version_observada":3}`
}

func nuevaPeticionConsultaRRHHPrueba(
	ruta string,
	cuerpo string,
) *http.Request {
	peticion := httptest.NewRequest(
		http.MethodPost,
		ruta,
		bytes.NewBufferString(cuerpo),
	)
	peticion.Header.Set("Content-Type", "application/json")
	peticion.Header.Set("Accept", "application/json")
	return peticion
}

func resumenRRHHPrueba() ports.ResumenExpedienteRRHH {
	return ports.ResumenExpedienteRRHH{
		ExpedienteRef:   "expediente:ct:0001",
		OrganizacionRef: "organizacion:rrhh:secreta",
		NumeroVisible:   "2026/CT-0001",
		Version:         3,
		FlujoRef:        "flujo:ct:ordinario",
		FlujoVersion:    2,
		FlujoHuella:     strings.Repeat("a", 64),
		FaseClave:       domain.ClaveFase("analisis"),
		EstadoClave:     domain.EstadoEnCurso,
		CentroRef:       "centro:dipgra:001",
		CategoriaRef:    "categoria:auxiliar:001",
		ModalidadClave:  domain.ClaveCatalogo("interinidad"),
		CreadoEn:        time.Date(2026, 7, 20, 8, 0, 0, 0, time.UTC),
		ActualizadoEn:   time.Date(2026, 7, 21, 9, 30, 0, 123000000, time.UTC),
	}
}

func paginaRRHHPrueba() ports.PaginaCuadroRRHH {
	return ports.PaginaCuadroRRHH{
		GeneradaEn:  time.Date(2026, 7, 21, 10, 0, 0, 0, time.UTC),
		Expedientes: []ports.ResumenExpedienteRRHH{resumenRRHHPrueba()},
	}
}

func detalleRRHHPrueba() ports.DetalleExpedienteRRHH {
	resumen := resumenRRHHPrueba()
	return ports.DetalleExpedienteRRHH{
		Resumen: resumen,
		Solicitud: ports.SolicitudOperativaRRHH{
			GrupoSubgrupo: "C2",
			MotivoClave:   domain.ClaveCatalogo("sustitucion"),
			PeriodoInicio: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
			PeriodoFin:    time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		},
		Hitos: []ports.HitoExpedienteRRHH{
			{
				Secuencia: 1, VersionExpediente: 1,
				AccionClave: "alta", RealizadaEn: resumen.CreadoEn,
				FaseDestino: "solicitud", EstadoOrigen: domain.EstadoPendiente,
				EstadoDestino: domain.EstadoEnCurso,
			},
			{
				Secuencia: 2, VersionExpediente: 2, AccionClave: "analizar",
				RealizadaEn: resumen.CreadoEn.Add(time.Hour),
				FaseOrigen:  "solicitud", FaseDestino: "analisis",
				EstadoOrigen:  domain.EstadoEnCurso,
				EstadoDestino: domain.EstadoEnCurso,
			},
			{
				Secuencia: 3, VersionExpediente: 3, AccionClave: "actualizar",
				RealizadaEn: resumen.ActualizadoEn,
				FaseOrigen:  "analisis", FaseDestino: "analisis",
				EstadoOrigen:  domain.EstadoEnCurso,
				EstadoDestino: domain.EstadoEnCurso,
			},
		},
	}
}

func TestManejadorConsultaCuadroRRHHExponeContratoMinimo(t *testing.T) {
	consultor := &consultorCuadroRRHHPrueba{pagina: paginaRRHHPrueba()}
	manejador, err := NuevoManejadorConsultaCuadroRRHH(consultor)
	if err != nil {
		t.Fatal(err)
	}
	respuesta := httptest.NewRecorder()
	respuesta.Header().Set("Set-Cookie", "forjada=1")
	manejador.ServeHTTP(
		respuesta,
		nuevaPeticionConsultaRRHHPrueba(
			RutaConsultaCuadroRRHH,
			cuerpoCuadroRRHHPrueba(),
		),
	)
	if respuesta.Code != http.StatusOK {
		t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body.String())
	}
	if consultor.llamadas != 1 ||
		consultor.solicitud.Texto() != "2026/CT" ||
		consultor.solicitud.EstadoClave() != domain.EstadoEnCurso ||
		consultor.solicitud.FaseClave() != "analisis" ||
		consultor.solicitud.Limite() != 50 ||
		consultor.solicitud.Cursor() != "" {
		t.Fatalf("intención alterada: %#v", consultor.solicitud)
	}
	var salida struct {
		Data struct {
			Esquema     string            `json:"esquema"`
			Expedientes []json.RawMessage `json:"expedientes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respuesta.Body.Bytes(), &salida); err != nil {
		t.Fatal(err)
	}
	if salida.Data.Esquema != esquemaConsultaCuadroRRHH ||
		len(salida.Data.Expedientes) != 1 {
		t.Fatalf("salida inesperada: %s", respuesta.Body)
	}
	cuerpo := respuesta.Body.String()
	for _, prohibido := range []string{
		"organizacion:rrhh:secreta", "organizacion_ref",
		"lectura", "auditoria", "sesion", "actor", "perfil",
	} {
		if strings.Contains(strings.ToLower(cuerpo), prohibido) {
			t.Fatalf("se publicó %q: %s", prohibido, cuerpo)
		}
	}
	comprobarCabecerasConsultaRRHH(t, respuesta)
}

func TestManejadorConsultaDetalleRRHHExponeContratoMinimo(t *testing.T) {
	consultor := &consultorDetalleRRHHPrueba{
		detalle: detalleRRHHPrueba(),
	}
	manejador, err := NuevoManejadorConsultaDetalleRRHH(consultor)
	if err != nil {
		t.Fatal(err)
	}
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(
		respuesta,
		nuevaPeticionConsultaRRHHPrueba(
			RutaConsultaDetalleRRHH,
			cuerpoDetalleRRHHPrueba(),
		),
	)
	if respuesta.Code != http.StatusOK {
		t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body.String())
	}
	if consultor.llamadas != 1 ||
		consultor.solicitud.ExpedienteRef() != "expediente:ct:0001" ||
		consultor.solicitud.VersionObservada() != 3 {
		t.Fatalf("intención alterada: %#v", consultor.solicitud)
	}
	var salida struct {
		Data struct {
			Esquema string            `json:"esquema"`
			Hitos   []json.RawMessage `json:"hitos"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respuesta.Body.Bytes(), &salida); err != nil {
		t.Fatal(err)
	}
	if salida.Data.Esquema != esquemaConsultaDetalleRRHH ||
		salida.Data.Hitos == nil || len(salida.Data.Hitos) != 3 {
		t.Fatalf("listas o esquema no canónicos: %s", respuesta.Body)
	}
	for _, prohibido := range []string{
		"organizacion:rrhh:secreta", "organizacion_ref",
		"lectura", "auditoria", "sesion", "actor", "perfil",
	} {
		if strings.Contains(strings.ToLower(respuesta.Body.String()), prohibido) {
			t.Fatalf("se publicó %q: %s", prohibido, respuesta.Body)
		}
	}
	comprobarCabecerasConsultaRRHH(t, respuesta)
}

func TestConstructoresManejadorConsultaRRHHCierranDependenciasNulas(
	t *testing.T,
) {
	if _, err := NuevoManejadorConsultaCuadroRRHH(nil); !errors.Is(
		err, ErrManejadorConsultaRRHHInvalido,
	) {
		t.Fatalf("cuadro nil: %v", err)
	}
	if _, err := NuevoManejadorConsultaCuadroRRHH(
		(*consultorCuadroRRHHPrueba)(nil),
	); !errors.Is(err, ErrManejadorConsultaRRHHInvalido) {
		t.Fatalf("cuadro typed nil: %v", err)
	}
	if _, err := NuevoManejadorConsultaDetalleRRHH(
		(*consultorDetalleRRHHPrueba)(nil),
	); !errors.Is(err, ErrManejadorConsultaRRHHInvalido) {
		t.Fatalf("detalle typed nil: %v", err)
	}
}

func TestManejadorConsultaRRHHClasificaErroresTipadosSinOraculo(t *testing.T) {
	casos := []struct {
		nombre string
		err    error
		estado int
		codigo string
	}{
		{"entrada", application.ErrSolicitudConsultaRRHHInvalida, 422, "contenido_no_valido"},
		{"no observable", application.ErrConsultaRRHHNoObservable, 404, "recurso_no_encontrado"},
		{"no confiable", application.ErrResultadoConsultaRRHHNoConfiable, 502, "resultado_no_confiable"},
		{"no disponible", application.ErrConsultaRRHHNoDisponible, 503, "servicio_no_disponible"},
		{"servicio inválido", application.ErrServicioConsultaRRHHInvalido, 503, "servicio_no_disponible"},
		{"cancelada", context.Canceled, 408, "peticion_cancelada"},
		{"plazo", context.DeadlineExceeded, 504, "plazo_agotado"},
		{"desconocido", errors.New("CAUSA_PRIVADA_123"), 500, "error_interno"},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			consultor := &consultorCuadroRRHHPrueba{err: caso.err}
			manejador, err := NuevoManejadorConsultaCuadroRRHH(consultor)
			if err != nil {
				t.Fatal(err)
			}
			respuesta := httptest.NewRecorder()
			manejador.ServeHTTP(
				respuesta,
				nuevaPeticionConsultaRRHHPrueba(
					RutaConsultaCuadroRRHH,
					cuerpoCuadroRRHHPrueba(),
				),
			)
			if respuesta.Code != caso.estado ||
				!strings.Contains(respuesta.Body.String(), `"codigo":"`+caso.codigo+`"`) {
				t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body)
			}
			for _, privado := range []string{
				"CAUSA_PRIVADA_123", "contratacion temporal:", "denegad",
			} {
				if strings.Contains(respuesta.Body.String(), privado) {
					t.Fatalf("causa privada publicada: %s", respuesta.Body)
				}
			}
			comprobarCabecerasConsultaRRHH(t, respuesta)
		})
	}
}

func comprobarCabecerasConsultaRRHH(
	t *testing.T,
	respuesta *httptest.ResponseRecorder,
) {
	t.Helper()
	if respuesta.Header().Get("Cache-Control") != "no-store, no-transform" ||
		respuesta.Header().Get("Pragma") != "no-cache" ||
		respuesta.Header().Get("Set-Cookie") != "" ||
		respuesta.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Fatalf("cabeceras inseguras: %#v", respuesta.Header())
	}
}
