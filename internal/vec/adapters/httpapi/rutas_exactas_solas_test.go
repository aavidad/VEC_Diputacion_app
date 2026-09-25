package httpapi

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"vec-diputacion-granada/internal/vec/ports"
)

func TestHandlerSoloRutasExactasDeniegaOtrasRutasSinCarcasa(t *testing.T) {
	t.Parallel()
	const ruta = "/api/vec/contratacion-temporal/seguimiento/consulta"
	manejador := &manejadorExactoPrueba{}
	autoridad := &autoridadRutasExactasEspia{}
	auditoria := &registradorAuditoriaFronteraRutaExactaEspia{}
	rutas := []RutaExacta{{Ruta: ruta, Manejador: manejador}}
	handler, err := NewHandlerSoloRutasExactas(rutas, autoridad, auditoria)
	if err != nil {
		t.Fatal(err)
	}
	rutas[0].Manejador = http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("se modificó la ruta tras construir el manejador")
	})

	for _, otra := range []string{
		"/api/vec/session",
		"/api/vec/modules",
		"/api/vec/dietas/road-route",
		"/api/vec/cronos/timecards",
		"/api/vec/contratacion-temporal/seguimiento/consulta/otra",
	} {
		respuesta := httptest.NewRecorder()
		handler.ServeHTTP(respuesta, httptest.NewRequest(http.MethodGet, otra, nil))
		if respuesta.Code != http.StatusNotFound {
			t.Fatalf("%s: estado=%d", otra, respuesta.Code)
		}
	}
	if llamadas, _ := autoridad.estado(); llamadas != 0 {
		t.Fatalf("autoridad invocada en rutas no declaradas: %d", llamadas)
	}
	respuesta := httptest.NewRecorder()
	handler.ServeHTTP(respuesta, httptest.NewRequest(http.MethodGet, ruta, nil))
	if respuesta.Code != http.StatusNoContent {
		t.Fatalf("ruta declarada: estado=%d", respuesta.Code)
	}
	if llamadas, recibida := autoridad.estado(); llamadas != 1 || recibida != ruta {
		t.Fatalf("autorización=(%d, %q)", llamadas, recibida)
	}
	if llamadas, recibida, _ := manejador.estado(); llamadas != 1 || recibida != ruta {
		t.Fatalf("manejador=(%d, %q)", llamadas, recibida)
	}
}

func TestHandlerSoloRutasExactasExigeDependenciasYAuditaDenegacion(t *testing.T) {
	t.Parallel()
	const ruta = "/api/vec/contratacion-temporal/seguimiento/consulta"
	manejador := &manejadorExactoPrueba{}
	rutas := []RutaExacta{{Ruta: ruta, Manejador: manejador}}
	autoridad := &autoridadRutasExactasEspia{err: ErrAutenticacionRutaExactaRequerida}
	auditoria := &registradorAuditoriaFronteraRutaExactaEspia{}
	for _, caso := range []struct {
		name      string
		rutas     []RutaExacta
		autoridad AutoridadRutasExactas
		auditoria ports.RegistradorAuditoriaFronteraRutaExacta
	}{
		{"sin rutas", nil, autoridad, auditoria},
		{"sin autoridad", rutas, nil, auditoria},
		{"sin auditoria", rutas, autoridad, nil},
		{"ruta invalida", []RutaExacta{{Ruta: "/api/vec/session", Manejador: manejador}}, autoridad, auditoria},
	} {
		if h, err := NewHandlerSoloRutasExactas(caso.rutas, caso.autoridad, caso.auditoria); h != nil || !errors.Is(err, ErrRutaExactaInvalida) {
			t.Fatalf("%s: manejador=%T error=%v", caso.name, h, err)
		}
	}
	handler, err := NewHandlerSoloRutasExactas(rutas, autoridad, auditoria)
	if err != nil {
		t.Fatal(err)
	}
	respuesta := httptest.NewRecorder()
	handler.ServeHTTP(respuesta, httptest.NewRequest(http.MethodGet, ruta, nil))
	if respuesta.Code != http.StatusUnauthorized {
		t.Fatalf("denegación: estado=%d", respuesta.Code)
	}
	if llamadas, _, _ := manejador.estado(); llamadas != 0 {
		t.Fatalf("manejador invocado tras denegación: %d", llamadas)
	}
	ordenes := auditoria.ordenesRegistradas()
	if len(ordenes) != 1 || ordenes[0].Validar() != nil ||
		ordenes[0].Ruta != ruta ||
		ordenes[0].Motivo != ports.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida {
		t.Fatalf("auditoría=%#v", ordenes)
	}
}
