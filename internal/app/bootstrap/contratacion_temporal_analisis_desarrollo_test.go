package bootstrap

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestRutaConfiguracionAnalisisDesarrolloPublicaContratoCerrado(t *testing.T) {
	ruta, err := nuevaRutaConfiguracionAnalisisContratacionTemporalDesarrollo()
	if err != nil {
		t.Fatal(err)
	}
	if ruta.Ruta != rutaConfiguracionAnalisisContratacionTemporalDesarrollo ||
		ruta.Manejador == nil {
		t.Fatalf("ruta inesperada: %+v", ruta)
	}
	peticion := httptest.NewRequest(http.MethodGet, ruta.Ruta, nil)
	respuesta := httptest.NewRecorder()
	ruta.Manejador.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusOK {
		t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body.String())
	}
	if respuesta.Header().Get("Set-Cookie") != "" ||
		respuesta.Header().Get("Cache-Control") != "no-store, no-transform" {
		t.Fatalf("cabeceras inseguras: %+v", respuesta.Header())
	}
	var contenido struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(respuesta.Body.Bytes(), &contenido); err != nil {
		t.Fatal(err)
	}
	campos := []string{
		"esquema",
		"artefacto_ref",
		"modalidades",
		"categorias",
		"causas",
		"entradas_rc",
		"motivos_rectificacion",
	}
	if len(contenido.Data) != len(campos) {
		t.Fatalf("contrato abierto o incompleto: %v", contenido.Data)
	}
	for _, campo := range campos {
		if _, existe := contenido.Data[campo]; !existe {
			t.Fatalf("falta %s", campo)
		}
	}
	var configuracion configuracionAnalisisContratacionTemporalDesarrollo
	if err := json.Unmarshal(respuesta.Body.Bytes(), &struct {
		Data *configuracionAnalisisContratacionTemporalDesarrollo `json:"data"`
	}{Data: &configuracion}); err != nil {
		t.Fatal(err)
	}
	if configuracion.Esquema != esquemaConfiguracionAnalisisContratacionTemporal ||
		configuracion.ArtefactoRef != artefactoAnalisisContratacionTemporalDesarrollo ||
		len(configuracion.Modalidades) != 5 ||
		len(configuracion.Categorias) != 1 ||
		len(configuracion.Categorias[0].GruposSubgrupos) != 1 ||
		len(configuracion.Causas) != 1 ||
		len(configuracion.EntradasRC) != 1 ||
		len(configuracion.MotivosRectificacion) != 0 {
		t.Fatalf("configuracion inesperada: %+v", configuracion)
	}
	esperadas := []string{
		"sustitucion",
		"vacante",
		"acumulacion_tareas",
		"programa",
		"relevo",
	}
	for indice, esperada := range esperadas {
		if configuracion.Modalidades[indice].Clave != esperada {
			t.Fatalf("modalidad[%d]=%q", indice, configuracion.Modalidades[indice].Clave)
		}
	}
	entrada := configuracion.EntradasRC[0]
	if entrada.Referencia != entradaRCAnalisisContratacionTemporalDesarrollo ||
		entrada.HuellaSHA256 != huellaEntradaRCAnalisisContratacionTemporalDesarrollo ||
		entrada.HuellaSHA256 == strings.Repeat("0", 64) {
		t.Fatalf("entrada RC inesperada: %+v", entrada)
	}

	peticionHEAD := httptest.NewRequest(http.MethodHead, ruta.Ruta, nil)
	respuestaHEAD := httptest.NewRecorder()
	ruta.Manejador.ServeHTTP(respuestaHEAD, peticionHEAD)
	if respuestaHEAD.Code != http.StatusOK || respuestaHEAD.Body.Len() != 0 ||
		respuestaHEAD.Header().Get("Content-Length") !=
			respuesta.Header().Get("Content-Length") {
		t.Fatalf("HEAD incoherente: estado=%d cabeceras=%v cuerpo=%q",
			respuestaHEAD.Code,
			respuestaHEAD.Header(),
			respuestaHEAD.Body.String(),
		)
	}
}

