package httppersonal

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/application/mibolsa"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

type ejecutorContactoPrueba struct {
	llamadas int
	bolsa    string
	version  int64
	err      error
}

func (e *ejecutorContactoPrueba) ConfirmarContacto(_ context.Context, _ mibolsa.Orden, bolsa string, version int64, _ string) (puertosbolsa.ReciboConfirmacionContacto, error) {
	e.llamadas++
	e.bolsa, e.version = bolsa, version
	return puertosbolsa.ReciboConfirmacionContacto{ReciboRef: "recibo:confirmacion-contacto:x", Version: version, ConfirmadaEn: time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)}, e.err
}

func TestContactoConfirmaLaVersionVista(t *testing.T) {
	e := new(ejecutorContactoPrueba)
	h, err := NuevoContacto(new(preparadorPrueba), e)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionPortal(RutaMiBolsaContacto, `{"bolsa":"bolsa:01","version":2,"clave":"clave-contacto-01"}`, nil))
	if w.Code != 201 || e.bolsa != "bolsa:01" || e.version != 2 || !strings.Contains(w.Body.String(), `"estado":"confirmado"`) {
		t.Fatalf("status=%d %s", w.Code, w.Body.String())
	}
	if AccionPortalEn(http.MethodPost, RutaMiBolsaContacto)[0] != puertosbolsa.AccionConfirmarContactoPropio || !EsRutaPortal(RutaMiBolsaContacto) {
		t.Fatal("ruta o acción de la confirmación")
	}
	for err, codigo := range map[error]string{
		puertosbolsa.ErrPortalContactoCambiado:     "contacto_cambiado",
		puertosbolsa.ErrPortalContactoYaConfirmado: "contacto_ya_confirmado",
		puertosbolsa.ErrPortalSinContacto:          "sin_contacto",
	} {
		h, _ := NuevoContacto(new(preparadorPrueba), &ejecutorContactoPrueba{err: err})
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionPortal(RutaMiBolsaContacto, `{"bolsa":"bolsa:01","version":2,"clave":"clave-contacto-01"}`, nil))
		if w.Code != 409 || !strings.Contains(w.Body.String(), codigo) {
			t.Fatalf("%v: %d %s", err, w.Code, w.Body.String())
		}
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, peticionPortal(RutaMiBolsaContacto, `{"bolsa":"bolsa:01","version":2,"clave":"clave-contacto-01","correo":"x@y.es"}`, nil))
	if w.Code != 400 || e.llamadas != 1 {
		t.Fatalf("campo ajeno: %d", w.Code)
	}
}

func TestRespuestaContactosMarcaVencimientoSinClaro(t *testing.T) {
	ahora := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	confirmada := ahora.Add(-time.Hour)
	i := puertosbolsa.InstantaneaMiBolsa{ConsultadaEn: ahora, Contactos: []puertosbolsa.ContactoPortalCandidato{
		{Bolsa: "bolsa:01", Version: 1, Origen: &puertosbolsa.OrigenContactoPortal{VigenteHasta: ahora, UltimoDia: "2026-09-24"}},
		{Bolsa: "bolsa:02", Version: 2, ConfirmadaEn: &confirmada},
	}}
	salida, _ := json.Marshal(respuestaContactos(i))
	if !strings.Contains(string(salida), `"estado":"vencido"`) || !strings.Contains(string(salida), `"origen":null`) || strings.Contains(string(salida), "@") {
		t.Fatalf("contactos: %s", salida)
	}
}
