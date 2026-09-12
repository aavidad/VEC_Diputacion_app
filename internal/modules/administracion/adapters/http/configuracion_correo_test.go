package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	adminapp "vec-diputacion-granada/internal/modules/administracion/application"
	admindomain "vec-diputacion-granada/internal/modules/administracion/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

func TestManejadorConfiguracionCorreoGETAutorizadoYRespuestaRedactada(t *testing.T) {
	ejecutor := &ejecutorConfiguracionCorreoPrueba{vista: vistaConfiguracionCorreoHTTPPrueba()}
	manejador := manejadorConfiguracionCorreoPrueba(t, &autoridadConfiguracionCorreoPrueba{}, ejecutor)
	respuesta := servirConfiguracionCorreo(manejador, http.MethodGet, RutaConfiguracionCorreo, "")

	if respuesta.Code != http.StatusOK || ejecutor.consultas != 1 || ejecutor.actualizaciones != 0 {
		t.Fatalf("GET autorizado inesperado: estado=%d consultas=%d actualizaciones=%d cuerpo=%s", respuesta.Code, ejecutor.consultas, ejecutor.actualizaciones, respuesta.Body.String())
	}
	if respuesta.Header().Get("Cache-Control") != "no-store, no-transform" || strings.Contains(respuesta.Body.String(), "secreto-smtp-sintetico") || strings.Contains(respuesta.Body.String(), `"secreto"`) {
		t.Fatalf("GET expone o almacena una respuesta sensible: cabeceras=%v cuerpo=%s", respuesta.Header(), respuesta.Body.String())
	}
}

func TestManejadorConfiguracionCorreoDeniegaSinAutoridad(t *testing.T) {
	ejecutor := &ejecutorConfiguracionCorreoPrueba{}
	manejador := manejadorConfiguracionCorreoPrueba(t, &autoridadConfiguracionCorreoPrueba{err: errors.New("canal no acreditado")}, ejecutor)
	respuesta := servirConfiguracionCorreo(manejador, http.MethodGet, RutaConfiguracionCorreo, "")
	if respuesta.Code != http.StatusForbidden || codigoErrorHTTP(t, respuesta) != "acceso_denegado" || ejecutor.consultas != 0 {
		t.Fatalf("GET sin autoridad no quedo denegado: estado=%d consultas=%d cuerpo=%s", respuesta.Code, ejecutor.consultas, respuesta.Body.String())
	}
}

func TestManejadorConfiguracionCorreoPUTCASYSecretoWriteOnly(t *testing.T) {
	const secreto = "secreto-smtp-sintetico"
	ejecutor := &ejecutorConfiguracionCorreoPrueba{vista: vistaConfiguracionCorreoHTTPPrueba()}
	manejador := manejadorConfiguracionCorreoPrueba(t, &autoridadConfiguracionCorreoPrueba{}, ejecutor)
	respuesta := servirConfiguracionCorreo(manejador, http.MethodPut, RutaConfiguracionCorreo, cuerpoConfiguracionCorreoHTTP(4, secreto))

	if respuesta.Code != http.StatusOK || ejecutor.actualizaciones != 1 {
		t.Fatalf("PUT CAS no llego al ejecutor: estado=%d actualizaciones=%d cuerpo=%s", respuesta.Code, ejecutor.actualizaciones, respuesta.Body.String())
	}
	entrada := ejecutor.ultimaActualizacion
	if entrada.Version != 0 || entrada.VersionEsperada != 4 || entrada.SecretoNuevo == nil {
		t.Fatalf("actualizacion CAS/write-only inesperada: %+v", entrada)
	}
	if err := entrada.SecretoNuevo.Consumir(func(valor []byte) error {
		if string(valor) != secreto {
			t.Fatalf("el ejecutor recibio un secreto distinto")
		}
		return nil
	}); err != nil {
		t.Fatalf("el secreto write-only no fue consumible: %v", err)
	}
	if strings.Contains(respuesta.Body.String(), secreto) || strings.Contains(respuesta.Body.String(), `"secreto"`) {
		t.Fatalf("PUT expone secreto en respuesta: %s", respuesta.Body.String())
	}
}

func TestManejadorConfiguracionCorreoRechazaRutaMetodoYCuerposInvalidos(t *testing.T) {
	manejador := manejadorConfiguracionCorreoPrueba(t, &autoridadConfiguracionCorreoPrueba{}, &ejecutorConfiguracionCorreoPrueba{vista: vistaConfiguracionCorreoHTTPPrueba()})
	casos := []struct {
		nombre, metodo, ruta, cuerpo string
		estado                       int
	}{
		{"ruta con consulta", http.MethodGet, RutaConfiguracionCorreo + "?x=1", "", http.StatusNotFound},
		{"metodo POST", http.MethodPost, RutaConfiguracionCorreo, "", http.StatusMethodNotAllowed},
		{"clave desconocida", http.MethodPut, RutaConfiguracionCorreo, `{"desconocida":true}`, http.StatusBadRequest},
		{"segundo JSON", http.MethodPut, RutaConfiguracionCorreo, cuerpoConfiguracionCorreoHTTP(4, "") + `{}`, http.StatusBadRequest},
		{"cuerpo excesivo", http.MethodPut, RutaConfiguracionCorreo, strings.Repeat(" ", 32*1024+1) + cuerpoConfiguracionCorreoHTTP(4, ""), http.StatusBadRequest},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			respuesta := servirConfiguracionCorreo(manejador, caso.metodo, caso.ruta, caso.cuerpo)
			if respuesta.Code != caso.estado {
				t.Fatalf("estado inesperado: obtenido=%d esperado=%d cuerpo=%s", respuesta.Code, caso.estado, respuesta.Body.String())
			}
			if caso.estado == http.StatusMethodNotAllowed && respuesta.Header().Get("Allow") != "GET, PUT" {
				t.Fatalf("Allow inesperado: %q", respuesta.Header().Get("Allow"))
			}
		})
	}
}