func TestRutaConfiguracionAnalisisDesarrolloFallaCerrada(t *testing.T) {
	ruta, err := nuevaRutaConfiguracionAnalisisContratacionTemporalDesarrollo()
	if err != nil {
		t.Fatal(err)
	}
	casos := []struct {
		nombre string
		crear  func() *http.Request
		estado int
	}{
		{
			nombre: "metodo",
			crear: func() *http.Request {
				return httptest.NewRequest(http.MethodPost, ruta.Ruta, nil)
			},
			estado: http.StatusMethodNotAllowed,
		},
		{
			nombre: "consulta",
			crear: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, ruta.Ruta+"?perfil=rrhh", nil)
			},
			estado: http.StatusBadRequest,
		},
		{
			nombre: "cuerpo",
			crear: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, ruta.Ruta, strings.NewReader("{}"))
			},
			estado: http.StatusBadRequest,
		},
		{
			nombre: "identidad libre",
			crear: func() *http.Request {
				peticion := httptest.NewRequest(http.MethodGet, ruta.Ruta, nil)
				peticion.Header.Set("X-User", "rrhh")
				return peticion
			},
			estado: http.StatusBadRequest,
		},
		{
			nombre: "cookie",
			crear: func() *http.Request {
				peticion := httptest.NewRequest(http.MethodGet, ruta.Ruta, nil)
				peticion.Header.Set("Cookie", "sesion=no")
				return peticion
			},
			estado: http.StatusBadRequest,
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			respuesta := httptest.NewRecorder()
			ruta.Manejador.ServeHTTP(respuesta, caso.crear())
			if respuesta.Code != caso.estado {
				t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body.String())
			}
			if respuesta.Header().Get("Set-Cookie") != "" {
				t.Fatal("la respuesta emitio una cookie")
			}
			var contenido map[string]json.RawMessage
			if err := json.Unmarshal(respuesta.Body.Bytes(), &contenido); err != nil {
				t.Fatal(err)
			}
			if _, existe := contenido["data"]; existe {
				t.Fatal("una solicitud denegada recibio datos")
			}
		})
	}
}

func TestPoliticaAnalisisDesarrolloSoloAdmiteRegistroConfigurado(t *testing.T) {
	instante := time.Now().UTC().Truncate(time.Microsecond)
	solicitud := ports.SolicitudResolverPoliticaOperacionAnalisis{
		Operacion:         ports.OperacionRegistrarAnalisis,
		OrganizacionRef:   organizacionAltaContratacionTemporalDesarrollo,
		ExpedienteRef:     "expediente:ct:desarrollo:analisis:001",
		VersionExpediente: 1,
		Flujo: domain.ReferenciaFlujo{
			DefinicionRef: "flujo:ct:desarrollo",
			Version:       1,
			HuellaSHA256:  huellaAltaContratacionTemporalDesarrollo("flujo"),
		},
		FasePrevia:            "solicitud",
		EstadoPrevio:          domain.EstadoEnCurso,
		ActorRef:              "principal:desarrollo:rrhh",
		PerfilRef:             "perfil:desarrollo:rrhh",
		ArtefactoRef:          artefactoAnalisisContratacionTemporalDesarrollo,
		ArtefactoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("artefacto"),
		Instante:              instante,
	}
	politica, err := (resolutorPoliticaOperacionAnalisisDesarrollo{}).
		ResolverPoliticaOperacionAnalisis(t.Context(), solicitud)
	if err != nil {
		t.Fatal(err)
	}
	if politica.ValidarPara(solicitud) != nil ||
		politica.Accion != domain.ClaveCatalogo(ports.AccionRegistrarAnalisis) ||
		politica.Finalidad != finalidadAnalisisContratacionTemporalDesarrollo ||
		politica.MotivoAutorizacion != referenciaMotivoAutorizacionAnalisisDesarrollo("registro") {
		t.Fatalf("politica inesperada: %+v", politica)
	}
	for _, caso := range []struct {
		nombre  string
		alterar func(*ports.SolicitudResolverPoliticaOperacionAnalisis)
	}{
		{
			nombre: "fase ajena",
			alterar: func(s *ports.SolicitudResolverPoliticaOperacionAnalisis) {
				s.FasePrevia = "cobertura"
			},
		},
		{
			nombre: "estado ajeno",
			alterar: func(s *ports.SolicitudResolverPoliticaOperacionAnalisis) {
				s.EstadoPrevio = domain.EstadoPendiente
			},
		},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			alterada := solicitud
			caso.alterar(&alterada)
			if _, err := (resolutorPoliticaOperacionAnalisisDesarrollo{}).
				ResolverPoliticaOperacionAnalisis(t.Context(), alterada); err == nil {
				t.Fatal("la politica acepto coordenadas fuera del paso real")
			}
		})
	}
	solicitud.Operacion = ports.OperacionRectificarAnalisis
	solicitud.ActorAnalisisAnteriorRef = "principal:desarrollo:anterior"
	solicitud.MotivoRectificacionClave = "correccion"
	if _, err := (resolutorPoliticaOperacionAnalisisDesarrollo{}).
		ResolverPoliticaOperacionAnalisis(t.Context(), solicitud); err == nil {
		t.Fatal("la rectificacion sin motivo configurado quedo abierta")
	}
}
