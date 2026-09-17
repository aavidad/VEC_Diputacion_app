package httpinterno

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application/diagnostico"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestRegistroDiagnosticoClasificaCentinelaSinExponerCausa(t *testing.T) {
	anterior := slog.Default()
	defer slog.SetDefault(anterior)
	for _, caso := range []struct {
		nombre, esperado string
		causa            error
	}{
		{"persistencia", "persistencia_no_disponible", ports.ErrPersistenciaNoDisponible},
		{"canal", "contexto_canal_no_disponible", ErrContextoCanalNoDisponible},
		{"desconocido", "no_clasificado", errors.New("otra causa")},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			var registro bytes.Buffer
			slog.SetDefault(slog.New(slog.NewJSONHandler(&registro, nil)))
			causa := fmt.Errorf("CONTENIDO_PRIVADO: %w", caso.causa)
			registrarFalloContratacion(
				httptest.NewRequest(http.MethodPost, RutaAltaSolicitudes, nil),
				http.StatusServiceUnavailable, "servicio_no_disponible", "corr_prueba", causa,
			)
			var entrada map[string]any
			if err := json.Unmarshal(registro.Bytes(), &entrada); err != nil {
				t.Fatal(err)
			}
			if entrada["centinela"] != caso.esperado {
				t.Fatalf("centinela=%#v, esperado %q", entrada["centinela"], caso.esperado)
			}
			if strings.Contains(registro.String(), "CONTENIDO_PRIVADO") {
				t.Fatalf("el log filtró causa privada: %s", registro.String())
			}
		})
	}
}

func TestFronterasContratacionCorrelacionanFalloSinContenidoPrivado(t *testing.T) {
	anterior := slog.Default()
	defer slog.SetDefault(anterior)
	var registro bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&registro, nil)))
	causa := &diagnostico.FalloConsultaRRHH{Etapa: diagnostico.EtapaSQL, CodigoSQL: "XX000", Causa: errors.New("CONTENIDO_PRIVADO")}
	problema := nuevoErrorCobertura(503, "servicio_no_disponible")
	for _, caso := range []struct {
		ruta      string
		responder func(http.ResponseWriter, *http.Request)
	}{
		{RutaAltaSolicitudes, func(w http.ResponseWriter, r *http.Request) {
			responderErrorAlta(w, r, errorServicioNoDisponible, causa)
		}},
		{RutaPropuestaCobertura, func(w http.ResponseWriter, r *http.Request) { responderErrorCobertura(w, r, problema, causa) }},
		{RutaRegistroAnalisisRRHH, func(w http.ResponseWriter, r *http.Request) { responderErrorCobertura(w, r, problema, causa) }},
		{RutaAsignaciones, func(w http.ResponseWriter, r *http.Request) { responderErrorAsignacion(w, r, problema, causa) }},
		{RutaPreparacionesInformeJuridico, func(w http.ResponseWriter, r *http.Request) { responderErrorInformeJuridico(w, r, problema, causa) }},
		{RutaResultadosFiscalizacion, func(w http.ResponseWriter, r *http.Request) { responderErrorFiscalizacion(w, r, problema, causa) }},
		{RutaSubsanacionReparos, func(w http.ResponseWriter, r *http.Request) {
			responderErrorSubsanacionReparos(w, r, 503, "servicio_no_disponible", causa)
		}},
		{RutaSeleccionLlamamiento, func(w http.ResponseWriter, r *http.Request) {
			responderErrorSeleccionLlamamiento(w, r, problema, causa)
		}},
		{RutaRegistroComunicacionLlamamiento, func(w http.ResponseWriter, r *http.Request) {
			responderErrorComunicacionLlamamiento(w, r, problema, causa)
		}},
		{RutaPropuestaFormalizacion, func(w http.ResponseWriter, r *http.Request) {
			responderErrorPropuestaFormalizacion(w, r, problema, causa)
		}},
		{RutaResolucionFormalizacion, func(w http.ResponseWriter, r *http.Request) {
			errorHTTPResolucion(w, r, 503, "servicio_no_disponible", causa)
		}},
		{RutaIncorporacionEjercicioV2, func(w http.ResponseWriter, r *http.Request) {
			errorHTTPIncorporacionEjercicioV2(w, r, 503, "servicio_no_disponible", causa)
		}},
		{RutaFichaGINPIXV2, func(w http.ResponseWriter, r *http.Request) {
			errorFichaGINPIXV2(w, r, 503, "servicio_no_disponible", causa)
		}},
		{RutaCerrarAdministrativamente, func(w http.ResponseWriter, r *http.Request) {
			responderErrorCierreAdministrativo(w, r, problema, causa)
		}},
		{RutaAnotacionesAdministrativas, func(w http.ResponseWriter, r *http.Request) { responderErrorAnotacion(w, r, problema, causa) }},
	} {
		t.Run(caso.ruta, func(t *testing.T) {
			registro.Reset()
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", caso.ruta+"?secreto=CONTENIDO_PRIVADO", nil)
			caso.responder(w, r)
			var cuerpo envoltorioErrorCobertura
			var log map[string]any
			if err := json.Unmarshal(w.Body.Bytes(), &cuerpo); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(registro.Bytes(), &log); err != nil {
				t.Fatalf("se esperaba una única línea: %v", err)
			}
			if w.Code != 503 || log["ruta"] != caso.ruta || log["operacion"] == "contratacion" || log["etapa"] != "sql" || log["sqlstate"] != "XX000" || cuerpo.Error.CorrelacionRef == "" || log["correlacion_ref"] != cuerpo.Error.CorrelacionRef {
				t.Fatalf("diagnóstico incompleto: %s", registro.String())
			}
			if strings.Contains(registro.String()+w.Body.String(), "CONTENIDO_PRIVADO") {
				t.Fatal("contenido privado publicado")
			}
		})
	}
}