func TestManejadorConfiguracionCorreoMapeaConflictoYNoDisponible(t *testing.T) {
	for nombre, resultado := range map[string]struct {
		errEjecutor error
		estado      int
		codigo      string
	}{
		"conflicto":     {adminapp.ErrConfiguracionCorreoConflicto, http.StatusConflict, "conflicto_version"},
		"no disponible": {adminapp.ErrConfiguracionCorreoNoDisponible, http.StatusServiceUnavailable, "servicio_no_disponible"},
	} {
		t.Run(nombre, func(t *testing.T) {
			ejecutor := &ejecutorConfiguracionCorreoPrueba{errActualizar: resultado.errEjecutor}
			manejador := manejadorConfiguracionCorreoPrueba(t, &autoridadConfiguracionCorreoPrueba{}, ejecutor)
			respuesta := servirConfiguracionCorreo(manejador, http.MethodPut, RutaConfiguracionCorreo, cuerpoConfiguracionCorreoHTTP(4, ""))
			if respuesta.Code != resultado.estado || codigoErrorHTTP(t, respuesta) != resultado.codigo {
				t.Fatalf("error HTTP inesperado: estado=%d cuerpo=%s", respuesta.Code, respuesta.Body.String())
			}
		})
	}
}

type autoridadConfiguracionCorreoPrueba struct{ err error }

func (a *autoridadConfiguracionCorreoPrueba) PrincipalConfiguracionCorreo(context.Context) (vecdomain.Principal, error) {
	if a.err != nil {
		return vecdomain.Principal{}, a.err
	}
	return vecdomain.Principal{ID: "admin-correo-http", AuthMethod: vecdomain.AuthMethodCertificate, AuthAssurance: vecdomain.AuthAssuranceHigh}, nil
}

type ejecutorConfiguracionCorreoPrueba struct {
	vista               admindomain.VistaConfiguracionCorreo
	errConsultar        error
	errActualizar       error
	consultas           int
	actualizaciones     int
	ultimaActualizacion admindomain.ActualizacionConfiguracionCorreo
}

func (e *ejecutorConfiguracionCorreoPrueba) Consultar(context.Context, vecdomain.Principal) (admindomain.VistaConfiguracionCorreo, error) {
	e.consultas++
	return e.vista, e.errConsultar
}
func (e *ejecutorConfiguracionCorreoPrueba) Actualizar(_ context.Context, _ vecdomain.Principal, entrada admindomain.ActualizacionConfiguracionCorreo) (admindomain.VistaConfiguracionCorreo, error) {
	e.actualizaciones++
	e.ultimaActualizacion = entrada
	return e.vista, e.errActualizar
}

func manejadorConfiguracionCorreoPrueba(t *testing.T, autoridad autoridadConfiguracionCorreo, ejecutor ejecutorConfiguracionCorreo) *ManejadorConfiguracionCorreo {
	t.Helper()
	manejador, err := NuevoManejadorConfiguracionCorreo(autoridad, ejecutor)
	if err != nil {
		t.Fatalf("no se creo manejador de prueba: %v", err)
	}
	return manejador
}

func servirConfiguracionCorreo(manejador http.Handler, metodo, ruta, cuerpo string) *httptest.ResponseRecorder {
	peticion := httptest.NewRequest(metodo, ruta, strings.NewReader(cuerpo))
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, peticion)
	return respuesta
}

func cuerpoConfiguracionCorreoHTTP(version uint64, secreto string) string {
	entrada := map[string]any{
		"configurada": true, "host": "smtp.intranet.local", "puerto": 465, "server_name": "smtp.intranet.local",
		"referencia_ca": "ca:correo-interno:v1", "remitente_fijo": "rrhh@diputacion.example", "usuario": "rrhh-smtp",
		"modo_tls": "tls_implicito", "modo_autenticacion": "xoauth2", "tiempo_maximo_ms": 5000,
		"secreto_configurado": true, "version_esperada": version,
	}
	if secreto != "" {
		entrada["secreto"] = secreto
	}
	datos, _ := json.Marshal(entrada)
	return string(datos)
}

func vistaConfiguracionCorreoHTTPPrueba() admindomain.VistaConfiguracionCorreo {
	return admindomain.VistaConfiguracionCorreo{Configurada: true, Host: "smtp.intranet.local", Puerto: 465, NombreServidor: "smtp.intranet.local", ReferenciaCA: "ca:correo-interno:v1", RemitenteFijo: "rrhh@diputacion.example", Usuario: "rrhh-smtp", ModoTLS: admindomain.ModoTLSCorreoImplicito, ModoAutenticacion: admindomain.ModoAutenticacionCorreoXOAUTH2, TiempoMaximoMillis: 5000, SecretoConfigurado: true, Version: 5}
}

func codigoErrorHTTP(t *testing.T, respuesta *httptest.ResponseRecorder) string {
	t.Helper()
	var cuerpo struct {
		Error struct {
			Codigo string `json:"codigo"`
		} `json:"error"`
	}
	if err := json.Unmarshal(respuesta.Body.Bytes(), &cuerpo); err != nil {
		t.Fatalf("respuesta de error no es JSON: %v", err)
	}
	return cuerpo.Error.Codigo
}
