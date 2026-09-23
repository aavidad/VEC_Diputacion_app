package contactopropio

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/ports"
)

type ejecutorOperacionesContactoPrueba struct {
	llamadas int
	fallo    error
	op       ports.OperacionContactoUsuario
}

func (e *ejecutorOperacionesContactoPrueba) PrepararOperacion(context.Context, string, uint64) (ports.OperacionContactoUsuario, error) {
	e.llamadas++
	return e.op, e.fallo
}
func (e *ejecutorOperacionesContactoPrueba) ConfirmarOperacion(context.Context, string, string, uint64) (ports.OperacionContactoUsuario, error) {
	e.llamadas++
	return e.op, e.fallo
}
func (e *ejecutorOperacionesContactoPrueba) CancelarOperacion(context.Context, string) (ports.OperacionContactoUsuario, error) {
	e.llamadas++
	return e.op, e.fallo
}
func (e *ejecutorOperacionesContactoPrueba) ListarOperaciones(context.Context, uint32, string) (ports.ResultadoListaOperacionesContacto, error) {
	e.llamadas++
	return ports.ResultadoListaOperacionesContacto{Operaciones: []ports.OperacionContactoUsuario{e.op}, SiguienteDesde: e.op.OperacionRef}, e.fallo
}
func (e *ejecutorOperacionesContactoPrueba) DetalleOperacion(context.Context, string) (ports.ResultadoDetalleOperacionContacto, error) {
	e.llamadas++
	return ports.ResultadoDetalleOperacionContacto{Encontrada: true, Operacion: e.op}, e.fallo
}

func TestOperacionesContactoSoloRutasFijasYJSONMinimo(t *testing.T) {
	ref := "opr_" + strings.Repeat("o", 22)
	e := &ejecutorOperacionesContactoPrueba{op: ports.OperacionContactoUsuario{OperacionRef: ref, Estado: ports.OperacionContactoPreparada}}
	rutas, err := NuevasRutasOperaciones(e, catalogoContactoPropioPrueba(t))
	if err != nil || len(rutas) != 5 {
		t.Fatal("rutas de operaciones incompletas")
	}
	casos := []struct{ ruta, cuerpo string }{
		{RutaOperacionContactoPreparar, `{"correo":"ana@example.test","version_esperada":0}`},
		{RutaOperacionContactoConfirmar, `{"operacion_ref":"` + ref + `","correo":"ana@example.test","version_esperada":0}`},
		{RutaOperacionContactoCancelar, `{"operacion_ref":"` + ref + `"}`},
		{RutaOperacionContactoConsultas, `{"limite":20}`},
		{RutaOperacionContactoDetalle, `{"operacion_ref":"` + ref + `"}`},
	}
	for i, caso := range casos {
		switch i {
		case 1:
			e.op.Estado, e.op.Version, e.op.ReciboRef = ports.OperacionContactoConfirmada, 1, "acc_"+strings.Repeat("a", 40)
		case 2:
			e.op.Estado, e.op.Version, e.op.ReciboRef = ports.OperacionContactoCancelada, 0, ""
		}
		r := httptest.NewRequest(http.MethodPost, caso.ruta, strings.NewReader(caso.cuerpo))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		rutas[i].Manejador.ServeHTTP(w, r)
		if w.Code < 200 || w.Code >= 300 || e.llamadas != i+1 || w.Header().Get("Cache-Control") != "no-store" || strings.Contains(w.Body.String(), "ana@example.test") || strings.Contains(w.Body.String(), "hmac") || strings.Contains(w.Body.String(), "cifrado") {
			t.Fatalf("ruta %d expuso datos o falló: estado=%d cuerpo=%s", i, w.Code, w.Body.String())
		}
	}
	for _, cuerpo := range []string{
		`{"correo":"ana@example.test","version_esperada":0,"persona_ref":"forjada"}`,
		`{"correo":"ana@example.test","correo":"otra@example.test","version_esperada":0}`,
		`{"correo":"ana@example.test","version_esperada":null}`,
	} {
		r := httptest.NewRequest(http.MethodPost, RutaOperacionContactoPreparar, strings.NewReader(cuerpo))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		rutas[0].Manejador.ServeHTTP(w, r)
		if w.Code != http.StatusBadRequest || e.llamadas != 5 {
			t.Fatalf("entrada no canónica admitida: %d", w.Code)
		}
	}
	e.op = ports.OperacionContactoUsuario{OperacionRef: ref, Estado: ports.OperacionContactoConfirmada, VersionEsperada: 0, Version: 1,
		ReciboRef: "acc_" + strings.Repeat("a", 40), ReplayConfirmado: true}
	r := httptest.NewRequest(http.MethodPost, RutaOperacionContactoPreparar, strings.NewReader(`{"correo":"ana@example.test","version_esperada":0}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	rutas[0].Manejador.ServeHTTP(w, r)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"estado":"confirmada"`) || !strings.Contains(w.Body.String(), `"recibo_ref"`) {
		t.Fatal("replay de preparación confirmada perdió recibo original")
	}
}

func TestOperacionContactoCommitInciertoConservaSelectorSinFalsoRecibo(t *testing.T) {
	ref := "opr_" + strings.Repeat("o", 22)
	e := &ejecutorOperacionesContactoPrueba{fallo: ErrContactoPropioCommitIncierto}
	rutas, err := NuevasRutasOperaciones(e, catalogoContactoPropioPrueba(t))
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodPost, RutaOperacionContactoConfirmar, strings.NewReader(`{"operacion_ref":"`+ref+`","correo":"ana@example.test","version_esperada":0}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	rutas[1].Manejador.ServeHTTP(w, r)
	if w.Code != http.StatusServiceUnavailable || !strings.Contains(w.Body.String(), `"codigo":"confirmacion_incierta"`) || !strings.Contains(w.Body.String(), ref) || strings.Contains(w.Body.String(), "recibo_ref") || strings.Contains(w.Body.String(), "ana@example.test") {
		t.Fatalf("respuesta incierta incorrecta: %d %s", w.Code, w.Body.String())
	}
	e.fallo = application.ErrOperacionContactoNoEncontrada
	r = httptest.NewRequest(http.MethodPost, RutaOperacionContactoDetalle, strings.NewReader(`{"operacion_ref":"`+ref+`"}`))
	r.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	rutas[4].Manejador.ServeHTTP(w, r)
	if w.Code != http.StatusNotFound {
		t.Fatalf("selector ajeno/ausente no quedó oculto: %d", w.Code)
	}
}