func TestAltaPropagaCausaHastaRegistroDeFrontera(t *testing.T) {
	anterior := slog.Default()
	defer slog.SetDefault(anterior)
	var registro bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&registro, nil)))
	causa := &diagnostico.FalloConsultaRRHH{Etapa: diagnostico.EtapaAplicacion, Causa: errors.New("CONTENIDO_PRIVADO")}
	h, err := NuevoManejadorAlta(&autoridadPrueba{contexto: contextoCanalValidoPrueba()}, &ejecutorPrueba{err: causa}, relojPrueba{instante: instantePrueba})
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, nuevaPeticionPrueba(t, cuerpoValidoPrueba()))
	var log map[string]any
	var cuerpo envoltorioErrorAlta
	if err := json.Unmarshal(registro.Bytes(), &log); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(w.Body.Bytes(), &cuerpo); err != nil {
		t.Fatal(err)
	}
	if w.Code != 500 || log["etapa"] != "aplicacion" || log["correlacion_ref"] != cuerpo.Error.CorrelacionRef || strings.Contains(registro.String()+w.Body.String(), "CONTENIDO_PRIVADO") {
		t.Fatal("causa perdida o expuesta")
	}
}

func TestSerializacionContratacionRegistraCorrelacionDelErrorFinal(t *testing.T) {
	anterior := slog.Default()
	defer slog.SetDefault(anterior)
	for _, serializar := range []func(http.ResponseWriter, *http.Request, int, any){
		func(w http.ResponseWriter, r *http.Request, s int, v any) { responderJSONAlta(w, r, s, v) },
		func(w http.ResponseWriter, r *http.Request, s int, v any) { responderJSONCobertura(w, r, s, v) },
	} {
		var registro bytes.Buffer
		slog.SetDefault(slog.New(slog.NewJSONHandler(&registro, nil)))
		w := httptest.NewRecorder()
		serializar(w, httptest.NewRequest("POST", RutaAltaSolicitudes, nil), 200, map[string]any{"CONTENIDO_PRIVADO": make(chan int)})
		var log map[string]any
		var cuerpo envoltorioErrorCobertura
		if err := json.Unmarshal(registro.Bytes(), &log); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(w.Body.Bytes(), &cuerpo); err != nil {
			t.Fatal(err)
		}
		if w.Code != 500 || log["etapa"] != "serializacion" || log["correlacion_ref"] != cuerpo.Error.CorrelacionRef || strings.Contains(registro.String()+w.Body.String(), "CONTENIDO_PRIVADO") {
			t.Fatal("fallo de serialización sin correlación segura")
		}
	}
}

func TestRespuestaRecibidaConservaDiagnosticoDelEjecutor(t *testing.T) {
	anterior := slog.Default()
	defer slog.SetDefault(anterior)
	var registro bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&registro, nil)))
	causa := &diagnostico.FalloConsultaRRHH{Etapa: diagnostico.EtapaSQL, CodigoSQL: "XX000", Causa: errors.New("CONTENIDO_PRIVADO")}
	h, err := NuevoManejadorRespuestaRecibida(&ejecutorRespuestaPrueba{err: causa})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(entradaRespuestaPrueba())
	r := httptest.NewRequest("POST", RutaRegistroRespuestaRecibida, bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	var log map[string]any
	var cuerpo envoltorioErrorCobertura
	if err := json.Unmarshal(registro.Bytes(), &log); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(w.Body.Bytes(), &cuerpo); err != nil {
		t.Fatal(err)
	}
	if w.Code != 503 || log["etapa"] != "sql" || log["sqlstate"] != "XX000" || log["correlacion_ref"] != cuerpo.Error.CorrelacionRef || strings.Contains(registro.String()+w.Body.String(), "CONTENIDO_PRIVADO") {
		t.Fatal("el manejador perdió o publicó la causa")
	}
}
