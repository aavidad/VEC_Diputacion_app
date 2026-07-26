package httpinterno

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
)

func TestRutasContratacionTemporalUsanListaPositivaCanonicaVEC(t *testing.T) {
	rutas := []struct {
		nombre   string
		obtenida string
		esperada string
	}{
		{"alta", RutaAltaSolicitudes, "/api/vec/contratacion-temporal/solicitudes"},
		{"propuesta", RutaPropuestaCobertura, "/api/vec/contratacion-temporal/cobertura/propuesta"},
		{"decisión", RutaDecisionCobertura, "/api/vec/contratacion-temporal/cobertura/decisiones"},
		{"rectificación", RutaRectificacionCobertura, "/api/vec/contratacion-temporal/cobertura/rectificaciones"},
	}
	for _, ruta := range rutas {
		t.Run(ruta.nombre, func(t *testing.T) {
			if ruta.obtenida != ruta.esperada {
				t.Errorf("ruta fuera del contrato interno canónico: %q; esperada %q", ruta.obtenida, ruta.esperada)
			}
		})
	}
}

func TestManejadorCoberturaClasificaContextoCorporativoSinFiltrarDetalle(t *testing.T) {
	casos := []struct {
		nombre string
		causa  error
		estado int
		codigo string
	}{
		{"ausente", ErrContextoCanalAusente, http.StatusUnauthorized, "autenticacion_requerida"},
		{"caducado", ErrContextoCanalCaducado, http.StatusUnauthorized, "autenticacion_requerida"},
		{"otra organización", ErrContextoCanalOrganizacionDenegada, http.StatusForbidden, "acceso_denegado"},
		{"autoridad no disponible", ErrContextoCanalNoDisponible, http.StatusServiceUnavailable, "servicio_no_disponible"},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			servicio := &servicioCoberturaPrueba{}
			manejador, err := NuevoManejadorCobertura(
				autoridadCoberturaPrueba{err: errors.Join(caso.causa, errors.New("detalle corporativo reservado"))},
				servicio,
				servicio,
			)
			if err != nil {
				t.Fatal(err)
			}
			respuesta := httptest.NewRecorder()
			manejador.ServeHTTP(
				respuesta,
				nuevaPeticionCoberturaPrueba(
					RutaPropuestaCobertura,
					`{"expediente_ref":"expediente:ct:0001","version_esperada":1}`,
				),
			)
			if respuesta.Code != caso.estado ||
				codigoErrorCoberturaPrueba(t, respuesta) != caso.codigo {
				t.Fatalf("estado/código = %d/%s", respuesta.Code, respuesta.Body.String())
			}
			if strings.Contains(respuesta.Body.String(), "reservado") {
				t.Fatalf("filtró detalle privado: %s", respuesta.Body.String())
			}
			if servicio.proponerLlamadas != 0 ||
				servicio.decidirLlamadas != 0 ||
				servicio.rectificarLlamadas != 0 {
				t.Fatal("ejecutó un caso de uso sin contexto corporativo confiable")
			}
			comprobarCabecerasSegurasPrueba(t, respuesta)
		})
	}
}

func TestManejadorCoberturaResultadoPendienteNoInduceReintento(t *testing.T) {
	servicio := &servicioCoberturaPrueba{
		err: errors.Join(
			application.ErrConfirmacionDecisionCoberturaPendiente,
			application.ErrConfirmacionDecisionCoberturaNoDisponible,
			errors.New("resultado privado posterior a COMMIT"),
		),
	}
	manejador, err := NuevoManejadorCobertura(
		autoridadCoberturaPrueba{contexto: contextoCoberturaValidoPrueba()},
		servicio,
		servicio,
	)
	if err != nil {
		t.Fatal(err)
	}
	respuesta := httptest.NewRecorder()
	respuesta.Header().Set("Retry-After", "30")
	manejador.ServeHTTP(
		respuesta,
		nuevaPeticionCoberturaPrueba(
			RutaDecisionCobertura,
			cuerpoDecisionCoberturaPrueba(false),
		),
	)
	if respuesta.Code != http.StatusServiceUnavailable ||
		codigoErrorCoberturaPrueba(t, respuesta) != "operacion_pendiente" {
		t.Fatalf("resultado pendiente = %d %s", respuesta.Code, respuesta.Body.String())
	}
	if respuesta.Header().Get("Retry-After") != "" ||
		strings.Contains(respuesta.Body.String(), "COMMIT") ||
		strings.Contains(respuesta.Body.String(), "privado") {
		t.Fatalf("respuesta induce reintento o filtra detalle: %v %s", respuesta.Header(), respuesta.Body.String())
	}
	comprobarCabecerasSegurasPrueba(t, respuesta)
}

func TestManejadorCoberturaNoConfundeIndisponibilidadConOperacionPendiente(t *testing.T) {
	servicio := &servicioCoberturaPrueba{
		err: application.ErrConfirmacionDecisionCoberturaNoDisponible,
	}
	manejador, err := NuevoManejadorCobertura(
		autoridadCoberturaPrueba{contexto: contextoCoberturaValidoPrueba()},
		servicio,
		servicio,
	)
	if err != nil {
		t.Fatal(err)
	}
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(
		respuesta,
		nuevaPeticionCoberturaPrueba(
			RutaDecisionCobertura,
			cuerpoDecisionCoberturaPrueba(false),
		),
	)
	if respuesta.Code != http.StatusServiceUnavailable ||
		codigoErrorCoberturaPrueba(t, respuesta) != "servicio_no_disponible" {
		t.Fatalf("indisponibilidad = %d %s", respuesta.Code, respuesta.Body.String())
	}
}

func codigoErrorCoberturaPrueba(
	t *testing.T,
	respuesta *httptest.ResponseRecorder,
) string {
	t.Helper()
	var envoltorio envoltorioErrorCobertura
	if err := json.Unmarshal(respuesta.Body.Bytes(), &envoltorio); err != nil {
		t.Fatalf("error no es JSON: %v; cuerpo=%q", err, respuesta.Body.String())
	}
	esperada := "api.contratacion_temporal.cobertura.error." +
		envoltorio.Error.Codigo
	if envoltorio.Error.ClaveI18n != esperada ||
		envoltorio.Error.CorrelacionRef == "" {
		t.Fatalf("error público incompleto: %+v", envoltorio.Error)
	}
	return envoltorio.Error.Codigo
}
