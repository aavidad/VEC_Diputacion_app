package httpinterno

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

func marcaOrigenHTTPPrueba() *dominiobolsa.MarcaOrigenDatosContacto {
	return &dominiobolsa.MarcaOrigenDatosContacto{
		Origen: dominiobolsa.OrigenDatosContactoConvoca, VigenteHasta: time.Date(2027, 9, 28, 22, 0, 0, 0, time.UTC),
		UltimoDia: "2027-09-28", ReglaRef: "vec.bolsa.reglas:1:b29.contacto_origen_convoca", ReglaHuella: strings.Repeat("d", 64),
	}
}

func TestHandlerDatosContactoRegistraYMuestraOrigenConvoca(t *testing.T) {
	ahora := time.Date(2026, 9, 28, 8, 0, 0, 0, time.UTC)
	prep := &preparadorDatosContactoPrueba{}
	op := &operadorDatosContactoPrueba{registro: puertosbolsa.RegistroDatosContactoParticipacion{ReciboRef: "recibo:datos-contacto:0001", ParticipacionRef: "participacion:b4", Version: 1, RegistradaEn: ahora, Origen: marcaOrigenHTTPPrueba()}}
	h, _ := NuevoHandlerDatosContactoParticipacion(prep, op)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, peticionDatosContacto(http.MethodPost, rutaDatosContactoPrueba, `{"correo":"c@dipgra.es","telefono_1":"600123456","telefono_2":"","motivo":"Traído de CONVOCA","origen":"convoca"}`))
	if rec.Code != http.StatusCreated || prep.ultimaEntrada.Origen != "convoca" {
		t.Fatalf("estado=%d entrada=%+v", rec.Code, prep.ultimaEntrada)
	}
	var salida struct {
		Data struct {
			Origen map[string]string `json:"origen"`
		} `json:"data"`
	}
	if json.Unmarshal(rec.Body.Bytes(), &salida) != nil || salida.Data.Origen["origen"] != "convoca" || salida.Data.Origen["ultimo_dia"] != "2027-09-28" || salida.Data.Origen["regla_ref"] == "" {
		t.Fatalf("cuerpo=%s", rec.Body.String())
	}
	// Un origen desconocido no llega al caso de uso.
	op.registros = 0
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, peticionDatosContacto(http.MethodPost, rutaDatosContactoPrueba, `{"correo":"c@dipgra.es","telefono_1":"600123456","telefono_2":"","motivo":"x","origen":"persona"}`))
	if rec.Code != http.StatusBadRequest || op.registros != 0 {
		t.Fatalf("origen desconocido: %d registros=%d", rec.Code, op.registros)
	}
	// Sin regla configurada: 422 explícito, nunca una vigencia supuesta.
	op.errRegist = puertosbolsa.ErrOrigenDatosContactoNoConfigurado
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, peticionDatosContacto(http.MethodPost, rutaDatosContactoPrueba, `{"correo":"c@dipgra.es","telefono_1":"600123456","telefono_2":"","motivo":"x","origen":"convoca"}`))
	if rec.Code != http.StatusUnprocessableEntity || !strings.Contains(rec.Body.String(), "origen_contacto_no_disponible") {
		t.Fatalf("sin regla: %d %s", rec.Code, rec.Body.String())
	}
}

func TestHandlerDatosContactoConsultaIncluyeEstadoDelOrigen(t *testing.T) {
	datos := dominiobolsa.DatosContactoParticipacion{ParticipacionRef: "participacion:b4", Correo: "candidata@dipgra.es", Telefono1: "600123456"}
	op := &operadorDatosContactoPrueba{leidos: puertosbolsa.DatosContactoParticipacionLeidos{ParticipacionRef: "participacion:b4", Version: 1, RegistradaEn: time.Now(), Datos: datos, Enmascarados: datos.Enmascarados(), Origen: marcaOrigenHTTPPrueba(), EstadoOrigen: dominiobolsa.EstadoOrigenContactoVencido}}
	h, _ := NuevoHandlerDatosContactoParticipacion(&preparadorDatosContactoPrueba{}, op)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, peticionDatosContacto(http.MethodGet, rutaDatosContactoPrueba, ""))
	var salida struct {
		Data struct {
			Origen map[string]string `json:"origen"`
		} `json:"data"`
	}
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &salida) != nil || salida.Data.Origen["estado"] != "vencido" || strings.Contains(rec.Body.String(), "candidata@dipgra.es") {
		t.Fatalf("lectura: %d %s", rec.Code, rec.Body.String())
	}
	// Un contacto propio lleva origen nulo.
	op.leidos.Origen, op.leidos.EstadoOrigen = nil, ""
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, peticionDatosContacto(http.MethodGet, rutaDatosContactoPrueba, ""))
	if !strings.Contains(rec.Body.String(), `"origen":null`) {
		t.Fatalf("contacto propio: %s", rec.Body.String())
	}
}
